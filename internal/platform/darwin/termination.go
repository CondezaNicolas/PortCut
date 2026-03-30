package darwin

import "strconv"

func TerminationCommand(pid int, force bool) (string, []string) {
	signal := "-TERM"
	if force {
		signal = "-KILL"
	}

	return "kill", []string{signal, strconv.Itoa(pid)}
}
