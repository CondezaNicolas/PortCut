package resident

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var ErrPortcutExecutableNotFound = errors.New("portcut executable not found")

type ExecutableResolver func() (string, error)

type Process interface {
	Start() error
	Wait() error
	PID() int
}

type ProcessFactory func(string) (Process, error)

type ProcessRequestFunc func(context.Context, Process) error

type ProcessSessionConfig struct {
	ResolveExecutable ExecutableResolver
	NewProcess        ProcessFactory
	RequestReopen     ProcessRequestFunc
	RequestShutdown   ProcessRequestFunc
}

type ProcessSession struct {
	mu sync.RWMutex

	snapshot SessionSnapshot
	events   chan SessionEvent

	resolveExecutable ExecutableResolver
	newProcess        ProcessFactory
	requestReopen     ProcessRequestFunc
	requestShutdown   ProcessRequestFunc

	process  Process
	waitDone chan struct{}
}

func NewProcessSession(config ProcessSessionConfig) *ProcessSession {
	resolveExecutable := config.ResolveExecutable
	if resolveExecutable == nil {
		resolveExecutable = ResolvePortcutExecutable
	}

	newProcess := config.NewProcess
	if newProcess == nil {
		newProcess = newExecProcess
	}

	return &ProcessSession{
		snapshot:          SessionSnapshot{State: SessionStateIdle},
		events:            make(chan SessionEvent, 8),
		resolveExecutable: resolveExecutable,
		newProcess:        newProcess,
		requestReopen:     config.RequestReopen,
		requestShutdown:   config.RequestShutdown,
	}
}

func ResolvePortcutExecutable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}

	candidate := filepath.Join(filepath.Dir(self), "portcut"+filepath.Ext(self))
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate, nil
	}

	lookupName := "portcut" + filepath.Ext(self)
	path, err := exec.LookPath(lookupName)
	if err == nil {
		return path, nil
	}

	if filepath.Ext(self) != "" {
		path, err = exec.LookPath("portcut")
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("%w", ErrPortcutExecutableNotFound)
}

func (s *ProcessSession) Open(ctx context.Context, request OpenRequest) error {
	if request.Reason == "" {
		request.Reason = OpenReasonLaunch
	}

	s.mu.Lock()
	if s.snapshot.State == SessionStateStopping {
		s.mu.Unlock()
		return ErrSessionStopping
	}
	if s.snapshot.Active {
		process := s.process
		reopen := s.requestReopen
		if request.Reason != OpenReasonReopen {
			s.mu.Unlock()
			return ErrSessionAlreadyRunning
		}
		s.publishLocked(SessionEvent{Type: SessionEventReopenRequested, Open: request})
		s.mu.Unlock()

		if reopen != nil && process != nil {
			return reopen(ctx, process)
		}

		return nil
	}

	executable, err := s.resolveExecutable()
	if err != nil {
		s.mu.Unlock()
		return err
	}

	process, err := s.newProcess(executable)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if err := process.Start(); err != nil {
		s.mu.Unlock()
		return err
	}

	waitDone := make(chan struct{})
	s.process = process
	s.waitDone = waitDone
	s.snapshot = SessionSnapshot{
		State:  SessionStateRunning,
		Open:   request,
		Active: true,
	}
	s.publishLocked(SessionEvent{Type: SessionEventOpened, Open: request})
	s.mu.Unlock()

	go s.waitForExit(process, waitDone)

	return nil
}

func (s *ProcessSession) Shutdown(ctx context.Context, request ShutdownRequest) error {
	s.mu.Lock()
	if !s.snapshot.Active {
		s.mu.Unlock()
		return nil
	}

	process := s.process
	waitDone := s.waitDone
	shutdown := s.requestShutdown
	if s.snapshot.State != SessionStateStopping {
		s.snapshot.State = SessionStateStopping
		s.publishLocked(SessionEvent{Type: SessionEventShutdownRequested})
	}
	s.mu.Unlock()

	var shutdownErr error
	if shutdown != nil && process != nil {
		shutdownErr = shutdown(ctx, process)
	}

	if !request.Wait || waitDone == nil {
		return shutdownErr
	}

	select {
	case <-waitDone:
		return shutdownErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ProcessSession) Snapshot() SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot
}

func (s *ProcessSession) Events() <-chan SessionEvent {
	return s.events
}

func (s *ProcessSession) waitForExit(process Process, waitDone chan struct{}) {
	err := process.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	defer close(waitDone)

	if s.process != process {
		return
	}

	s.process = nil
	s.waitDone = nil
	s.snapshot = SessionSnapshot{State: SessionStateIdle}
	s.publishLocked(SessionEvent{Type: SessionEventExited, Err: err})
}

func (s *ProcessSession) publishLocked(event SessionEvent) {
	select {
	case s.events <- event:
	default:
	}
}

func newExecProcess(executable string) (Process, error) {
	cmd := exec.Command(executable)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return &execProcess{cmd: cmd}, nil
}

func readChildPID(r io.Reader) (int, error) {
	if r == nil {
		return 0, errors.New("child pid stream is required")
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		pid, err := strconv.Atoi(line)
		if err != nil {
			return 0, fmt.Errorf("parse child pid %q: %w", line, err)
		}

		if pid <= 0 {
			return 0, fmt.Errorf("parse child pid %q: pid must be positive", line)
		}

		return pid, nil
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, errors.New("child pid stream ended before reporting pid")
}

type execProcess struct {
	cmd *exec.Cmd
}

func (p *execProcess) Start() error {
	return p.cmd.Start()
}

func (p *execProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *execProcess) PID() int {
	if p.cmd.Process == nil {
		return 0
	}

	return p.cmd.Process.Pid
}
