package resident

import (
	"fmt"
	"io"
	"os/exec"

	residentwindows "portcut/internal/resident/windows"
)

type windowsConsoleProcessConfig struct {
	BuildCommand func(string) *exec.Cmd
	ReadChildPID func(io.Reader) (int, error)
}

func newWindowsConsoleProcess(executable string) (Process, error) {
	return newWindowsConsoleProcessWithConfig(executable, windowsConsoleProcessConfig{})
}

func newWindowsConsoleProcessWithConfig(executable string, config windowsConsoleProcessConfig) (Process, error) {
	buildCommand := config.BuildCommand
	if buildCommand == nil {
		buildCommand = residentwindows.NewConsoleCommand
	}

	cmd := buildCommand(executable)
	if cmd == nil {
		return nil, fmt.Errorf("build windows console command: command is required")
	}

	readPID := config.ReadChildPID
	if readPID == nil {
		readPID = readChildPID
	}

	return &windowsConsoleProcess{
		cmd:          cmd,
		readChildPID: readPID,
	}, nil
}

type windowsConsoleProcess struct {
	cmd          *exec.Cmd
	stdout       io.ReadCloser
	readChildPID func(io.Reader) (int, error)
	pid          int
}

func (p *windowsConsoleProcess) Start() error {
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("capture windows console launcher pid output: %w", err)
	}
	p.stdout = stdout

	if p.cmd.Stderr == nil {
		p.cmd.Stderr = io.Discard
	}

	if err := p.cmd.Start(); err != nil {
		return err
	}

	pid, err := p.readChildPID(stdout)
	if err != nil {
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		return fmt.Errorf("read windows child pid: %w", err)
	}

	p.pid = pid
	return nil
}

func (p *windowsConsoleProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *windowsConsoleProcess) PID() int {
	return p.pid
}
