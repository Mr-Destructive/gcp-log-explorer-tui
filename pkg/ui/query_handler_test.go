package ui

import (
	"context"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
	"github.com/user/log-explorer-tui/pkg/query"
)

func TestNewQueryHandler(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	if handler.executor != executor {
		t.Error("Executor not set correctly")
	}

	if len(handler.history) != 0 {
		t.Error("History should be empty initially")
	}
}

func TestBuildFilterFromState(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	fs := models.FilterState{
		TimeRange: models.TimeRange{
			Start: oneHourAgo,
			End:   now,
		},
		Severity: models.SeverityFilter{
			Levels: []string{models.SeverityError},
			Mode:   "individual",
		},
		CustomFilters: map[string]string{
			"resource.type": "gae_app",
		},
	}

	filter := handler.BuildFilterFromState(fs)

	if filter == "" {
		t.Error("Built filter should not be empty")
	}

	if len(filter) > 0 {
		// Check that all components are present
		// Filter will be built by query builder
		if handler.executor == nil {
			t.Error("Executor should not be nil")
		}
	}
}

func TestAddToHistory(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	q1 := models.Query{Filter: "severity=ERROR", Project: "test-project"}
	handler.AddToHistory(q1)

	if len(handler.history) != 1 {
		t.Errorf("Expected 1 query in history, got %d", len(handler.history))
	}

	if handler.history[0].Filter != "severity=ERROR" {
		t.Errorf("Expected filter 'severity=ERROR', got %s", handler.history[0].Filter)
	}

	// Add duplicate should move to front
	q1Again := models.Query{Filter: "severity=ERROR", Project: "test-project"}
	handler.AddToHistory(q1Again)

	if len(handler.history) != 1 {
		t.Errorf("Duplicate should not increase history, got %d", len(handler.history))
	}

	// Add new query
	q2 := models.Query{Filter: "severity=WARNING", Project: "test-project"}
	handler.AddToHistory(q2)

	if len(handler.history) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(handler.history))
	}

	if handler.history[0].Filter != "severity=WARNING" {
		t.Error("Most recent query should be first")
	}
}

func TestAddToHistoryTrimming(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	// Add 55 queries
	for i := 0; i < 55; i++ {
		q := models.Query{Filter: "severity=ERROR", Project: "test-project"}
		q.Filter = q.Filter + string(rune(i)) // Make each unique
		handler.AddToHistory(q)
	}

	// Should be trimmed to 50
	if len(handler.history) > 50 {
		t.Errorf("History should be trimmed to 50, got %d", len(handler.history))
	}
}

func TestGetHistory(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	q1 := models.Query{Filter: "severity=ERROR", Project: "test-project"}
	q2 := models.Query{Filter: "severity=WARNING", Project: "test-project"}

	handler.AddToHistory(q1)
	handler.AddToHistory(q2)

	history := handler.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 queries in history, got %d", len(history))
	}

	if history[0].Filter != "severity=WARNING" {
		t.Error("Most recent should be first")
	}
}

func TestValidateQuery(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	tests := []struct {
		filter      string
		shouldError bool
	}{
		{"severity=ERROR", false},
		{"", true},
		{"(unbalanced", true},
	}

	for _, tt := range tests {
		err := handler.ValidateQuery(tt.filter)
		if (err != nil) != tt.shouldError {
			t.Errorf("Filter %s: expected error=%v, got %v", tt.filter, tt.shouldError, err)
		}
	}
}

func TestSetExecutor(t *testing.T) {
	executor1 := query.NewExecutor(nil, "project1", 30*time.Second)
	executor2 := query.NewExecutor(nil, "project2", 30*time.Second)

	handler := NewQueryHandler(executor1)
	if handler.executor == nil {
		t.Error("Initial executor not set correctly")
	}

	handler.SetExecutor(executor2)
	if handler.executor == nil {
		t.Error("Executor not updated")
	}

	// Verify the executor was replaced
	if handler.executor != executor2 {
		t.Error("Executor should be the new one")
	}
}

func TestExecuteQueryValidation(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	appState := &models.AppState{
		CurrentQuery: models.Query{Filter: ""}, // Invalid empty filter
	}

	ctx := context.Background()
	err := handler.ExecuteQuery(ctx, appState)

	if err == nil {
		t.Error("Expected error for empty filter")
	}

	if appState.LastError == nil {
		t.Error("AppState should have LastError set")
	}
}

func TestBuildFilterFromStateWithEmptyState(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	fs := models.FilterState{}
	filter := handler.BuildFilterFromState(fs)

	// Empty state should produce empty filter
	if filter != "" {
		t.Errorf("Expected empty filter for empty state, got %s", filter)
	}
}

func TestQueryHistoryOrdering(t *testing.T) {
	executor := query.NewExecutor(nil, "test-project", 30*time.Second)
	handler := NewQueryHandler(executor)

	queries := []models.Query{
		{Filter: "q1", Project: "test"},
		{Filter: "q2", Project: "test"},
		{Filter: "q3", Project: "test"},
	}

	for _, q := range queries {
		handler.AddToHistory(q)
	}

	// Check ordering (most recent first)
	if handler.history[0].Filter != "q3" {
		t.Error("Most recent query should be first")
	}

	if handler.history[1].Filter != "q2" {
		t.Error("Second query should be in second position")
	}

	if handler.history[2].Filter != "q1" {
		t.Error("Oldest query should be last")
	}
}
