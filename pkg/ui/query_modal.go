package ui

import (
	"fmt"
	"strings"
)

// QueryModal handles query/filter editing
type QueryModal struct {
	visible     bool
	input       string
	cursorPos   int
	selectAll   bool
	suggestions []string
}

// NewQueryModal creates a new query modal
func NewQueryModal() *QueryModal {
	return &QueryModal{
		visible:   false,
		input:     "",
		cursorPos: 0,
		selectAll: false,
		suggestions: []string{
			"severity=ERROR",
			"severity=WARNING",
			"severity>=ERROR",
			"resource.type=cloud_run",
			"resource.type=cloud_function",
			"labels.env=prod",
		},
	}
}

// Show displays the modal
func (qm *QueryModal) Show() {
	qm.visible = true
}

// Hide hides the modal
func (qm *QueryModal) Hide() {
	qm.visible = false
}

// IsVisible returns if modal is shown
func (qm *QueryModal) IsVisible() bool {
	return qm.visible
}

// HandleKey processes keyboard input
func (qm *QueryModal) HandleKey(key string) {
	if qm.selectAll {
		switch key {
		case "left", "right", "up", "down", "home", "end", "line-home", "line-end", "word-left", "word-right":
			qm.selectAll = false
		case "backspace", "delete":
			qm.input = ""
			qm.cursorPos = 0
			qm.selectAll = false
			return
		default:
			if key == "newline" || key == "\r" || len(key) == 1 {
				qm.input = ""
				qm.cursorPos = 0
				qm.selectAll = false
			}
		}
	}

	switch key {
	case "\r":
		// Normalize CR from CRLF clipboard pastes.
		qm.input = qm.input[:qm.cursorPos] + "\n" + qm.input[qm.cursorPos:]
		qm.cursorPos++
	case "newline":
		qm.input = qm.input[:qm.cursorPos] + "\n" + qm.input[qm.cursorPos:]
		qm.cursorPos++
	case "backspace":
		if qm.cursorPos > 0 {
			qm.input = qm.input[:qm.cursorPos-1] + qm.input[qm.cursorPos:]
			qm.cursorPos--
		}
	case "delete":
		if qm.cursorPos < len(qm.input) {
			qm.input = qm.input[:qm.cursorPos] + qm.input[qm.cursorPos+1:]
		}
	case "left":
		if qm.cursorPos > 0 {
			qm.cursorPos--
		}
	case "right":
		if qm.cursorPos < len(qm.input) {
			qm.cursorPos++
		}
	case "up":
		qm.moveVertical(-1)
	case "down":
		qm.moveVertical(1)
	case "home":
		qm.cursorPos = 0
	case "end":
		qm.cursorPos = len(qm.input)
	case "line-home":
		qm.cursorPos = qm.currentLineStart()
		qm.selectAll = false
	case "line-end":
		qm.cursorPos = qm.currentLineEnd()
		qm.selectAll = false
	case "select-all":
		qm.cursorPos = len(qm.input)
		qm.selectAll = true
	case "word-left":
		qm.cursorPos = qm.wordLeft(qm.cursorPos)
		qm.selectAll = false
	case "word-right":
		qm.cursorPos = qm.wordRight(qm.cursorPos)
		qm.selectAll = false
	case "delete-word-left":
		start := qm.wordLeft(qm.cursorPos)
		if start < qm.cursorPos {
			qm.input = qm.input[:start] + qm.input[qm.cursorPos:]
			qm.cursorPos = start
		}
	case "delete-word-right":
		end := qm.wordRight(qm.cursorPos)
		if end > qm.cursorPos {
			qm.input = qm.input[:qm.cursorPos] + qm.input[end:]
		}
	case "toggle-comment":
		qm.toggleCommentOnCurrentLine()
	case "duplicate-line":
		qm.duplicateCurrentLine()
	case "indent":
		qm.indentCurrentLine()
	case "unindent":
		qm.unindentCurrentLine()
	default:
		// Handle regular character input
		if len(key) == 1 {
			qm.input = qm.input[:qm.cursorPos] + key + qm.input[qm.cursorPos:]
			qm.cursorPos++
			qm.selectAll = false
		}
	}
}

// GetInput returns the current query input
func (qm *QueryModal) GetInput() string {
	return qm.input
}

// GetInputWithCursor returns input with a visible cursor marker at current position.
func (qm *QueryModal) GetInputWithCursor() string {
	if qm.cursorPos <= len(qm.input) {
		return qm.input[:qm.cursorPos] + "│" + qm.input[qm.cursorPos:]
	}
	return qm.input + "│"
}

// SetInput sets the query input
func (qm *QueryModal) SetInput(input string) {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	qm.input = input
	qm.cursorPos = len(input)
	qm.selectAll = false
}

// SetSuggestions replaces the suggestion list shown in the query editor.
func (qm *QueryModal) SetSuggestions(suggestions []string) {
	if len(suggestions) == 0 {
		return
	}
	qm.suggestions = append([]string{}, suggestions...)
}

// Render renders the modal
func (qm *QueryModal) Render(width, height int) string {
	if !qm.visible {
		return ""
	}

	var sb strings.Builder

	// Title
	sb.WriteString("┏━━ QUERY EDITOR " + strings.Repeat("━", width-18) + "\n")

	// Input line with cursor
	inputDisplay := qm.input
	if qm.cursorPos <= len(inputDisplay) {
		inputDisplay = qm.input[:qm.cursorPos] + "│" + qm.input[qm.cursorPos:]
	} else {
		inputDisplay = qm.input + "│"
	}
	lines := strings.Split(inputDisplay, "\n")
	maxInputLines := 6
	sb.WriteString("┃ Filter:\n")
	for i, ln := range lines {
		if i >= maxInputLines {
			sb.WriteString("┃   ...\n")
			break
		}
		if len(ln) > width-8 {
			ln = ln[:width-11] + "..."
		}
		sb.WriteString(fmt.Sprintf("┃   %s\n", ln))
	}

	// Separator
	sb.WriteString("┣" + strings.Repeat("━", width-1) + "\n")

	// Suggestions
	sb.WriteString("┃ Suggestions:\n")
	for i, sugg := range qm.suggestions {
		if i >= 4 {
			break
		}
		sb.WriteString("┃   • " + sugg + "\n")
	}

	// Help
	sb.WriteString("┣" + strings.Repeat("━", width-1) + "\n")
	if qm.selectAll {
		sb.WriteString("┃ Selection: all query text\n")
	}
	sb.WriteString("┃ Enter run | Ctrl+A select all | Ctrl+/ comment | Ctrl+left/right word | Tab indent\n")

	return sb.String()
}

// Clear clears the input
func (qm *QueryModal) Clear() {
	qm.input = ""
	qm.cursorPos = 0
	qm.selectAll = false
}

func (qm *QueryModal) moveVertical(delta int) {
	currentLine, currentCol := lineColAt(qm.input, qm.cursorPos)
	targetLine := currentLine + delta
	if targetLine < 0 {
		targetLine = 0
	}

	lines := strings.Split(qm.input, "\n")
	if targetLine >= len(lines) {
		targetLine = len(lines) - 1
	}
	if targetLine < 0 {
		targetLine = 0
	}

	targetCol := currentCol
	if targetCol > len(lines[targetLine]) {
		targetCol = len(lines[targetLine])
	}
	qm.cursorPos = indexAtLineCol(lines, targetLine, targetCol)
}

func lineColAt(input string, cursor int) (int, int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(input) {
		cursor = len(input)
	}
	line := 0
	col := 0
	for i, r := range input {
		if i >= cursor {
			break
		}
		if r == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

func indexAtLineCol(lines []string, targetLine, targetCol int) int {
	if len(lines) == 0 {
		return 0
	}
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine >= len(lines) {
		targetLine = len(lines) - 1
	}
	if targetCol < 0 {
		targetCol = 0
	}
	if targetCol > len(lines[targetLine]) {
		targetCol = len(lines[targetLine])
	}

	idx := 0
	for i := 0; i < targetLine; i++ {
		idx += len(lines[i]) + 1
	}
	return idx + targetCol
}

func (qm *QueryModal) currentLineStart() int {
	if qm.cursorPos <= 0 {
		return 0
	}
	before := qm.input[:qm.cursorPos]
	idx := strings.LastIndex(before, "\n")
	if idx < 0 {
		return 0
	}
	return idx + 1
}

func (qm *QueryModal) currentLineEnd() int {
	if qm.cursorPos >= len(qm.input) {
		return len(qm.input)
	}
	rest := qm.input[qm.cursorPos:]
	idx := strings.Index(rest, "\n")
	if idx < 0 {
		return len(qm.input)
	}
	return qm.cursorPos + idx
}

func (qm *QueryModal) currentLineBounds() (int, int) {
	start := qm.currentLineStart()
	end := qm.currentLineEnd()
	return start, end
}

func (qm *QueryModal) toggleCommentOnCurrentLine() {
	start, end := qm.currentLineBounds()
	line := qm.input[start:end]
	trimmed := strings.TrimLeft(line, " \t")
	indentLen := len(line) - len(trimmed)
	indent := line[:indentLen]
	switch {
	case strings.HasPrefix(trimmed, "-- "):
		trimmed = strings.TrimPrefix(trimmed, "-- ")
	case strings.HasPrefix(trimmed, "--"):
		trimmed = strings.TrimPrefix(trimmed, "--")
	case strings.HasPrefix(trimmed, "# "):
		trimmed = strings.TrimPrefix(trimmed, "# ")
	case strings.HasPrefix(trimmed, "#"):
		trimmed = strings.TrimPrefix(trimmed, "#")
	default:
		trimmed = "-- " + trimmed
	}
	updated := indent + trimmed
	qm.input = qm.input[:start] + updated + qm.input[end:]
	if qm.cursorPos > end {
		qm.cursorPos += len(updated) - len(line)
	} else if qm.cursorPos >= start {
		qm.cursorPos = minInt(start+len(updated), len(qm.input))
	}
}

func (qm *QueryModal) duplicateCurrentLine() {
	start, end := qm.currentLineBounds()
	line := qm.input[start:end]
	insert := "\n" + line
	if end < len(qm.input) {
		insert = "\n" + line
	}
	qm.input = qm.input[:end] + insert + qm.input[end:]
	qm.cursorPos = end + len(insert)
}

func (qm *QueryModal) indentCurrentLine() {
	start, _ := qm.currentLineBounds()
	qm.input = qm.input[:start] + "  " + qm.input[start:]
	if qm.cursorPos >= start {
		qm.cursorPos += 2
	}
}

func (qm *QueryModal) unindentCurrentLine() {
	start, end := qm.currentLineBounds()
	line := qm.input[start:end]
	remove := 0
	if strings.HasPrefix(line, "  ") {
		remove = 2
	} else if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
		remove = 1
	}
	if remove == 0 {
		return
	}
	qm.input = qm.input[:start] + line[remove:] + qm.input[end:]
	if qm.cursorPos > start {
		qm.cursorPos = maxInt(start, qm.cursorPos-remove)
	}
}

func (qm *QueryModal) wordLeft(pos int) int {
	if pos <= 0 {
		return 0
	}
	i := pos
	for i > 0 && isWordBoundary(qm.input[i-1]) {
		i--
	}
	for i > 0 && !isWordBoundary(qm.input[i-1]) {
		i--
	}
	return i
}

func (qm *QueryModal) wordRight(pos int) int {
	if pos >= len(qm.input) {
		return len(qm.input)
	}
	i := pos
	for i < len(qm.input) && isWordBoundary(qm.input[i]) {
		i++
	}
	for i < len(qm.input) && !isWordBoundary(qm.input[i]) {
		i++
	}
	return i
}

func isWordBoundary(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' ||
		ch == '"' || ch == '\'' || ch == ',' || ch == ':' || ch == ';' || ch == '='
}
