package ui

import (
	"fmt"

	"github.com/user/log-explorer-tui/pkg/models"
)

// ClipboardManager handles copying log entries
type ClipboardManager struct {
	lastCopied string
	copyFormat string // "line", "full", "json"
}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager() *ClipboardManager {
	return &ClipboardManager{
		lastCopied: "",
		copyFormat: "line",
	}
}

// CopyEntry copies a log entry to clipboard
func (cm *ClipboardManager) CopyEntry(entry *models.LogEntry, format string) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("entry is nil")
	}

	var content string

	switch format {
	case "line":
		content = cm.formatLine(entry)
	case "full":
		content = cm.formatFull(entry)
	case "json":
		content = cm.formatJSON(entry)
	case "message":
		content = entry.Message
	default:
		return "", fmt.Errorf("invalid format: %s", format)
	}

	cm.lastCopied = content
	return content, nil
}

// formatLine formats entry as a single line
func (cm *ClipboardManager) formatLine(entry *models.LogEntry) string {
	return fmt.Sprintf("[%s] %s: %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Severity,
		entry.Message,
	)
}

// formatFull formats entry with all details
func (cm *ClipboardManager) formatFull(entry *models.LogEntry) string {
	formatter := NewLogFormatter(120, false)
	return formatter.FormatLogDetails(*entry)
}

// formatJSON formats entry as JSON-like text
func (cm *ClipboardManager) formatJSON(entry *models.LogEntry) string {
	return fmt.Sprintf(`{
  "timestamp": "%s",
  "severity": "%s",
  "message": "%s",
  "labels": %v,
  "resource": {
    "type": "%s",
    "labels": %v
  }
}`,
		entry.Timestamp.Format("2006-01-02T15:04:05Z"),
		entry.Severity,
		entry.Message,
		entry.Labels,
		entry.Resource.Type,
		entry.Resource.Labels,
	)
}

// GetLastCopied returns the last copied content
func (cm *ClipboardManager) GetLastCopied() string {
	return cm.lastCopied
}

// SetCopyFormat sets the default copy format
func (cm *ClipboardManager) SetCopyFormat(format string) error {
	validFormats := map[string]bool{
		"line":    true,
		"full":    true,
		"json":    true,
		"message": true,
	}

	if !validFormats[format] {
		return fmt.Errorf("invalid format: %s", format)
	}

	cm.copyFormat = format
	return nil
}

// GetCopyFormat returns the current copy format
func (cm *ClipboardManager) GetCopyFormat() string {
	return cm.copyFormat
}

// CopyEntryDefault copies an entry with the default format
func (cm *ClipboardManager) CopyEntryDefault(entry *models.LogEntry) (string, error) {
	return cm.CopyEntry(entry, cm.copyFormat)
}
