package ui

import (
	"fmt"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// LoadDemoData populates app state with sample logs for testing/demo purposes
func LoadDemoData(appState *models.AppState) {
	now := time.Now()

	sampleLogs := []models.LogEntry{
		{
			ID:        "log-001",
			Timestamp: now.Add(-50 * time.Minute),
			Severity:  "INFO",
			Message:   "Application started successfully",
			Labels:    map[string]string{"env": "prod", "version": "1.2.3"},
		},
		{
			ID:        "log-002",
			Timestamp: now.Add(-45 * time.Minute),
			Severity:  "INFO",
			Message:   "Database connection established",
			Labels:    map[string]string{"service": "api", "region": "us-west"},
		},
		{
			ID:        "log-003",
			Timestamp: now.Add(-40 * time.Minute),
			Severity:  "WARNING",
			Message:   "High memory usage detected (85% utilized)",
			Labels:    map[string]string{"alert": "true"},
		},
		{
			ID:        "log-004",
			Timestamp: now.Add(-35 * time.Minute),
			Severity:  "ERROR",
			Message:   "Failed to connect to cache server: timeout",
			Labels:    map[string]string{"component": "cache", "retry": "3"},
		},
		{
			ID:        "log-005",
			Timestamp: now.Add(-30 * time.Minute),
			Severity:  "ERROR",
			Message:   "Request processing failed: invalid authorization header",
			Labels:    map[string]string{"user_id": "user-123"},
		},
		{
			ID:        "log-006",
			Timestamp: now.Add(-25 * time.Minute),
			Severity:  "INFO",
			Message:   "Request completed successfully in 245ms",
			Labels:    map[string]string{"method": "GET", "status": "200"},
		},
		{
			ID:        "log-007",
			Timestamp: now.Add(-20 * time.Minute),
			Severity:  "DEBUG",
			Message:   "Query execution: SELECT * FROM users WHERE active=true (12 rows)",
			Labels:    map[string]string{"query": "indexed"},
		},
		{
			ID:        "log-008",
			Timestamp: now.Add(-15 * time.Minute),
			Severity:  "CRITICAL",
			Message:   "Service health check failed - marking unhealthy",
			Labels:    map[string]string{"health_check": "database"},
		},
		{
			ID:        "log-009",
			Timestamp: now.Add(-10 * time.Minute),
			Severity:  "INFO",
			Message:   "Cache warmed up, 1250 entries loaded",
			Labels:    map[string]string{"component": "cache", "duration": "3.2s"},
		},
		{
			ID:        "log-010",
			Timestamp: now.Add(-5 * time.Minute),
			Severity:  "WARNING",
			Message:   "API response time degradation: 5.2s (expected <1s)",
			Labels:    map[string]string{"endpoint": "/api/v1/users", "percentile": "p95"},
		},
		{
			ID:        "log-011",
			Timestamp: now,
			Severity:  "INFO",
			Message:   "Scheduled backup completed successfully",
			Labels:    map[string]string{"backup": "daily", "duration": "42s"},
		},
		{
			ID:        "log-012",
			Timestamp: now.Add(5 * time.Minute),
			Severity:  "ERROR",
			Message:   "Database connection pool exhausted (50/50 connections)",
			Labels:    map[string]string{"pool": "primary", "threshold": "exceeded"},
		},
		{
			ID:        "log-013",
			Timestamp: now.Add(10 * time.Minute),
			Severity:  "CRITICAL",
			Message:   "Memory limit exceeded, initiating graceful shutdown",
			Labels:    map[string]string{"memory_used": "7.8GB", "limit": "8GB"},
		},
		{
			ID:        "log-014",
			Timestamp: now.Add(15 * time.Minute),
			Severity:  "INFO",
			Message:   "Service restarted after maintenance",
			Labels:    map[string]string{"version": "1.2.4", "uptime": "0s"},
		},
		{
			ID:        "log-015",
			Timestamp: now.Add(20 * time.Minute),
			Severity:  "WARNING",
			Message:   "Unusual traffic pattern detected: 5x normal request rate",
			Labels:    map[string]string{"alert_level": "medium", "rps": "2500"},
		},
	}

	// Add 50 more logs to test scrolling
	for i := 16; i <= 65; i++ {
		severity := []string{"INFO", "WARNING", "ERROR", "DEBUG", "CRITICAL"}[i%5]
		sampleLogs = append(sampleLogs, models.LogEntry{
			ID:        fmt.Sprintf("log-%03d", i),
			Timestamp: now.Add(-time.Duration((66-i)*5) * time.Minute),
			Severity:  severity,
			Message:   fmt.Sprintf("Log message %d with some content", i),
			Labels:    map[string]string{"index": fmt.Sprintf("%d", i)},
		})
	}

	appState.LogListState.Logs = sampleLogs
}
