package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config represents application configuration
type Config struct {
	InitialBatchSize   int           `json:"initialBatchSize"`
	LoadChunkSize      int           `json:"loadChunkSize"`
	StreamRefreshMs    int           `json:"streamRefreshMs"`
	MaxHistoryEntries  int           `json:"maxHistoryEntries"`
	TimeoutSeconds     int           `json:"timeoutSeconds"`
	VimMode            bool          `json:"vimMode"`
	DefaultProject     string        `json:"defaultProject,omitempty"`
}

// DefaultConfig returns default configuration values
func DefaultConfig() Config {
	return Config{
		InitialBatchSize:   100,
		LoadChunkSize:      50,
		StreamRefreshMs:    2000,
		MaxHistoryEntries:  50,
		TimeoutSeconds:     30,
		VimMode:            true,
	}
}

// GetConfigDir returns the XDG config directory for log-explorer-tui
func GetConfigDir() (string, error) {
	var configDir string
	
	// Try XDG_CONFIG_HOME first
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		configDir = filepath.Join(xdgHome, "log-explorer-tui")
	} else {
		// Fall back to ~/.config
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config", "log-explorer-tui")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}
	
	return configDir, nil
}

// LoadConfig loads configuration from disk, returns default if file doesn't exist
func LoadConfig() (Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultConfig(), err
	}
	
	configPath := filepath.Join(configDir, "config.json")
	
	// If file doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), err
	}
	
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	
	return cfg, nil
}

// SaveConfig saves configuration to disk
func SaveConfig(cfg Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	configPath := filepath.Join(configDir, "config.json")
	
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0600)
}

// State represents persistent application state
type State struct {
	CurrentProject string    `json:"currentProject,omitempty"`
	LastQuery      string    `json:"lastQuery,omitempty"`
	LastUpdated    time.Time `json:"lastUpdated"`
}

// LoadState loads state from disk
func LoadState() (State, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return State{}, err
	}
	
	statePath := filepath.Join(configDir, "state.json")
	
	// If file doesn't exist, return empty state
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return State{LastUpdated: time.Now()}, nil
	}
	
	data, err := os.ReadFile(statePath)
	if err != nil {
		return State{}, err
	}
	
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	
	return state, nil
}

// SaveState saves state to disk
func SaveState(state State) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	state.LastUpdated = time.Now()
	statePath := filepath.Join(configDir, "state.json")
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(statePath, data, 0600)
}

// QueryHistory represents saved queries
type QueryHistory struct {
	Queries []QueryRecord `json:"queries"`
}

// QueryRecord is a single query in history
type QueryRecord struct {
	Filter      string    `json:"filter"`
	Project     string    `json:"project"`
	ExecutedAt  time.Time `json:"executedAt"`
	ExecuteCount int      `json:"executeCount"`
}

// LoadQueryHistory loads query history from disk
func LoadQueryHistory() (QueryHistory, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return QueryHistory{}, err
	}
	
	historyPath := filepath.Join(configDir, "history.json")
	
	// If file doesn't exist, return empty history
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return QueryHistory{Queries: []QueryRecord{}}, nil
	}
	
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return QueryHistory{}, err
	}
	
	var history QueryHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return QueryHistory{}, err
	}
	
	return history, nil
}

// SaveQueryHistory saves query history to disk
func SaveQueryHistory(history QueryHistory) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	historyPath := filepath.Join(configDir, "history.json")
	
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(historyPath, data, 0600)
}

// AddQueryToHistory adds a query to history and maintains max size
func AddQueryToHistory(history QueryHistory, filter, project string, maxEntries int) QueryHistory {
	// Check if query already exists
	for i, q := range history.Queries {
		if q.Filter == filter && q.Project == project {
			// Move to front and increment count
			history.Queries[i].ExecuteCount++
			history.Queries[i].ExecutedAt = time.Now()
			record := history.Queries[i]
			history.Queries = append([]QueryRecord{record}, append(history.Queries[:i], history.Queries[i+1:]...)...)
			return history
		}
	}
	
	// Add new query
	record := QueryRecord{
		Filter:      filter,
		Project:     project,
		ExecutedAt:  time.Now(),
		ExecuteCount: 1,
	}
	
	history.Queries = append([]QueryRecord{record}, history.Queries...)
	
	// Trim to max size
	if len(history.Queries) > maxEntries {
		history.Queries = history.Queries[:maxEntries]
	}
	
	return history
}
