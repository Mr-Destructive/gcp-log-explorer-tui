package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func createTestLogs(count int) []models.LogEntry {
	logs := make([]models.LogEntry, count)
	for i := 0; i < count; i++ {
		logs[i] = models.LogEntry{
			ID:        "id-" + string(rune(48+i)),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Severity:  "INFO",
			Message:   "Test message " + string(rune(48+i)),
			Labels: map[string]string{
				"env": "test",
			},
			Resource: models.Resource{
				Type: "gae_app",
			},
		}
	}
	return logs
}

func TestNewExporter(t *testing.T) {
	exp := NewExporter()

	if exp.lastExportPath != "" {
		t.Error("lastExportPath should be empty initially")
	}
}

func TestExportToCSV(t *testing.T) {
	tmpFile := "test_export.csv"
	defer os.Remove(tmpFile)

	exp := NewExporter()
	logs := createTestLogs(3)

	err := exp.ExportToCSV(logs, tmpFile)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	if !exp.FileExists(tmpFile) {
		t.Error("File should exist after export")
	}

	if exp.GetLastExportPath() != tmpFile {
		t.Error("Last export path should be set")
	}

	// Verify content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Timestamp") {
		t.Error("CSV should contain header")
	}

	if !strings.Contains(contentStr, "INFO") {
		t.Error("CSV should contain severity")
	}
}

func TestExportToCSVEmpty(t *testing.T) {
	exp := NewExporter()
	logs := []models.LogEntry{}

	err := exp.ExportToCSV(logs, "test.csv")
	if err == nil {
		t.Error("Should error on empty logs")
	}
}

func TestExportToJSON(t *testing.T) {
	tmpFile := "test_export.json"
	defer os.Remove(tmpFile)

	exp := NewExporter()
	logs := createTestLogs(2)

	err := exp.ExportToJSON(logs, tmpFile, true)
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "timestamp") {
		t.Error("JSON should contain timestamp field")
	}

	if !strings.Contains(contentStr, "INFO") {
		t.Error("JSON should contain severity")
	}
}

func TestExportToJSONL(t *testing.T) {
	tmpFile := "test_export.jsonl"
	defer os.Remove(tmpFile)

	exp := NewExporter()
	logs := createTestLogs(2)

	err := exp.ExportToJSONL(logs, tmpFile)
	if err != nil {
		t.Fatalf("ExportToJSONL failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestExportToText(t *testing.T) {
	tmpFile := "test_export.txt"
	defer os.Remove(tmpFile)

	exp := NewExporter()
	logs := createTestLogs(1)

	err := exp.ExportToText(logs, tmpFile)
	if err != nil {
		t.Fatalf("ExportToText failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "LOG ENTRY DETAILS") {
		t.Error("Text export should contain header")
	}
}

func TestGetDefaultFileName(t *testing.T) {
	exp := NewExporter()

	tests := []struct {
		format   string
		expected string
	}{
		{"csv", ".csv"},
		{"json", ".json"},
		{"jsonl", ".jsonl"},
		{"text", ".txt"},
	}

	for _, tt := range tests {
		filename := exp.GetDefaultFileName(tt.format)
		if !strings.HasPrefix(filename, "logs_") {
			t.Errorf("Filename should start with 'logs_'")
		}
		if !strings.HasSuffix(filename, tt.expected) {
			t.Errorf("Expected suffix %s, got %s", tt.expected, filename)
		}
	}
}

func TestFileExists(t *testing.T) {
	exp := NewExporter()

	// Non-existent file
	if exp.FileExists("nonexistent_file_xyz.txt") {
		t.Error("Should return false for non-existent file")
	}

	// Create temp file
	tmpFile := "test_exists.txt"
	os.Create(tmpFile)
	defer os.Remove(tmpFile)

	if !exp.FileExists(tmpFile) {
		t.Error("Should return true for existing file")
	}
}

func TestEstimateSize(t *testing.T) {
	exp := NewExporter()
	logs := createTestLogs(100)

	csvSize := exp.EstimateSize(logs, "csv")
	if csvSize <= 0 {
		t.Error("Estimated size should be positive")
	}

	jsonSize := exp.EstimateSize(logs, "json")
	if jsonSize <= csvSize {
		t.Error("JSON size should be larger than CSV")
	}

	textSize := exp.EstimateSize(logs, "text")
	if textSize <= jsonSize {
		t.Error("Text size should be larger than JSON")
	}
}

func TestEstimateSizeEmpty(t *testing.T) {
	exp := NewExporter()
	logs := []models.LogEntry{}

	size := exp.EstimateSize(logs, "csv")
	if size != 0 {
		t.Errorf("Expected 0 for empty logs, got %d", size)
	}
}
