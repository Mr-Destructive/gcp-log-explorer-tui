package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/user/log-explorer-tui/pkg/models"
)

// SavedQueryRecord stores a reusable named query.
type SavedQueryRecord struct {
	Name      string    `json:"name"`
	Filter    string    `json:"filter"`
	Project   string    `json:"project,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	UseCount  int       `json:"useCount"`
}

// QueryLibrary stores saved queries.
type QueryLibrary struct {
	Queries []SavedQueryRecord `json:"queries"`
}

// CachedQueryRecord stores query results for replay.
type CachedQueryRecord struct {
	Key      string            `json:"key"`
	Filter   string            `json:"filter"`
	Project  string            `json:"project,omitempty"`
	StoredAt time.Time         `json:"storedAt"`
	Logs     []models.LogEntry `json:"logs"`
}

// QueryResultCache stores cached query results.
type QueryResultCache struct {
	Entries []CachedQueryRecord `json:"entries"`
}

// LoadQueryLibrary loads saved query library from disk.
func LoadQueryLibrary() (QueryLibrary, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return QueryLibrary{}, err
	}
	path := filepath.Join(configDir, "query_library.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return QueryLibrary{Queries: []SavedQueryRecord{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return QueryLibrary{}, err
	}
	var lib QueryLibrary
	if err := json.Unmarshal(data, &lib); err != nil {
		return QueryLibrary{}, err
	}
	if lib.Queries == nil {
		lib.Queries = []SavedQueryRecord{}
	}
	return lib, nil
}

// SaveQueryLibrary saves query library to disk.
func SaveQueryLibrary(lib QueryLibrary) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(configDir, "query_library.json")
	data, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// UpsertSavedQuery inserts or updates a saved query.
func UpsertSavedQuery(lib QueryLibrary, record SavedQueryRecord, maxEntries int) QueryLibrary {
	record.Name = strings.TrimSpace(record.Name)
	record.Filter = strings.TrimSpace(record.Filter)
	record.Project = strings.TrimSpace(record.Project)
	if record.Name == "" || record.Filter == "" {
		return lib
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now()
	}
	if record.UseCount <= 0 {
		record.UseCount = 1
	}

	for i := range lib.Queries {
		q := lib.Queries[i]
		if q.Name == record.Name || q.Filter == record.Filter {
			record.UseCount = q.UseCount + 1
			lib.Queries[i] = record
			sort.SliceStable(lib.Queries, func(a, b int) bool {
				return lib.Queries[a].UpdatedAt.After(lib.Queries[b].UpdatedAt)
			})
			if maxEntries > 0 && len(lib.Queries) > maxEntries {
				lib.Queries = lib.Queries[:maxEntries]
			}
			return lib
		}
	}

	lib.Queries = append([]SavedQueryRecord{record}, lib.Queries...)
	if maxEntries > 0 && len(lib.Queries) > maxEntries {
		lib.Queries = lib.Queries[:maxEntries]
	}
	return lib
}

// LoadQueryResultCache loads persisted query result cache.
func LoadQueryResultCache() (QueryResultCache, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return QueryResultCache{}, err
	}
	path := filepath.Join(configDir, "query_cache.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return QueryResultCache{Entries: []CachedQueryRecord{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return QueryResultCache{}, err
	}
	var cache QueryResultCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return QueryResultCache{}, err
	}
	if cache.Entries == nil {
		cache.Entries = []CachedQueryRecord{}
	}
	return cache, nil
}

// SaveQueryResultCache persists query result cache.
func SaveQueryResultCache(cache QueryResultCache) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(configDir, "query_cache.json")
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
