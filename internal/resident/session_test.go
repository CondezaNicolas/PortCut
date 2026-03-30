package resident

import (
	"context"
	"errors"
	"testing"
)

func TestSessionTrackerRejectsSecondLaunchButAllowsReopen(t *testing.T) {
	tracker := NewSessionTracker()

	if err := tracker.Open(context.Background(), OpenRequest{Reason: OpenReasonLaunch}); err != nil {
		t.Fatalf("expected first launch success, got %v", err)
	}
	<-tracker.Events()

	err := tracker.Open(context.Background(), OpenRequest{Reason: OpenReasonLaunch})
	if !errors.Is(err, ErrSessionAlreadyRunning) {
		t.Fatalf("expected already-running error, got %v", err)
	}

	if err := tracker.Open(context.Background(), OpenRequest{Reason: OpenReasonReopen}); err != nil {
		t.Fatalf("expected reopen success, got %v", err)
	}
	event := <-tracker.Events()
	if event.Type != SessionEventReopenRequested {
		t.Fatalf("expected reopen event, got %#v", event)
	}
}

func TestSessionTrackerMarksExitAndPublishesFailure(t *testing.T) {
	tracker := NewSessionTracker()
	if err := tracker.Open(context.Background(), OpenRequest{}); err != nil {
		t.Fatalf("expected open success, got %v", err)
	}
	<-tracker.Events()

	wantErr := errors.New("child exited")
	tracker.MarkExited(wantErr)
	event := <-tracker.Events()
	if event.Type != SessionEventExited {
		t.Fatalf("expected exited event, got %#v", event)
	}
	if !errors.Is(event.Err, wantErr) {
		t.Fatalf("expected exit error propagation, got %v", event.Err)
	}

	snapshot := tracker.Snapshot()
	if snapshot.State != SessionStateIdle || snapshot.Active {
		t.Fatalf("expected idle snapshot after exit, got %#v", snapshot)
	}
}
