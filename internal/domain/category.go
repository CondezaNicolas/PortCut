package domain

import (
	"slices"
	"strings"
	"unicode"
)

type Category string

const (
	CategoryAll            Category = "all"
	CategoryNodeJS         Category = "node-js"
	CategorySystem         Category = "system"
	CategoryContainersWSL  Category = "containers-wsl"
	CategoryDatabases      Category = "databases"
	CategoryBrowsers       Category = "browsers"
	CategoryServersProxies Category = "servers-proxies"
	CategoryOther          Category = "other"
	CategoryUnknown        Category = "unknown"
)

type CategorySummary struct {
	Category Category
	Label    string
	Count    int
}

var orderedCategories = []Category{
	CategoryAll,
	CategoryNodeJS,
	CategorySystem,
	CategoryContainersWSL,
	CategoryDatabases,
	CategoryBrowsers,
	CategoryServersProxies,
	CategoryOther,
	CategoryUnknown,
}

var categoryLabels = map[Category]string{
	CategoryAll:            "All",
	CategoryNodeJS:         "Node / JS",
	CategorySystem:         "System",
	CategoryContainersWSL:  "Containers / WSL",
	CategoryDatabases:      "Databases",
	CategoryBrowsers:       "Browsers",
	CategoryServersProxies: "Servers / Proxies",
	CategoryOther:          "Other",
	CategoryUnknown:        "Unknown",
}

var categoryHeuristics = map[Category]map[string]struct{}{
	CategoryNodeJS: {
		"bun":     {},
		"deno":    {},
		"node":    {},
		"nodejs":  {},
		"npm":     {},
		"pnpm":    {},
		"vite":    {},
		"webpack": {},
		"yarn":    {},
	},
	CategorySystem: {
		"init":            {},
		"kerneltask":      {},
		"launchd":         {},
		"mdnsresponder":   {},
		"services":        {},
		"svchost":         {},
		"system":          {},
		"systemd":         {},
		"systemdresolved": {},
	},
	CategoryContainersWSL: {
		"colima":         {},
		"containerd":     {},
		"containerdshim": {},
		"docker":         {},
		"dockercompose":  {},
		"dockerdesktop":  {},
		"dockerd":        {},
		"podman":         {},
		"podmanmachine":  {},
		"rancherdesktop": {},
		"vmmem":          {},
		"vmmemwsl":       {},
		"wsl":            {},
		"wslhost":        {},
	},
	CategoryDatabases: {
		"mariadbd":    {},
		"mongod":      {},
		"mysql":       {},
		"mysqld":      {},
		"postgres":    {},
		"postgresql":  {},
		"redis":       {},
		"redisserver": {},
		"valkey":      {},
	},
	CategoryBrowsers: {
		"arc":           {},
		"brave":         {},
		"bravebrowser":  {},
		"chrome":        {},
		"chromium":      {},
		"firefox":       {},
		"googlechrome":  {},
		"microsoftedge": {},
		"msedge":        {},
		"opera":         {},
		"safari":        {},
	},
	CategoryServersProxies: {
		"apache":  {},
		"caddy":   {},
		"envoy":   {},
		"haproxy": {},
		"httpd":   {},
		"nginx":   {},
		"traefik": {},
	},
}

func OrderedCategories() []Category {
	return slices.Clone(orderedCategories)
}

func (c Category) Label() string {
	if label, ok := categoryLabels[c]; ok {
		return label
	}

	return categoryLabels[CategoryUnknown]
}

func ClassifyProcessName(name string) Category {
	normalized := normalizeProcessName(name)
	if normalized == "" {
		return CategoryUnknown
	}

	for _, category := range orderedCategories {
		if category == CategoryAll || category == CategoryOther || category == CategoryUnknown {
			continue
		}
		if _, ok := categoryHeuristics[category][normalized]; ok {
			return category
		}
	}

	return CategoryOther
}

func CategoryForEntry(entry PortProcessEntry) Category {
	return ClassifyProcessName(entry.ProcessName)
}

func BuildCategorySummaries(entries []PortProcessEntry) []CategorySummary {
	grouped := GroupEntriesByCategory(entries)
	summaries := make([]CategorySummary, 0, len(orderedCategories))
	for _, category := range orderedCategories {
		summaries = append(summaries, CategorySummary{
			Category: category,
			Label:    category.Label(),
			Count:    len(grouped[category]),
		})
	}

	return summaries
}

func GroupEntriesByCategory(entries []PortProcessEntry) map[Category][]PortProcessEntry {
	sorted := SortEntries(entries)
	grouped := make(map[Category][]PortProcessEntry, len(orderedCategories))
	for _, category := range orderedCategories {
		grouped[category] = make([]PortProcessEntry, 0)
	}

	grouped[CategoryAll] = slices.Clone(sorted)
	for _, entry := range sorted {
		category := CategoryForEntry(entry)
		grouped[category] = append(grouped[category], entry)
	}

	return grouped
}

func EntriesForCategory(entries []PortProcessEntry, category Category) []PortProcessEntry {
	grouped := GroupEntriesByCategory(entries)
	return slices.Clone(grouped[category])
}

func normalizeProcessName(name string) string {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = strings.TrimSuffix(trimmed, ".exe")

	var normalized strings.Builder
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		}
	}

	return normalized.String()
}
