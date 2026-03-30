package darwin

import (
	"strconv"

	residentassets "portcut/internal/resident/assets"
)

const Title = "Portcut"
const Tooltip = "Portcut resident mode"

func Icon() []byte {
	return residentassets.TrayPNG()
}

func ForegroundCommand(pid int) (string, []string) {
	script := "tell application \"System Events\" to set frontmost of the first process whose unix id is " + strconv.Itoa(pid) + " to true"
	return "osascript", []string{"-e", script}
}

func ShutdownCommand(pid int) (string, []string) {
	return "kill", []string{"-TERM", strconv.Itoa(pid)}
}
