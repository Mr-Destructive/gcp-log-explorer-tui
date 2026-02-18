package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// Builder constructs GCP logging filter queries
type Builder struct {
	baseFilter string
	filters    []string
}

// NewBuilder creates a new query builder
func NewBuilder(baseFilter string) *Builder {
	return &Builder{
		baseFilter: baseFilter,
		filters:    []string{},
	}
}

// AddSeverity adds a severity filter
func (qb *Builder) AddSeverity(severityFilter models.SeverityFilter) *Builder {
	if severityFilter.Mode == "individual" && len(severityFilter.Levels) > 0 {
		// Individual: OR together severity levels
		var parts []string
		for _, level := range severityFilter.Levels {
			parts = append(parts, fmt.Sprintf("severity=%s", level))
		}
		qb.filters = append(qb.filters, "("+strings.Join(parts, " OR ")+")")
	} else if severityFilter.Mode == "range" && severityFilter.MinLevel != "" {
		// Range: severity >= minLevel
		qb.filters = append(qb.filters, fmt.Sprintf("severity>=%s", severityFilter.MinLevel))
	}
	return qb
}

// AddTimeRange adds a time range filter
func (qb *Builder) AddTimeRange(timeRange models.TimeRange) *Builder {
	if !timeRange.Start.IsZero() && !timeRange.End.IsZero() {
		startStr := timeRange.Start.Format(time.RFC3339)
		endStr := timeRange.End.Format(time.RFC3339)
		qb.filters = append(qb.filters, fmt.Sprintf("timestamp>=%q", startStr))
		qb.filters = append(qb.filters, fmt.Sprintf("timestamp<=%q", endStr))
	}
	return qb
}

// AddCustomFilter adds a custom filter clause
func (qb *Builder) AddCustomFilter(filter string) *Builder {
	if filter != "" {
		qb.filters = append(qb.filters, filter)
	}
	return qb
}

// AddResourceFilter filters by resource type
func (qb *Builder) AddResourceFilter(resourceType string) *Builder {
	if resourceType != "" {
		qb.filters = append(qb.filters, fmt.Sprintf("resource.type=%q", resourceType))
	}
	return qb
}

// AddLabelFilter filters by label key-value pair
func (qb *Builder) AddLabelFilter(key, value string) *Builder {
	if key != "" && value != "" {
		qb.filters = append(qb.filters, fmt.Sprintf("labels.%s=%q", key, value))
	}
	return qb
}

// Build constructs the final filter string
func (qb *Builder) Build() string {
	if qb.baseFilter == "" && len(qb.filters) == 0 {
		return ""
	}

	var parts []string
	if qb.baseFilter != "" {
		parts = append(parts, qb.baseFilter)
	}
	parts = append(parts, qb.filters...)

	return strings.Join(parts, " AND ")
}

// Validator validates query syntax
type Validator struct {
	reservedWords map[string]bool
}

// NewValidator creates a query validator
func NewValidator() *Validator {
	return &Validator{
		reservedWords: map[string]bool{
			"AND":           true,
			"OR":            true,
			"NOT":           true,
			"severity":      true,
			"timestamp":     true,
			"resource":      true,
			"labels":        true,
			"logName":       true,
			"textPayload":   true,
			"jsonPayload":   true,
			"sourceLocation": true,
		},
	}
}

// ValidateFilter checks if a filter string is valid
func (v *Validator) ValidateFilter(filter string) error {
	if filter == "" {
		return ErrEmptyFilter
	}

	// Check for balanced parentheses
	openCount := strings.Count(filter, "(")
	closeCount := strings.Count(filter, ")")
	if openCount != closeCount {
		return ErrUnbalancedParens
	}

	// Check for basic syntax patterns
	if strings.Contains(filter, "==") {
		return ErrInvalidOperator
	}

	// Check for valid comparison operators
	validOps := []string{"=", "!=", "<", ">", "<=", ">=", ":"}
	hasOp := false
	for _, op := range validOps {
		if strings.Contains(filter, op) {
			hasOp = true
			break
		}
	}

	// A filter should have at least one operator
	if !hasOp && !strings.Contains(filter, " AND ") && !strings.Contains(filter, " OR ") {
		return ErrInvalidSyntax
	}

	return nil
}

// SanitizeFilter removes potentially dangerous characters
func (v *Validator) SanitizeFilter(filter string) string {
	// Remove leading/trailing whitespace
	filter = strings.TrimSpace(filter)
	// Prevent injection by escaping quotes
	filter = strings.ReplaceAll(filter, "\"", "\"")
	return filter
}

// Error types
var (
	ErrEmptyFilter       = fmt.Errorf("filter cannot be empty")
	ErrInvalidOperator   = fmt.Errorf("invalid operator '==' (use '=' instead)")
	ErrUnbalancedParens  = fmt.Errorf("unbalanced parentheses in filter")
	ErrInvalidSyntax     = fmt.Errorf("invalid filter syntax")
)
