package resident

import (
	"context"
	"errors"
	"testing"
)

func TestNewHostRequiresSession(t *testing.T) {
	_, err := NewHost(nil)
	if err == nil {
		t.Fatal("expected host validation error")
	}
	if !errors.Is(err, ErrInvalidHost) {
		t.Fatalf("expected invalid host error, got %v", err)
	}
}

func TestHostOpenPortcutLaunchesWhenIdle(t *testing.T) {
	session := &hostSessionDouble{}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	if err := host.OpenPortcut(context.Background()); err != nil {
		t.Fatalf("expected first open to succeed, got %v", err)
	}
	if len(session.openRequests) != 1 {
		t.Fatalf("expected one open request, got %d", len(session.openRequests))
	}
	if session.openRequests[0].Reason != OpenReasonLaunch {
		t.Fatalf("expected launch open request, got %#v", session.openRequests[0])
	}
}

func TestHostOpenPortcutRequestsReopenWhileActive(t *testing.T) {
	session := &hostSessionDouble{
		snapshot: SessionSnapshot{State: SessionStateRunning, Active: true},
	}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	if err := host.OpenPortcut(context.Background()); err != nil {
		t.Fatalf("expected reopen request to succeed, got %v", err)
	}
	if len(session.openRequests) != 1 {
		t.Fatalf("expected one open request, got %d", len(session.openRequests))
	}
	if session.openRequests[0].Reason != OpenReasonReopen {
		t.Fatalf("expected reopen open request, got %#v", session.openRequests[0])
	}
}

func TestHostOpenPortcutLaunchesAgainAfterChildExit(t *testing.T) {
	session := &hostSessionDouble{}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	if err := host.OpenPortcut(context.Background()); err != nil {
		t.Fatalf("expected first open success, got %v", err)
	}
	session.snapshot = SessionSnapshot{State: SessionStateRunning, Active: true}

	session.snapshot = SessionSnapshot{State: SessionStateIdle}
	if err := host.OpenPortcut(context.Background()); err != nil {
		t.Fatalf("expected open after child exit to succeed, got %v", err)
	}
	if len(session.openRequests) != 2 {
		t.Fatalf("expected two open requests, got %d", len(session.openRequests))
	}
	if session.openRequests[1].Reason != OpenReasonLaunch {
		t.Fatalf("expected launch after exit, got %#v", session.openRequests[1])
	}
}

func TestHostQuitRequestsCoordinatedShutdownWhenChildRunning(t *testing.T) {
	session := &hostSessionDouble{
		snapshot: SessionSnapshot{State: SessionStateRunning, Active: true},
	}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	if err := host.Quit(context.Background()); err != nil {
		t.Fatalf("expected quit success, got %v", err)
	}
	if len(session.shutdownRequests) != 1 {
		t.Fatalf("expected one shutdown request, got %d", len(session.shutdownRequests))
	}
	if !session.shutdownRequests[0].Wait {
		t.Fatal("expected coordinated quit to wait for child exit")
	}
}

func TestHostQuitIsSafeWithoutRunningChild(t *testing.T) {
	session := &hostSessionDouble{}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	if err := host.Quit(context.Background()); err != nil {
		t.Fatalf("expected idle quit success, got %v", err)
	}
	if len(session.shutdownRequests) != 1 {
		t.Fatalf("expected one shutdown request, got %d", len(session.shutdownRequests))
	}
	if !session.shutdownRequests[0].Wait {
		t.Fatal("expected quit to keep wait semantics even when idle")
	}
}

type hostSessionDouble struct {
	snapshot         SessionSnapshot
	openRequests     []OpenRequest
	shutdownRequests []ShutdownRequest
	openErr          error
	shutdownErr      error
	openHook         func(OpenRequest)
	shutdownHook     func(ShutdownRequest)
	events           chan SessionEvent
}

func (s *hostSessionDouble) Open(_ context.Context, request OpenRequest) error {
	s.openRequests = append(s.openRequests, request)
	if s.openHook != nil {
		s.openHook(request)
	}

	return s.openErr
}

func (s *hostSessionDouble) Shutdown(_ context.Context, request ShutdownRequest) error {
	s.shutdownRequests = append(s.shutdownRequests, request)
	if s.shutdownHook != nil {
		s.shutdownHook(request)
	}

	return s.shutdownErr
}

func (s *hostSessionDouble) Snapshot() SessionSnapshot {
	return s.snapshot
}

func (s *hostSessionDouble) Events() <-chan SessionEvent {
	if s.events == nil {
		s.events = make(chan SessionEvent)
	}

	return s.events
}
