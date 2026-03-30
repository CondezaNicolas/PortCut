package darwin

import (
	"strings"
	"testing"
)

func TestForegroundCommandTargetsUnixProcessID(t *testing.T) {
	name, args := ForegroundCommand(4242)
	if name != "osascript" {
		t.Fatalf("expected osascript foreground command, got %q", name)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "unix id is 4242") {
		t.Fatalf("expected process id in foreground command, got %q", joined)
	}
	if !strings.Contains(joined, "frontmost") {
		t.Fatalf("expected frontmost activation script, got %q", joined)
	}
}

func TestShutdownCommandUsesTermSignal(t *testing.T) {
	name, args := ShutdownCommand(4242)
	if name != "kill" {
		t.Fatalf("expected kill shutdown command, got %q", name)
	}
	joined := strings.Join(args, " ")
	if joined != "-TERM 4242" {
		t.Fatalf("expected TERM shutdown args, got %q", joined)
	}
}

func TestIconDecodesForTrayUsage(t *testing.T) {
	if len(Icon()) == 0 {
		t.Fatal("expected tray icon bytes")
	}
}
