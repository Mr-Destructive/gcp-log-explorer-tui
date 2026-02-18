package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
	"github.com/user/log-explorer-tui/pkg/query"
)

// StreamManager manages streaming/auto-refresh of logs
type StreamManager struct {
	enabled         bool
	interval        time.Duration
	lastRefresh     time.Time
	refreshCallback func() error
	newLogsCount    int
	isRunning       bool
	stopChan        chan struct{}
}

// NewStreamManager creates a new stream manager
func NewStreamManager(interval time.Duration) *StreamManager {
	if interval < time.Second {
		interval = time.Second
	}

	return &StreamManager{
		enabled:      false,
		interval:     interval,
		lastRefresh:  time.Now(),
		newLogsCount: 0,
		isRunning:    false,
		stopChan:     make(chan struct{}),
	}
}

// Enable enables streaming mode
func (sm *StreamManager) Enable() error {
	if sm.enabled {
		return fmt.Errorf("streaming already enabled")
	}

	sm.enabled = true
	sm.newLogsCount = 0
	sm.lastRefresh = time.Now()

	return nil
}

// Disable disables streaming mode
func (sm *StreamManager) Disable() error {
	if !sm.enabled {
		return fmt.Errorf("streaming not enabled")
	}

	sm.enabled = false

	if sm.isRunning {
		sm.StopStreaming()
	}

	return nil
}

// IsEnabled returns whether streaming is enabled
func (sm *StreamManager) IsEnabled() bool {
	return sm.enabled
}

// SetRefreshCallback sets the callback function for refresh
func (sm *StreamManager) SetRefreshCallback(callback func() error) {
	sm.refreshCallback = callback
}

// StartStreaming starts the streaming loop
func (sm *StreamManager) StartStreaming(ctx context.Context) error {
	if !sm.enabled {
		return fmt.Errorf("streaming not enabled")
	}

	if sm.isRunning {
		return fmt.Errorf("streaming already running")
	}

	sm.isRunning = true
	sm.stopChan = make(chan struct{})

	go sm.streamLoop(ctx)

	return nil
}

// StopStreaming stops the streaming loop
func (sm *StreamManager) StopStreaming() {
	if sm.isRunning {
		sm.isRunning = false
		close(sm.stopChan)
	}
}

// streamLoop runs the streaming refresh loop
func (sm *StreamManager) streamLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sm.isRunning = false
			return
		case <-sm.stopChan:
			sm.isRunning = false
			return
		case <-ticker.C:
			if sm.refreshCallback != nil {
				if err := sm.refreshCallback(); err != nil {
					// Log error but continue
					_ = err
				}
				sm.lastRefresh = time.Now()
			}
		}
	}
}

// GetLastRefreshTime returns the time of last refresh
func (sm *StreamManager) GetLastRefreshTime() time.Time {
	return sm.lastRefresh
}

// GetTimeSinceLastRefresh returns duration since last refresh
func (sm *StreamManager) GetTimeSinceLastRefresh() time.Duration {
	return time.Since(sm.lastRefresh)
}

// SetInterval sets the refresh interval
func (sm *StreamManager) SetInterval(interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be at least 1 second")
	}

	if sm.isRunning {
		sm.StopStreaming()
		sm.interval = interval
		// Restart with new interval
		return nil
	}

	sm.interval = interval
	return nil
}

// GetInterval returns the current interval
func (sm *StreamManager) GetInterval() time.Duration {
	return sm.interval
}

// IncrementNewLogsCount increments the new logs count
func (sm *StreamManager) IncrementNewLogsCount(count int) {
	sm.newLogsCount += count
}

// GetNewLogsCount returns the count of new logs
func (sm *StreamManager) GetNewLogsCount() int {
	return sm.newLogsCount
}

// ResetNewLogsCount resets the new logs count
func (sm *StreamManager) ResetNewLogsCount() {
	sm.newLogsCount = 0
}

// GetStatus returns the current streaming status
func (sm *StreamManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":      sm.enabled,
		"running":      sm.isRunning,
		"interval":     sm.interval.String(),
		"last_refresh": sm.lastRefresh,
		"new_logs":     sm.newLogsCount,
	}
}

// ApplyToStreamState applies stream manager state to model
func (sm *StreamManager) ApplyToStreamState(ss *models.StreamState) {
	ss.Enabled = sm.enabled
	ss.LastFetchTime = sm.lastRefresh
	ss.RefreshInterval = sm.interval
	ss.NewLogsCount = sm.newLogsCount
}

// UpdateFromStreamState updates from model state
func (sm *StreamManager) UpdateFromStreamState(ss models.StreamState) {
	sm.enabled = ss.Enabled
	sm.lastRefresh = ss.LastFetchTime
	sm.interval = ss.RefreshInterval
	sm.newLogsCount = ss.NewLogsCount
}

// GetNextRefreshTime returns the estimated time of next refresh
func (sm *StreamManager) GetNextRefreshTime() time.Time {
	if !sm.isRunning {
		return time.Time{}
	}
	return sm.lastRefresh.Add(sm.interval)
}

// ExecuteQuery helper for streaming queries
func (sm *StreamManager) ExecuteQuery(ctx context.Context, executor *query.Executor, req query.ExecuteRequest) (query.ExecuteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return executor.Execute(ctx, req)
}
