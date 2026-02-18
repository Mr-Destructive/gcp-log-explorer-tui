package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewClipboardManager(t *testing.T) {
	cm := NewClipboardManager()

	if cm.lastCopied != "" {
		t.Error("lastCopied should be empty initially")
	}

	if cm.copyFormat != "line" {
		t.Errorf("Expected default format 'line', got %s", cm.copyFormat)
	}
}

func TestCopyEntryLine(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Test error message",
	}

	content, err := cm.CopyEntry(entry, "line")
	if err != nil {
		t.Errorf("CopyEntry failed: %v", err)
	}

	if !strings.Contains(content, "2024-01-01") {
		t.Error("Content should contain date")
	}

	if !strings.Contains(content, "ERROR") {
		t.Error("Content should contain severity")
	}

	if !strings.Contains(content, "Test error message") {
		t.Error("Content should contain message")
	}
}

func TestCopyEntryFull(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "ERROR",
		Message:   "Test error",
		Labels: map[string]string{
			"env": "prod",
		},
	}

	content, err := cm.CopyEntry(entry, "full")
	if err != nil {
		t.Errorf("CopyEntry failed: %v", err)
	}

	if !strings.Contains(content, "Timestamp:") {
		t.Error("Full format should contain timestamp label")
	}

	if !strings.Contains(content, "prod") {
		t.Error("Full format should contain labels")
	}
}

func TestCopyEntryJSON(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "WARNING",
		Message:   "Test warning",
		Resource: models.Resource{
			Type: "gae_app",
		},
	}

	content, err := cm.CopyEntry(entry, "json")
	if err != nil {
		t.Errorf("CopyEntry failed: %v", err)
	}

	if !strings.Contains(content, "timestamp") {
		t.Error("JSON should contain timestamp field")
	}

	if !strings.Contains(content, "severity") {
		t.Error("JSON should contain severity field")
	}

	if !strings.Contains(content, "gae_app") {
		t.Error("JSON should contain resource type")
	}
}

func TestCopyEntryMessage(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{
		Message: "Just the message",
	}

	content, err := cm.CopyEntry(entry, "message")
	if err != nil {
		t.Errorf("CopyEntry failed: %v", err)
	}

	if content != "Just the message" {
		t.Errorf("Expected just message, got %s", content)
	}
}

func TestCopyEntryInvalidFormat(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{}

	_, err := cm.CopyEntry(entry, "invalid")
	if err == nil {
		t.Error("Should error on invalid format")
	}
}

func TestCopyEntryNil(t *testing.T) {
	cm := NewClipboardManager()

	_, err := cm.CopyEntry(nil, "line")
	if err == nil {
		t.Error("Should error on nil entry")
	}
}

func TestGetLastCopied(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{Message: "Test"}

	cm.CopyEntry(entry, "message")
	lastCopied := cm.GetLastCopied()

	if lastCopied != "Test" {
		t.Errorf("Expected 'Test', got %s", lastCopied)
	}
}

func TestSetCopyFormat(t *testing.T) {
	cm := NewClipboardManager()

	validFormats := []string{"line", "full", "json", "message"}
	for _, format := range validFormats {
		err := cm.SetCopyFormat(format)
		if err != nil {
			t.Errorf("SetCopyFormat failed for %s: %v", format, err)
		}

		if cm.copyFormat != format {
			t.Errorf("Expected format %s, got %s", format, cm.copyFormat)
		}
	}

	// Invalid format
	err := cm.SetCopyFormat("invalid")
	if err == nil {
		t.Error("Should error on invalid format")
	}
}

func TestGetCopyFormat(t *testing.T) {
	cm := NewClipboardManager()

	cm.SetCopyFormat("json")
	if cm.GetCopyFormat() != "json" {
		t.Errorf("Expected 'json', got %s", cm.GetCopyFormat())
	}
}

func TestCopyEntryDefault(t *testing.T) {
	cm := NewClipboardManager()
	entry := &models.LogEntry{
		Timestamp: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		Severity:  "INFO",
		Message:   "Default format test",
	}

	cm.SetCopyFormat("line")
	content, err := cm.CopyEntryDefault(entry)
	if err != nil {
		t.Errorf("CopyEntryDefault failed: %v", err)
	}

	if !strings.Contains(content, "2024-01-01") {
		t.Error("Should use line format by default")
	}
}

func TestCopyMultipleEntries(t *testing.T) {
	cm := NewClipboardManager()

	entries := []*models.LogEntry{
		{Message: "Message 1"},
		{Message: "Message 2"},
		{Message: "Message 3"},
	}

	for _, entry := range entries {
		cm.CopyEntry(entry, "message")
	}

	lastCopied := cm.GetLastCopied()
	if lastCopied != "Message 3" {
		t.Errorf("Expected last copied to be 'Message 3', got %s", lastCopied)
	}
}
