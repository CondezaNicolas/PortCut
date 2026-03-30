package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"portcut/internal/domain"
)

type commandSpec struct {
	Name string
	Args []string
}

type discoverParser func([]byte) ([]domain.PortProcessEntry, error)

type terminateSpecBuilder func(domain.KillTarget, bool) commandSpec

type commandRunner func(context.Context, commandSpec) ([]byte, error)

type commandService struct {
	capabilities   Capabilities
	discoverSpec   commandSpec
	parseDiscover  discoverParser
	buildTerminate terminateSpecBuilder
	run            commandRunner
	now            func() time.Time
}

func newCommandService(capabilities Capabilities, discoverSpec commandSpec, parse discoverParser, buildTerminate terminateSpecBuilder) Service {
	return commandService{
		capabilities:   capabilities,
		discoverSpec:   discoverSpec,
		parseDiscover:  parse,
		buildTerminate: buildTerminate,
		run:            defaultCommandRunner,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (s commandService) Discover(ctx context.Context, _ DiscoverRequest) (DiscoverResult, error) {
	output, err := s.run(ctx, s.discoverSpec)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return DiscoverResult{}, fmt.Errorf("discover ports with %s: %w: %s", s.discoverSpec.Name, err, message)
		}

		return DiscoverResult{}, fmt.Errorf("discover ports with %s: %w", s.discoverSpec.Name, err)
	}

	entries, err := s.parseDiscover(output)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("parse discovery output from %s: %w", s.discoverSpec.Name, err)
	}

	return DiscoverResult{
		Entries:     domain.SortEntries(entries),
		CollectedAt: s.now(),
	}, nil
}

func (s commandService) Terminate(ctx context.Context, request TerminateRequest) (TerminateResult, error) {
	outcomes := make([]TerminationOutcome, 0, len(request.Targets))
	if !request.Force && !s.capabilities.GracefulTermination {
		for _, target := range request.Targets {
			outcomes = append(outcomes, unsupportedTerminationOutcome(target, s.capabilities.Platform, request.Force))
		}

		return TerminateResult{Outcomes: outcomes, CompletedAt: s.now()}, nil
	}
	if request.Force && !s.capabilities.ForceTermination {
		for _, target := range request.Targets {
			outcomes = append(outcomes, unsupportedTerminationOutcome(target, s.capabilities.Platform, request.Force))
		}

		return TerminateResult{Outcomes: outcomes, CompletedAt: s.now()}, nil
	}

	for _, target := range domain.SortTargets(request.Targets) {
		spec := s.buildTerminate(target, request.Force)
		output, err := s.run(ctx, spec)
		outcomes = append(outcomes, classifyTerminationOutcome(target, request.Force, output, err))
	}

	return TerminateResult{Outcomes: outcomes, CompletedAt: s.now()}, nil
}

func (s commandService) Capabilities() Capabilities {
	return s.capabilities
}

func unsupportedTerminationOutcome(target domain.KillTarget, platform string, force bool) TerminationOutcome {
	mode := "graceful"
	if force {
		mode = "force"
	}

	return TerminationOutcome{
		Target:  target,
		Status:  TerminationStatusSkipped,
		Kind:    TerminationOutcomeKindUnsupported,
		Message: fmt.Sprintf("%s termination unsupported on %s", mode, platform),
	}
}

func classifyTerminationOutcome(target domain.KillTarget, force bool, output []byte, err error) TerminationOutcome {
	if err == nil {
		message := "terminated"
		if force {
			message = "force terminated"
		}

		return TerminationOutcome{
			Target:  target,
			Status:  TerminationStatusCompleted,
			Kind:    TerminationOutcomeKindTerminated,
			Message: message,
		}
	}

	message := strings.TrimSpace(string(output))
	if message == "" {
		message = strings.TrimSpace(err.Error())
	}

	text := strings.ToLower(strings.Join([]string{message, err.Error()}, " "))
	switch {
	case strings.Contains(text, "no such process"),
		strings.Contains(text, "cannot find a process"),
		strings.Contains(text, "process with process id"),
		strings.Contains(text, "not found"):
		return TerminationOutcome{
			Target:  target,
			Status:  TerminationStatusSkipped,
			Kind:    TerminationOutcomeKindAlreadyExited,
			Message: message,
		}
	case strings.Contains(text, "permission denied"),
		strings.Contains(text, "operation not permitted"),
		strings.Contains(text, "access is denied"):
		return TerminationOutcome{
			Target:  target,
			Status:  TerminationStatusFailed,
			Kind:    TerminationOutcomeKindPermissionDenied,
			Message: message,
		}
	default:
		return TerminationOutcome{
			Target:  target,
			Status:  TerminationStatusFailed,
			Kind:    TerminationOutcomeKindUnknownError,
			Message: message,
		}
	}
}

func terminateCommandSpec(name string, args []string) commandSpec {
	return commandSpec{Name: name, Args: args}
}

func defaultCommandRunner(ctx context.Context, spec commandSpec) ([]byte, error) {
	return exec.CommandContext(ctx, spec.Name, spec.Args...).CombinedOutput()
}
