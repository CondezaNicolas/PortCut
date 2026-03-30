package mock

import (
	"context"

	"portcut/internal/platform"
)

type Service struct {
	DiscoverFunc       func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error)
	TerminateFunc      func(context.Context, platform.TerminateRequest) (platform.TerminateResult, error)
	CapabilitiesValue  platform.Capabilities
	DiscoverRequests   []platform.DiscoverRequest
	TerminateRequests  []platform.TerminateRequest
	DiscoverCallCount  int
	TerminateCallCount int
}

func (s *Service) Discover(ctx context.Context, request platform.DiscoverRequest) (platform.DiscoverResult, error) {
	s.DiscoverCallCount++
	s.DiscoverRequests = append(s.DiscoverRequests, request)
	if s.DiscoverFunc != nil {
		return s.DiscoverFunc(ctx, request)
	}
	return platform.DiscoverResult{}, nil
}

func (s *Service) Terminate(ctx context.Context, request platform.TerminateRequest) (platform.TerminateResult, error) {
	s.TerminateCallCount++
	s.TerminateRequests = append(s.TerminateRequests, request)
	if s.TerminateFunc != nil {
		return s.TerminateFunc(ctx, request)
	}
	return platform.TerminateResult{}, nil
}

func (s *Service) Capabilities() platform.Capabilities {
	return s.CapabilitiesValue
}
