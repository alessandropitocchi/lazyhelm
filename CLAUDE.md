# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LazyHelm is a Terminal User Interface (TUI) application for managing Helm charts. It provides an interactive way to browse Helm repositories, charts, versions, and values files without needing to remember complex Helm CLI commands.

## Build and Run Commands

### Building the Application
```bash
# Build the binary
go build -o lazyhelm ./cmd/lazyhelm

# Build with all dependencies fetched
go mod download && go build -o lazyhelm ./cmd/lazyhelm
```

### Running the Application
```bash
# Run directly
go run ./cmd/lazyhelm/main.go

# Run the built binary
./lazyhelm
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/helm
```

### Go Module Management
```bash
# Download dependencies
go mod download

# Tidy dependencies (remove unused, add missing)
go mod tidy

# Verify dependencies
go mod verify
```

## Code Architecture

### High-Level Structure

The application follows a clean architecture pattern with clear separation of concerns:

- **cmd/lazyhelm/main.go**: Entry point and TUI implementation using the Elm architecture (Model-View-Update pattern)
- **internal/helm/client.go**: Wrapper around Helm v3 SDK providing a simplified interface for Helm operations
- **internal/helm/cache.go**: Thread-safe caching system for Helm values with TTL support
- **internal/ui/yaml_highlighter.go**: YAML syntax highlighting with regex-based parsing
- **internal/ui/yaml_utils.go**: YAML path extraction and diff utilities
- **internal/models/**: Reserved for domain models (currently empty)

### TUI Architecture (Bubbletea Pattern)

The main application in `cmd/lazyhelm/main.go` follows the Bubbletea framework's Elm architecture:

1. **Model**: State container holding all application data
   - Navigation state machine: RepoList -> ChartList -> ChartDetail -> ValueViewer
   - Input modes (normal, search, template, export, etc.)
   - Bubbles components (list.Model, viewport.Model, textinput.Model, help.Model)
   - Loading states for async operations
   - Selected indices for tracking navigation depth

2. **Update**: Message handler that processes events and returns updated model + commands
   - Key events trigger state changes via navigation handlers
   - Async operations (loading charts/values/versions) return messages
   - Modal input modes (search, path entry, repo addition)
   - State-aware event routing based on current navigationState

3. **View**: Pure function that renders the current model state
   - Single active panel based on navigationState
   - Breadcrumb navigation showing current path
   - Help panel overlay (press '?')
   - Status messages and loading indicators

### Helm Client Wrapper

The `internal/helm/client.go` provides a simplified interface for Helm operations:

- Uses `helm.sh/helm/v3/pkg/cli` for configuration
- Executes Helm CLI commands via `os/exec` for operations
- Returns structured data (Repository, Chart, ChartVersion types)
- Handles JSON parsing from Helm CLI output

**Key Operations**:
- `ListRepositories()`: Lists configured Helm repos from repo config file
- `SearchCharts(repoName)`: Searches for charts in a specific repository
- `GetChartVersions(chartName)`: Retrieves all versions of a chart
- `GetChartValues(chartName)`: Fetches default values for a chart
- `GetChartValuesByVersion(chartName, version)`: Fetches values for specific version
- `GenerateTemplate(chartName, valuesFile, outputPath)`: Generates Helm templates
- `AddRepository(name, url)`: Adds and updates a new Helm repository

## Key Implementation Details

### Navigation State Machine
The application uses a hierarchical navigation pattern:
- **Enter**: Drills down into selected item (RepoList -> ChartList -> ChartDetail -> ValueViewer)
- **Esc**: Returns to previous level in hierarchy
- **Breadcrumb**: Shows current path (e.g., "LazyHelm > bitnami > postgresql > v12.1.2 > values")
- Each state maintains its own selection index for quick navigation back

Navigation flow:
```
stateRepoList (select repo)
    -> stateChartList (loads charts for repo, select chart)
        -> stateChartDetail (loads versions for chart, select version)
            -> stateValueViewer (loads values for version)
```

### Search Functionality
- Press '/' in any list view (repos, charts, versions) to start fuzzy search
- Uses `github.com/sahilm/fuzzy` for intelligent matching (e.g., "pg" matches "postgresql")
- Search is live: results update as you type
- Press Enter to confirm search, Esc to cancel and restore full list
- Search context is preserved per state

### YAML Syntax Highlighting
- Values are automatically highlighted with color-coded syntax:
  - Keys: Blue
  - String values: Green
  - Numbers: Yellow
  - Booleans (true/false/yes/no): Orange
  - Comments: Gray
  - Null values: Gray
- Highlighting is applied in real-time as values are loaded

### Caching System
- Values are cached for 30 minutes to avoid redundant Helm API calls
- Cache is keyed by chartName and version
- Cache is thread-safe with read/write locks
- Significantly improves performance when navigating back to previously viewed values
- Cache persists for the entire application session

### Copy YAML Path
- Press 'y' in the values viewer to copy the current YAML path to clipboard
- Automatically builds the full hierarchical path (e.g., "persistence.postgresql.size")
- Uses system clipboard via `github.com/atotto/clipboard`
- Success message confirms what was copied

### Version Diff
- Press 'd' in chart detail view to enter diff mode
- Select first version, then press Enter on second version to compare
- Diff view shows:
  - Added lines (green with +)
  - Removed lines (red with -)
  - Modified lines (yellow with ~)
  - Unchanged lines (gray)
- Uses cache for fast diff generation
- Press Esc to exit diff view and return to version list

### Bubbles Components
The application leverages Bubbletea's bubbles library for reusable components:
- **list.Model**: Used for repositories, charts, and versions lists with built-in filtering
- **viewport.Model**: Used for scrollable values display
- **textinput.Model**: Used for all input modes (search, add repo, export, template)
- **help.Model**: Provides consistent keybinding help display
- **key.Binding**: Defines and documents all keyboard shortcuts

### Async Operations
All Helm operations are asynchronous using Bubbletea commands:
- Loading states prevent UI blocking
- Results return via typed messages (chartsLoadedMsg, valuesLoadedMsg, etc.)
- Errors are captured and displayed in the UI
- Multiple async operations can be in flight simultaneously

## Development Guidelines

### Adding New Helm Operations
1. Add method to `internal/helm/client.go` with error handling
2. Define corresponding message type in `cmd/lazyhelm/main.go`
3. Create command function that returns tea.Cmd
4. Handle message in Update() function's message switch
5. Update appropriate render function to show results or loading state

### Adding New Navigation States
1. Add new navigationState constant
2. Update handleEnter() and handleBack() to include new state transitions
3. Add render function for the new state (e.g., renderNewState())
4. Update getBreadcrumb() to include new state in path
5. Add state case to View() switch statement

### Adding New Input Modes
1. Add new inputMode constant
2. Add key binding to keyMap struct and defaultKeys
3. Implement handling in handleInputMode() function
4. Add input prompt rendering in renderInputPrompt()
5. Define what happens on Enter/Esc for this mode

### Working with Bubbletea
- Commands (tea.Cmd) are functions that return messages
- Use tea.Batch() to execute multiple commands together
- Messages flow: User Input → Update → State Change → View Render
- Never mutate external state in View() - it must be pure
- Delegate updates to bubbles components in their respective states

### Working with Bubbles Components
- Each bubbles component has its own Update() and View() methods
- Forward relevant messages to active component's Update()
- Components manage their own internal state (cursor position, scroll, etc.)
- Use component's built-in features when possible (e.g., list filtering)

## Dependencies

Key dependencies:
- **github.com/charmbracelet/bubbletea**: TUI framework (Elm architecture)
- **github.com/charmbracelet/bubbles**: Reusable TUI components (list, viewport, textinput, help, key)
- **github.com/charmbracelet/lipgloss**: Styling and layout
- **github.com/sahilm/fuzzy**: Fuzzy string matching for search
- **github.com/atotto/clipboard**: Cross-platform clipboard access
- **helm.sh/helm/v3**: Helm SDK for configuration and repo file parsing

## Terminal Requirements

The application uses:
- Alt screen mode for full terminal takeover
- Mouse cell motion for potential future mouse support
- ANSI colors and styling via lipgloss
- Minimum recommended terminal size: 160x40 for comfortable viewing
