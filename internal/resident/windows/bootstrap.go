package windows

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
)

const BootstrapFlag = "--portcut-resident-bootstrap"
const BootstrapEnv = "PORTCUT_RESIDENT_BOOTSTRAP"

var ErrInvalidBootstrapState = errors.New("invalid bootstrap state")

type BootstrapMode int

const (
	BootstrapInline BootstrapMode = iota
	BootstrapRelaunched
)

type BootstrapConfig struct {
	GOOS       string
	Args       []string
	Env        []string
	Executable string
	Detach     func(*exec.Cmd) error
	Start      func(*exec.Cmd) error
}

type BootstrapError struct {
	Op  string
	Err error
}

func (e BootstrapError) Error() string {
	if e.Op == "" {
		return fmt.Sprintf("windows detached bootstrap: %v", e.Err)
	}

	return fmt.Sprintf("windows detached bootstrap %s: %v", e.Op, e.Err)
}

func (e BootstrapError) Unwrap() error {
	return e.Err
}

func IsBootstrapError(err error) bool {
	var target BootstrapError
	return errors.As(err, &target)
}

func EnsureDetached(config BootstrapConfig) (BootstrapMode, error) {
	goos := config.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos != "windows" {
		return BootstrapInline, nil
	}

	args := slices.Clone(config.Args)
	if len(args) == 0 {
		return BootstrapInline, BootstrapError{
			Op:  "validate launch arguments",
			Err: errors.New("missing process arguments"),
		}
	}

	hasFlag := containsBootstrapFlag(args[1:])
	hasEnv := hasBootstrapEnv(config.Env)
	if hasFlag != hasEnv {
		return BootstrapInline, BootstrapError{
			Op: "validate bootstrap markers",
			Err: fmt.Errorf(
				"%w: %s flag and %s=1 marker must be present together",
				ErrInvalidBootstrapState,
				BootstrapFlag,
				BootstrapEnv,
			),
		}
	}
	if hasFlag && hasEnv {
		return BootstrapInline, nil
	}

	if config.Executable == "" {
		return BootstrapInline, BootstrapError{
			Op:  "locate executable",
			Err: errors.New("executable path is required"),
		}
	}

	detach := config.Detach
	if detach == nil {
		detach = applyDetachedProcessAttributes
	}

	cmd := exec.Command(config.Executable, bootstrapArgs(args)...)
	cmd.Env = bootstrapEnv(config.Env)

	if err := detach(cmd); err != nil {
		return BootstrapInline, BootstrapError{Op: "configure detached launch", Err: err}
	}

	start := config.Start
	if start == nil {
		start = func(cmd *exec.Cmd) error {
			return cmd.Start()
		}
	}

	if err := start(cmd); err != nil {
		return BootstrapInline, BootstrapError{Op: "start detached resident host", Err: err}
	}

	return BootstrapRelaunched, nil
}

func bootstrapArgs(args []string) []string {
	childArgs := make([]string, 0, len(args))
	for i, arg := range args {
		if i == 0 || arg == BootstrapFlag {
			continue
		}
		childArgs = append(childArgs, arg)
	}

	return append(childArgs, BootstrapFlag)
}

func bootstrapEnv(env []string) []string {
	childEnv := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, BootstrapEnv+"=") {
			continue
		}
		childEnv = append(childEnv, entry)
	}

	return append(childEnv, BootstrapEnv+"=1")
}

func hasBootstrapEnv(env []string) bool {
	for _, entry := range env {
		if strings.EqualFold(entry, BootstrapEnv+"=1") {
			return true
		}
	}

	return false
}

func containsBootstrapFlag(args []string) bool {
	for _, arg := range args {
		if arg == BootstrapFlag {
			return true
		}
	}

	return false
}
