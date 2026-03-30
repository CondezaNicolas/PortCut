package resident

import "testing"

func TestProcessSessionConfigForWindowsAndDarwinIncludesHooks(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			config := ProcessSessionConfigFor(goos)
			if goos == "windows" && config.NewProcess == nil {
				t.Fatal("expected windows launcher override")
			}
			if config.RequestReopen == nil {
				t.Fatalf("expected reopen hook for %s", goos)
			}
			if config.RequestShutdown == nil {
				t.Fatalf("expected shutdown hook for %s", goos)
			}
		})
	}
}

func TestProcessSessionConfigForLinuxUsesGracefulShutdownWithoutForegroundHook(t *testing.T) {
	config := ProcessSessionConfigFor("linux")
	if config.NewProcess != nil {
		t.Fatal("expected linux launcher to remain default")
	}
	if config.RequestReopen != nil {
		t.Fatal("expected linux reopen hook to remain unset")
	}
	if config.RequestShutdown == nil {
		t.Fatal("expected linux shutdown hook")
	}
}
