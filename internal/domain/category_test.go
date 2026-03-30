package domain

import (
	"reflect"
	"testing"
)

func TestClassifyProcessNameMapsKnownHeuristics(t *testing.T) {
	tests := map[string]Category{
		" node ":         CategoryNodeJS,
		"svchost.exe":    CategorySystem,
		"docker-desktop": CategoryContainersWSL,
		"postgres":       CategoryDatabases,
		"Google Chrome":  CategoryBrowsers,
		"nginx":          CategoryServersProxies,
		"my-custom-proc": CategoryOther,
	}

	for processName, want := range tests {
		if got := ClassifyProcessName(processName); got != want {
			t.Fatalf("expected %q to map to %q, got %q", processName, want, got)
		}
	}
}

func TestClassifyProcessNameFallsBackToUnknownForUnusableNames(t *testing.T) {
	tests := []string{"", "   ", "---", "...", "\"\""}

	for _, processName := range tests {
		if got := ClassifyProcessName(processName); got != CategoryUnknown {
			t.Fatalf("expected %q to map to unknown, got %q", processName, got)
		}
	}
}

func TestBuildCategorySummariesIncludesFullMVPSetInOrder(t *testing.T) {
	entries := []PortProcessEntry{
		NewPortProcessEntry(PortProcessEntryInput{Port: 5432, PID: 5432, ProcessName: "postgres"}),
		NewPortProcessEntry(PortProcessEntryInput{Port: 8080, PID: 1001, ProcessName: "node"}),
	}

	summaries := BuildCategorySummaries(entries)
	if len(summaries) != len(OrderedCategories()) {
		t.Fatalf("expected %d summaries, got %d", len(OrderedCategories()), len(summaries))
	}

	gotOrder := make([]Category, 0, len(summaries))
	counts := make(map[Category]int, len(summaries))
	for _, summary := range summaries {
		gotOrder = append(gotOrder, summary.Category)
		counts[summary.Category] = summary.Count
	}

	if wantOrder := OrderedCategories(); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("expected summary order %v, got %v", wantOrder, gotOrder)
	}
	if counts[CategoryAll] != 2 {
		t.Fatalf("expected all count 2, got %d", counts[CategoryAll])
	}
	if counts[CategoryNodeJS] != 1 {
		t.Fatalf("expected node/js count 1, got %d", counts[CategoryNodeJS])
	}
	if counts[CategoryDatabases] != 1 {
		t.Fatalf("expected databases count 1, got %d", counts[CategoryDatabases])
	}
	if counts[CategoryBrowsers] != 0 {
		t.Fatalf("expected browsers count 0, got %d", counts[CategoryBrowsers])
	}
}

func TestEntriesForCategoryReturnsDeterministicSortedProjection(t *testing.T) {
	entryB := NewPortProcessEntry(PortProcessEntryInput{Port: 8081, PID: 1001, ProcessName: "node"})
	entryA := NewPortProcessEntry(PortProcessEntryInput{Port: 8080, PID: 1001, ProcessName: "node"})
	entryC := NewPortProcessEntry(PortProcessEntryInput{Port: 3000, PID: 2002, ProcessName: "postgres"})

	entries := []PortProcessEntry{entryB, entryC, entryA}

	projected := EntriesForCategory(entries, CategoryNodeJS)
	if len(projected) != 2 {
		t.Fatalf("expected 2 projected entries, got %d", len(projected))
	}
	if projected[0].ID != entryA.ID || projected[1].ID != entryB.ID {
		t.Fatalf("expected deterministic sort order by stable ids, got %#v", projected)
	}

	allEntries := EntriesForCategory(entries, CategoryAll)
	if len(allEntries) != 3 {
		t.Fatalf("expected all category to retain all entries, got %d", len(allEntries))
	}
}
