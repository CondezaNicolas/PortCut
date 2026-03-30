package resident

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProcessSessionStartsOneChildAtATimeAndPublishesExit(t *testing.T) {
	process := newProcessDouble()
	session := NewProcessSession(ProcessSessionConfig{
		ResolveExecutable: func() (string, error) {
			return "/tmp/portcut", nil
		},
		NewProcess: func(path string) (Process, error) {
			if path != "/tmp/portcut" {
				t.Fatalf("expected resolved executable path, got %q", path)
			}
			return process, nil
		},
	})

	if err := session.Open(context.Background(), OpenRequest{}); err != nil {
		t.Fatalf("expected first launch success, got %v", err)
	}
	opened := <-session.Events()
	if opened.Type != SessionEventOpened || opened.Open.Reason != OpenReasonLaunch {
		t.Fatalf("unexpected opened event: %#v", opened)
	}

	if err := session.Open(context.Background(), OpenRequest{Reason: OpenReasonLaunch}); !errors.Is(err, ErrSessionAlreadyRunning) {
		t.Fatalf("expected already-running error, got %v", err)
	}

	wantErr := errors.New("child exited")
	process.finish(wantErr)
	if event := <-session.Events(); event.Type != SessionEventExited {
		t.Fatalf("expected exited event, got %#v", event)
	} else if !errors.Is(event.Err, wantErr) {
		t.Fatalf("expected exit error propagation, got %v", event.Err)
	}

	snapshot := session.Snapshot()
	if snapshot.State != SessionStateIdle || snapshot.Active {
		t.Fatalf("expected idle snapshot after child exit, got %#v", snapshot)
	}
}

func TestProcessSessionRequestsReopenForRunningChild(t *testing.T) {
	process := newProcessDouble()
	reopenCalls := 0
	session := NewProcessSession(ProcessSessionConfig{
		ResolveExecutable: func() (string, error) { return "/tmp/portcut", nil },
		NewProcess:        func(string) (Process, error) { return process, nil },
		RequestReopen: func(_ context.Context, child Process) error {
			reopenCalls++
			if child.PID() != 4242 {
				t.Fatalf("expected active child pid, got %d", child.PID())
			}
			return nil
		},
	})

	if err := session.Open(context.Background(), OpenRequest{}); err != nil {
		t.Fatalf("expected first launch success, got %v", err)
	}
	<-session.Events()

	if err := session.Open(context.Background(), OpenRequest{Reason: OpenReasonReopen}); err != nil {
		t.Fatalf("expected reopen success, got %v", err)
	}
	event := <-session.Events()
	if event.Type != SessionEventReopenRequested || event.Open.Reason != OpenReasonReopen {
		t.Fatalf("expected reopen event, got %#v", event)
	}
	if reopenCalls != 1 {
		t.Fatalf("expected one reopen request, got %d", reopenCalls)
	}
}

func TestProcessSessionQuitWaitsForChildExit(t *testing.T) {
	process := newProcessDouble()
	shutdownCalls := 0
	session := NewProcessSession(ProcessSessionConfig{
		ResolveExecutable: func() (string, error) { return "/tmp/portcut", nil },
		NewProcess:        func(string) (Process, error) { return process, nil },
		RequestShutdown: func(_ context.Context, child Process) error {
			shutdownCalls++
			if child.PID() != 4242 {
				t.Fatalf("expected active child pid, got %d", child.PID())
			}
			return nil
		},
	})

	if err := session.Open(context.Background(), OpenRequest{}); err != nil {
		t.Fatalf("expected first launch success, got %v", err)
	}
	<-session.Events()

	quitDone := make(chan error, 1)
	go func() {
		quitDone <- session.Shutdown(context.Background(), ShutdownRequest{Wait: true})
	}()

	shutdownRequested := <-session.Events()
	if shutdownRequested.Type != SessionEventShutdownRequested {
		t.Fatalf("expected shutdown requested event, got %#v", shutdownRequested)
	}
	if shutdownCalls != 1 {
		t.Fatalf("expected one shutdown coordination request, got %d", shutdownCalls)
	}

	select {
	case err := <-quitDone:
		t.Fatalf("expected quit to block until child exit, got %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	process.finish(nil)
	if event := <-session.Events(); event.Type != SessionEventExited {
		t.Fatalf("expected exited event, got %#v", event)
	}

	select {
	case err := <-quitDone:
		if err != nil {
			t.Fatalf("expected quit success after child exit, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected quit to finish after child exit")
	}
}

func TestProcessSessionShutdownReturnsImmediatelyWhenIdle(t *testing.T) {
	session := NewProcessSession(ProcessSessionConfig{})
	if err := session.Shutdown(context.Background(), ShutdownRequest{Wait: true}); err != nil {
		t.Fatalf("expected idle shutdown success, got %v", err)
	}
}

type processDouble struct {
	pid       int
	startErr  error
	finished  chan error
	started   bool
	startPath string
}

func newProcessDouble() *processDouble {
	return &processDouble{
		pid:      4242,
		finished: make(chan error, 1),
	}
}

func (p *processDouble) Start() error {
	p.started = true
	return p.startErr
}

func (p *processDouble) Wait() error {
	return <-p.finished
}

func (p *processDouble) PID() int {
	return p.pid
}

func (p *processDouble) finish(err error) {
	p.finished <- err
}
