package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"portcut/internal/domain"
	"portcut/internal/platform"
)

var ErrReviewNotReady = errors.New("termination review not ready")

type Workflow struct {
	service platform.Service
}

type Inventory struct {
	Entries           []domain.PortProcessEntry
	CategorySummaries []domain.CategorySummary
	EntriesByCategory map[domain.Category][]domain.PortProcessEntry
	CollectedAt       time.Time
}

type ReviewStatus string

const (
	ReviewStatusReady       ReviewStatus = "ready"
	ReviewStatusEmpty       ReviewStatus = "empty"
	ReviewStatusStale       ReviewStatus = "stale"
	ReviewStatusUnsupported ReviewStatus = "unsupported"
)

type TerminationReview struct {
	Force             bool
	Status            ReviewStatus
	Capabilities      platform.Capabilities
	Inventory         Inventory
	SelectedEntries   []domain.PortProcessEntry
	Targets           []domain.KillTarget
	StaleSelectionIDs []string
}

type TerminationExecution struct {
	Review      TerminationReview
	Termination platform.TerminateResult
	Refreshed   Inventory
}

func NewWorkflow(service platform.Service) Workflow {
	return Workflow{service: service}
}

func (w Workflow) Refresh(ctx context.Context) (Inventory, error) {
	result, err := w.service.Discover(ctx, platform.DiscoverRequest{})
	if err != nil {
		return Inventory{}, err
	}

	entries := domain.SortEntries(result.Entries)

	return Inventory{
		Entries:           entries,
		CategorySummaries: domain.BuildCategorySummaries(entries),
		EntriesByCategory: domain.GroupEntriesByCategory(entries),
		CollectedAt:       result.CollectedAt,
	}, nil
}

func (w Workflow) ReviewTermination(ctx context.Context, selection domain.Selection, force bool) (TerminationReview, error) {
	inventory, err := w.Refresh(ctx)
	if err != nil {
		return TerminationReview{}, err
	}

	selectedEntries := domain.SelectedEntries(inventory.Entries, selection)
	targets := domain.DeduplicateKillTargets(selectedEntries)
	staleSelectionIDs := staleSelectionIDs(selection, inventory.Entries)
	status := reviewStatus(selection, targets, staleSelectionIDs, w.service.Capabilities(), force)

	return TerminationReview{
		Force:             force,
		Status:            status,
		Capabilities:      w.service.Capabilities(),
		Inventory:         inventory,
		SelectedEntries:   selectedEntries,
		Targets:           targets,
		StaleSelectionIDs: staleSelectionIDs,
	}, nil
}

func (w Workflow) ExecuteTermination(ctx context.Context, review TerminationReview) (TerminationExecution, error) {
	if review.Status != ReviewStatusReady {
		return TerminationExecution{}, fmt.Errorf("%w: %s", ErrReviewNotReady, review.Status)
	}

	termination, terminateErr := w.service.Terminate(ctx, platform.TerminateRequest{
		Targets: review.Targets,
		Force:   review.Force,
	})

	refreshed, refreshErr := w.Refresh(ctx)
	execution := TerminationExecution{
		Review:      review,
		Termination: termination,
		Refreshed:   refreshed,
	}

	if terminateErr != nil || refreshErr != nil {
		return execution, errors.Join(terminateErr, refreshErr)
	}

	return execution, nil
}

func reviewStatus(selection domain.Selection, targets []domain.KillTarget, staleSelectionIDs []string, capabilities platform.Capabilities, force bool) ReviewStatus {
	if selection.Count() == 0 || len(targets) == 0 {
		return ReviewStatusEmpty
	}
	if len(staleSelectionIDs) > 0 {
		return ReviewStatusStale
	}
	if force && !capabilities.ForceTermination {
		return ReviewStatusUnsupported
	}
	if !force && !capabilities.GracefulTermination {
		return ReviewStatusUnsupported
	}

	return ReviewStatusReady
}

func staleSelectionIDs(selection domain.Selection, entries []domain.PortProcessEntry) []string {
	liveIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		liveIDs[entry.ID] = struct{}{}
	}

	stale := make([]string, 0)
	for _, id := range selection.IDs() {
		if _, ok := liveIDs[id]; ok {
			continue
		}
		stale = append(stale, id)
	}

	return stale
}
