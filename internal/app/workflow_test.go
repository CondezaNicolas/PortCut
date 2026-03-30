package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"portcut/internal/domain"
	"portcut/internal/platform"
	platformmock "portcut/internal/platform/mock"
)

func TestWorkflowReviewTerminationDetectsStaleSelection(t *testing.T) {
	liveEntry := newEntry(3000, 101, "api")
	staleEntry := newEntry(4000, 202, "worker")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{liveEntry},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	workflow := NewWorkflow(service)
	review, err := workflow.ReviewTermination(context.Background(), domain.NewSelection(liveEntry.ID, staleEntry.ID), false)
	if err != nil {
		t.Fatalf("expected review success, got %v", err)
	}

	if review.Status != ReviewStatusStale {
		t.Fatalf("expected stale status, got %s", review.Status)
	}
	if len(review.StaleSelectionIDs) != 1 || review.StaleSelectionIDs[0] != staleEntry.ID {
		t.Fatalf("unexpected stale selection ids: %#v", review.StaleSelectionIDs)
	}
	if len(review.SelectedEntries) != 1 || review.SelectedEntries[0].ID != liveEntry.ID {
		t.Fatalf("unexpected live selection: %#v", review.SelectedEntries)
	}
	if service.TerminateCallCount != 0 {
		t.Fatalf("expected no termination calls, got %d", service.TerminateCallCount)
	}
}

func TestWorkflowRefreshEnrichesInventoryWithCategories(t *testing.T) {
	browserEntry := newEntry(9222, 333, "Google Chrome")
	nodeEntry := newEntry(3000, 101, "node")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{browserEntry, nodeEntry},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	workflow := NewWorkflow(service)
	inventory, err := workflow.Refresh(context.Background())
	if err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}

	if len(inventory.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(inventory.Entries))
	}
	if len(inventory.CategorySummaries) != len(domain.OrderedCategories()) {
		t.Fatalf("expected %d category summaries, got %d", len(domain.OrderedCategories()), len(inventory.CategorySummaries))
	}
	if len(inventory.EntriesByCategory[domain.CategoryAll]) != 2 {
		t.Fatalf("expected all category to include all entries, got %d", len(inventory.EntriesByCategory[domain.CategoryAll]))
	}
	if len(inventory.EntriesByCategory[domain.CategoryBrowsers]) != 1 || inventory.EntriesByCategory[domain.CategoryBrowsers][0].ID != browserEntry.ID {
		t.Fatalf("unexpected browser category projection: %#v", inventory.EntriesByCategory[domain.CategoryBrowsers])
	}

	browsersCount := -1
	unknownCount := -1
	for _, summary := range inventory.CategorySummaries {
		switch summary.Category {
		case domain.CategoryBrowsers:
			browsersCount = summary.Count
		case domain.CategoryUnknown:
			unknownCount = summary.Count
		}
	}
	if browsersCount != 1 {
		t.Fatalf("expected browsers summary count 1, got %d", browsersCount)
	}
	if unknownCount != 0 {
		t.Fatalf("expected unknown summary count 0, got %d", unknownCount)
	}
}

func TestWorkflowReviewTerminationUsesGlobalInventoryWithCategoryMetadata(t *testing.T) {
	browserEntry := newEntry(9222, 333, "Google Chrome")
	nodeEntry := newEntry(3000, 101, "node")
	databaseEntry := newEntry(5432, 5432, "postgres")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{browserEntry, nodeEntry, databaseEntry},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	workflow := NewWorkflow(service)
	review, err := workflow.ReviewTermination(context.Background(), domain.NewSelection(browserEntry.ID), false)
	if err != nil {
		t.Fatalf("expected review success, got %v", err)
	}

	if review.Status != ReviewStatusReady {
		t.Fatalf("expected ready review, got %s", review.Status)
	}
	if len(review.Inventory.Entries) != 3 {
		t.Fatalf("expected review inventory to retain full global entries, got %d", len(review.Inventory.Entries))
	}
	if len(review.SelectedEntries) != 1 || review.SelectedEntries[0].ID != browserEntry.ID {
		t.Fatalf("expected selected entries to follow global selection ids, got %#v", review.SelectedEntries)
	}
	if len(review.Inventory.EntriesByCategory[domain.CategoryBrowsers]) != 1 {
		t.Fatalf("expected browser category projection in review inventory, got %#v", review.Inventory.EntriesByCategory[domain.CategoryBrowsers])
	}
	if len(review.Targets) != 1 || review.Targets[0].PID != browserEntry.PID {
		t.Fatalf("expected global termination target for browser entry, got %#v", review.Targets)
	}
}

func TestWorkflowExecuteTerminationDeduplicatesTargetsAndRefreshes(t *testing.T) {
	entryA := newEntry(3000, 101, "api")
	entryB := newEntry(3001, 101, "api")
	postAction := newEntry(9090, 909, "postgres")

	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		switch service.DiscoverCallCount {
		case 1:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entryA, entryB}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		case 2:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{postAction}, CollectedAt: time.Unix(200, 0).UTC()}, nil
		default:
			return platform.DiscoverResult{}, errors.New("unexpected discover call")
		}
	}
	service.TerminateFunc = func(_ context.Context, request platform.TerminateRequest) (platform.TerminateResult, error) {
		if len(request.Targets) != 1 {
			t.Fatalf("expected one deduplicated target, got %d", len(request.Targets))
		}
		if request.Targets[0].PID != 101 {
			t.Fatalf("expected pid 101, got %d", request.Targets[0].PID)
		}
		if len(request.Targets[0].Ports) != 2 || request.Targets[0].Ports[0] != 3000 || request.Targets[0].Ports[1] != 3001 {
			t.Fatalf("unexpected deduplicated ports: %#v", request.Targets[0].Ports)
		}

		return platform.TerminateResult{
			Outcomes: []platform.TerminationOutcome{{
				Target:  request.Targets[0],
				Status:  platform.TerminationStatusCompleted,
				Kind:    platform.TerminationOutcomeKindTerminated,
				Message: "terminated",
			}},
			CompletedAt: time.Unix(150, 0).UTC(),
		}, nil
	}

	workflow := NewWorkflow(service)
	review, err := workflow.ReviewTermination(context.Background(), domain.NewSelection(entryA.ID, entryB.ID), false)
	if err != nil {
		t.Fatalf("expected review success, got %v", err)
	}
	if review.Status != ReviewStatusReady {
		t.Fatalf("expected ready review, got %s", review.Status)
	}

	execution, err := workflow.ExecuteTermination(context.Background(), review)
	if err != nil {
		t.Fatalf("expected execution success, got %v", err)
	}

	if service.DiscoverCallCount != 2 {
		t.Fatalf("expected two discover calls, got %d", service.DiscoverCallCount)
	}
	if service.TerminateCallCount != 1 {
		t.Fatalf("expected one terminate call, got %d", service.TerminateCallCount)
	}
	if len(execution.Refreshed.Entries) != 1 || execution.Refreshed.Entries[0].ID != postAction.ID {
		t.Fatalf("unexpected refreshed inventory: %#v", execution.Refreshed.Entries)
	}
}

func TestWorkflowExecuteTerminationReturnsPartialFailures(t *testing.T) {
	entryA := newEntry(3000, 101, "api")
	entryB := newEntry(4000, 202, "worker")

	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		if service.DiscoverCallCount == 1 {
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entryA, entryB}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		}

		return platform.DiscoverResult{Entries: nil, CollectedAt: time.Unix(200, 0).UTC()}, nil
	}
	service.TerminateFunc = func(_ context.Context, request platform.TerminateRequest) (platform.TerminateResult, error) {
		return platform.TerminateResult{
			Outcomes: []platform.TerminationOutcome{
				{Target: request.Targets[0], Status: platform.TerminationStatusCompleted, Kind: platform.TerminationOutcomeKindTerminated, Message: "terminated"},
				{Target: request.Targets[1], Status: platform.TerminationStatusFailed, Kind: platform.TerminationOutcomeKindPermissionDenied, Message: "permission denied"},
			},
			CompletedAt: time.Unix(150, 0).UTC(),
		}, nil
	}

	workflow := NewWorkflow(service)
	review, err := workflow.ReviewTermination(context.Background(), domain.NewSelection(entryA.ID, entryB.ID), false)
	if err != nil {
		t.Fatalf("expected review success, got %v", err)
	}

	execution, err := workflow.ExecuteTermination(context.Background(), review)
	if err != nil {
		t.Fatalf("expected execution success, got %v", err)
	}
	if !execution.Termination.HasFailures() {
		t.Fatal("expected partial failure to be surfaced in terminate result")
	}
	if len(execution.Termination.Outcomes) != 2 {
		t.Fatalf("expected two outcomes, got %d", len(execution.Termination.Outcomes))
	}
	if execution.Termination.Outcomes[1].Kind != platform.TerminationOutcomeKindPermissionDenied {
		t.Fatalf("expected permission denied outcome, got %#v", execution.Termination.Outcomes[1])
	}
}

func TestWorkflowExecuteTerminationRefreshesAfterTerminateError(t *testing.T) {
	entry := newEntry(3000, 101, "api")
	postAction := newEntry(5050, 505, "replacement")
	terminateErr := errors.New("shell failed")

	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		switch service.DiscoverCallCount {
		case 1:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		case 2:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{postAction}, CollectedAt: time.Unix(200, 0).UTC()}, nil
		default:
			return platform.DiscoverResult{}, errors.New("unexpected discover call")
		}
	}
	service.TerminateFunc = func(context.Context, platform.TerminateRequest) (platform.TerminateResult, error) {
		return platform.TerminateResult{}, terminateErr
	}

	workflow := NewWorkflow(service)
	review, err := workflow.ReviewTermination(context.Background(), domain.NewSelection(entry.ID), false)
	if err != nil {
		t.Fatalf("expected review success, got %v", err)
	}

	execution, err := workflow.ExecuteTermination(context.Background(), review)
	if !errors.Is(err, terminateErr) {
		t.Fatalf("expected terminate error, got %v", err)
	}
	if service.DiscoverCallCount != 2 {
		t.Fatalf("expected refresh after terminate error, got %d discover calls", service.DiscoverCallCount)
	}
	if len(execution.Refreshed.Entries) != 1 || execution.Refreshed.Entries[0].ID != postAction.ID {
		t.Fatalf("unexpected refreshed inventory after error: %#v", execution.Refreshed.Entries)
	}
}

func newEntry(port uint16, pid int, name string) domain.PortProcessEntry {
	return domain.NewPortProcessEntry(domain.PortProcessEntryInput{Port: port, PID: pid, ProcessName: name})
}

func supportedCapabilities() platform.Capabilities {
	return platform.Capabilities{
		Platform:            "linux",
		Discovery:           true,
		GracefulTermination: true,
		ForceTermination:    true,
		Shell:               "sh",
	}
}
