//go:build windows

package windows

import (
	"errors"
	"os/exec"
	"syscall"

	win "golang.org/x/sys/windows"
)

func applyDetachedProcessAttributes(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("command is required")
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: win.CREATE_NEW_PROCESS_GROUP | win.DETACHED_PROCESS,
	}

	return nil
}
