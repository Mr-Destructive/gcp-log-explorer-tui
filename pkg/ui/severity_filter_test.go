package ui

import (
	"testing"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestNewSeverityFilterPanel(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	if sfp.mode != "individual" {
		t.Errorf("Expected mode 'individual', got %s", sfp.mode)
	}

	if sfp.minLevel != models.SeverityError {
		t.Errorf("Expected default minLevel '%s', got %s", models.SeverityError, sfp.minLevel)
	}

	if len(sfp.severityLevels) != len(models.SeverityLevels) {
		t.Errorf("Severity levels count mismatch")
	}
}

func TestSetMode(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	// Valid modes
	for _, mode := range []string{"individual", "range"} {
		err := sfp.SetMode(mode)
		if err != nil {
			t.Errorf("SetMode failed for %s: %v", mode, err)
		}
		if sfp.mode != mode {
			t.Errorf("Mode not set correctly: expected %s, got %s", mode, sfp.mode)
		}
	}

	// Invalid mode
	err := sfp.SetMode("invalid")
	if err == nil {
		t.Error("SetMode should error on invalid mode")
	}
}

func TestToggleLevel(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	// Toggle ERROR on
	err := sfp.ToggleLevel(models.SeverityError)
	if err != nil {
		t.Errorf("ToggleLevel failed: %v", err)
	}

	if !sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should be selected after toggle")
	}

	// Toggle ERROR off
	err = sfp.ToggleLevel(models.SeverityError)
	if err != nil {
		t.Errorf("ToggleLevel failed: %v", err)
	}

	if sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should not be selected after second toggle")
	}

	// Invalid level
	err = sfp.ToggleLevel("INVALID")
	if err == nil {
		t.Error("ToggleLevel should error on invalid level")
	}
}

func TestSetLevel(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	err := sfp.SetLevel(models.SeverityError, true)
	if err != nil {
		t.Errorf("SetLevel failed: %v", err)
	}

	if !sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should be selected")
	}

	err = sfp.SetLevel(models.SeverityError, false)
	if err != nil {
		t.Errorf("SetLevel failed: %v", err)
	}

	if sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should not be selected")
	}
}

func TestSetMinimumLevel(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	err := sfp.SetMinimumLevel(models.SeverityWarning)
	if err != nil {
		t.Errorf("SetMinimumLevel failed: %v", err)
	}

	if sfp.minLevel != models.SeverityWarning {
		t.Errorf("Expected minLevel '%s', got %s", models.SeverityWarning, sfp.minLevel)
	}

	// Invalid level
	err = sfp.SetMinimumLevel("INVALID")
	if err == nil {
		t.Error("SetMinimumLevel should error on invalid level")
	}
}

func TestGetSelectedLevels(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	sfp.SetLevel(models.SeverityError, true)
	sfp.SetLevel(models.SeverityCritical, true)

	selected := sfp.GetSelectedLevels()
	if len(selected) != 2 {
		t.Errorf("Expected 2 selected levels, got %d", len(selected))
	}

	// Check that both are in the list
	hasError := false
	hasCritical := false
	for _, level := range selected {
		if level == models.SeverityError {
			hasError = true
		}
		if level == models.SeverityCritical {
			hasCritical = true
		}
	}

	if !hasError || !hasCritical {
		t.Error("Selected levels should contain ERROR and CRITICAL")
	}
}

func TestGetMinimumLevel(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	sfp.SetMinimumLevel(models.SeverityWarning)
	if sfp.GetMinimumLevel() != models.SeverityWarning {
		t.Errorf("Expected %s, got %s", models.SeverityWarning, sfp.GetMinimumLevel())
	}
}

func TestApplyToFilterStateIndividual(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	sfp.SetLevel(models.SeverityError, true)
	sfp.SetLevel(models.SeverityCritical, true)

	fs := &models.FilterState{}
	err := sfp.ApplyToFilterState(fs)
	if err != nil {
		t.Errorf("ApplyToFilterState failed: %v", err)
	}

	if fs.Severity.Mode != "individual" {
		t.Errorf("Expected mode 'individual', got %s", fs.Severity.Mode)
	}

	if len(fs.Severity.Levels) != 2 {
		t.Errorf("Expected 2 levels in filter, got %d", len(fs.Severity.Levels))
	}
}

func TestApplyToFilterStateRange(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	sfp.SetMode("range")
	sfp.SetMinimumLevel(models.SeverityWarning)

	fs := &models.FilterState{}
	err := sfp.ApplyToFilterState(fs)
	if err != nil {
		t.Errorf("ApplyToFilterState failed: %v", err)
	}

	if fs.Severity.Mode != "range" {
		t.Errorf("Expected mode 'range', got %s", fs.Severity.Mode)
	}

	if fs.Severity.MinLevel != models.SeverityWarning {
		t.Errorf("Expected minLevel '%s', got %s", models.SeverityWarning, fs.Severity.MinLevel)
	}
}

func TestApplyToFilterStateNoSelection(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	// Don't select anything

	fs := &models.FilterState{}
	err := sfp.ApplyToFilterState(fs)

	if err == nil {
		t.Error("ApplyToFilterState should error when no levels selected")
	}
}

func TestSelectAllLevels(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	sfp.SelectAllLevels()
	selected := sfp.GetSelectedLevels()

	if len(selected) != len(sfp.severityLevels) {
		t.Errorf("Expected all levels selected, got %d of %d", len(selected), len(sfp.severityLevels))
	}
}

func TestDeselectAllLevels(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	sfp.SelectAllLevels()
	sfp.DeselectAllLevels()

	selected := sfp.GetSelectedLevels()
	if len(selected) != 0 {
		t.Errorf("Expected no levels selected, got %d", len(selected))
	}
}

func TestSeverityFilterReset(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	// Change state
	sfp.SetMode("range")
	sfp.SetLevel(models.SeverityError, true)
	sfp.SetMinimumLevel(models.SeverityWarning)

	// Reset
	sfp.Reset()

	if sfp.mode != "individual" {
		t.Errorf("Mode should be reset to 'individual', got %s", sfp.mode)
	}

	if sfp.minLevel != models.SeverityError {
		t.Errorf("minLevel should be reset, got %s", sfp.minLevel)
	}

	if len(sfp.GetSelectedLevels()) != 0 {
		t.Error("Selected levels should be cleared")
	}
}

func TestCountSelectedLevels(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	if sfp.CountSelectedLevels() != 0 {
		t.Error("Count should be 0 initially")
	}

	sfp.SetLevel(models.SeverityError, true)
	if sfp.CountSelectedLevels() != 1 {
		t.Error("Count should be 1")
	}

	sfp.SetLevel(models.SeverityCritical, true)
	if sfp.CountSelectedLevels() != 2 {
		t.Error("Count should be 2")
	}
}

func TestGetMode(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	if sfp.GetMode() != "individual" {
		t.Errorf("Initial mode should be 'individual', got %s", sfp.GetMode())
	}

	sfp.SetMode("range")
	if sfp.GetMode() != "range" {
		t.Errorf("Mode should be 'range', got %s", sfp.GetMode())
	}
}

func TestGetFilterPresets(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	presets := sfp.GetFilterPresets()

	if len(presets) != 3 {
		t.Errorf("Expected 3 presets, got %d", len(presets))
	}

	expectedNames := []string{"Errors & Critical", "Warnings & Above", "All Levels"}
	for i, preset := range presets {
		if preset.Name != expectedNames[i] {
			t.Errorf("Preset %d: expected %s, got %s", i, expectedNames[i], preset.Name)
		}
	}
}

func TestApplyPreset(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	presets := sfp.GetFilterPresets()

	// Apply "Errors & Critical" preset
	err := sfp.ApplyPreset(presets[0])
	if err != nil {
		t.Errorf("ApplyPreset failed: %v", err)
	}

	if sfp.mode != "individual" {
		t.Errorf("Expected mode 'individual', got %s", sfp.mode)
	}

	selected := sfp.GetSelectedLevels()
	if len(selected) != 2 {
		t.Errorf("Expected 2 selected levels, got %d", len(selected))
	}
}

func TestApplyPresetRange(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	presets := sfp.GetFilterPresets()

	// Apply "Warnings & Above" preset
	err := sfp.ApplyPreset(presets[1])
	if err != nil {
		t.Errorf("ApplyPreset failed: %v", err)
	}

	if sfp.mode != "range" {
		t.Errorf("Expected mode 'range', got %s", sfp.mode)
	}

	if sfp.minLevel != models.SeverityWarning {
		t.Errorf("Expected minLevel '%s', got %s", models.SeverityWarning, sfp.minLevel)
	}
}

func TestIsLevelSelected(t *testing.T) {
	sfp := NewSeverityFilterPanel()

	if sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should not be selected initially")
	}

	sfp.SetLevel(models.SeverityError, true)
	if !sfp.IsLevelSelected(models.SeverityError) {
		t.Error("ERROR should be selected")
	}
}

func TestGetSeverityLevels(t *testing.T) {
	sfp := NewSeverityFilterPanel()
	levels := sfp.GetSeverityLevels()

	if len(levels) != len(models.SeverityLevels) {
		t.Errorf("Expected %d levels, got %d", len(models.SeverityLevels), len(levels))
	}

	for i, level := range levels {
		if level != models.SeverityLevels[i] {
			t.Errorf("Level %d mismatch", i)
		}
	}
}
