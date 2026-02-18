package ui

import (
	"strings"
	"testing"
)

// TestNewHelpModal tests help modal creation
func TestNewHelpModal(t *testing.T) {
	hm := NewHelpModal()
	if hm == nil {
		t.Fatal("HelpModal should not be nil")
	}
	if hm.IsVisible() {
		t.Error("Help modal should not be visible by default")
	}
}

// TestSetVisible tests visibility toggling
func TestSetVisible(t *testing.T) {
	hm := NewHelpModal()

	hm.SetVisible(true)
	if !hm.IsVisible() {
		t.Error("Help modal should be visible")
	}

	hm.SetVisible(false)
	if hm.IsVisible() {
		t.Error("Help modal should not be visible")
	}
}

// TestRenderHidden tests rendering when not visible
func TestRenderHidden(t *testing.T) {
	hm := NewHelpModal()
	hm.SetVisible(false)

	output := hm.Render(80, 24)
	if output != "" {
		t.Error("Hidden help modal should render empty string")
	}
}

// TestRenderVisible tests rendering when visible
func TestRenderVisible(t *testing.T) {
	hm := NewHelpModal()
	hm.SetVisible(true)

	output := hm.Render(80, 24)
	if output == "" {
		t.Fatal("Visible help modal should not be empty")
	}

	// Check for key content
	if !strings.Contains(output, "LOG EXPLORER TUI HELP") {
		t.Error("Help should contain title")
	}
	if !strings.Contains(output, "KEY") || !strings.Contains(output, "ACTION") {
		t.Error("Help should contain table headers")
	}
	if strings.Contains(output, "📖") || strings.Contains(output, "💡") {
		t.Error("Help should not contain emoji")
	}
}

// TestGetShortHelp tests short help text
func TestGetShortHelp(t *testing.T) {
	hm := NewHelpModal()
	shortHelp := hm.GetShortHelp()

	if shortHelp == "" {
		t.Fatal("Short help should not be empty")
	}

	if !strings.Contains(shortHelp, "?") {
		t.Error("Short help should mention help key")
	}
	if !strings.Contains(shortHelp, "arrows") {
		t.Error("Short help should mention arrow navigation")
	}
}

// TestRenderWithDifferentDimensions tests rendering with various dimensions
func TestRenderWithDifferentDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"Small terminal", 40, 12},
		{"Standard terminal", 80, 24},
		{"Large terminal", 120, 40},
		{"Wide terminal", 200, 30},
	}

	hm := NewHelpModal()
	hm.SetVisible(true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := hm.Render(tt.width, tt.height)
			if output == "" {
				t.Error("Output should not be empty")
			}
		})
	}
}

// TestHelpContentStructure tests help content organization
func TestHelpContentStructure(t *testing.T) {
	hm := NewHelpModal()
	hm.SetVisible(true)

	output := hm.Render(100, 40)

	// Check for key commands - looking for partial matches since styling may interfere
	commands := []string{
		"Open query editor",
		"Previous / next query history",
		"Time range filter",
		"Severity filter",
		"selected log",
		"loaded logs (CSV)",
		"Project",
		"Quit",
	}

	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Help should mention: %s", cmd)
		}
	}
}
