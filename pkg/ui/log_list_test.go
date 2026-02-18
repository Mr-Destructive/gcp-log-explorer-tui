package ui

import (
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func createTestLog(idx int) models.LogEntry {
	return models.LogEntry{
		ID:        "id-" + string(rune(idx)),
		Timestamp: time.Now().Add(-time.Duration(idx) * time.Minute),
		Severity:  "INFO",
		Message:   "Test log message " + string(rune(48+idx)),
	}
}

func TestNewLogListView(t *testing.T) {
	llv := NewLogListView(20)

	if llv.viewportHeight != 20 {
		t.Errorf("Expected height 20, got %d", llv.viewportHeight)
	}

	if len(llv.logs) != 0 {
		t.Error("Should start with no logs")
	}

	if llv.GetLogCount() != 0 {
		t.Error("Count should be 0")
	}
}

func TestSetLogs(t *testing.T) {
	llv := NewLogListView(20)

	logs := make([]models.LogEntry, 5)
	for i := 0; i < 5; i++ {
		logs[i] = createTestLog(i)
	}

	llv.SetLogs(logs)

	if llv.GetLogCount() != 5 {
		t.Errorf("Expected 5 logs, got %d", llv.GetLogCount())
	}

	if llv.selectedIdx != 0 {
		t.Error("Should reset selected index")
	}

	if llv.scrollOffset != 0 {
		t.Error("Should reset scroll offset")
	}
}

func TestAddLogs(t *testing.T) {
	llv := NewLogListView(20)

	logs1 := []models.LogEntry{createTestLog(0), createTestLog(1)}
	llv.SetLogs(logs1)

	logs2 := []models.LogEntry{createTestLog(2), createTestLog(3)}
	llv.AddLogs(logs2)

	if llv.GetLogCount() != 4 {
		t.Errorf("Expected 4 logs after add, got %d", llv.GetLogCount())
	}
}

func TestGetVisibleLogs(t *testing.T) {
	llv := NewLogListView(5)

	logs := make([]models.LogEntry, 10)
	for i := 0; i < 10; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	visible := llv.GetVisibleLogs()
	if len(visible) != 5 {
		t.Errorf("Expected 5 visible logs, got %d", len(visible))
	}

	llv.ScrollDown()
	llv.ScrollDown()

	visible = llv.GetVisibleLogs()
	if len(visible) != 5 {
		t.Errorf("After scroll, expected 5 visible logs, got %d", len(visible))
	}
}

func TestScrollUp(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 20)
	for i := 0; i < 20; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.scrollOffset = 5
	llv.ScrollUp()

	if llv.scrollOffset != 4 {
		t.Errorf("Expected offset 4, got %d", llv.scrollOffset)
	}

	// Should not go below 0
	for i := 0; i < 10; i++ {
		llv.ScrollUp()
	}

	if llv.scrollOffset != 0 {
		t.Error("Should not scroll below 0")
	}
}

func TestScrollDown(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 20)
	for i := 0; i < 20; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.ScrollDown()
	if llv.scrollOffset != 1 {
		t.Errorf("Expected offset 1, got %d", llv.scrollOffset)
	}
}

func TestPageUp(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 50)
	for i := 0; i < 50; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.scrollOffset = 20
	llv.PageUp()

	if llv.scrollOffset != 10 {
		t.Errorf("Expected offset 10 after page up, got %d", llv.scrollOffset)
	}
}

func TestPageDown(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 50)
	for i := 0; i < 50; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.PageDown()

	if llv.scrollOffset != 10 {
		t.Errorf("Expected offset 10 after page down, got %d", llv.scrollOffset)
	}
}

func TestJumpToTop(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 50)
	for i := 0; i < 50; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.scrollOffset = 25
	llv.JumpToTop()

	if llv.scrollOffset != 0 {
		t.Errorf("Expected offset 0, got %d", llv.scrollOffset)
	}
}

func TestJumpToBottom(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 50)
	for i := 0; i < 50; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.JumpToBottom()
	maxScroll := llv.GetMaxScroll()

	if llv.scrollOffset != maxScroll {
		t.Errorf("Expected offset %d, got %d", maxScroll, llv.scrollOffset)
	}
}

func TestSearch(t *testing.T) {
	llv := NewLogListView(10)
	logs := []models.LogEntry{
		{Message: "Error in database"},
		{Message: "Info message"},
		{Message: "Another error"},
		{Message: "Debug line"},
	}
	llv.SetLogs(logs)

	llv.Search("error")

	if llv.GetLogCount() != 2 {
		t.Errorf("Expected 2 matches for 'error', got %d", llv.GetLogCount())
	}
}

func TestSearchCasInsensitive(t *testing.T) {
	llv := NewLogListView(10)
	logs := []models.LogEntry{
		{Message: "ERROR in system"},
		{Message: "Error in database"},
		{Message: "Info"},
	}
	llv.SetLogs(logs)

	llv.Search("ERROR")

	if llv.GetLogCount() != 2 {
		t.Errorf("Search should be case insensitive, expected 2, got %d", llv.GetLogCount())
	}
}

func TestClearSearch(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 5)
	for i := 0; i < 5; i++ {
		logs[i] = models.LogEntry{Message: "test"}
	}
	llv.SetLogs(logs)

	llv.Search("test")
	if llv.GetLogCount() != 5 {
		t.Errorf("Expected 5 after search, got %d", llv.GetLogCount())
	}

	llv.ClearSearch()

	if llv.searchTerm != "" {
		t.Error("Search term should be cleared")
	}

	if llv.GetLogCount() != 5 {
		t.Errorf("Expected 5 after clear, got %d", llv.GetLogCount())
	}
}

func TestSelectLog(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 5)
	for i := 0; i < 5; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	ok := llv.SelectLog(2)
	if !ok {
		t.Error("SelectLog should succeed")
	}

	if llv.selectedIdx != 2 {
		t.Errorf("Expected index 2, got %d", llv.selectedIdx)
	}

	ok = llv.SelectLog(10)
	if ok {
		t.Error("SelectLog should fail on invalid index")
	}
}

func TestGetSelectedLog(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 3)
	for i := 0; i < 3; i++ {
		logs[i] = createTestLog(i)
		logs[i].Message = "Log " + string(rune(48+i))
	}
	llv.SetLogs(logs)

	selected := llv.GetSelectedLog()
	if selected == nil {
		t.Error("Selected log should not be nil")
	}

	if selected.Message != "Log 0" {
		t.Errorf("Expected 'Log 0', got %s", selected.Message)
	}
}

func TestGetMaxScroll(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 25)
	for i := 0; i < 25; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	maxScroll := llv.GetMaxScroll()
	if maxScroll != 15 {
		t.Errorf("Expected max scroll 15, got %d", maxScroll)
	}
}

func TestClear(t *testing.T) {
	llv := NewLogListView(10)
	logs := make([]models.LogEntry, 5)
	for i := 0; i < 5; i++ {
		logs[i] = createTestLog(i)
	}
	llv.SetLogs(logs)

	llv.Clear()

	if len(llv.logs) != 0 {
		t.Error("Logs should be cleared")
	}

	if llv.GetLogCount() != 0 {
		t.Error("Count should be 0 after clear")
	}
}

func TestSetViewportHeight(t *testing.T) {
	llv := NewLogListView(10)
	llv.SetViewportHeight(20)

	if llv.viewportHeight != 20 {
		t.Errorf("Expected height 20, got %d", llv.viewportHeight)
	}
}

func TestGetVisibleLines(t *testing.T) {
	llv := NewLogListView(5)
	logs := make([]models.LogEntry, 10)
	for i := 0; i < 10; i++ {
		logs[i] = createTestLog(i)
		logs[i].Message = "Message " + string(rune(48+i))
	}
	llv.SetLogs(logs)

	lines := llv.GetVisibleLines(80)

	if len(lines) != 5 {
		t.Errorf("Expected 5 visible lines, got %d", len(lines))
	}

	for _, line := range lines {
		if len(line) == 0 {
			t.Error("Line should not be empty")
		}
	}
}
