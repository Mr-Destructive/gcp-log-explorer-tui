package ui

import (
	"context"
	"fmt"

	"github.com/user/log-explorer-tui/pkg/models"
	"github.com/user/log-explorer-tui/pkg/query"
)

// QueryHandler manages query execution and history
type QueryHandler struct {
	executor *query.Executor
	builder  *query.Builder
	history  []models.Query
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(executor *query.Executor) *QueryHandler {
	return &QueryHandler{
		executor: executor,
		builder:  query.NewBuilder(""),
		history:  []models.Query{},
	}
}

// ExecuteQuery executes a query and returns results
func (qh *QueryHandler) ExecuteQuery(ctx context.Context, appState *models.AppState) error {
	// Get the current filter from query or build from filters
	filter := appState.CurrentQuery.Filter
	if filter == "" {
		// Build filter from filter state
		filter = qh.BuildFilterFromState(appState.FilterState)
	}

	// Validate filter
	validatedFilter, err := qh.executor.ValidateAndBuild(filter)
	if err != nil {
		appState.LastError = err
		return fmt.Errorf("query validation failed: %w", err)
	}

	// Execute query
	req := query.ExecuteRequest{
		Filter:   validatedFilter,
		PageSize: 100,
	}

	resp, err := qh.executor.Execute(ctx, req)
	if err != nil {
		appState.LastError = err
		return fmt.Errorf("query execution failed: %w", err)
	}

	// Update log list state
	appState.LogListState.Logs = resp.Entries
	appState.LogListState.IsLoading = false
	appState.LogListState.ErrorMessage = ""

	// Add to history
	qh.AddToHistory(appState.CurrentQuery)

	return nil
}

// BuildFilterFromState constructs a filter string from filter state
func (qh *QueryHandler) BuildFilterFromState(fs models.FilterState) string {
	builder := query.NewBuilder("")

	// Add custom filters
	for key, value := range fs.CustomFilters {
		builder.AddCustomFilter(fmt.Sprintf("%s=%q", key, value))
	}

	// Add severity filter
	if fs.Severity.Mode != "" {
		builder.AddSeverity(fs.Severity)
	}

	// Add time range
	if !fs.TimeRange.Start.IsZero() && !fs.TimeRange.End.IsZero() {
		builder.AddTimeRange(fs.TimeRange)
	}

	return builder.Build()
}

// AddToHistory adds a query to history
func (qh *QueryHandler) AddToHistory(q models.Query) {
	// Check if query already exists
	for i, existingQ := range qh.history {
		if existingQ.Filter == q.Filter && existingQ.Project == q.Project {
			// Move to front
			qh.history = append([]models.Query{q}, append(qh.history[:i], qh.history[i+1:]...)...)
			return
		}
	}

	// Add new query
	qh.history = append([]models.Query{q}, qh.history...)

	// Trim to max 50
	if len(qh.history) > 50 {
		qh.history = qh.history[:50]
	}
}

// GetHistory returns query history
func (qh *QueryHandler) GetHistory() []models.Query {
	return qh.history
}

// ValidateQuery checks if a query is valid
func (qh *QueryHandler) ValidateQuery(filter string) error {
	_, err := qh.executor.ValidateAndBuild(filter)
	return err
}

// SetExecutor sets the executor (for testing or switching)
func (qh *QueryHandler) SetExecutor(executor *query.Executor) {
	qh.executor = executor
}
