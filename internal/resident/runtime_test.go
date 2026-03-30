package resident

import (
	"context"
	"errors"
	"testing"

	"portcut/internal/platform"
	residentwindows "portcut/internal/resident/windows"
)

func TestRunComposesSessionHostAndAdapter(t *testing.T) {
	session := &hostSessionDouble{}
	host := &hostDouble{}
	adapter := &adapterDouble{}

	err := Run(context.Background(), RuntimeConfig{
		NewSession: func() Session {
			return session
		},
		NewHost: func(got Session) (Host, error) {
			if got != session {
				t.Fatal("expected runtime to pass session into host factory")
			}
			return host, nil
		},
		NewAdapter: func() (Adapter, error) {
			return adapter, nil
		},
	})
	if err != nil {
		t.Fatalf("expected runtime success, got %v", err)
	}
	if adapter.host != host {
		t.Fatal("expected runtime to pass host into adapter")
	}
	if adapter.ctx == nil {
		t.Fatal("expected runtime to pass context into adapter")
	}
}

func TestRunReturnsFactoryErrors(t *testing.T) {
	wantErr := errors.New("boom")

	for _, tc := range []struct {
		name   string
		config RuntimeConfig
	}{
		{
			name: "host",
			config: RuntimeConfig{
				NewSession: func() Session { return &hostSessionDouble{} },
				NewHost: func(Session) (Host, error) {
					return nil, wantErr
				},
			},
		},
		{
			name: "adapter",
			config: RuntimeConfig{
				NewSession: func() Session { return &hostSessionDouble{} },
				NewHost: func(Session) (Host, error) {
					return &hostDouble{}, nil
				},
				NewAdapter: func() (Adapter, error) {
					return nil, wantErr
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Run(context.Background(), tc.config)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected factory error, got %v", err)
			}
		})
	}
}

func TestRunRejectsMissingRuntimeDependencies(t *testing.T) {
	err := Run(context.Background(), RuntimeConfig{
		NewSession: func() Session { return nil },
	})
	if err == nil {
		t.Fatal("expected missing session error")
	}
	if !errors.Is(err, ErrInvalidRuntime) {
		t.Fatalf("expected invalid runtime error, got %v", err)
	}

	err = Run(context.Background(), RuntimeConfig{
		NewSession: func() Session { return &hostSessionDouble{} },
		NewHost: func(Session) (Host, error) {
			return &hostDouble{}, nil
		},
		NewAdapter: func() (Adapter, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("expected missing adapter error")
	}
	if !errors.Is(err, ErrInvalidRuntime) {
		t.Fatalf("expected invalid runtime error, got %v", err)
	}
}

func TestFormatLaunchError(t *testing.T) {
	linuxErr := AdapterUnavailableError{GOOS: "linux", Reason: "desktop environment does not advertise supported tray integration"}
	if got := FormatLaunchError(linuxErr); got != "portcut resident mode is unavailable on linux: desktop environment does not advertise supported tray integration" {
		t.Fatalf("unexpected linux unavailable message: %q", got)
	}

	unsupported := platform.UnsupportedPlatformError{GOOS: "plan9"}
	if got := FormatLaunchError(unsupported); got != "portcut resident mode is unsupported on this platform: unsupported platform: plan9" {
		t.Fatalf("unexpected unsupported-platform message: %q", got)
	}

	bootstrap := residentwindows.BootstrapError{Op: "start detached resident host", Err: errors.New("access denied")}
	if got := FormatLaunchError(bootstrap); got != "portcut resident mode failed to start detached on windows: windows detached bootstrap start detached resident host: access denied" {
		t.Fatalf("unexpected windows bootstrap message: %q", got)
	}

	generic := errors.New("boom")
	if got := FormatLaunchError(generic); got != "portcut resident mode failed: boom" {
		t.Fatalf("unexpected generic message: %q", got)
	}
}

type hostDouble struct{}

func (h *hostDouble) Menu() []MenuItem {
	return DefaultMenu()
}

func (h *hostDouble) OpenPortcut(context.Context) error {
	return nil
}

func (h *hostDouble) Quit(context.Context) error {
	return nil
}

type adapterDouble struct {
	ctx  context.Context
	host Host
	err  error
}

func (a *adapterDouble) Run(ctx context.Context, host Host) error {
	a.ctx = ctx
	a.host = host
	return a.err
}
