package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// ExecuteUsingGcloud executes a query using gcloud CLI instead of Go client
func (e *Executor) ExecuteUsingGcloud(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	startTime := time.Now()

	// Validate filter
	if err := e.validator.ValidateFilter(req.Filter); err != nil {
		return ExecuteResponse{}, fmt.Errorf("query validation failed: %w", err)
	}

	// Set defaults
	if req.PageSize <= 0 {
		req.PageSize = 100
	}

	// Call gcloud logging read with JSON output
	cmd := exec.CommandContext(ctx, "gcloud", "logging", "read", req.Filter,
		fmt.Sprintf("--project=%s", e.projectID),
		fmt.Sprintf("--limit=%d", req.PageSize),
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		return ExecuteResponse{
			Entries:    []models.LogEntry{},
			TotalCount: 0,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("gcloud command failed: %w", err)
	}

	// Parse JSON output
	var gcloudEntries []map[string]interface{}
	err = json.Unmarshal(output, &gcloudEntries)
	if err != nil {
		return ExecuteResponse{
			Entries:    []models.LogEntry{},
			TotalCount: 0,
			ExecutedAt: time.Now(),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("failed to parse gcloud output: %w", err)
	}

	// Convert gcloud entries to our model
	entries := []models.LogEntry{}
	for _, entry := range gcloudEntries {
		modelEntry := models.LogEntry{}

		if ts, ok := entry["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				modelEntry.Timestamp = t
			}
		}

		if sev, ok := entry["severity"].(string); ok {
			modelEntry.Severity = sev
		}

		// Try to get message from various fields
		if text, ok := entry["textPayload"].(string); ok && text != "" {
			modelEntry.Message = text
		} else if jsonPayload, ok := entry["jsonPayload"].(map[string]interface{}); ok {
			if msg, ok := jsonPayload["message"].(string); ok {
				modelEntry.Message = msg
			}
		}

		// Get labels if they exist
		if labels, ok := entry["labels"].(map[string]interface{}); ok {
			modelEntry.Labels = make(map[string]string)
			for k, v := range labels {
				if str, ok := v.(string); ok {
					modelEntry.Labels[k] = str
				}
			}
		}

		entries = append(entries, modelEntry)
	}

	return ExecuteResponse{
		Entries:    entries,
		TotalCount: len(entries),
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}, nil
}
