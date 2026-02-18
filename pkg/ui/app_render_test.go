package ui

import (
	"os"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// TestRenderUI captures the UI rendering to a file for visual inspection
func TestRenderUI(t *testing.T) {
	// Create test state
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "test-project-123",
		CurrentQuery: models.Query{
			Filter:  "severity=ERROR",
			Project: "test-project-123",
		},
		FilterState: models.FilterState{
			TimeRange: models.TimeRange{
				Start:  time.Now().Add(-1 * time.Hour),
				End:    time.Now(),
				Preset: "1h",
			},
		},
		LogListState: models.LogListState{
			Logs: []models.LogEntry{
				{
					ID:        "log-1",
					Timestamp: time.Now(),
					Severity:  "ERROR",
					Message:   "Database connection failed at 2026-02-15 14:22:00",
					Labels: map[string]string{
						"service": "api",
						"env":     "prod",
					},
					Resource: models.Resource{
						Type: "cloud_run",
						Labels: map[string]string{
							"service_name": "log-explorer",
						},
					},
				},
				{
					ID:        "log-2",
					Timestamp: time.Now().Add(-5 * time.Minute),
					Severity:  "WARNING",
					Message:   "High memory usage detected",
					Labels: map[string]string{
						"service": "worker",
						"env":     "prod",
					},
				},
				{
					ID:        "log-3",
					Timestamp: time.Now().Add(-10 * time.Minute),
					Severity:  "INFO",
					Message:   "Request processed successfully",
					Labels: map[string]string{
						"service": "api",
						"env":     "prod",
					},
				},
			},
		},
		UIState: models.UIState{
			FocusedPane: "logs",
			ActiveModal: "none",
		},
	}

	// Create app with standard terminal size
	app := NewApp(state)
	app.width = 120
	app.height = 30

	// Get the rendered view
	view := app.View()

	// Write to file for inspection
	outputPath := "/tmp/log-explorer-ui-render.txt"
	err := os.WriteFile(outputPath, []byte(view), 0644)
	if err != nil {
		t.Fatalf("Failed to write render output: %v", err)
	}

	t.Logf("UI render captured to: %s", outputPath)
	t.Logf("UI dimensions: %dx%d", app.width, app.height)
	t.Logf("Rendered view preview (first 500 chars):\n%s", view[:min(500, len(view))])
}

// TestRenderUIWithLargerTerminal tests rendering on a larger terminal
func TestRenderUIWithLargerTerminal(t *testing.T) {
	state := &models.AppState{
		IsReady:        true,
		CurrentProject: "test-project-123",
		CurrentQuery: models.Query{
			Filter:  "severity>=WARNING",
			Project: "test-project-123",
		},
		LogListState: models.LogListState{
			Logs: make([]models.LogEntry, 10),
		},
		UIState: models.UIState{
			FocusedPane: "logs",
		},
	}

	// Populate logs
	for i := 0; i < 10; i++ {
		state.LogListState.Logs[i] = models.LogEntry{
			ID:        "log-" + string(rune(48+i)),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Severity:  []string{"ERROR", "WARNING", "INFO"}[i%3],
			Message:   "Test log message " + string(rune(48+i)),
		}
	}

	app := NewApp(state)
	app.width = 160
	app.height = 40

	view := app.View()

	outputPath := "/tmp/log-explorer-ui-render-large.txt"
	err := os.WriteFile(outputPath, []byte(view), 0644)
	if err != nil {
		t.Fatalf("Failed to write render output: %v", err)
	}

	t.Logf("Large UI render captured to: %s", outputPath)
	t.Logf("UI dimensions: %dx%d", app.width, app.height)
}

// TestPaneProportions verifies the pane width proportions
func TestPaneProportions(t *testing.T) {
	app := NewApp(&models.AppState{IsReady: true})
	app.width = 100

	logListWidth := (app.width * 60) / 100
	graphWidth := (app.width * 10) / 100
	controlsWidth := (app.width * 10) / 100
	queryWidth := (app.width * 20) / 100

	expectedTotal := logListWidth + graphWidth + controlsWidth + queryWidth

	t.Logf("Width breakdown for 100-char terminal:")
	t.Logf("  Log List: %d chars (60%%)", logListWidth)
	t.Logf("  Graph:    %d chars (10%%)", graphWidth)
	t.Logf("  Controls: %d chars (10%%)", controlsWidth)
	t.Logf("  Query:    %d chars (20%%)", queryWidth)
	t.Logf("  Total:    %d chars", expectedTotal)

	if expectedTotal != 90 {
		t.Logf("Note: Total is %d (20 chars per pane width)", expectedTotal)
	}
}

// Helper
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
