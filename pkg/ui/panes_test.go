package ui

import (
	"strings"
	"testing"
)

func TestLogListPane(t *testing.T) {
	tests := []struct {
		name       string
		action     func(*LogListPane)
		checkValue func(*LogListPane) bool
	}{
		{
			name: "initial scroll offset is 0",
			action: func(lp *LogListPane) {},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 0
			},
		},
		{
			name: "scroll down increments offset",
			action: func(lp *LogListPane) {
				lp.ScrollDown()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 1
			},
		},
		{
			name: "scroll up decrements offset",
			action: func(lp *LogListPane) {
				lp.ScrollDown()
				lp.ScrollDown()
				lp.ScrollUp()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 1
			},
		},
		{
			name: "scroll up at top doesn't go negative",
			action: func(lp *LogListPane) {
				lp.ScrollUp()
				lp.ScrollUp()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 0
			},
		},
		{
			name: "page down adds 10 to offset",
			action: func(lp *LogListPane) {
				lp.PageDown()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 10
			},
		},
		{
			name: "page up subtracts 10 from offset",
			action: func(lp *LogListPane) {
				lp.PageDown()
				lp.PageDown()
				lp.PageUp()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 10
			},
		},
		{
			name: "jump to top sets offset to 0",
			action: func(lp *LogListPane) {
				lp.PageDown()
				lp.JumpToTop()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset == 0
			},
		},
		{
			name: "jump to bottom sets offset high",
			action: func(lp *LogListPane) {
				lp.JumpToBottom()
			},
			checkValue: func(lp *LogListPane) bool {
				return lp.scrollOffset > 0
			},
		},
		{
			name: "render produces output",
			action: func(lp *LogListPane) {},
			checkValue: func(lp *LogListPane) bool {
				output := lp.Render(80, 20)
				return len(output) > 0 && strings.Contains(output, "Logs")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewLogListPane()
			tt.action(lp)
			if !tt.checkValue(lp) {
				t.Errorf("Check failed for: %s (scrollOffset=%d)", tt.name, lp.scrollOffset)
			}
		})
	}
}

func TestQueryPane(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		checkValue func(*QueryPane) bool
	}{
		{
			name:  "empty input initially",
			input: "",
			checkValue: func(qp *QueryPane) bool {
				return qp.GetInput() == ""
			},
		},
		{
			name:  "set and get input",
			input: "severity=ERROR",
			checkValue: func(qp *QueryPane) bool {
				return qp.GetInput() == "severity=ERROR"
			},
		},
		{
			name:  "render produces output",
			input: "test",
			checkValue: func(qp *QueryPane) bool {
				output := qp.Render(80, 4)
				return len(output) > 0 && strings.Contains(output, "Query")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qp := NewQueryPane()
			qp.SetInput(tt.input)
			if !tt.checkValue(qp) {
				t.Errorf("Check failed for: %s (input=%s)", tt.name, tt.input)
			}
		})
	}
}

func TestGraphPane(t *testing.T) {
	gp := NewGraphPane()

	if len(gp.dataPoints) != 0 {
		t.Errorf("Expected empty data points initially, got %d", len(gp.dataPoints))
	}

	gp.SetDataPoints([]int{1, 2, 3, 4, 5})
	if len(gp.dataPoints) != 5 {
		t.Errorf("Expected 5 data points, got %d", len(gp.dataPoints))
	}

	output := gp.Render(40, 15)
	if len(output) == 0 {
		t.Error("Render produced no output")
	}

	if !strings.Contains(output, "Timeline") {
		t.Error("Render should contain 'Timeline'")
	}
}

func TestControlsPane(t *testing.T) {
	cp := NewControlsPane()

	output := cp.Render(40, 10)
	if len(output) == 0 {
		t.Error("Render produced no output")
	}

	if !strings.Contains(output, "Controls") {
		t.Error("Render should contain 'Controls'")
	}

	if !strings.Contains(output, "q - Quit") {
		t.Error("Render should contain keybindings")
	}
}

func TestPanes(t *testing.T) {
	tests := []struct {
		name          string
		action        func(*Panes)
		expectedFocus string
	}{
		{
			name: "initial focus is logs",
			action: func(p *Panes) {},
			expectedFocus: "logs",
		},
		{
			name: "focus next moves to query",
			action: func(p *Panes) {
				p.FocusNext()
			},
			expectedFocus: "query",
		},
		{
			name: "focus next cycles through panes",
			action: func(p *Panes) {
				p.FocusNext() // query
				p.FocusNext() // graph
				p.FocusNext() // controls
				p.FocusNext() // logs (cycle)
			},
			expectedFocus: "logs",
		},
		{
			name: "focus previous cycles backward",
			action: func(p *Panes) {
				p.FocusPrevious()
			},
			expectedFocus: "controls",
		},
		{
			name: "set focus by name",
			action: func(p *Panes) {
				p.SetFocus("graph")
			},
			expectedFocus: "graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panes := NewPanes()
			tt.action(panes)
			if panes.GetFocusedPane() != tt.expectedFocus {
				t.Errorf("Expected focus '%s', got '%s'", tt.expectedFocus, panes.GetFocusedPane())
			}
		})
	}
}

func TestLogListPaneRender(t *testing.T) {
	lp := NewLogListPane()

	tests := []struct {
		width  int
		height int
		name   string
	}{
		{40, 10, "small pane"},
		{80, 20, "medium pane"},
		{120, 30, "large pane"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := lp.Render(tt.width, tt.height)
			lines := strings.Split(output, "\n")
			
			if len(lines) == 0 {
				t.Error("Render produced no lines")
			}

			if !strings.Contains(output, "Logs") {
				t.Error("Output should contain 'Logs' title")
			}
		})
	}
}
