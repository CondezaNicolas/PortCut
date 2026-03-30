package resident

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"portcut/internal/platform"
	residentwindows "portcut/internal/resident/windows"
)

func TestDefaultMenuIsStableAndNonDestructive(t *testing.T) {
	menu := DefaultMenu()
	if len(menu) != 2 {
		t.Fatalf("expected two resident menu items, got %d", len(menu))
	}
	if menu[0] != (MenuItem{Label: "Open Portcut", Action: MenuActionOpen}) {
		t.Fatalf("unexpected first menu item: %#v", menu[0])
	}
	if menu[1] != (MenuItem{Label: "Quit", Action: MenuActionQuit}) {
		t.Fatalf("unexpected second menu item: %#v", menu[1])
	}
	if err := ValidateMenu(menu); err != nil {
		t.Fatalf("expected default menu to validate, got %v", err)
	}
}

func TestNewAdapterForRejectsUnsupportedPlatform(t *testing.T) {
	_, err := NewAdapterFor("plan9")
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
	if !IsUnsupportedPlatform(err) {
		t.Fatalf("expected unsupported platform classification, got %v", err)
	}

	var typedErr platform.UnsupportedPlatformError
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected typed unsupported platform error, got %T", err)
	}
}

func TestNewAdapterForReturnsLinuxAdapterWhenDesktopIsSupported(t *testing.T) {
	adapter, err := newAdapterForEnvironment("linux", map[string]string{
		"WAYLAND_DISPLAY":     "wayland-0",
		"XDG_CURRENT_DESKTOP": "KDE",
	})
	if err != nil {
		t.Fatalf("expected linux adapter, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil linux adapter")
	}
}

func TestNewAdapterForSurfacesLinuxSupportBoundaries(t *testing.T) {
	adapter, err := newAdapterForEnvironment("linux", map[string]string{
		"DISPLAY":             ":0",
		"XDG_CURRENT_DESKTOP": "GNOME",
	})
	if adapter != nil {
		t.Fatal("expected linux adapter creation to fail for unsupported desktop")
	}
	if !IsAdapterUnavailable(err) {
		t.Fatalf("expected adapter unavailable error, got %v", err)
	}
	if !strings.Contains(err.Error(), "gnome") {
		t.Fatalf("expected gnome support boundary in error, got %v", err)
	}
}

func TestNewAdapterForReturnsPlatformAdaptersForWindowsAndDarwin(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			adapter, err := NewAdapterFor(goos)
			if err != nil {
				t.Fatalf("expected adapter for %s, got %v", goos, err)
			}
			if adapter == nil {
				t.Fatalf("expected non-nil adapter for %s", goos)
			}
		})
	}
}

func TestNewAdapterForWindowsPreservesIconAccessorContract(t *testing.T) {
	adapter, err := NewAdapterFor("windows")
	if err != nil {
		t.Fatalf("expected windows adapter, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil windows adapter")
	}

	first := residentwindows.Icon()
	second := residentwindows.Icon()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("expected windows icon bytes from accessor")
	}
	if !bytes.Equal(first, second) {
		t.Fatal("expected repeated Icon calls to return the same payload")
	}
	if &first[0] == &second[0] {
		t.Fatal("expected Icon to preserve clone-per-call behavior")
	}

	originalFirstByte := second[0]
	first[0] ^= 0xff
	third := residentwindows.Icon()
	if third[0] != originalFirstByte {
		t.Fatal("expected Icon mutations to stay isolated from future callers")
	}
}
