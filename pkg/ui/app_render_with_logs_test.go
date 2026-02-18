package ui

import (
	"os"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// TestRenderUIWithLogs shows the UI with actual log entries
func TestRenderUIWithLogs(t *testing.T) {
	// Create realistic log entries
	now := time.Now()
	logs := []models.LogEntry{
		{
			ID:        "log-001",
			Timestamp: now,
			Severity:  "ERROR",
			Message:   "[14:22:28] Database connection timeout after 30s",
			Labels: map[string]string{
				"service": "api-server",
				"env":     "production",
			},
			Resource: models.Resource{
				Type: "cloud_run",
				Labels: map[string]string{
					"service_name": "log-explorer-api",
				},
			},
		},
		{
			ID:        "log-002",
			Timestamp: now.Add(-2 * time.Minute),
			Severity:  "WARNING",
			Message:   "[14:20:15] Memory usage exceeded 80% threshold",
			Labels: map[string]string{
				"service": "worker",
				"env":     "production",
			},
			Resource: models.Resource{
				Type: "cloud_run",
				Labels: map[string]string{
					"service_name": "background-processor",
				},
			},
		},
		{
			ID:        "log-003",
			Timestamp: now.Add(-5 * time.Minute),
			Severity:  "ERROR",
			Message:   "[14:17:45] Failed to authenticate user: invalid credentials",
			Labels: map[string]string{
				"service": "auth",
				"env":     "production",
			},
		},
		{
			ID:        "log-004",
			Timestamp: now.Add(-8 * time.Minute),
			Severity:  "INFO",
			Message:   "[14:14:30] Cache invalidation completed for 1254 keys",
			Labels: map[string]string{
				"service": "cache",
				"env":     "production",
			},
		},
		{
			ID:        "log-005",
			Timestamp: now.Add(-12 * time.Minute),
			Severity:  "DEBUG",
			Message:   "[14:10:15] Processing batch job: 500 items queued",
			Labels: map[string]string{
				"service": "job-processor",
				"env":     "production",
			},
		},
	}

	// Create detailed app state
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "cloud-run-testing-272918",
		CurrentQuery: models.Query{
			Filter:  "severity>=WARNING resource.type=cloud_run",
			Project: "cloud-run-testing-272918",
		},
		FilterState: models.FilterState{
			TimeRange: models.TimeRange{
				Start:  now.Add(-1 * time.Hour),
				End:    now,
				Preset: "1h",
			},
			Severity: models.SeverityFilter{
				Levels: []string{models.SeverityError, models.SeverityWarning},
				Mode:   "individual",
			},
		},
		LogListState: models.LogListState{
			Logs:         logs,
			CurrentIndex: 0,
			IsLoading:    false,
		},
		UIState: models.UIState{
			FocusedPane: "logs",
			ActiveModal: "none",
		},
	}

	app := NewApp(state)
	app.width = 140
	app.height = 35

	view := app.View()

	outputPath := "/tmp/log-explorer-ui-with-logs.txt"
	err := os.WriteFile(outputPath, []byte(view), 0644)
	if err != nil {
		t.Fatalf("Failed to write render output: %v", err)
	}

	// Also print summary
	t.Logf("\n=== UI RENDER TEST RESULTS ===")
	t.Logf("Output saved to: %s", outputPath)
	t.Logf("Dimensions: %dx%d", app.width, app.height)
	t.Logf("Project: %s", state.CurrentProject)
	t.Logf("Query: %s", state.CurrentQuery.Filter)
	t.Logf("Logs loaded: %d", len(state.LogListState.Logs))
	t.Logf("\n=== PREVIEW ===")
	
	// Show first 1000 chars
	if len(view) > 1000 {
		t.Logf("Preview:\n%s", view[:1000])
	} else {
		t.Logf("Preview:\n%s", view)
	}
}
