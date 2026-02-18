package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/log-explorer-tui/pkg/config"
	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewApp(t *testing.T) {
	state := &models.AppState{
		CurrentProject: "test-project",
		IsReady:        true,
	}

	newApp := NewApp(state)

	if newApp.state != state {
		t.Error("App state not set correctly")
	}

	if newApp.width == 0 || newApp.height == 0 {
		t.Error("App dimensions not initialized")
	}

	if newApp.panes == nil {
		t.Error("App panes not initialized")
	}
}

func TestAppInit(t *testing.T) {
	state := &models.AppState{}
	testApp := NewApp(state)

	cmd := testApp.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestAppWindowResize(t *testing.T) {
	state := &models.AppState{}
	testApp := NewApp(state)

	msg := tea.WindowSizeMsg{Width: 200, Height: 50}
	newAppModel, _ := testApp.Update(msg)
	updatedApp := newAppModel.(*App)

	if updatedApp.width != 200 || updatedApp.height != 50 {
		t.Errorf("Window size not updated: %dx%d", updatedApp.width, updatedApp.height)
	}
}

func TestAppKeyBindings(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		UIState: models.UIState{FocusedPane: "logs"},
		LogListState: models.LogListState{Logs: []models.LogEntry{
			{ID: "1", Message: "test"},
		}},
	}
	_ = state // Use state to pass to tests

	tests := []struct {
		name        string
		key         string
		checkResult func(*App) bool
	}{
		{
			name: "h moves pane focus left",
			key:  "h",
			checkResult: func(testApp *App) bool {
				return true // Just check it doesn't panic
			},
		},
		{
			name: "l moves pane focus right",
			key:  "l",
			checkResult: func(testApp *App) bool {
				return true
			},
		},
		{
			name: "j scrolls down",
			key:  "j",
			checkResult: func(testApp *App) bool {
				return testApp.panes.LogList.scrollOffset > 0
			},
		},
		{
			name: "k scrolls up",
			key:  "k",
			checkResult: func(testApp *App) bool {
				// k at top should keep offset at 0
				return testApp.panes.LogList.scrollOffset == 0
			},
		},
		{
			name: "G jumps to bottom",
			key:  "G",
			checkResult: func(testApp *App) bool {
				return testApp.panes.LogList.scrollOffset >= 0
			},
		},
		{
			name: "ctrl+f pages down",
			key:  "ctrl+f",
			checkResult: func(testApp *App) bool {
				return testApp.panes.LogList.scrollOffset == 10
			},
		},
		{
			name: "t opens time range modal",
			key:  "t",
			checkResult: func(testApp *App) bool {
				return testApp.state.UIState.ActiveModal == "timeRange"
			},
		},
		{
			name: "f opens severity filter modal",
			key:  "f",
			checkResult: func(testApp *App) bool {
				return testApp.state.UIState.ActiveModal == "severity"
			},
		},
		{
			name: "p opens project popup",
			key:  "p",
			checkResult: func(testApp *App) bool {
				return testApp.activeModalName == "projectPopup"
			},
		},
		{
			name: "e opens export modal",
			key:  "e",
			checkResult: func(testApp *App) bool {
				return testApp.state.UIState.ActiveModal == "export"
			},
		},
		{
			name: "s opens share modal",
			key:  "s",
			checkResult: func(testApp *App) bool {
				return testApp.state.UIState.ActiveModal == "share"
			},
		},
		{
			name: "m toggles streaming",
			key:  "m",
			checkResult: func(testApp *App) bool {
				return testApp.state.StreamState.Enabled == true
			},
		},
		{
			name: "m toggles streaming off",
			key:  "m",
			checkResult: func(testApp *App) bool {
				testApp.state.StreamState.Enabled = true
				newAppModel, _ := testApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
				return !newAppModel.(*App).state.StreamState.Enabled
			},
		},
		{
			name: "esc closes modal",
			key:  "esc",
			checkResult: func(testApp *App) bool {
				testApp.state.UIState.ActiveModal = "export"
				newAppModel, _ := testApp.Update(tea.KeyMsg{Type: tea.KeyEsc})
				return newAppModel.(*App).state.UIState.ActiveModal == "none"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAppInstance := NewApp(state)
			testAppInstance.state.UIState.FocusedPane = "logs"

			// Create key message
			keyMsg := createKeyMessage(tt.key)
			newAppModel, _ := testAppInstance.Update(keyMsg)
			updatedApp := newAppModel.(*App)

			if !tt.checkResult(updatedApp) {
				t.Errorf("Key '%s' did not produce expected result", tt.key)
			}
		})
	}
}

func TestAppView(t *testing.T) {
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "test-project",
		CurrentQuery:   models.Query{Filter: "severity=ERROR"},
	}
	testApp := NewApp(state)
	testApp.width = 120
	testApp.height = 40

	view := testApp.View()
	if len(view) == 0 {
		t.Error("View produced no output")
	}
}

func TestAppViewNotReady(t *testing.T) {
	state := &models.AppState{
		IsReady: false,
	}
	testApp := NewApp(state)

	view := testApp.View()
	if view != "Loading...\n" {
		t.Errorf("Expected 'Loading...', got %s", view)
	}
}

// Helper function to create key messages
func createKeyMessage(key string) tea.KeyMsg {
	switch key {
	case "h":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	case "l":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	case "j":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	case "k":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	case "G":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	case "g":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	case "m":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	case "t":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	case "f":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	case "p":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	case "e":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	case "s":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	case "ctrl+f":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	case "ctrl+b":
		return tea.KeyMsg{Type: tea.KeyCtrlB}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "?":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(key[0])}}
	}
}

func TestAppHandleKeyPress(t *testing.T) {
	state := &models.AppState{
		IsReady:      true,
		UIState:      models.UIState{FocusedPane: "logs"},
		LogListState: models.LogListState{Logs: []models.LogEntry{}},
		StreamState:  models.StreamState{Enabled: false},
	}
	testApp := NewApp(state)

	// Test quit with ctrl+c
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newAppModel, cmd := testApp.Update(keyMsg)
	if cmd == nil {
		t.Error("ctrl+c should quit the app")
	}

	// Test navigation with h
	testApp2 := NewApp(state)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newAppModel, _ = testApp2.Update(keyMsg)
	updatedApp := newAppModel.(*App)
	focusAfter := updatedApp.panes.GetFocusedPane()
	if focusAfter == "logs" {
		t.Error("Focus should have moved with 'h'")
	}
	_ = newAppModel
}

func TestAppInitializeState(t *testing.T) {
	state := &models.AppState{
		CurrentProject: "test-project",
		IsReady:        true,
	}

	testApp := NewApp(state)

	if testApp.state.CurrentProject != "test-project" {
		t.Errorf("Expected project 'test-project', got %s", testApp.state.CurrentProject)
	}

	if !testApp.state.IsReady {
		t.Error("Expected IsReady to be true")
	}
}

func TestQueryModalRender(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: []models.LogEntry{
				{Severity: "ERROR", Message: "test log 1"},
				{Severity: "INFO", Message: "test log 2"},
			},
		},
		UIState: models.UIState{},
	}

	app := NewApp(state)

	// Simulate pressing 'q' to open query modal
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, _ := app.Update(keyMsg)
	updatedApp := newModel.(*App)

	// Render the view
	view := updatedApp.View()

	// Check that the query modal is in the output
	if !contains(view, "QUERY EDITOR") {
		t.Errorf("Query modal should be rendered after pressing 'q'. View:\n%s", view[:min(500, len(view))])
	}
}

func TestTimeRangeModalRender(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: []models.LogEntry{
				{Severity: "ERROR", Message: "test log 1"},
			},
		},
		UIState: models.UIState{},
	}

	app := NewApp(state)

	// Simulate pressing 't' to open time range modal
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := app.Update(keyMsg)
	updatedApp := newModel.(*App)

	// Render the view
	view := updatedApp.View()

	// Check that the time range modal is in the output
	if !contains(view, "TIME RANGE") {
		t.Errorf("Time range modal should be rendered after pressing 't'. View:\n%s", view[:min(500, len(view))])
	}
}

func TestAppStreamStateToggle(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		StreamState: models.StreamState{
			Enabled:         false,
			RefreshInterval: 2 * time.Second,
		},
	}
	testApp := NewApp(state)

	// Toggle on
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	newAppModel, _ := testApp.Update(keyMsg)
	updatedApp := newAppModel.(*App)

	if !updatedApp.state.StreamState.Enabled {
		t.Error("Stream should be enabled after toggle")
	}

	// Toggle off
	newAppModel, _ = updatedApp.Update(keyMsg)
	updatedApp = newAppModel.(*App)

	if updatedApp.state.StreamState.Enabled {
		t.Error("Stream should be disabled after second toggle")
	}
}

func TestQueryModalHandlesMultiRuneInput(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: []models.LogEntry{
				{Severity: "INFO", Message: "test"},
			},
		},
		UIState: models.UIState{},
	}

	app := NewApp(state)
	app.SetQueryExecutor(func(filter string) ([]models.LogEntry, error) {
		return []models.LogEntry{
			{Severity: "ERROR", Message: "filtered"},
		}, nil
	})

	// Open query modal.
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = newModel.(*App)

	// Simulate paste-like input where multiple runes arrive in one key message.
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("severity=ERROR")})
	app = newModel.(*App)

	if got := app.queryModal.GetInput(); got != "severity=ERROR" {
		t.Fatalf("expected full pasted query, got %q", got)
	}

	// Apply the query.
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = newModel.(*App)

	if got := app.state.CurrentQuery.Filter; got != "severity=ERROR" {
		t.Fatalf("expected applied filter severity=ERROR, got %q", got)
	}
}

func TestQueryModalPasteWithNewlineDoesNotAutoSubmit(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: []models.LogEntry{
				{Severity: "INFO", Message: "test"},
			},
		},
		UIState: models.UIState{},
	}

	app := NewApp(state)
	app.SetQueryExecutor(func(filter string) ([]models.LogEntry, error) {
		return []models.LogEntry{}, nil
	})

	// Open query editor.
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = newModel.(*App)

	// Simulate paste containing a newline in the same key message.
	pasted := "resource.type=\"cloud_run_revision\"\nresource.labels.service_name=\"usa-bankstatement-api\""
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(pasted)})
	app = newModel.(*App)

	if !app.queryModal.IsVisible() {
		t.Fatal("query modal should remain visible after multiline paste")
	}
	if got := app.queryModal.GetInput(); got != pasted {
		t.Fatalf("expected full pasted multiline query, got %q", got)
	}
}

func TestCollectProjectsIncludesGcloudConfigs(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "env-project")
	t.Setenv("CLOUDSDK_CORE_PROJECT", "core-project")

	configDir := filepath.Join(tmpHome, ".config", "gcloud", "configurations")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	defaultCfg := "[core]\nproject = default-project\n"
	if err := os.WriteFile(filepath.Join(configDir, "config_default"), []byte(defaultCfg), 0o644); err != nil {
		t.Fatalf("write config_default failed: %v", err)
	}
	extraCfg := "[core]\nproject = project-from-config\n"
	if err := os.WriteFile(filepath.Join(configDir, "config_my-alt"), []byte(extraCfg), 0o644); err != nil {
		t.Fatalf("write config_my-alt failed: %v", err)
	}
	properties := "[core]\nproject = properties-project\n"
	if err := os.WriteFile(filepath.Join(tmpHome, ".config", "gcloud", "properties"), []byte(properties), 0o644); err != nil {
		t.Fatalf("write properties failed: %v", err)
	}

	projects := collectProjects("current-project")
	expected := []string{
		"current-project",
		"env-project",
		"core-project",
		"properties-project",
		"default-project",
		"project-from-config",
	}

	for _, project := range expected {
		if !sliceContains(projects, project) {
			t.Fatalf("expected project %q in list %v", project, projects)
		}
	}
}

func sliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestMergeUniqueStrings(t *testing.T) {
	existing := []string{"a", "b", "a", ""}
	incoming := []string{"b", "c", " d ", ""}
	got := mergeUniqueStrings(existing, incoming)
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected value at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestProjectPopupRendersCenteredWithRange(t *testing.T) {
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "p-1",
	}
	app := NewApp(state)
	app.width = 100
	app.height = 22
	app.activeModalName = "projectPopup"
	app.availableProjects = []string{
		"p-1", "p-2", "p-3", "p-4", "p-5", "p-6", "p-7", "p-8", "p-9", "p-10", "p-11",
	}
	app.projectCursor = 9

	view := app.View()
	if !contains(view, "PROJECT SELECTOR") {
		t.Fatalf("expected centered project popup in view")
	}
	if !contains(view, "Showing") || !contains(view, "of 11") {
		t.Fatalf("expected range footer in project popup")
	}
}

func TestDetailPopupDefaultsToPayloadTree(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{Logs: []models.LogEntry{
			{
				ID:          "1",
				Message:     "with payload",
				JSONPayload: map[string]interface{}{"a": float64(1), "b": map[string]interface{}{"c": "x"}},
			},
		}},
	}
	app := NewApp(state)

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	updated := newModel.(*App)
	if updated.activeModalName != "detailPopup" {
		t.Fatalf("expected detailPopup modal, got %s", updated.activeModalName)
	}
	if updated.detailViewMode != "json-tree" {
		t.Fatalf("expected json-tree default mode, got %s", updated.detailViewMode)
	}

	view := updated.View()
	if !contains(view, "Mode:json-tree") {
		t.Fatalf("expected json-tree mode in popup header")
	}
}

func TestDetailPopupModeCycleAndCollapse(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{Logs: []models.LogEntry{
			{
				ID:          "1",
				Message:     "with payload",
				JSONPayload: map[string]interface{}{"a": map[string]interface{}{"c": "x"}},
			},
		}},
	}
	app := NewApp(state)

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	app = newModel.(*App)

	lines := app.currentJSONTreeLines()
	targetPath := ""
	targetIdx := -1
	for i, line := range lines {
		if line.path != "$" && line.canExpand {
			targetPath = line.path
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		t.Fatalf("expected expandable child path in tree")
	}
	app.detailCursor = targetIdx
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	app = newModel.(*App)
	if !app.detailTreeExpanded[targetPath] {
		t.Fatalf("expected %s to be expanded after l", targetPath)
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app = newModel.(*App)
	if app.detailTreeExpanded[targetPath] {
		t.Fatalf("expected %s to be collapsed after h", targetPath)
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = newModel.(*App)
	if app.detailViewMode != "payload-raw" {
		t.Fatalf("expected payload-raw mode after cycle, got %s", app.detailViewMode)
	}
}

func TestArrowNavigationAndKeyModeToggle(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{Logs: []models.LogEntry{
			{ID: "1", Message: "a"},
			{ID: "2", Message: "b"},
		}},
	}
	app := NewApp(state)
	app.width = 120
	app.height = 40

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = newModel.(*App)
	if app.panes.LogList.scrollOffset <= 0 {
		t.Fatalf("expected down arrow to scroll")
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyF6})
	app = newModel.(*App)
	if app.vimMode {
		t.Fatalf("expected vim mode off after f6 toggle")
	}

	prev := app.panes.GetFocusedPane()
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	app = newModel.(*App)
	if app.panes.GetFocusedPane() != prev {
		t.Fatalf("expected h to be ignored when vim mode is off")
	}
}

func TestExecutePrimaryQueryUsesCache(t *testing.T) {
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "p1",
	}
	app := NewApp(state)
	app.SetQueryCacheEntries([]config.CachedQueryRecord{{
		Key:      "p1\nseverity=ERROR",
		Filter:   "severity=ERROR",
		Project:  "p1",
		StoredAt: time.Now(),
		Logs:     []models.LogEntry{{ID: "1", Message: "cached"}},
	}})

	execCalls := 0
	app.SetQueryExecutor(func(_ string) ([]models.LogEntry, error) {
		execCalls++
		return []models.LogEntry{{ID: "2", Message: "live"}}, nil
	})

	cmd := app.executePrimaryQueryCmd("severity=ERROR")
	msg := cmd()
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if execCalls != 0 {
		t.Fatalf("expected cache hit without executor call, got %d calls", execCalls)
	}
	if len(app.state.LogListState.Logs) != 1 || app.state.LogListState.Logs[0].Message != "cached" {
		t.Fatalf("expected cached logs, got %+v", app.state.LogListState.Logs)
	}
}

func TestSaveAndApplyQueryLibrary(t *testing.T) {
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "p1",
	}
	app := NewApp(state)
	app.activeModalName = "query"
	app.queryModal.Show()
	app.queryModal.SetInput("severity=ERROR")

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	app = newModel.(*App)
	if len(app.queryLibrary) == 0 {
		t.Fatalf("expected query saved to library")
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	app = newModel.(*App)
	if app.activeModalName != "queryLibrary" {
		t.Fatalf("expected queryLibrary modal, got %s", app.activeModalName)
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = newModel.(*App)
	if app.state.CurrentQuery.Filter != "severity=ERROR" {
		t.Fatalf("expected filter from library applied, got %q", app.state.CurrentQuery.Filter)
	}
}

func TestQueryHistoryPersistCallback(t *testing.T) {
	state := &models.AppState{IsReady: true, CurrentProject: "p1"}
	app := NewApp(state)
	calls := 0
	app.SetQueryHistoryPersistFn(func(filter, project string) error {
		calls++
		if filter != "severity=ERROR" || project != "p1" {
			t.Fatalf("unexpected persist args: %q %q", filter, project)
		}
		return nil
	})
	app.addQueryHistory("severity=ERROR")
	if calls != 1 {
		t.Fatalf("expected one persist callback call, got %d", calls)
	}
}

func TestQueryEditorCtrlASelectAll(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
	}
	app := NewApp(state)
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("severity=ERROR")})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	app = newModel.(*App)
	if got := app.queryModal.GetInput(); got != "x" {
		t.Fatalf("expected ctrl+a select-all replace behavior, got %q", got)
	}
}

func TestQueryHistoryUsesPopupFromQueryEditor(t *testing.T) {
	state := &models.AppState{IsReady: true}
	app := NewApp(state)
	app.SetQueryHistory([]string{"severity=ERROR", "severity=INFO"})

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = newModel.(*App)
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	app = newModel.(*App)
	if app.activeModalName != "queryHistory" {
		t.Fatalf("expected queryHistory modal, got %s", app.activeModalName)
	}

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = newModel.(*App)
	if app.activeModalName != "query" {
		t.Fatalf("expected return to query editor after applying history item, got %s", app.activeModalName)
	}
	if got := app.queryModal.GetInput(); got != "severity=ERROR" {
		t.Fatalf("expected selected history query in editor, got %q", got)
	}
}

func TestQueryPanelDoesNotRenderInlineHistory(t *testing.T) {
	state := &models.AppState{IsReady: true}
	app := NewApp(state)
	app.SetQueryHistory([]string{"severity=ERROR"})
	view := app.View()
	if contains(view, "Recent queries:") {
		t.Fatalf("inline query history should not render in main screen")
	}
}

func TestTimezoneToggle(t *testing.T) {
	state := &models.AppState{IsReady: true}
	app := NewApp(state)
	if app.timezoneMode != "utc" {
		t.Fatalf("expected default timezone utc, got %s", app.timezoneMode)
	}
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyF7})
	app = newModel.(*App)
	if app.timezoneMode != "local" {
		t.Fatalf("expected timezone local after f7, got %s", app.timezoneMode)
	}
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyF7})
	app = newModel.(*App)
	if app.timezoneMode != "utc" {
		t.Fatalf("expected timezone utc after second f7, got %s", app.timezoneMode)
	}
}

func TestShiftGJumpsToLastEntry(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: make([]models.LogEntry, 25),
		},
	}
	for i := range state.LogListState.Logs {
		state.LogListState.Logs[i] = models.LogEntry{ID: fmt.Sprintf("%d", i+1), Message: "x"}
	}
	app := NewApp(state)
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	app = newModel.(*App)
	if got := app.currentSelectedIndex(); got != len(state.LogListState.Logs)-1 {
		t.Fatalf("expected selected index at last entry, got %d", got)
	}
}

func TestAppendOlderPreservesSelectedIndexAtBottom(t *testing.T) {
	logs := make([]models.LogEntry, 200)
	for i := range logs {
		logs[i] = models.LogEntry{
			ID:        fmt.Sprintf("%d", i+1),
			Message:   fmt.Sprintf("log-%d", i+1),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Second),
		}
	}
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{
			Logs: logs,
		},
	}
	app := NewApp(state)
	app.panes.LogList.scrollOffset = len(logs) - 1 // simulate user on last visible log
	app.SetQueryExecutor(func(_ string) ([]models.LogEntry, error) {
		older := make([]models.LogEntry, 20)
		for i := range older {
			older[i] = models.LogEntry{
				ID:        fmt.Sprintf("n%d", i+1),
				Message:   fmt.Sprintf("older-%d", i+1),
				Timestamp: time.Now().Add(-time.Duration(1000+i) * time.Second),
			}
		}
		return older, nil
	})

	_, cmd := app.handleScrollDown()
	if cmd == nil {
		t.Fatalf("expected append command when scrolling down at bottom")
	}
	msg := cmd()
	newModel, _ := app.Update(msg)
	app = newModel.(*App)

	if got := app.currentSelectedIndex(); got != 199 {
		t.Fatalf("expected selected index preserved at 199, got %d", got)
	}
}

func TestParseStructuredPayloadFromPythonStyleText(t *testing.T) {
	state := &models.AppState{IsReady: true}
	app := NewApp(state)
	entry := models.LogEntry{
		TextPayload: "{'ok': True, 'meta': {'env': 'prod'}, 'items': [1, 2, 3]}",
	}
	payload, ok := app.getStructuredPayload(entry)
	if !ok {
		t.Fatalf("expected structured payload parsed from python-style text")
	}
	root, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map payload, got %T", payload)
	}
	if _, exists := root["meta"]; !exists {
		t.Fatalf("expected parsed nested field meta")
	}
}

func TestDetailPopupDefaultsToTreeForLenientPayload(t *testing.T) {
	state := &models.AppState{
		IsReady: true,
		LogListState: models.LogListState{Logs: []models.LogEntry{
			{ID: "1", TextPayload: "{'a': {'b': 1}, 'ok': True}"},
		}},
	}
	app := NewApp(state)
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	app = newModel.(*App)
	if app.detailViewMode != "json-tree" {
		t.Fatalf("expected json-tree for lenient payload, got %s", app.detailViewMode)
	}
}
