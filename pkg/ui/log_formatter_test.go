package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewLogFormatter(t *testing.T) {
	lf := NewLogFormatter(120, true)

	if lf.maxWidth != 120 {
		t.Errorf("Expected maxWidth 120, got %d", lf.maxWidth)
	}

	if !lf.useColor {
		t.Error("Expected useColor true")
	}
}

func TestFormatLogLine(t *testing.T) {
	lf := NewLogFormatter(120, false)
	entry := models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Database connection failed",
	}

	line := lf.FormatLogLine(entry, 80)

	if !strings.Contains(line, "12:30:45") {
		t.Error("Line should contain timestamp")
	}

	if !strings.Contains(line, "ERROR") {
		t.Error("Line should contain severity")
	}

	if !strings.Contains(line, "Database") {
		t.Error("Line should contain message")
	}
}

func TestFormatLogLineMessageTruncation(t *testing.T) {
	lf := NewLogFormatter(120, false)
	entry := models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "INFO",
		Message:   strings.Repeat("A", 200),
	}

	line := lf.FormatLogLine(entry, 50)

	if len(line) > 50 {
		t.Errorf("Line should be truncated to ~50 chars, got %d", len(line))
	}

	if !strings.Contains(line, "...") {
		t.Error("Truncated line should have ellipsis")
	}
}

func TestFormatLogDetails(t *testing.T) {
	entry := models.LogEntry{
		ID:        "test-id",
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Test error message",
		Labels: map[string]string{
			"env": "production",
		},
		Resource: models.Resource{
			Type: "gae_app",
			Labels: map[string]string{
				"service_name": "api",
			},
		},
	}

	lf := NewLogFormatter(120, false)
	details := lf.FormatLogDetails(entry)

	if !strings.Contains(details, "Timestamp:") {
		t.Error("Details should contain timestamp label")
	}

	if !strings.Contains(details, entry.Message) {
		t.Error("Details should contain message")
	}

	if !strings.Contains(details, "ERROR") {
		t.Error("Details should contain severity")
	}

	if !strings.Contains(details, "production") {
		t.Error("Details should contain labels")
	}

	if !strings.Contains(details, "gae_app") {
		t.Error("Details should contain resource type")
	}
}

func TestFormatCompact(t *testing.T) {
	entry := models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "WARNING",
		Message:   "Low memory warning",
		Labels: map[string]string{
			"service": "worker",
		},
	}

	lf := NewLogFormatter(120, false)
	compact := lf.FormatCompact(entry)

	if !strings.Contains(compact, "Time:") {
		t.Error("Compact should contain time")
	}

	if !strings.Contains(compact, "Severity:") {
		t.Error("Compact should contain severity")
	}

	if !strings.Contains(compact, "Message:") {
		t.Error("Compact should contain message")
	}

	if !strings.Contains(compact, "worker") {
		t.Error("Compact should contain labels")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"abc", 2, "..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q", tt.input, tt.maxLen, tt.expected, result)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		minLen int
	}{
		{"hello", 10, 10},
		{"test", 4, 4},
		{"a", 5, 5},
	}

	for _, tt := range tests {
		result := padRight(tt.input, tt.width)
		if len(result) < tt.minLen {
			t.Errorf("padRight(%q, %d): result too short, got %d", tt.input, tt.width, len(result))
		}
	}
}

func TestSetTimeFormat(t *testing.T) {
	lf := NewLogFormatter(120, false)
	lf.SetTimeFormat("2006-01-02")

	if lf.timeFormat != "2006-01-02" {
		t.Errorf("Expected timeFormat to be set")
	}
}

func TestSetMaxWidth(t *testing.T) {
	lf := NewLogFormatter(120, false)
	lf.SetMaxWidth(200)

	if lf.maxWidth != 200 {
		t.Errorf("Expected maxWidth 200, got %d", lf.maxWidth)
	}
}

func TestHighlightMessage(t *testing.T) {
	lf := NewLogFormatter(120, false)

	message := "This is an error message"
	highlighted := lf.HighlightMessage(message, "error")

	if !strings.Contains(highlighted, "[error]") {
		t.Error("Message should be highlighted")
	}
}

func TestHighlightMessageEmpty(t *testing.T) {
	lf := NewLogFormatter(120, false)

	message := "This is a message"
	highlighted := lf.HighlightMessage(message, "")

	if highlighted != message {
		t.Error("Empty keyword should not change message")
	}
}

func TestFormatLogDetailsWithSourceLocation(t *testing.T) {
	entry := models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Test error",
		SourceLocation: &models.SourceLocation{
			File:     "main.go",
			Line:     42,
			Function: "main",
		},
	}

	lf := NewLogFormatter(120, false)
	details := lf.FormatLogDetails(entry)

	if !strings.Contains(details, "main.go") {
		t.Error("Details should contain source file")
	}

	if !strings.Contains(details, "42") {
		t.Error("Details should contain line number")
	}

	if !strings.Contains(details, "main") {
		t.Error("Details should contain function name")
	}
}

func TestFormatLogDetailsWithTrace(t *testing.T) {
	entry := models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Test error",
		Trace:     "projects/my-project/traces/abcd1234",
		SpanID:    "span-123",
	}

	lf := NewLogFormatter(120, false)
	details := lf.FormatLogDetails(entry)

	if !strings.Contains(details, "abcd1234") {
		t.Error("Details should contain trace ID")
	}

	if !strings.Contains(details, "span-123") {
		t.Error("Details should contain span ID")
	}
}
