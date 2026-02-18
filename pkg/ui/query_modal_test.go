package ui

import "testing"

func TestQueryModalToggleCommentAndWordEditing(t *testing.T) {
	qm := NewQueryModal()
	qm.SetInput("severity=ERROR")

	qm.HandleKey("line-home")
	qm.HandleKey("toggle-comment")
	if got := qm.GetInput(); got != "-- severity=ERROR" {
		t.Fatalf("expected commented line, got %q", got)
	}

	qm.HandleKey("toggle-comment")
	if got := qm.GetInput(); got != "severity=ERROR" {
		t.Fatalf("expected uncommented line, got %q", got)
	}

	qm.HandleKey("word-left")
	qm.HandleKey("delete-word-left")
	if got := qm.GetInput(); got != "ERROR" {
		t.Fatalf("expected previous word deleted, got %q", got)
	}
}

func TestQueryModalIndentUnindentAndDuplicateLine(t *testing.T) {
	qm := NewQueryModal()
	qm.SetInput("a=1")

	qm.HandleKey("indent")
	if got := qm.GetInput(); got != "  a=1" {
		t.Fatalf("expected indented line, got %q", got)
	}

	qm.HandleKey("unindent")
	if got := qm.GetInput(); got != "a=1" {
		t.Fatalf("expected unindented line, got %q", got)
	}

	qm.HandleKey("duplicate-line")
	if got := qm.GetInput(); got != "a=1\na=1" {
		t.Fatalf("expected duplicated line, got %q", got)
	}
}

func TestQueryModalSelectAllReplace(t *testing.T) {
	qm := NewQueryModal()
	qm.SetInput("severity=ERROR")
	qm.HandleKey("select-all")
	qm.HandleKey("x")
	if got := qm.GetInput(); got != "x" {
		t.Fatalf("expected select-all replace, got %q", got)
	}
}
