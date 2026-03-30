package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"portcut/internal/domain"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type Service interface {
	Discover(context.Context, DiscoverRequest) (DiscoverResult, error)
	Terminate(context.Context, TerminateRequest) (TerminateResult, error)
	Capabilities() Capabilities
}

type DiscoverRequest struct{}

type DiscoverResult struct {
	Entries     []domain.PortProcessEntry
	CollectedAt time.Time
}

type TerminateRequest struct {
	Targets []domain.KillTarget
	Force   bool
}

type TerminationStatus string

const (
	TerminationStatusCompleted TerminationStatus = "completed"
	TerminationStatusSkipped   TerminationStatus = "skipped"
	TerminationStatusFailed    TerminationStatus = "failed"
)

type TerminationOutcomeKind string

const (
	TerminationOutcomeKindTerminated       TerminationOutcomeKind = "terminated"
	TerminationOutcomeKindAlreadyExited    TerminationOutcomeKind = "already_exited"
	TerminationOutcomeKindPermissionDenied TerminationOutcomeKind = "permission_denied"
	TerminationOutcomeKindUnsupported      TerminationOutcomeKind = "unsupported"
	TerminationOutcomeKindUnknownError     TerminationOutcomeKind = "unknown_error"
)

type TerminationOutcome struct {
	Target  domain.KillTarget
	Status  TerminationStatus
	Kind    TerminationOutcomeKind
	Message string
}

type TerminateResult struct {
	Outcomes    []TerminationOutcome
	CompletedAt time.Time
}

type Capabilities struct {
	Platform            string
	Discovery           bool
	GracefulTermination bool
	ForceTermination    bool
	Shell               string
}

type UnsupportedPlatformError struct {
	GOOS string
}

type UnsupportedTerminationError struct {
	Platform string
	Force    bool
}

func (e UnsupportedPlatformError) Error() string {
	return fmt.Sprintf("%s: %s", ErrUnsupportedPlatform, e.GOOS)
}

func (e UnsupportedPlatformError) Unwrap() error {
	return ErrUnsupportedPlatform
}

func (e UnsupportedTerminationError) Error() string {
	mode := "graceful"
	if e.Force {
		mode = "force"
	}

	return fmt.Sprintf("termination mode %s unsupported on %s", mode, e.Platform)
}

func IsUnsupportedPlatform(err error) bool {
	return errors.Is(err, ErrUnsupportedPlatform)
}

func (r TerminateResult) HasFailures() bool {
	for _, outcome := range r.Outcomes {
		if outcome.Status == TerminationStatusFailed {
			return true
		}
	}

	return false
}
