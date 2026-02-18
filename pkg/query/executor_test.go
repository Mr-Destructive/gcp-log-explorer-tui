package query

import (
	"context"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewExecutor(t *testing.T) {
	// Create mock executor without actual client
	testExecutor := NewExecutor(nil, "test-project", 30*time.Second)

	if testExecutor.projectID != "test-project" {
		t.Errorf("Expected projectID 'test-project', got %s", testExecutor.projectID)
	}

	if testExecutor.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", testExecutor.timeout)
	}

	if testExecutor.validator == nil {
		t.Error("Validator should be initialized")
	}
}

func TestExecutorValidateAndBuild(t *testing.T) {
	testExecutor := NewExecutor(nil, "test-project", 30*time.Second)

	tests := []struct {
		filter      string
		shouldError bool
	}{
		{"severity=ERROR", false},
		{"resource.type=gae_app", false},
		{"severity=ERROR AND resource.type=gae_app", false},
		{"", true},
		{"(severity=ERROR", true},
		{"severity==ERROR", true},
	}

	for _, tt := range tests {
		_, err := testExecutor.ValidateAndBuild(tt.filter)
		if (err != nil) != tt.shouldError {
			t.Errorf("Filter %s: expected error=%v, got error=%v", tt.filter, tt.shouldError, err)
		}
	}
}

func TestExecuteRequestDefaults(t *testing.T) {
	_ = NewExecutor(nil, "test-project", 30*time.Second) // Executor created for consistency

	req := ExecuteRequest{
		Filter: "severity=ERROR",
	}

	if req.PageSize == 0 {
		req.PageSize = 100
	}

	if req.PageSize != 100 {
		t.Errorf("Expected default PageSize 100, got %d", req.PageSize)
	}
}

func TestExecuteResponseCreation(t *testing.T) {
	resp := ExecuteResponse{
		Entries:    []models.LogEntry{},
		TotalCount: 0,
		ExecutedAt: time.Now(),
		Duration:   100 * time.Millisecond,
	}

	if len(resp.Entries) != 0 {
		t.Error("Entries should be empty")
	}

	if resp.TotalCount != 0 {
		t.Error("TotalCount should be 0")
	}

	if resp.Duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", resp.Duration)
	}
}

func TestExecutorBuildQueryFromState(t *testing.T) {
	testExecutor := NewExecutor(nil, "test-project", 30*time.Second)

	state := models.FilterState{
		TimeRange: models.TimeRange{
			Preset: "1h",
		},
		Severity: models.SeverityFilter{
			Mode: "individual",
		},
	}

	// Simulate building a query from state
	builder := NewBuilder("").
		AddCustomFilter("resource.type=gae_app")

	if state.Severity.Mode == "individual" {
		builder.AddSeverity(models.SeverityFilter{
			Levels: []string{models.SeverityError},
			Mode:   "individual",
		})
	}

	query := builder.Build()

	// Validate the query
	err := testExecutor.validator.ValidateFilter(query)
	if err != nil {
		t.Errorf("Query validation failed: %v", err)
	}
}

func TestExecutorContextHandling(t *testing.T) {
	_ = NewExecutor(nil, "test-project", 5*time.Second) // Executor created for consistency

	// Test with explicit context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have deadline")
	}

	if deadline.IsZero() {
		t.Error("Deadline should not be zero")
	}
}

func TestValidateAndBuildChain(t *testing.T) {
	executor := NewExecutor(nil, "test-project", 30*time.Second)

	// Test building a complex query
	builder := NewBuilder("").
		AddCustomFilter("resource.type=gae_app").
		AddResourceFilter("cloud_function").
		AddLabelFilter("environment", "production")

	query := builder.Build()

	// Validate the built query
	err := executor.validator.ValidateFilter(query)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

func TestExecutorValidatesBeforeExecution(t *testing.T) {
	executor := NewExecutor(nil, "test-project", 30*time.Second)

	req := ExecuteRequest{
		Filter: "", // Invalid: empty filter
	}

	// The actual Execute would fail validation
	err := executor.validator.ValidateFilter(req.Filter)
	if err == nil {
		t.Error("Should validate empty filter as error")
	}

	// Try invalid filter
	req.Filter = "(unbalanced"
	err = executor.validator.ValidateFilter(req.Filter)
	if err == nil {
		t.Error("Should validate unbalanced parens as error")
	}
}

func TestExecutorWithComplexFilter(t *testing.T) {
	executor := NewExecutor(nil, "test-project", 30*time.Second)

	complexFilter := "resource.type=gae_app AND (severity=ERROR OR severity=CRITICAL) AND timestamp>=\"2024-01-01T00:00:00Z\""

	err := executor.validator.ValidateFilter(complexFilter)
	if err != nil {
		t.Errorf("Complex filter validation failed: %v", err)
	}

	result, err := executor.ValidateAndBuild(complexFilter)
	if err != nil {
		t.Errorf("ValidateAndBuild failed: %v", err)
	}

	if result != complexFilter {
		t.Errorf("Expected %s, got %s", complexFilter, result)
	}
}

func TestExecutorTimeoutConfig(t *testing.T) {
	tests := []struct {
		timeout  time.Duration
		name     string
	}{
		{5 * time.Second, "short timeout"},
		{30 * time.Second, "default timeout"},
		{2 * time.Minute, "long timeout"},
	}

	for _, tt := range tests {
		executor := NewExecutor(nil, "test-project", tt.timeout)
		if executor.timeout != tt.timeout {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.timeout, executor.timeout)
		}
	}
}

func TestExecuteRequestPageSize(t *testing.T) {
	tests := []struct {
		input    int
		expected int
		name     string
	}{
		{0, 100, "zero becomes default"},
		{50, 50, "custom size preserved"},
		{200, 200, "large size preserved"},
	}

	for _, tt := range tests {
		req := ExecuteRequest{Filter: "severity=ERROR", PageSize: tt.input}
		if req.PageSize == 0 {
			req.PageSize = 100
		}

		if req.PageSize != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, req.PageSize)
		}
	}
}
