package models

import (
	"time"
)

// LogEntry represents a single log entry from GCP
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	JSONPayload map[string]interface{} `json:"jsonPayload,omitempty"`
	TextPayload string                `json:"textPayload,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Resource  Resource               `json:"resource,omitempty"`
	SourceLocation *SourceLocation    `json:"sourceLocation,omitempty"`
	Trace     string                 `json:"trace,omitempty"`
	SpanID    string                 `json:"spanId,omitempty"`
	Raw       interface{}            `json:"-"` // Store original protobuf for detailed view
}

// Resource represents the resource that produced the log
type Resource struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels,omitempty"`
}

// SourceLocation represents where the log originated
type SourceLocation struct {
	File     string `json:"file,omitempty"`
	Line     int64  `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

// Query represents a log query
type Query struct {
	Filter   string    `json:"filter"`
	Project  string    `json:"project"`
	Advanced map[string]interface{} `json:"advanced,omitempty"`
}

// TimeRange represents a time filter
type TimeRange struct {
	Start  time.Time
	End    time.Time
	Preset string // "1h", "24h", "7d", "30d", "custom"
}

// SeverityFilter represents severity filtering options
type SeverityFilter struct {
	Levels    []string `json:"levels"` // e.g., ["ERROR", "CRITICAL"]
	Mode      string   `json:"mode"`   // "individual" or "range"
	MinLevel  string   `json:"minLevel,omitempty"` // For range mode: "WARNING" means WARNING and above
}

// LogGraphPoint represents a point on the log count graph
type LogGraphPoint struct {
	Timestamp time.Time
	Count     int
	Severity  map[string]int // e.g., {"ERROR": 5, "INFO": 20}
}

// FilterState represents all active filters
type FilterState struct {
	TimeRange      TimeRange       `json:"timeRange"`
	Severity       SeverityFilter  `json:"severity"`
	CustomFilters  map[string]string `json:"customFilters"`
	SearchTerm     string          `json:"searchTerm"`
}

// PaginationState tracks pagination cursors
type PaginationState struct {
	NextPageTokenOlder string // For loading older logs
	NextPageTokenNewer string // For loading newer logs
	TopBoundaryReached bool
	BottomBoundaryReached bool
}

// LogListState represents the state of the log list view
type LogListState struct {
	Logs                  []LogEntry
	CurrentIndex          int
	SelectedLogID         string
	IsLoading             bool
	ErrorMessage          string
	PaginationState       PaginationState
}

// StreamState represents streaming mode state
type StreamState struct {
	Enabled         bool
	LastFetchTime   time.Time
	RefreshInterval time.Duration
	NewLogsCount    int // Count of new logs fetched since last user view
}

// UIState represents UI-specific state
type UIState struct {
	FocusedPane      string // "logs", "query", "graph", "controls"
	ExpandedLogID    string // For side panel
	ActiveModal      string // "none", "timeRange", "export", "severity", "query"
	MessageQueue     []string
	HelpVisible      bool
	SearchMode       bool
}

// AppState represents the complete application state
type AppState struct {
	CurrentProject   string
	CurrentQuery     Query
	FilterState      FilterState
	LogListState     LogListState
	StreamState      StreamState
	UIState          UIState
	LastError        error
	IsReady          bool
}

// SeverityLevel constants
const (
	SeverityDefault   = "DEFAULT"
	SeverityDebug     = "DEBUG"
	SeverityInfo      = "INFO"
	SeverityNotice    = "NOTICE"
	SeverityWarning   = "WARNING"
	SeverityError     = "ERROR"
	SeverityCritical  = "CRITICAL"
	SeverityAlert     = "ALERT"
	SeverityEmergency = "EMERGENCY"
)

// SeverityLevels in order
var SeverityLevels = []string{
	SeverityDefault,
	SeverityDebug,
	SeverityInfo,
	SeverityNotice,
	SeverityWarning,
	SeverityError,
	SeverityCritical,
	SeverityAlert,
	SeverityEmergency,
}

// TimeRangePresets
var TimeRangePresets = map[string]time.Duration{
	"1h":  1 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}
