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
)

// App represents the main TUI application
type App struct {
	state              *models.AppState
	width              int
	height             int
	panes              *Panes
	lastErr            string
	helpModal          *HelpModal
	queryModal         *QueryModal
	timePicker         *TimePicker
	severityFilter     *SeverityFilterPanel
	exporter           *Exporter
	formatter          *LogFormatter
	timelineBuilder    *TimelineBuilder
	statusBar          string
	queryExec          func(string) ([]models.LogEntry, error)
	activeModalName    string // Track which modal is open
	previousModalName  string
	vimMode            bool
	loadingOlder       bool
	loadingNewer       bool
	detailScroll       int
	detailCursor       int
	detailViewMode     string
	detailTreeExpanded map[string]bool
	severityCursor     int
	autoLoadAll        bool
	projectPopup       bool
	projectCursor      int
	availableProjects  []string
	queryHistory       []string
	queryHistoryCursor int
	startupFilter      string
	projectListFn      func() ([]string, error)
	loadingProjects    bool
	queryLibrary       []config.SavedQueryRecord
	queryLibraryCursor int
	queryCache         map[string]config.CachedQueryRecord
	queryCacheTTL      time.Duration
	queryCacheMax      int
	bypassNextCache    bool
	persistHistoryFn   func(filter, project string) error
	persistLibraryFn   func([]config.SavedQueryRecord) error
	persistCacheFn     func([]config.CachedQueryRecord) error
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
		state:              appState,
		width:              120,
		height:             40,
		panes:              panes,
		lastErr:            "",
		helpModal:          helpModal,
		queryModal:         queryModal,
		timePicker:         timePicker,
		severityFilter:     severityFilter,
		exporter:           exporter,
		formatter:          formatter,
		timelineBuilder:    timelineBuilder,
		statusBar:          helpModal.GetShortHelp(),
		queryExec:          nil,
		activeModalName:    "none",
		previousModalName:  "none",
		vimMode:            true,
		loadingOlder:       false,
		loadingNewer:       false,
		detailScroll:       0,
		detailCursor:       0,
		detailViewMode:     "full",
		detailTreeExpanded: map[string]bool{"$": true},
		severityCursor:     0,
		autoLoadAll:        false,
		projectPopup:       false,
		projectCursor:      0,
		availableProjects:  availableProjects,
		queryHistory:       []string{},
		queryHistoryCursor: -1,
		startupFilter:      "",
		projectListFn:      nil,
		loadingProjects:    false,
		queryLibrary:       []config.SavedQueryRecord{},
		queryLibraryCursor: 0,
		queryCache:         map[string]config.CachedQueryRecord{},
		queryCacheTTL:      15 * time.Minute,
		queryCacheMax:      40,
		bypassNextCache:    false,
		persistHistoryFn:   nil,
		persistLibraryFn:   nil,
		persistCacheFn:     nil,
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
			switch msg.mode {
			case "append":
				before := len(a.state.LogListState.Logs)
				a.state.LogListState.Logs = mergeUniqueLogs(a.state.LogListState.Logs, msg.logs, false)
				if msg.preserveAnchor {
					a.panes.LogList.scrollOffset = msg.anchorOffset
				}
				a.lastErr = fmt.Sprintf("Loaded older logs: +%d", len(a.state.LogListState.Logs)-before)
			case "prepend":
				before := len(a.state.LogListState.Logs)
				a.state.LogListState.Logs = mergeUniqueLogs(a.state.LogListState.Logs, msg.logs, true)
				// Keep viewport near current context when prepending.
				a.panes.LogList.scrollOffset += len(a.state.LogListState.Logs) - before
				a.lastErr = fmt.Sprintf("Loaded newer logs: +%d", len(a.state.LogListState.Logs)-before)
			default:
				a.state.LogListState.Logs = msg.logs
				a.panes.LogList.scrollOffset = 0
				if msg.fromCache {
					a.lastErr = fmt.Sprintf("Query cache hit: %d logs", len(msg.logs))
				} else {
					a.lastErr = fmt.Sprintf("Query complete: %d logs", len(msg.logs))
					a.storeQueryResultCache(msg.filter, msg.logs)
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
	footerLines := 2
	if a.lastErr != "" {
		footerLines = 3
	}
	topBarLines := strings.Count(topBar, "\n")
	timelineLines := strings.Count(timeline, "\n")
	overhead := topBarLines + headerLines + footerLines + timelineLines + detailsLines

	logsHeight := a.height - overhead - 2
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
		output = output + "\n" + a.renderTimePickerModal()
	case "severity":
		output = output + "\n" + a.renderSeverityFilterModal()
	case "export":
		output = output + "\n" + a.renderExportModal()
	case "detailPopup":
		output = output + "\n" + a.renderDetailPopup()
	case "help":
		output = a.helpModal.Render(a.width, a.height)
	case "projectPopup":
		output = a.renderCenteredPopup(output, a.renderProjectDropdown())
	case "queryLibrary":
		output = a.renderCenteredPopup(output, a.renderQueryLibraryPopup())
	}

	return output
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
	case "help":
		// Help modal just dismisses on any key
		if msg.String() == "esc" || msg.String() == "?" {
			a.activeModalName = "none"
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
		a.panes.LogList.JumpToBottom()
		a.panes.LogList.scrollOffset = a.maxWindowStart()
		return a, nil
	case "g":
		if a.vimMode {
			a.panes.LogList.JumpToTop()
		}
		return a, nil
	case "G":
		if a.vimMode {
			a.panes.LogList.JumpToBottom()
			a.panes.LogList.scrollOffset = a.maxWindowStart()
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
		a.vimMode = !a.vimMode
		if a.vimMode {
			a.lastErr = "Key mode: vim"
		} else {
			a.lastErr = "Key mode: standard"
		}
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
		a.cycleQueryHistory(-1)
		return a, nil
	case "ctrl+g":
		a.cycleQueryHistory(1)
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
	anchor := a.currentWindowStart()
	a.panes.LogList.ScrollDown()
	if a.queryExec != nil && !a.loadingOlder && len(a.state.LogListState.Logs) > 0 && wasAtBottom {
		a.loadingOlder = true
		a.state.LogListState.IsLoading = true
		return a, a.runQueryCmdWithAnchor(a.buildOlderFilter(), "append", true, anchor)
	}
	return a, nil
}

func (a *App) handleScrollUp() (tea.Model, tea.Cmd) {
	prev := a.panes.LogList.scrollOffset
	a.panes.LogList.ScrollUp()
	if a.queryExec != nil && !a.loadingNewer && prev == 0 && len(a.state.LogListState.Logs) > 0 {
		a.loadingNewer = true
		a.state.LogListState.IsLoading = true
		return a, a.runQueryCmd(a.buildNewerFilter(), "prepend")
	}
	return a, nil
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

	var sb strings.Builder
	sb.WriteString("┏━━ TIME RANGE " + strings.Repeat("━", a.width-16) + "\n")

	for i, preset := range presets {
		prefix := "  "
		if i == selectedIdx {
			prefix = "▶ "
		}
		sb.WriteString(fmt.Sprintf("┃ %s%-20s\n", prefix, preset.Name))
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
		sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
		sb.WriteString("┃ Custom Range (UTC):\n")
		sb.WriteString(fmt.Sprintf("┃ %s Start: %s\n", startPrefix, startInput))
		sb.WriteString(fmt.Sprintf("┃ %s End:   %s\n", endPrefix, endInput))
	}

	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	if a.timePicker.IsCustomSelected() {
		sb.WriteString("┃ Type exact datetime (YYYY-MM-DD HH:MM:SS or RFC3339)\n")
		sb.WriteString("┃ h/l/Tab switch field | c clear field | Backspace edit | Ctrl+u clear | j/k +/-15m | J/K +/-1h | Enter apply | Esc cancel\n")
	} else {
		sb.WriteString("┃ Press j/k to navigate, Enter to select, Esc to cancel\n")
	}

	if errMsg := a.timePicker.GetError(); errMsg != "" {
		if len(errMsg) > a.width-6 {
			errMsg = errMsg[:a.width-9] + "..."
		}
		sb.WriteString(fmt.Sprintf("┃ Error: %s\n", errMsg))
	}

	return sb.String()
}

// renderSeverityFilterModal renders the severity filter modal
func (a *App) renderSeverityFilterModal() string {
	levels := a.severityFilter.GetSeverityLevels()
	mode := a.severityFilter.GetMode()

	var sb strings.Builder
	sb.WriteString("┏━━ SEVERITY FILTER " + strings.Repeat("━", a.width-21) + "\n")
	sb.WriteString(fmt.Sprintf("┃ Mode: %s (press 'm' to toggle)\n", mode))
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")

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
			sb.WriteString(fmt.Sprintf("┃ %s\n", levelText))
		}
		sb.WriteString(fmt.Sprintf("┃ Selected: %d\n", a.severityFilter.CountSelectedLevels()))
	} else {
		minLevel := a.severityFilter.GetMinimumLevel()
		sb.WriteString(fmt.Sprintf("┃ Minimum level: %s\n", minLevel))
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
			sb.WriteString(fmt.Sprintf("┃ %s\n", levelText))
		}
		sb.WriteString("┃ (Shows logs at this level and above)\n")
	}

	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	sb.WriteString("┃ j/k: move | Space: toggle/set | m: mode | a: all | d: none | Enter: apply | Esc: cancel\n")

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
		first := a.state.LogListState.Logs[0].Timestamp
		last := a.state.LogListState.Logs[len(a.state.LogListState.Logs)-1].Timestamp
		// Keep range in chronological order for readability.
		if first.Before(last) {
			first, last = last, first
		}
		rangeText = fmt.Sprintf("Range: %s -> %s", last.Format("2006-01-02 15:04:05"), first.Format("2006-01-02 15:04:05"))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("┣%s\n", strings.Repeat("─", a.width-1)))
	sb.WriteString(fmt.Sprintf("┃ %s %s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Render("Timeline"), spark))
	sb.WriteString(fmt.Sprintf("┃ %s Critical:%d  Warning:%d  Info/Other:%d\n",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")).Render("Severity Mix"),
		crit, warn, info))
	sb.WriteString(fmt.Sprintf("┃ %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(rangeText)))
	return sb.String()
}

func (a *App) renderLogsPanel(height int) (string, int, int) {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")).Render("LOG STREAM")
	sb.WriteString(fmt.Sprintf("┏ %s (%d) %s\n", title, len(a.state.LogListState.Logs), strings.Repeat("━", maxInt(0, a.width-18))))
	sb.WriteString(fmt.Sprintf("┃ %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("IDX   TIMESTAMP           SEV      MESSAGE")))

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
		timePart := log.Timestamp.Format("2006-01-02 15:04:05")
		sevBadge := a.styleSeverityBadge(log.Severity)
		msgMax := maxInt(12, a.width-47)
		msg := log.Message
		if len(msg) > msgMax {
			msg = msg[:msgMax-3] + "..."
		}

		row := fmt.Sprintf("%-4d  %s  %s  %s", i+1, timePart, sevBadge, msg)
		if i == a.currentSelectedIndex() {
			row = a.styleSelectedRow(row)
		} else if i%2 == 0 {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(row)
		} else {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(row)
		}
		sb.WriteString("┃ " + row + "\n")
	}

	for i := end - start; i < height-1; i++ {
		sb.WriteString("┃ \n")
	}
	return sb.String(), start + 1, end
}

func (a *App) renderQueryPanel(query string, editing bool) string {
	var sb strings.Builder
	title := "QUERY EDITOR"
	if editing {
		title = "QUERY EDITOR [EDITING]"
	}
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("63")).Padding(0, 1).Render(title)
	sb.WriteString(fmt.Sprintf("┏ %s %s\n", titleStyled, strings.Repeat("━", maxInt(0, a.width-22))))

	if strings.TrimSpace(query) == "" {
		query = "No filter. Press q to edit."
	}
	maxQueryLines := minInt(16, maxInt(6, a.height/3))
	lines := wrapMultiline(query, maxInt(30, a.width-6), maxQueryLines)
	for _, line := range lines {
		if editing && strings.Contains(line, "│") {
			sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Render(line) + "\n")
		} else {
			sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Render(line) + "\n")
		}
	}
	hint := "Enter run | Ctrl+A select all | Ctrl+/ comment | Ctrl+S save | Ctrl+Y library"
	sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(hint) + "\n")
	if len(a.queryHistory) > 0 {
		sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render("Recent queries:") + "\n")
		for i := 0; i < minInt(3, len(a.queryHistory)); i++ {
			q := strings.ReplaceAll(a.queryHistory[i], "\n", " ↩ ")
			q = strings.TrimSpace(q)
			if len(q) > a.width-12 {
				q = q[:a.width-15] + "..."
			}
			sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(fmt.Sprintf("  %d) %s", i+1, q)) + "\n")
		}
	}
	return sb.String()
}

func (a *App) renderStatusPanel(windowStart, windowEnd int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("┣%s\n", strings.Repeat("━", a.width-1)))
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
	sb.WriteString(fmt.Sprintf("┃ %d-%d/%d  %s  sev:%s  load:%s  stream:%s  keys:%s  cache:%d  ?\n",
		windowStart, windowEnd, total, a.getTimeRangeLabel(), a.getSeveritySummary(), loadMode, streamMode, keyMode, len(a.cachedQueryRecords())))
	if a.lastErr != "" {
		errLine := a.lastErr
		if len(errLine) > a.width-4 {
			errLine = errLine[:a.width-7] + "..."
		}
		sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(errLine) + "\n")
	}
	return sb.String()
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
	left := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("27")).Padding(0, 1).Render("GCP Log Explorer")
	keys := "std"
	if a.vimMode {
		keys = "vim"
	}
	rightText := fmt.Sprintf("project:%s ▾  mode:%s  keys:%s", project, queryMode, keys)
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("31")).Padding(0, 1).Render(rightText)
	fill := maxInt(0, a.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", fill) + right + "\n"
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
	sb.WriteString(fmt.Sprintf("┣%s\n", strings.Repeat("─", a.width-1)))
	sb.WriteString(fmt.Sprintf("┃ %s %d/%d\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")).Render("DETAILS"), idx+1, len(a.state.LogListState.Logs)))
	for _, line := range lines {
		if len(line) > a.width-4 {
			line = line[:a.width-7] + "..."
		}
		sb.WriteString("┃ " + lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(line) + "\n")
	}
	sb.WriteString("┃ Esc/Enter/Ctrl+D: close | Ctrl+P: popup | Ctrl+O: open entry | Ctrl+L: open list\n")
	return sb.String()
}

func (a *App) renderDetailPopup() string {
	entry := a.getSelectedLog()
	if entry == nil {
		return ""
	}

	lines, selectedIndex := a.detailPopupLines(*entry)
	visibleHeight := maxInt(8, a.height-8)
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
	sb.WriteString("┏━━ FULL LOG POPUP " + strings.Repeat("━", maxInt(0, a.width-19)) + "\n")
	sb.WriteString(fmt.Sprintf("┃ Entry %d/%d  Mode:%s  Scroll %d/%d\n", a.currentSelectedIndex()+1, len(a.state.LogListState.Logs), a.detailViewMode, start+1, maxInt(1, len(lines))))
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	for i := start; i < end; i++ {
		line := lines[i]
		prefix := "  "
		if i == selectedIndex {
			prefix = "▸ "
		}
		if len(line) > a.width-4 {
			line = line[:a.width-7] + "..."
		}
		sb.WriteString("┃ " + prefix + line + "\n")
	}
	for i := end; i < start+visibleHeight; i++ {
		sb.WriteString("┃ \n")
	}
	sb.WriteString("┣" + strings.Repeat("━", a.width-1) + "\n")
	if selectedPath, selectedType := a.selectedDetailNodeInfo(); selectedPath != "" {
		meta := fmt.Sprintf("selected:%s (%s)", selectedPath, selectedType)
		if len(meta) > a.width-4 {
			meta = meta[:a.width-7] + "..."
		}
		sb.WriteString("┃ " + meta + "\n")
	}
	sb.WriteString("┃ j/k:move  h/l:collapse/expand  z/Z:collapse/expand all  v/tab:mode  y/Y:copy  Ctrl+E:open payload\n")
	sb.WriteString("┃ Ctrl+O:open entry  Ctrl+L:open list(JSON)  Ctrl+Shift+L/Alt+L:open list(CSV)  Esc/Ctrl+P:close\n")
	return sb.String()
}

func (a *App) renderProjectDropdown() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(44, a.width-20), 100)
	title := " PROJECT SELECTOR "
	sb.WriteString("┏" + title + strings.Repeat("━", maxInt(0, popupWidth-len(title)-2)) + "\n")
	if a.loadingProjects {
		sb.WriteString("┃ Discovering projects from gcloud account...\n")
	}
	if len(a.availableProjects) == 0 {
		sb.WriteString("┃ No projects discovered yet\n")
		sb.WriteString("┣" + strings.Repeat("━", popupWidth-1) + "\n")
		sb.WriteString("┃ Enter: switch project | Esc: cancel\n")
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
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(line)
		}
		sb.WriteString("┃ " + line + "\n")
	}
	sb.WriteString("┣" + strings.Repeat("━", popupWidth-1) + "\n")
	sb.WriteString(fmt.Sprintf("┃ Showing %d-%d of %d | j/k move | Enter select | Esc close\n", start+1, end, len(a.availableProjects)))
	return sb.String()
}

func (a *App) renderQueryLibraryPopup() string {
	var sb strings.Builder
	popupWidth := minInt(maxInt(44, a.width-20), 110)
	title := " QUERY LIBRARY "
	sb.WriteString("┏" + title + strings.Repeat("━", maxInt(0, popupWidth-len(title)-2)) + "\n")
	if len(a.queryLibrary) == 0 {
		sb.WriteString("┃ No saved queries yet\n")
		sb.WriteString("┣" + strings.Repeat("━", popupWidth-1) + "\n")
		sb.WriteString("┃ Ctrl+S in query editor to save | Esc close\n")
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
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(line)
		}
		sb.WriteString("┃ " + line + "\n")
	}
	sb.WriteString("┣" + strings.Repeat("━", popupWidth-1) + "\n")
	sb.WriteString(fmt.Sprintf("┃ %d-%d of %d | j/k move | Enter apply | Esc close\n", start+1, end, len(a.queryLibrary)))
	return sb.String()
}

func (a *App) openQueryLibraryModal(previous string) {
	a.previousModalName = previous
	a.activeModalName = "queryLibrary"
	a.queryLibraryCursor = 0
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
	baseLines := strings.Split(strings.TrimRight(base, "\n"), "\n")
	if len(baseLines) < a.height {
		for len(baseLines) < a.height {
			baseLines = append(baseLines, "")
		}
	}
	popupLines := strings.Split(strings.TrimRight(popup, "\n"), "\n")
	if len(popupLines) == 0 {
		return base
	}
	startRow := maxInt(1, (a.height-len(popupLines))/2)
	for i, line := range popupLines {
		row := startRow + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		leftPad := maxInt(0, (a.width-len(line))/2)
		baseLines[row] = strings.Repeat(" ", leftPad) + line
	}
	return strings.Join(baseLines, "\n")
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
	oldest := a.state.LogListState.Logs[len(a.state.LogListState.Logs)-1].Timestamp
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
	newest := a.state.LogListState.Logs[0].Timestamp
	timeClause := fmt.Sprintf("timestamp>%q", newest.Format(time.RFC3339))
	if base == "" {
		return timeClause
	}
	return fmt.Sprintf("(%s) AND %s", base, timeClause)
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
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(line)
	case models.SeverityWarning:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(line)
	case models.SeverityInfo, models.SeverityNotice:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Render(line)
	case models.SeverityDebug, models.SeverityDefault:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(line)
	default:
		return line
	}
}

func (a *App) styleSeverityBadge(severity string) string {
	switch severity {
	case models.SeverityError, models.SeverityCritical, models.SeverityAlert, models.SeverityEmergency:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("160")).Padding(0, 1).Render("ERR")
	case models.SeverityWarning:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("214")).Padding(0, 1).Render("WRN")
	case models.SeverityInfo, models.SeverityNotice:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("81")).Padding(0, 1).Render("INF")
	case models.SeverityDebug, models.SeverityDefault:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("245")).Padding(0, 1).Render("DBG")
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("250")).Padding(0, 1).Render("LOG")
	}
}

func (a *App) styleSelectedRow(line string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("25")).
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
	footerLines := 2
	if a.lastErr != "" {
		footerLines = 3
	}
	overhead := strings.Count(topBar, "\n") + strings.Count(header, "\n") + strings.Count(timeline, "\n") + detailsLines + footerLines
	logsHeight := a.height - overhead - 2
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

func (a *App) detailPopupLines(entry models.LogEntry) ([]string, int) {
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
		return wrapTextByWidth(strings.Split(payloadText, "\n"), maxInt(30, a.width-8)), -1
	default:
		raw := a.formatter.FormatLogDetails(entry)
		return wrapTextByWidth(strings.Split(raw, "\n"), maxInt(30, a.width-8)), -1
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
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
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
	keyFor := func(e models.LogEntry) string {
		return e.Timestamp.Format(time.RFC3339Nano) + "|" + e.Severity + "|" + e.Message
	}

	out := make([]models.LogEntry, 0, len(existing)+len(incoming))
	if prepend {
		for _, e := range incoming {
			k := keyFor(e)
			if !seen[k] {
				seen[k] = true
				out = append(out, e)
			}
		}
		for _, e := range existing {
			k := keyFor(e)
			if !seen[k] {
				seen[k] = true
				out = append(out, e)
			}
		}
		return out
	}

	for _, e := range existing {
		k := keyFor(e)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	for _, e := range incoming {
		k := keyFor(e)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	return out
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
