package config

import (
	"os"
	"testing"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

func TestLoadSaveQueryLibrary(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	lib, err := LoadQueryLibrary()
	if err != nil {
		t.Fatalf("LoadQueryLibrary failed: %v", err)
	}
	if len(lib.Queries) != 0 {
		t.Fatalf("expected empty library")
	}

	lib = UpsertSavedQuery(lib, SavedQueryRecord{Name: "Errors", Filter: "severity=ERROR", Project: "p1", UpdatedAt: time.Now()}, 50)
	if err := SaveQueryLibrary(lib); err != nil {
		t.Fatalf("SaveQueryLibrary failed: %v", err)
	}

	loaded, err := LoadQueryLibrary()
	if err != nil {
		t.Fatalf("LoadQueryLibrary after save failed: %v", err)
	}
	if len(loaded.Queries) != 1 || loaded.Queries[0].Name != "Errors" {
		t.Fatalf("unexpected loaded library: %+v", loaded.Queries)
	}
}

func TestLoadSaveQueryResultCache(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache, err := LoadQueryResultCache()
	if err != nil {
		t.Fatalf("LoadQueryResultCache failed: %v", err)
	}
	if len(cache.Entries) != 0 {
		t.Fatalf("expected empty cache")
	}

	cache.Entries = []CachedQueryRecord{{
		Key:      "k1",
		Filter:   "severity=ERROR",
		Project:  "p1",
		StoredAt: time.Now(),
		Logs:     []models.LogEntry{{ID: "1", Message: "x"}},
	}}
	if err := SaveQueryResultCache(cache); err != nil {
		t.Fatalf("SaveQueryResultCache failed: %v", err)
	}

	loaded, err := LoadQueryResultCache()
	if err != nil {
		t.Fatalf("LoadQueryResultCache after save failed: %v", err)
	}
	if len(loaded.Entries) != 1 || loaded.Entries[0].Key != "k1" {
		t.Fatalf("unexpected loaded cache: %+v", loaded.Entries)
	}
}
