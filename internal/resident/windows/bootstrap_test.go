package windows

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func TestEnsureDetachedRelaunchesInteractiveWindowsParent(t *testing.T) {
	t.Helper()

	var started *exec.Cmd
	detachCalls := 0

	mode, err := EnsureDetached(BootstrapConfig{
		GOOS:       "windows",
		Args:       []string{"portcut-resident.exe", "--verbose"},
		Env:        []string{"PATH=C:/Windows/System32", BootstrapEnv + "=stale"},
		Executable: "C:/tools/portcut-resident.exe",
		Detach: func(cmd *exec.Cmd) error {
			detachCalls++
			if cmd == nil {
				t.Fatal("expected detach command")
			}
			return nil
		},
		Start: func(cmd *exec.Cmd) error {
			started = cmd
			return nil
		},
	})
	if err != nil {
		t.Fatalf("expected detached relaunch, got %v", err)
	}
	if mode != BootstrapRelaunched {
		t.Fatalf("expected relaunch mode, got %v", mode)
	}
	if detachCalls != 1 {
		t.Fatalf("expected one detach call, got %d", detachCalls)
	}
	if started == nil {
		t.Fatal("expected detached command to start")
	}
	if started.Path != "C:/tools/portcut-resident.exe" {
		t.Fatalf("unexpected command path: %q", started.Path)
	}

	wantArgs := []string{"C:/tools/portcut-resident.exe", "--verbose", BootstrapFlag}
	if !reflect.DeepEqual(started.Args, wantArgs) {
		t.Fatalf("unexpected detached args: %#v", started.Args)
	}

	if !containsEnvEntry(started.Env, "PATH=C:/Windows/System32") {
		t.Fatalf("expected child env to preserve existing entries, got %#v", started.Env)
	}
	if !containsEnvEntry(started.Env, BootstrapEnv+"=1") {
		t.Fatalf("expected bootstrap marker in child env, got %#v", started.Env)
	}
	if containsEnvEntry(started.Env, BootstrapEnv+"=stale") {
		t.Fatalf("expected stale bootstrap marker to be replaced, got %#v", started.Env)
	}
}

func TestEnsureDetachedAllowsBootstrappedWindowsChildToContinueInline(t *testing.T) {
	mode, err := EnsureDetached(BootstrapConfig{
		GOOS:       "windows",
		Args:       []string{"portcut-resident.exe", BootstrapFlag},
		Env:        []string{BootstrapEnv + "=1"},
		Executable: "C:/tools/portcut-resident.exe",
		Detach: func(*exec.Cmd) error {
			t.Fatal("did not expect child process to reconfigure detach")
			return nil
		},
		Start: func(*exec.Cmd) error {
			t.Fatal("did not expect child process to relaunch")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("expected child to continue inline, got %v", err)
	}
	if mode != BootstrapInline {
		t.Fatalf("expected inline mode, got %v", mode)
	}
}

func TestEnsureDetachedRejectsPartialBootstrapMarkers(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  []string
	}{
		{name: "flag without env", args: []string{"portcut-resident.exe", BootstrapFlag}},
		{name: "env without flag", args: []string{"portcut-resident.exe"}, env: []string{BootstrapEnv + "=1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := EnsureDetached(BootstrapConfig{
				GOOS:       "windows",
				Args:       tt.args,
				Env:        tt.env,
				Executable: "C:/tools/portcut-resident.exe",
			})
			if mode != BootstrapInline {
				t.Fatalf("expected inline mode on invalid state, got %v", mode)
			}
			if !errors.Is(err, ErrInvalidBootstrapState) {
				t.Fatalf("expected invalid bootstrap state, got %v", err)
			}
			if !IsBootstrapError(err) {
				t.Fatalf("expected bootstrap error classification, got %v", err)
			}
		})
	}
}

func TestEnsureDetachedDoesNothingOutsideWindows(t *testing.T) {
	mode, err := EnsureDetached(BootstrapConfig{
		GOOS:       "linux",
		Args:       []string{"portcut-resident"},
		Executable: "/usr/local/bin/portcut-resident",
		Detach: func(*exec.Cmd) error {
			t.Fatal("did not expect non-windows launch to detach")
			return nil
		},
		Start: func(*exec.Cmd) error {
			t.Fatal("did not expect non-windows launch to relaunch")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("expected non-windows no-op, got %v", err)
	}
	if mode != BootstrapInline {
		t.Fatalf("expected inline mode, got %v", mode)
	}
}

func TestEnsureDetachedWrapsDetachedLaunchFailures(t *testing.T) {
	wantErr := errors.New("boom")

	mode, err := EnsureDetached(BootstrapConfig{
		GOOS:       "windows",
		Args:       []string{"portcut-resident.exe"},
		Executable: "C:/tools/portcut-resident.exe",
		Detach: func(*exec.Cmd) error {
			return nil
		},
		Start: func(*exec.Cmd) error {
			return wantErr
		},
	})
	if mode != BootstrapInline {
		t.Fatalf("expected inline mode on failure, got %v", mode)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected start failure to be wrapped, got %v", err)
	}
	if !IsBootstrapError(err) {
		t.Fatalf("expected bootstrap error classification, got %v", err)
	}
}

func containsEnvEntry(entries []string, want string) bool {
	for _, entry := range entries {
		if entry == want {
			return true
		}
	}

	return false
}
