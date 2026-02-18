package ui

import (
	"strings"

	"github.com/user/log-explorer-tui/pkg/models"
)

// LogListView manages the log list with virtual rendering
type LogListView struct {
	logs           []models.LogEntry
	selectedIdx    int
	scrollOffset   int
	viewportHeight int
	formatter      *LogFormatter
	searchTerm     string
	filteredIndices []int
}

// NewLogListView creates a new log list view
func NewLogListView(height int) *LogListView {
	return &LogListView{
		logs:            []models.LogEntry{},
		selectedIdx:     0,
		scrollOffset:    0,
		viewportHeight:  height,
		formatter:       NewLogFormatter(120, true),
		searchTerm:      "",
		filteredIndices: []int{},
	}
}

// SetLogs sets the list of logs to display
func (llv *LogListView) SetLogs(logs []models.LogEntry) {
	llv.logs = logs
	llv.selectedIdx = 0
	llv.scrollOffset = 0
	llv.ApplySearch()
}

// AddLogs appends logs to the list
func (llv *LogListView) AddLogs(logs []models.LogEntry) {
	llv.logs = append(llv.logs, logs...)
	llv.ApplySearch()
}

// GetVisibleLogs returns the logs visible in the current viewport
func (llv *LogListView) GetVisibleLogs() []models.LogEntry {
	if len(llv.filteredIndices) == 0 {
		if llv.scrollOffset >= len(llv.logs) {
			return []models.LogEntry{}
		}

		end := llv.scrollOffset + llv.viewportHeight
		if end > len(llv.logs) {
			end = len(llv.logs)
		}

		return llv.logs[llv.scrollOffset:end]
	}

	var visible []models.LogEntry
	start := llv.scrollOffset
	end := llv.scrollOffset + llv.viewportHeight

	if start >= len(llv.filteredIndices) {
		return visible
	}

	if end > len(llv.filteredIndices) {
		end = len(llv.filteredIndices)
	}

	for i := start; i < end; i++ {
		if llv.filteredIndices[i] < len(llv.logs) {
			visible = append(visible, llv.logs[llv.filteredIndices[i]])
		}
	}

	return visible
}

// GetVisibleLines returns formatted lines for the current viewport
func (llv *LogListView) GetVisibleLines(maxLen int) []string {
	var lines []string
	for _, log := range llv.GetVisibleLogs() {
		line := llv.formatter.FormatLogLine(log, maxLen)
		lines = append(lines, line)
	}
	return lines
}

// ScrollUp moves up in the log list
func (llv *LogListView) ScrollUp() {
	if llv.scrollOffset > 0 {
		llv.scrollOffset--
	}
}

// ScrollDown moves down in the log list
func (llv *LogListView) ScrollDown() {
	maxScroll := llv.GetMaxScroll()
	if llv.scrollOffset < maxScroll {
		llv.scrollOffset++
	}
}

// PageUp pages up
func (llv *LogListView) PageUp() {
	llv.scrollOffset -= llv.viewportHeight
	if llv.scrollOffset < 0 {
		llv.scrollOffset = 0
	}
}

// PageDown pages down
func (llv *LogListView) PageDown() {
	llv.scrollOffset += llv.viewportHeight
	maxScroll := llv.GetMaxScroll()
	if llv.scrollOffset > maxScroll {
		llv.scrollOffset = maxScroll
	}
}

// JumpToTop jumps to the beginning
func (llv *LogListView) JumpToTop() {
	llv.scrollOffset = 0
}

// JumpToBottom jumps to the end
func (llv *LogListView) JumpToBottom() {
	llv.scrollOffset = llv.GetMaxScroll()
}

// GetMaxScroll returns the maximum scroll offset
func (llv *LogListView) GetMaxScroll() int {
	totalLogs := len(llv.filteredIndices)
	if totalLogs == 0 {
		totalLogs = len(llv.logs)
	}

	maxScroll := totalLogs - llv.viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

// SetViewportHeight sets the height of the viewport
func (llv *LogListView) SetViewportHeight(height int) {
	llv.viewportHeight = height
}

// GetSelectedLog returns the currently selected log entry
func (llv *LogListView) GetSelectedLog() *models.LogEntry {
	visible := llv.GetVisibleLogs()
	if llv.selectedIdx >= len(visible) {
		return nil
	}
	return &visible[llv.selectedIdx]
}

// SelectLog selects a log by index (within visible logs)
func (llv *LogListView) SelectLog(idx int) bool {
	if idx < 0 || idx >= len(llv.GetVisibleLogs()) {
		return false
	}
	llv.selectedIdx = idx
	return true
}

// Search searches logs for a keyword
func (llv *LogListView) Search(term string) {
	llv.searchTerm = strings.ToLower(term)
	llv.ApplySearch()
}

// ApplySearch applies the current search term
func (llv *LogListView) ApplySearch() {
	llv.filteredIndices = []int{}

	if llv.searchTerm == "" {
		return
	}

	for i, log := range llv.logs {
		if strings.Contains(strings.ToLower(log.Message), llv.searchTerm) {
			llv.filteredIndices = append(llv.filteredIndices, i)
		}
	}

	llv.scrollOffset = 0
}

// ClearSearch clears the search filter
func (llv *LogListView) ClearSearch() {
	llv.searchTerm = ""
	llv.filteredIndices = []int{}
	llv.scrollOffset = 0
}

// GetSearchTerm returns the current search term
func (llv *LogListView) GetSearchTerm() string {
	return llv.searchTerm
}

// GetLogCount returns the total count of (filtered) logs
func (llv *LogListView) GetLogCount() int {
	if len(llv.filteredIndices) > 0 {
		return len(llv.filteredIndices)
	}
	return len(llv.logs)
}

// GetAllLogs returns all logs (unfiltered)
func (llv *LogListView) GetAllLogs() []models.LogEntry {
	return llv.logs
}

// Clear clears all logs
func (llv *LogListView) Clear() {
	llv.logs = []models.LogEntry{}
	llv.filteredIndices = []int{}
	llv.selectedIdx = 0
	llv.scrollOffset = 0
}

// GetFormattedDetails returns formatted details for a log
func (llv *LogListView) GetFormattedDetails(entry *models.LogEntry) string {
	if entry == nil {
		return "No log selected"
	}
	return llv.formatter.FormatLogDetails(*entry)
}
