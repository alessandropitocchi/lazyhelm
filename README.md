# LazyHelm 🚀

> A fast, intuitive Terminal User Interface (TUI) for managing Helm charts

LazyHelm brings the speed and elegance of lazygit/lazydocker to Helm chart management. Browse repositories, explore charts, compare versions, and edit values—all without leaving your terminal.

## ✨ Features

- 🔍 **Interactive browsing** of Helm repositories, charts, and versions
- 🎨 **Syntax highlighting** for YAML values
- 📝 **External editor integration** (nvim/vim/vi with full config)
- 🔄 **Version comparison** with inline diff viewer
- 🔎 **Fuzzy search** across repos, charts, versions, and values
- 📋 **YAML path copying** for quick reference
- 💾 **Export values** to custom files with validation
- 🎯 **Template generation** for previewing deployments
- ⚡ **Smart caching** for fast navigation
- 🎭 **Diff viewer** to compare values between versions

## 📦 Installation

### Using Go Install (Recommended)

```bash
go install github.com/alessandropitocchi/lazyhelm/cmd/lazyhelm@latest
```

### Using Install Script

```bash
curl -sSL https://raw.githubusercontent.com/alessandropitocchi/lazyhelm/main/install.sh | bash
```

### Manual Installation

```bash
git clone https://github.com/alessandropitocchi/lazyhelm.git
cd lazyhelm
go build -o lazyhelm ./cmd/lazyhelm
sudo mv lazyhelm /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/alessandropitocchi/lazyhelm.git
cd lazyhelm
make install
```

## 🚀 Quick Start

```bash
# Launch LazyHelm
lazyhelm

# Set your preferred editor (optional)
export EDITOR=nvim
lazyhelm
```

## ⌨️ Keybindings

### Navigation
| Key | Action |
|-----|--------|
| `↑/k`, `↓/j` | Move up/down |
| `←/h`, `→/l` | Scroll left/right (in values view) |
| `enter` | Select item / Go deeper |
| `esc` | Go back / Cancel |
| `q` | Quit |

### Actions
| Key | Action |
|-----|--------|
| `/` | Fuzzy search in current view |
| `n` / `N` | Next/Previous search result |
| `a` | Add repository |
| `e` | Edit values in external editor |
| `w` | Write/export values to file |
| `t` | Generate Helm template |
| `v` | View versions (in chart list) |
| `y` | Copy YAML path to clipboard |
| `d` | Diff two versions |
| `?` | Show help |

## 🎯 Usage Examples

### Browse and Edit Values

1. Launch `lazyhelm`
2. Navigate to a repository (e.g., `bitnami`)
3. Select a chart (e.g., `postgresql`)
4. Choose a version
5. Press `e` to edit in your editor
6. Make changes, save (`:wq`)
7. Specify output path for the custom values

### Compare Versions

1. Navigate to chart detail view
2. Press `d` on the first version
3. Press `enter` on the second version
4. View the diff with changes highlighted

### Export Values

1. Navigate to any chart version
2. Press `w` (write/export)
3. Enter the output path (e.g., `./my-values.yaml`)

### Search

- Press `/` in any list to start fuzzy search
- Type to filter (e.g., "pg" matches "postgresql")
- Press `esc` to clear and show all items

## 🎨 Editor Integration

LazyHelm respects your editor preferences:

```bash
# Set your preferred editor
export EDITOR=nvim        # Neovim
export EDITOR=vim         # Vim
export EDITOR="code --wait"  # VS Code
export EDITOR=nano        # Nano

# If not set, LazyHelm will auto-detect (nvim → vim → vi)
```

The editor opens with:
- Full syntax highlighting for YAML
- Your complete configuration and plugins
- Line numbers and all editor features

## 🔧 Configuration

LazyHelm uses Helm's standard configuration:
- Repositories: `~/.config/helm/repositories.yaml`
- Cache: Uses Helm's cache directory

### Requirements

- Go 1.21+ (for building from source)
- Helm 3.x installed and configured
- A terminal that supports ANSI colors

## 📚 How It Works

LazyHelm is built with:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - The Elm Architecture for Go
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- Helm v3 SDK - Chart operations

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

```bash
# Clone the repository
git clone https://github.com/alessandropitocchi/lazyhelm.git
cd lazyhelm

# Install dependencies
go mod download

# Build
go build -o lazyhelm ./cmd/lazyhelm

# Run
./lazyhelm

# Run tests
go test ./...
```

## 📝 Roadmap

- [ ] Helm operations (install, upgrade, uninstall)
- [ ] Live Kubernetes integration (deployed releases)
- [ ] Repository management (update, remove)
- [ ] Configuration file support
- [ ] Favorites/bookmarks
- [ ] Multi-cluster support

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

- Inspired by [lazygit](https://github.com/jesseduffield/lazygit) and [lazydocker](https://github.com/jesseduffield/lazydocker)
- Built with [Charm](https://charm.sh/) TUI tools
- Helm community for the amazing package manager

## 📮 Contact

- GitHub Issues: [Report a bug](https://github.com/alessandropitocchi/lazyhelm/issues)
- Pull Requests: [Contribute](https://github.com/alessandropitocchi/lazyhelm/pulls)

---

Made with ❤️ for the Kubernetes community
