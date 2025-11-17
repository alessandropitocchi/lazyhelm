# LazyHelm

A terminal UI for browsing and managing Helm charts. Inspired by lazygit and lazydocker.

## Demo

![LazyHelm Demo](demo.gif)

*Demo updated to version 0.2.2*

## What it does

LazyHelm brings an intuitive, lazygit-style interface to Helm chart management:

- **Navigate with ease** - Organized menu system to browse local repositories or search Artifact Hub
- **Explore charts** - Browse your configured Helm repos and discover all available charts
- **Search globally** - Find charts across Artifact Hub directly from the terminal
- **Manage repositories** - Add, remove, and update repository indexes with simple keystrokes
- **View & compare** - Inspect chart values with syntax highlighting and diff between versions
- **Edit efficiently** - Modify values in your preferred editor ($EDITOR) with YAML validation
- **Search & filter** - Fuzzy search through repositories, charts, and values
- **Copy & export** - Copy YAML paths to clipboard or export values to files
- **Preview templates** - Generate Helm templates to see what will be deployed
- **Cluster integration** - View deployed releases, explore revision history, and inspect values from any revision

All in your terminal. No need to remember helm commands or manually fetch values.

## Features

- **Intuitive menu system** - Main menu with organized sections for browsing and cluster management
- Interactive browsing of Helm repositories and charts
- Search and browse Artifact Hub directly from the UI
- Add repositories from Artifact Hub with package info and security reports
- **Update repository indexes** - Keep your local chart indexes up to date with `helm repo update`
- **Cluster Releases** - View and manage deployed Helm releases in your Kubernetes cluster:
  - Browse releases across all namespaces or filter by specific namespace
  - View release details: status, chart version, app version, namespace
  - Interactive revision history with full details for each deployment
  - Inspect values used in any revision (current or historical)
  - Automatically uses your current kubectl context
- Syntax-highlighted YAML viewing
- Edit values in your preferred editor (nvim/vim/vi/etc)
- Compare values between versions with diff view
- Fuzzy search and filtering with quick clear
- Copy YAML paths to clipboard
- Export values to files
- Template generation preview
- Repository management (add/remove/update)

## Installation

### Homebrew

```bash
brew tap alessandropitocchi/lazyhelm
brew install lazyhelm
```

Or in one command:
```bash
brew install alessandropitocchi/lazyhelm/lazyhelm
```

### Install script

```bash
curl -sSL https://raw.githubusercontent.com/alessandropitocchi/lazyhelm/main/install.sh | bash
```

### From source

```bash
git clone https://github.com/alessandropitocchi/lazyhelm.git
cd lazyhelm
make install
```

## Usage

Just run:
```bash
lazyhelm
```

Set your editor if you want (defaults to nvim → vim → vi):
```bash
export EDITOR=nvim
```

### Menu Structure

LazyHelm uses an intuitive menu system to organize functionality:

```
Main Menu
├── Browse Repositories
│   ├── Local Repositories - Browse your configured Helm repos
│   └── Search Artifact Hub - Search charts on Artifact Hub
├── Cluster Releases - View and manage deployed Helm releases
│   ├── All Namespaces - View releases across all namespaces
│   └── Select Namespace - Choose a specific namespace to browse
└── Settings (Coming Soon) - Configure LazyHelm
```

## Keybindings

### Navigation
- `↑/k`, `↓/j` - Move up/down
- `←/h`, `→/l` - Scroll left/right (in values view)
- `enter` - Select item / Go deeper
- `esc` - Go back to previous screen
- `q` - Quit application
- `?` - Toggle help screen

### Search & Filter
- `/` - Search/filter in current view
- `c` - Clear search filter
- `n` - Next search result
- `N` - Previous search result

### Repository Management
- `a` - Add new repository
- `r` - Remove selected repository
- `u` - Update repository index (helm repo update)
- `s` - Search Artifact Hub

### Chart & Version Actions
- `v` - View all versions (in chart list)
- `d` - Diff two versions (select first, then second)

### Cluster Releases
- `v` - View current release values (in release detail)
- `h` - Open interactive revision history (in release detail)
- `enter` - Select revision to view its values (in history list)

### Values View
- `e` - Edit values in external editor ($EDITOR)
- `w` - Write/export values to file
- `t` - Generate Helm template
- `y` - Copy YAML path to clipboard
- `←/→`, `h/l` - Scroll horizontally for long lines

## How it works

Uses the Helm SDK to interact with chart repos and the [Bubbletea](https://github.com/charmbracelet/bubbletea) framework for the TUI.

Reads from your existing Helm config (`~/.config/helm/repositories.yaml`) and caches data locally for faster browsing.

## Requirements

- Helm 3.x installed
- Go 1.21+ (if building from source)
- Terminal with ANSI color support
- kubectl configured with valid context (optional, for Cluster Releases feature)

## Development

```bash
git clone https://github.com/alessandropitocchi/lazyhelm.git
cd lazyhelm
go mod download
go build -o lazyhelm ./cmd/lazyhelm
./lazyhelm
```

## TODO

- Helm operations (install/upgrade/uninstall)
- Diff between release revisions
- Config file
- Bookmarks

## License

Licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2025 Alessandro Pitocchi
