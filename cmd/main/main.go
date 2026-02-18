package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/log-explorer-tui/pkg/config"
	"github.com/user/log-explorer-tui/pkg/models"
	"github.com/user/log-explorer-tui/pkg/query"
	"github.com/user/log-explorer-tui/pkg/ui"
)

func main() {
	// Phase 1: Bootstrap
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load state
	state, err := config.LoadState()
	if err != nil {
		log.Fatalf("Failed to load state: %v", err)
	}

	// Initialize app state
	appState := initializeAppState(cfg, state)

	// Attempt authentication
	projectID := appState.CurrentProject
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	// Try to get project from gcloud config if not set
	if projectID == "" {
		projectID = getGcloudProject()
	}

	if projectID == "" {
		fmt.Println("Error: No project ID found.")
		fmt.Println("Please set default GCP project:")
		fmt.Println("\n  gcloud config set project PROJECT_ID")
		fmt.Println("\nOr set environment variable:")
		fmt.Println("  export GOOGLE_CLOUD_PROJECT=your-project-id")
		os.Exit(1)
	}

	// Note: Using gcloud CLI directly, so we don't need the Go logging client
	// This avoids ADC credential issues
	appState.CurrentProject = projectID
	appState.IsReady = true

	// Save updated state
	state.CurrentProject = projectID
	if err := config.SaveState(state); err != nil {
		log.Printf("Warning: Failed to save state: %v", err)
	}

	// Phase 2: Start TUI
	// Build initial startup filter for the past day; execution happens in TUI Init().
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	filter := fmt.Sprintf(`timestamp>="%s"`, oneDayAgo.Format(time.RFC3339))

	// Create and run the app
	app := ui.NewApp(&appState)
	app.SetVimMode(cfg.VimMode)
	historyStore, err := config.LoadQueryHistory()
	if err == nil {
		historyFilters := make([]string, 0, len(historyStore.Queries))
		seen := map[string]bool{}
		for _, q := range historyStore.Queries {
			filter := strings.TrimSpace(q.Filter)
			if filter == "" || seen[filter] {
				continue
			}
			seen[filter] = true
			historyFilters = append(historyFilters, filter)
		}
		app.SetQueryHistory(historyFilters)
	}
	app.SetQueryHistoryPersistFn(func(filter, project string) error {
		historyStore = config.AddQueryToHistory(historyStore, filter, project, cfg.MaxHistoryEntries)
		return config.SaveQueryHistory(historyStore)
	})
	libraryStore, err := config.LoadQueryLibrary()
	if err == nil {
		app.SetQueryLibrary(libraryStore.Queries)
	}
	app.SetQueryLibraryPersistFn(func(entries []config.SavedQueryRecord) error {
		libraryStore = config.QueryLibrary{Queries: entries}
		return config.SaveQueryLibrary(libraryStore)
	})
	cacheStore, err := config.LoadQueryResultCache()
	if err == nil {
		app.SetQueryCacheEntries(cacheStore.Entries)
	}
	app.SetQueryCachePersistFn(func(entries []config.CachedQueryRecord) error {
		cacheStore = config.QueryResultCache{Entries: entries}
		return config.SaveQueryResultCache(cacheStore)
	})

	// Set up query executor that uses gcloud CLI
	app.SetQueryExecutor(func(filter string) ([]models.LogEntry, error) {
		queryCtx, queryCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer queryCancel()
		project := strings.TrimSpace(appState.CurrentProject)
		if project == "" {
			project = projectID
		}
		executor := query.NewExecutor(nil, project, 30*time.Second)

		req := query.ExecuteRequest{
			Filter:   filter,
			PageSize: 100,
			OrderBy:  "timestamp desc",
		}
		resp, err := executor.ExecuteUsingGcloud(queryCtx, req)
		if err != nil {
			return []models.LogEntry{}, err
		}
		return resp.Entries, nil
	})
	app.SetProjectLister(func() ([]string, error) {
		listCtx, listCancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer listCancel()
		cmd := exec.CommandContext(listCtx, "gcloud", "projects", "list", "--format=value(projectId)")
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(out), "\n")
		seen := map[string]bool{}
		projects := make([]string, 0, len(lines))
		for _, line := range lines {
			project := strings.TrimSpace(line)
			if project == "" || seen[project] {
				continue
			}
			seen[project] = true
			projects = append(projects, project)
		}
		sort.Strings(projects)
		return projects, nil
	})
	app.SetStartupFilter(filter)

	if err := tea.NewProgram(app, tea.WithAltScreen()).Start(); err != nil {
		fmt.Printf("Error running app: %v\n", err)
		os.Exit(1)
	}
}

// getGcloudProject reads the default project from gcloud config
func getGcloudProject() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Try the active configuration file
	configPath := filepath.Join(home, ".config", "gcloud", "configurations", "config_default")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fall back to properties file
		configPath = filepath.Join(home, ".config", "gcloud", "properties")
		data, err = os.ReadFile(configPath)
		if err != nil {
			return ""
		}
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "project") {
			// Handle both "project =" and "project=" formats
			if strings.Contains(line, "=") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					project := strings.TrimSpace(parts[1])
					if project != "" {
						return project
					}
				}
			}
		}
	}

	return ""
}

// initializeAppState creates initial app state from config and saved state
func initializeAppState(cfg config.Config, state config.State) models.AppState {
	return models.AppState{
		CurrentProject: state.CurrentProject,
		CurrentQuery: models.Query{
			Filter:  state.LastQuery,
			Project: state.CurrentProject,
		},
		FilterState: models.FilterState{
			TimeRange: models.TimeRange{
				Preset: "24h",
			},
			Severity: models.SeverityFilter{
				Mode: "individual",
			},
			CustomFilters: make(map[string]string),
		},
		LogListState: models.LogListState{
			Logs:            []models.LogEntry{},
			PaginationState: models.PaginationState{},
		},
		StreamState: models.StreamState{
			Enabled:         false,
			RefreshInterval: time.Duration(cfg.StreamRefreshMs) * time.Millisecond,
		},
		UIState: models.UIState{
			FocusedPane: "logs",
			ActiveModal: "none",
		},
		IsReady: false,
	}
}
