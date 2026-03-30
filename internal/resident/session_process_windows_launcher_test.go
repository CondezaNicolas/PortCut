package resident

import (
	"os"
	"os/exec"
	"testing"
)

func TestWindowsConsoleProcessTracksChildPIDFromLauncherOutput(t *testing.T) {
	process, err := newWindowsConsoleProcessWithConfig("C:/tools/portcut.exe", windowsConsoleProcessConfig{
		BuildCommand: func(executable string) *exec.Cmd {
			if executable != "C:/tools/portcut.exe" {
				t.Fatalf("expected executable path, got %q", executable)
			}
			return windowsConsoleHelperCommand("pid")
		},
	})
	if err != nil {
		t.Fatalf("expected process creation success, got %v", err)
	}

	if err := process.Start(); err != nil {
		t.Fatalf("expected launcher start success, got %v", err)
	}
	if process.PID() != 4242 {
		t.Fatalf("expected tracked child pid 4242, got %d", process.PID())
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("expected helper wait success, got %v", err)
	}
}

func TestWindowsConsoleProcessRejectsLauncherWithoutChildPID(t *testing.T) {
	process, err := newWindowsConsoleProcessWithConfig("C:/tools/portcut.exe", windowsConsoleProcessConfig{
		BuildCommand: func(string) *exec.Cmd {
			return windowsConsoleHelperCommand("bad-pid")
		},
	})
	if err != nil {
		t.Fatalf("expected process creation success, got %v", err)
	}

	if err := process.Start(); err == nil {
		t.Fatal("expected invalid pid output to fail")
	}
	if process.PID() != 0 {
		t.Fatalf("expected missing child pid after failed start, got %d", process.PID())
	}
}

func TestWindowsConsoleProcessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PORTCUT_WINDOWS_CONSOLE_HELPER") != "1" {
		return
	}

	switch os.Getenv("PORTCUT_WINDOWS_CONSOLE_HELPER_MODE") {
	case "pid":
		_, _ = os.Stdout.WriteString("4242\n")
		os.Exit(0)
	case "bad-pid":
		_, _ = os.Stdout.WriteString("not-a-pid\n")
		os.Exit(0)
	default:
		os.Exit(2)
	}
}

func windowsConsoleHelperCommand(mode string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestWindowsConsoleProcessHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_PORTCUT_WINDOWS_CONSOLE_HELPER=1",
		"PORTCUT_WINDOWS_CONSOLE_HELPER_MODE="+mode,
	)
	return cmd
}
