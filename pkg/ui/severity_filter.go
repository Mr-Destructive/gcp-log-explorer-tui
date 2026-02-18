package ui

import (
	"fmt"

	"github.com/user/log-explorer-tui/pkg/models"
)

// SeverityFilterPanel handles severity level filtering
type SeverityFilterPanel struct {
	mode           string   // "individual" or "range"
	selectedLevels map[string]bool
	minLevel       string
	severityLevels []string
}

// NewSeverityFilterPanel creates a new severity filter panel
func NewSeverityFilterPanel() *SeverityFilterPanel {
	return &SeverityFilterPanel{
		mode:           "individual",
		selectedLevels: make(map[string]bool),
		minLevel:       models.SeverityError,
		severityLevels: models.SeverityLevels,
	}
}

// SetMode sets the filtering mode ("individual" or "range")
func (sfp *SeverityFilterPanel) SetMode(mode string) error {
	if mode != "individual" && mode != "range" {
		return fmt.Errorf("invalid mode: %s (must be 'individual' or 'range')", mode)
	}
	sfp.mode = mode
	return nil
}

// ToggleLevel toggles a severity level for individual mode
func (sfp *SeverityFilterPanel) ToggleLevel(level string) error {
	// Validate level
	if !sfp.isValidLevel(level) {
		return fmt.Errorf("invalid severity level: %s", level)
	}

	sfp.selectedLevels[level] = !sfp.selectedLevels[level]
	return nil
}

// SetLevel sets a severity level (for individual mode)
func (sfp *SeverityFilterPanel) SetLevel(level string, selected bool) error {
	if !sfp.isValidLevel(level) {
		return fmt.Errorf("invalid severity level: %s", level)
	}

	sfp.selectedLevels[level] = selected
	return nil
}

// SetMinimumLevel sets the minimum severity level (for range mode)
func (sfp *SeverityFilterPanel) SetMinimumLevel(level string) error {
	if !sfp.isValidLevel(level) {
		return fmt.Errorf("invalid severity level: %s", level)
	}

	sfp.minLevel = level
	return nil
}

// GetSelectedLevels returns currently selected levels (individual mode)
func (sfp *SeverityFilterPanel) GetSelectedLevels() []string {
	var selected []string
	for _, level := range sfp.severityLevels {
		if sfp.selectedLevels[level] {
			selected = append(selected, level)
		}
	}
	return selected
}

// GetMinimumLevel returns the minimum severity level (range mode)
func (sfp *SeverityFilterPanel) GetMinimumLevel() string {
	return sfp.minLevel
}

// GetSeverityLevels returns all available severity levels
func (sfp *SeverityFilterPanel) GetSeverityLevels() []string {
	return sfp.severityLevels
}

// IsLevelSelected returns whether a level is selected (individual mode)
func (sfp *SeverityFilterPanel) IsLevelSelected(level string) bool {
	return sfp.selectedLevels[level]
}

// ApplyToFilterState applies the severity filter to a FilterState
func (sfp *SeverityFilterPanel) ApplyToFilterState(fs *models.FilterState) error {
	if sfp.mode == "individual" {
		selected := sfp.GetSelectedLevels()
		if len(selected) == 0 {
			return fmt.Errorf("at least one severity level must be selected in individual mode")
		}

		fs.Severity = models.SeverityFilter{
			Levels: selected,
			Mode:   "individual",
		}
	} else if sfp.mode == "range" {
		fs.Severity = models.SeverityFilter{
			MinLevel: sfp.minLevel,
			Mode:     "range",
		}
	} else {
		return fmt.Errorf("invalid mode: %s", sfp.mode)
	}

	return nil
}

// SelectAllLevels selects all severity levels (individual mode)
func (sfp *SeverityFilterPanel) SelectAllLevels() {
	for _, level := range sfp.severityLevels {
		sfp.selectedLevels[level] = true
	}
}

// DeselectAllLevels deselects all severity levels (individual mode)
func (sfp *SeverityFilterPanel) DeselectAllLevels() {
	for _, level := range sfp.severityLevels {
		sfp.selectedLevels[level] = false
	}
}

// Reset resets the panel to defaults
func (sfp *SeverityFilterPanel) Reset() {
	sfp.mode = "individual"
	sfp.selectedLevels = make(map[string]bool)
	sfp.minLevel = models.SeverityError
}

// GetMode returns the current filtering mode
func (sfp *SeverityFilterPanel) GetMode() string {
	return sfp.mode
}

// CountSelectedLevels returns the number of selected levels
func (sfp *SeverityFilterPanel) CountSelectedLevels() int {
	count := 0
	for _, selected := range sfp.selectedLevels {
		if selected {
			count++
		}
	}
	return count
}

// isValidLevel checks if a severity level is valid
func (sfp *SeverityFilterPanel) isValidLevel(level string) bool {
	for _, l := range sfp.severityLevels {
		if l == level {
			return true
		}
	}
	return false
}

// GetFilterPresets returns common filter presets
func (sfp *SeverityFilterPanel) GetFilterPresets() []SeverityPreset {
	return []SeverityPreset{
		{
			Name:   "Errors & Critical",
			Levels: []string{models.SeverityError, models.SeverityCritical},
			Mode:   "individual",
		},
		{
			Name:   "Warnings & Above",
			MinLevel: models.SeverityWarning,
			Mode:   "range",
		},
		{
			Name:   "All Levels",
			Levels: sfp.severityLevels,
			Mode:   "individual",
		},
	}
}

// SeverityPreset represents a pre-configured severity filter
type SeverityPreset struct {
	Name     string
	Levels   []string
	MinLevel string
	Mode     string
}

// ApplyPreset applies a preset filter
func (sfp *SeverityFilterPanel) ApplyPreset(preset SeverityPreset) error {
	if err := sfp.SetMode(preset.Mode); err != nil {
		return err
	}

	if preset.Mode == "individual" {
		sfp.DeselectAllLevels()
		for _, level := range preset.Levels {
			if err := sfp.SetLevel(level, true); err != nil {
				return err
			}
		}
	} else if preset.Mode == "range" {
		if err := sfp.SetMinimumLevel(preset.MinLevel); err != nil {
			return err
		}
	}

	return nil
}
