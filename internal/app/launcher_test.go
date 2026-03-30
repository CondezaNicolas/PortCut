package app

import (
	"errors"
	"testing"

	"portcut/internal/platform"
	platformmock "portcut/internal/platform/mock"
)

func TestLauncherRunBuildsWorkflowFromService(t *testing.T) {
	service := &platformmock.Service{
		CapabilitiesValue: platform.Capabilities{Platform: "linux", Shell: "sh"},
	}

	called := false
	launcher := NewLauncher(
		func() (platform.Service, error) {
			return service, nil
		},
		func(workflow Workflow, capabilities platform.Capabilities) ProgramRunner {
			called = true
			if workflow.service != service {
				t.Fatal("expected launcher to pass the created service into workflow")
			}
			if capabilities.Platform != "linux" {
				t.Fatalf("expected linux platform, got %q", capabilities.Platform)
			}

			return func() error {
				return nil
			}
		},
	)

	if err := launcher.Run(); err != nil {
		t.Fatalf("expected launcher success, got %v", err)
	}
	if !called {
		t.Fatal("expected program factory to be called")
	}
}

func TestLauncherRunReturnsServiceFactoryError(t *testing.T) {
	wantErr := errors.New("boom")
	launcher := NewLauncher(
		func() (platform.Service, error) {
			return nil, wantErr
		},
		func(Workflow, platform.Capabilities) ProgramRunner {
			t.Fatal("program factory should not be called after service error")
			return nil
		},
	)

	err := launcher.Run()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected service error, got %v", err)
	}
}

func TestLauncherRunReturnsProgramError(t *testing.T) {
	wantErr := errors.New("run failed")
	launcher := NewLauncher(
		func() (platform.Service, error) {
			return &platformmock.Service{
				CapabilitiesValue: platform.Capabilities{Platform: "windows", Shell: "powershell"},
			}, nil
		},
		func(Workflow, platform.Capabilities) ProgramRunner {
			return func() error {
				return wantErr
			}
		},
	)

	err := launcher.Run()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected program error, got %v", err)
	}
}
