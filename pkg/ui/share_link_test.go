package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewShareLinkGenerator(t *testing.T) {
	slg := NewShareLinkGenerator("https://example.com")

	if slg.baseURL != "https://example.com" {
		t.Errorf("Expected base URL to be set")
	}
}

func TestNewShareLinkGeneratorDefault(t *testing.T) {
	slg := NewShareLinkGenerator("")

	if slg.baseURL == "" {
		t.Error("Should have default base URL")
	}

	if !strings.Contains(slg.baseURL, "cloud.google.com") {
		t.Error("Default should be GCP URL")
	}
}

func TestGenerateLink(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	query := models.Query{
		Project: "test-project",
		Filter:  "severity=ERROR",
	}

	filterState := models.FilterState{
		TimeRange: models.TimeRange{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	link, err := slg.GenerateLink(query, filterState)
	if err != nil {
		t.Fatalf("GenerateLink failed: %v", err)
	}

	if !strings.Contains(link, "test-project") {
		t.Error("Link should contain project")
	}

	if !strings.Contains(link, "severity") && !strings.Contains(link, "ERROR") {
		t.Error("Link should contain filter (may be URL encoded)")
	}

	if !strings.Contains(link, "startTime") {
		t.Error("Link should contain start time")
	}

	if !strings.Contains(link, "endTime") {
		t.Error("Link should contain end time")
	}
}

func TestGenerateCompactLink(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	query := models.Query{
		Project: "test-project",
		Filter:  "severity=ERROR",
	}

	filterState := models.FilterState{}

	link, err := slg.GenerateCompactLink(query, filterState)
	if err != nil {
		t.Fatalf("GenerateCompactLink failed: %v", err)
	}

	if !strings.Contains(link, "q=") {
		t.Error("Compact link should have q parameter")
	}
}

func TestDecodeLink(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	query := models.Query{
		Project: "test-project",
		Filter:  "severity=ERROR",
	}

	filterState := models.FilterState{}

	// Generate a link
	link, err := slg.GenerateLink(query, filterState)
	if err != nil {
		t.Fatalf("GenerateLink failed: %v", err)
	}

	// Decode it
	decodedQuery, _, err := slg.DecodeLink(link)
	if err != nil {
		t.Fatalf("DecodeLink failed: %v", err)
	}

	if decodedQuery.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got %s", decodedQuery.Project)
	}

	if decodedQuery.Filter != "severity=ERROR" {
		t.Errorf("Expected filter 'severity=ERROR', got %s", decodedQuery.Filter)
	}
}

func TestGetShareableQueryString(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	query := models.Query{
		Project: "test-project",
		Filter:  "severity=ERROR",
	}

	qs := slg.GetShareableQueryString(query)

	if !strings.Contains(qs, "test-project") {
		t.Error("Should contain project")
	}

	if !strings.Contains(qs, "severity=ERROR") {
		t.Error("Should contain filter")
	}

	if !strings.Contains(qs, ":") {
		t.Error("Should contain separator")
	}
}

func TestParseShareableQueryString(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	qs := "test-project:severity=ERROR"

	query, err := slg.ParseShareableQueryString(qs)
	if err != nil {
		t.Fatalf("ParseShareableQueryString failed: %v", err)
	}

	if query.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got %s", query.Project)
	}

	if query.Filter != "severity=ERROR" {
		t.Errorf("Expected filter 'severity=ERROR', got %s", query.Filter)
	}
}

func TestParseShareableQueryStringInvalid(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	_, err := slg.ParseShareableQueryString("invalid")
	if err == nil {
		t.Error("Should error on invalid format")
	}
}

func TestValidate(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	if !slg.Validate("https://example.com?foo=bar") {
		t.Error("Valid URL should validate")
	}

	if slg.Validate("not a url at all @#$%") {
		t.Error("Invalid URL should not validate")
	}
}

func TestGetQueryURL(t *testing.T) {
	slg := NewShareLinkGenerator("")

	url := slg.GetQueryURL("test-project", "severity=ERROR")

	if !strings.Contains(url, "console.cloud.google.com") {
		t.Error("Should contain GCP console URL")
	}

	if !strings.Contains(url, "test-project") {
		t.Error("Should contain project")
	}

	if !strings.Contains(url, "query=") {
		t.Error("Should contain query parameter")
	}
}

func TestGenerateLinkWithSeverity(t *testing.T) {
	slg := NewShareLinkGenerator("https://test.com")

	query := models.Query{
		Project: "test-project",
		Filter:  "resource.type=gae_app",
	}

	filterState := models.FilterState{
		Severity: models.SeverityFilter{
			Levels: []string{models.SeverityError, models.SeverityCritical},
			Mode:   "individual",
		},
	}

	link, err := slg.GenerateLink(query, filterState)
	if err != nil {
		t.Fatalf("GenerateLink failed: %v", err)
	}

	if !strings.Contains(link, "severity=") {
		t.Error("Link should contain severity parameter")
	}

	if !strings.Contains(link, "severityMode=") {
		t.Error("Link should contain severity mode")
	}
}
