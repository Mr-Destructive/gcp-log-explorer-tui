package ui

import (
	"strings"
	"testing"
)

func TestBuildJSONTreeLinesExpandCollapse(t *testing.T) {
	payload := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   float64(42),
			"name": "alice",
		},
		"ok": true,
	}
	expanded := map[string]bool{"$": true}
	lines := buildJSONTreeLines(payload, expanded)
	if len(lines) < 3 {
		t.Fatalf("expected root and top-level keys, got %d lines", len(lines))
	}
	foundUser := false
	for _, line := range lines {
		if strings.Contains(line.text, "user") {
			foundUser = true
			break
		}
	}
	if !foundUser {
		t.Fatalf("expected user key in tree lines")
	}

	expanded["$.user"] = true
	lines = buildJSONTreeLines(payload, expanded)
	foundName := false
	for _, line := range lines {
		if strings.Contains(line.text, "name") {
			foundName = true
			break
		}
	}
	if !foundName {
		t.Fatalf("expected nested key after expansion")
	}
}

func TestFormatJSONValueForCopy(t *testing.T) {
	if got := formatJSONValueForCopy("abc"); got != "abc" {
		t.Fatalf("expected raw string copy, got %q", got)
	}
	obj := map[string]interface{}{"a": float64(1)}
	got := formatJSONValueForCopy(obj)
	if !strings.Contains(got, "\"a\": 1") {
		t.Fatalf("expected marshaled object, got %q", got)
	}
}
