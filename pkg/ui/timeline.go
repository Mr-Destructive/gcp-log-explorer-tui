package ui

import (
	"fmt"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// TimelineBuilder builds log count timelines and graphs
type TimelineBuilder struct {
	bucketSize time.Duration
}

// NewTimelineBuilder creates a new timeline builder
func NewTimelineBuilder(bucketSize time.Duration) *TimelineBuilder {
	if bucketSize < time.Second {
		bucketSize = time.Second
	}
	return &TimelineBuilder{
		bucketSize: bucketSize,
	}
}

// BuildTimeline creates a timeline from logs
func (tb *TimelineBuilder) BuildTimeline(logs []models.LogEntry) []models.LogGraphPoint {
	if len(logs) == 0 {
		return []models.LogGraphPoint{}
	}

	// Create time buckets
	buckets := make(map[int64]*models.LogGraphPoint)

	for _, log := range logs {
		bucketTime := log.Timestamp.Truncate(tb.bucketSize)
		bucketKey := bucketTime.Unix()

		if _, exists := buckets[bucketKey]; !exists {
			buckets[bucketKey] = &models.LogGraphPoint{
				Timestamp: bucketTime,
				Count:     0,
				Severity:  make(map[string]int),
			}
		}

		buckets[bucketKey].Count++
		buckets[bucketKey].Severity[log.Severity]++
	}

	// Convert to sorted slice
	points := make([]models.LogGraphPoint, 0, len(buckets))
	for _, point := range buckets {
		points = append(points, *point)
	}

	// Sort by timestamp
	tb.sortPoints(points)
	return points
}

// BuildSeverityDistribution creates a distribution of logs by severity
func (tb *TimelineBuilder) BuildSeverityDistribution(logs []models.LogEntry) map[string]int {
	distribution := make(map[string]int)

	for _, log := range logs {
		distribution[log.Severity]++
	}

	return distribution
}

// RenderSparkline renders a simple text-based sparkline
func (tb *TimelineBuilder) RenderSparkline(points []models.LogGraphPoint, width int) string {
	if len(points) == 0 {
		return ""
	}

	if width < 10 {
		width = 10
	}

	// Find max count for scaling
	maxCount := 0
	for _, point := range points {
		if point.Count > maxCount {
			maxCount = point.Count
		}
	}

	if maxCount == 0 {
		return ""
	}

	// Downsample to fit width
	step := len(points) / width
	if step < 1 {
		step = 1
	}

	sparkchars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var sparkline string
	for i := 0; i < len(points); i += step {
		point := points[i]
		level := (point.Count * len(sparkchars)) / maxCount
		if level >= len(sparkchars) {
			level = len(sparkchars) - 1
		}
		sparkline += string(sparkchars[level])
	}

	return sparkline
}

// RenderDistributionBar renders a simple bar chart for severity distribution
func (tb *TimelineBuilder) RenderDistributionBar(distribution map[string]int, maxWidth int) string {
	if len(distribution) == 0 {
		return "No data"
	}

	// Find max count
	maxCount := 0
	for _, count := range distribution {
		if count > maxCount {
			maxCount = count
		}
	}

	if maxCount == 0 {
		return "No data"
	}

	var result string
	severities := []string{
		models.SeverityError,
		models.SeverityCritical,
		models.SeverityWarning,
		models.SeverityInfo,
		models.SeverityDebug,
	}

	for _, severity := range severities {
		count, exists := distribution[severity]
		if !exists || count == 0 {
			continue
		}

		// Calculate bar length
		barLength := (count * maxWidth) / maxCount
		if barLength < 1 && count > 0 {
			barLength = 1
		}

		bar := fmt.Sprintf("%-10s [%s] %d\n", severity, repeat("█", barLength), count)
		result += bar
	}

	return result
}

// GetBucketSize returns the current bucket size
func (tb *TimelineBuilder) GetBucketSize() time.Duration {
	return tb.bucketSize
}

// SetBucketSize sets the bucket size
func (tb *TimelineBuilder) SetBucketSize(size time.Duration) {
	if size < time.Second {
		size = time.Second
	}
	tb.bucketSize = size
}

// GetTimelineStats returns statistics about the timeline
func (tb *TimelineBuilder) GetTimelineStats(points []models.LogGraphPoint) map[string]interface{} {
	if len(points) == 0 {
		return make(map[string]interface{})
	}

	totalLogs := 0
	maxCount := 0
	minCount := -1
	var firstTime, lastTime time.Time

	for i, point := range points {
		totalLogs += point.Count

		if point.Count > maxCount {
			maxCount = point.Count
		}

		if minCount < 0 || point.Count < minCount {
			minCount = point.Count
		}

		if i == 0 {
			firstTime = point.Timestamp
		}
		lastTime = point.Timestamp
	}

	avgCount := totalLogs / len(points)

	return map[string]interface{}{
		"total_logs":   totalLogs,
		"max_count":    maxCount,
		"min_count":    minCount,
		"avg_count":    avgCount,
		"bucket_count": len(points),
		"first_time":   firstTime,
		"last_time":    lastTime,
		"duration":     lastTime.Sub(firstTime),
	}
}

// Helper functions

func (tb *TimelineBuilder) sortPoints(points []models.LogGraphPoint) {
	// Simple bubble sort for small datasets
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].Timestamp.After(points[j].Timestamp) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
}

func repeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
