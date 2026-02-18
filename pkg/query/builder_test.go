package query

import (
	"strings"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("severity=ERROR")
	if builder.baseFilter != "severity=ERROR" {
		t.Errorf("Expected baseFilter 'severity=ERROR', got %s", builder.baseFilter)
	}
}

func TestBuilderAddSeverityIndividual(t *testing.T) {
	builder := NewBuilder("")
	filter := models.SeverityFilter{
		Levels: []string{models.SeverityError, models.SeverityCritical},
		Mode:   "individual",
	}

	builder.AddSeverity(filter)
	result := builder.Build()

	if !strings.Contains(result, "severity=ERROR") {
		t.Error("Result should contain 'severity=ERROR'")
	}

	if !strings.Contains(result, "severity=CRITICAL") {
		t.Error("Result should contain 'severity=CRITICAL'")
	}

	if !strings.Contains(result, " OR ") {
		t.Error("Result should use OR between severity levels")
	}
}

func TestBuilderAddSeverityRange(t *testing.T) {
	builder := NewBuilder("")
	filter := models.SeverityFilter{
		MinLevel: models.SeverityWarning,
		Mode:     "range",
	}

	builder.AddSeverity(filter)
	result := builder.Build()

	if !strings.Contains(result, "severity>=WARNING") {
		t.Errorf("Expected 'severity>=WARNING', got %s", result)
	}
}

func TestBuilderAddTimeRange(t *testing.T) {
	builder := NewBuilder("")
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	tr := models.TimeRange{
		Start: oneHourAgo,
		End:   now,
	}

	builder.AddTimeRange(tr)
	result := builder.Build()

	if !strings.Contains(result, "timestamp>=") {
		t.Error("Result should contain timestamp>= filter")
	}

	if !strings.Contains(result, "timestamp<=") {
		t.Error("Result should contain timestamp<= filter")
	}
}

func TestBuilderAddCustomFilter(t *testing.T) {
	builder := NewBuilder("")
	builder.AddCustomFilter("resource.type=gae_app")
	result := builder.Build()

	if result != "resource.type=gae_app" {
		t.Errorf("Expected 'resource.type=gae_app', got %s", result)
	}
}

func TestBuilderAddResourceFilter(t *testing.T) {
	builder := NewBuilder("")
	builder.AddResourceFilter("cloud_function")
	result := builder.Build()

	if !strings.Contains(result, `resource.type="cloud_function"`) {
		t.Errorf("Expected resource.type filter, got %s", result)
	}
}

func TestBuilderAddLabelFilter(t *testing.T) {
	builder := NewBuilder("")
	builder.AddLabelFilter("environment", "production")
	result := builder.Build()

	if !strings.Contains(result, `labels.environment="production"`) {
		t.Errorf("Expected label filter, got %s", result)
	}
}

func TestBuilderChaining(t *testing.T) {
	builder := NewBuilder("").
		AddCustomFilter("severity=ERROR").
		AddResourceFilter("gae_app").
		AddLabelFilter("env", "prod")

	result := builder.Build()

	if !strings.Contains(result, "severity=ERROR") {
		t.Error("Result should contain severity filter")
	}

	if !strings.Contains(result, "resource.type") {
		t.Error("Result should contain resource filter")
	}

	if !strings.Contains(result, "labels.env") {
		t.Error("Result should contain label filter")
	}

	if !strings.Contains(result, " AND ") {
		t.Error("Filters should be joined with AND")
	}
}

func TestBuilderBuildEmpty(t *testing.T) {
	builder := NewBuilder("")
	result := builder.Build()

	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestBuilderBuildWithBase(t *testing.T) {
	builder := NewBuilder("resource.type=gae_app")
	builder.AddCustomFilter("severity>=ERROR")
	result := builder.Build()

	if !strings.HasPrefix(result, "resource.type=gae_app") {
		t.Errorf("Expected result to start with base filter, got %s", result)
	}

	if !strings.Contains(result, "severity>=ERROR") {
		t.Error("Result should contain additional filter")
	}
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Error("Validator should not be nil")
	}

	if len(v.reservedWords) == 0 {
		t.Error("Validator should have reserved words")
	}
}

func TestValidatorValidateFilterEmpty(t *testing.T) {
	v := NewValidator()
	err := v.ValidateFilter("")

	if err != ErrEmptyFilter {
		t.Errorf("Expected ErrEmptyFilter, got %v", err)
	}
}

func TestValidatorValidateFilterUnbalancedParens(t *testing.T) {
	v := NewValidator()
	tests := []string{
		"(severity=ERROR",
		"severity=ERROR)",
		"((severity=ERROR)",
	}

	for _, filter := range tests {
		err := v.ValidateFilter(filter)
		if err != ErrUnbalancedParens {
			t.Errorf("Expected ErrUnbalancedParens for %s, got %v", filter, err)
		}
	}
}

func TestValidatorValidateFilterInvalidOperator(t *testing.T) {
	v := NewValidator()
	err := v.ValidateFilter("severity==ERROR")

	if err != ErrInvalidOperator {
		t.Errorf("Expected ErrInvalidOperator, got %v", err)
	}
}

func TestValidatorValidateFilterValid(t *testing.T) {
	v := NewValidator()
	validFilters := []string{
		"severity=ERROR",
		"resource.type=gae_app",
		"timestamp>=2024-01-01T00:00:00Z",
		"(severity=ERROR OR severity=WARNING)",
		"resource.type=gae_app AND severity>=ERROR",
	}

	for _, filter := range validFilters {
		err := v.ValidateFilter(filter)
		if err != nil {
			t.Errorf("Expected nil for valid filter %s, got %v", filter, err)
		}
	}
}

func TestValidatorSanitizeFilter(t *testing.T) {
	v := NewValidator()
	tests := []struct {
		input    string
		expected string
	}{
		{"  severity=ERROR  ", "severity=ERROR"},
		{"severity=ERROR", "severity=ERROR"},
		{"severity=\"ERROR\"", "severity=\"ERROR\""},
	}

	for _, tt := range tests {
		result := v.SanitizeFilter(tt.input)
		if result != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, result)
		}
	}
}

func TestBuilderComplexQuery(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	builder := NewBuilder("").
		AddCustomFilter("resource.type=gae_app").
		AddTimeRange(models.TimeRange{Start: oneHourAgo, End: now}).
		AddSeverity(models.SeverityFilter{
			Levels: []string{models.SeverityError, models.SeverityCritical},
			Mode:   "individual",
		})

	result := builder.Build()

	if !strings.Contains(result, "resource.type=gae_app") {
		t.Error("Result should contain resource filter")
	}

	if !strings.Contains(result, "timestamp>=") {
		t.Error("Result should contain start timestamp")
	}

	if !strings.Contains(result, "timestamp<=") {
		t.Error("Result should contain end timestamp")
	}

	if !strings.Contains(result, "severity=ERROR") || !strings.Contains(result, "severity=CRITICAL") {
		t.Error("Result should contain severity filters")
	}
}
