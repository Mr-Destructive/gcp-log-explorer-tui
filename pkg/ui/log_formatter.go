package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// LogFormatter handles formatting log entries for display
type LogFormatter struct {
	maxWidth     int
	timeFormat   string
	useColor     bool
}

// NewLogFormatter creates a new log formatter
func NewLogFormatter(maxWidth int, useColor bool) *LogFormatter {
	return &LogFormatter{
		maxWidth:   maxWidth,
		timeFormat: "15:04:05",
		useColor:   useColor,
	}
}

// FormatLogLine formats a log entry as a single line for list display
func (lf *LogFormatter) FormatLogLine(entry models.LogEntry, maxLen int) string {
	// Format: [TIME] [SEVERITY] MESSAGE
	timestamp := entry.Timestamp.Format(lf.timeFormat)
	severity := padRight(entry.Severity, 8)
	message := truncate(entry.Message, maxLen-30)

	return fmt.Sprintf("[%s] %s %s", timestamp, severity, message)
}

// FormatLogDetails formats a log entry with full details
func (lf *LogFormatter) FormatLogDetails(entry models.LogEntry) string {
	var sb strings.Builder

	// Header
	sb.WriteString("═══════════════════════════════════════════════════════\n")
	sb.WriteString("LOG ENTRY DETAILS\n")
	sb.WriteString("═══════════════════════════════════════════════════════\n\n")

	// Timestamp
	sb.WriteString(fmt.Sprintf("Timestamp:  %s\n", entry.Timestamp.Format(time.RFC3339)))

	// Severity
	sb.WriteString(fmt.Sprintf("Severity:   %s\n", entry.Severity))

	// Message
	sb.WriteString(fmt.Sprintf("\nMessage:\n%s\n", entry.Message))

	// Labels
	if len(entry.Labels) > 0 {
		sb.WriteString("\nLabels:\n")
		for key, value := range entry.Labels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// Resource
	if entry.Resource.Type != "" {
		sb.WriteString(fmt.Sprintf("\nResource Type: %s\n", entry.Resource.Type))
		if len(entry.Resource.Labels) > 0 {
			sb.WriteString("Resource Labels:\n")
			for key, value := range entry.Resource.Labels {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}
	}

	// Source Location
	if entry.SourceLocation != nil {
		sb.WriteString(fmt.Sprintf("\nSource Location:\n"))
		sb.WriteString(fmt.Sprintf("  File: %s\n", entry.SourceLocation.File))
		if entry.SourceLocation.Line > 0 {
			sb.WriteString(fmt.Sprintf("  Line: %d\n", entry.SourceLocation.Line))
		}
		if entry.SourceLocation.Function != "" {
			sb.WriteString(fmt.Sprintf("  Function: %s\n", entry.SourceLocation.Function))
		}
	}

	// Trace
	if entry.Trace != "" {
		sb.WriteString(fmt.Sprintf("\nTrace ID: %s\n", entry.Trace))
	}

	// Span ID
	if entry.SpanID != "" {
		sb.WriteString(fmt.Sprintf("Span ID: %s\n", entry.SpanID))
	}

	// JSON Payload
	if entry.JSONPayload != nil {
		sb.WriteString(fmt.Sprintf("\nJSON Payload:\n%v\n", entry.JSONPayload))
	}

	// Text Payload
	if entry.TextPayload != "" {
		sb.WriteString(fmt.Sprintf("\nText Payload:\n%s\n", entry.TextPayload))
	}

	sb.WriteString("\n═══════════════════════════════════════════════════════\n")

	return sb.String()
}

// FormatCompact formats a log entry in compact form (for side panel)
func (lf *LogFormatter) FormatCompact(entry models.LogEntry) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Time: %s\n", entry.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", entry.Severity))
	sb.WriteString(fmt.Sprintf("Message: %s\n", entry.Message))

	if len(entry.Labels) > 0 {
		sb.WriteString("Labels: ")
		var labels []string
		for k, v := range entry.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		sb.WriteString(strings.Join(labels, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncate truncates a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// padRight pads a string to the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// SetTimeFormat sets the time format for display
func (lf *LogFormatter) SetTimeFormat(format string) {
	lf.timeFormat = format
}

// SetMaxWidth sets the maximum line width
func (lf *LogFormatter) SetMaxWidth(width int) {
	lf.maxWidth = width
}

// HighlightMessage highlights a keyword in a message
func (lf *LogFormatter) HighlightMessage(message, keyword string) string {
	if keyword == "" {
		return message
	}

	// Simple highlight by surrounding with markers
	// In actual implementation, would use ANSI codes
	return strings.ReplaceAll(message, keyword, fmt.Sprintf("[%s]", keyword))
}
