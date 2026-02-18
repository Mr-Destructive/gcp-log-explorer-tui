package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpModal provides an interactive help interface
type HelpModal struct {
	visible bool
	width   int
	height  int
}

// NewHelpModal creates a new help modal
func NewHelpModal() *HelpModal {
	return &HelpModal{
		visible: false,
		width:   80,
		height:  24,
	}
}

// SetVisible toggles visibility
func (hm *HelpModal) SetVisible(visible bool) {
	hm.visible = visible
}

// IsVisible returns current visibility state
func (hm *HelpModal) IsVisible() bool {
	return hm.visible
}

// Render renders the help modal
func (hm *HelpModal) Render(width, height int) string {
	if !hm.visible {
		return ""
	}

	hm.width = width
	hm.height = height

	// Create content
	content := hm.getHelpContent()

	// Style the modal
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1).
		Width(width - 4)

	// Add semi-transparent background effect (via styling)
	return style.Render(content)
}

// getHelpContent returns the help text
func (hm *HelpModal) getHelpContent() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("LOG EXPLORER TUI HELP"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Efficient shortcuts reference"))
	sb.WriteString("\n\n")

	rows := [][3]string{
		{"Query", "q", "Open query editor"},
		{"Query", "enter", "Run query in editor"},
		{"Query", "ctrl+n", "Insert newline in query"},
		{"Query", "ctrl+a", "Select all query text"},
		{"Query", "ctrl+/", "Toggle comment on current line"},
		{"Query", "ctrl+left/right", "Move by word"},
		{"Query", "ctrl+w / ctrl+delete", "Delete previous / next word"},
		{"Query", "ctrl+r / ctrl+g", "Open query history popup (prev/next)"},
		{"Query", "ctrl+s", "Save current query to query library"},
		{"Query", "ctrl+y", "Open query library picker"},
		{"Query", "-- or #", "Comment line in query editor"},
		{"Filter", "t", "Time range filter"},
		{"Filter", "f", "Severity filter"},
		{"Filter", "m", "Toggle stream mode"},
		{"Filter", "ctrl+a", "Toggle auto-load all pages"},
		{"Nav", "j / k", "Scroll logs (edge triggers paging)"},
		{"Nav", "g / G", "Top / bottom"},
		{"Nav", "ctrl+f / ctrl+b", "Page down / up"},
		{"Log", "enter / ctrl+d", "Toggle details panel"},
		{"Log", "ctrl+p", "Full log popup"},
		{"Log", "v / tab", "Cycle popup view (full, payload tree, raw)"},
		{"Log", "h / l", "Collapse / expand payload JSON node"},
		{"Log", "z / Z", "Collapse all / expand all payload JSON nodes"},
		{"Log", "y / Y", "Copy selected JSON node / full payload"},
		{"Log", "ctrl+e", "Open payload only in $EDITOR (select/copy)"},
		{"Log", "ctrl+o", "Open selected log in $EDITOR"},
		{"Log", "ctrl+l", "Open loaded logs (JSON) in $EDITOR"},
		{"Log", "ctrl+shift+l / alt+l", "Open loaded logs (CSV) in $EDITOR"},
		{"Other", "e", "Export logs"},
		{"Other", "s", "Generate share link"},
		{"Other", "P", "Project selector"},
		{"Other", "L", "Open query library"},
		{"Other", "r", "Rerun query (bypass cache)"},
		{"Other", "f6", "Toggle key mode (vim/standard)"},
		{"Other", "f7", "Toggle timezone (UTC/local)"},
		{"Other", "esc", "Close active modal"},
		{"Other", "?", "Toggle help"},
		{"Other", "ctrl+c", "Quit"},
	}

	sb.WriteString(hm.renderResponsiveTable(rows))

	return sb.String()
}

func (hm *HelpModal) renderResponsiveTable(rows [][3]string) string {
	var sb strings.Builder
	contentWidth := hm.width - 10
	if contentWidth < 56 {
		contentWidth = 56
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	if contentWidth >= 90 {
		groupWidth := 8
		keyWidth := 24
		actionWidth := contentWidth - groupWidth - keyWidth - 6
		sb.WriteString(headerStyle.Render(fmt.Sprintf("%-*s %-*s %-*s\n", groupWidth, "GROUP", keyWidth, "KEY", actionWidth, "ACTION")))
		sb.WriteString(separatorStyle.Render(strings.Repeat("─", groupWidth+keyWidth+actionWidth+2)))
		sb.WriteString("\n")
		for _, row := range rows {
			actionLines := wrapWords(row[2], actionWidth)
			for i, line := range actionLines {
				groupCell := ""
				keyCell := ""
				if i == 0 {
					groupCell = row[0]
					keyCell = row[1]
				}
				sb.WriteString(fmt.Sprintf("%-*s %-*s %-*s\n", groupWidth, groupCell, keyWidth, keyCell, actionWidth, line))
			}
		}
		return sb.String()
	}

	keyWidth := 22
	actionWidth := contentWidth - keyWidth - 4
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%-*s %-*s\n", keyWidth, "KEY", actionWidth, "ACTION")))
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", keyWidth+actionWidth+1)))
	sb.WriteString("\n")
	for _, row := range rows {
		label := row[1]
		actionLines := wrapWords(row[2], actionWidth)
		for i, line := range actionLines {
			keyCell := ""
			if i == 0 {
				keyCell = label
			}
			sb.WriteString(fmt.Sprintf("%-*s %-*s\n", keyWidth, keyCell, actionWidth, line))
		}
	}
	return sb.String()
}

func wrapWords(input string, width int) []string {
	if width < 8 {
		return []string{input}
	}
	words := strings.Fields(input)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, 2)
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= width {
			current += " " + w
			continue
		}
		lines = append(lines, current)
		current = w
	}
	lines = append(lines, current)
	return lines
}

// GetShortHelp returns a quick reference string
func (hm *HelpModal) GetShortHelp() string {
	return "? help | arrows nav | q query | enter details | ctrl+p popup | f6 key mode | ctrl+c quit"
}
