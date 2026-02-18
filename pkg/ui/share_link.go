package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// ShareLinkGenerator generates shareable links for log queries
type ShareLinkGenerator struct {
	baseURL string
}

// NewShareLinkGenerator creates a new share link generator
func NewShareLinkGenerator(baseURL string) *ShareLinkGenerator {
	if baseURL == "" {
		baseURL = "https://cloud.google.com/logs-explorer"
	}
	return &ShareLinkGenerator{
		baseURL: baseURL,
	}
}

// GenerateLink generates a shareable link for a query
func (slg *ShareLinkGenerator) GenerateLink(query models.Query, filter models.FilterState) (string, error) {
	// Build query parameters
	params := map[string]string{
		"project": query.Project,
		"filter":  query.Filter,
	}

	// Add time range
	if !filter.TimeRange.Start.IsZero() && !filter.TimeRange.End.IsZero() {
		params["startTime"] = filter.TimeRange.Start.Format("2006-01-02T15:04:05Z")
		params["endTime"] = filter.TimeRange.End.Format("2006-01-02T15:04:05Z")
	}

	// Add severity filter if present
	if len(filter.Severity.Levels) > 0 {
		params["severity"] = strings.Join(filter.Severity.Levels, ",")
		params["severityMode"] = filter.Severity.Mode
	}

	// Encode as query string
	queryStr := url.Values{}
	for k, v := range params {
		queryStr.Add(k, v)
	}

	link := fmt.Sprintf("%s?%s", slg.baseURL, queryStr.Encode())
	return link, nil
}

// GenerateCompactLink generates a compact encoded link
func (slg *ShareLinkGenerator) GenerateCompactLink(query models.Query, filter models.FilterState) (string, error) {
	// Create a compact representation
	compact := map[string]interface{}{
		"p": query.Project,
		"f": query.Filter,
		"t": map[string]interface{}{
			"s": filter.TimeRange.Start.Unix(),
			"e": filter.TimeRange.End.Unix(),
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(compact)
	if err != nil {
		return "", fmt.Errorf("failed to marshal: %w", err)
	}

	// Encode as base64
	encoded := base64.URLEncoding.EncodeToString(jsonData)

	link := fmt.Sprintf("%s?q=%s", slg.baseURL, encoded)
	return link, nil
}

// DecodeLink decodes a shareable link
func (slg *ShareLinkGenerator) DecodeLink(link string) (models.Query, models.FilterState, error) {
	// Parse URL
	parsedURL, err := url.Parse(link)
	if err != nil {
		return models.Query{}, models.FilterState{}, fmt.Errorf("invalid URL: %w", err)
	}

	query := models.Query{}
	filterState := models.FilterState{
		CustomFilters: make(map[string]string),
	}

	// Check for compact format (q parameter)
	if q := parsedURL.Query().Get("q"); q != "" {
		decoded, err := base64.URLEncoding.DecodeString(q)
		if err != nil {
			return models.Query{}, models.FilterState{}, fmt.Errorf("failed to decode: %w", err)
		}

		var compact map[string]interface{}
		if err := json.Unmarshal(decoded, &compact); err != nil {
			return models.Query{}, models.FilterState{}, fmt.Errorf("failed to unmarshal: %w", err)
		}

		if p, ok := compact["p"].(string); ok {
			query.Project = p
		}
		if f, ok := compact["f"].(string); ok {
			query.Filter = f
		}

		return query, filterState, nil
	}

	// Parse standard format
	query.Project = parsedURL.Query().Get("project")
	query.Filter = parsedURL.Query().Get("filter")

	// Parse time range
	if st := parsedURL.Query().Get("startTime"); st != "" {
		// Parse as RFC3339
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			filterState.TimeRange.Start = t
		}
	}

	if et := parsedURL.Query().Get("endTime"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			filterState.TimeRange.End = t
		}
	}

	// Parse severity
	if sev := parsedURL.Query().Get("severity"); sev != "" {
		filterState.Severity.Levels = strings.Split(sev, ",")
		filterState.Severity.Mode = parsedURL.Query().Get("severityMode")
	}

	return query, filterState, nil
}

// GetShareableQueryString returns just the query portion as a string
func (slg *ShareLinkGenerator) GetShareableQueryString(query models.Query) string {
	// Format as "project_id:filter_expression"
	return fmt.Sprintf("%s:%s", query.Project, query.Filter)
}

// ParseShareableQueryString parses a shareable query string
func (slg *ShareLinkGenerator) ParseShareableQueryString(s string) (models.Query, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return models.Query{}, fmt.Errorf("invalid format, expected 'project:filter'")
	}

	return models.Query{
		Project: parts[0],
		Filter:  parts[1],
	}, nil
}

// Validate validates a link format
func (slg *ShareLinkGenerator) Validate(link string) bool {
	_, err := url.Parse(link)
	return err == nil
}

// GetQueryURL returns a direct GCP Logs Explorer URL
func (slg *ShareLinkGenerator) GetQueryURL(projectID, filter string) string {
	encoded := url.QueryEscape(filter)
	return fmt.Sprintf("https://console.cloud.google.com/logs/query?project=%s&query=%s", projectID, encoded)
}
