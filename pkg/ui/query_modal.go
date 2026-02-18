package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// QueryModal handles query/filter editing
type QueryModal struct {
	visible     bool
	editor      textarea.Model
	selectAll   bool
	suggestions []string
}

// NewQueryModal creates a new query modal
func NewQueryModal() *QueryModal {
	ed := textarea.New()
	ed.Prompt = ""
	ed.Placeholder = ""
	ed.ShowLineNumbers = false
	ed.SetWidth(120)
	ed.SetHeight(8)
	ed.Focus()

	return &QueryModal{
		visible:   false,
		editor:    ed,
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
	qm.editor.Focus()
}

// Hide hides the modal
func (qm *QueryModal) Hide() {
	qm.visible = false
	qm.editor.Blur()
}

// IsVisible returns if modal is shown
func (qm *QueryModal) IsVisible() bool {
	return qm.visible
}

// SelectAllActive returns whether full-text selection is active.
func (qm *QueryModal) SelectAllActive() bool {
	return qm.selectAll
}

// HandleKey processes keyboard input
func (qm *QueryModal) HandleKey(key string) {
	if qm.selectAll {
		switch key {
		case "left", "right", "up", "down", "home", "end", "line-home", "line-end", "word-left", "word-right":
			qm.selectAll = false
		case "backspace", "delete":
			qm.editor.SetValue("")
			qm.editor.CursorStart()
			qm.selectAll = false
			return
		default:
			if key == "newline" || len(key) == 1 {
				qm.editor.SetValue("")
				qm.editor.CursorStart()
				qm.selectAll = false
			}
		}
	}

	switch key {
	case "newline":
		qm.editor.InsertString("\n")
		qm.selectAll = false
	case "backspace":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyBackspace})
		qm.selectAll = false
	case "delete":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyDelete})
		qm.selectAll = false
	case "left":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyLeft})
	case "right":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyRight})
	case "up":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyUp})
	case "down":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyDown})
	case "home":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyHome})
	case "end":
		qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyEnd})
	case "line-home":
		qm.setCursorIndex(qm.currentLineStart())
	case "line-end":
		qm.setCursorIndex(qm.currentLineEnd())
	case "select-all":
		qm.selectAll = true
		qm.editor.CursorEnd()
	case "word-left":
		qm.setCursorIndex(qm.wordLeft(qm.currentCursorIndex()))
	case "word-right":
		qm.setCursorIndex(qm.wordRight(qm.currentCursorIndex()))
	case "delete-word-left":
		val := qm.editor.Value()
		cursor := qm.currentCursorIndex()
		start := qm.wordLeft(cursor)
		if start < cursor {
			val = val[:start] + val[cursor:]
			qm.editor.SetValue(val)
			qm.setCursorIndex(start)
		}
	case "delete-word-right":
		val := qm.editor.Value()
		cursor := qm.currentCursorIndex()
		end := qm.wordRight(cursor)
		if end > cursor {
			val = val[:cursor] + val[end:]
			qm.editor.SetValue(val)
			qm.setCursorIndex(cursor)
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
		if len([]rune(key)) == 1 {
			qm.applyKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			qm.selectAll = false
		}
	}
}

func (qm *QueryModal) applyKeyMsg(msg tea.KeyMsg) {
	model, _ := qm.editor.Update(msg)
	qm.editor = model
}

// GetInput returns the current query input
func (qm *QueryModal) GetInput() string {
	return qm.editor.Value()
}

// GetInputWithCursor returns input with a visible cursor marker at current position.
func (qm *QueryModal) GetInputWithCursor() string {
	input := qm.editor.Value()
	idx := qm.currentCursorIndex()
	if idx < 0 {
		idx = 0
	}
	if idx > len(input) {
		idx = len(input)
	}
	return input[:idx] + "│" + input[idx:]
}

// SetInput sets the query input
func (qm *QueryModal) SetInput(input string) {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	qm.editor.SetValue(input)
	qm.editor.CursorEnd()
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
	sb.WriteString("┏━━ QUERY EDITOR " + strings.Repeat("━", width-18) + "\n")
	sb.WriteString("┃ Filter:\n")
	for _, ln := range wrapMultiline(qm.GetInputWithCursor(), maxInt(20, width-8), 6) {
		sb.WriteString(fmt.Sprintf("┃   %s\n", ln))
	}
	sb.WriteString("┣" + strings.Repeat("━", width-1) + "\n")
	sb.WriteString("┃ Suggestions:\n")
	for i, sugg := range qm.suggestions {
		if i >= 4 {
			break
		}
		sb.WriteString("┃   • " + sugg + "\n")
	}
	sb.WriteString("┣" + strings.Repeat("━", width-1) + "\n")
	if qm.selectAll {
		sb.WriteString("┃ Selection: all query text\n")
	}
	sb.WriteString("┃ Enter run | Ctrl+A all | Ctrl+/ comment | Ctrl+R history | Ctrl+left/right word\n")
	return sb.String()
}

// Clear clears the input
func (qm *QueryModal) Clear() {
	qm.editor.SetValue("")
	qm.editor.CursorStart()
	qm.selectAll = false
}

func (qm *QueryModal) currentCursorIndex() int {
	val := qm.editor.Value()
	if val == "" {
		return 0
	}
	line := qm.editor.Line()
	col := qm.editor.LineInfo().CharOffset
	lines := strings.Split(val, "\n")
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	return indexAtLineCol(lines, line, col)
}

func (qm *QueryModal) setCursorIndex(cursor int) {
	val := qm.editor.Value()
	line, col := lineColAt(val, cursor)
	qm.editor.CursorStart()
	for i := 0; i < line; i++ {
		qm.editor.CursorDown()
	}
	qm.editor.SetCursor(col)
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
	cursor := qm.currentCursorIndex()
	if cursor <= 0 {
		return 0
	}
	before := qm.editor.Value()[:cursor]
	idx := strings.LastIndex(before, "\n")
	if idx < 0 {
		return 0
	}
	return idx + 1
}

func (qm *QueryModal) currentLineEnd() int {
	cursor := qm.currentCursorIndex()
	input := qm.editor.Value()
	if cursor >= len(input) {
		return len(input)
	}
	after := input[cursor:]
	rel := strings.Index(after, "\n")
	if rel < 0 {
		return len(input)
	}
	return cursor + rel
}

func (qm *QueryModal) wordLeft(pos int) int {
	input := qm.editor.Value()
	if pos <= 0 {
		return 0
	}
	i := pos - 1
	for i >= 0 && isWordDelimiter(input[i]) {
		i--
	}
	for i >= 0 && !isWordDelimiter(input[i]) {
		i--
	}
	return i + 1
}

func (qm *QueryModal) wordRight(pos int) int {
	input := qm.editor.Value()
	if pos >= len(input) {
		return len(input)
	}
	i := pos
	for i < len(input) && isWordDelimiter(input[i]) {
		i++
	}
	for i < len(input) && !isWordDelimiter(input[i]) {
		i++
	}
	return i
}

func isWordDelimiter(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '(', ')', '{', '}', '[', ']', ':', ',', '=', '"', '\'', '|':
		return true
	default:
		return false
	}
}

func (qm *QueryModal) toggleCommentOnCurrentLine() {
	input := qm.editor.Value()
	cursor := qm.currentCursorIndex()
	line, col := lineColAt(input, cursor)
	lines := strings.Split(input, "\n")
	if line < 0 || line >= len(lines) {
		return
	}

	current := lines[line]
	trimmed := strings.TrimLeft(current, " ")
	indent := len(current) - len(trimmed)
	commentAdded := false
	commentRemoved := false

	switch {
	case strings.HasPrefix(trimmed, "-- "):
		trimmed = strings.TrimPrefix(trimmed, "-- ")
		commentRemoved = true
	case strings.HasPrefix(trimmed, "# "):
		trimmed = strings.TrimPrefix(trimmed, "# ")
		commentRemoved = true
	case strings.HasPrefix(trimmed, "--"):
		trimmed = strings.TrimPrefix(trimmed, "--")
		trimmed = strings.TrimPrefix(trimmed, " ")
		commentRemoved = true
	case strings.HasPrefix(trimmed, "#"):
		trimmed = strings.TrimPrefix(trimmed, "#")
		trimmed = strings.TrimPrefix(trimmed, " ")
		commentRemoved = true
	default:
		trimmed = "-- " + trimmed
		commentAdded = true
	}

	lines[line] = strings.Repeat(" ", indent) + trimmed
	out := strings.Join(lines, "\n")
	newCol := col
	if commentAdded && col >= indent {
		newCol += 3
	}
	if commentRemoved {
		if col >= indent+3 {
			newCol -= 3
		} else if col > indent {
			newCol = indent
		}
	}
	qm.editor.SetValue(out)
	qm.setCursorIndex(indexAtLineCol(strings.Split(out, "\n"), line, newCol))
}

func (qm *QueryModal) duplicateCurrentLine() {
	input := qm.editor.Value()
	cursor := qm.currentCursorIndex()
	line, col := lineColAt(input, cursor)
	lines := strings.Split(input, "\n")
	if line < 0 || line >= len(lines) {
		return
	}

	dup := lines[line]
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:line+1]...)
	newLines = append(newLines, dup)
	if line+1 < len(lines) {
		newLines = append(newLines, lines[line+1:]...)
	}
	out := strings.Join(newLines, "\n")
	qm.editor.SetValue(out)
	qm.setCursorIndex(indexAtLineCol(newLines, line+1, col))
}

func (qm *QueryModal) indentCurrentLine() {
	input := qm.editor.Value()
	cursor := qm.currentCursorIndex()
	line, col := lineColAt(input, cursor)
	lines := strings.Split(input, "\n")
	if line < 0 || line >= len(lines) {
		return
	}
	lines[line] = "  " + lines[line]
	out := strings.Join(lines, "\n")
	qm.editor.SetValue(out)
	qm.setCursorIndex(indexAtLineCol(lines, line, col+2))
}

func (qm *QueryModal) unindentCurrentLine() {
	input := qm.editor.Value()
	cursor := qm.currentCursorIndex()
	line, col := lineColAt(input, cursor)
	lines := strings.Split(input, "\n")
	if line < 0 || line >= len(lines) {
		return
	}
	remove := 0
	for remove < 2 && remove < len(lines[line]) && lines[line][remove] == ' ' {
		remove++
	}
	if remove > 0 {
		lines[line] = lines[line][remove:]
		if col >= remove {
			col -= remove
		} else {
			col = 0
		}
	}
	out := strings.Join(lines, "\n")
	qm.editor.SetValue(out)
	qm.setCursorIndex(indexAtLineCol(lines, line, col))
}
