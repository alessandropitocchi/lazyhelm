# LazyHelm

A terminal UI for browsing and managing Helm charts. Inspired by lazygit and lazydocker.

## What it does

Browse Helm repos, explore chart versions, view and edit values files, and compare versions - all in your terminal. No need to remember helm commands or manually fetch values.

## Features

- Interactive browsing of Helm repositories and charts
- Search and browse Artifact Hub directly from the UI
- Add repositories from Artifact Hub with package info and security reports
- Syntax-highlighted YAML viewing
- Edit values in your preferred editor (nvim/vim/vi/etc)
- Compare values between versions with diff view
- Fuzzy search and filtering with quick clear
- Copy YAML paths to clipboard
- Export values to files
- Template generation preview
- Repository management (add/remove)

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

## Keybindings

### Navigation
- `↑/k`, `↓/j` - Move up/down
- `←/h`, `→/l` - Scroll horizontally (values view)
- `enter` - Select / Go deeper
- `esc` - Go back
- `q` - Quit

### Actions
- `/` - Search/filter current view
- `c` - Clear search filter
- `n` / `N` - Next/Previous search result
- `s` - Search Artifact Hub
- `a` - Add repository
- `r` - Remove repository
- `e` - Edit values in external editor
- `w` - Export values to file
- `t` - Generate Helm template
- `v` - View versions
- `y` - Copy YAML path
- `d` - Diff two versions
- `?` - Help

## How it works

Uses the Helm SDK to interact with chart repos and the [Bubbletea](https://github.com/charmbracelet/bubbletea) framework for the TUI.

Reads from your existing Helm config (`~/.config/helm/repositories.yaml`) and caches data locally for faster browsing.

## Requirements

- Helm 3.x installed
- Go 1.21+ (if building from source)
- Terminal with ANSI color support

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
- Show deployed releases from K8s
- Repository update functionality
- Config file
- Bookmarks

## License

Licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2025 Alessandro Pitocchi
