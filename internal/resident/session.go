package resident

import (
	"context"
	"errors"
	"sync"
)

var ErrSessionAlreadyRunning = errors.New("resident session already running")
var ErrSessionStopping = errors.New("resident session is stopping")

type OpenReason string

const (
	OpenReasonLaunch OpenReason = "launch"
	OpenReasonReopen OpenReason = "reopen"
)

type SessionState string

const (
	SessionStateIdle     SessionState = "idle"
	SessionStateRunning  SessionState = "running"
	SessionStateStopping SessionState = "stopping"
)

type OpenRequest struct {
	Reason OpenReason
}

type ShutdownRequest struct {
	Wait bool
}

type SessionSnapshot struct {
	State SessionState
	Open  OpenRequest

	Active bool
}

type Session interface {
	Open(context.Context, OpenRequest) error
	Shutdown(context.Context, ShutdownRequest) error
	Snapshot() SessionSnapshot
	Events() <-chan SessionEvent
}

type SessionEventType string

const (
	SessionEventOpened            SessionEventType = "opened"
	SessionEventReopenRequested   SessionEventType = "reopen_requested"
	SessionEventShutdownRequested SessionEventType = "shutdown_requested"
	SessionEventExited            SessionEventType = "exited"
)

type SessionEvent struct {
	Type SessionEventType
	Open OpenRequest
	Err  error
}

type SessionTracker struct {
	mu       sync.RWMutex
	snapshot SessionSnapshot
	events   chan SessionEvent
}

func NewSessionTracker() *SessionTracker {
	return &SessionTracker{
		snapshot: SessionSnapshot{State: SessionStateIdle},
		events:   make(chan SessionEvent, 8),
	}
}

func (t *SessionTracker) Open(_ context.Context, request OpenRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if request.Reason == "" {
		request.Reason = OpenReasonLaunch
	}

	if t.snapshot.Active {
		if request.Reason == OpenReasonReopen {
			t.publishLocked(SessionEvent{Type: SessionEventReopenRequested, Open: request})
			return nil
		}

		return ErrSessionAlreadyRunning
	}

	t.snapshot = SessionSnapshot{
		State:  SessionStateRunning,
		Open:   request,
		Active: true,
	}
	t.publishLocked(SessionEvent{Type: SessionEventOpened, Open: request})

	return nil
}

func (t *SessionTracker) Shutdown(_ context.Context, request ShutdownRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.snapshot.Active {
		return nil
	}

	t.snapshot.State = SessionStateStopping
	t.publishLocked(SessionEvent{Type: SessionEventShutdownRequested})
	if !request.Wait {
		t.snapshot = SessionSnapshot{State: SessionStateIdle}
		t.publishLocked(SessionEvent{Type: SessionEventExited})
	}

	return nil
}

func (t *SessionTracker) MarkExited(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.snapshot = SessionSnapshot{State: SessionStateIdle}
	t.publishLocked(SessionEvent{Type: SessionEventExited, Err: err})
}

func (t *SessionTracker) Snapshot() SessionSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.snapshot
}

func (t *SessionTracker) Events() <-chan SessionEvent {
	return t.events
}

func (t *SessionTracker) publishLocked(event SessionEvent) {
	select {
	case t.events <- event:
	default:
	}
}
