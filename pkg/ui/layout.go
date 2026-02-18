package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHorizontalSplit renders multiple panes side-by-side
func renderHorizontalSplit(panes []string, widths []int) string {
	if len(panes) == 0 {
		return ""
	}

	// Split each pane into lines
	paneLines := make([][]string, len(panes))
	maxHeight := 0

	for i, pane := range panes {
		lines := strings.Split(pane, "\n")
		paneLines[i] = lines
		if len(lines) > maxHeight {
			maxHeight = len(lines)
		}
	}

	// Normalize heights
	for i := range paneLines {
		for len(paneLines[i]) < maxHeight {
			paneLines[i] = append(paneLines[i], "")
		}
	}

	// Combine lines
	var result []string
	for lineIdx := 0; lineIdx < maxHeight; lineIdx++ {
		var lineParts []string
		for paneIdx := range paneLines {
			line := paneLines[paneIdx][lineIdx]
			// Pad line to width
			width := widths[paneIdx]
			if len(line) < width {
				line += strings.Repeat(" ", width-len(line))
			} else if len(line) > width {
				line = line[:width]
			}
			lineParts = append(lineParts, line)
		}
		result = append(result, strings.Join(lineParts, ""))
	}

	return strings.Join(result, "\n")
}

// renderVerticalSplit renders multiple panes stacked vertically
func renderVerticalSplit(panes []string, heights []int) string {
	return strings.Join(panes, "\n")
}

// StyleBorder applies border styling to content
func StyleBorder(content string, width, height int, title string, focused bool) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	if focused {
		style = style.BorderStyle(lipgloss.ThickBorder())
	} else {
		style = style.BorderStyle(lipgloss.NormalBorder())
	}

	return style.Render(content)
}

// CreateHeader creates a pane header with title
func CreateHeader(title string, width int, focused bool) string {
	corner := "├"
	if focused {
		corner = "┣"
	}

	line := corner + "─" + title + "─"
	for i := len(line); i < width; i++ {
		line += "─"
	}
	return line
}
