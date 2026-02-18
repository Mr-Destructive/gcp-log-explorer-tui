package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"InitialBatchSize", 100, cfg.InitialBatchSize},
		{"LoadChunkSize", 50, cfg.LoadChunkSize},
		{"StreamRefreshMs", 2000, cfg.StreamRefreshMs},
		{"MaxHistoryEntries", 50, cfg.MaxHistoryEntries},
		{"TimeoutSeconds", 30, cfg.TimeoutSeconds},
		{"VimMode", true, cfg.VimMode},
	}

	for _, tt := range tests {
		if tt.expected != tt.actual {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.actual)
		}
	}
}

func TestGetConfigDir(t *testing.T) {
	// Test with default ~/.config
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}

	if dir == "" {
		t.Fatal("GetConfigDir returned empty string")
	}

	// Check if directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("Config directory was not created: %s", dir)
	}

	// Verify it contains expected path
	expectedSuffix := filepath.Join(".config", "log-explorer-tui")
	if !endsWith(dir, expectedSuffix) {
		t.Errorf("Config dir does not have expected suffix. Got: %s, Expected suffix: %s", dir, expectedSuffix)
	}
}

func TestGetConfigDirWithXDGEnv(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir with XDG failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "log-explorer-tui")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test loading default config when file doesn't exist
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	defaultCfg := DefaultConfig()
	if cfg != defaultCfg {
		t.Errorf("Loaded config doesn't match default")
	}

	// Test saving config
	cfg.InitialBatchSize = 200
	cfg.VimMode = false
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Test loading saved config
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save failed: %v", err)
	}

	if loaded.InitialBatchSize != 200 {
		t.Errorf("Expected InitialBatchSize 200, got %d", loaded.InitialBatchSize)
	}

	if loaded.VimMode != false {
		t.Errorf("Expected VimMode false, got %v", loaded.VimMode)
	}
}

func TestLoadAndSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test loading default state when file doesn't exist
	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state.CurrentProject != "" {
		t.Errorf("Expected empty CurrentProject, got %s", state.CurrentProject)
	}

	// Test saving state
	state.CurrentProject = "my-project"
	state.LastQuery = "resource.type=gae_app"
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Test loading saved state
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState after save failed: %v", err)
	}

	if loaded.CurrentProject != "my-project" {
		t.Errorf("Expected CurrentProject 'my-project', got %s", loaded.CurrentProject)
	}

	if loaded.LastQuery != "resource.type=gae_app" {
		t.Errorf("Expected LastQuery 'resource.type=gae_app', got %s", loaded.LastQuery)
	}
}

func TestAddQueryToHistory(t *testing.T) {
	tests := []struct {
		name           string
		initialHistory QueryHistory
		filter         string
		project        string
		maxEntries     int
		expectedCount  int
		expectedFirst  string
	}{
		{
			name:           "add to empty history",
			initialHistory: QueryHistory{Queries: []QueryRecord{}},
			filter:         "severity=ERROR",
			project:        "my-project",
			maxEntries:     50,
			expectedCount:  1,
			expectedFirst:  "severity=ERROR",
		},
		{
			name: "add duplicate moves to front",
			initialHistory: QueryHistory{Queries: []QueryRecord{
				{Filter: "severity=ERROR", Project: "my-project", ExecuteCount: 1},
				{Filter: "severity=INFO", Project: "my-project", ExecuteCount: 1},
			}},
			filter:         "severity=ERROR",
			project:        "my-project",
			maxEntries:     50,
			expectedCount:  2,
			expectedFirst:  "severity=ERROR",
		},
		{
			name: "trim to max entries",
			initialHistory: QueryHistory{Queries: []QueryRecord{
				{Filter: "query-1", Project: "my-project", ExecuteCount: 1},
			}},
			filter:         "new-query",
			project:        "my-project",
			maxEntries:     1,
			expectedCount:  1,
			expectedFirst:  "new-query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddQueryToHistory(tt.initialHistory, tt.filter, tt.project, tt.maxEntries)

			if len(result.Queries) != tt.expectedCount {
				t.Errorf("Expected %d queries, got %d", tt.expectedCount, len(result.Queries))
			}

			if len(result.Queries) > 0 && result.Queries[0].Filter != tt.expectedFirst {
				t.Errorf("Expected first query '%s', got '%s'", tt.expectedFirst, result.Queries[0].Filter)
			}
		})
	}
}

func TestLoadAndSaveQueryHistory(t *testing.T) {
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test loading default history when file doesn't exist
	history, err := LoadQueryHistory()
	if err != nil {
		t.Fatalf("LoadQueryHistory failed: %v", err)
	}

	if len(history.Queries) != 0 {
		t.Errorf("Expected empty history, got %d queries", len(history.Queries))
	}

	// Test saving history
	history = AddQueryToHistory(history, "severity=ERROR", "my-project", 50)
	history = AddQueryToHistory(history, "severity=INFO", "my-project", 50)

	if err := SaveQueryHistory(history); err != nil {
		t.Fatalf("SaveQueryHistory failed: %v", err)
	}

	// Test loading saved history
	loaded, err := LoadQueryHistory()
	if err != nil {
		t.Fatalf("LoadQueryHistory after save failed: %v", err)
	}

	if len(loaded.Queries) != 2 {
		t.Errorf("Expected 2 queries in history, got %d", len(loaded.Queries))
	}

	if loaded.Queries[0].Filter != "severity=INFO" {
		t.Errorf("Expected first query 'severity=INFO', got '%s'", loaded.Queries[0].Filter)
	}
}

// Helper function
func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
