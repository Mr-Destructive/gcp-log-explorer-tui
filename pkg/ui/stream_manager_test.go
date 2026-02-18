package ui

import (
	"context"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewStreamManager(t *testing.T) {
	sm := NewStreamManager(2 * time.Second)

	if sm.interval != 2*time.Second {
		t.Errorf("Expected interval 2s, got %v", sm.interval)
	}

	if sm.enabled {
		t.Error("Should not be enabled initially")
	}

	if sm.isRunning {
		t.Error("Should not be running initially")
	}
}

func TestNewStreamManagerMinimum(t *testing.T) {
	sm := NewStreamManager(100 * time.Millisecond)

	if sm.interval != time.Second {
		t.Errorf("Interval should be at least 1s, got %v", sm.interval)
	}
}

func TestEnable(t *testing.T) {
	sm := NewStreamManager(time.Second)

	err := sm.Enable()
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	if !sm.enabled {
		t.Error("Should be enabled after Enable()")
	}

	// Try enabling again
	err = sm.Enable()
	if err == nil {
		t.Error("Should error when already enabled")
	}
}

func TestDisable(t *testing.T) {
	sm := NewStreamManager(time.Second)

	sm.Enable()
	err := sm.Disable()
	if err != nil {
		t.Errorf("Disable failed: %v", err)
	}

	if sm.enabled {
		t.Error("Should be disabled after Disable()")
	}

	// Try disabling again
	err = sm.Disable()
	if err == nil {
		t.Error("Should error when not enabled")
	}
}

func TestIsEnabled(t *testing.T) {
	sm := NewStreamManager(time.Second)

	if sm.IsEnabled() {
		t.Error("Should not be enabled initially")
	}

	sm.Enable()
	if !sm.IsEnabled() {
		t.Error("Should be enabled after Enable()")
	}
}

func TestSetRefreshCallback(t *testing.T) {
	sm := NewStreamManager(time.Second)

	sm.SetRefreshCallback(func() error {
		return nil
	})

	if sm.refreshCallback == nil {
		t.Error("Callback should be set")
	}
}

func TestIncrementNewLogsCount(t *testing.T) {
	sm := NewStreamManager(time.Second)

	if sm.GetNewLogsCount() != 0 {
		t.Error("Count should be 0 initially")
	}

	sm.IncrementNewLogsCount(5)
	if sm.GetNewLogsCount() != 5 {
		t.Errorf("Expected 5, got %d", sm.GetNewLogsCount())
	}

	sm.IncrementNewLogsCount(3)
	if sm.GetNewLogsCount() != 8 {
		t.Errorf("Expected 8, got %d", sm.GetNewLogsCount())
	}
}

func TestResetNewLogsCount(t *testing.T) {
	sm := NewStreamManager(time.Second)

	sm.IncrementNewLogsCount(5)
	sm.ResetNewLogsCount()

	if sm.GetNewLogsCount() != 0 {
		t.Errorf("Expected 0, got %d", sm.GetNewLogsCount())
	}
}

func TestSetInterval(t *testing.T) {
	sm := NewStreamManager(time.Second)

	err := sm.SetInterval(5 * time.Second)
	if err != nil {
		t.Errorf("SetInterval failed: %v", err)
	}

	if sm.GetInterval() != 5*time.Second {
		t.Errorf("Expected 5s, got %v", sm.GetInterval())
	}

	// Try invalid interval
	err = sm.SetInterval(100 * time.Millisecond)
	if err == nil {
		t.Error("Should error on invalid interval")
	}
}

func TestGetStatus(t *testing.T) {
	sm := NewStreamManager(2 * time.Second)
	sm.Enable()
	sm.IncrementNewLogsCount(3)

	status := sm.GetStatus()

	if !status["enabled"].(bool) {
		t.Error("Status should show enabled")
	}

	if status["new_logs"].(int) != 3 {
		t.Errorf("Expected 3 new logs, got %v", status["new_logs"])
	}
}

func TestApplyToStreamState(t *testing.T) {
	sm := NewStreamManager(2 * time.Second)
	sm.Enable()
	sm.IncrementNewLogsCount(5)

	ss := &models.StreamState{}
	sm.ApplyToStreamState(ss)

	if !ss.Enabled {
		t.Error("StreamState should be enabled")
	}

	if ss.NewLogsCount != 5 {
		t.Errorf("Expected 5 new logs, got %d", ss.NewLogsCount)
	}
}

func TestUpdateFromStreamState(t *testing.T) {
	sm := NewStreamManager(time.Second)

	ss := models.StreamState{
		Enabled:         true,
		RefreshInterval: 5 * time.Second,
		NewLogsCount:    10,
	}

	sm.UpdateFromStreamState(ss)

	if !sm.enabled {
		t.Error("Should be enabled after update")
	}

	if sm.GetInterval() != 5*time.Second {
		t.Errorf("Expected 5s, got %v", sm.GetInterval())
	}

	if sm.GetNewLogsCount() != 10 {
		t.Errorf("Expected 10, got %d", sm.GetNewLogsCount())
	}
}

func TestGetLastRefreshTime(t *testing.T) {
	sm := NewStreamManager(time.Second)

	lastTime := sm.GetLastRefreshTime()
	if lastTime.IsZero() {
		t.Error("Last refresh time should not be zero")
	}
}

func TestGetTimeSinceLastRefresh(t *testing.T) {
	sm := NewStreamManager(time.Second)

	time.Sleep(100 * time.Millisecond)
	duration := sm.GetTimeSinceLastRefresh()

	if duration < 100*time.Millisecond {
		t.Errorf("Expected at least 100ms, got %v", duration)
	}
}

func TestGetNextRefreshTime(t *testing.T) {
	sm := NewStreamManager(2 * time.Second)

	// Not running
	nextTime := sm.GetNextRefreshTime()
	if !nextTime.IsZero() {
		t.Error("Should return zero time when not running")
	}
}

func TestStartStreamingNotEnabled(t *testing.T) {
	sm := NewStreamManager(time.Second)

	ctx := context.Background()
	err := sm.StartStreaming(ctx)

	if err == nil {
		t.Error("Should error when not enabled")
	}
}

func TestStartStreamingAlreadyRunning(t *testing.T) {
	sm := NewStreamManager(time.Second)
	sm.Enable()

	ctx := context.Background()
	err := sm.StartStreaming(ctx)
	if err != nil {
		t.Errorf("StartStreaming failed: %v", err)
	}

	err = sm.StartStreaming(ctx)
	if err == nil {
		t.Error("Should error when already running")
	}

	sm.StopStreaming()
}

func TestStopStreaming(t *testing.T) {
	sm := NewStreamManager(time.Second)
	sm.Enable()

	ctx := context.Background()
	sm.StartStreaming(ctx)

	if !sm.isRunning {
		t.Error("Should be running")
	}

	sm.StopStreaming()

	if sm.isRunning {
		t.Error("Should not be running after stop")
	}
}

func TestStreamingContextCancel(t *testing.T) {
	sm := NewStreamManager(100 * time.Millisecond)
	sm.Enable()

	ctx, cancel := context.WithCancel(context.Background())
	err := sm.StartStreaming(ctx)
	if err != nil {
		t.Errorf("StartStreaming failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()

	time.Sleep(50 * time.Millisecond)

	if sm.isRunning {
		t.Error("Should stop when context is cancelled")
	}
}
