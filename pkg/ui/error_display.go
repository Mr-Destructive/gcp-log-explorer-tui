package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ErrorDisplay manages error message display
type ErrorDisplay struct {
	messages []ErrorMessage
	maxSize  int
}

// ErrorMessage represents a single error message
type ErrorMessage struct {
	Text      string
	Timestamp time.Time
	Duration  time.Duration
}

// NewErrorDisplay creates a new error display
func NewErrorDisplay() *ErrorDisplay {
	return &ErrorDisplay{
		messages: []ErrorMessage{},
		maxSize:  10,
	}
}

// AddError adds an error message
func (ed *ErrorDisplay) AddError(text string, duration time.Duration) {
	ed.messages = append(ed.messages, ErrorMessage{
		Text:      text,
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// Keep only maxSize messages
	if len(ed.messages) > ed.maxSize {
		ed.messages = ed.messages[len(ed.messages)-ed.maxSize:]
	}
}

// ClearExpired removes expired messages
func (ed *ErrorDisplay) ClearExpired() {
	now := time.Now()
	var active []ErrorMessage

	for _, msg := range ed.messages {
		if msg.Duration == 0 || now.Sub(msg.Timestamp) < msg.Duration {
			active = append(active, msg)
		}
	}

	ed.messages = active
}

// GetLatest returns the most recent error message
func (ed *ErrorDisplay) GetLatest() *ErrorMessage {
	ed.ClearExpired()
	if len(ed.messages) == 0 {
		return nil
	}
	return &ed.messages[len(ed.messages)-1]
}

// HasErrors returns true if there are active errors
func (ed *ErrorDisplay) HasErrors() bool {
	ed.ClearExpired()
	return len(ed.messages) > 0
}

// RenderToast renders a single error message as a toast notification
func (ed *ErrorDisplay) RenderToast(width int) string {
	latest := ed.GetLatest()
	if latest == nil {
		return ""
	}

	// Truncate message if needed
	msg := latest.Text
	if len(msg) > width-4 {
		msg = msg[:width-7] + "..."
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).  // Red
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(0, 1).
		Width(width - 2)

	return style.Render("⚠ " + msg)
}

// RenderList renders all active error messages
func (ed *ErrorDisplay) RenderList(width, maxHeight int) string {
	ed.ClearExpired()

	if len(ed.messages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("Recent Errors"))
	sb.WriteString("\n")

	count := len(ed.messages)
	if count > maxHeight {
		count = maxHeight
	}

	for i := len(ed.messages) - count; i < len(ed.messages); i++ {
		msg := ed.messages[i]
		text := msg.Text
		if len(text) > width-5 {
			text = text[:width-8] + "..."
		}
		sb.WriteString("  • ")
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	return sb.String()
}

// Clear removes all messages
func (ed *ErrorDisplay) Clear() {
	ed.messages = []ErrorMessage{}
}
