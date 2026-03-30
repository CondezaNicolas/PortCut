package resident

import (
	"context"
	"errors"
	"fmt"

	"portcut/internal/platform"
)

var ErrAdapterUnavailable = errors.New("resident adapter unavailable")

type Adapter interface {
	Run(context.Context, Host) error
}

type AdapterUnavailableError struct {
	GOOS   string
	Reason string
}

func (e AdapterUnavailableError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s: %s: %s", ErrAdapterUnavailable, e.GOOS, e.Reason)
	}

	return fmt.Sprintf("%s: %s", ErrAdapterUnavailable, e.GOOS)
}

func (e AdapterUnavailableError) Unwrap() error {
	return ErrAdapterUnavailable
}

func IsAdapterUnavailable(err error) bool {
	return errors.Is(err, ErrAdapterUnavailable)
}

func IsUnsupportedPlatform(err error) bool {
	return platform.IsUnsupportedPlatform(err)
}
