package platform

import (
	"runtime"

	"portcut/internal/domain"
	"portcut/internal/platform/darwin"
	linuxdiscovery "portcut/internal/platform/linux"
	windowsdiscovery "portcut/internal/platform/windows"
)

func NewService() (Service, error) {
	return NewServiceFor(runtime.GOOS)
}

func NewServiceFor(goos string) (Service, error) {
	switch goos {
	case "linux":
		return newCommandService(
			Capabilities{
				Platform:            "linux",
				Discovery:           true,
				GracefulTermination: true,
				ForceTermination:    true,
				Shell:               "sh",
			},
			commandSpec{Name: linuxdiscovery.CommandName, Args: linuxdiscovery.CommandArgs()},
			linuxdiscovery.ParseDiscoveryOutput,
			func(target domain.KillTarget, force bool) commandSpec {
				name, args := linuxdiscovery.TerminationCommand(target.PID, force)
				return terminateCommandSpec(name, args)
			},
		), nil
	case "darwin":
		return newCommandService(
			Capabilities{
				Platform:            "darwin",
				Discovery:           true,
				GracefulTermination: true,
				ForceTermination:    true,
				Shell:               "sh",
			},
			commandSpec{Name: darwin.CommandName, Args: darwin.CommandArgs()},
			darwin.ParseDiscoveryOutput,
			func(target domain.KillTarget, force bool) commandSpec {
				name, args := darwin.TerminationCommand(target.PID, force)
				return terminateCommandSpec(name, args)
			},
		), nil
	case "windows":
		return newCommandService(
			Capabilities{
				Platform:            "windows",
				Discovery:           true,
				GracefulTermination: false,
				ForceTermination:    true,
				Shell:               "powershell",
			},
			commandSpec{Name: windowsdiscovery.CommandName, Args: windowsdiscovery.CommandArgs()},
			windowsdiscovery.ParseDiscoveryOutput,
			func(target domain.KillTarget, force bool) commandSpec {
				name, args := windowsdiscovery.TerminationCommand(target.PID, force)
				return terminateCommandSpec(name, args)
			},
		), nil
	default:
		return nil, UnsupportedPlatformError{GOOS: goos}
	}
}
