//go:build !windows

package windows

import (
	"errors"
	"os/exec"
)

func applyDetachedProcessAttributes(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("command is required")
	}

	return errors.New("windows detached launch attributes require a windows build")
}
