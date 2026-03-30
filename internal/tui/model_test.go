package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"portcut/internal/app"
	"portcut/internal/domain"
	"portcut/internal/platform"
	platformmock "portcut/internal/platform/mock"
)

func TestModelSelectionBehavior(t *testing.T) {
	readonly := newEntry(3000, 0, "")
	selected := newEntry(4000, 101, "api")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{selected, readonly},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if model.selection.Count() != 0 {
		t.Fatalf("expected read-only row to stay unselected, got %d", model.selection.Count())
	}
	if !strings.Contains(model.status, "PID is unavailable") {
		t.Fatalf("expected read-only warning, got %q", model.status)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if !model.selection.Has(selected.ID) {
		t.Fatalf("expected selectable row %q to be selected", selected.ID)
	}
	if model.selection.Count() != 1 {
		t.Fatalf("expected one selected row, got %d", model.selection.Count())
	}
	if model.detailCursor != 1 {
		t.Fatalf("expected detail cursor to move to second row, got %d", model.detailCursor)
	}
	if !strings.Contains(model.View(), "Selections are global: 1 total, 1 shown here.") {
		t.Fatalf("expected explicit global selection cue, got %q", model.View())
	}
}

func TestModelCategoryNavigationPreservesGlobalSelection(t *testing.T) {
	browser := newEntry(9222, 303, "chrome")
	database := newEntry(5432, 202, "postgres")
	other := newEntry(8080, 404, "api")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{browser, database, other},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	model := newLoadedModel(t, service)
	if model.mode != categoryListMode {
		t.Fatalf("expected category list mode on load, got %s", model.mode)
	}
	if !strings.Contains(model.View(), "Browse categories") {
		t.Fatalf("expected category list view on load, got %q", model.View())
	}

	model = openCategory(t, model, domain.CategoryDatabases)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if !model.selection.Has(database.ID) {
		t.Fatalf("expected database row %q to be selected", database.ID)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.mode != categoryListMode {
		t.Fatalf("expected category list mode after going back, got %s", model.mode)
	}
	listView := model.View()
	if !strings.Contains(listView, "Selections stay global across categories. 1 row selected.") {
		t.Fatalf("expected global selection summary in category list, got %q", listView)
	}
	if !strings.Contains(listView, "1 selected") {
		t.Fatalf("expected selected category cue in category list, got %q", listView)
	}

	model = openCategory(t, model, domain.CategoryBrowsers)
	if !strings.Contains(model.View(), "Selections are global: 1 total, 0 shown here.") {
		t.Fatalf("expected cross-category selection cue before toggling, got %q", model.View())
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if !model.selection.Has(browser.ID) {
		t.Fatalf("expected browser row %q to be selected", browser.ID)
	}
	if model.selection.Count() != 2 {
		t.Fatalf("expected two selected rows across categories, got %d", model.selection.Count())
	}
	if !strings.Contains(model.View(), "Selections are global: 2 total, 1 shown here.") {
		t.Fatalf("expected updated global selection cue, got %q", model.View())
	}
}

func TestModelConfirmationGating(t *testing.T) {
	entry := newEntry(4000, 101, "api")
	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		switch service.DiscoverCallCount {
		case 1, 2, 3:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		case 4:
			return platform.DiscoverResult{Entries: nil, CollectedAt: time.Unix(200, 0).UTC()}, nil
		default:
			return platform.DiscoverResult{}, errors.New("unexpected discover call")
		}
	}
	service.TerminateFunc = func(context.Context, platform.TerminateRequest) (platform.TerminateResult, error) {
		return platform.TerminateResult{
			Outcomes: []platform.TerminationOutcome{{
				Target:  domain.KillTarget{PID: entry.PID, ProcessName: entry.DisplayProcessName(), Ports: []uint16{entry.Port}},
				Status:  platform.TerminationStatusCompleted,
				Kind:    platform.TerminationOutcomeKindTerminated,
				Message: "terminated",
			}},
			CompletedAt: time.Unix(150, 0).UTC(),
		}, nil
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.mode != confirmMode {
		t.Fatalf("expected confirm mode after review, got %s", model.mode)
	}
	if service.TerminateCallCount != 0 {
		t.Fatalf("expected confirm gate to block termination, got %d calls", service.TerminateCallCount)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if model.mode != categoryDetailMode {
		t.Fatalf("expected category detail mode after cancel, got %s", model.mode)
	}
	if service.TerminateCallCount != 0 {
		t.Fatalf("expected cancel to avoid termination, got %d calls", service.TerminateCallCount)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if service.TerminateCallCount != 1 {
		t.Fatalf("expected one termination call after confirm, got %d", service.TerminateCallCount)
	}
	if model.mode != categoryListMode {
		t.Fatalf("expected category list mode after execution, got %s", model.mode)
	}
	if model.selection.Count() != 0 {
		t.Fatalf("expected selection cleared after execution, got %d", model.selection.Count())
	}
	if !strings.Contains(model.status, "terminated 1 process target") {
		t.Fatalf("expected execution summary, got %q", model.status)
	}
	if len(model.inventory.Entries) != 0 {
		t.Fatalf("expected refreshed inventory to be empty, got %d rows", len(model.inventory.Entries))
	}
}

func TestModelConfirmationCancelSupportsEscape(t *testing.T) {
	entry := newEntry(4000, 101, "api")
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.mode != confirmMode {
		t.Fatalf("expected confirm mode after review, got %s", model.mode)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.mode != categoryDetailMode {
		t.Fatalf("expected escape cancel to return to category detail mode, got %s", model.mode)
	}
	if model.status != "Termination cancelled." {
		t.Fatalf("expected escape cancel status, got %q", model.status)
	}
	if !model.selection.Has(entry.ID) {
		t.Fatalf("expected selection to remain after escape cancel")
	}
}

func TestModelRefreshShowsCategoryListWhenEmpty(t *testing.T) {
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{Entries: nil, CollectedAt: time.Unix(100, 0).UTC()}, nil
		},
	}

	model := newLoadedModel(t, service)
	if service.DiscoverCallCount != 1 {
		t.Fatalf("expected one refresh on init, got %d", service.DiscoverCallCount)
	}
	if model.loading {
		t.Fatal("expected loading to finish after successful refresh")
	}
	if model.mode != categoryListMode {
		t.Fatalf("expected category list mode, got %s", model.mode)
	}
	if model.status != "No listening TCP ports found. Categories remain available; press r to refresh or q to quit." {
		t.Fatalf("unexpected empty-state status: %q", model.status)
	}

	view := model.View()
	for _, label := range []string{"All", "Node / JS", "Databases", "Browsers", "Unknown"} {
		if !strings.Contains(view, label) {
			t.Fatalf("expected empty category list to include %q, got %q", label, view)
		}
	}

	model = openCategory(t, model, domain.CategoryBrowsers)
	if !strings.Contains(model.View(), "No listening TCP rows are currently available in this category.") {
		t.Fatalf("expected empty category detail view, got %q", model.View())
	}
}

func TestModelEmptyCategoryBrowseTarget(t *testing.T) {
	entry := newEntry(5432, 202, "postgres")
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		},
	}

	model := newLoadedModel(t, service)
	if !strings.Contains(model.View(), "Browsers") || !strings.Contains(model.View(), "(empty)") {
		t.Fatalf("expected category list to show empty browse targets, got %q", model.View())
	}

	model = openCategory(t, model, domain.CategoryBrowsers)
	if model.mode != categoryDetailMode {
		t.Fatalf("expected category detail mode for empty category, got %s", model.mode)
	}
	if model.status != "Browsing Browsers. No listening TCP rows are currently available in this category." {
		t.Fatalf("unexpected empty category status: %q", model.status)
	}
	if !strings.Contains(model.View(), "No listening TCP rows are currently available in this category.") {
		t.Fatalf("expected empty category detail copy, got %q", model.View())
	}
}

func TestModelReviewUsesGlobalSelectionAcrossCategories(t *testing.T) {
	browser := newEntry(9222, 303, "chrome")
	database := newEntry(5432, 202, "postgres")
	other := newEntry(8080, 404, "api")

	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries:     []domain.PortProcessEntry{browser, database, other},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	model := newLoadedModel(t, service)
	model = openCategory(t, model, domain.CategoryDatabases)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	model = openCategory(t, model, domain.CategoryBrowsers)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if service.DiscoverCallCount != 2 {
		t.Fatalf("expected review to refresh inventory once, got %d discover calls", service.DiscoverCallCount)
	}
	if model.mode != confirmMode {
		t.Fatalf("expected confirm mode after review, got %s", model.mode)
	}
	if len(model.review.SelectedEntries) != 2 {
		t.Fatalf("expected two selected entries in review, got %#v", model.review.SelectedEntries)
	}
	if len(model.review.Targets) != 2 {
		t.Fatalf("expected two global targets in review, got %#v", model.review.Targets)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.mode != categoryDetailMode {
		t.Fatalf("expected cancel to return to category detail mode, got %s", model.mode)
	}
	if model.selectedCategory != domain.CategoryBrowsers {
		t.Fatalf("expected to return to browsers detail, got %q", model.selectedCategory)
	}
	if !strings.Contains(model.View(), "Selections are global: 2 total, 1 shown here.") {
		t.Fatalf("expected browsers detail to keep cross-category selection cue, got %q", model.View())
	}
}

func TestModelManualRefreshKeyReloadsInventory(t *testing.T) {
	initial := newEntry(4000, 101, "api")
	updated := newEntry(5000, 202, "worker")

	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		switch service.DiscoverCallCount {
		case 1:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{initial}, CollectedAt: time.Unix(100, 0).UTC()}, nil
		case 2:
			return platform.DiscoverResult{Entries: []domain.PortProcessEntry{updated}, CollectedAt: time.Unix(200, 0).UTC()}, nil
		default:
			return platform.DiscoverResult{}, errors.New("unexpected discover call")
		}
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if !model.selection.Has(initial.ID) {
		t.Fatalf("expected initial row %q to be selected", initial.ID)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if service.DiscoverCallCount != 2 {
		t.Fatalf("expected manual refresh to trigger second discover call, got %d", service.DiscoverCallCount)
	}
	if model.loading {
		t.Fatal("expected loading to finish after manual refresh")
	}
	if model.mode != categoryListMode {
		t.Fatalf("expected refresh to return to category list mode, got %s", model.mode)
	}
	if len(model.inventory.Entries) != 1 || model.inventory.Entries[0].ID != updated.ID {
		t.Fatalf("unexpected refreshed inventory: %#v", model.inventory.Entries)
	}
	if model.selection.Count() != 0 {
		t.Fatalf("expected stale selection to be dropped after refresh, got %d", model.selection.Count())
	}
	if model.status != "Loaded 1 listening TCP port row. Choose a category to browse." {
		t.Fatalf("unexpected refresh status: %q", model.status)
	}
}

func TestModelDetailViewUsesFallbackVisibleWindowBeforeResize(t *testing.T) {
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries: []domain.PortProcessEntry{
					newEntry(4000, 100, "svc-0"),
					newEntry(4001, 101, "svc-1"),
					newEntry(4002, 102, "svc-2"),
					newEntry(4003, 103, "svc-3"),
					newEntry(4004, 104, "svc-4"),
					newEntry(4005, 105, "svc-5"),
					newEntry(4006, 106, "svc-6"),
					newEntry(4007, 107, "svc-7"),
					newEntry(4008, 108, "svc-8"),
					newEntry(4009, 109, "svc-9"),
				},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	if model.detailVisibleCapacity() != defaultDetailVisibleCapacity {
		t.Fatalf("expected fallback capacity %d, got %d", defaultDetailVisibleCapacity, model.detailVisibleCapacity())
	}

	view := model.View()
	if !strings.Contains(view, "rows 1-8 of 10 (v more)") {
		t.Fatalf("expected fallback visible range cue, got %q", view)
	}
	if !strings.Contains(view, "port 4007") {
		t.Fatalf("expected last fallback-visible row in view, got %q", view)
	}
	if strings.Contains(view, "port 4008") {
		t.Fatalf("expected rows beyond fallback window to stay hidden, got %q", view)
	}

	for range 8 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}

	if model.detailCursor != 8 {
		t.Fatalf("expected absolute detail cursor 8, got %d", model.detailCursor)
	}
	if model.detailWindowStart != 1 {
		t.Fatalf("expected detail window to follow cursor, got start %d", model.detailWindowStart)
	}

	view = model.View()
	if !strings.Contains(view, "rows 2-9 of 10 (^ more, v more)") {
		t.Fatalf("expected shifted visible range cue, got %q", view)
	}
	if !strings.Contains(view, "port 4008") {
		t.Fatalf("expected cursor-follow window to reveal moved row, got %q", view)
	}
	if strings.Contains(view, "port 4000") {
		t.Fatalf("expected top row to scroll out of view, got %q", view)
	}
	if !strings.Contains(view, "> [ ] port 4008") {
		t.Fatalf("expected absolute cursor highlight on visible row, got %q", view)
	}
}

func TestModelDetailViewResizesAndClampsVisibleWindow(t *testing.T) {
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{
				Entries: []domain.PortProcessEntry{
					newEntry(5000, 200, "svc-0"),
					newEntry(5001, 201, "svc-1"),
					newEntry(5002, 202, "svc-2"),
					newEntry(5003, 203, "svc-3"),
					newEntry(5004, 204, "svc-4"),
					newEntry(5005, 205, "svc-5"),
				},
				CollectedAt: time.Unix(100, 0).UTC(),
			}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 12})
	if model.terminalHeight != 12 {
		t.Fatalf("expected terminal height to be stored, got %d", model.terminalHeight)
	}
	if model.detailVisibleCapacity() != 3 {
		t.Fatalf("expected resized capacity 3, got %d", model.detailVisibleCapacity())
	}

	view := model.View()
	if !strings.Contains(view, "rows 1-3 of 6 (v more)") {
		t.Fatalf("expected resized visible range cue, got %q", view)
	}
	if strings.Contains(view, "port 5003") {
		t.Fatalf("expected fourth row to stay hidden after resize, got %q", view)
	}

	for range 5 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}

	if model.detailCursor != 5 {
		t.Fatalf("expected absolute detail cursor 5, got %d", model.detailCursor)
	}
	if model.detailWindowStart != 3 {
		t.Fatalf("expected detail window start 3 after resize scroll, got %d", model.detailWindowStart)
	}

	view = model.View()
	if !strings.Contains(view, "rows 4-6 of 6 (^ more)") {
		t.Fatalf("expected bottom-clamped range cue, got %q", view)
	}
	if !strings.Contains(view, "> [ ] port 5005") {
		t.Fatalf("expected last row to remain visible and focused, got %q", view)
	}
	if strings.Contains(view, "port 5002") {
		t.Fatalf("expected pre-window rows to be hidden after clamp, got %q", view)
	}

	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 10})
	if model.detailVisibleCapacity() != 1 {
		t.Fatalf("expected minimum visible capacity of 1, got %d", model.detailVisibleCapacity())
	}
	if model.detailWindowStart != 5 {
		t.Fatalf("expected window start to clamp to cursor on tiny resize, got %d", model.detailWindowStart)
	}

	view = model.View()
	if !strings.Contains(view, "rows 6-6 of 6 (^ more)") {
		t.Fatalf("expected tiny-window range cue, got %q", view)
	}
	if !strings.Contains(view, "> [ ] port 5005") {
		t.Fatalf("expected focused row to remain visible in tiny window, got %q", view)
	}
}

func TestModelRefreshResetsScrolledDetailState(t *testing.T) {
	entries := entrySeries(6100, 10)
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{Entries: entries, CollectedAt: time.Unix(100, 0).UTC()}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 12})
	for range 5 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}

	if model.detailCursor != 5 || model.detailWindowStart != 3 {
		t.Fatalf("expected scrolled detail state before refresh, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if model.mode != categoryListMode {
		t.Fatalf("expected refresh to return to category list mode, got %s", model.mode)
	}
	if model.selectedCategory != domain.CategoryAll {
		t.Fatalf("expected refresh to restore all category target, got %q", model.selectedCategory)
	}
	if model.detailCursor != 0 || model.detailWindowStart != 0 {
		t.Fatalf("expected refresh to reset detail scroll state, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = openCategory(t, model, domain.CategoryAll)
	view := model.View()
	if !strings.Contains(view, "rows 1-3 of 10 (v more)") {
		t.Fatalf("expected refresh reopen to restart at top window, got %q", view)
	}
	if strings.Contains(view, "port 6103") {
		t.Fatalf("expected reopened view to hide rows beyond the reset window, got %q", view)
	}
}

func TestModelReviewTransitionClampsScrolledDetailState(t *testing.T) {
	initialEntries := entrySeries(6200, 10)
	shrunkEntries := entrySeries(6200, 4)
	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		if service.DiscoverCallCount == 1 {
			return platform.DiscoverResult{Entries: initialEntries, CollectedAt: time.Unix(100, 0).UTC()}, nil
		}
		return platform.DiscoverResult{Entries: shrunkEntries, CollectedAt: time.Unix(200, 0).UTC()}, nil
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 12})
	for range 8 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}
	model.selection = domain.NewSelection(initialEntries[3].ID)

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if service.DiscoverCallCount != 2 {
		t.Fatalf("expected review refresh discover call, got %d", service.DiscoverCallCount)
	}
	if model.mode != confirmMode {
		t.Fatalf("expected ready review to enter confirm mode, got %s", model.mode)
	}
	if model.detailCursor != 3 || model.detailWindowStart != 1 {
		t.Fatalf("expected review refresh to clamp scroll state, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}
	if model.selection.Count() != 1 || !model.selection.Has(initialEntries[3].ID) {
		t.Fatalf("expected surviving selection to remain after review refresh, got %#v", model.selection.IDs())
	}
	if !strings.Contains(model.status, "Review ready for 1 process target") {
		t.Fatalf("expected review-ready status, got %q", model.status)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.mode != categoryDetailMode {
		t.Fatalf("expected cancel to return to detail view, got %s", model.mode)
	}

	view := model.View()
	if !strings.Contains(view, "rows 2-4 of 4 (^ more)") {
		t.Fatalf("expected clamped visible range after review refresh, got %q", view)
	}
	if !strings.Contains(view, "> [x] port 6203") {
		t.Fatalf("expected last remaining row to stay focused after clamp, got %q", view)
	}
}

func TestModelExecutionTransitionResetsScrolledDetailState(t *testing.T) {
	entries := entrySeries(6300, 6)
	service := &platformmock.Service{CapabilitiesValue: supportedCapabilities()}
	service.DiscoverFunc = func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
		switch service.DiscoverCallCount {
		case 1, 2:
			return platform.DiscoverResult{Entries: entries, CollectedAt: time.Unix(100, 0).UTC()}, nil
		case 3:
			return platform.DiscoverResult{Entries: nil, CollectedAt: time.Unix(200, 0).UTC()}, nil
		default:
			return platform.DiscoverResult{}, errors.New("unexpected discover call")
		}
	}
	service.TerminateFunc = func(context.Context, platform.TerminateRequest) (platform.TerminateResult, error) {
		return platform.TerminateResult{
			Outcomes: []platform.TerminationOutcome{{
				Target:  domain.KillTarget{PID: entries[5].PID, ProcessName: entries[5].DisplayProcessName(), Ports: []uint16{entries[5].Port}},
				Status:  platform.TerminationStatusCompleted,
				Kind:    platform.TerminationOutcomeKindTerminated,
				Message: "terminated",
			}},
			CompletedAt: time.Unix(150, 0).UTC(),
		}, nil
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 12})
	for range 5 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}
	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
	if model.detailCursor != 5 || model.detailWindowStart != 3 {
		t.Fatalf("expected scrolled detail state before execution review, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.mode != confirmMode {
		t.Fatalf("expected confirm mode before execution, got %s", model.mode)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if service.TerminateCallCount != 1 {
		t.Fatalf("expected one termination call, got %d", service.TerminateCallCount)
	}
	if service.DiscoverCallCount != 3 {
		t.Fatalf("expected init, review, and execution refresh discovers, got %d", service.DiscoverCallCount)
	}
	if model.mode != categoryListMode {
		t.Fatalf("expected execution to return to category list mode, got %s", model.mode)
	}
	if model.selectedCategory != domain.CategoryAll {
		t.Fatalf("expected execution to reset selected category, got %q", model.selectedCategory)
	}
	if model.detailCursor != 0 || model.detailWindowStart != 0 {
		t.Fatalf("expected execution to reset detail scroll state, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}
	if model.selection.Count() != 0 {
		t.Fatalf("expected execution to clear selection, got %d", model.selection.Count())
	}
	if !strings.Contains(model.status, "terminated 1 process target") {
		t.Fatalf("expected execution summary status, got %q", model.status)
	}
}

func TestModelDetailViewSinglePageAndScrollEdgesStayStable(t *testing.T) {
	entries := entrySeries(6400, 2)
	service := &platformmock.Service{
		CapabilitiesValue: supportedCapabilities(),
		DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
			return platform.DiscoverResult{Entries: entries, CollectedAt: time.Unix(100, 0).UTC()}, nil
		},
	}

	model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 20})

	view := model.View()
	if !strings.Contains(view, "rows 1-2 of 2") {
		t.Fatalf("expected single-page range label, got %q", view)
	}
	if strings.Contains(view, "more") {
		t.Fatalf("expected no overflow cues when all rows fit, got %q", view)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyUp})
	if model.detailCursor != 0 || model.detailWindowStart != 0 {
		t.Fatalf("expected top-edge up movement to stay clamped, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	for range 5 {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}
	if model.detailCursor != 1 || model.detailWindowStart != 0 {
		t.Fatalf("expected bottom-edge movement to clamp without scrolling, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 10})
	if model.detailVisibleCapacity() != 1 {
		t.Fatalf("expected tiny resize capacity 1, got %d", model.detailVisibleCapacity())
	}
	if model.detailWindowStart != 1 {
		t.Fatalf("expected tiny resize to follow bottom cursor, got start=%d", model.detailWindowStart)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.detailCursor != 1 || model.detailWindowStart != 1 {
		t.Fatalf("expected bottom-edge down movement to remain clamped, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyUp})
	if model.detailCursor != 0 || model.detailWindowStart != 0 {
		t.Fatalf("expected moving back up to restore top window, got cursor=%d start=%d", model.detailCursor, model.detailWindowStart)
	}

	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})
	if model.detailWindowStart != 0 {
		t.Fatalf("expected large resize to clamp single-page window back to zero, got start=%d", model.detailWindowStart)
	}
}

func TestModelEmptyAndErrorStates(t *testing.T) {
	t.Run("empty review requires selection", func(t *testing.T) {
		entry := newEntry(4000, 101, "api")
		service := &platformmock.Service{
			CapabilitiesValue: supportedCapabilities(),
			DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
				return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
			},
		}

		model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
		if model.mode != categoryDetailMode {
			t.Fatalf("expected to stay in category detail mode, got %s", model.mode)
		}
		if service.TerminateCallCount != 0 {
			t.Fatalf("expected no termination calls, got %d", service.TerminateCallCount)
		}
		if !strings.Contains(model.status, "Select at least one terminable row") {
			t.Fatalf("expected empty selection message, got %q", model.status)
		}
	})

	t.Run("unsupported graceful review suggests force mode", func(t *testing.T) {
		entry := newEntry(4000, 101, "api")
		service := &platformmock.Service{
			CapabilitiesValue: platform.Capabilities{
				Platform:            "windows",
				Discovery:           true,
				GracefulTermination: false,
				ForceTermination:    true,
				Shell:               "powershell",
			},
			DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
				return platform.DiscoverResult{Entries: []domain.PortProcessEntry{entry}, CollectedAt: time.Unix(100, 0).UTC()}, nil
			},
		}

		model := openCategory(t, newLoadedModel(t, service), domain.CategoryAll)
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeySpace})
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
		if service.DiscoverCallCount != 1 {
			t.Fatalf("expected unsupported graceful review to avoid extra refresh, got %d discover calls", service.DiscoverCallCount)
		}
		if !strings.Contains(model.status, "Press f to review force termination") {
			t.Fatalf("expected force hint in status, got %q", model.status)
		}

		view := model.View()
		if strings.Contains(view, "enter review") {
			t.Fatalf("expected windows help to hide graceful review shortcut, got %q", view)
		}
		if !strings.Contains(view, "f review force") {
			t.Fatalf("expected windows help to keep force shortcut, got %q", view)
		}
		if !strings.Contains(view, "termination: force") {
			t.Fatalf("expected capability summary in view, got %q", view)
		}
	})

	t.Run("refresh error is surfaced", func(t *testing.T) {
		service := &platformmock.Service{
			CapabilitiesValue: supportedCapabilities(),
			DiscoverFunc: func(context.Context, platform.DiscoverRequest) (platform.DiscoverResult, error) {
				return platform.DiscoverResult{}, errors.New("boom")
			},
		}

		model := NewModel(app.NewWorkflow(service), service.CapabilitiesValue)
		model = runCmdUpdate(t, model, model.Init())
		if !strings.Contains(model.status, "Refresh failed: boom") {
			t.Fatalf("expected refresh error in status, got %q", model.status)
		}
		if len(model.inventory.Entries) != 0 {
			t.Fatalf("expected no inventory on refresh error, got %d rows", len(model.inventory.Entries))
		}
	})
}

func newLoadedModel(t *testing.T, service *platformmock.Service) Model {
	t.Helper()

	model := NewModel(app.NewWorkflow(service), service.CapabilitiesValue)
	return runCmdUpdate(t, model, model.Init())
}

func openCategory(t *testing.T, model Model, category domain.Category) Model {
	t.Helper()

	index := -1
	for i, summary := range model.categorySummaries() {
		if summary.Category == category {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatalf("category %q not found in summaries: %#v", category, model.categorySummaries())
	}

	for model.categoryCursor < index {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	}
	for model.categoryCursor > index {
		model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyUp})
	}

	model = keyUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.mode != categoryDetailMode {
		t.Fatalf("expected category detail mode after opening %q, got %s", category, model.mode)
	}
	if model.selectedCategory != category {
		t.Fatalf("expected selected category %q, got %q", category, model.selectedCategory)
	}

	return model
}

func keyUpdate(t *testing.T, model Model, msg tea.KeyMsg) Model {
	t.Helper()

	updated, cmd := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model update result, got %T", updated)
	}
	if cmd == nil {
		return next
	}

	return runCmdUpdate(t, next, cmd)
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()

	updated, cmd := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model update result, got %T", updated)
	}
	if cmd != nil {
		t.Fatalf("expected no follow-up command for %T, got one", msg)
	}

	return next
}

func runCmdUpdate(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()

	if cmd == nil {
		return model
	}

	msg := cmd()
	updated, nextCmd := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model update result, got %T", updated)
	}
	if nextCmd != nil {
		t.Fatalf("expected command chain to settle, got follow-up command")
	}

	return next
}

func newEntry(port uint16, pid int, name string) domain.PortProcessEntry {
	return domain.NewPortProcessEntry(domain.PortProcessEntryInput{Port: port, PID: pid, ProcessName: name})
}

func entrySeries(startPort uint16, count int) []domain.PortProcessEntry {
	entries := make([]domain.PortProcessEntry, 0, count)
	for i := range count {
		entries = append(entries, newEntry(startPort+uint16(i), 100+i, fmt.Sprintf("svc-%d", i)))
	}
	return entries
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
