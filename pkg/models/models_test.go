package models

import (
	"testing"
	"time"
)

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		ID:        "test-id-123",
		Timestamp: time.Now(),
		Severity:  "ERROR",
		Message:   "Something went wrong",
		Labels: map[string]string{
			"env": "production",
		},
	}

	if entry.ID != "test-id-123" {
		t.Errorf("Expected ID 'test-id-123', got %s", entry.ID)
	}

	if entry.Severity != "ERROR" {
		t.Errorf("Expected Severity 'ERROR', got %s", entry.Severity)
	}

	if entry.Labels["env"] != "production" {
		t.Errorf("Expected label 'production', got %s", entry.Labels["env"])
	}
}

func TestFilterState(t *testing.T) {
	now := time.Now()
	fs := FilterState{
		TimeRange: TimeRange{
			Start:  now.Add(-1 * time.Hour),
			End:    now,
			Preset: "1h",
		},
		Severity: SeverityFilter{
			Levels: []string{SeverityError, SeverityCritical},
			Mode:   "individual",
		},
		SearchTerm: "database",
	}

	if fs.TimeRange.Preset != "1h" {
		t.Errorf("Expected preset '1h', got %s", fs.TimeRange.Preset)
	}

	if len(fs.Severity.Levels) != 2 {
		t.Errorf("Expected 2 severity levels, got %d", len(fs.Severity.Levels))
	}

	if fs.SearchTerm != "database" {
		t.Errorf("Expected search term 'database', got %s", fs.SearchTerm)
	}
}

func TestSeverityLevels(t *testing.T) {
	if len(SeverityLevels) == 0 {
		t.Fatal("SeverityLevels is empty")
	}

	expectedLevels := []string{
		SeverityDefault,
		SeverityDebug,
		SeverityInfo,
		SeverityNotice,
		SeverityWarning,
		SeverityError,
		SeverityCritical,
		SeverityAlert,
		SeverityEmergency,
	}

	if len(SeverityLevels) != len(expectedLevels) {
		t.Errorf("Expected %d severity levels, got %d", len(expectedLevels), len(SeverityLevels))
	}

	for i, level := range expectedLevels {
		if SeverityLevels[i] != level {
			t.Errorf("Expected level %s at position %d, got %s", level, i, SeverityLevels[i])
		}
	}
}

func TestTimeRangePresets(t *testing.T) {
	tests := []struct {
		preset   string
		expected time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		duration, exists := TimeRangePresets[tt.preset]
		if !exists {
			t.Errorf("Preset %s not found in TimeRangePresets", tt.preset)
		}

		if duration != tt.expected {
			t.Errorf("Preset %s: expected %v, got %v", tt.preset, tt.expected, duration)
		}
	}
}

func TestPaginationState(t *testing.T) {
	ps := PaginationState{
		NextPageTokenOlder: "token-older",
		NextPageTokenNewer: "token-newer",
		TopBoundaryReached: false,
		BottomBoundaryReached: false,
	}

	if ps.NextPageTokenOlder != "token-older" {
		t.Errorf("Expected NextPageTokenOlder 'token-older', got %s", ps.NextPageTokenOlder)
	}

	if ps.NextPageTokenNewer != "token-newer" {
		t.Errorf("Expected NextPageTokenNewer 'token-newer', got %s", ps.NextPageTokenNewer)
	}

	if ps.TopBoundaryReached || ps.BottomBoundaryReached {
		t.Error("Expected boundaries to not be reached initially")
	}
}

func TestAppState(t *testing.T) {
	appState := AppState{
		CurrentProject: "my-project",
		CurrentQuery: Query{
			Filter:  "severity=ERROR",
			Project: "my-project",
		},
		IsReady: true,
	}

	if appState.CurrentProject != "my-project" {
		t.Errorf("Expected project 'my-project', got %s", appState.CurrentProject)
	}

	if !appState.IsReady {
		t.Error("Expected IsReady to be true")
	}

	if appState.CurrentQuery.Filter != "severity=ERROR" {
		t.Errorf("Expected filter 'severity=ERROR', got %s", appState.CurrentQuery.Filter)
	}
}

func TestLogListState(t *testing.T) {
	entries := []LogEntry{
		{
			ID:        "log-1",
			Timestamp: time.Now(),
			Severity:  "INFO",
			Message:   "Test message 1",
		},
		{
			ID:        "log-2",
			Timestamp: time.Now(),
			Severity:  "ERROR",
			Message:   "Test message 2",
		},
	}

	logState := LogListState{
		Logs:         entries,
		CurrentIndex: 0,
		IsLoading:    false,
	}

	if len(logState.Logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logState.Logs))
	}

	if logState.Logs[0].ID != "log-1" {
		t.Errorf("Expected first log ID 'log-1', got %s", logState.Logs[0].ID)
	}
}

func TestStreamState(t *testing.T) {
	ss := StreamState{
		Enabled:         true,
		LastFetchTime:   time.Now(),
		RefreshInterval: 2 * time.Second,
		NewLogsCount:    5,
	}

	if !ss.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if ss.RefreshInterval != 2*time.Second {
		t.Errorf("Expected RefreshInterval 2s, got %v", ss.RefreshInterval)
	}

	if ss.NewLogsCount != 5 {
		t.Errorf("Expected NewLogsCount 5, got %d", ss.NewLogsCount)
	}
}

func TestSeverityFilter(t *testing.T) {
	tests := []struct {
		name     string
		filter   SeverityFilter
		validate func(SeverityFilter) bool
	}{
		{
			name: "individual mode",
			filter: SeverityFilter{
				Levels: []string{SeverityError, SeverityCritical},
				Mode:   "individual",
			},
			validate: func(f SeverityFilter) bool {
				return f.Mode == "individual" && len(f.Levels) == 2
			},
		},
		{
			name: "range mode",
			filter: SeverityFilter{
				MinLevel: SeverityWarning,
				Mode:     "range",
			},
			validate: func(f SeverityFilter) bool {
				return f.Mode == "range" && f.MinLevel == SeverityWarning
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.validate(tt.filter) {
				t.Errorf("Validation failed for filter: %+v", tt.filter)
			}
		})
	}
}
