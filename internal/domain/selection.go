package domain

import "slices"

type Selection struct {
	selected map[string]struct{}
}

func NewSelection(ids ...string) Selection {
	selection := Selection{selected: make(map[string]struct{}, len(ids))}
	for _, id := range ids {
		if id == "" {
			continue
		}
		selection.selected[id] = struct{}{}
	}

	return selection
}

func (s *Selection) Toggle(id string) {
	if id == "" {
		return
	}
	if s.selected == nil {
		s.selected = map[string]struct{}{}
	}
	if _, ok := s.selected[id]; ok {
		delete(s.selected, id)
		return
	}

	s.selected[id] = struct{}{}
}

func (s Selection) Has(id string) bool {
	_, ok := s.selected[id]
	return ok
}

func (s Selection) IDs() []string {
	ids := make([]string, 0, len(s.selected))
	for id := range s.selected {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (s Selection) Count() int {
	return len(s.selected)
}

func SelectedEntries(entries []PortProcessEntry, selection Selection) []PortProcessEntry {
	selected := make([]PortProcessEntry, 0, selection.Count())
	for _, entry := range entries {
		if selection.Has(entry.ID) {
			selected = append(selected, entry)
		}
	}

	return SortEntries(selected)
}

func DeduplicateKillTargets(entries []PortProcessEntry) []KillTarget {
	byPID := map[int]KillTarget{}
	for _, entry := range entries {
		target, ok := entry.KillTarget()
		if !ok {
			continue
		}

		current, exists := byPID[target.PID]
		if !exists {
			byPID[target.PID] = target
			continue
		}

		current.ProcessName = target.ProcessName
		current.Ports = mergeKillTargetPorts(current.Ports, entry.Port)
		byPID[target.PID] = current
	}

	targets := make([]KillTarget, 0, len(byPID))
	for _, target := range byPID {
		targets = append(targets, target)
	}

	return SortTargets(targets)
}

func SortEntries(entries []PortProcessEntry) []PortProcessEntry {
	cloned := slices.Clone(entries)
	slices.SortFunc(cloned, compareEntries)
	return cloned
}

func SortTargets(targets []KillTarget) []KillTarget {
	cloned := slices.Clone(targets)
	slices.SortFunc(cloned, compareTargets)
	return cloned
}

func compareEntries(left, right PortProcessEntry) int {
	if left.Port != right.Port {
		if left.Port < right.Port {
			return -1
		}
		return 1
	}
	if left.PID != right.PID {
		if left.PID < right.PID {
			return -1
		}
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

func compareTargets(left, right KillTarget) int {
	if left.PID != right.PID {
		if left.PID < right.PID {
			return -1
		}
		return 1
	}
	if left.ProcessName < right.ProcessName {
		return -1
	}
	if left.ProcessName > right.ProcessName {
		return 1
	}
	return 0
}
