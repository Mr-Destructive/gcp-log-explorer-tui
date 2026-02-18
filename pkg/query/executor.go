package query

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/user/log-explorer-tui/pkg/models"
)

// Executor handles query execution against GCP Logging
type Executor struct {
	client      *logging.Client
	adminClient *logadmin.Client
	projectID   string
	timeout     time.Duration
	validator   *Validator
}

// NewExecutor creates a new query executor
func NewExecutor(client *logging.Client, projectID string, timeout time.Duration) *Executor {
	return &Executor{
		client:      client,
		adminClient: nil, // Will be created lazily if needed
		projectID:   projectID,
		timeout:     timeout,
		validator:   NewValidator(),
	}
}

// ExecuteRequest represents parameters for query execution
type ExecuteRequest struct {
	Filter      string
	PageSize    int
	PageToken   string
	OrderBy     string
}

// ExecuteResponse represents the result of query execution
type ExecuteResponse struct {
	Entries       []models.LogEntry
	NextPageToken string
	PrevPageToken string
	TotalCount    int
	ExecutedAt    time.Time
	Duration      time.Duration
}

// Execute runs a query and returns results
func (e *Executor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	// Validate filter
	if err := e.validator.ValidateFilter(req.Filter); err != nil {
		return ExecuteResponse{}, fmt.Errorf("query validation failed: %w", err)
	}

	// Add timeout to context
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	startTime := time.Now()

	// Set defaults
	if req.PageSize <= 0 {
		req.PageSize = 100
	}
	if req.OrderBy == "" {
		req.OrderBy = "timestamp desc"
	}

	// Query logs from GCP using the admin client
	entries := []models.LogEntry{}

	if e.client == nil {
		// No client available, return empty (used in tests)
		return ExecuteResponse{
			Entries:    entries,
			TotalCount: 0,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, nil
	}

	// Create admin client if not already created
	if e.adminClient == nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Creating logadmin client for project: %s\n", e.projectID)
		adminClient, err := logadmin.NewClient(ctx, e.projectID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to create admin client: %v\n", err)
			return ExecuteResponse{
				Entries:    entries,
				TotalCount: 0,
				ExecutedAt: time.Now(),
				Duration:   time.Since(startTime),
			}, fmt.Errorf("failed to create admin client: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] Admin client created successfully\n")
		e.adminClient = adminClient
		defer adminClient.Close()
	}

	// Build options for Entries call
	opts := []logadmin.EntriesOption{}

	if req.Filter != "" {
		opts = append(opts, logadmin.Filter(req.Filter))
	}

	if req.PageSize > 0 {
		opts = append(opts, logadmin.PageSize(int32(req.PageSize)))
	}

	// Check if OrderBy contains "desc" for NewestFirst
	if req.OrderBy == "timestamp desc" {
		opts = append(opts, logadmin.NewestFirst())
	}

	// Use admin client to list entries
	fmt.Fprintf(os.Stderr, "[DEBUG] Executing query with filter: %s\n", req.Filter)
	fmt.Fprintf(os.Stderr, "[DEBUG] Query options: %d opts\n", len(opts))
	iter := e.adminClient.Entries(ctx, opts...)

	count := 0
	for {
		entry, err := iter.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Iterator.Next() returned error: %v\n", err)
			// End of list or error
			break
		}
		if entry == nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Iterator returned nil entry\n")
			break
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Got log entry: %s - %s\n", entry.Severity, entry.Timestamp)
		entries = append(entries, ConvertLoggingEntry(entry))
		count++

		// Respect page size limit
		if count >= req.PageSize {
			fmt.Fprintf(os.Stderr, "[DEBUG] Reached page size limit: %d\n", req.PageSize)
			break
		}
	}
	
	fmt.Fprintf(os.Stderr, "[DEBUG] Query completed, got %d entries\n", count)

	return ExecuteResponse{
		Entries:    entries,
		TotalCount: count,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, nil
}

// ConvertLoggingEntry converts cloud.google.com/go/logging.Entry to models.LogEntry
func ConvertLoggingEntry(entry *logging.Entry) models.LogEntry {
	modelEntry := models.LogEntry{
		Timestamp: entry.Timestamp,
		Severity:  entry.Severity.String(),
		Message:   extractMessage(entry),
		Labels:    entry.Labels,
	}

	// Convert resource
	if entry.Resource != nil {
		modelEntry.Resource = models.Resource{
			Type:   entry.Resource.Type,
			Labels: entry.Resource.Labels,
		}
	}

	// Store trace info
	modelEntry.Trace = entry.Trace
	modelEntry.SpanID = entry.SpanID

	// Store raw entry for detailed view
	modelEntry.Raw = entry

	return modelEntry
}

// extractMessage extracts the best message representation from an entry
func extractMessage(entry *logging.Entry) string {
	// Priority: Payload (which could be string or interface{})
	if payload, ok := entry.Payload.(string); ok {
		return payload
	}

	return fmt.Sprintf("%v", entry.Payload)
}

// ValidateAndBuild validates a filter and returns the final filter string
func (e *Executor) ValidateAndBuild(baseFilter string) (string, error) {
	if err := e.validator.ValidateFilter(baseFilter); err != nil {
		return "", err
	}
	return baseFilter, nil
}

// GetCount returns approximate count of matching logs (for display purposes)
func (e *Executor) GetCount(ctx context.Context, filter string) (int, error) {
	if err := e.validator.ValidateFilter(filter); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Placeholder for actual count implementation
	// Would use aggregation queries in production
	_ = ctx
	_ = filter

	return 0, nil
}
