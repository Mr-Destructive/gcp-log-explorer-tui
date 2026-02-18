package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/log-explorer-tui/pkg/config"
	"github.com/user/log-explorer-tui/pkg/models"
	"github.com/user/log-explorer-tui/pkg/query"
)

var (
	pythonBoolRegex = regexp.MustCompile(`\b(True|False|None)\b`)
	ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

const (
	colorGCPBlue       = "33"
	colorGCPBlueDark   = "24"
	colorGCPBlueLight  = "117"
	colorGCPGreen      = "42"
	colorGCPWarn       = "220"
	colorGCPError      = "203"
	colorNeutralText   = "252"
	colorNeutralSubtle = "244"
	colorSelectionBG   = "25"
	colorSelectionFG   = "230"
	colorBadgeTextDark = "16"
	colorBadgeTextLite = "230"
)

// App represents the main TUI application
type App struct {
	state                   *models.AppState
	width                   int
	height                  int
	panes                   *Panes
	lastErr                 string
	helpModal               *HelpModal
	queryModal              *QueryModal
	timePicker              *TimePicker
	severityFilter          *SeverityFilterPanel
	exporter                *Exporter
	formatter               *LogFormatter
	timelineBuilder         *TimelineBuilder
	statusBar               string
	queryExec               func(string) ([]models.LogEntry, error)
	activeModalName         string // Track which modal is open
	previousModalName       string
	vimMode                 bool
	timezoneMode            string // "utc" or "local"
	logOrder                string // "latest_bottom" or "latest_top"
	keyModeCursor           int
	timezoneCursor          int
	loadingOlder            bool
	loadingNewer            bool
	detailScroll            int
	detailCursor            int
	detailViewMode          string
	detailTreeExpanded      map[string]bool
	severityCursor          int
	autoLoadAll             bool
	projectPopup            bool
	projectCursor           int
	availableProjects       []string
	queryHistory            []string
	queryHistoryCursor      int
	queryHistoryPopupCursor int
	startupFilter           string
	projectListFn           func() ([]string, error)
	loadingProjects         bool
	queryLibrary            []config.SavedQueryRecord
	queryLibraryCursor      int
	queryCache              map[string]config.CachedQueryRecord
	queryCacheTTL           time.Duration
	queryCacheMax           int
	bypassNextCache         bool
	persistHistoryFn        func(filter, project string) error
	persistLibraryFn        func([]config.SavedQueryRecord) error
	persistCacheFn          func([]config.CachedQueryRecord) error
}

type queryResultMsg struct {
	filter         string
	logs           []models.LogEntry
	err            error
	mode           string // replace, append, prepend
	anchorOffset   int
	preserveAnchor bool
	fromCache      bool
}

type editorResultMsg struct {
	err    error
	target string
}

type projectListMsg struct {
	projects []string
	err      error
}

// NewApp creates a new TUI application
func NewApp(appState *models.AppState) *App {
	panes := NewPanes()
	helpModal := NewHelpModal()
	queryModal := NewQueryModal()
	timePicker := NewTimePicker()
	severityFilter := NewSeverityFilterPanel()
	exporter := NewExporter()
	formatter := NewLogFormatter(120, false)
	timelineBuilder := NewTimelineBuilder(5 * time.Minute)
	availableProjects := collectProjects(appState.CurrentProject)
	return &App{
		state:                   appState,
		width:                   120,
		height:                  40,
		panes:                   panes,
		lastErr:                 "",
		helpModal:               helpModal,
		queryModal:              queryModal,
		timePicker:              timePicker,
		severityFilter:          severityFilter,
		exporter:                exporter,
		formatter:               formatter,
		timelineBuilder:         timelineBuilder,
		statusBar:               helpModal.GetShortHelp(),
		queryExec:               nil,
		activeModalName:         "none",
		previousModalName:       "none",
		vimMode:                 true,
		timezoneMode:            "utc",
		logOrder:                "latest_bottom",
		keyModeCursor:           0,
		timezoneCursor:          0,
		loadingOlder:            false,
		loadingNewer:            false,
		detailScroll:            0,
		detailCursor:            0,
		detailViewMode:          "full",
		detailTreeExpanded:      map[string]bool{"$": true},
		severityCursor:          0,
		autoLoadAll:             false,
		projectPopup:            false,
		projectCursor:           0,
		availableProjects:       availableProjects,
		queryHistory:            []string{},
		queryHistoryCursor:      -1,
		queryHistoryPopupCursor: 0,
		startupFilter:           "",
		projectListFn:           nil,
		loadingProjects:         false,
		queryLibrary:            []config.SavedQueryRecord{},
		queryLibraryCursor:      0,
		queryCache:              map[string]config.CachedQueryRecord{},
		queryCacheTTL:           15 * time.Minute,
		queryCacheMax:           40,
		bypassNextCache:         false,
		persistHistoryFn:        nil,
		persistLibraryFn:        nil,
		persistCacheFn:          nil,
	}
}

func collectProjects(current string) []string {
	set := map[string]bool{}
	list := []string{}
	add := func(project string) {
		project = strings.TrimSpace(project)
		if project == "" || set[project] {
			return
		}
		set[project] = true
		list = append(list, project)
	}

	add(current)
	add(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	add(os.Getenv("CLOUDSDK_CORE_PROJECT"))
	for _, project := range discoverGcloudProjects() {
		add(project)
	}

	return list
}

func discoverGcloudProjects() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	root := filepath.Join(home, ".config", "gcloud")
	configDir := filepath.Join(root, "configurations")
	projects := []string{}
	seen := map[string]bool{}
	add := func(project string) {
		project = strings.TrimSpace(project)
		if project == "" || seen[project] {
			return
		}
		seen[project] = true
		projects = append(projects, project)
	}
	addFromFile := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		if project := parseProjectFromGcloudConfig(string(data)); project != "" {
			add(project)
		}
	}

	addFromFile(filepath.Join(root, "properties"))
	if entries, err := os.ReadDir(configDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), "config_") {
				continue
			}
			addFromFile(filepath.Join(configDir, entry.Name()))
		}
	}

	return projects
}

func parseProjectFromGcloudConfig(contents string) string {
	lines := strings.Split(contents, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key != "project" {
			continue
		}
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// SetQueryExecutor sets the query execution function
func (a *App) SetQueryExecutor(fn func(string) ([]models.LogEntry, error)) {
	a.queryExec = fn
}

// SetStartupFilter configures a filter to execute automatically during Init.
func (a *App) SetStartupFilter(filter string) {
	a.startupFilter = filter
}

// SetVimMode enables or disables vim-style navigation keys.
func (a *App) SetVimMode(enabled bool) {
	a.vimMode = enabled
}

func (a *App) toggleTimezoneMode() {
	if a.timezoneMode == "local" {
		a.timezoneMode = "utc"
		a.formatter.SetUseLocalTime(false)
		a.lastErr = "Timezone: UTC"
		return
	}
	a.timezoneMode = "local"
	a.formatter.SetUseLocalTime(true)
	a.lastErr = "Timezone: local"
}

func (a *App) toggleLogOrder() {
	selectedKey := ""
	if len(a.state.LogListState.Logs) > 0 {
		selectedKey = logEntryKey(a.state.LogListState.Logs[a.currentSelectedIndex()])
	}
	if a.logOrder == "latest_bottom" {
		a.logOrder = "latest_top"
	} else {
		a.logOrder = "latest_bottom"
	}
	a.state.LogListState.Logs = a.sortLogsForDisplay(a.state.LogListState.Logs)
	if selectedKey != "" {
		if idx, ok := a.findLogIndexByKey(selectedKey); ok {
			a.panes.LogList.scrollOffset = idx
		}
	}
	a.lastErr = "Log order: " + strings.ReplaceAll(a.logOrder, "_", " ")
}

func (a *App) displayTime(t time.Time) time.Time {
	if a.timezoneMode == "local" {
		return t.Local()
	}
	return t.UTC()
}

func (a *App) sortLogsForDisplay(logs []models.LogEntry) []models.LogEntry {
	out := append([]models.LogEntry{}, logs...)
	if len(out) < 2 {
		return out
	}
	if a.logOrder == "latest_bottom" {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].Timestamp.Before(out[j].Timestamp)
		})
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out
}

func (a *App) findLogIndexByKey(key string) (int, bool) {
	for i, entry := range a.state.LogListState.Logs {
		if logEntryKey(entry) == key {
			return i, true
		}
	}
	return 0, false
}

// SetQueryHistory sets initial query history (most recent first).
func (a *App) SetQueryHistory(history []string) {
	a.queryHistory = append([]string{}, history...)
	if len(a.queryHistory) > 25 {
		a.queryHistory = a.queryHistory[:25]
	}
}

// SetQueryHistoryPersistFn sets persistence callback for query history appends.
func (a *App) SetQueryHistoryPersistFn(fn func(filter, project string) error) {
	a.persistHistoryFn = fn
}

// SetQueryLibrary sets initial query library records.
func (a *App) SetQueryLibrary(queries []config.SavedQueryRecord) {
	a.queryLibrary = append([]config.SavedQueryRecord{}, queries...)
}

// SetQueryLibraryPersistFn sets persistence callback for query library.
func (a *App) SetQueryLibraryPersistFn(fn func([]config.SavedQueryRecord) error) {
	a.persistLibraryFn = fn
}

// SetQueryCacheEntries loads persisted cache entries.
func (a *App) SetQueryCacheEntries(entries []config.CachedQueryRecord) {
	a.queryCache = map[string]config.CachedQueryRecord{}
	now := time.Now()
	for _, entry := range entries {
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		if a.queryCacheTTL > 0 && now.Sub(entry.StoredAt) > a.queryCacheTTL {
			continue
		}
		a.queryCache[entry.Key] = entry
	}
}

// SetQueryCachePersistFn sets persistence callback for query cache.
func (a *App) SetQueryCachePersistFn(fn func([]config.CachedQueryRecord) error) {
	a.persistCacheFn = fn
}

// SetProjectLister sets a callback used to discover all projects for the selector.
func (a *App) SetProjectLister(fn func() ([]string, error)) {
	a.projectListFn = fn
}

// Init initializes the app (required by Bubble Tea)
func (a *App) Init() tea.Cmd {
	if a.queryExec == nil {
		return nil
	}
	if len(a.state.LogListState.Logs) > 0 {
		return nil
	}
	if strings.TrimSpace(a.startupFilter) == "" {
		return nil
	}
	return a.executePrimaryQueryCmd(a.buildEffectiveFilter(a.startupFilter))
}

// Update handles events and state mutations
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Handle window resize
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case queryResultMsg:
		if msg.err != nil {
			a.lastErr = fmt.Sprintf("Query error: %v", msg.err)
		} else {
			orderedLogs := a.sortLogsForDisplay(msg.logs)
			switch msg.mode {
			case "append":
				before := len(a.state.LogListState.Logs)
				a.state.LogListState.Logs = mergeUniqueLogs(a.state.LogListState.Logs, orderedLogs, false)
				if msg.preserveAnchor {
					a.panes.LogList.scrollOffset = minInt(maxInt(0, msg.anchorOffset), maxInt(0, len(a.state.LogListState.Logs)-1))
				}
				a.lastErr = fmt.Sprintf("Loaded logs: +%d", len(a.state.LogListState.Logs)-before)
			case "prepend":
				before := len(a.state.LogListState.Logs)
				a.state.LogListState.Logs = mergeUniqueLogs(a.state.LogListState.Logs, orderedLogs, true)
				// Keep viewport near current context when prepending.
				a.panes.LogList.scrollOffset += len(a.state.LogListState.Logs) - before
				a.lastErr = fmt.Sprintf("Loaded logs: +%d", len(a.state.LogListState.Logs)-before)
			default:
				a.state.LogListState.Logs = orderedLogs
				if a.logOrder == "latest_bottom" {
					a.panes.LogList.scrollOffset = maxInt(0, len(a.state.LogListState.Logs)-1)
				} else {
					a.panes.LogList.scrollOffset = 0
				}
				if msg.fromCache {
					a.lastErr = fmt.Sprintf("Query cache hit: %d logs", len(orderedLogs))
				} else {
					a.lastErr = fmt.Sprintf("Query complete: %d logs", len(orderedLogs))
					a.storeQueryResultCache(msg.filter, orderedLogs)
				}
			}
		}
		if msg.mode == "append" {
			a.loadingOlder = false
		}
		if msg.mode == "prepend" {
			a.loadingNewer = false
		}
		if msg.mode == "replace" {
			a.state.LogListState.IsLoading = false
		} else {
			a.state.LogListState.IsLoading = a.loadingOlder || a.loadingNewer
		}
		return a, nil

	case editorResultMsg:
		if msg.err != nil {
			a.lastErr = fmt.Sprintf("Open editor failed: %v", msg.err)
		} else {
			a.lastErr = fmt.Sprintf("Closed external editor (%s)", msg.target)
		}
		return a, nil

	case projectListMsg:
		a.loadingProjects = false
		if msg.err != nil {
			a.lastErr = fmt.Sprintf("Project discovery failed: %v", msg.err)
			return a, nil
		}
		a.availableProjects = mergeUniqueStrings(a.availableProjects, msg.projects)
		if len(a.availableProjects) == 0 {
			a.projectCursor = 0
		} else if a.projectCursor >= len(a.availableProjects) {
			a.projectCursor = len(a.availableProjects) - 1
		}
		a.lastErr = fmt.Sprintf("Loaded %d projects", len(a.availableProjects))
		return a, nil

	// Handle keyboard input
	case tea.KeyMsg:
		// Debug: log the key press
		keyStr := msg.String()
		if keyStr != "" {
			// Uncomment to debug: fmt.Fprintf(os.Stderr, "Key pressed: %s\n", keyStr)
		}
		return a.handleKeyPress(msg)
	}

	return a, nil
}

// View renders the UI
func (a *App) View() string {
	if !a.state.IsReady {
		return "Loading...\n"
	}

	queryText := a.state.CurrentQuery.Filter
	if a.activeModalName == "query" {
		queryText = a.queryModal.GetInputWithCursor()
	}

	topBar := a.renderTopBar()
	header := a.renderQueryPanel(queryText, a.activeModalName == "query")
	timeline := a.renderGraphPanel()

	details := ""
	detailsLines := 0
	if a.activeModalName == "details" {
		details = a.renderDetailsPanel()
		detailsLines = strings.Count(details, "\n")
	}

	headerLines := strings.Count(header, "\n")
	footerLines := 3
	if a.lastErr != "" {
		footerLines = 4
	}
	topBarLines := strings.Count(topBar, "\n")
	timelineLines := strings.Count(timeline, "\n")
	overhead := topBarLines + headerLines + footerLines + timelineLines + detailsLines

	screenHeight := maxInt(1, a.height-1)
	logsHeight := screenHeight - overhead - 2
	if logsHeight < 6 {
		logsHeight = 6
	}
	logsPanel, windowStart, windowEnd := a.renderLogsPanel(logsHeight)
	footer := a.renderStatusPanel(windowStart, windowEnd)

	var result strings.Builder
	result.WriteString(topBar)
	result.WriteString(header)
	result.WriteString(logsPanel)
	result.WriteString(timeline)
	if details != "" {
		result.WriteString(details)
	}
	result.WriteString(footer)
	output := result.String()

	// Show active modal if any
	switch a.activeModalName {
	case "query":
		// Query editor is rendered as a persistent top panel.
	case "timeRange":
		output = a.renderCenteredPopup(output, a.renderTimePickerModal())
	case "severity":
		output = a.renderCenteredPopup(output, a.renderSeverityFilterModal())
	case "export":
		output = a.renderCenteredPopup(output, a.renderExportModal())
	case "detailPopup":
		output = a.renderCenteredPopup(output, a.renderDetailPopup())
	case "help":
		output = a.renderCenteredPopup(output, a.helpModal.Render(a.width, a.height))
	case "projectPopup":
		output = a.renderCenteredPopup(output, a.renderProjectDropdown())
	case "queryLibrary":
		output = a.renderCenteredPopup(output, a.renderQueryLibraryPopup())
	case "queryHistory":
		output = a.renderCenteredPopup(output, a.renderQueryHistoryPopup())
	case "keyModePopup":
		output = a.renderCenteredPopup(output, a.renderKeyModePopup())
	case "timezonePopup":
		output = a.renderCenteredPopup(output, a.renderTimezonePopup())
	}

	return a.fitToViewport(output)
}

// handleKeyPress processes keyboard input
func (a *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input based on active modal
	switch a.activeModalName {
	case "query":
		return a.handleQueryModalInput(msg)
	case "timeRange":
		return a.handleTimePickerInput(msg)
	case "severity":
		return a.handleSeverityFilterInput(msg)
	case "export":
		return a.handleExportInput(msg)
	case "details":
		switch msg.String() {
		case "esc", "enter", "ctrl+d":
			a.activeModalName = "none"
		case "ctrl+p":
			a.activeModalName = "detailPopup"
			a.resetDetailPopupState()
		}
		return a, nil
	case "detailPopup":
		switch msg.String() {
		case "esc", "ctrl+p":
			a.activeModalName = "none"
			a.detailScroll = 0
			a.detailCursor = 0
		case "j", "down":
			a.moveDetailPopupDown()
		case "k", "up":
			a.moveDetailPopupUp()
		case "h", "left":
			a.collapseDetailTreeNode()
		case "l", "right", "enter":
			a.expandDetailTreeNode()
		case "z":
			a.collapseAllDetailTreeNodes()
		case "Z":
			a.expandAllDetailTreeNodes()
		case "tab", "v":
			a.cycleDetailViewMode()
		case "y":
			a.copySelectedDetailNode()
		case "Y":
			a.copyDetailPayload()
		case "ctrl+o":
			return a, a.openSelectedLogInEditorCmd()
		case "ctrl+e":
			return a, a.openSelectedPayloadInEditorCmd()
		case "ctrl+l":
			return a, a.openLogListInEditorCmd()
		case "ctrl+L", "alt+l":
			return a, a.openLogListCSVInEditorCmd()
		}
		return a, nil
	case "projectPopup":
		switch msg.String() {
		case "esc":
			a.activeModalName = "none"
		case "j", "down":
			a.projectCursor++
			if a.projectCursor >= len(a.availableProjects) {
				a.projectCursor = len(a.availableProjects) - 1
			}
		case "k", "up":
			a.projectCursor--
			if a.projectCursor < 0 {
				a.projectCursor = 0
			}
		case "enter":
			a.selectProject()
		}
		return a, nil
	case "queryLibrary":
		switch msg.String() {
		case "esc":
			a.activeModalName = a.previousModalName
		case "j", "down":
			if len(a.queryLibrary) > 0 {
				a.queryLibraryCursor++
				if a.queryLibraryCursor >= len(a.queryLibrary) {
					a.queryLibraryCursor = len(a.queryLibrary) - 1
				}
			}
		case "k", "up":
			if len(a.queryLibrary) > 0 {
				a.queryLibraryCursor--
				if a.queryLibraryCursor < 0 {
					a.queryLibraryCursor = 0
				}
			}
		case "enter":
			return a.applySelectedQueryLibraryEntry()
		}
		return a, nil
	case "queryHistory":
		switch msg.String() {
		case "esc":
			a.activeModalName = a.previousModalName
		case "j", "down", "ctrl+g":
			if len(a.queryHistory) > 0 {
				a.queryHistoryPopupCursor++
				if a.queryHistoryPopupCursor >= len(a.queryHistory) {
					a.queryHistoryPopupCursor = len(a.queryHistory) - 1
				}
			}
		case "k", "up", "ctrl+r":
			if len(a.queryHistory) > 0 {
				a.queryHistoryPopupCursor--
				if a.queryHistoryPopupCursor < 0 {
					a.queryHistoryPopupCursor = 0
				}
			}
		case "enter":
			return a.applySelectedQueryHistoryEntry()
		}
		return a, nil
	case "keyModePopup":
		switch msg.String() {
		case "esc":
			a.activeModalName = "none"
		case "j", "down":
			a.keyModeCursor++
			if a.keyModeCursor > 1 {
				a.keyModeCursor = 1
			}
		case "k", "up":
			a.keyModeCursor--
			if a.keyModeCursor < 0 {
				a.keyModeCursor = 0
			}
		case "enter":
			a.vimMode = a.keyModeCursor == 1
			if a.vimMode {
				a.lastErr = "Key mode: vim"
			} else {
				a.lastErr = "Key mode: standard"
			}
			a.activeModalName = "none"
		}
		return a, nil
	case "timezonePopup":
		switch msg.String() {
		case "esc":
			a.activeModalName = "none"
		case "j", "down":
			a.timezoneCursor++
			if a.timezoneCursor > 1 {
				a.timezoneCursor = 1
			}
		case "k", "up":
			a.timezoneCursor--
			if a.timezoneCursor < 0 {
				a.timezoneCursor = 0
			}
		case "enter":
			if a.timezoneCursor == 0 {
				a.timezoneMode = "utc"
				a.formatter.SetUseLocalTime(false)
				a.lastErr = "Timezone: UTC"
			} else {
				a.timezoneMode = "local"
				a.formatter.SetUseLocalTime(true)
				a.lastErr = "Timezone: local"
			}
			a.activeModalName = "none"
		}
		return a, nil
	case "help":
		switch msg.String() {
		case "esc", "?":
			a.activeModalName = "none"
		case "tab", "right", "l", "j", "down":
			a.helpModal.NextSection()
		case "shift+tab", "left", "h", "k", "up":
			a.helpModal.PrevSection()
		}
		return a, nil
	}

	// Normal app keyboard handling when no modal is active
	switch msg.String() {
	// Quit
	case "ctrl+c":
		return a, tea.Quit

	// Query editor
	case "q":
		a.activeModalName = "query"
		a.queryModal.Show()
		a.queryModal.SetInput(a.state.CurrentQuery.Filter)
		a.queryModal.SetSuggestions(a.buildQuerySuggestions())
		a.queryHistoryCursor = -1
		return a, nil
	case "L":
		a.openQueryLibraryModal("none")
		return a, nil

	// Pane navigation (always available)
	case "left":
		a.panes.FocusPrevious()
		return a, nil
	case "right":
		a.panes.FocusNext()
		return a, nil
	case "h":
		if a.vimMode {
			a.panes.FocusPrevious()
		}
		return a, nil
	case "l":
		if a.vimMode {
			a.panes.FocusNext()
		}
		return a, nil

	// Log list navigation
	case "down":
		return a.handleScrollDown()
	case "up":
		return a.handleScrollUp()
	case "j":
		if a.vimMode {
			return a.handleScrollDown()
		}
		return a, nil
	case "k":
		if a.vimMode {
			return a.handleScrollUp()
		}
		return a, nil
	case "home":
		a.panes.LogList.JumpToTop()
		return a, nil
	case "end":
		a.jumpToLastLogEntry()
		return a, nil
	case "g":
		if a.vimMode {
			a.panes.LogList.JumpToTop()
		}
		return a, nil
	case "G":
		if a.vimMode {
			a.jumpToLastLogEntry()
		}
		return a, nil

	// Page navigation
	case "ctrl+f":
		a.panes.LogList.PageDown()
		return a, nil
	case "ctrl+b":
		a.panes.LogList.PageUp()
		return a, nil
	case "pgdown":
		a.panes.LogList.PageDown()
		return a, nil
	case "pgup":
		a.panes.LogList.PageUp()
		return a, nil

	// Actions
	case "t":
		a.activeModalName = "timeRange"
		a.state.UIState.ActiveModal = "timeRange"
		return a, nil
	case "f":
		a.activeModalName = "severity"
		a.state.UIState.ActiveModal = "severity"
		a.syncSeverityPanelFromState()
		return a, nil
	case "e":
		a.activeModalName = "export"
		a.state.UIState.ActiveModal = "export"
		return a, nil
	case "s":
		a.state.UIState.ActiveModal = "share"
		return a, nil
	case "m":
		a.state.StreamState.Enabled = !a.state.StreamState.Enabled
		return a, nil
	case "f6":
		if a.vimMode {
			a.keyModeCursor = 1
		} else {
			a.keyModeCursor = 0
		}
		a.activeModalName = "keyModePopup"
		return a, nil
	case "f7":
		if a.timezoneMode == "local" {
			a.timezoneCursor = 1
		} else {
			a.timezoneCursor = 0
		}
		a.activeModalName = "timezonePopup"
		return a, nil
	case "f8":
		a.toggleLogOrder()
		return a, nil
	case "ctrl+a":
		a.autoLoadAll = !a.autoLoadAll
		if a.autoLoadAll {
			a.lastErr = "Auto-load all pages: ON"
		} else {
			a.lastErr = "Auto-load all pages: OFF"
		}
		return a, nil
	case "r":
		if a.queryExec != nil {
			a.bypassNextCache = true
			return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(""))
		}
		return a, nil
	case "?":
		a.activeModalName = "help"
		a.helpModal.SetVisible(true)
		return a, nil
	case "esc":
		a.activeModalName = "none"
		a.state.UIState.ActiveModal = "none"
		return a, nil
	case "enter":
		if len(a.state.LogListState.Logs) > 0 {
			a.activeModalName = "details"
			a.detailScroll = 0
		}
		return a, nil
	case "ctrl+d":
		if len(a.state.LogListState.Logs) > 0 {
			if a.activeModalName == "details" {
				a.activeModalName = "none"
			} else {
				a.activeModalName = "details"
				a.detailScroll = 0
			}
		}
		return a, nil
	case "ctrl+p":
		if len(a.state.LogListState.Logs) > 0 {
			a.activeModalName = "detailPopup"
			a.resetDetailPopupState()
		}
		return a, nil
	case "ctrl+o":
		if len(a.state.LogListState.Logs) > 0 {
			return a, a.openSelectedLogInEditorCmd()
		}
		return a, nil
	case "ctrl+l":
		if len(a.state.LogListState.Logs) > 0 {
			return a, a.openLogListInEditorCmd()
		}
		return a, nil
	case "p", "P":
		a.availableProjects = collectProjects(a.state.CurrentProject)
		a.activeModalName = "projectPopup"
		a.projectCursor = 0
		if a.projectListFn != nil {
			a.loadingProjects = true
			return a, a.runProjectListCmd()
		}
		return a, nil
	case "ctrl+L", "alt+l", "ctrl+shift+l", "ctrl+Shift+l":
		if len(a.state.LogListState.Logs) > 0 {
			return a, a.openLogListCSVInEditorCmd()
		}
		return a, nil
	}

	return a, nil
}

// handleQueryModalInput handles input when query modal is active
func (a *App) handleQueryModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle pasted/typed runes first so newline in paste becomes text, not submit.
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		for _, r := range msg.Runes {
			switch r {
			case '\r', '\n':
				a.queryModal.HandleKey("newline")
			default:
				a.queryModal.HandleKey(string(r))
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "esc":
		a.activeModalName = "none"
		a.queryModal.Hide()
		a.queryHistoryCursor = -1
		return a, nil
	case "ctrl+r":
		a.openQueryHistoryModal("query")
		return a, nil
	case "ctrl+g":
		a.openQueryHistoryModal("query")
		return a, nil
	case "ctrl+y":
		a.openQueryLibraryModal("query")
		return a, nil
	case "ctrl+s", "ctrl+shift+s":
		a.saveCurrentQueryToLibrary()
		return a, nil
	case "ctrl+/", "ctrl+?", "ctrl+_":
		a.queryModal.HandleKey("toggle-comment")
		return a, nil
	case "ctrl+a":
		a.queryModal.HandleKey("select-all")
		return a, nil
	case "ctrl+home", "alt+a":
		a.queryModal.HandleKey("line-home")
		return a, nil
	case "ctrl+e":
		a.queryModal.HandleKey("line-end")
		return a, nil
	case "ctrl+left", "alt+b":
		a.queryModal.HandleKey("word-left")
		return a, nil
	case "ctrl+right", "alt+f":
		a.queryModal.HandleKey("word-right")
		return a, nil
	case "ctrl+w", "alt+backspace":
		a.queryModal.HandleKey("delete-word-left")
		return a, nil
	case "ctrl+delete", "alt+d":
		a.queryModal.HandleKey("delete-word-right")
		return a, nil
	case "ctrl+d":
		a.queryModal.HandleKey("duplicate-line")
		return a, nil
	case "tab":
		a.queryModal.HandleKey("indent")
		return a, nil
	case "shift+tab", "backtab":
		a.queryModal.HandleKey("unindent")
		return a, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		filter := a.queryModal.GetInput()
		a.activeModalName = "none"
		a.queryModal.Hide()
		a.queryHistoryCursor = -1
		a.state.CurrentQuery.Filter = filter
		a.addQueryHistory(filter)

		// Execute the query if we have an executor
		if a.queryExec != nil {
			return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(filter))
		}
		return a, nil
	case tea.KeyBackspace:
		a.queryModal.HandleKey("backspace")
		return a, nil
	case tea.KeyDelete:
		a.queryModal.HandleKey("delete")
		return a, nil
	case tea.KeyLeft:
		a.queryModal.HandleKey("left")
		return a, nil
	case tea.KeyRight:
		a.queryModal.HandleKey("right")
		return a, nil
	case tea.KeyHome:
		a.queryModal.HandleKey("home")
		return a, nil
	case tea.KeyEnd:
		a.queryModal.HandleKey("end")
		return a, nil
	case tea.KeyUp:
		a.queryModal.HandleKey("up")
		return a, nil
	case tea.KeyDown:
		a.queryModal.HandleKey("down")
		return a, nil
	case tea.KeySpace:
		a.queryModal.HandleKey(" ")
		return a, nil
	}

	switch msg.String() {
	case "ctrl+n":
		a.queryModal.HandleKey("newline")
		return a, nil
	default:
		return a, nil
	}
}

func (a *App) handleScrollDown() (tea.Model, tea.Cmd) {
	wasAtBottom := a.currentWindowStart() >= a.maxWindowStart()
	anchor := a.currentSelectedIndex()
	a.panes.LogList.ScrollDown()
	if a.queryExec != nil && len(a.state.LogListState.Logs) > 0 && wasAtBottom {
		if a.logOrder == "latest_bottom" && !a.loadingNewer {
			a.loadingNewer = true
			a.state.LogListState.IsLoading = true
			return a, a.runQueryCmdWithAnchor(a.buildNewerFilter(), "append", true, anchor)
		}
		if a.logOrder != "latest_bottom" && !a.loadingOlder {
			a.loadingOlder = true
			a.state.LogListState.IsLoading = true
			return a, a.runQueryCmdWithAnchor(a.buildOlderFilter(), "append", true, anchor)
		}
	}
	return a, nil
}

func (a *App) handleScrollUp() (tea.Model, tea.Cmd) {
	prev := a.panes.LogList.scrollOffset
	a.panes.LogList.ScrollUp()
	if a.queryExec != nil && prev == 0 && len(a.state.LogListState.Logs) > 0 {
		if a.logOrder == "latest_bottom" && !a.loadingOlder {
			a.loadingOlder = true
			a.state.LogListState.IsLoading = true
			return a, a.runQueryCmd(a.buildOlderFilter(), "prepend")
		}
		if a.logOrder != "latest_bottom" && !a.loadingNewer {
			a.loadingNewer = true
			a.state.LogListState.IsLoading = true
			return a, a.runQueryCmd(a.buildNewerFilter(), "prepend")
		}
	}
	return a, nil
}

func (a *App) jumpToLastLogEntry() {
	if len(a.state.LogListState.Logs) == 0 {
		a.panes.LogList.scrollOffset = 0
		return
	}
	// Keep selected row on the true last item, not at window start.
	a.panes.LogList.scrollOffset = len(a.state.LogListState.Logs) - 1
}

// handleTimePickerInput handles input when time picker modal is active
func (a *App) handleTimePickerInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.timePicker.IsCustomSelected() {
		switch msg.String() {
		case "esc":
			a.activeModalName = "none"
			return a, nil
		case "enter":
			err := a.timePicker.ApplyCustomInputs()
			if err != nil {
				a.timePicker.SetError(err.Error())
				return a, nil
			}
			err = a.timePicker.ApplyToFilterState(&a.state.FilterState)
			if err == nil {
				a.activeModalName = "none"
				if a.queryExec != nil {
					return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(""))
				}
			} else {
				a.timePicker.SetError(err.Error())
			}
			return a, nil
		case "h", "l", "tab":
			a.timePicker.ToggleCustomField()
			return a, nil
		case "j", "down":
			a.timePicker.ShiftCustomFocused(-15 * time.Minute)
			return a, nil
		case "k", "up":
			a.timePicker.ShiftCustomFocused(15 * time.Minute)
			return a, nil
		case "J":
			a.timePicker.ShiftCustomFocused(-1 * time.Hour)
			return a, nil
		case "K":
			a.timePicker.ShiftCustomFocused(1 * time.Hour)
			return a, nil
		case "backspace":
			a.timePicker.BackspaceFocusedInput()
			return a, nil
		case "ctrl+u":
			a.timePicker.ClearFocusedInput()
			return a, nil
		case "c":
			a.timePicker.ClearFocusedInput()
			return a, nil
		}
		if len(msg.Runes) > 0 {
			for _, r := range msg.Runes {
				if strings.ContainsRune("0123456789-: TZ+.", r) {
					a.timePicker.AppendToFocusedInput(string(r))
				}
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "esc":
		a.activeModalName = "none"
		return a, nil
	case "enter":
		// Apply selected time range
		err := a.timePicker.ApplyToFilterState(&a.state.FilterState)
		if err == nil {
			a.activeModalName = "none"
			if a.queryExec != nil {
				return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(""))
			}
		}
		return a, nil
	case "j", "down":
		a.timePicker.MoveSelection(1)
		return a, nil
	case "k", "up":
		a.timePicker.MoveSelection(-1)
		return a, nil
	}
	return a, nil
}

// handleSeverityFilterInput handles input when severity filter modal is active
func (a *App) handleSeverityFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	levels := a.severityFilter.GetSeverityLevels()
	if len(levels) == 0 {
		return a, nil
	}

	switch msg.String() {
	case "esc":
		a.activeModalName = "none"
		return a, nil
	case "enter":
		// Apply selected severity filter
		err := a.severityFilter.ApplyToFilterState(&a.state.FilterState)
		if err == nil {
			a.activeModalName = "none"
			if a.queryExec != nil {
				return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(""))
			}
		}
		return a, nil
	case "j", "down":
		a.severityCursor++
		if a.severityCursor >= len(levels) {
			a.severityCursor = len(levels) - 1
		}
		return a, nil
	case "k", "up":
		a.severityCursor--
		if a.severityCursor < 0 {
			a.severityCursor = 0
		}
		return a, nil
	case "space", " ":
		mode := a.severityFilter.GetMode()
		current := levels[a.severityCursor]
		if mode == "individual" {
			_ = a.severityFilter.ToggleLevel(current)
		} else {
			_ = a.severityFilter.SetMinimumLevel(current)
		}
		return a, nil
	case "a":
		// Select all
		a.severityFilter.SelectAllLevels()
		return a, nil
	case "d":
		// Deselect all
		a.severityFilter.DeselectAllLevels()
		return a, nil
	case "m":
		// Toggle mode (individual vs range)
		mode := a.severityFilter.GetMode()
		if mode == "individual" {
			a.severityFilter.SetMode("range")
			_ = a.severityFilter.SetMinimumLevel(levels[a.severityCursor])
		} else {
			a.severityFilter.SetMode("individual")
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleExportInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		a.activeModalName = "none"
		return a, nil
	}

	logs := a.state.LogListState.Logs
	if len(logs) == 0 {
		a.lastErr = "No logs available to export"
		a.activeModalName = "none"
		return a, nil
	}

	now := time.Now().Format("20060102_150405")
	base := filepath.Join(".", fmt.Sprintf("logs_%s", now))

	var err error
	switch msg.String() {
	case "1":
		path := base + ".csv"
		err = a.exporter.ExportToCSV(logs, path)
		if err == nil {
			a.lastErr = "Exported CSV: " + path
		}
	case "2":
		path := base + ".json"
		err = a.exporter.ExportToJSON(logs, path, true)
		if err == nil {
			a.lastErr = "Exported JSON: " + path
		}
	case "3":
		path := base + ".jsonl"
		err = a.exporter.ExportToJSONL(logs, path)
		if err == nil {
			a.lastErr = "Exported JSONL: " + path
		}
	case "4":
		path := base + ".txt"
		err = a.exporter.ExportToText(logs, path)
		if err == nil {
			a.lastErr = "Exported text: " + path
		}
	default:
		return a, nil
	}

	if err != nil {
		a.lastErr = "Export failed: " + err.Error()
	}
	a.activeModalName = "none"
	return a, nil
}

// renderTimePickerModal renders the time picker modal
func (a *App) renderTimePickerModal() string {
	presets := a.timePicker.GetPresets()
	selectedIdx := a.timePicker.GetSelectedIdx()
	popupWidth := minInt(maxInt(56, a.width-24), 120)

	var sb strings.Builder
	sb.WriteString(a.popupTop(popupWidth, "TIME RANGE"))

	for i, preset := range presets {
		prefix := "  "
		if i == selectedIdx {
			prefix = "▶ "
		}
		sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("%s%-20s", prefix, preset.Name)))
	}

	if a.timePicker.IsCustomSelected() {
		a.timePicker.EnsureCustomDefaults()
		startInput, endInput := a.timePicker.GetCustomInputs()
		startPrefix := " "
		endPrefix := " "
		if a.timePicker.GetCustomField() == 0 {
			startPrefix = "▶"
		} else {
			endPrefix = "▶"
		}
		sb.WriteString(a.popupSeparator(popupWidth, '━'))
		sb.WriteString(a.popupLine(popupWidth, "Custom Range (UTC):"))
		sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("%s Start: %s", startPrefix, startInput)))
		sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("%s End:   %s", endPrefix, endInput)))
	}

	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	if a.timePicker.IsCustomSelected() {
		sb.WriteString(a.popupLine(popupWidth, "Type exact datetime (YYYY-MM-DD HH:MM:SS or RFC3339)"))
		sb.WriteString(a.popupLine(popupWidth, "h/l/Tab switch field | c clear | Backspace edit | Ctrl+u clear | j/k +/-15m | J/K +/-1h"))
		sb.WriteString(a.popupLine(popupWidth, "Enter apply | Esc cancel"))
	} else {
		sb.WriteString(a.popupLine(popupWidth, "Press j/k to navigate, Enter to select, Esc to cancel"))
	}

	if errMsg := a.timePicker.GetError(); errMsg != "" {
		if len(errMsg) > popupWidth-6 {
			errMsg = errMsg[:popupWidth-9] + "..."
		}
		sb.WriteString(a.popupLine(popupWidth, "Error: "+errMsg))
	}
	sb.WriteString(a.popupBottom(popupWidth, '━'))

	return sb.String()
}

// renderSeverityFilterModal renders the severity filter modal
func (a *App) renderSeverityFilterModal() string {
	levels := a.severityFilter.GetSeverityLevels()
	mode := a.severityFilter.GetMode()
	popupWidth := minInt(maxInt(56, a.width-24), 120)

	var sb strings.Builder
	sb.WriteString(a.popupTop(popupWidth, "SEVERITY FILTER"))
	sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("Mode: %s (press 'm' to toggle)", mode)))
	sb.WriteString(a.popupSeparator(popupWidth, '━'))

	if mode == "individual" {
		for i, level := range levels {
			checked := "☐"
			if a.severityFilter.IsLevelSelected(level) {
				checked = "☑"
			}
			prefix := "  "
			if i == a.severityCursor {
				prefix = "▶ "
			}
			levelText := fmt.Sprintf("%s%s %s", prefix, checked, level)
			if i == a.severityCursor {
				levelText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Render(levelText)
			}
			sb.WriteString(a.popupLine(popupWidth, levelText))
		}
		sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("Selected: %d", a.severityFilter.CountSelectedLevels())))
	} else {
		minLevel := a.severityFilter.GetMinimumLevel()
		sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("Minimum level: %s", minLevel)))
		for i, level := range levels {
			prefix := "  "
			if i == a.severityCursor {
				prefix = "▶ "
			}
			active := " "
			if level == minLevel {
				active = "●"
			}
			levelText := fmt.Sprintf("%s%s %s", prefix, active, level)
			if i == a.severityCursor {
				levelText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Render(levelText)
			}
			sb.WriteString(a.popupLine(popupWidth, levelText))
		}
		sb.WriteString(a.popupLine(popupWidth, "(Shows logs at this level and above)"))
	}

	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, "j/k move | Space toggle/set | m mode | a all | d none | Enter apply | Esc cancel"))
	sb.WriteString(a.popupBottom(popupWidth, '━'))

	return sb.String()
}

func (a *App) syncSeverityPanelFromState() {
	levels := a.severityFilter.GetSeverityLevels()
	if len(levels) == 0 {
		return
	}

	fs := a.state.FilterState.Severity
	if fs.Mode == "range" {
		_ = a.severityFilter.SetMode("range")
		if fs.MinLevel != "" {
			_ = a.severityFilter.SetMinimumLevel(fs.MinLevel)
		}
	} else {
		_ = a.severityFilter.SetMode("individual")
		a.severityFilter.DeselectAllLevels()
		if len(fs.Levels) == 0 {
			// default to all selected so modal starts in a usable state
			a.severityFilter.SelectAllLevels()
		} else {
			for _, lvl := range fs.Levels {
				_ = a.severityFilter.SetLevel(lvl, true)
			}
		}
	}

	a.severityCursor = 0
	for i, lvl := range levels {
		if (a.severityFilter.GetMode() == "range" && lvl == a.severityFilter.GetMinimumLevel()) ||
			(a.severityFilter.GetMode() == "individual" && a.severityFilter.IsLevelSelected(lvl)) {
			a.severityCursor = i
			break
		}
	}
}

func (a *App) renderExportModal() string {
	var sb strings.Builder
	sb.WriteString("┏━━ EXPORT LOGS " + strings.Repeat("━", a.width-17) + "\n")
	sb.WriteString("┃ Choose export format:\n")
	sb.WriteString("┃   1) CSV\n")
	sb.WriteString("┃   2) JSON (pretty)\n")
	sb.WriteString("┃   3) JSONL\n")
	sb.WriteString("┃   4) Plain text\n")
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	sb.WriteString("┃ Files are saved in the current directory as logs_YYYYMMDD_HHMMSS.*\n")
	sb.WriteString("┃ Press 1-4 to export, Esc to cancel\n")
	sb.WriteString("┗" + strings.Repeat("━", a.width-1) + "┛\n")
	return sb.String()
}

func (a *App) renderDetailsModal() string {
	if len(a.state.LogListState.Logs) == 0 {
		return ""
	}

	idx := a.panes.LogList.scrollOffset
	if idx < 0 {
		idx = 0
	}
	if idx >= len(a.state.LogListState.Logs) {
		idx = len(a.state.LogListState.Logs) - 1
	}
	entry := a.state.LogListState.Logs[idx]
	details := a.formatter.FormatCompact(entry)

	lines := strings.Split(details, "\n")
	maxLines := 10
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var sb strings.Builder
	sb.WriteString("┏━━ LOG DETAILS " + strings.Repeat("━", a.width-17) + "\n")
	sb.WriteString(fmt.Sprintf("┃ Entry %d/%d\n", idx+1, len(a.state.LogListState.Logs)))
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	for _, line := range lines {
		if len(line) > a.width-4 {
			line = line[:a.width-7] + "..."
		}
		sb.WriteString("┃ " + line + "\n")
	}
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	sb.WriteString("┃ Esc/Enter: close\n")
	sb.WriteString("┗" + strings.Repeat("━", a.width-1) + "┛\n")
	return sb.String()
}

func (a *App) renderGraphPanel() string {
	points := a.timelineBuilder.BuildTimeline(a.state.LogListState.Logs)
	spark := a.timelineBuilder.RenderSparkline(points, a.width-24)
	if spark == "" {
		spark = "No timeline data"
	}
	dist := a.timelineBuilder.BuildSeverityDistribution(a.state.LogListState.Logs)
	crit := dist["ERROR"] + dist["CRITICAL"] + dist["ALERT"] + dist["EMERGENCY"]
	warn := dist["WARNING"]
	info := dist["INFO"] + dist["NOTICE"] + dist["DEBUG"] + dist["DEFAULT"]
	rangeText := "Range: n/a"
	if len(a.state.LogListState.Logs) > 0 {
		oldest := a.oldestLoadedTimestamp()
		newest := a.newestLoadedTimestamp()
		rangeText = fmt.Sprintf("Range: %s -> %s", a.displayTime(oldest).Format("2006-01-02 15:04:05"), a.displayTime(newest).Format("2006-01-02 15:04:05"))
	}

	var sb strings.Builder
	sb.WriteString(a.panelSeparator('─'))
	sb.WriteString(a.panelLine(fmt.Sprintf("%s %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorGCPBlueLight)).Render("Timeline"), spark)))
	sb.WriteString(a.panelLine(fmt.Sprintf("%s Critical:%d  Warning:%d  Info/Other:%d",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorGCPBlue)).Render("Severity Mix"),
		crit, warn, info)))
	sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralSubtle)).Render(rangeText)))
	return sb.String()
}

func (a *App) renderLogsPanel(height int) (string, int, int) {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorGCPBlueLight)).Render("LOG STREAM")
	sb.WriteString(a.panelTop())
	sb.WriteString(a.panelLine(fmt.Sprintf("%s (%d)", title, len(a.state.LogListState.Logs))))
	sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralSubtle)).Render("IDX   TIMESTAMP           SEV      MESSAGE")))

	visibleRows := maxInt(1, height-1)
	start := a.currentWindowStart()
	maxStart := maxInt(0, len(a.state.LogListState.Logs)-visibleRows)
	if start > maxStart {
		start = maxStart
	}
	end := start + visibleRows
	if end > len(a.state.LogListState.Logs) {
		end = len(a.state.LogListState.Logs)
	}

	for i := start; i < end; i++ {
		log := a.state.LogListState.Logs[i]
		timePart := a.displayTime(log.Timestamp).Format("2006-01-02 15:04:05")
		sevBadge := a.styleSeverityBadge(log.Severity)
		msgMax := maxInt(12, a.width-47)
		msg := log.Message
		if len(msg) > msgMax {
			msg = msg[:msgMax-3] + "..."
		}

		row := fmt.Sprintf("%-4d  %s  %s  %s", i+1, timePart, sevBadge, msg)
		if i == a.currentSelectedIndex() {
			row = a.styleSelectedRow(row)
		} else {
			row = a.styleSeverityRow(log.Severity, row)
		}
		sb.WriteString(a.panelLine(row))
	}

	for i := end - start; i < height-1; i++ {
		sb.WriteString(a.panelLine(""))
	}
	return sb.String(), start + 1, end
}

func (a *App) renderQueryPanel(query string, editing bool) string {
	var sb strings.Builder
	title := "QUERY EDITOR"
	if editing {
		title = "QUERY EDITOR [EDITING]"
	}
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color(colorGCPBlue)).Padding(0, 1).Render(title)
	sb.WriteString(a.panelTop())
	sb.WriteString(a.panelLine(titleStyled))

	if strings.TrimSpace(query) == "" {
		query = "No filter. Press q to edit."
	}
	maxQueryLines := minInt(16, maxInt(6, a.height/3))
	lines := wrapMultiline(query, maxInt(30, a.width-6), maxQueryLines)
	for _, line := range lines {
		if editing && (strings.Contains(line, "│") || a.queryModal.SelectAllActive()) {
			sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color(colorGCPBlueDark)).Render(line)))
		} else {
			sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralText)).Render(line)))
		}
	}
	hint := "Enter run | Ctrl+A all | Ctrl+/ comment | Ctrl+R history | Ctrl+S save | Ctrl+Y library"
	sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color(colorGCPBlueLight)).Render(hint)))
	return sb.String()
}

func (a *App) renderStatusPanel(windowStart, windowEnd int) string {
	var sb strings.Builder
	sb.WriteString(a.panelSeparator('━'))
	total := len(a.state.LogListState.Logs)
	if total == 0 {
		windowStart = 0
		windowEnd = 0
	}
	loadMode := "paged"
	if a.autoLoadAll {
		loadMode = "all"
	}
	streamMode := "off"
	if a.state.StreamState.Enabled {
		streamMode = "on"
	}
	keyMode := "std"
	if a.vimMode {
		keyMode = "vim"
	}
	tzMode := strings.ToUpper(a.timezoneMode)
	sb.WriteString(a.panelLine(fmt.Sprintf("%d-%d/%d  %s  sev:%s  load:%s  stream:%s  keys:%s  tz:%s  order:%s  cache:%d  ?",
		windowStart, windowEnd, total, a.getTimeRangeLabel(), a.getSeveritySummary(), loadMode, streamMode, keyMode, tzMode, a.logOrderLabel(), len(a.cachedQueryRecords()))))
	if a.lastErr != "" {
		errLine := a.lastErr
		if len(errLine) > a.width-4 {
			errLine = errLine[:a.width-7] + "..."
		}
		sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color(colorGCPError)).Render(errLine)))
	}
	sb.WriteString(a.panelBottom('━'))
	return sb.String()
}

func (a *App) panelInnerWidth() int {
	return maxInt(1, a.width-4)
}

func (a *App) panelTop() string {
	return "┏" + strings.Repeat("━", maxInt(0, a.width-2)) + "┓\n"
}

func (a *App) panelSeparator(fill rune) string {
	return "┣" + strings.Repeat(string(fill), maxInt(0, a.width-2)) + "┫\n"
}

func (a *App) panelBottom(fill rune) string {
	return "┗" + strings.Repeat(string(fill), maxInt(0, a.width-2)) + "┛\n"
}

func (a *App) panelLine(content string) string {
	content = strings.TrimSuffix(content, "\n")
	line := lipgloss.NewStyle().
		Width(a.panelInnerWidth()).
		MaxWidth(a.panelInnerWidth()).
		Render(content)
	return "┃ " + line + " ┃\n"
}

func (a *App) popupInnerWidth(width int) int {
	return maxInt(1, width-4)
}

func (a *App) popupTop(width int, title string) string {
	label := " " + strings.TrimSpace(title) + " "
	maxLabel := maxInt(0, width-2)
	if lipgloss.Width(label) > maxLabel {
		label = lipgloss.NewStyle().MaxWidth(maxLabel).Render(label)
	}
	return "┏" + label + strings.Repeat("━", maxInt(0, width-2-lipgloss.Width(label))) + "┓\n"
}

func (a *App) popupSeparator(width int, fill rune) string {
	return "┣" + strings.Repeat(string(fill), maxInt(0, width-2)) + "┫\n"
}

func (a *App) popupBottom(width int, fill rune) string {
	return "┗" + strings.Repeat(string(fill), maxInt(0, width-2)) + "┛\n"
}

func (a *App) popupLine(width int, content string) string {
	content = strings.TrimSuffix(content, "\n")
	line := lipgloss.NewStyle().
		Width(a.popupInnerWidth(width)).
		MaxWidth(a.popupInnerWidth(width)).
		Render(content)
	return "┃ " + line + " ┃\n"
}

func (a *App) renderTopBar() string {
	project := a.state.CurrentProject
	if strings.TrimSpace(project) == "" {
		project = "unknown-project"
	}
	queryMode := "ready"
	if a.activeModalName == "query" {
		queryMode = "editing query"
	}
	if a.state.LogListState.IsLoading {
		queryMode = "loading"
	}
	keys := "std"
	if a.vimMode {
		keys = "vim"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorGCPBlueLight)).Render("GCP Log Explorer")
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralSubtle)).Render(" | ")
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralText))
	line := title + sep + metaStyle.Render(fmt.Sprintf("project:%s", project)) +
		sep + metaStyle.Render(fmt.Sprintf("mode:%s", queryMode)) +
		sep + metaStyle.Render(fmt.Sprintf("load:%s", a.loadingStateShort())) +
		sep + metaStyle.Render(fmt.Sprintf("keys:%s", keys)) +
		sep + metaStyle.Render(fmt.Sprintf("tz:%s", strings.ToUpper(a.timezoneMode))) +
		sep + metaStyle.Render(fmt.Sprintf("order:%s", a.logOrderLabel()))
	if lipgloss.Width(line) > a.width {
		if a.width > 3 {
			line = line[:a.width-3] + "..."
		} else {
			line = line[:a.width]
		}
	}
	if lipgloss.Width(line) < a.width {
		line += strings.Repeat(" ", a.width-lipgloss.Width(line))
	}
	return line + "\n"
}

func (a *App) addQueryHistory(filter string) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return
	}
	for i, existing := range a.queryHistory {
		if existing == filter {
			a.queryHistory = append([]string{filter}, append(a.queryHistory[:i], a.queryHistory[i+1:]...)...)
			if a.persistHistoryFn != nil {
				if err := a.persistHistoryFn(filter, a.state.CurrentProject); err != nil {
					a.lastErr = "Persist history failed: " + err.Error()
				}
			}
			return
		}
	}
	a.queryHistory = append([]string{filter}, a.queryHistory...)
	if len(a.queryHistory) > 25 {
		a.queryHistory = a.queryHistory[:25]
	}
	if a.persistHistoryFn != nil {
		if err := a.persistHistoryFn(filter, a.state.CurrentProject); err != nil {
			a.lastErr = "Persist history failed: " + err.Error()
		}
	}
}

func (a *App) cycleQueryHistory(delta int) {
	if len(a.queryHistory) == 0 {
		return
	}
	if a.queryHistoryCursor < 0 {
		if delta < 0 {
			a.queryHistoryCursor = 0
		} else {
			a.queryHistoryCursor = len(a.queryHistory) - 1
		}
	} else {
		a.queryHistoryCursor += delta
		if a.queryHistoryCursor < 0 {
			a.queryHistoryCursor = 0
		}
		if a.queryHistoryCursor >= len(a.queryHistory) {
			a.queryHistoryCursor = len(a.queryHistory) - 1
		}
	}
	a.queryModal.SetInput(a.queryHistory[a.queryHistoryCursor])
}

func (a *App) buildQuerySuggestions() []string {
	suggestions := make([]string, 0, len(a.queryHistory)+len(a.queryModal.suggestions))
	seen := map[string]bool{}
	for _, q := range a.queryHistory {
		key := strings.TrimSpace(q)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		suggestions = append(suggestions, q)
		if len(suggestions) >= 8 {
			return suggestions
		}
	}
	for _, q := range a.queryModal.suggestions {
		key := strings.TrimSpace(q)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		suggestions = append(suggestions, q)
		if len(suggestions) >= 8 {
			break
		}
	}
	return suggestions
}

func (a *App) getLoadingStatus() string {
	if !a.state.LogListState.IsLoading {
		return "idle"
	}
	if a.loadingOlder && a.loadingNewer {
		return "fetching newer and older pages"
	}
	if a.loadingOlder {
		return "fetching older logs"
	}
	if a.loadingNewer {
		return "fetching newer logs"
	}
	if a.autoLoadAll {
		return "running query (auto-load all pages)"
	}
	return "running query"
}

func (a *App) loadingStateShort() string {
	if !a.state.LogListState.IsLoading {
		return "idle"
	}
	if a.loadingOlder && a.loadingNewer {
		return "older+newer"
	}
	if a.loadingOlder {
		return "older"
	}
	if a.loadingNewer {
		return "newer"
	}
	return "query"
}

func (a *App) logOrderLabel() string {
	if a.logOrder == "latest_bottom" {
		return "bottom"
	}
	return "top"
}

func sanitizeFilterForExecution(filter string) string {
	filter = strings.ReplaceAll(filter, "\r\n", "\n")
	filter = strings.ReplaceAll(filter, "\r", "\n")
	lines := strings.Split(filter, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		clean = append(clean, line)
	}
	return strings.TrimSpace(strings.Join(clean, "\n"))
}

func (a *App) renderDetailsPanel() string {
	if len(a.state.LogListState.Logs) == 0 {
		return ""
	}

	idx := a.currentSelectedIndex()
	entry := a.state.LogListState.Logs[idx]
	details := a.formatter.FormatLogDetails(entry)
	lines := strings.Split(details, "\n")
	maxLines := 10
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var sb strings.Builder
	sb.WriteString(a.panelSeparator('─'))
	sb.WriteString(a.panelLine(fmt.Sprintf("%s %d/%d", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")).Render("DETAILS"), idx+1, len(a.state.LogListState.Logs))))
	for _, line := range lines {
		if len(line) > a.width-4 {
			line = line[:a.width-7] + "..."
		}
		sb.WriteString(a.panelLine(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(line)))
	}
	sb.WriteString(a.panelLine("Esc/Enter/Ctrl+D: close | Ctrl+P: popup | Ctrl+O: open entry | Ctrl+L: open list"))
	return sb.String()
}

func (a *App) renderDetailPopup() string {
	entry := a.getSelectedLog()
	if entry == nil {
		return ""
	}

	popupWidth := minInt(maxInt(88, a.width-18), 150)
	lines, selectedIndex := a.detailPopupLines(*entry, popupWidth)
	visibleHeight := maxInt(8, a.height-10)
	start := a.detailScroll
	if start < 0 {
		start = 0
	}
	if start > len(lines)-1 {
		start = maxInt(0, len(lines)-1)
	}
	end := start + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	sb.WriteString(a.popupTop(popupWidth, "FULL LOG POPUP"))
	sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("Entry %d/%d  Mode:%s  Scroll %d/%d", a.currentSelectedIndex()+1, len(a.state.LogListState.Logs), a.detailViewMode, start+1, maxInt(1, len(lines)))))
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	for i := start; i < end; i++ {
		line := lines[i]
		prefix := "  "
		if i == selectedIndex {
			prefix = "▶ "
		}
		sb.WriteString(a.popupLine(popupWidth, prefix+line))
	}
	for i := end; i < start+visibleHeight; i++ {
		sb.WriteString(a.popupLine(popupWidth, ""))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	if selectedPath, selectedType := a.selectedDetailNodeInfo(); selectedPath != "" {
		meta := fmt.Sprintf("selected:%s (%s)", selectedPath, selectedType)
		sb.WriteString(a.popupLine(popupWidth, meta))
	}
	sb.WriteString(a.popupLine(popupWidth, "j/k:move  h/l:collapse/expand  z/Z:collapse/expand all  v/tab:mode  y/Y:copy  Ctrl+E:open payload"))
	sb.WriteString(a.popupLine(popupWidth, "Ctrl+O:open entry  Ctrl+L:open list(JSON)  Ctrl+Shift+L/Alt+L:open list(CSV)  Esc/Ctrl+P:close"))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) renderProjectDropdown() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(44, a.width-20), 100)
	sb.WriteString(a.popupTop(popupWidth, "PROJECT SELECTOR"))
	if a.loadingProjects {
		sb.WriteString(a.popupLine(popupWidth, "Discovering projects from gcloud account..."))
	}
	if len(a.availableProjects) == 0 {
		sb.WriteString(a.popupLine(popupWidth, "No projects discovered yet"))
		sb.WriteString(a.popupSeparator(popupWidth, '━'))
		sb.WriteString(a.popupLine(popupWidth, "Enter: switch project | Esc: cancel"))
		sb.WriteString(a.popupBottom(popupWidth, '━'))
		return sb.String()
	}

	maxVisible := maxInt(6, a.height-14)
	start := 0
	if a.projectCursor >= maxVisible {
		start = a.projectCursor - maxVisible + 1
	}
	end := minInt(len(a.availableProjects), start+maxVisible)
	for i := start; i < end; i++ {
		project := a.availableProjects[i]
		prefix := "  "
		if i == a.projectCursor {
			prefix = "▶ "
		}
		line := fmt.Sprintf("%s%s", prefix, project)
		if len(line) > popupWidth-4 {
			line = line[:popupWidth-7] + "..."
		}
		if i == a.projectCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelectionFG)).Background(lipgloss.Color(colorSelectionBG)).Render(line)
		}
		sb.WriteString(a.popupLine(popupWidth, line))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("Showing %d-%d of %d | j/k move | Enter select | Esc close", start+1, end, len(a.availableProjects))))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) renderQueryLibraryPopup() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(44, a.width-20), 110)
	sb.WriteString(a.popupTop(popupWidth, "QUERY LIBRARY"))
	if len(a.queryLibrary) == 0 {
		sb.WriteString(a.popupLine(popupWidth, "No saved queries yet"))
		sb.WriteString(a.popupSeparator(popupWidth, '━'))
		sb.WriteString(a.popupLine(popupWidth, "Ctrl+S in query editor to save | Esc close"))
		sb.WriteString(a.popupBottom(popupWidth, '━'))
		return sb.String()
	}
	maxVisible := maxInt(6, a.height-14)
	start := 0
	if a.queryLibraryCursor >= maxVisible {
		start = a.queryLibraryCursor - maxVisible + 1
	}
	end := minInt(len(a.queryLibrary), start+maxVisible)
	for i := start; i < end; i++ {
		item := a.queryLibrary[i]
		prefix := "  "
		if i == a.queryLibraryCursor {
			prefix = "▶ "
		}
		line := fmt.Sprintf("%s%s | %s", prefix, item.Name, strings.ReplaceAll(strings.TrimSpace(item.Filter), "\n", " "))
		if len(line) > popupWidth-4 {
			line = line[:popupWidth-7] + "..."
		}
		if i == a.queryLibraryCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelectionFG)).Background(lipgloss.Color(colorSelectionBG)).Render(line)
		}
		sb.WriteString(a.popupLine(popupWidth, line))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("%d-%d of %d | j/k move | Enter apply | Esc close", start+1, end, len(a.queryLibrary))))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) openQueryLibraryModal(previous string) {
	a.previousModalName = previous
	a.activeModalName = "queryLibrary"
	a.queryLibraryCursor = 0
}

func (a *App) openQueryHistoryModal(previous string) {
	a.previousModalName = previous
	a.activeModalName = "queryHistory"
	a.queryHistoryPopupCursor = 0
}

func (a *App) renderQueryHistoryPopup() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(44, a.width-20), 120)
	sb.WriteString(a.popupTop(popupWidth, "QUERY HISTORY"))
	if len(a.queryHistory) == 0 {
		sb.WriteString(a.popupLine(popupWidth, "No executed queries yet"))
		sb.WriteString(a.popupSeparator(popupWidth, '━'))
		sb.WriteString(a.popupLine(popupWidth, "Run a query first | Esc close"))
		sb.WriteString(a.popupBottom(popupWidth, '━'))
		return sb.String()
	}
	maxVisible := maxInt(6, a.height-14)
	start := 0
	if a.queryHistoryPopupCursor >= maxVisible {
		start = a.queryHistoryPopupCursor - maxVisible + 1
	}
	end := minInt(len(a.queryHistory), start+maxVisible)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == a.queryHistoryPopupCursor {
			prefix = "▶ "
		}
		queryLine := strings.ReplaceAll(strings.TrimSpace(a.queryHistory[i]), "\n", " ↩ ")
		line := fmt.Sprintf("%s%d) %s", prefix, i+1, queryLine)
		if len(line) > popupWidth-4 {
			line = line[:popupWidth-7] + "..."
		}
		if i == a.queryHistoryPopupCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelectionFG)).Background(lipgloss.Color(colorSelectionBG)).Render(line)
		}
		sb.WriteString(a.popupLine(popupWidth, line))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, fmt.Sprintf("%d-%d of %d | j/k or Ctrl+R/Ctrl+G | Enter apply | Esc close", start+1, end, len(a.queryHistory))))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) renderKeyModePopup() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(36, a.width-40), 56)
	sb.WriteString(a.popupTop(popupWidth, "KEY MODE"))
	options := []string{"standard", "vim"}
	for i, opt := range options {
		prefix := "  "
		if i == a.keyModeCursor {
			prefix = "▶ "
		}
		line := prefix + opt
		if i == a.keyModeCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelectionFG)).Background(lipgloss.Color(colorSelectionBG)).Render(line)
		}
		sb.WriteString(a.popupLine(popupWidth, line))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, "j/k move | Enter apply | Esc close"))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) renderTimezonePopup() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(36, a.width-40), 56)
	sb.WriteString(a.popupTop(popupWidth, "TIMEZONE"))
	options := []string{"UTC", "local"}
	for i, opt := range options {
		prefix := "  "
		if i == a.timezoneCursor {
			prefix = "▶ "
		}
		line := prefix + opt
		if i == a.timezoneCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorSelectionFG)).Background(lipgloss.Color(colorSelectionBG)).Render(line)
		}
		sb.WriteString(a.popupLine(popupWidth, line))
	}
	sb.WriteString(a.popupSeparator(popupWidth, '━'))
	sb.WriteString(a.popupLine(popupWidth, "j/k move | Enter apply | Esc close"))
	sb.WriteString(a.popupBottom(popupWidth, '━'))
	return sb.String()
}

func (a *App) applySelectedQueryHistoryEntry() (tea.Model, tea.Cmd) {
	if len(a.queryHistory) == 0 {
		a.lastErr = "Query history is empty"
		return a, nil
	}
	if a.queryHistoryPopupCursor < 0 {
		a.queryHistoryPopupCursor = 0
	}
	if a.queryHistoryPopupCursor >= len(a.queryHistory) {
		a.queryHistoryPopupCursor = len(a.queryHistory) - 1
	}
	selected := a.queryHistory[a.queryHistoryPopupCursor]
	a.state.CurrentQuery.Filter = selected
	if a.previousModalName == "query" {
		a.activeModalName = "query"
		a.queryModal.SetInput(selected)
		return a, nil
	}
	a.activeModalName = "none"
	if a.queryExec != nil {
		return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(selected))
	}
	return a, nil
}

func (a *App) applySelectedQueryLibraryEntry() (tea.Model, tea.Cmd) {
	if len(a.queryLibrary) == 0 {
		a.lastErr = "Query library is empty"
		return a, nil
	}
	if a.queryLibraryCursor < 0 {
		a.queryLibraryCursor = 0
	}
	if a.queryLibraryCursor >= len(a.queryLibrary) {
		a.queryLibraryCursor = len(a.queryLibrary) - 1
	}
	selected := a.queryLibrary[a.queryLibraryCursor]
	a.state.CurrentQuery.Filter = selected.Filter
	if a.previousModalName == "query" {
		a.activeModalName = "query"
		a.queryModal.SetInput(selected.Filter)
		return a, nil
	}
	a.activeModalName = "none"
	if a.queryExec != nil {
		return a, a.executePrimaryQueryCmd(a.buildEffectiveFilter(selected.Filter))
	}
	return a, nil
}

func (a *App) saveCurrentQueryToLibrary() {
	filter := sanitizeFilterForExecution(a.queryModal.GetInput())
	if strings.TrimSpace(filter) == "" {
		a.lastErr = "Cannot save empty query"
		return
	}
	record := config.SavedQueryRecord{
		Name:      deriveQueryLibraryName(filter),
		Filter:    filter,
		Project:   a.state.CurrentProject,
		UpdatedAt: time.Now(),
		UseCount:  1,
	}
	lib := config.QueryLibrary{Queries: a.queryLibrary}
	lib = config.UpsertSavedQuery(lib, record, 100)
	a.queryLibrary = lib.Queries
	if a.persistLibraryFn != nil {
		if err := a.persistLibraryFn(a.queryLibrary); err != nil {
			a.lastErr = "Save query library failed: " + err.Error()
			return
		}
	}
	a.lastErr = "Saved query to library"
}

func deriveQueryLibraryName(filter string) string {
	lines := strings.Split(filter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) > 42 {
			line = line[:42] + "..."
		}
		return line
	}
	return "Saved query"
}

func (a *App) renderCenteredPopup(base, popup string) string {
	_ = base
	popupLines := strings.Split(strings.TrimRight(popup, "\n"), "\n")
	if len(popupLines) == 0 {
		return base
	}

	header := ansiEscapeRegex.ReplaceAllString(strings.TrimSuffix(a.renderTopBar(), "\n"), "")
	headerWidth := lipgloss.Width(header)
	if headerWidth < a.width {
		header += strings.Repeat(" ", a.width-headerWidth)
	}
	if headerWidth > a.width {
		header = header[:a.width]
	}
	separator := strings.Repeat("─", maxInt(1, a.width))

	backdropLine := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Render(strings.Repeat(" ", maxInt(1, a.width)))
	screenHeight := maxInt(1, a.height-1)
	bodyHeight := maxInt(1, screenHeight-2)
	canvas := make([]string, 0, bodyHeight)
	for i := 0; i < bodyHeight; i++ {
		canvas = append(canvas, backdropLine)
	}

	startRow := maxInt(0, (bodyHeight-len(popupLines))/2)
	for i, line := range popupLines {
		row := startRow + i
		if row < 0 || row >= len(canvas) {
			continue
		}
		lineWidth := lipgloss.Width(line)
		if lineWidth > a.width {
			line = lipgloss.NewStyle().Width(a.width).MaxWidth(a.width).Render(line)
			lineWidth = lipgloss.Width(line)
		}
		leftPadCount := maxInt(0, (a.width-lineWidth)/2)
		rightPadCount := maxInt(0, a.width-leftPadCount-lineWidth)
		padStyle := lipgloss.NewStyle().Background(lipgloss.Color("235"))
		leftPad := padStyle.Render(strings.Repeat(" ", leftPadCount))
		rightPad := padStyle.Render(strings.Repeat(" ", rightPadCount))
		canvas[row] = leftPad + line + rightPad
	}

	return a.fitToViewport(header + "\n" + separator + "\n" + strings.Join(canvas, "\n"))
}

func (a *App) runQueryCmd(filter, mode string) tea.Cmd {
	return a.runQueryCmdWithAnchor(filter, mode, false, 0)
}

func (a *App) runQueryCmdWithAnchor(filter, mode string, preserveAnchor bool, anchorOffset int) tea.Cmd {
	return func() tea.Msg {
		logs, err := a.queryExec(filter)
		return queryResultMsg{
			filter:         filter,
			logs:           logs,
			err:            err,
			mode:           mode,
			preserveAnchor: preserveAnchor,
			anchorOffset:   anchorOffset,
		}
	}
}

func (a *App) runProjectListCmd() tea.Cmd {
	return func() tea.Msg {
		if a.projectListFn == nil {
			return projectListMsg{projects: nil, err: nil}
		}
		projects, err := a.projectListFn()
		return projectListMsg{projects: projects, err: err}
	}
}

func (a *App) executePrimaryQueryCmd(filter string) tea.Cmd {
	a.state.LogListState.IsLoading = true
	if !a.bypassNextCache {
		if logs, ok := a.lookupQueryResultCache(filter); ok {
			a.state.LogListState.IsLoading = false
			return func() tea.Msg {
				return queryResultMsg{
					filter:    filter,
					logs:      logs,
					err:       nil,
					mode:      "replace",
					fromCache: true,
				}
			}
		}
	}
	a.bypassNextCache = false
	if a.autoLoadAll {
		return a.runLoadAllCmd(filter)
	}
	return a.runQueryCmd(filter, "replace")
}

func (a *App) runLoadAllCmd(baseFilter string) tea.Cmd {
	return func() tea.Msg {
		if a.queryExec == nil {
			return queryResultMsg{filter: baseFilter, logs: []models.LogEntry{}, err: fmt.Errorf("query executor not configured"), mode: "replace"}
		}

		firstPage, err := a.queryExec(baseFilter)
		if err != nil {
			return queryResultMsg{filter: baseFilter, logs: []models.LogEntry{}, err: err, mode: "replace"}
		}

		all := mergeUniqueLogs([]models.LogEntry{}, firstPage, false)
		const maxPages = 200
		for page := 0; page < maxPages; page++ {
			if len(all) == 0 {
				break
			}
			oldest := all[len(all)-1].Timestamp
			timeClause := fmt.Sprintf("timestamp<%q", oldest.Format(time.RFC3339))
			nextFilter := baseFilter
			if strings.TrimSpace(nextFilter) == "" {
				nextFilter = timeClause
			} else {
				nextFilter = fmt.Sprintf("(%s) AND %s", nextFilter, timeClause)
			}

			nextPage, err := a.queryExec(nextFilter)
			if err != nil {
				return queryResultMsg{filter: baseFilter, logs: all, err: err, mode: "replace"}
			}
			if len(nextPage) == 0 {
				break
			}
			before := len(all)
			all = mergeUniqueLogs(all, nextPage, false)
			if len(all) == before {
				break
			}
		}

		return queryResultMsg{
			filter: baseFilter,
			logs:   all,
			err:    nil,
			mode:   "replace",
		}
	}
}

func (a *App) buildEffectiveFilter(baseFilter string) string {
	base := sanitizeFilterForExecution(baseFilter)
	if base == "" {
		base = sanitizeFilterForExecution(a.state.CurrentQuery.Filter)
	}

	builder := query.NewBuilder("")
	if base != "" {
		builder.AddCustomFilter(base)
	}

	if !a.state.FilterState.TimeRange.Start.IsZero() && !a.state.FilterState.TimeRange.End.IsZero() {
		builder.AddTimeRange(a.state.FilterState.TimeRange)
	}

	sf := a.state.FilterState.Severity
	if sf.Mode == "range" && sf.MinLevel != "" {
		builder.AddSeverity(sf)
	}
	if sf.Mode == "individual" && len(sf.Levels) > 0 {
		builder.AddSeverity(sf)
	}

	return builder.Build()
}

func (a *App) queryCacheKey(filter string) string {
	project := strings.TrimSpace(a.state.CurrentProject)
	filter = sanitizeFilterForExecution(filter)
	return project + "\n" + filter
}

func (a *App) lookupQueryResultCache(filter string) ([]models.LogEntry, bool) {
	if len(a.queryCache) == 0 {
		return nil, false
	}
	key := a.queryCacheKey(filter)
	entry, ok := a.queryCache[key]
	if !ok {
		return nil, false
	}
	if a.queryCacheTTL > 0 && time.Since(entry.StoredAt) > a.queryCacheTTL {
		delete(a.queryCache, key)
		a.persistQueryCache()
		return nil, false
	}
	return append([]models.LogEntry{}, entry.Logs...), true
}

func (a *App) storeQueryResultCache(filter string, logs []models.LogEntry) {
	if strings.TrimSpace(filter) == "" {
		return
	}
	key := a.queryCacheKey(filter)
	a.queryCache[key] = config.CachedQueryRecord{
		Key:      key,
		Filter:   sanitizeFilterForExecution(filter),
		Project:  strings.TrimSpace(a.state.CurrentProject),
		StoredAt: time.Now(),
		Logs:     append([]models.LogEntry{}, logs...),
	}
	entries := a.cachedQueryRecords()
	if a.queryCacheMax > 0 && len(entries) > a.queryCacheMax {
		entries = entries[:a.queryCacheMax]
	}
	a.queryCache = map[string]config.CachedQueryRecord{}
	for _, entry := range entries {
		a.queryCache[entry.Key] = entry
	}
	a.persistQueryCache()
}

func (a *App) cachedQueryRecords() []config.CachedQueryRecord {
	entries := make([]config.CachedQueryRecord, 0, len(a.queryCache))
	for _, entry := range a.queryCache {
		if a.queryCacheTTL > 0 && time.Since(entry.StoredAt) > a.queryCacheTTL {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StoredAt.After(entries[j].StoredAt)
	})
	return entries
}

func (a *App) persistQueryCache() {
	if a.persistCacheFn == nil {
		return
	}
	if err := a.persistCacheFn(a.cachedQueryRecords()); err != nil {
		a.lastErr = "Persist cache failed: " + err.Error()
	}
}

func (a *App) buildOlderFilter() string {
	base := a.buildEffectiveFilter("")
	if len(a.state.LogListState.Logs) == 0 {
		return base
	}
	oldest := a.oldestLoadedTimestamp()
	timeClause := fmt.Sprintf("timestamp<%q", oldest.Format(time.RFC3339))
	if base == "" {
		return timeClause
	}
	return fmt.Sprintf("(%s) AND %s", base, timeClause)
}

func (a *App) buildNewerFilter() string {
	base := a.buildEffectiveFilter("")
	if len(a.state.LogListState.Logs) == 0 {
		return base
	}
	newest := a.newestLoadedTimestamp()
	timeClause := fmt.Sprintf("timestamp>%q", newest.Format(time.RFC3339))
	if base == "" {
		return timeClause
	}
	return fmt.Sprintf("(%s) AND %s", base, timeClause)
}

func (a *App) oldestLoadedTimestamp() time.Time {
	if len(a.state.LogListState.Logs) == 0 {
		return time.Time{}
	}
	oldest := a.state.LogListState.Logs[0].Timestamp
	for _, entry := range a.state.LogListState.Logs[1:] {
		if entry.Timestamp.Before(oldest) {
			oldest = entry.Timestamp
		}
	}
	return oldest
}

func (a *App) newestLoadedTimestamp() time.Time {
	if len(a.state.LogListState.Logs) == 0 {
		return time.Time{}
	}
	newest := a.state.LogListState.Logs[0].Timestamp
	for _, entry := range a.state.LogListState.Logs[1:] {
		if entry.Timestamp.After(newest) {
			newest = entry.Timestamp
		}
	}
	return newest
}

func (a *App) getTimeRangeLabel() string {
	if a.state.FilterState.TimeRange.Preset != "" {
		return a.state.FilterState.TimeRange.Preset
	}
	if !a.state.FilterState.TimeRange.Start.IsZero() || !a.state.FilterState.TimeRange.End.IsZero() {
		return "custom"
	}
	return "none"
}

func (a *App) getSeveritySummary() string {
	sf := a.state.FilterState.Severity
	if sf.Mode == "range" && sf.MinLevel != "" {
		return ">=" + sf.MinLevel
	}
	if sf.Mode == "individual" && len(sf.Levels) > 0 {
		if len(sf.Levels) == len(models.SeverityLevels) {
			return "all"
		}
		if len(sf.Levels) <= 2 {
			return strings.Join(sf.Levels, ",")
		}
		return fmt.Sprintf("%d selected", len(sf.Levels))
	}
	return "all"
}

func (a *App) styleSeverityRow(severity, line string) string {
	switch severity {
	case models.SeverityError, models.SeverityCritical, models.SeverityAlert, models.SeverityEmergency:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorGCPError)).Render(line)
	case models.SeverityWarning:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorGCPWarn)).Render(line)
	case models.SeverityInfo, models.SeverityNotice:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorGCPBlueLight)).Render(line)
	case models.SeverityDebug, models.SeverityDefault:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralSubtle)).Render(line)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorNeutralText)).Render(line)
	}
}

func (a *App) styleSeverityBadge(severity string) string {
	switch severity {
	case models.SeverityError, models.SeverityCritical, models.SeverityAlert, models.SeverityEmergency:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBadgeTextLite)).Background(lipgloss.Color(colorGCPError)).Padding(0, 1).Render("ERR")
	case models.SeverityWarning:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBadgeTextDark)).Background(lipgloss.Color(colorGCPWarn)).Padding(0, 1).Render("WRN")
	case models.SeverityInfo, models.SeverityNotice:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBadgeTextLite)).Background(lipgloss.Color(colorGCPBlue)).Padding(0, 1).Render("INF")
	case models.SeverityDebug, models.SeverityDefault:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBadgeTextDark)).Background(lipgloss.Color("248")).Padding(0, 1).Render("DBG")
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBadgeTextDark)).Background(lipgloss.Color("250")).Padding(0, 1).Render("LOG")
	}
}

func (a *App) styleSelectedRow(line string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorSelectionFG)).
		Background(lipgloss.Color(colorSelectionBG)).
		Render(line)
}

func (a *App) currentSelectedIndex() int {
	if len(a.state.LogListState.Logs) == 0 {
		return 0
	}
	idx := a.panes.LogList.scrollOffset
	if idx < 0 {
		return 0
	}
	if idx >= len(a.state.LogListState.Logs) {
		return len(a.state.LogListState.Logs) - 1
	}
	return idx
}

func (a *App) maxWindowStart() int {
	rows := a.logViewportRows()
	return maxInt(0, len(a.state.LogListState.Logs)-rows)
}

func (a *App) currentWindowStart() int {
	start := a.panes.LogList.scrollOffset
	if start < 0 {
		start = 0
	}
	maxStart := a.maxWindowStart()
	if start > maxStart {
		start = maxStart
	}
	return start
}

func (a *App) logViewportRows() int {
	// Mirror the same layout math used in View()/renderLogsPanel.
	queryText := a.state.CurrentQuery.Filter
	if a.activeModalName == "query" {
		queryText = a.queryModal.GetInputWithCursor()
	}
	header := a.renderQueryPanel(queryText, a.activeModalName == "query")
	timeline := a.renderGraphPanel()
	topBar := a.renderTopBar()
	detailsLines := 0
	if a.activeModalName == "details" {
		detailsLines = strings.Count(a.renderDetailsPanel(), "\n")
	}
	footerLines := 3
	if a.lastErr != "" {
		footerLines = 4
	}
	overhead := strings.Count(topBar, "\n") + strings.Count(header, "\n") + strings.Count(timeline, "\n") + detailsLines + footerLines
	screenHeight := maxInt(1, a.height-1)
	logsHeight := screenHeight - overhead - 2
	if logsHeight < 6 {
		logsHeight = 6
	}
	return maxInt(1, logsHeight-1)
}

func (a *App) getSelectedLog() *models.LogEntry {
	if len(a.state.LogListState.Logs) == 0 {
		return nil
	}
	idx := a.currentSelectedIndex()
	if idx < 0 || idx >= len(a.state.LogListState.Logs) {
		return nil
	}
	return &a.state.LogListState.Logs[idx]
}

func (a *App) resetDetailPopupState() {
	a.detailScroll = 0
	a.detailCursor = 0
	a.detailTreeExpanded = map[string]bool{"$": true}
	entry := a.getSelectedLog()
	if entry == nil {
		a.detailViewMode = "full"
		return
	}
	a.detailViewMode = "json-tree"
	if a.entryHasAnyPayload(*entry) {
		return
	}
}

func (a *App) moveDetailPopupDown() {
	if a.detailViewMode != "json-tree" {
		a.detailScroll++
		return
	}
	lines := a.currentJSONTreeLines()
	if len(lines) == 0 {
		return
	}
	if a.detailCursor < len(lines)-1 {
		a.detailCursor++
	}
	a.ensureDetailCursorVisible()
}

func (a *App) moveDetailPopupUp() {
	if a.detailViewMode != "json-tree" {
		if a.detailScroll > 0 {
			a.detailScroll--
		}
		return
	}
	if a.detailCursor > 0 {
		a.detailCursor--
	}
	a.ensureDetailCursorVisible()
}

func (a *App) ensureDetailCursorVisible() {
	visibleHeight := maxInt(8, a.height-8)
	if a.detailCursor < a.detailScroll {
		a.detailScroll = a.detailCursor
		return
	}
	if a.detailCursor >= a.detailScroll+visibleHeight {
		a.detailScroll = a.detailCursor - visibleHeight + 1
	}
}

func (a *App) cycleDetailViewMode() {
	entry := a.getSelectedLog()
	if entry == nil {
		return
	}
	modes := a.availableDetailModes(*entry)
	if len(modes) == 0 {
		return
	}
	current := 0
	for i, mode := range modes {
		if mode == a.detailViewMode {
			current = i
			break
		}
	}
	a.detailViewMode = modes[(current+1)%len(modes)]
	a.detailScroll = 0
	a.detailCursor = 0
}

func (a *App) availableDetailModes(entry models.LogEntry) []string {
	modes := []string{"json-tree", "full"}
	if a.entryHasAnyPayload(entry) {
		modes = append(modes, "payload-raw")
	}
	return modes
}

func (a *App) detailPopupLines(entry models.LogEntry, popupWidth int) ([]string, int) {
	contentWidth := maxInt(30, popupWidth-8)
	switch a.detailViewMode {
	case "json-tree":
		treeLines := a.currentJSONTreeLines()
		if len(treeLines) == 0 {
			return []string{"No structured data available"}, -1
		}
		out := make([]string, 0, len(treeLines))
		for _, line := range treeLines {
			out = append(out, line.text)
		}
		if a.detailCursor < 0 {
			a.detailCursor = 0
		}
		if a.detailCursor >= len(out) {
			a.detailCursor = len(out) - 1
		}
		a.ensureDetailCursorVisible()
		return out, a.detailCursor
	case "payload-raw":
		payloadText, ok := a.getPayloadDisplayText(entry)
		if !ok {
			return []string{"No payload available"}, -1
		}
		return wrapTextByWidth(strings.Split(payloadText, "\n"), contentWidth), -1
	default:
		raw := a.formatter.FormatLogDetails(entry)
		return wrapTextByWidth(strings.Split(raw, "\n"), contentWidth), -1
	}
}

func (a *App) currentJSONTreeLines() []jsonTreeLine {
	entry := a.getSelectedLog()
	if entry == nil {
		return nil
	}
	return buildJSONTreeLines(a.detailEntryJSONObject(*entry), a.detailTreeExpanded)
}

func (a *App) detailEntryJSONObject(entry models.LogEntry) map[string]interface{} {
	root := map[string]interface{}{
		"timestamp": a.displayTime(entry.Timestamp).Format(time.RFC3339Nano),
		"severity":  entry.Severity,
		"message":   entry.Message,
	}
	if len(entry.Labels) > 0 {
		root["labels"] = entry.Labels
	}
	if entry.Resource.Type != "" || len(entry.Resource.Labels) > 0 {
		root["resource"] = map[string]interface{}{
			"type":   entry.Resource.Type,
			"labels": entry.Resource.Labels,
		}
	}
	if entry.SourceLocation != nil {
		root["sourceLocation"] = map[string]interface{}{
			"file":     entry.SourceLocation.File,
			"line":     entry.SourceLocation.Line,
			"function": entry.SourceLocation.Function,
		}
	}
	if entry.Trace != "" {
		root["trace"] = entry.Trace
	}
	if entry.SpanID != "" {
		root["spanId"] = entry.SpanID
	}
	if entry.JSONPayload != nil {
		root["payload"] = entry.JSONPayload
	} else if parsed, ok := parseStructuredJSONPayload(entry.TextPayload); ok {
		root["payload"] = parsed
	} else if strings.TrimSpace(entry.TextPayload) != "" {
		root["textPayload"] = entry.TextPayload
	}
	return root
}

func (a *App) selectedDetailNodeInfo() (string, string) {
	if a.detailViewMode != "json-tree" {
		return "", ""
	}
	lines := a.currentJSONTreeLines()
	if len(lines) == 0 || a.detailCursor < 0 || a.detailCursor >= len(lines) {
		return "", ""
	}
	line := lines[a.detailCursor]
	return line.path, jsonTypeLabel(line.value)
}

func (a *App) collapseAllDetailTreeNodes() {
	if a.detailViewMode != "json-tree" {
		return
	}
	a.detailTreeExpanded = map[string]bool{"$": true}
	a.detailCursor = 0
	a.detailScroll = 0
}

func (a *App) expandAllDetailTreeNodes() {
	if a.detailViewMode != "json-tree" {
		return
	}
	lines := a.currentJSONTreeLines()
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		if line.canExpand {
			a.detailTreeExpanded[line.path] = true
		}
	}
}

func (a *App) expandDetailTreeNode() {
	if a.detailViewMode != "json-tree" {
		return
	}
	lines := a.currentJSONTreeLines()
	if len(lines) == 0 || a.detailCursor < 0 || a.detailCursor >= len(lines) {
		return
	}
	line := lines[a.detailCursor]
	if line.canExpand {
		a.detailTreeExpanded[line.path] = true
	}
}

func (a *App) collapseDetailTreeNode() {
	if a.detailViewMode != "json-tree" {
		return
	}
	lines := a.currentJSONTreeLines()
	if len(lines) == 0 || a.detailCursor < 0 || a.detailCursor >= len(lines) {
		return
	}
	line := lines[a.detailCursor]
	if line.canExpand && line.expanded {
		a.detailTreeExpanded[line.path] = false
		return
	}
	parent := parentJSONPath(line.path)
	if parent == "" {
		return
	}
	for i := range lines {
		if lines[i].path == parent {
			a.detailCursor = i
			a.ensureDetailCursorVisible()
			return
		}
	}
}

func parentJSONPath(path string) string {
	if path == "$" {
		return ""
	}
	if strings.HasSuffix(path, "]") {
		if idx := strings.LastIndex(path, "["); idx > 0 {
			return path[:idx]
		}
	}
	if idx := strings.LastIndex(path, "."); idx > 0 {
		return path[:idx]
	}
	return "$"
}

func (a *App) copySelectedDetailNode() {
	entry := a.getSelectedLog()
	if entry == nil {
		a.lastErr = "No log selected"
		return
	}
	if a.detailViewMode == "json-tree" {
		lines := a.currentJSONTreeLines()
		if len(lines) == 0 || a.detailCursor < 0 || a.detailCursor >= len(lines) {
			a.lastErr = "No payload node selected"
			return
		}
		text := formatJSONValueForCopy(lines[a.detailCursor].value)
		if err := copyTextToClipboard(text); err != nil {
			a.lastErr = "Copy failed: " + err.Error()
			return
		}
		a.lastErr = "Copied selected payload node"
		return
	}
	a.copyDetailPayload()
}

func (a *App) copyDetailPayload() {
	entry := a.getSelectedLog()
	if entry == nil {
		a.lastErr = "No log selected"
		return
	}
	payloadText, ok := a.getPayloadDisplayText(*entry)
	if !ok {
		a.lastErr = "No payload available"
		return
	}
	if err := copyTextToClipboard(payloadText); err != nil {
		a.lastErr = "Copy failed: " + err.Error()
		return
	}
	a.lastErr = "Copied payload"
}

func (a *App) entryHasAnyPayload(entry models.LogEntry) bool {
	if entry.JSONPayload != nil || strings.TrimSpace(entry.TextPayload) != "" {
		return true
	}
	_, ok := parseStructuredJSONPayload(entry.Message)
	return ok
}

func (a *App) entryHasStructuredPayload(entry models.LogEntry) bool {
	_, ok := a.getStructuredPayload(entry)
	return ok
}

func (a *App) getStructuredPayload(entry models.LogEntry) (interface{}, bool) {
	if entry.JSONPayload != nil {
		return entry.JSONPayload, true
	}
	if payload, ok := parseStructuredJSONPayload(entry.TextPayload); ok {
		return payload, true
	}
	if payload, ok := parseStructuredJSONPayload(entry.Message); ok {
		return payload, true
	}
	return nil, false
}

func (a *App) getPayloadDisplayText(entry models.LogEntry) (string, bool) {
	if entry.JSONPayload != nil {
		data, err := json.MarshalIndent(entry.JSONPayload, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", entry.JSONPayload), true
		}
		return string(data), true
	}
	textPayload := strings.TrimSpace(entry.TextPayload)
	if parsed, ok := parseStructuredJSONPayload(textPayload); ok {
		data, err := json.MarshalIndent(parsed, "", "  ")
		if err == nil {
			return string(data), true
		}
	}
	if textPayload != "" {
		return entry.TextPayload, true
	}
	if parsed, ok := parseStructuredJSONPayload(entry.Message); ok {
		data, err := json.MarshalIndent(parsed, "", "  ")
		if err == nil {
			return string(data), true
		}
	}
	return "", false
}

func parseStructuredJSONPayload(raw string) (interface{}, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	candidates := []string{raw}
	candidates = append(candidates, extractJSONCandidates(raw)...)
	for _, candidate := range candidates {
		var parsed interface{}
		if err := json.Unmarshal([]byte(candidate), &parsed); err == nil && isStructuredJSONValue(parsed) {
			return parsed, true
		}
		normalized := normalizeLenientJSON(candidate)
		if normalized == candidate {
			continue
		}
		if err := json.Unmarshal([]byte(normalized), &parsed); err == nil && isStructuredJSONValue(parsed) {
			return parsed, true
		}
	}
	return nil, false
}

func extractJSONCandidates(raw string) []string {
	out := make([]string, 0, 2)
	if start := strings.Index(raw, "{"); start >= 0 {
		if end := strings.LastIndex(raw, "}"); end > start {
			out = append(out, raw[start:end+1])
		}
	}
	if start := strings.Index(raw, "["); start >= 0 {
		if end := strings.LastIndex(raw, "]"); end > start {
			out = append(out, raw[start:end+1])
		}
	}
	return out
}

func normalizeLenientJSON(raw string) string {
	normalized := strings.TrimSpace(raw)
	if strings.Contains(normalized, "'") {
		normalized = strings.ReplaceAll(normalized, "'", "\"")
	}
	normalized = pythonBoolRegex.ReplaceAllStringFunc(normalized, func(in string) string {
		switch in {
		case "True":
			return "true"
		case "False":
			return "false"
		case "None":
			return "null"
		default:
			return in
		}
	})
	return normalized
}

func isStructuredJSONValue(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return true
	default:
		return false
	}
}

func copyTextToClipboard(text string) error {
	candidates := [][]string{
		{"pbcopy"},
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"clip"},
	}
	var lastErr error
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no clipboard utility found (pbcopy, wl-copy, xclip, xsel)")
}

func (a *App) openSelectedLogInEditorCmd() tea.Cmd {
	entry := a.getSelectedLog()
	if entry == nil {
		return func() tea.Msg { return editorResultMsg{err: fmt.Errorf("no log selected"), target: "entry"} }
	}

	details := a.formatter.FormatLogDetails(*entry)
	tmpFile, err := os.CreateTemp("", "log-explorer-entry-*.log")
	if err != nil {
		return func() tea.Msg { return editorResultMsg{err: err, target: "entry"} }
	}
	if _, err := tmpFile.WriteString(details); err != nil {
		_ = tmpFile.Close()
		return func() tea.Msg { return editorResultMsg{err: err, target: "entry"} }
	}
	_ = tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorResultMsg{err: err, target: "entry"}
	})
}

func (a *App) openSelectedPayloadInEditorCmd() tea.Cmd {
	entry := a.getSelectedLog()
	if entry == nil {
		return func() tea.Msg { return editorResultMsg{err: fmt.Errorf("no log selected"), target: "payload"} }
	}
	payloadText, ok := a.getPayloadDisplayText(*entry)
	if !ok {
		return func() tea.Msg { return editorResultMsg{err: fmt.Errorf("no payload available"), target: "payload"} }
	}
	ext := ".txt"
	if a.entryHasStructuredPayload(*entry) {
		ext = ".json"
	}
	tmpFile, err := os.CreateTemp("", "log-explorer-payload-*"+ext)
	if err != nil {
		return func() tea.Msg { return editorResultMsg{err: err, target: "payload"} }
	}
	if _, err := tmpFile.WriteString(payloadText); err != nil {
		_ = tmpFile.Close()
		return func() tea.Msg { return editorResultMsg{err: err, target: "payload"} }
	}
	_ = tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorResultMsg{err: err, target: "payload"}
	})
}

func (a *App) selectProject() {
	if len(a.availableProjects) == 0 {
		return
	}
	project := a.availableProjects[a.projectCursor]
	a.state.CurrentProject = project
	a.activeModalName = "none"
}

func (a *App) openLogListInEditorCmd() tea.Cmd {
	if len(a.state.LogListState.Logs) == 0 {
		return func() tea.Msg { return editorResultMsg{err: fmt.Errorf("no logs loaded"), target: "list"} }
	}

	tmpFile, err := os.CreateTemp("", "log-explorer-list-*.json")
	if err != nil {
		return func() tea.Msg { return editorResultMsg{err: err, target: "list"} }
	}

	exportLogs := make([]map[string]interface{}, 0, len(a.state.LogListState.Logs))
	for i, log := range a.state.LogListState.Logs {
		exportLogs = append(exportLogs, map[string]interface{}{
			"index":       i + 1,
			"timestamp":   log.Timestamp.Format(time.RFC3339Nano),
			"severity":    log.Severity,
			"message":     log.Message,
			"labels":      log.Labels,
			"resource":    log.Resource,
			"trace":       log.Trace,
			"span_id":     log.SpanID,
			"jsonPayload": log.JSONPayload,
			"textPayload": log.TextPayload,
		})
	}
	payload := map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"total_logs":   len(a.state.LogListState.Logs),
		"logs":         exportLogs,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tmpFile.Close()
		return func() tea.Msg { return editorResultMsg{err: err, target: "list"} }
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return func() tea.Msg { return editorResultMsg{err: err, target: "list"} }
	}
	_ = tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorResultMsg{err: err, target: "list"}
	})
}

func (a *App) openLogListCSVInEditorCmd() tea.Cmd {
	if len(a.state.LogListState.Logs) == 0 {
		return func() tea.Msg { return editorResultMsg{err: fmt.Errorf("no logs loaded"), target: "list-csv"} }
	}

	tmpFile, err := os.CreateTemp("", "log-explorer-list-*.csv")
	if err != nil {
		return func() tea.Msg { return editorResultMsg{err: err, target: "list-csv"} }
	}
	_ = tmpFile.Close()

	if err := a.exporter.ExportToCSV(a.state.LogListState.Logs, tmpFile.Name()); err != nil {
		return func() tea.Msg { return editorResultMsg{err: err, target: "list-csv"} }
	}

	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorResultMsg{err: err, target: "list-csv"}
	})
}

func mergeUniqueLogs(existing []models.LogEntry, incoming []models.LogEntry, prepend bool) []models.LogEntry {
	if len(incoming) == 0 {
		return existing
	}

	seen := make(map[string]bool, len(existing)+len(incoming))
	out := make([]models.LogEntry, 0, len(existing)+len(incoming))
	if prepend {
		for _, e := range incoming {
			k := logEntryKey(e)
			if !seen[k] {
				seen[k] = true
				out = append(out, e)
			}
		}
		for _, e := range existing {
			k := logEntryKey(e)
			if !seen[k] {
				seen[k] = true
				out = append(out, e)
			}
		}
		return out
	}

	for _, e := range existing {
		k := logEntryKey(e)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	for _, e := range incoming {
		k := logEntryKey(e)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	return out
}

func logEntryKey(e models.LogEntry) string {
	return e.Timestamp.Format(time.RFC3339Nano) + "|" + e.Severity + "|" + e.Message
}

func mergeUniqueStrings(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing)+len(incoming))
	out := make([]string, 0, len(existing)+len(incoming))
	for _, item := range existing {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	for _, item := range incoming {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func wrapMultiline(input string, width int, maxLines int) []string {
	if width < 8 {
		width = 8
	}
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	rawLines := strings.Split(input, "\n")
	out := make([]string, 0, len(rawLines))
	for _, raw := range rawLines {
		if raw == "" {
			out = append(out, "")
			continue
		}
		line := raw
		for len(line) > width {
			out = append(out, line[:width])
			line = line[width:]
		}
		out = append(out, line)
	}
	if len(out) > maxLines {
		out = out[:maxLines]
		last := out[maxLines-1]
		if len(last) > width-3 {
			last = last[:width-3]
		}
		out[maxLines-1] = last + "..."
	}
	return out
}

func wrapTextByWidth(lines []string, width int) []string {
	if width < 8 {
		width = 8
	}
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		if raw == "" {
			out = append(out, "")
			continue
		}
		line := raw
		for len(line) > width {
			out = append(out, line[:width])
			line = line[width:]
		}
		out = append(out, line)
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *App) fitToViewport(output string) string {
	if a.height <= 0 {
		return output
	}
	viewHeight := maxInt(1, a.height-1)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) <= viewHeight {
		return output
	}
	return strings.Join(lines[:viewHeight], "\n")
}
