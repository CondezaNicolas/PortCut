package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"portcut/internal/app"
	"portcut/internal/domain"
	"portcut/internal/platform"
)

type viewMode string

const (
	categoryListMode   viewMode = "category-list"
	categoryDetailMode viewMode = "category-detail"
	confirmMode        viewMode = "confirm"
)

type refreshFinishedMsg struct {
	inventory app.Inventory
	err       error
}

type reviewFinishedMsg struct {
	review app.TerminationReview
	err    error
}

type executionFinishedMsg struct {
	execution app.TerminationExecution
	err       error
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Back    key.Binding
	Toggle  key.Binding
	Review  key.Binding
	Force   key.Binding
	Refresh key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Quit    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "move down"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "left", "h"),
			key.WithHelp("esc", "back"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "x"),
			key.WithHelp("space", "toggle row"),
		),
		Review: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "review graceful"),
		),
		Force: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "review force"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

type styles struct {
	title     lipgloss.Style
	muted     lipgloss.Style
	status    lipgloss.Style
	selected  lipgloss.Style
	cursor    lipgloss.Style
	warning   lipgloss.Style
	errorText lipgloss.Style
	box       lipgloss.Style
}

func newStyles() styles {
	return styles{
		title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		muted: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1),
		selected:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		cursor:    lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true),
		warning:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		errorText: lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
	}
}

type Model struct {
	workflow          app.Workflow
	capabilities      platform.Capabilities
	keys              keyMap
	styles            styles
	mode              viewMode
	browseMode        viewMode
	loading           bool
	terminalWidth     int
	terminalHeight    int
	categoryCursor    int
	detailCursor      int
	detailWindowStart int
	selectedCategory  domain.Category
	inventory         app.Inventory
	selection         domain.Selection
	review            app.TerminationReview
	status            string
}

const (
	defaultDetailVisibleCapacity = 8
	detailViewNonRowHeight       = 9
)

func NewModel(workflow app.Workflow, capabilities platform.Capabilities) Model {
	return Model{
		workflow:         workflow,
		capabilities:     capabilities,
		keys:             newKeyMap(),
		styles:           newStyles(),
		mode:             categoryListMode,
		browseMode:       categoryListMode,
		selectedCategory: domain.CategoryAll,
		selection:        domain.NewSelection(),
		status:           "Refreshing listening TCP inventory...",
	}
}

func (m Model) Init() tea.Cmd {
	m.loading = true
	return refreshCmd(m.workflow)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshFinishedMsg:
		return m.handleRefresh(msg), nil
	case reviewFinishedMsg:
		return m.handleReview(msg), nil
	case executionFinishedMsg:
		return m.handleExecution(msg), nil
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		return m.syncDetailWindow(), nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) View() string {
	lines := []string{
		m.styles.title.Render("portcut"),
		m.styles.muted.Render(fmt.Sprintf("platform: %s (%s)", m.capabilities.Platform, m.capabilities.Shell)),
		m.styles.muted.Render(capabilitySummary(m.capabilities)),
		m.styles.status.Render(m.status),
	}

	if m.loading {
		lines = append(lines, m.styles.muted.Render("Working..."))
	}

	if m.mode == confirmMode {
		lines = append(lines, m.confirmationView())
		lines = append(lines, m.styles.muted.Render(confirmHelp()))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	if m.mode == categoryDetailMode {
		lines = append(lines, m.categoryDetailView())
		lines = append(lines, m.styles.muted.Render(detailHelp(m.capabilities)))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	lines = append(lines, m.categoryListView())
	lines = append(lines, m.styles.muted.Render(categoryListHelp()))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	if m.loading {
		return m, nil
	}

	if m.mode == confirmMode {
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Back) {
			m.mode = m.browseMode
			m.review = app.TerminationReview{}
			m.status = "Termination cancelled."
			return m, nil
		}
		if key.Matches(msg, m.keys.Confirm) {
			m.loading = true
			m.status = "Executing termination workflow..."
			return m, executeCmd(m.workflow, m.review)
		}

		return m, nil
	}

	if key.Matches(msg, m.keys.Refresh) {
		m.loading = true
		m.mode = categoryListMode
		m.browseMode = categoryListMode
		m.selectedCategory = domain.CategoryAll
		m.review = app.TerminationReview{}
		m.status = "Refreshing listening TCP inventory..."
		return m, refreshCmd(m.workflow)
	}

	if m.mode == categoryListMode {
		return m.handleCategoryListKey(msg)
	}

	return m.handleCategoryDetailKey(msg)
}

func (m Model) handleCategoryListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.categoryCursor > 0 {
			m.categoryCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.categoryCursor < len(m.categorySummaries())-1 {
			m.categoryCursor++
		}
	case key.Matches(msg, m.keys.Review):
		summary, ok := m.currentCategorySummary()
		if !ok {
			m.status = "No categories are available to browse."
			return m, nil
		}
		m.selectedCategory = summary.Category
		m.detailCursor = 0
		m.detailWindowStart = 0
		m.mode = categoryDetailMode
		m.browseMode = categoryDetailMode
		m = m.syncDetailWindow()
		if summary.Count == 0 {
			m.status = fmt.Sprintf("Browsing %s. No listening TCP rows are currently available in this category.", summary.Label)
			return m, nil
		}
		m.status = fmt.Sprintf("Browsing %s.", summary.Label)
	}

	return m, nil
}

func (m Model) handleCategoryDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = categoryListMode
		m.browseMode = categoryListMode
		m.status = "Returned to category list."
	case key.Matches(msg, m.keys.Up):
		if m.detailCursor > 0 {
			m.detailCursor--
			m.detailWindowStart = m.keepDetailCursorVisible()
		}
	case key.Matches(msg, m.keys.Down):
		if m.detailCursor < len(m.detailEntries())-1 {
			m.detailCursor++
			m.detailWindowStart = m.keepDetailCursorVisible()
		}
	case key.Matches(msg, m.keys.Toggle):
		entry, ok := m.currentEntry()
		if !ok {
			m.status = fmt.Sprintf("No listening TCP rows are available in %s.", m.selectedCategory.Label())
			return m, nil
		}
		if !entry.CanTerminate() {
			m.status = fmt.Sprintf("Port %d cannot be terminated because the PID is unavailable.", entry.Port)
			return m, nil
		}
		m.selection.Toggle(entry.ID)
		m.status = fmt.Sprintf("Selected %d row%s.", m.selection.Count(), plural(m.selection.Count()))
	case key.Matches(msg, m.keys.Review):
		if !m.capabilities.GracefulTermination {
			m.status = fmt.Sprintf("Graceful termination is unsupported on %s. Press f to review force termination.", m.capabilities.Platform)
			return m, nil
		}
		m.loading = true
		m.browseMode = categoryDetailMode
		m.status = "Refreshing before graceful termination review..."
		return m, reviewCmd(m.workflow, m.selection, false)
	case key.Matches(msg, m.keys.Force):
		if !m.capabilities.ForceTermination {
			m.status = fmt.Sprintf("Force termination is unsupported on %s.", m.capabilities.Platform)
			return m, nil
		}
		m.loading = true
		m.browseMode = categoryDetailMode
		m.status = "Refreshing before force termination review..."
		return m, reviewCmd(m.workflow, m.selection, true)
	}

	return m, nil
}

func (m Model) handleRefresh(msg refreshFinishedMsg) Model {
	m.loading = false
	m.mode = categoryListMode
	m.browseMode = categoryListMode
	m.selectedCategory = domain.CategoryAll
	m.review = app.TerminationReview{}

	if msg.err != nil {
		m.status = fmt.Sprintf("Refresh failed: %v", msg.err)
		return m
	}

	m.inventory = msg.inventory
	m.selection = retainLiveSelection(m.selection, msg.inventory.Entries)
	m.categoryCursor = clampCursor(m.categoryCursor, len(m.categorySummaries()))
	m.detailCursor = 0
	m.detailWindowStart = 0
	m = m.syncDetailWindow()

	if len(msg.inventory.Entries) == 0 {
		m.status = "No listening TCP ports found. Categories remain available; press r to refresh or q to quit."
		return m
	}

	m.status = fmt.Sprintf("Loaded %d listening TCP port row%s. Choose a category to browse.", len(msg.inventory.Entries), plural(len(msg.inventory.Entries)))
	return m
}

func (m Model) handleReview(msg reviewFinishedMsg) Model {
	m.loading = false

	if msg.err != nil {
		m.status = fmt.Sprintf("Review failed: %v", msg.err)
		return m
	}

	m.inventory = msg.review.Inventory
	m.selection = selectionFromEntries(msg.review.SelectedEntries)
	m.categoryCursor = clampCursor(m.categoryCursor, len(m.categorySummaries()))
	m.detailCursor = clampCursor(m.detailCursor, len(m.detailEntries()))
	m = m.syncDetailWindow()

	switch msg.review.Status {
	case app.ReviewStatusReady:
		m.browseMode = m.mode
		m.mode = confirmMode
		m.review = msg.review
		m.status = fmt.Sprintf("Review ready for %d process target%s.", len(msg.review.Targets), plural(len(msg.review.Targets)))
	case app.ReviewStatusEmpty:
		m.review = app.TerminationReview{}
		m.status = "Select at least one terminable row before reviewing termination."
	case app.ReviewStatusStale:
		m.review = app.TerminationReview{}
		m.status = "Selection changed after refresh. Review the updated rows before terminating."
	case app.ReviewStatusUnsupported:
		m.review = app.TerminationReview{}
		m.status = fmt.Sprintf("%s termination is unsupported on %s.", terminationMode(msg.review.Force), msg.review.Capabilities.Platform)
	default:
		m.review = app.TerminationReview{}
		m.status = "Termination review returned an unknown state."
	}

	return m
}

func (m Model) handleExecution(msg executionFinishedMsg) Model {
	m.loading = false
	m.mode = categoryListMode
	m.browseMode = categoryListMode
	m.selectedCategory = domain.CategoryAll
	m.review = app.TerminationReview{}
	m.inventory = msg.execution.Refreshed
	m.selection = domain.NewSelection()
	m.categoryCursor = clampCursor(m.categoryCursor, len(m.categorySummaries()))
	m.detailCursor = 0
	m.detailWindowStart = 0
	m = m.syncDetailWindow()

	summary := summarizeExecution(msg.execution.Termination)
	if msg.err != nil {
		if summary == "" {
			m.status = fmt.Sprintf("Termination failed: %v", msg.err)
			return m
		}
		m.status = fmt.Sprintf("%s Error: %v", summary, msg.err)
		return m
	}

	if summary == "" {
		m.status = "Termination completed."
		return m
	}

	m.status = summary
	return m
}

func (m Model) categoryListView() string {
	summaries := m.categorySummaries()
	rows := make([]string, 0, len(summaries)+2)
	rows = append(rows, "Browse categories")
	rows = append(rows, m.styles.muted.Render(fmt.Sprintf("Selections stay global across categories. %d row%s selected.", m.selection.Count(), plural(m.selection.Count()))))
	for index, summary := range summaries {
		cursor := " "
		if index == m.categoryCursor {
			cursor = m.styles.cursor.Render(">")
		}

		selectedCount := m.selectedCount(m.entriesForCategory(summary.Category))
		row := fmt.Sprintf("%s %-18s %2d row%s", cursor, summary.Label, summary.Count, plural(summary.Count))
		if selectedCount > 0 {
			row = fmt.Sprintf("%s  %s", row, m.styles.selected.Render(fmt.Sprintf("%d selected", selectedCount)))
		}
		if summary.Count == 0 {
			row = fmt.Sprintf("%s %s", row, m.styles.muted.Render("(empty)"))
		}
		rows = append(rows, row)
	}

	return m.styles.box.Render(strings.Join(rows, "\n"))
}

func (m Model) categoryDetailView() string {
	entries := m.detailEntries()
	visibleSelected := m.selectedCount(entries)
	windowStart, windowEnd := m.detailWindowBounds(len(entries))
	rows := []string{
		m.detailHeader(len(entries), windowStart, windowEnd),
		m.styles.muted.Render(fmt.Sprintf("Selections are global: %d total, %d shown here.", m.selection.Count(), visibleSelected)),
	}

	if len(entries) == 0 {
		rows = append(rows, m.styles.muted.Render("No listening TCP rows are currently available in this category."))
		return m.styles.box.Render(strings.Join(rows, "\n"))
	}

	for index, entry := range entries[windowStart:windowEnd] {
		absoluteIndex := windowStart + index
		cursor := " "
		if absoluteIndex == m.detailCursor {
			cursor = m.styles.cursor.Render(">")
		}

		selected := "[ ]"
		if m.selection.Has(entry.ID) {
			selected = m.styles.selected.Render("[x]")
		}

		row := fmt.Sprintf("%s %s port %-5d pid %-6s %s", cursor, selected, entry.Port, entry.DisplayPID(), entry.DisplayProcessName())
		if !entry.CanTerminate() {
			row = fmt.Sprintf("%s %s", row, m.styles.warning.Render("(read-only)"))
		}
		rows = append(rows, row)
	}

	return m.styles.box.Render(strings.Join(rows, "\n"))
}

func (m Model) confirmationView() string {
	modeLabel := terminationMode(m.review.Force)
	rows := []string{
		fmt.Sprintf("Confirm %s termination", modeLabel),
		m.styles.muted.Render(fmt.Sprintf("%d row%s -> %d process target%s", len(m.review.SelectedEntries), plural(len(m.review.SelectedEntries)), len(m.review.Targets), plural(len(m.review.Targets)))),
	}

	for _, target := range m.review.Targets {
		rows = append(rows, fmt.Sprintf("- pid %d  %s  ports %s", target.PID, target.ProcessName, joinPorts(target.Ports)))
	}

	rows = append(rows, m.styles.warning.Render("Press y to terminate or n to cancel."))
	return m.styles.box.Render(strings.Join(rows, "\n"))
}

func (m Model) currentEntry() (domain.PortProcessEntry, bool) {
	entries := m.detailEntries()
	if len(entries) == 0 || m.detailCursor < 0 || m.detailCursor >= len(entries) {
		return domain.PortProcessEntry{}, false
	}

	return entries[m.detailCursor], true
}

func (m Model) categorySummaries() []domain.CategorySummary {
	if len(m.inventory.CategorySummaries) > 0 {
		return m.inventory.CategorySummaries
	}

	return domain.BuildCategorySummaries(m.inventory.Entries)
}

func (m Model) currentCategorySummary() (domain.CategorySummary, bool) {
	summaries := m.categorySummaries()
	if len(summaries) == 0 || m.categoryCursor < 0 || m.categoryCursor >= len(summaries) {
		return domain.CategorySummary{}, false
	}

	return summaries[m.categoryCursor], true
}

func (m Model) entriesForCategory(category domain.Category) []domain.PortProcessEntry {
	if entries, ok := m.inventory.EntriesByCategory[category]; ok {
		return entries
	}

	return domain.EntriesForCategory(m.inventory.Entries, category)
}

func (m Model) detailEntries() []domain.PortProcessEntry {
	return m.entriesForCategory(m.selectedCategory)
}

func (m Model) detailVisibleCapacity() int {
	if m.terminalHeight <= 0 {
		return defaultDetailVisibleCapacity
	}

	return maxInt(1, m.terminalHeight-detailViewNonRowHeight)
}

func (m Model) detailWindowBounds(total int) (int, int) {
	if total == 0 {
		return 0, 0
	}

	capacity := m.detailVisibleCapacity()
	start := clampDetailWindowStart(m.detailWindowStart, total, capacity)
	start = keepCursorVisible(start, m.detailCursor, total, capacity)
	end := start + capacity
	if end > total {
		end = total
	}

	return start, end
}

func (m Model) keepDetailCursorVisible() int {
	return keepCursorVisible(m.detailWindowStart, m.detailCursor, len(m.detailEntries()), m.detailVisibleCapacity())
}

func (m Model) syncDetailWindow() Model {
	m.detailCursor = clampCursor(m.detailCursor, len(m.detailEntries()))
	m.detailWindowStart = m.keepDetailCursorVisible()
	return m
}

func (m Model) detailHeader(total int, windowStart int, windowEnd int) string {
	header := fmt.Sprintf("Category: %s", m.selectedCategory.Label())
	if total == 0 {
		return header
	}

	cues := make([]string, 0, 2)
	if windowStart > 0 {
		cues = append(cues, "^ more")
	}
	if windowEnd < total {
		cues = append(cues, "v more")
	}

	rangeLabel := fmt.Sprintf("rows %d-%d of %d", windowStart+1, windowEnd, total)
	if len(cues) > 0 {
		rangeLabel = fmt.Sprintf("%s (%s)", rangeLabel, strings.Join(cues, ", "))
	}

	return fmt.Sprintf("%s  %s", header, m.styles.muted.Render(rangeLabel))
}

func (m Model) selectedCount(entries []domain.PortProcessEntry) int {
	count := 0
	for _, entry := range entries {
		if m.selection.Has(entry.ID) {
			count++
		}
	}

	return count
}

func refreshCmd(workflow app.Workflow) tea.Cmd {
	return func() tea.Msg {
		inventory, err := workflow.Refresh(context.Background())
		return refreshFinishedMsg{inventory: inventory, err: err}
	}
}

func reviewCmd(workflow app.Workflow, selection domain.Selection, force bool) tea.Cmd {
	return func() tea.Msg {
		review, err := workflow.ReviewTermination(context.Background(), selection, force)
		return reviewFinishedMsg{review: review, err: err}
	}
}

func executeCmd(workflow app.Workflow, review app.TerminationReview) tea.Cmd {
	return func() tea.Msg {
		execution, err := workflow.ExecuteTermination(context.Background(), review)
		return executionFinishedMsg{execution: execution, err: err}
	}
}

func retainLiveSelection(selection domain.Selection, entries []domain.PortProcessEntry) domain.Selection {
	live := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		live[entry.ID] = struct{}{}
	}

	ids := make([]string, 0, selection.Count())
	for _, id := range selection.IDs() {
		if _, ok := live[id]; ok {
			ids = append(ids, id)
		}
	}

	return domain.NewSelection(ids...)
}

func selectionFromEntries(entries []domain.PortProcessEntry) domain.Selection {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	return domain.NewSelection(ids...)
}

func clampCursor(cursor int, length int) int {
	if length == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func clampDetailWindowStart(start int, total int, capacity int) int {
	if total == 0 || capacity <= 0 || total <= capacity {
		return 0
	}
	if start < 0 {
		return 0
	}

	maxStart := total - capacity
	if start > maxStart {
		return maxStart
	}

	return start
}

func keepCursorVisible(start int, cursor int, total int, capacity int) int {
	if total == 0 || capacity <= 0 {
		return 0
	}

	start = clampDetailWindowStart(start, total, capacity)
	cursor = clampCursor(cursor, total)
	if cursor < start {
		start = cursor
	}
	if cursor >= start+capacity {
		start = cursor - capacity + 1
	}

	return clampDetailWindowStart(start, total, capacity)
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}

	return right
}

func summarizeExecution(result platform.TerminateResult) string {
	if len(result.Outcomes) == 0 {
		return ""
	}

	completed := 0
	skipped := 0
	failed := 0
	for _, outcome := range result.Outcomes {
		switch outcome.Status {
		case platform.TerminationStatusCompleted:
			completed++
		case platform.TerminationStatusSkipped:
			skipped++
		case platform.TerminationStatusFailed:
			failed++
		}
	}

	parts := make([]string, 0, 3)
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("terminated %d process target%s", completed, plural(completed)))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("skipped %d", skipped))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("failed %d", failed))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("Termination finished: %s.", strings.Join(parts, ", "))
}

func joinPorts(ports []uint16) string {
	labels := make([]string, 0, len(ports))
	for _, port := range ports {
		labels = append(labels, fmt.Sprintf("%d", port))
	}

	return strings.Join(labels, ",")
}

func terminationMode(force bool) string {
	if force {
		return "force"
	}
	return "graceful"
}

func capabilitySummary(capabilities platform.Capabilities) string {
	modes := make([]string, 0, 2)
	if capabilities.GracefulTermination {
		modes = append(modes, "graceful")
	}
	if capabilities.ForceTermination {
		modes = append(modes, "force")
	}
	if len(modes) == 0 {
		return "termination: unavailable"
	}

	return fmt.Sprintf("termination: %s", strings.Join(modes, "+"))
}

func categoryListHelp() string {
	return "up/down move  enter browse  r refresh  q quit"
}

func detailHelp(capabilities platform.Capabilities) string {
	parts := []string{"up/down move", "space select"}
	if capabilities.GracefulTermination {
		parts = append(parts, "enter review")
	}
	if capabilities.ForceTermination {
		parts = append(parts, "f review force")
	}
	parts = append(parts, "esc back", "r refresh", "q quit")
	return strings.Join(parts, "  ")
}

func confirmHelp() string {
	return "y confirm  n cancel  q quit"
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
