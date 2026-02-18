package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpModal provides an interactive help interface
type HelpModal struct {
	visible        bool
	width          int
	height         int
	currentSection int
}

// NewHelpModal creates a new help modal
func NewHelpModal() *HelpModal {
	return &HelpModal{
		visible:        false,
		width:          80,
		height:         24,
		currentSection: 0,
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

	sections := hm.getSections()
	if len(sections) == 0 {
		return "No help sections available"
	}
	if hm.currentSection < 0 {
		hm.currentSection = 0
	}
	if hm.currentSection >= len(sections) {
		hm.currentSection = len(sections) - 1
	}
	section := sections[hm.currentSection]

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("LOG EXPLORER HELP"))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Compact, sectioned shortcuts"))
	sb.WriteString("\n\n")
	sb.WriteString(hm.renderTabs(sections))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(section.summary))
	sb.WriteString("\n\n")
	sb.WriteString(hm.renderResponsiveTable(section.rows))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Tab/←/→/h/l/j/k switch section | Esc/? close"))

	return sb.String()
}

type helpSection struct {
	title   string
	summary string
	rows    [][2]string
}

func (hm *HelpModal) getSections() []helpSection {
	return []helpSection{
		{
			title:   "Core",
			summary: "Everyday navigation and open/close actions",
			rows: [][2]string{
				{"q", "Open query editor"},
				{"j/k or ↑/↓", "Move through log list"},
				{"g / G", "Jump to first / last visible log"},
				{"Enter / Ctrl+D", "Toggle details panel"},
				{"Ctrl+P", "Open full log popup"},
				{"Esc / ?", "Close modal / help"},
			},
		},
		{
			title:   "Query",
			summary: "Browser-like query editing and saved queries",
			rows: [][2]string{
				{"Enter", "Run query in editor"},
				{"Ctrl+A", "Select all query text"},
				{"Ctrl+/", "Toggle comment line"},
				{"Ctrl+Left/Right", "Move by word"},
				{"Ctrl+W / Ctrl+Delete", "Delete previous / next word"},
				{"Ctrl+R/Ctrl+G", "Open query history popup"},
				{"Ctrl+S / Ctrl+Y", "Save query / open library"},
			},
		},
		{
			title:   "Logs",
			summary: "Inspect payloads and export selected/all logs",
			rows: [][2]string{
				{"v / Tab", "Cycle popup view modes"},
				{"h/l", "Collapse / expand JSON node"},
				{"z / Z", "Collapse all / expand all nodes"},
				{"y / Y", "Copy selected node / full payload"},
				{"Ctrl+E", "Open payload in $EDITOR"},
				{"Ctrl+O", "Open selected log in $EDITOR"},
				{"Ctrl+L / Alt+L", "Open loaded logs as JSON / CSV"},
			},
		},
		{
			title:   "Filters",
			summary: "Filtering, paging strategy, and stream controls",
			rows: [][2]string{
				{"t", "Time range filter"},
				{"f", "Severity filter"},
				{"m", "Toggle stream mode"},
				{"Ctrl+A", "Toggle auto-load all pages"},
				{"r", "Rerun query (bypass cache)"},
				{"F8", "Toggle log order bottom/top"},
			},
		},
		{
			title:   "System",
			summary: "Session and environment controls",
			rows: [][2]string{
				{"P", "Project selector popup"},
				{"L", "Open query library popup"},
				{"F6", "Open key mode dropdown"},
				{"F7", "Open timezone dropdown"},
				{"Ctrl+C", "Quit app"},
			},
		},
	}
}

func (hm *HelpModal) renderTabs(sections []helpSection) string {
	items := make([]string, 0, len(sections))
	for i, sec := range sections {
		if i == hm.currentSection {
			items = append(items, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1).Render(sec.title))
		} else {
			items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1).Render(sec.title))
		}
	}
	return strings.Join(items, " ")
}

func (hm *HelpModal) renderResponsiveTable(rows [][2]string) string {
	var sb strings.Builder
	contentWidth := hm.width - 10
	if contentWidth < 56 {
		contentWidth = 56
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	keyWidth := 24
	actionWidth := contentWidth - keyWidth - 4
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%-*s %-*s\n", keyWidth, "KEY", actionWidth, "ACTION")))
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", keyWidth+actionWidth+1)))
	sb.WriteString("\n")
	for _, row := range rows {
		label := row[0]
		actionLines := wrapWords(row[1], actionWidth)
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
	return "? help | arrows nav | q query | enter details | ctrl+p popup | f6/f7/f8 modes | ctrl+c quit"
}

func (hm *HelpModal) NextSection() {
	sections := hm.getSections()
	if len(sections) == 0 {
		hm.currentSection = 0
		return
	}
	hm.currentSection = (hm.currentSection + 1) % len(sections)
}

func (hm *HelpModal) PrevSection() {
	sections := hm.getSections()
	if len(sections) == 0 {
		hm.currentSection = 0
		return
	}
	hm.currentSection--
	if hm.currentSection < 0 {
		hm.currentSection = len(sections) - 1
	}
}
