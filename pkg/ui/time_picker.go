package ui

import (
	"fmt"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// TimePicker handles time range selection
type TimePicker struct {
	mode        string // "preset" or "custom"
	selectedIdx int
	customStart time.Time
	customEnd   time.Time
	customStartInput string
	customEndInput   string
	customField int // 0=start, 1=end
	error       string
}

// NewTimePicker creates a new time picker
func NewTimePicker() *TimePicker {
	return &TimePicker{
		mode:        "preset",
		selectedIdx: 1, // Default to "24h"
		customStart: time.Time{},
		customEnd:   time.Time{},
		customStartInput: "",
		customEndInput:   "",
		customField: 0,
	}
}

// GetPresets returns available time range presets
func (tp *TimePicker) GetPresets() []TimePreset {
	now := time.Now()
	return []TimePreset{
		{
			Name:     "Last 1 hour",
			Key:      "1h",
			Duration: 1 * time.Hour,
			Start:    now.Add(-1 * time.Hour),
			End:      now,
		},
		{
			Name:     "Last 24 hours",
			Key:      "24h",
			Duration: 24 * time.Hour,
			Start:    now.Add(-24 * time.Hour),
			End:      now,
		},
		{
			Name:     "Last 7 days",
			Key:      "7d",
			Duration: 7 * 24 * time.Hour,
			Start:    now.Add(-7 * 24 * time.Hour),
			End:      now,
		},
		{
			Name:     "Last 8 days",
			Key:      "8d",
			Duration: 8 * 24 * time.Hour,
			Start:    now.Add(-8 * 24 * time.Hour),
			End:      now,
		},
		{
			Name:     "Last 30 days",
			Key:      "30d",
			Duration: 30 * 24 * time.Hour,
			Start:    now.Add(-30 * 24 * time.Hour),
			End:      now,
		},
		{
			Name:     "Custom",
			Key:      "custom",
			Duration: 0,
			Start:    time.Time{},
			End:      time.Time{},
		},
	}
}

// TimePreset represents a preset time range
type TimePreset struct {
	Name     string
	Key      string
	Duration time.Duration
	Start    time.Time
	End      time.Time
}

// SelectPreset selects a preset by index
func (tp *TimePicker) SelectPreset(idx int) error {
	presets := tp.GetPresets()
	if idx < 0 || idx >= len(presets) {
		return fmt.Errorf("invalid preset index: %d", idx)
	}

	tp.selectedIdx = idx
	if presets[idx].Key == "custom" {
		tp.mode = "custom"
		tp.ensureCustomDefaults()
	} else {
		tp.mode = "preset"
	}
	tp.error = ""
	return nil
}

// SetCustomRange sets custom start and end times
func (tp *TimePicker) SetCustomRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return fmt.Errorf("start and end times cannot be zero")
	}

	if start.After(end) {
		return fmt.Errorf("start time cannot be after end time")
	}

	maxDuration := 90 * 24 * time.Hour
	if end.Sub(start) > maxDuration {
		return fmt.Errorf("time range cannot exceed 90 days")
	}

	tp.customStart = start
	tp.customEnd = end
	tp.customStartInput = tp.formatCustomTime(start)
	tp.customEndInput = tp.formatCustomTime(end)
	tp.mode = "custom"
	tp.error = ""
	return nil
}

// GetSelectedRange returns the currently selected time range
func (tp *TimePicker) GetSelectedRange() (models.TimeRange, error) {
	if tp.mode == "preset" {
		presets := tp.GetPresets()
		if tp.selectedIdx >= len(presets) {
			return models.TimeRange{}, fmt.Errorf("invalid preset index")
		}

		preset := presets[tp.selectedIdx]
		return models.TimeRange{
			Start:  preset.Start,
			End:    preset.End,
			Preset: preset.Key,
		}, nil
	}

	if tp.mode == "custom" {
		if tp.customStart.IsZero() || tp.customEnd.IsZero() {
			return models.TimeRange{}, fmt.Errorf("custom range not set")
		}

		return models.TimeRange{
			Start:  tp.customStart,
			End:    tp.customEnd,
			Preset: "custom",
		}, nil
	}

	return models.TimeRange{}, fmt.Errorf("invalid mode: %s", tp.mode)
}

// GetCurrentPresetName returns the name of the current preset
func (tp *TimePicker) GetCurrentPresetName() string {
	presets := tp.GetPresets()
	if tp.selectedIdx < len(presets) {
		return presets[tp.selectedIdx].Name
	}
	return "Unknown"
}

// GetSelectedIdx returns the currently selected preset index
func (tp *TimePicker) GetSelectedIdx() int {
	return tp.selectedIdx
}

// MoveSelection moves selection up/down
func (tp *TimePicker) MoveSelection(delta int) {
	if tp.mode == "custom" {
		return
	}

	presets := tp.GetPresets()
	newIdx := tp.selectedIdx + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= len(presets) {
		newIdx = len(presets) - 1
	}
	tp.selectedIdx = newIdx
	if presets[newIdx].Key == "custom" {
		tp.mode = "custom"
		tp.ensureCustomDefaults()
	} else {
		tp.mode = "preset"
	}
}

// ApplyToFilterState applies the selected time range to a FilterState
func (tp *TimePicker) ApplyToFilterState(fs *models.FilterState) error {
	tr, err := tp.GetSelectedRange()
	if err != nil {
		return err
	}

	fs.TimeRange = tr
	return nil
}

// Reset resets the time picker to defaults
func (tp *TimePicker) Reset() {
	tp.mode = "preset"
	tp.selectedIdx = 1 // Default to "24h"
	tp.customStart = time.Time{}
	tp.customEnd = time.Time{}
	tp.customStartInput = ""
	tp.customEndInput = ""
	tp.customField = 0
	tp.error = ""
}

// GetError returns the last error message
func (tp *TimePicker) GetError() string {
	return tp.error
}

// SetError sets an error message
func (tp *TimePicker) SetError(msg string) {
	tp.error = msg
}

// IsCustomSelected returns true when custom mode is active.
func (tp *TimePicker) IsCustomSelected() bool {
	return tp.mode == "custom"
}

// EnsureCustomDefaults initializes custom range if unset.
func (tp *TimePicker) EnsureCustomDefaults() {
	tp.ensureCustomDefaults()
}

// ToggleCustomField switches focused field between start/end.
func (tp *TimePicker) ToggleCustomField() {
	if tp.customField == 0 {
		tp.customField = 1
	} else {
		tp.customField = 0
	}
}

// GetCustomField returns current focused custom field (0=start, 1=end).
func (tp *TimePicker) GetCustomField() int {
	return tp.customField
}

// ShiftCustomFocused shifts the focused custom field by the given duration.
func (tp *TimePicker) ShiftCustomFocused(delta time.Duration) {
	tp.ensureCustomDefaults()
	if tp.customField == 0 {
		tp.customStart = tp.customStart.Add(delta)
		if tp.customStart.After(tp.customEnd) {
			tp.customStart = tp.customEnd.Add(-1 * time.Minute)
		}
		tp.customStartInput = tp.formatCustomTime(tp.customStart)
	} else {
		tp.customEnd = tp.customEnd.Add(delta)
		if tp.customEnd.Before(tp.customStart) {
			tp.customEnd = tp.customStart.Add(1 * time.Minute)
		}
		tp.customEndInput = tp.formatCustomTime(tp.customEnd)
	}
}

// GetCustomRange returns current custom start/end.
func (tp *TimePicker) GetCustomRange() (time.Time, time.Time) {
	tp.ensureCustomDefaults()
	return tp.customStart, tp.customEnd
}

// GetCustomInputs returns editable custom input strings.
func (tp *TimePicker) GetCustomInputs() (string, string) {
	tp.ensureCustomDefaults()
	return tp.customStartInput, tp.customEndInput
}

// AppendToFocusedInput appends typed text to active custom field.
func (tp *TimePicker) AppendToFocusedInput(text string) {
	if tp.customField == 0 {
		tp.customStartInput += text
	} else {
		tp.customEndInput += text
	}
}

// BackspaceFocusedInput removes one character from active custom field.
func (tp *TimePicker) BackspaceFocusedInput() {
	if tp.customField == 0 {
		if len(tp.customStartInput) > 0 {
			tp.customStartInput = tp.customStartInput[:len(tp.customStartInput)-1]
		}
	} else {
		if len(tp.customEndInput) > 0 {
			tp.customEndInput = tp.customEndInput[:len(tp.customEndInput)-1]
		}
	}
}

// ClearFocusedInput clears active custom field.
func (tp *TimePicker) ClearFocusedInput() {
	if tp.customField == 0 {
		tp.customStartInput = ""
	} else {
		tp.customEndInput = ""
	}
}

// ApplyCustomInputs parses custom text input and sets start/end.
func (tp *TimePicker) ApplyCustomInputs() error {
	start, err := tp.parseCustomTime(tp.customStartInput)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	end, err := tp.parseCustomTime(tp.customEndInput)
	if err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}
	return tp.SetCustomRange(start, end)
}

func (tp *TimePicker) ensureCustomDefaults() {
	if !tp.customStart.IsZero() && !tp.customEnd.IsZero() {
		return
	}
	now := time.Now().UTC().Truncate(time.Minute)
	tp.customEnd = now
	tp.customStart = now.Add(-1 * time.Hour)
	tp.customStartInput = tp.formatCustomTime(tp.customStart)
	tp.customEndInput = tp.formatCustomTime(tp.customEnd)
}

func (tp *TimePicker) formatCustomTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

func (tp *TimePicker) parseCustomTime(input string) (time.Time, error) {
	s := input
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("expected YYYY-MM-DD HH:MM:SS or RFC3339")
}
