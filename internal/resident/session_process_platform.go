package resident

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	residentdarwin "portcut/internal/resident/darwin"
	residentlinux "portcut/internal/resident/linux"
	residentwindows "portcut/internal/resident/windows"
)

func NewPlatformProcessSession() *ProcessSession {
	return NewPlatformProcessSessionFor(runtime.GOOS)
}

func NewPlatformProcessSessionFor(goos string) *ProcessSession {
	return NewProcessSession(ProcessSessionConfigFor(goos))
}

func ProcessSessionConfigFor(goos string) ProcessSessionConfig {
	switch goos {
	case "windows":
		return ProcessSessionConfig{
			NewProcess:      newWindowsConsoleProcess,
			RequestReopen:   commandProcessRequest(residentwindows.ForegroundCommand),
			RequestShutdown: commandProcessRequest(residentwindows.ShutdownCommand),
		}
	case "darwin":
		return ProcessSessionConfig{
			RequestReopen:   commandProcessRequest(residentdarwin.ForegroundCommand),
			RequestShutdown: commandProcessRequest(residentdarwin.ShutdownCommand),
		}
	case "linux":
		return ProcessSessionConfig{
			RequestShutdown: commandProcessRequest(residentlinux.ShutdownCommand),
		}
	default:
		return ProcessSessionConfig{}
	}
}

type processCommandBuilder func(int) (string, []string)

func commandProcessRequest(build processCommandBuilder) ProcessRequestFunc {
	return func(ctx context.Context, process Process) error {
		if build == nil || process == nil {
			return nil
		}

		name, args := build(process.PID())
		if name == "" {
			return nil
		}

		cmd := exec.CommandContext(ctx, name, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run %s hook: %w", name, err)
		}

		return nil
	}
}
