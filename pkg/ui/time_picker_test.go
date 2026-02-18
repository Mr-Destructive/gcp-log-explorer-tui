package ui

import (
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewTimePicker(t *testing.T) {
	tp := NewTimePicker()

	if tp.mode != "preset" {
		t.Errorf("Expected mode 'preset', got %s", tp.mode)
	}

	if tp.selectedIdx != 1 {
		t.Errorf("Expected default index 1, got %d", tp.selectedIdx)
	}
}

func TestGetPresets(t *testing.T) {
	tp := NewTimePicker()
	presets := tp.GetPresets()

	if len(presets) != 6 {
		t.Errorf("Expected 6 presets, got %d", len(presets))
	}

	expectedNames := []string{"Last 1 hour", "Last 24 hours", "Last 7 days", "Last 8 days", "Last 30 days", "Custom"}
	for i, preset := range presets {
		if preset.Name != expectedNames[i] {
			t.Errorf("Preset %d: expected %s, got %s", i, expectedNames[i], preset.Name)
		}
	}
}

func TestSelectPreset(t *testing.T) {
	tp := NewTimePicker()

	// Select preset at index 2 (7 days)
	err := tp.SelectPreset(2)
	if err != nil {
		t.Errorf("SelectPreset failed: %v", err)
	}

	if tp.selectedIdx != 2 {
		t.Errorf("Expected index 2, got %d", tp.selectedIdx)
	}

	// Try invalid index
	err = tp.SelectPreset(10)
	if err == nil {
		t.Error("SelectPreset should error on invalid index")
	}
}

func TestSetCustomRange(t *testing.T) {
	tp := NewTimePicker()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	err := tp.SetCustomRange(oneHourAgo, now)
	if err != nil {
		t.Errorf("SetCustomRange failed: %v", err)
	}

	if tp.mode != "custom" {
		t.Errorf("Expected mode 'custom', got %s", tp.mode)
	}

	// Test zero times
	err = tp.SetCustomRange(time.Time{}, now)
	if err == nil {
		t.Error("SetCustomRange should error on zero time")
	}

	// Test start after end
	err = tp.SetCustomRange(now, oneHourAgo)
	if err == nil {
		t.Error("SetCustomRange should error when start > end")
	}
}

func TestSetCustomRangeMaxDuration(t *testing.T) {
	tp := NewTimePicker()

	now := time.Now()
	tooOld := now.Add(-100 * 24 * time.Hour)

	err := tp.SetCustomRange(tooOld, now)
	if err == nil {
		t.Error("SetCustomRange should error when range > 90 days")
	}
}

func TestGetSelectedRange(t *testing.T) {
	tp := NewTimePicker()

	// Default to 24h preset
	tr, err := tp.GetSelectedRange()
	if err != nil {
		t.Errorf("GetSelectedRange failed: %v", err)
	}

	if tr.Preset != "24h" {
		t.Errorf("Expected preset '24h', got %s", tr.Preset)
	}

	if tr.Start.IsZero() || tr.End.IsZero() {
		t.Error("Start and end times should not be zero")
	}

	if tr.End.Before(tr.Start) {
		t.Error("End time should be after start time")
	}
}

func TestGetCurrentPresetName(t *testing.T) {
	tp := NewTimePicker()

	name := tp.GetCurrentPresetName()
	if name != "Last 24 hours" {
		t.Errorf("Expected 'Last 24 hours', got %s", name)
	}

	tp.SelectPreset(0)
	name = tp.GetCurrentPresetName()
	if name != "Last 1 hour" {
		t.Errorf("Expected 'Last 1 hour', got %s", name)
	}
}

func TestMoveSelection(t *testing.T) {
	tp := NewTimePicker()
	initialIdx := tp.selectedIdx

	tp.MoveSelection(1)
	if tp.selectedIdx != initialIdx+1 {
		t.Error("MoveSelection should increment index")
	}

	tp.MoveSelection(-2)
	if tp.selectedIdx != initialIdx-1 {
		t.Error("MoveSelection should decrement index")
	}

	// Test bounds
	tp.selectedIdx = 0
	tp.MoveSelection(-5)
	if tp.selectedIdx != 0 {
		t.Error("MoveSelection should not go below 0")
	}

	tp.selectedIdx = 5
	tp.MoveSelection(10)
	if tp.selectedIdx != 5 {
		t.Error("MoveSelection should not exceed max")
	}
}

func TestApplyToFilterState(t *testing.T) {
	tp := NewTimePicker()
	fs := &models.FilterState{}

	err := tp.ApplyToFilterState(fs)
	if err != nil {
		t.Errorf("ApplyToFilterState failed: %v", err)
	}

	if fs.TimeRange.Preset != "24h" {
		t.Errorf("Expected preset '24h', got %s", fs.TimeRange.Preset)
	}

	if fs.TimeRange.Start.IsZero() || fs.TimeRange.End.IsZero() {
		t.Error("FilterState times should be set")
	}
}

func TestTimePickerReset(t *testing.T) {
	tp := NewTimePicker()

	// Change state
	tp.SelectPreset(3)
	tp.SetError("test error")

	// Reset
	tp.Reset()

	if tp.mode != "preset" {
		t.Errorf("Mode should be reset to 'preset', got %s", tp.mode)
	}

	if tp.selectedIdx != 1 {
		t.Errorf("Index should be reset to 1, got %d", tp.selectedIdx)
	}

	if tp.error != "" {
		t.Errorf("Error should be cleared, got %s", tp.error)
	}
}

func TestErrorHandling(t *testing.T) {
	tp := NewTimePicker()

	if tp.GetError() != "" {
		t.Error("Error should be empty initially")
	}

	tp.SetError("test error")
	if tp.GetError() != "test error" {
		t.Errorf("Expected 'test error', got %s", tp.GetError())
	}
}

func TestCustomRangeAccuracy(t *testing.T) {
	tp := NewTimePicker()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	err := tp.SetCustomRange(start, end)
	if err != nil {
		t.Errorf("SetCustomRange failed: %v", err)
	}

	tr, err := tp.GetSelectedRange()
	if err != nil {
		t.Errorf("GetSelectedRange failed: %v", err)
	}

	if !tr.Start.Equal(start) {
		t.Errorf("Start time mismatch: expected %v, got %v", start, tr.Start)
	}

	if !tr.End.Equal(end) {
		t.Errorf("End time mismatch: expected %v, got %v", end, tr.End)
	}
}

func TestMultiplePresetSelections(t *testing.T) {
	tp := NewTimePicker()

	presets := []int{0, 1, 2, 3, 4, 5}
	for _, idx := range presets {
		err := tp.SelectPreset(idx)
		if err != nil {
			t.Errorf("SelectPreset %d failed: %v", idx, err)
		}

		if tp.selectedIdx != idx {
			t.Errorf("Index mismatch: expected %d, got %d", idx, tp.selectedIdx)
		}
	}
}

func TestGetSelectedRangeWithCustom(t *testing.T) {
	tp := NewTimePicker()

	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()

	err := tp.SetCustomRange(start, end)
	if err != nil {
		t.Errorf("SetCustomRange failed: %v", err)
	}

	tr, err := tp.GetSelectedRange()
	if err != nil {
		t.Errorf("GetSelectedRange failed: %v", err)
	}

	if tr.Preset != "custom" {
		t.Errorf("Expected preset 'custom', got %s", tr.Preset)
	}
}

func TestCustomPresetInitializesDefaults(t *testing.T) {
	tp := NewTimePicker()
	if err := tp.SelectPreset(5); err != nil {
		t.Fatalf("SelectPreset(custom) failed: %v", err)
	}

	if !tp.IsCustomSelected() {
		t.Fatal("expected custom mode after selecting custom preset")
	}

	start, end := tp.GetCustomRange()
	if start.IsZero() || end.IsZero() {
		t.Fatal("expected custom start/end defaults to be initialized")
	}
	if !start.Before(end) {
		t.Fatal("expected custom start to be before end")
	}
}

func TestCustomFieldShiftMaintainsOrdering(t *testing.T) {
	tp := NewTimePicker()
	if err := tp.SelectPreset(5); err != nil {
		t.Fatalf("SelectPreset(custom) failed: %v", err)
	}

	startBefore, endBefore := tp.GetCustomRange()
	tp.ShiftCustomFocused(30 * time.Minute) // shift start forward
	startAfter, endAfter := tp.GetCustomRange()
	if !startAfter.After(startBefore) {
		t.Fatal("expected start to move forward")
	}
	if endAfter.Before(startAfter) {
		t.Fatal("end should never be before start")
	}

	tp.ToggleCustomField()
	tp.ShiftCustomFocused(-30 * time.Minute) // shift end backward
	_, endAfter2 := tp.GetCustomRange()
	if !endAfter2.Before(endBefore) {
		t.Fatal("expected end to move backward")
	}
}

func TestApplyCustomInputs(t *testing.T) {
	tp := NewTimePicker()
	if err := tp.SelectPreset(5); err != nil {
		t.Fatalf("SelectPreset(custom) failed: %v", err)
	}

	tp.ClearFocusedInput()
	tp.AppendToFocusedInput("2026-02-01 10:20:30")
	tp.ToggleCustomField()
	tp.ClearFocusedInput()
	tp.AppendToFocusedInput("2026-02-03 12:00:00")

	if err := tp.ApplyCustomInputs(); err != nil {
		t.Fatalf("ApplyCustomInputs failed: %v", err)
	}

	start, end := tp.GetCustomRange()
	if start.Year() != 2026 || start.Month() != 2 || start.Day() != 1 || start.Second() != 30 {
		t.Fatalf("unexpected parsed start: %v", start)
	}
	if end.Year() != 2026 || end.Month() != 2 || end.Day() != 3 || end.Hour() != 12 {
		t.Fatalf("unexpected parsed end: %v", end)
	}
}
