# GCP Log Explorer TUI

A modern terminal user interface for Google Cloud Platform's Cloud Logging service. Built with Go and Bubble Tea, this tool provides a fast, intuitive way to explore, filter, and analyze GCP logs directly from your terminal.

## Features

- 🚀 **Fast & Responsive**: Vim-keybindings for power users
- 🔍 **Advanced Filtering**: Time ranges, severity levels, and custom filters
- 📊 **Log Timeline**: Visual timeline of log distribution
- 💾 **Query History**: Save and reuse your favorite queries
- 📋 **Export Options**: Export logs as CSV or JSON
- 🔄 **Streaming Mode**: Real-time log monitoring
- ⌨️ **Vim Keybindings**: Navigate and act like a vim power user
- 🎯 **Project Switching**: Seamlessly switch between GCP projects

## Installation

### Prerequisites
- Go 1.21+ (to build from source)
- `gcloud` CLI configured with valid credentials
- GCP project with Cloud Logging enabled

### Quick Install (Prebuilt Binary)

Download the latest release for your platform:

```bash
# macOS (Intel)
curl -L https://github.com/Mr-Destructive/gcp-log-explorer-tui/releases/latest/download/log-explorer-darwin-amd64 -o log-explorer
chmod +x log-explorer
./log-explorer

# macOS (Apple Silicon)
curl -L https://github.com/Mr-Destructive/gcp-log-explorer-tui/releases/latest/download/log-explorer-darwin-arm64 -o log-explorer
chmod +x log-explorer
./log-explorer

# Linux (x86_64)
curl -L https://github.com/Mr-Destructive/gcp-log-explorer-tui/releases/latest/download/log-explorer-linux-amd64 -o log-explorer
chmod +x log-explorer
./log-explorer

# Linux (ARM64)
curl -L https://github.com/Mr-Destructive/gcp-log-explorer-tui/releases/latest/download/log-explorer-linux-arm64 -o log-explorer
chmod +x log-explorer
./log-explorer
```

### Install to PATH

```bash
# Download to /usr/local/bin
curl -L https://github.com/Mr-Destructive/gcp-log-explorer-tui/releases/latest/download/log-explorer-linux-amd64 -o /usr/local/bin/log-explorer
chmod +x /usr/local/bin/log-explorer

# Now run from anywhere
log-explorer
```

### Build from Source

```bash
git clone https://github.com/Mr-Destructive/gcp-log-explorer-tui.git
cd gcp-log-explorer-tui
go build -o log-explorer ./cmd/main
./log-explorer
```

## Usage

### Basic Commands

```bash
log-explorer
```

Once running, use these keybindings:

#### Navigation
| Key | Action |
|-----|--------|
| `h/l` | Move between panes |
| `j/k` | Scroll logs up/down |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl+f` | Page down |
| `Ctrl+b` | Page up |

#### Actions
| Key | Action |
|-----|--------|
| `q` | Write/edit query |
| `/` | Search logs |
| `t` | Time range picker |
| `f` | Severity filter |
| `e` | Export logs |
| `s` | Share link |
| `m` | Stream toggle |
| `Enter` | Expand log details |
| `Esc` | Close modal |
| `?` | Help |
| `:q` | Quit |

### Authentication

The tool uses your existing `gcloud` CLI configuration:

```bash
# Set your GCP project
gcloud config set project YOUR_PROJECT_ID

# Or use environment variable
export GOOGLE_CLOUD_PROJECT=YOUR_PROJECT_ID

# Ensure you're authenticated
gcloud auth login
```

### Configuration

Configuration is stored in `~/.config/log-explorer-tui/`:

- `state.json` - Current project and last query
- `history.json` - Query history (max 50 entries)
- `preferences.json` - UI preferences and settings
- `query_library.json` - Saved filter library
- `query_cache.json` - Cached query results

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed system design and component hierarchy.

## Development

### Building

```bash
go build -o log-explorer ./cmd/main
```

### Running Tests

```bash
go test ./...
```

### Building Releases

```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o log-explorer-darwin-amd64 ./cmd/main
GOOS=darwin GOARCH=arm64 go build -o log-explorer-darwin-arm64 ./cmd/main

# Linux
GOOS=linux GOARCH=amd64 go build -o log-explorer-linux-amd64 ./cmd/main
GOOS=linux GOARCH=arm64 go build -o log-explorer-linux-arm64 ./cmd/main

# Windows
GOOS=windows GOARCH=amd64 go build -o log-explorer-windows-amd64.exe ./cmd/main
```

## Project Structure

```
.
├── cmd/main/          # CLI entry point
├── pkg/
│   ├── auth/          # GCP authentication
│   ├── config/        # Configuration and state management
│   ├── gcp/           # GCP API integration
│   ├── models/        # Data models
│   ├── query/         # Query building and execution
│   └── ui/            # TUI components
├── ARCHITECTURE.md    # System design documentation
└── go.mod            # Go module definition
```

## Troubleshooting

### "No project ID found"
```bash
# Set your default GCP project
gcloud config set project YOUR_PROJECT_ID
```

### "Permission denied" when running downloaded binary
```bash
chmod +x log-explorer
```

### Query execution times out
- Ensure your query is valid GCP Cloud Logging syntax
- Check your network connection to GCP APIs
- Try reducing the time range or adding more specific filters

### Cannot find gcloud CLI
```bash
# Install gcloud SDK
# macOS: brew install google-cloud-sdk
# Linux: see https://cloud.google.com/sdk/docs/install

gcloud --version  # Verify installation
```

## Contributing

Contributions are welcome. Please open an issue or submit a pull request.

## License

MIT

## Support

For issues, questions, or feature requests, please open an issue on GitHub.

## Credits

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Charm Ecosystem](https://charm.sh/) - Terminal UI libraries
- [Google Cloud Logging API](https://cloud.google.com/logging/docs) - GCP integration
