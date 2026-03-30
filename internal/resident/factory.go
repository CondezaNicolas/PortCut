package resident

import (
	"os"
	"runtime"

	"portcut/internal/platform"
	residentdarwin "portcut/internal/resident/darwin"
	residentlinux "portcut/internal/resident/linux"
	residentwindows "portcut/internal/resident/windows"
)

func NewAdapter() (Adapter, error) {
	return NewAdapterFor(runtime.GOOS)
}

func NewAdapterFor(goos string) (Adapter, error) {
	return newAdapterForEnvironment(goos, currentEnvironment())
}

func newAdapterForEnvironment(goos string, env map[string]string) (Adapter, error) {
	switch goos {
	case "windows":
		return newSystrayAdapter(systrayAdapterConfig{
			title:   residentwindows.Title,
			tooltip: residentwindows.Tooltip,
			icon:    residentwindows.Icon(),
		}), nil
	case "darwin":
		return newSystrayAdapter(systrayAdapterConfig{
			title:   residentdarwin.Title,
			tooltip: residentdarwin.Tooltip,
			icon:    residentdarwin.Icon(),
		}), nil
	case "linux":
		support := residentlinux.SupportStatusFor(env)
		if !support.Supported {
			return nil, AdapterUnavailableError{GOOS: goos, Reason: support.Reason}
		}

		return newSystrayAdapter(systrayAdapterConfig{
			title:   residentlinux.Title,
			tooltip: residentlinux.Tooltip,
			icon:    residentlinux.Icon(),
		}), nil
	default:
		return nil, platform.UnsupportedPlatformError{GOOS: goos}
	}
}

func currentEnvironment() map[string]string {
	keys := []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_CURRENT_DESKTOP", "XDG_SESSION_DESKTOP", "DESKTOP_SESSION"}
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}

	return env
}
