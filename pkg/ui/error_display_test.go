package ui

import (
	"strings"
	"testing"
	"time"
)

// TestNewErrorDisplay tests error display creation
func TestNewErrorDisplay(t *testing.T) {
	ed := NewErrorDisplay()
	if ed == nil {
		t.Fatal("ErrorDisplay should not be nil")
	}
	if ed.HasErrors() {
		t.Error("New ErrorDisplay should have no errors")
	}
}

// TestAddError tests adding error messages
func TestAddError(t *testing.T) {
	ed := NewErrorDisplay()

	ed.AddError("Connection failed", 5*time.Second)
	if !ed.HasErrors() {
		t.Error("ErrorDisplay should have errors after adding")
	}

	latest := ed.GetLatest()
	if latest == nil {
		t.Fatal("Latest error should not be nil")
	}
	if latest.Text != "Connection failed" {
		t.Errorf("Expected 'Connection failed', got %s", latest.Text)
	}
}

// TestMultipleErrors tests adding multiple error messages
func TestMultipleErrors(t *testing.T) {
	ed := NewErrorDisplay()

	ed.AddError("Error 1", 5*time.Second)
	ed.AddError("Error 2", 5*time.Second)
	ed.AddError("Error 3", 5*time.Second)

	if ed.GetLatest().Text != "Error 3" {
		t.Error("Latest error should be Error 3")
	}
}

// TestClearExpired tests expiration of messages
func TestClearExpired(t *testing.T) {
	ed := NewErrorDisplay()

	// Add an expired message
	ed.messages = append(ed.messages, ErrorMessage{
		Text:      "Expired error",
		Timestamp: time.Now().Add(-10 * time.Second),
		Duration:  5 * time.Second,
	})

	// Add an active message
	ed.AddError("Active error", 10*time.Second)

	ed.ClearExpired()

	if len(ed.messages) != 1 {
		t.Errorf("Expected 1 message after clearing expired, got %d", len(ed.messages))
	}

	if ed.GetLatest().Text != "Active error" {
		t.Error("Only active error should remain")
	}
}

// TestRenderToast tests toast rendering
func TestRenderToast(t *testing.T) {
	ed := NewErrorDisplay()
	ed.AddError("Test error", 5*time.Second)

	toast := ed.RenderToast(50)
	if toast == "" {
		t.Fatal("Toast should not be empty")
	}

	if !strings.Contains(toast, "Test error") {
		t.Error("Toast should contain error message")
	}
}

// TestRenderToastEmpty tests rendering when no errors exist
func TestRenderToastEmpty(t *testing.T) {
	ed := NewErrorDisplay()
	toast := ed.RenderToast(50)

	if toast != "" {
		t.Error("Toast should be empty when no errors")
	}
}

// TestRenderToastTruncation tests message truncation
func TestRenderToastTruncation(t *testing.T) {
	ed := NewErrorDisplay()
	longError := strings.Repeat("X", 100)
	ed.AddError(longError, 5*time.Second)

	toast := ed.RenderToast(20)
	// Toast should be created (with styling it may be longer but the message itself is truncated)
	if toast == "" {
		t.Error("Toast should not be empty")
	}
	// Just verify it doesn't contain the full 100-char error untruncated
	if strings.Contains(toast, longError) {
		t.Error("Toast should truncate long messages")
	}
}

// TestRenderList tests list rendering
func TestRenderList(t *testing.T) {
	ed := NewErrorDisplay()
	ed.AddError("Error 1", 10*time.Second)
	ed.AddError("Error 2", 10*time.Second)
	ed.AddError("Error 3", 10*time.Second)

	list := ed.RenderList(50, 10)
	if list == "" {
		t.Fatal("List should not be empty")
	}

	if !strings.Contains(list, "Error 1") {
		t.Error("List should contain Error 1")
	}
	if !strings.Contains(list, "Error 2") {
		t.Error("List should contain Error 2")
	}
	if !strings.Contains(list, "Error 3") {
		t.Error("List should contain Error 3")
	}
}

// TestRenderListMaxHeight tests max height limit
func TestRenderListMaxHeight(t *testing.T) {
	ed := NewErrorDisplay()
	for i := 0; i < 10; i++ {
		ed.AddError("Error "+string(rune(i)), 10*time.Second)
	}

	list := ed.RenderList(50, 3)
	lines := strings.Count(list, "\n")

	if lines > 5 { // 3 errors + header + extra
		t.Error("List should respect max height limit")
	}
}

// TestClearErrors tests clearing all messages
func TestClearErrors(t *testing.T) {
	ed := NewErrorDisplay()
	ed.AddError("Error 1", 5*time.Second)
	ed.AddError("Error 2", 5*time.Second)

	ed.Clear()

	if ed.HasErrors() {
		t.Error("ErrorDisplay should have no errors after Clear")
	}
}

// TestPersistentError tests error with zero duration (persistent)
func TestPersistentError(t *testing.T) {
	ed := NewErrorDisplay()

	// Add persistent error (duration = 0)
	ed.AddError("Persistent error", 0)

	time.Sleep(100 * time.Millisecond)
	ed.ClearExpired()

	if !ed.HasErrors() {
		t.Error("Persistent error should remain after ClearExpired")
	}

	latest := ed.GetLatest()
	if latest.Text != "Persistent error" {
		t.Error("Persistent error should still be there")
	}
}

// TestMaxSizeLimit tests that message buffer doesn't exceed maxSize
func TestMaxSizeLimit(t *testing.T) {
	ed := NewErrorDisplay()
	ed.maxSize = 5

	// Add 10 messages
	for i := 0; i < 10; i++ {
		ed.AddError("Error", 10*time.Second)
	}

	if len(ed.messages) > ed.maxSize {
		t.Errorf("Expected at most %d messages, got %d", ed.maxSize, len(ed.messages))
	}
}
