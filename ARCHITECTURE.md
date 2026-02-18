# Log Explorer TUI - Architecture & Phase Plan

## Project Overview
A Go TUI application for Google Cloud Platform's Log Explorer with vim keybindings, real-time streaming, and advanced filtering.

**Tech Stack:**
- Language: Go
- TUI Framework: Bubble Tea + Charm ecosystem (lipgloss, bubbles, huh!)
- GCP Integration: Official `cloud.google.com/logging/apiv2` SDK
- Configuration: XDG Base Directory standard (`~/.config/log-explorer-tui/`)
- Testing: Go's built-in `testing` package + table-driven tests

---

## Core Architecture

### 1. Component Hierarchy
```
App (Main State)
├── Auth (GCP Credentials)
├── Config (User preferences, cache)
├── Query Engine (Build & execute queries)
├── Data Layer (Log fetching, caching, pagination)
├── UI Layers
│   ├── Main Screen
│   │   ├── Log List Pane (60%)
│   │   ├── Query Pane (20%)
│   │   ├── Log Graph Pane (10%)
│   │   └── Controls Pane (10%)
│   ├── Modal/Popup Handlers
│   │   ├── Time Range Picker
│   │   ├── Severity Filter
│   │   ├── Log Details (Side Panel)
│   │   ├── Export Dialog
│   │   ├── Share Link Dialog
│   │   └── Streaming Toggle
```

### 2. State Management (Bubble Tea)
```
Root Model
├── CurrentProject string
├── CurrentQuery Query
├── FilterState
│   ├── TimeRange (StartTime, EndTime, Preset)
│   ├── Severity (Levels, Mode: Individual/Range)
│   └── CustomFilters map[string]string
├── LogState
│   ├── Logs []LogEntry
│   ├── CurrentIndex int
│   ├── TopBoundaryReached bool
│   ├── BottomBoundaryReached bool
│   ├── IsLoading bool
│   └── PaginationCursors (NextOlder, NextNewer)
├── StreamState
│   ├── Enabled bool
│   ├── LastFetchTime time.Time
│   └── RefreshInterval time.Duration
├── UIState
│   ├── FocusedPane (LogList/Query/Graph)
│   ├── ExpandedLogID string (for side panel)
│   ├── ActiveModal (None/TimeRange/Export/etc)
│   └── MessageQueue []string (for status messages)
```

### 3. Data Flow
```
User Input (KeyMsg, MouseMsg)
    ↓
Update (handles all state mutations)
    ↓
Query Validation & Build
    ↓
GCP API Call (with pagination cursors)
    ↓
Response → Parse → Cache → Update LogState
    ↓
View (renders UI based on state)
    ↓
Terminal Output
```

### 4. GCP Integration Strategy
- **Auth**: Use default credentials from `gcloud` CLI (GOOGLE_APPLICATION_CREDENTIALS env var fallback)
- **Query Execution**: Use `cloud.google.com/logging/apiv2` SDK
- **Pagination**: Use `ListLogsRequest.PageToken` for bidirectional loading
- **Caching**: Cache only:
  - Project ID
  - Query history (last 50)
  - Saved filters/presets
  - NOT log results (too volatile)

### 5. Cache Structure (XDG Standard)
```
~/.config/log-explorer-tui/
├── state.json (current project, last query)
├── history.json (query history, 50 max)
├── preferences.json (batch sizes, refresh interval, vim mode)
└── saved_filters.json (favorite filters)
```

### 6. Configuration & Defaults
```go
type Config struct {
    InitialBatchSize   int           // Default: 100, configurable
    LoadChunkSize      int           // Default: 50, configurable
    StreamRefreshMs    int           // Default: 2000ms
    MaxHistoryEntries  int           // Default: 50
    TimeoutSeconds     int           // Default: 30
}
```

### 7. Key Features by Phase

**Phase 1: Bootstrap & Auth**
- Project structure
- Config loading/saving (XDG)
- GCP auth integration

**Phase 2: Core TUI Shell**
- Pane layout (Log List, Query, Graph, Controls)
- Navigation between panes
- Basic vim keybindings (hjkl, jk for scroll, etc.)

**Phase 3: Query Engine**
- Query builder (text input + syntax validation)
- Execute query via GCP API
- Display results in Log List pane

**Phase 4: Log Pagination**
- Bidirectional lazy loading (scroll up/down)
- Boundary detection & messaging
- Loading state indicator

**Phase 5: Filtering**
- Time range picker (custom date + quick presets)
- Severity filter UI (both modes: individual & range)
- Apply filters to query

**Phase 6: Log Details & Interactions**
- Expandable side panel for log details
- Copy log entry
- Expand/collapse in list
- Search within logs

**Phase 7: Graph & Advanced Features**
- Log count over time graph (timeline)
- Severity distribution
- Export (CSV, JSON)
- Share link generation
- Streaming mode toggle

**Phase 8: Polish & Testing**
- Comprehensive test suite
- Error handling & edge cases
- Performance optimization
- Keybinding documentation (internal)

---

## Keybindings Reference

### Navigation
- `h/l` - Move between panes
- `j/k` - Scroll logs up/down
- `g` - Jump to top
- `G` - Jump to bottom
- `ctrl+f` - Page down
- `ctrl+b` - Page up

### Actions
- `q` - Write/edit query
- `/` - Search logs
- `t` - Time range picker
- `f` - Severity filter
- `e` - Export dialog
- `s` - Share link
- `m` - Stream toggle
- `<Enter>` - Expand log details / Open side panel
- `<Esc>` - Close modal / Close side panel
- `:` - Jump to prompt (if needed)

### Global
- `?` - Help
- `:q` - Quit

---

## Testing Strategy

**Unit Tests:**
- Query builder validation
- Filter application logic
- Pagination cursor handling
- Config serialization/deserialization
- Time range parsing & presets

**Integration Tests:**
- GCP API mock responses
- State mutation flows
- Cache read/write operations

**Patterns:**
- Table-driven tests for edge cases
- Mock GCP client for deterministic testing
- Test fixtures for log entries & responses

---

## Error Handling

- **Auth failures**: Prompt user to run `gcloud auth login`
- **API errors**: Display in footer with retry option
- **Network timeout**: Graceful degradation, show cached results if available
- **Invalid query syntax**: Highlight in query pane with error message
- **File I/O (cache)**: Log error, continue without cache

---

## Performance Considerations

1. **Log rendering**: Virtualize display (only render visible logs)
2. **Pagination**: Use cursors, not full result sets
3. **Graph rendering**: Aggregate data points, resample if >1000 points
4. **Search**: Debounce input, search in-memory only
5. **Streaming**: Non-blocking async fetch with goroutines

---

## Success Criteria

- ✓ TUI boots in <2s
- ✓ Query executes within API timeout
- ✓ Smooth pagination (no janky loading)
- ✓ All keybindings responsive (<100ms)
- ✓ Streaming doesn't block UI
- ✓ 80%+ test coverage for business logic
- ✓ Graceful handling of all error scenarios
