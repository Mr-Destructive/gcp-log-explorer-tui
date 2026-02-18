package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panes manages all UI panes
type Panes struct {
	LogList  *LogListPane
	Query    *QueryPane
	Graph    *GraphPane
	Controls *ControlsPane
	focused  int // 0=logs, 1=query, 2=graph, 3=controls
}

// NewPanes creates all panes
func NewPanes() *Panes {
	return &Panes{
		LogList:  NewLogListPane(),
		Query:    NewQueryPane(),
		Graph:    NewGraphPane(),
		Controls: NewControlsPane(),
		focused:  0,
	}
}

// FocusPrevious moves focus to previous pane
func (p *Panes) FocusPrevious() {
	p.focused = (p.focused - 1) % 4
	if p.focused < 0 {
		p.focused = 3
	}
}

// FocusNext moves focus to next pane
func (p *Panes) FocusNext() {
	p.focused = (p.focused + 1) % 4
}

// SetFocus sets focus to a specific pane by name
func (p *Panes) SetFocus(paneName string) {
	switch paneName {
	case "logs":
		p.focused = 0
	case "query":
		p.focused = 1
	case "graph":
		p.focused = 2
	case "controls":
		p.focused = 3
	}
}

// GetFocusedPane returns the currently focused pane
func (p *Panes) GetFocusedPane() string {
	switch p.focused {
	case 0:
		return "logs"
	case 1:
		return "query"
	case 2:
		return "graph"
	case 3:
		return "controls"
	default:
		return "logs"
	}
}

// LogListPane represents the log list view
type LogListPane struct {
	scrollOffset int
	selectedIdx  int
	focused      bool
	logs         []string
}

// NewLogListPane creates a new log list pane
func NewLogListPane() *LogListPane {
	return &LogListPane{
		scrollOffset: 0,
		selectedIdx:  0,
		focused:      true,
		logs:         []string{},
	}
}

// ScrollUp moves up in the log list
func (lp *LogListPane) ScrollUp() {
	if lp.scrollOffset > 0 {
		lp.scrollOffset--
	}
}

// ScrollDown moves down in the log list
func (lp *LogListPane) ScrollDown() {
	lp.scrollOffset++
}

// PageUp pages up in the log list
func (lp *LogListPane) PageUp() {
	lp.scrollOffset -= 10
	if lp.scrollOffset < 0 {
		lp.scrollOffset = 0
	}
}

// PageDown pages down in the log list
func (lp *LogListPane) PageDown() {
	lp.scrollOffset += 10
}

// JumpToTop jumps to the top
func (lp *LogListPane) JumpToTop() {
	lp.scrollOffset = 0
}

// JumpToBottom jumps to the bottom
func (lp *LogListPane) JumpToBottom() {
	// Will be set based on actual log count
	lp.scrollOffset = 999999
}

// SetLogs updates the logs to display
func (lp *LogListPane) SetLogs(logs []string) {
	lp.logs = logs
	lp.scrollOffset = 0
	lp.selectedIdx = 0
}

// Render renders the log list pane
func (lp *LogListPane) Render(width, height int) string {
	corner := "├"
	if lp.focused {
		corner = "┣"
	}

	title := fmt.Sprintf(" Logs (%d) ", len(lp.logs))
	header := corner + "─" + title + "─"
	for i := len(header); i < width; i++ {
		header += "─"
	}

	// Render logs
	var content string
	if len(lp.logs) == 0 {
		content = "No logs loaded yet\n"
		for i := 1; i < height; i++ {
			content += "\n"
		}
	} else {
		// Show visible logs with scrolling
		visibleLogs := []string{}
		start := lp.scrollOffset
		end := lp.scrollOffset + height - 2

		if start >= len(lp.logs) {
			start = len(lp.logs) - 1
		}
		if start < 0 {
			start = 0
		}
		if end > len(lp.logs) {
			end = len(lp.logs)
		}

		for i := start; i < end; i++ {
			log := lp.logs[i]
			// Truncate log to fit width
			if len(log) > width-3 {
				log = log[:width-6] + "..."
			}
			visibleLogs = append(visibleLogs, log)
		}

		// Pad to height
		for len(visibleLogs) < height-1 {
			visibleLogs = append(visibleLogs, "")
		}

		content = strings.Join(visibleLogs, "\n") + "\n"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(false).
		Width(width - 1).
		Height(height - 1)

	if lp.focused {
		style = style.BorderStyle(lipgloss.ThickBorder())
	}

	return header + "\n" + style.Render(content)
}

// QueryPane represents the query input pane
type QueryPane struct {
	input   string
	focused bool
}

// NewQueryPane creates a new query pane
func NewQueryPane() *QueryPane {
	return &QueryPane{
		input:   "",
		focused: false,
	}
}

// SetInput sets the query input text
func (qp *QueryPane) SetInput(text string) {
	qp.input = text
}

// GetInput returns the query input text
func (qp *QueryPane) GetInput() string {
	return qp.input
}

// Render renders the query pane
func (qp *QueryPane) Render(width, height int) string {
	title := " Query "
	header := "├─" + title + "─"
	for i := len(header); i < width; i++ {
		header += "─"
	}

	content := qp.input
	if qp.focused {
		content += " |"
	}

	for len(content) < width-2 {
		content += " "
	}

	style := lipgloss.NewStyle().
		Width(width - 2).
		Height(height - 1)

	if qp.focused {
		style = style.BorderStyle(lipgloss.ThickBorder()).Border(lipgloss.NormalBorder())
	}

	return header + "\n" + style.Render(content)
}

// GraphPane represents the log graph pane
type GraphPane struct {
	dataPoints []int
	focused    bool
}

// NewGraphPane creates a new graph pane
func NewGraphPane() *GraphPane {
	return &GraphPane{
		dataPoints: []int{},
		focused:    false,
	}
}

// SetDataPoints sets the graph data
func (gp *GraphPane) SetDataPoints(points []int) {
	gp.dataPoints = points
}

// Render renders the graph pane
func (gp *GraphPane) Render(width, height int) string {
	title := " Timeline "
	header := "├─" + title + "─"
	for i := len(header); i < width; i++ {
		header += "─"
	}

	// Simple sparkline-like graph
	content := fmt.Sprintf("Logs over time\n")
	for i := 1; i < height-1; i++ {
		content += "│\n"
	}

	style := lipgloss.NewStyle().
		Width(width - 1).
		Height(height - 1)

	if gp.focused {
		style = style.BorderStyle(lipgloss.ThickBorder())
	}

	return header + "\n" + style.Render(content)
}

// ControlsPane represents the controls and info pane
type ControlsPane struct {
	focused bool
}

// NewControlsPane creates a new controls pane
func NewControlsPane() *ControlsPane {
	return &ControlsPane{
		focused: false,
	}
}

// Render renders the controls pane
func (cp *ControlsPane) Render(width, height int) string {
	title := " Controls "
	header := "├─" + title + "─"
	for i := len(header); i < width; i++ {
		header += "─"
	}

	content := `h/l - Panes
j/k - Scroll
? - Help
q - Quit`

	style := lipgloss.NewStyle().
		Width(width - 1).
		Height(height - 1).
		Padding(1)

	if cp.focused {
		style = style.BorderStyle(lipgloss.ThickBorder())
	}

	return header + "\n" + style.Render(content)
}
