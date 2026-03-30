package resident

import (
	"context"
	"errors"
	"fmt"

	residentwindows "portcut/internal/resident/windows"
)

var ErrInvalidRuntime = errors.New("invalid resident runtime")

type SessionFactory func() Session

type HostFactory func(Session) (Host, error)

type AdapterFactory func() (Adapter, error)

type RuntimeConfig struct {
	NewSession SessionFactory
	NewHost    HostFactory
	NewAdapter AdapterFactory
}

func Run(ctx context.Context, config RuntimeConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}

	newSession := config.NewSession
	if newSession == nil {
		newSession = func() Session {
			return NewPlatformProcessSession()
		}
	}

	session := newSession()
	if session == nil {
		return fmt.Errorf("%w: session is required", ErrInvalidRuntime)
	}

	newHost := config.NewHost
	if newHost == nil {
		newHost = func(session Session) (Host, error) {
			return NewHost(session)
		}
	}

	host, err := newHost(session)
	if err != nil {
		return err
	}

	newAdapter := config.NewAdapter
	if newAdapter == nil {
		newAdapter = NewAdapter
	}

	adapter, err := newAdapter()
	if err != nil {
		return err
	}
	if adapter == nil {
		return fmt.Errorf("%w: adapter is required", ErrInvalidRuntime)
	}

	return adapter.Run(ctx, host)
}

func FormatLaunchError(err error) string {
	if err == nil {
		return ""
	}

	var unavailable AdapterUnavailableError
	if IsAdapterUnavailable(err) && errors.As(err, &unavailable) {
		if unavailable.Reason != "" {
			return fmt.Sprintf("portcut resident mode is unavailable on %s: %s", unavailable.GOOS, unavailable.Reason)
		}

		return fmt.Sprintf("portcut resident mode is unavailable on %s", unavailable.GOOS)
	}
	if IsUnsupportedPlatform(err) {
		return fmt.Sprintf("portcut resident mode is unsupported on this platform: %v", err)
	}
	if residentwindows.IsBootstrapError(err) {
		return fmt.Sprintf("portcut resident mode failed to start detached on windows: %v", err)
	}

	return fmt.Sprintf("portcut resident mode failed: %v", err)
}
