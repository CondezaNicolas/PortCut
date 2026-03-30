package linux

import (
	"strings"
	"testing"
)

func TestSupportStatusForAcceptsSupportedWaylandDesktop(t *testing.T) {
	status := SupportStatusFor(map[string]string{
		"WAYLAND_DISPLAY":     "wayland-0",
		"XDG_CURRENT_DESKTOP": "KDE",
	})
	if !status.Supported {
		t.Fatalf("expected supported linux desktop, got %#v", status)
	}
	if status.Reason != "" {
		t.Fatalf("expected empty reason for supported desktop, got %q", status.Reason)
	}
}

func TestSupportStatusForRejectsHeadlessLinuxSession(t *testing.T) {
	status := SupportStatusFor(map[string]string{"XDG_CURRENT_DESKTOP": "KDE"})
	if status.Supported {
		t.Fatalf("expected unsupported headless session, got %#v", status)
	}
	if !strings.Contains(status.Reason, "graphical desktop session") {
		t.Fatalf("expected graphical-session failure reason, got %q", status.Reason)
	}
}

func TestSupportStatusForRejectsGNOMEBoundary(t *testing.T) {
	status := SupportStatusFor(map[string]string{
		"DISPLAY":             ":0",
		"XDG_CURRENT_DESKTOP": "GNOME:GNOME",
	})
	if status.Supported {
		t.Fatalf("expected gnome session to remain unsupported, got %#v", status)
	}
	if !strings.Contains(strings.ToLower(status.Reason), "gnome") {
		t.Fatalf("expected gnome-specific reason, got %q", status.Reason)
	}
}

func TestShutdownCommandUsesTermSignal(t *testing.T) {
	name, args := ShutdownCommand(4242)
	if name != "kill" {
		t.Fatalf("expected kill shutdown command, got %q", name)
	}
	if strings.Join(args, " ") != "-TERM 4242" {
		t.Fatalf("expected TERM shutdown args, got %q", strings.Join(args, " "))
	}
}

func TestIconDecodesForTrayUsage(t *testing.T) {
	if len(Icon()) == 0 {
		t.Fatal("expected tray icon bytes")
	}
}
