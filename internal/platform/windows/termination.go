package windows

import "fmt"

func TerminationCommand(pid int, force bool) (string, []string) {
	command := fmt.Sprintf("Stop-Process -Id %d -ErrorAction Stop", pid)
	if force {
		command = fmt.Sprintf("Stop-Process -Id %d -Force -ErrorAction Stop", pid)
	}

	return CommandName, []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command}
}
