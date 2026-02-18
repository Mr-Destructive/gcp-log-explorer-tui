package ui

import (
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewTimelineBuilder(t *testing.T) {
	tb := NewTimelineBuilder(5 * time.Minute)

	if tb.bucketSize != 5*time.Minute {
		t.Errorf("Expected bucket size 5m, got %v", tb.bucketSize)
	}
}

func TestNewTimelineBuilderMinSize(t *testing.T) {
	tb := NewTimelineBuilder(100 * time.Millisecond)

	if tb.bucketSize != time.Second {
		t.Errorf("Bucket size should be at least 1s, got %v", tb.bucketSize)
	}
}

func TestBuildTimeline(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	now := time.Now()
	logs := []models.LogEntry{
		{Timestamp: now, Severity: "INFO", Message: "msg1"},
		{Timestamp: now.Add(30 * time.Second), Severity: "ERROR", Message: "msg2"},
		{Timestamp: now.Add(1 * time.Minute), Severity: "INFO", Message: "msg3"},
		{Timestamp: now.Add(2 * time.Minute), Severity: "WARNING", Message: "msg4"},
	}

	points := tb.BuildTimeline(logs)

	if len(points) == 0 {
		t.Error("Should have timeline points")
	}

	// Check first bucket has multiple severities
	found := false
	for _, point := range points {
		if point.Severity[models.SeverityInfo] > 0 && point.Severity[models.SeverityError] > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should have bucket with multiple severities")
	}
}

func TestBuildTimelineEmpty(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	points := tb.BuildTimeline([]models.LogEntry{})

	if len(points) != 0 {
		t.Error("Should return empty for empty logs")
	}
}

func TestBuildSeverityDistribution(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	logs := []models.LogEntry{
		{Severity: models.SeverityError},
		{Severity: models.SeverityError},
		{Severity: models.SeverityWarning},
		{Severity: models.SeverityInfo},
	}

	dist := tb.BuildSeverityDistribution(logs)

	if dist[models.SeverityError] != 2 {
		t.Errorf("Expected 2 errors, got %d", dist[models.SeverityError])
	}

	if dist[models.SeverityWarning] != 1 {
		t.Errorf("Expected 1 warning, got %d", dist[models.SeverityWarning])
	}

	if dist[models.SeverityInfo] != 1 {
		t.Errorf("Expected 1 info, got %d", dist[models.SeverityInfo])
	}
}

func TestRenderSparkline(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	now := time.Now()
	points := []models.LogGraphPoint{
		{Timestamp: now, Count: 5, Severity: make(map[string]int)},
		{Timestamp: now.Add(time.Minute), Count: 10, Severity: make(map[string]int)},
		{Timestamp: now.Add(2 * time.Minute), Count: 3, Severity: make(map[string]int)},
	}

	sparkline := tb.RenderSparkline(points, 20)

	if len(sparkline) == 0 {
		t.Error("Sparkline should not be empty")
	}

	// Should have sparkline characters
	if !containsSparklineChar(sparkline) {
		t.Error("Sparkline should contain sparkline characters")
	}
}

func TestRenderSparklineEmpty(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	sparkline := tb.RenderSparkline([]models.LogGraphPoint{}, 20)

	if sparkline != "" {
		t.Error("Empty points should produce empty sparkline")
	}
}

func TestRenderDistributionBar(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	dist := map[string]int{
		models.SeverityError:    10,
		models.SeverityWarning:  5,
		models.SeverityInfo:     2,
	}

	bar := tb.RenderDistributionBar(dist, 20)

	if bar == "No data" {
		t.Error("Should have data")
	}

	if !contains(bar, "ERROR") {
		t.Error("Bar should contain ERROR")
	}

	if !contains(bar, "WARNING") {
		t.Error("Bar should contain WARNING")
	}
}

func TestRenderDistributionBarEmpty(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	bar := tb.RenderDistributionBar(map[string]int{}, 20)

	if bar != "No data" {
		t.Error("Empty distribution should return 'No data'")
	}
}

func TestGetBucketSize(t *testing.T) {
	tb := NewTimelineBuilder(5 * time.Minute)

	if tb.GetBucketSize() != 5*time.Minute {
		t.Errorf("Expected 5m, got %v", tb.GetBucketSize())
	}
}

func TestSetBucketSize(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	tb.SetBucketSize(10 * time.Minute)

	if tb.GetBucketSize() != 10*time.Minute {
		t.Errorf("Expected 10m, got %v", tb.GetBucketSize())
	}
}

func TestSetBucketSizeMinimum(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	tb.SetBucketSize(100 * time.Millisecond)

	if tb.GetBucketSize() != time.Second {
		t.Errorf("Should enforce minimum 1s, got %v", tb.GetBucketSize())
	}
}

func TestGetTimelineStats(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	now := time.Now()
	points := []models.LogGraphPoint{
		{Timestamp: now, Count: 5},
		{Timestamp: now.Add(time.Minute), Count: 10},
		{Timestamp: now.Add(2 * time.Minute), Count: 3},
	}

	stats := tb.GetTimelineStats(points)

	if stats["total_logs"] != 18 {
		t.Errorf("Expected total 18, got %v", stats["total_logs"])
	}

	if stats["max_count"] != 10 {
		t.Errorf("Expected max 10, got %v", stats["max_count"])
	}

	if stats["min_count"] != 3 {
		t.Errorf("Expected min 3, got %v", stats["min_count"])
	}

	if stats["bucket_count"] != 3 {
		t.Errorf("Expected 3 buckets, got %v", stats["bucket_count"])
	}
}

func TestGetTimelineStatsEmpty(t *testing.T) {
	tb := NewTimelineBuilder(time.Minute)

	stats := tb.GetTimelineStats([]models.LogGraphPoint{})

	if len(stats) != 0 {
		t.Error("Empty points should return empty stats")
	}
}

// Helper functions

func containsSparklineChar(s string) bool {
	sparkchars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	for _, c := range s {
		for _, sc := range sparkchars {
			if c == sc {
				return true
			}
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if len(s)-i >= len(substr) {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
	}
	return false
}
