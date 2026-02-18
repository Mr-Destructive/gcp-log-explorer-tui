package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// Exporter handles exporting logs to various formats
type Exporter struct {
	lastExportPath string
}

// NewExporter creates a new exporter
func NewExporter() *Exporter {
	return &Exporter{
		lastExportPath: "",
	}
}

// ExportToCSV exports logs to CSV format
func (e *Exporter) ExportToCSV(logs []models.LogEntry, filepath string) error {
	if len(logs) == 0 {
		return fmt.Errorf("no logs to export")
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Severity", "Message", "Labels", "Resource Type"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	for _, log := range logs {
		labels := e.formatLabels(log.Labels)
		record := []string{
			log.Timestamp.Format(time.RFC3339),
			log.Severity,
			log.Message,
			labels,
			log.Resource.Type,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	e.lastExportPath = filepath
	return nil
}

// ExportToJSON exports logs to JSON format
func (e *Exporter) ExportToJSON(logs []models.LogEntry, filepath string, pretty bool) error {
	if len(logs) == 0 {
		return fmt.Errorf("no logs to export")
	}

	// Convert to JSON-serializable format
	jsonLogs := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		jsonLogs[i] = map[string]interface{}{
			"timestamp":   log.Timestamp.Format(time.RFC3339),
			"severity":    log.Severity,
			"message":     log.Message,
			"labels":      log.Labels,
			"resource":    log.Resource,
			"trace":       log.Trace,
			"span_id":     log.SpanID,
		}
	}

	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(jsonLogs, "", "  ")
	} else {
		data, err = json.Marshal(jsonLogs)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	e.lastExportPath = filepath
	return nil
}

// ExportToJSONL exports logs to JSONL format (one JSON per line)
func (e *Exporter) ExportToJSONL(logs []models.LogEntry, filepath string) error {
	if len(logs) == 0 {
		return fmt.Errorf("no logs to export")
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	for _, log := range logs {
		jsonLog := map[string]interface{}{
			"timestamp":   log.Timestamp.Format(time.RFC3339),
			"severity":    log.Severity,
			"message":     log.Message,
			"labels":      log.Labels,
			"resource":    log.Resource,
			"trace":       log.Trace,
			"span_id":     log.SpanID,
		}

		data, err := json.Marshal(jsonLog)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		if _, err := file.WriteString(string(data) + "\n"); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	e.lastExportPath = filepath
	return nil
}

// ExportToText exports logs to plain text format
func (e *Exporter) ExportToText(logs []models.LogEntry, filepath string) error {
	if len(logs) == 0 {
		return fmt.Errorf("no logs to export")
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	formatter := NewLogFormatter(120, false)

	for _, log := range logs {
		details := formatter.FormatLogDetails(log)
		if _, err := file.WriteString(details + "\n"); err != nil {
			return fmt.Errorf("failed to write log: %w", err)
		}
	}

	e.lastExportPath = filepath
	return nil
}

// GetLastExportPath returns the path of the last export
func (e *Exporter) GetLastExportPath() string {
	return e.lastExportPath
}

// FileExists checks if a file exists
func (e *Exporter) FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

// GetDefaultFileName generates a default filename for export
func (e *Exporter) GetDefaultFileName(format string) string {
	timestamp := time.Now().Format("20060102_150405")
	ext := "txt"

	switch format {
	case "csv":
		ext = "csv"
	case "json":
		ext = "json"
	case "jsonl":
		ext = "jsonl"
	case "text":
		ext = "txt"
	}

	return fmt.Sprintf("logs_%s.%s", timestamp, ext)
}

// formatLabels formats labels as a string
func (e *Exporter) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(parts, ";")
}

// EstimateSize estimates the export file size
func (e *Exporter) EstimateSize(logs []models.LogEntry, format string) int {
	if len(logs) == 0 {
		return 0
	}

	avgLogSize := 0
	for _, log := range logs {
		avgLogSize += len(log.Message) + len(log.Severity) + 100
	}
	avgLogSize /= len(logs)

	switch format {
	case "csv":
		return (avgLogSize + 50) * len(logs)
	case "json", "jsonl":
		return (avgLogSize + 150) * len(logs)
	case "text":
		return (avgLogSize + 500) * len(logs)
	default:
		return 0
	}
}
