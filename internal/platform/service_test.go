package platform

import (
	"context"
	"errors"
	"testing"
	"time"

	"portcut/internal/domain"
)

func TestNewServiceForSelectsSupportedAdapter(t *testing.T) {
	tests := []struct {
		name             string
		goos             string
		wantPlatform     string
		wantShell        string
		wantGracefulKill bool
	}{
		{name: "linux", goos: "linux", wantPlatform: "linux", wantShell: "sh", wantGracefulKill: true},
		{name: "darwin", goos: "darwin", wantPlatform: "darwin", wantShell: "sh", wantGracefulKill: true},
		{name: "windows", goos: "windows", wantPlatform: "windows", wantShell: "powershell", wantGracefulKill: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceFor(tt.goos)
			if err != nil {
				t.Fatalf("expected supported platform, got error: %v", err)
			}

			capabilities := service.Capabilities()
			if capabilities.Platform != tt.wantPlatform {
				t.Fatalf("expected platform %q, got %q", tt.wantPlatform, capabilities.Platform)
			}
			if capabilities.Shell != tt.wantShell {
				t.Fatalf("expected shell %q, got %q", tt.wantShell, capabilities.Shell)
			}
			if capabilities.GracefulTermination != tt.wantGracefulKill {
				t.Fatalf("expected graceful kill %t, got %t", tt.wantGracefulKill, capabilities.GracefulTermination)
			}
			if !capabilities.Discovery {
				t.Fatal("expected discovery capability to be true")
			}
			if !capabilities.ForceTermination {
				t.Fatal("expected force termination capability to be true")
			}
		})
	}
}

func TestNewServiceForRejectsUnsupportedPlatform(t *testing.T) {
	_, err := NewServiceFor("plan9")
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}

	if !IsUnsupportedPlatform(err) {
		t.Fatalf("expected IsUnsupportedPlatform to match, got %v", err)
	}

	var typedErr UnsupportedPlatformError
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected UnsupportedPlatformError, got %T", err)
	}

	if typedErr.GOOS != "plan9" {
		t.Fatalf("expected goos plan9, got %q", typedErr.GOOS)
	}
}

func TestCommandServiceTerminateClassifiesUnsupportedGracefulKill(t *testing.T) {
	service := newCommandService(
		Capabilities{Platform: "windows", ForceTermination: true},
		commandSpec{Name: "discover"},
		func([]byte) ([]domain.PortProcessEntry, error) { return nil, nil },
		func(domain.KillTarget, bool) commandSpec { return commandSpec{Name: "terminate"} },
	).(commandService)
	service.now = func() time.Time { return time.Unix(100, 0).UTC() }

	result, err := service.Terminate(context.Background(), TerminateRequest{Targets: []domain.KillTarget{{PID: 123, ProcessName: "api"}}})
	if err != nil {
		t.Fatalf("expected graceful unsupported to surface as typed outcome, got %v", err)
	}
	if len(result.Outcomes) != 1 {
		t.Fatalf("expected one outcome, got %d", len(result.Outcomes))
	}
	if result.Outcomes[0].Status != TerminationStatusSkipped || result.Outcomes[0].Kind != TerminationOutcomeKindUnsupported {
		t.Fatalf("unexpected unsupported outcome: %#v", result.Outcomes[0])
	}
}

func TestCommandServiceTerminateClassifiesPermissionDenied(t *testing.T) {
	service := newCommandService(
		Capabilities{Platform: "linux", GracefulTermination: true, ForceTermination: true},
		commandSpec{Name: "discover"},
		func([]byte) ([]domain.PortProcessEntry, error) { return nil, nil },
		func(domain.KillTarget, bool) commandSpec {
			return commandSpec{Name: "kill", Args: []string{"-TERM", "123"}}
		},
	).(commandService)
	service.run = func(_ context.Context, spec commandSpec) ([]byte, error) {
		if spec.Name != "kill" {
			return nil, errors.New("unexpected command")
		}
		return []byte("kill: (123) - Operation not permitted"), errors.New("exit status 1")
	}

	result, err := service.Terminate(context.Background(), TerminateRequest{Targets: []domain.KillTarget{{PID: 123, ProcessName: "api"}}})
	if err != nil {
		t.Fatalf("expected typed permission outcome, got %v", err)
	}
	if len(result.Outcomes) != 1 {
		t.Fatalf("expected one outcome, got %d", len(result.Outcomes))
	}
	if result.Outcomes[0].Status != TerminationStatusFailed || result.Outcomes[0].Kind != TerminationOutcomeKindPermissionDenied {
		t.Fatalf("unexpected permission outcome: %#v", result.Outcomes[0])
	}
}
