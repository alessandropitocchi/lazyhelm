package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alessandropitocchi/lazyhelm/internal/helm"
	"github.com/alessandropitocchi/lazyhelm/internal/ui"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"gopkg.in/yaml.v3"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("111")). // Azzurro chiaro
			Bold(true).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("250")). // Grigio chiaro/bianco
			Padding(1, 2)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("255")). // Bianco
				Padding(1, 2)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")). // Azzurro chiaro brillante
			Bold(true).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")). // Bianco
			Background(lipgloss.Color("28")).  // Sfondo verde
			Bold(true).
			Padding(0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")). // Bianco
			Background(lipgloss.Color("196")). // Sfondo rosso brillante
			Bold(true).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Grigio molto chiaro

	addedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120")). // Verde chiaro delicato
			Bold(true)

	removedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("210")). // Rosa salmone chiaro (invece di rosso acceso)
			Bold(true)

	modifiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")). // Giallo brillante
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")). // Azzurro chiaro
			Bold(true).
			Padding(0, 2)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Giallo brillante
			Background(lipgloss.Color("235")). // Sfondo grigio scuro per contrasto
			Bold(true).
			Padding(0, 2)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("11")). // Giallo brillante
			Foreground(lipgloss.Color("0")).  // Nero
			Bold(true)

	searchInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117")). // Azzurro chiaro
				Padding(0, 1).
				Bold(true)
)

type navigationState int

const (
	stateRepoList navigationState = iota
	stateChartList
	stateChartDetail
	stateValueViewer
	stateDiffViewer
	stateHelp
)

type inputMode int

const (
	normalMode inputMode = iota
	searchMode
	addRepoMode
	templatePathMode
	templateValuesMode
	exportValuesMode
	saveEditMode
)

type model struct {
	helmClient   *helm.Client
	cache        *helm.Cache
	chartCache   map[string]chartCacheEntry
	versionCache map[string]versionCacheEntry
	state        navigationState
	mode         inputMode

	repos        []helm.Repository
	charts       []helm.Chart
	versions     []helm.ChartVersion
	values       string
	valuesLines  []string
	diffLines    []string // Lines for diff viewer (for search)
	selectedRepo int
	selectedChart int
	selectedVersion int
	compareVersion  int

	// Search in values and diff
	searchMatches      []int    // Line numbers of matches
	currentMatchIndex  int      // Current match being viewed
	lastSearchQuery    string   // Last search query

	// Horizontal scrolling in values
	horizontalOffset   int      // Horizontal scroll offset for long lines

	repoList     list.Model
	chartList    list.Model
	versionList  list.Model
	valuesView   viewport.Model
	diffView     viewport.Model
	searchInput  textinput.Model
	helpView     help.Model
	keys         keyMap

	loading      bool
	loadingVals  bool
	diffMode     bool
	successMsg   string
	err          error
	termWidth    int
	termHeight   int

	templatePath   string
	templateValues string
	exportPath     string
	newRepoName    string
	newRepoURL     string
	addRepoStep    int
	editedContent  string // Content from external editor
	editTempFile   string // Temp file path for editing
}

type chartCacheEntry struct {
	charts    []helm.Chart
	timestamp time.Time
}

type versionCacheEntry struct {
	versions  []helm.ChartVersion
	timestamp time.Time
}

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Back        key.Binding
	Quit        key.Binding
	Search      key.Binding
	NextMatch   key.Binding
	PrevMatch   key.Binding
	Help        key.Binding
	AddRepo     key.Binding
	Export      key.Binding
	Template    key.Binding
	Versions    key.Binding
	Copy        key.Binding
	Diff        key.Binding
	Edit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back, k.Search, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Search, k.AddRepo, k.Export, k.Template},
		{k.Versions, k.Copy, k.Diff, k.Edit},
		{k.Help, k.Quit},
	}
}

var defaultKeys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "scroll left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "scroll right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	NextMatch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	),
	PrevMatch: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "prev match"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	AddRepo: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add repo"),
	),
	Export: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "write/export values"),
	),
	Template: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "template"),
	),
	Versions: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view versions"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy yaml path"),
	),
	Diff: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "diff versions"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit in $EDITOR"),
	),
}

type chartsLoadedMsg struct {
	charts []helm.Chart
	err    error
}

type valuesLoadedMsg struct {
	values string
	err    error
}

type versionsLoadedMsg struct {
	versions []helm.ChartVersion
	err      error
}

type operationDoneMsg struct {
	success string
	err     error
}

type reposReloadedMsg struct {
	repos []helm.Repository
	err   error
}

type editorFinishedMsg struct {
	content  string
	filePath string
	err      error
}

type listItem struct {
	title       string
	description string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }
func (i listItem) FilterValue() string { return i.title }

func loadCharts(client *helm.Client, chartCache map[string]chartCacheEntry, repoName string) tea.Cmd {
	return func() tea.Msg {
		// Check cache first (30 minute TTL)
		if entry, exists := chartCache[repoName]; exists {
			if time.Since(entry.timestamp) < 30*time.Minute {
				return chartsLoadedMsg{charts: entry.charts, err: nil}
			}
		}

		charts, err := client.SearchCharts(repoName)
		if err == nil && len(charts) > 0 {
			chartCache[repoName] = chartCacheEntry{
				charts:    charts,
				timestamp: time.Now(),
			}
		}
		return chartsLoadedMsg{charts: charts, err: err}
	}
}

func loadValues(client *helm.Client, cache *helm.Cache, chartName string) tea.Cmd {
	return func() tea.Msg {
		if cached, found := cache.Get(chartName, ""); found {
			return valuesLoadedMsg{values: cached, err: nil}
		}

		values, err := client.GetChartValues(chartName)
		if err == nil {
			cache.Set(chartName, "", values)
		}
		return valuesLoadedMsg{values: values, err: err}
	}
}

func loadValuesByVersion(client *helm.Client, cache *helm.Cache, chartName, version string) tea.Cmd {
	return func() tea.Msg {
		if cached, found := cache.Get(chartName, version); found {
			return valuesLoadedMsg{values: cached, err: nil}
		}

		values, err := client.GetChartValuesByVersion(chartName, version)
		if err == nil {
			cache.Set(chartName, version, values)
		}
		return valuesLoadedMsg{values: values, err: err}
	}
}

func loadVersions(client *helm.Client, versionCache map[string]versionCacheEntry, chartName string) tea.Cmd {
	return func() tea.Msg {
		// Check cache first (30 minute TTL)
		if entry, exists := versionCache[chartName]; exists {
			if time.Since(entry.timestamp) < 30*time.Minute {
				return versionsLoadedMsg{versions: entry.versions, err: nil}
			}
		}

		versions, err := client.GetChartVersions(chartName)
		if err == nil && len(versions) > 0 {
			versionCache[chartName] = versionCacheEntry{
				versions:  versions,
				timestamp: time.Now(),
			}
		}
		return versionsLoadedMsg{versions: versions, err: err}
	}
}

func addRepository(client *helm.Client, name, url string) tea.Cmd {
	return func() tea.Msg {
		err := client.AddRepository(name, url)
		if err != nil {
			return operationDoneMsg{err: err}
		}

		repos, repoErr := client.ListRepositories()
		if repoErr != nil {
			return operationDoneMsg{success: fmt.Sprintf("Repository '%s' added, but failed to reload list", name)}
		}

		return reposReloadedMsg{repos: repos}
	}
}

func exportValues(client *helm.Client, chartName, outputFile string) tea.Cmd {
	return func() tea.Msg {
		err := client.ExportValues(chartName, outputFile)
		if err != nil {
			return operationDoneMsg{err: err}
		}
		return operationDoneMsg{success: fmt.Sprintf("Values exported to %s", outputFile)}
	}
}

func generateTemplate(client *helm.Client, chartName, valuesFile, outputPath string) tea.Cmd {
	return func() tea.Msg {
		err := client.GenerateTemplate(chartName, valuesFile, outputPath)
		if err != nil {
			return operationDoneMsg{err: err}
		}
		return operationDoneMsg{success: fmt.Sprintf("Template generated in %s", outputPath)}
	}
}

func initialModel() model {
	client := helm.NewClient()
	cache := helm.NewCache(30 * time.Minute)
	repos, err := client.ListRepositories()

	repoItems := make([]list.Item, len(repos))
	for i, repo := range repos {
		repoItems[i] = listItem{
			title:       repo.Name,
			description: repo.URL,
		}
	}

	// Create custom delegate with better colors
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("11")).   // Giallo brillante
		Bold(true).
		Underline(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("117"))   // Azzurro chiaro
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("255"))   // Bianco
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("250"))   // Grigio chiaro

	repoList := list.New(repoItems, delegate, 0, 0)
	repoList.Title = "Repositories"
	repoList.SetShowStatusBar(false)
	repoList.SetFilteringEnabled(true)
	repoList.Styles.Title = titleStyle
	repoList.Styles.FilterPrompt = searchInputStyle
	repoList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("231"))

	chartDelegate := list.NewDefaultDelegate()
	chartDelegate.Styles = delegate.Styles
	chartList := list.New([]list.Item{}, chartDelegate, 0, 0)
	chartList.Title = "Charts"
	chartList.SetShowStatusBar(false)
	chartList.SetFilteringEnabled(true)
	chartList.Styles.Title = titleStyle
	chartList.Styles.FilterPrompt = searchInputStyle
	chartList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("231"))

	versionDelegate := list.NewDefaultDelegate()
	versionDelegate.Styles = delegate.Styles
	versionList := list.New([]list.Item{}, versionDelegate, 0, 0)
	versionList.Title = "Versions"
	versionList.SetShowStatusBar(false)
	versionList.SetFilteringEnabled(true)
	versionList.Styles.Title = titleStyle
	versionList.Styles.FilterPrompt = searchInputStyle
	versionList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("231"))

	valuesView := viewport.New(0, 0)
	diffView := viewport.New(0, 0)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."

	helpView := help.New()

	return model{
		helmClient:   client,
		cache:        cache,
		chartCache:   make(map[string]chartCacheEntry),
		versionCache: make(map[string]versionCacheEntry),
		state:        stateRepoList,
		mode:         normalMode,
		repos:        repos,
		repoList:     repoList,
		chartList:    chartList,
		versionList:  versionList,
		valuesView:   valuesView,
		diffView:     diffView,
		searchInput:  searchInput,
		helpView:     helpView,
		keys:         defaultKeys,
		err:          err,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

		h := msg.Height - 10
		w := msg.Width - 4

		m.repoList.SetSize(w/3, h)
		m.chartList.SetSize(w/2, h)
		m.versionList.SetSize(w/3, h)

		// Values view takes full screen
		m.valuesView.Width = msg.Width - 6  // Full width minus border padding
		m.valuesView.Height = msg.Height - 8 // Full height minus header/footer

		m.diffView.Width = msg.Width - 6
		m.diffView.Height = msg.Height - 8

		return m, nil

	case tea.KeyMsg:
		if m.state == stateHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.state = stateRepoList
			}
			return m, nil
		}

		if m.mode != normalMode {
			return m.handleInputMode(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.state = stateHelp
			return m, nil

		case key.Matches(msg, m.keys.Back):
			return m.handleBack()

		case key.Matches(msg, m.keys.Enter):
			return m.handleEnter()

		case key.Matches(msg, m.keys.Search):
			return m.handleSearch()

		case key.Matches(msg, m.keys.AddRepo):
			if m.state == stateRepoList {
				m.mode = addRepoMode
				m.addRepoStep = 0
				m.searchInput.Reset()
				m.searchInput.Placeholder = "Repository name..."
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keys.Export):
			if m.state == stateChartDetail || m.state == stateValueViewer {
				m.mode = exportValuesMode
				m.searchInput.Reset()
				m.searchInput.Placeholder = "./values.yaml"
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keys.Template):
			if m.state == stateChartDetail || m.state == stateValueViewer {
				m.mode = templatePathMode
				m.searchInput.Reset()
				m.searchInput.Placeholder = "./output/"
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keys.Versions):
			if m.state == stateChartList && len(m.charts) > 0 {
				m.state = stateChartDetail
				m.loading = true
				idx := m.chartList.Index()
				if idx < len(m.charts) {
					return m, loadVersions(m.helmClient, m.versionCache, m.charts[idx].Name)
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Copy):
			if m.state == stateValueViewer && len(m.valuesLines) > 0 {
				var lineNum int
				// If we have search matches, use the current match line
				if len(m.searchMatches) > 0 && m.currentMatchIndex < len(m.searchMatches) {
					lineNum = m.searchMatches[m.currentMatchIndex]
				} else {
					// Otherwise use the current viewport position (center of visible area)
					lineNum = m.valuesView.YOffset + m.valuesView.Height/2
					if lineNum >= len(m.valuesLines) {
						lineNum = len(m.valuesLines) - 1
					}
				}

				yamlPath := ui.GetYAMLPath(m.valuesLines, lineNum)
				if yamlPath != "" {
					err := clipboard.WriteAll(yamlPath)
					if err != nil {
						m.successMsg = "Failed to copy to clipboard"
					} else {
						m.successMsg = "Copied: " + yamlPath
					}
				} else {
					m.successMsg = "No YAML path found for current line"
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Diff):
			if m.state == stateChartDetail && len(m.versions) > 1 {
				m.diffMode = true
				m.compareVersion = m.versionList.Index()
			}
			return m, nil

		case key.Matches(msg, m.keys.Edit):
			if m.state == stateValueViewer {
				if m.values == "" {
					m.successMsg = "No values to edit"
					return m, nil
				}
				// Show which editor will be used
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = os.Getenv("VISUAL")
				}
				if editor == "" {
					// Check which editor will be found
					for _, cmd := range []string{"nvim", "vim", "vi"} {
						if _, err := exec.LookPath(cmd); err == nil {
							editor = cmd
							break
						}
					}
				}
				m.successMsg = fmt.Sprintf("Opening %s...", editor)
				return m, openEditorCmd(m.values)
			}
			return m, nil

		case key.Matches(msg, m.keys.NextMatch):
			if (m.state == stateValueViewer || m.state == stateDiffViewer) && len(m.searchMatches) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex + 1) % len(m.searchMatches)
				if m.state == stateValueViewer {
					m.updateValuesViewWithSearch()
				} else if m.state == stateDiffViewer {
					m.updateDiffViewWithSearch()
				}
				return m.jumpToMatch(), nil
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevMatch):
			if (m.state == stateValueViewer || m.state == stateDiffViewer) && len(m.searchMatches) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
				if m.state == stateValueViewer {
					m.updateValuesViewWithSearch()
				} else if m.state == stateDiffViewer {
					m.updateDiffViewWithSearch()
				}
				return m.jumpToMatch(), nil
			}
			return m, nil

		case key.Matches(msg, m.keys.Left):
			if m.state == stateValueViewer {
				if m.horizontalOffset > 0 {
					m.horizontalOffset -= 5 // Scroll by 5 characters
					if m.horizontalOffset < 0 {
						m.horizontalOffset = 0
					}
					m.updateValuesViewWithSearch()
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Right):
			if m.state == stateValueViewer {
				m.horizontalOffset += 5 // Scroll by 5 characters
				m.updateValuesViewWithSearch()
			}
			return m, nil
		}

	case chartsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.charts = msg.charts
		items := make([]list.Item, len(msg.charts))
		for i, chart := range msg.charts {
			name := chart.Name
			if m.selectedRepo < len(m.repos) {
				name = strings.TrimPrefix(name, m.repos[m.selectedRepo].Name+"/")
			}
			items[i] = listItem{
				title:       name,
				description: chart.Description,
			}
		}
		m.chartList.SetItems(items)
		return m, nil

	case versionsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.versions = msg.versions
		items := make([]list.Item, len(msg.versions))
		for i, ver := range msg.versions {
			desc := ""
			if ver.AppVersion != "" {
				desc = "App: " + ver.AppVersion
			}
			items[i] = listItem{
				title:       "v" + ver.Version,
				description: desc,
			}
		}
		m.versionList.SetItems(items)
		return m, nil

	case valuesLoadedMsg:
		m.loadingVals = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.values = msg.values
		m.valuesLines = strings.Split(msg.values, "\n")
		highlighted := ui.HighlightYAMLContent(msg.values)
		m.valuesView.SetContent(highlighted)
		m.updateValuesViewWithSearch()
		return m, nil

	case operationDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.successMsg = msg.success
		}
		return m, nil

	case reposReloadedMsg:
		if msg.err == nil {
			m.repos = msg.repos
			items := make([]list.Item, len(msg.repos))
			for i, repo := range msg.repos {
				items[i] = listItem{
					title:       repo.Name,
					description: repo.URL,
				}
			}
			m.repoList.SetItems(items)
			m.successMsg = fmt.Sprintf("Repository '%s' added successfully", m.newRepoName)
			m.mode = normalMode
		}
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.successMsg = fmt.Sprintf("Editor error: %v", msg.err)
			return m, nil
		}

		// Validate YAML
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(msg.content), &yamlData); err != nil {
			m.successMsg = fmt.Sprintf("Invalid YAML: %v", err)
			// Clean up temp file
			if msg.filePath != "" {
				os.Remove(msg.filePath)
			}
			return m, nil
		}

		// Save edited content and temp file path, then ask where to save
		m.editedContent = msg.content
		m.editTempFile = msg.filePath
		m.mode = saveEditMode
		m.searchInput.Reset()
		m.searchInput.Placeholder = "./custom-values.yaml"
		m.searchInput.Focus()
		return m, nil
	}

	switch m.state {
	case stateRepoList:
		m.repoList, cmd = m.repoList.Update(msg)
		cmds = append(cmds, cmd)
	case stateChartList:
		m.chartList, cmd = m.chartList.Update(msg)
		cmds = append(cmds, cmd)
	case stateChartDetail:
		m.versionList, cmd = m.versionList.Update(msg)
		cmds = append(cmds, cmd)
	case stateValueViewer:
		m.valuesView, cmd = m.valuesView.Update(msg)
		cmds = append(cmds, cmd)
	case stateDiffViewer:
		m.diffView, cmd = m.diffView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleBack() (tea.Model, tea.Cmd) {
	// Clear success message and search results
	m.successMsg = ""
	m.searchMatches = []int{}
	m.lastSearchQuery = ""
	m.horizontalOffset = 0

	if m.diffMode {
		m.diffMode = false
		return m, nil
	}

	switch m.state {
	case stateChartList:
		m.state = stateRepoList
		m.charts = nil
		m.chartList.SetItems([]list.Item{})
	case stateChartDetail:
		m.state = stateChartList
		m.versions = nil
		m.versionList.SetItems([]list.Item{})
	case stateValueViewer:
		m.state = stateChartDetail
		m.values = ""
		m.valuesLines = nil
	case stateDiffViewer:
		m.state = stateChartDetail
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	// Clear success message
	m.successMsg = ""

	switch m.state {
	case stateRepoList:
		idx := m.repoList.Index()
		if idx < len(m.repos) {
			m.selectedRepo = idx
			m.state = stateChartList
			m.loading = true
			return m, loadCharts(m.helmClient, m.chartCache, m.repos[idx].Name)
		}

	case stateChartList:
		idx := m.chartList.Index()
		if idx < len(m.charts) {
			m.selectedChart = idx
			m.state = stateChartDetail
			m.loading = true
			return m, loadVersions(m.helmClient, m.versionCache, m.charts[idx].Name)
		}

	case stateChartDetail:
		idx := m.versionList.Index()
		if idx < len(m.versions) {
			if m.diffMode {
				if idx == m.compareVersion {
					m.successMsg = "Please select a different version to compare"
					return m, nil
				}

				chartName := m.charts[m.selectedChart].Name
				version1 := m.versions[m.compareVersion].Version
				version2 := m.versions[idx].Version

				values1, found1 := m.cache.Get(chartName, version1)
				if !found1 {
					v, err := m.helmClient.GetChartValuesByVersion(chartName, version1)
					if err != nil {
						m.err = err
						m.diffMode = false
						return m, nil
					}
					values1 = v
					m.cache.Set(chartName, version1, values1)
				}

				values2, found2 := m.cache.Get(chartName, version2)
				if !found2 {
					v, err := m.helmClient.GetChartValuesByVersion(chartName, version2)
					if err != nil {
						m.err = err
						m.diffMode = false
						return m, nil
					}
					values2 = v
					m.cache.Set(chartName, version2, values2)
				}

				diffLines := ui.DiffYAML(values1, values2)
				diffContent := m.renderDiffContent(diffLines, version1, version2)

				// Save diff lines for search functionality
				m.diffLines = strings.Split(diffContent, "\n")

				m.diffView.SetContent(diffContent)
				m.state = stateDiffViewer
				m.diffMode = false
				return m, nil
			}

			m.selectedVersion = idx
			m.state = stateValueViewer
			m.loadingVals = true
			chartName := m.charts[m.selectedChart].Name
			version := m.versions[idx].Version
			return m, loadValuesByVersion(m.helmClient, m.cache, chartName, version)
		}
	}

	return m, nil
}

func (m model) handleSearch() (tea.Model, tea.Cmd) {
	if m.state == stateRepoList || m.state == stateChartList || m.state == stateChartDetail || m.state == stateValueViewer || m.state == stateDiffViewer {
		m.successMsg = "" // Clear success message
		m.mode = searchMode
		m.searchInput.Reset()
		m.searchInput.Placeholder = "Search..."
		m.searchInput.Focus()
	}
	return m, nil
}

func (m model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Clean up temp file if canceling save edit mode
		if m.mode == saveEditMode && m.editTempFile != "" {
			os.Remove(m.editTempFile)
			m.editTempFile = ""
			m.editedContent = ""
		}

		// Restore original lists if we were in search mode
		if m.mode == searchMode {
			switch m.state {
			case stateRepoList:
				items := make([]list.Item, len(m.repos))
				for i, repo := range m.repos {
					items[i] = listItem{
						title:       repo.Name,
						description: repo.URL,
					}
				}
				m.repoList.SetItems(items)

			case stateChartList:
				items := make([]list.Item, len(m.charts))
				for i, chart := range m.charts {
					name := chart.Name
					if m.selectedRepo < len(m.repos) {
						name = strings.TrimPrefix(name, m.repos[m.selectedRepo].Name+"/")
					}
					items[i] = listItem{
						title:       name,
						description: chart.Description,
					}
				}
				m.chartList.SetItems(items)

			case stateChartDetail:
				items := make([]list.Item, len(m.versions))
				for i, ver := range m.versions {
					desc := ""
					if ver.AppVersion != "" {
						desc = "App: " + ver.AppVersion
					}
					items[i] = listItem{
						title:       "v" + ver.Version,
						description: desc,
					}
				}
				m.versionList.SetItems(items)

			case stateValueViewer:
				// Clear search results
				m.searchMatches = []int{}
				m.lastSearchQuery = ""

			case stateDiffViewer:
				// Clear search results and restore original content
				m.searchMatches = []int{}
				m.lastSearchQuery = ""
				m.updateDiffViewWithSearch() // Restore original without highlights
			}
		}

		m.mode = normalMode
		m.searchInput.Blur()
		m.addRepoStep = 0
		return m, nil

	case "enter":
		switch m.mode {
		case searchMode:
			m.mode = normalMode
			m.searchInput.Blur()

		case addRepoMode:
			if m.addRepoStep == 0 {
				m.newRepoName = m.searchInput.Value()
				m.addRepoStep = 1
				m.searchInput.Reset()
				m.searchInput.Placeholder = "Repository URL..."
			} else {
				m.newRepoURL = m.searchInput.Value()
				m.mode = normalMode
				m.searchInput.Blur()
				return m, addRepository(m.helmClient, m.newRepoName, m.newRepoURL)
			}

		case exportValuesMode:
			path := m.searchInput.Value()
			if path == "" {
				path = "./values.yaml"
			}
			m.mode = normalMode
			m.searchInput.Blur()

			chartName := m.charts[m.selectedChart].Name
			if m.state == stateValueViewer && m.selectedVersion < len(m.versions) {
				version := m.versions[m.selectedVersion].Version
				return m, tea.Batch(func() tea.Msg {
					values, err := m.helmClient.GetChartValuesByVersion(chartName, version)
					if err != nil {
						return operationDoneMsg{err: err}
					}
					err = os.WriteFile(path, []byte(values), 0644)
					if err != nil {
						return operationDoneMsg{err: err}
					}
					return operationDoneMsg{success: fmt.Sprintf("Values (v%s) exported to %s", version, path)}
				})
			}
			return m, exportValues(m.helmClient, chartName, path)

		case templatePathMode:
			m.templatePath = m.searchInput.Value()
			if m.templatePath == "" {
				m.templatePath = "./output/"
			}
			m.mode = templateValuesMode
			m.searchInput.Reset()
			m.searchInput.Placeholder = "Values file (optional)..."

		case templateValuesMode:
			m.templateValues = m.searchInput.Value()
			m.mode = normalMode
			m.searchInput.Blur()

			chartName := m.charts[m.selectedChart].Name
			if m.state == stateValueViewer && m.selectedVersion < len(m.versions) {
				version := m.versions[m.selectedVersion].Version
				chartName = fmt.Sprintf("%s --version %s", chartName, version)
			}
			return m, generateTemplate(m.helmClient, chartName, m.templateValues, m.templatePath)

		case saveEditMode:
			path := m.searchInput.Value()
			if path == "" {
				path = "./custom-values.yaml"
			}
			m.mode = normalMode
			m.searchInput.Blur()

			// Expand home directory if needed
			if strings.HasPrefix(path, "~/") {
				home, err := os.UserHomeDir()
				if err == nil {
					path = filepath.Join(home, path[2:])
				}
			}

			// Save the edited values
			err := os.WriteFile(path, []byte(m.editedContent), 0644)

			// Clean up temp file
			if m.editTempFile != "" {
				os.Remove(m.editTempFile)
				m.editTempFile = ""
			}

			if err != nil {
				m.successMsg = fmt.Sprintf("Error saving: %v", err)
			} else {
				m.successMsg = fmt.Sprintf("✓ Values saved to %s", path)
			}
			m.editedContent = "" // Clear edited content
			return m, nil
		}
		return m, nil
	}

	m.searchInput, cmd = m.searchInput.Update(msg)

	if m.mode == searchMode && m.searchInput.Value() != "" {
		query := strings.ToLower(m.searchInput.Value())

		switch m.state {
		case stateRepoList:
			matches := fuzzy.Find(query, reposToStrings(m.repos))
			items := make([]list.Item, len(matches))
			for i, match := range matches {
				repo := m.repos[match.Index]
				items[i] = listItem{
					title:       repo.Name,
					description: repo.URL,
				}
			}
			m.repoList.SetItems(items)

		case stateChartList:
			matches := fuzzy.Find(query, chartsToStrings(m.charts))
			items := make([]list.Item, len(matches))
			for i, match := range matches {
				chart := m.charts[match.Index]
				name := chart.Name
				if m.selectedRepo < len(m.repos) {
					name = strings.TrimPrefix(name, m.repos[m.selectedRepo].Name+"/")
				}
				items[i] = listItem{
					title:       name,
					description: chart.Description,
				}
			}
			m.chartList.SetItems(items)

		case stateChartDetail:
			matches := fuzzy.Find(query, versionsToStrings(m.versions))
			items := make([]list.Item, len(matches))
			for i, match := range matches {
				ver := m.versions[match.Index]
				desc := ""
				if ver.AppVersion != "" {
					desc = "App: " + ver.AppVersion
				}
				items[i] = listItem{
					title:       "v" + ver.Version,
					description: desc,
				}
			}
			m.versionList.SetItems(items)

		case stateValueViewer:
			// Find all matches in values
			m.searchMatches = []int{}
			m.lastSearchQuery = query
			for i, line := range m.valuesLines {
				if strings.Contains(strings.ToLower(line), query) {
					m.searchMatches = append(m.searchMatches, i)
				}
			}

			// Update the view with highlighted search terms
			m.updateValuesViewWithSearch()

			// Jump to first match
			if len(m.searchMatches) > 0 {
				m.currentMatchIndex = 0
				targetLine := m.searchMatches[0]
				if targetLine > m.valuesView.Height/2 {
					targetLine = targetLine - m.valuesView.Height/2
				} else {
					targetLine = 0
				}
				m.valuesView.YOffset = targetLine
			}

		case stateDiffViewer:
			// Find all matches in diff
			m.searchMatches = []int{}
			m.lastSearchQuery = query
			for i, line := range m.diffLines {
				if strings.Contains(strings.ToLower(line), query) {
					m.searchMatches = append(m.searchMatches, i)
				}
			}

			// Update the view with highlighted search terms
			m.updateDiffViewWithSearch()

			// Jump to first match
			if len(m.searchMatches) > 0 {
				m.currentMatchIndex = 0
				targetLine := m.searchMatches[0]
				if targetLine > m.diffView.Height/2 {
					targetLine = targetLine - m.diffView.Height/2
				} else {
					targetLine = 0
				}
				m.diffView.YOffset = targetLine
			}
		}
	}

	return m, cmd
}

func reposToStrings(repos []helm.Repository) []string {
	result := make([]string, len(repos))
	for i, r := range repos {
		result[i] = r.Name
	}
	return result
}

func chartsToStrings(charts []helm.Chart) []string {
	result := make([]string, len(charts))
	for i, c := range charts {
		result[i] = c.Name
	}
	return result
}

func versionsToStrings(versions []helm.ChartVersion) []string {
	result := make([]string, len(versions))
	for i, v := range versions {
		result[i] = v.Version
	}
	return result
}

func (m model) jumpToMatch() model {
	if len(m.searchMatches) == 0 {
		return m
	}

	targetLine := m.searchMatches[m.currentMatchIndex]

	// Center the match on screen based on current state
	if m.state == stateValueViewer {
		if targetLine > m.valuesView.Height/2 {
			m.valuesView.YOffset = targetLine - m.valuesView.Height/2
		} else {
			m.valuesView.YOffset = 0
		}
	} else if m.state == stateDiffViewer {
		if targetLine > m.diffView.Height/2 {
			m.diffView.YOffset = targetLine - m.diffView.Height/2
		} else {
			m.diffView.YOffset = 0
		}
	}

	return m
}

func (m *model) updateValuesViewWithSearch() {
	lines := strings.Split(m.values, "\n")
	viewportWidth := m.valuesView.Width
	if viewportWidth <= 0 {
		viewportWidth = m.termWidth - 6 // Default to full screen minus borders/padding
	}

	// Get the current match line (only this one should be highlighted)
	var currentMatchLine int = -1
	if len(m.searchMatches) > 0 && m.currentMatchIndex < len(m.searchMatches) {
		currentMatchLine = m.searchMatches[m.currentMatchIndex]
	}

	query := strings.ToLower(m.lastSearchQuery)
	highlightedLines := make([]string, len(lines))

	for i, line := range lines {
		// Apply horizontal scrolling
		visibleLine := line
		hasMore := false

		// Calculate actual display width considering the line content
		if len(line) > m.horizontalOffset {
			visibleLine = line[m.horizontalOffset:]

			// Truncate if longer than viewport width
			if len(visibleLine) > viewportWidth-3 { // -3 for indicator
				visibleLine = visibleLine[:viewportWidth-3]
				hasMore = true
			}
		} else {
			visibleLine = ""
		}

		// Apply syntax highlighting
		var highlighted string
		// Only highlight if this is THE CURRENT match (not all matches)
		if i == currentMatchLine && query != "" {
			// This line is the CURRENT match - find and highlight it
			lowerLine := strings.ToLower(visibleLine)
			idx := strings.Index(lowerLine, query)
			if idx >= 0 && idx+len(query) <= len(visibleLine) {
				// Split the line into 3 parts
				before := visibleLine[:idx]
				match := visibleLine[idx : idx+len(query)]
				after := visibleLine[idx+len(query):]

				// Apply YAML highlighting to before and after, but not to match
				beforeHighlighted := ui.HighlightYAMLLine(before)
				afterHighlighted := ui.HighlightYAMLLine(after)
				matchHighlighted := highlightStyle.Render(match)

				highlighted = beforeHighlighted + matchHighlighted + afterHighlighted
			} else {
				// Fallback to normal highlighting if match not found in visible portion
				highlighted = ui.HighlightYAMLLine(visibleLine)
			}
		} else {
			// Normal line - just apply YAML highlighting
			highlighted = ui.HighlightYAMLLine(visibleLine)
		}

		// Add continuation indicator if line continues
		if hasMore {
			arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
			highlighted += arrowStyle.Render(" →")
		}

		highlightedLines[i] = highlighted
	}

	m.valuesView.SetContent(strings.Join(highlightedLines, "\n"))
}

func (m *model) updateDiffViewWithSearch() {
	if len(m.diffLines) == 0 {
		return
	}

	// Get the current match line (only this one should be highlighted)
	var currentMatchLine int = -1
	if len(m.searchMatches) > 0 && m.currentMatchIndex < len(m.searchMatches) {
		currentMatchLine = m.searchMatches[m.currentMatchIndex]
	}

	query := strings.ToLower(m.lastSearchQuery)
	highlightedLines := make([]string, len(m.diffLines))

	for i, line := range m.diffLines {
		// Only highlight if this is THE CURRENT match
		if i == currentMatchLine && query != "" {
			// This line is the CURRENT match - find and highlight it
			lowerLine := strings.ToLower(line)
			idx := strings.Index(lowerLine, query)
			if idx >= 0 && idx+len(query) <= len(line) {
				// Split the line into 3 parts
				before := line[:idx]
				match := line[idx : idx+len(query)]
				after := line[idx+len(query):]

				// Highlight the match in yellow background
				matchHighlighted := highlightStyle.Render(match)

				highlightedLines[i] = before + matchHighlighted + after
			} else {
				// Fallback to normal line if match not found
				highlightedLines[i] = line
			}
		} else {
			// Normal line - no highlighting
			highlightedLines[i] = line
		}
	}

	m.diffView.SetContent(strings.Join(highlightedLines, "\n"))
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf(" Error: %v ", m.err)) + "\n\n" +
			helpStyle.Render("Press 'q' to quit")
	}

	if m.state == stateHelp {
		return m.renderHelp()
	}

	var content string

	breadcrumb := m.getBreadcrumb()
	if breadcrumb != "" {
		content += breadcrumbStyle.Render(" " + breadcrumb + " ") + "\n\n"
	}

	// Show search info AFTER breadcrumb for better visibility
	if (m.state == stateValueViewer || m.state == stateDiffViewer) && len(m.searchMatches) > 0 {
		content += m.renderSearchHeader() + "\n"
	}

	switch m.state {
	case stateRepoList:
		content += m.renderRepoList()
	case stateChartList:
		content += m.renderChartList()
	case stateChartDetail:
		content += m.renderChartDetail()
	case stateValueViewer:
		content += m.renderValueViewer()
	case stateDiffViewer:
		content += m.renderDiffViewer()
	}

	footer := "\n"
	if m.successMsg != "" {
		footer += successStyle.Render(" " + m.successMsg + " ") + "\n"
	}

	if m.mode != normalMode {
		footer += m.renderInputPrompt() + "\n"
	}

	footer += "\n" + helpStyle.Render(" "+m.helpView.ShortHelpView(m.keys.ShortHelp())+" ")

	return content + footer
}

func (m model) renderSearchHeader() string {
	if len(m.searchMatches) == 0 {
		return ""
	}

	var header string

	// Match counter - always visible
	matchInfo := fmt.Sprintf(" Match %d/%d ", m.currentMatchIndex+1, len(m.searchMatches))
	header += infoStyle.Render(matchInfo) + " "

	// Show YAML path or line content based on state
	if m.state == stateValueViewer {
		matchLine := m.searchMatches[m.currentMatchIndex]
		yamlPath := ui.GetYAMLPath(m.valuesLines, matchLine)

		if yamlPath != "" {
			header += pathStyle.Render(" " + yamlPath + " ")
		} else if matchLine < len(m.valuesLines) {
			lineContent := strings.TrimSpace(m.valuesLines[matchLine])
			if len(lineContent) > 60 {
				lineContent = lineContent[:60] + "..."
			}
			header += pathStyle.Render(fmt.Sprintf(" Line %d: %s ", matchLine+1, lineContent))
		}
		header += " " + helpStyle.Render("n=next N=prev y=copy")
	} else if m.state == stateDiffViewer {
		matchLine := m.searchMatches[m.currentMatchIndex]
		if matchLine < len(m.diffLines) {
			lineContent := strings.TrimSpace(m.diffLines[matchLine])
			if len(lineContent) > 80 {
				lineContent = lineContent[:80] + "..."
			}
			header += pathStyle.Render(fmt.Sprintf(" %s ", lineContent))
		}
		header += " " + helpStyle.Render("n=next N=prev")
	}

	return header
}

func (m model) getBreadcrumb() string {
	parts := []string{"LazyHelm"}

	if m.selectedRepo < len(m.repos) {
		parts = append(parts, m.repos[m.selectedRepo].Name)
	}

	if m.state >= stateChartList && m.selectedChart < len(m.charts) {
		name := m.charts[m.selectedChart].Name
		if m.selectedRepo < len(m.repos) {
			name = strings.TrimPrefix(name, m.repos[m.selectedRepo].Name+"/")
		}
		parts = append(parts, name)
	}

	if m.state >= stateChartDetail && m.selectedVersion < len(m.versions) {
		parts = append(parts, "v"+m.versions[m.selectedVersion].Version)
	}

	if m.state == stateValueViewer {
		parts = append(parts, "values")
	}

	return strings.Join(parts, " > ")
}

func (m model) renderRepoList() string {
	if len(m.repos) == 0 {
		return "No repositories found.\nPress 'a' to add a repository.\n\nPress 'q' to quit\n"
	}
	return activePanelStyle.Render(m.repoList.View())
}

func (m model) renderChartList() string {
	if m.loading {
		return "Loading charts..."
	}
	if len(m.charts) == 0 {
		return "No charts found."
	}
	return activePanelStyle.Render(m.chartList.View())
}

func (m model) renderChartDetail() string {
	if m.loading {
		return activePanelStyle.Render("Loading versions...")
	}
	if len(m.versions) == 0 {
		return activePanelStyle.Render("No versions found.")
	}

	if m.diffMode {
		selectedVersion := "unknown"
		if m.compareVersion < len(m.versions) {
			selectedVersion = "v" + m.versions[m.compareVersion].Version
		}
		diffMsg := fmt.Sprintf(" Diff mode: First version = %s | Select second version to compare ", selectedVersion)
		return infoStyle.Render(diffMsg) + "\n\n" + activePanelStyle.Render(m.versionList.View())
	}

	return activePanelStyle.Render(m.versionList.View())
}

func (m model) renderValueViewer() string {
	if m.loadingVals {
		return activePanelStyle.Render("Loading values...")
	}
	if m.values == "" {
		return activePanelStyle.Render("No values available.")
	}

	var header string

	// Show horizontal scroll indicator if scrolled
	if m.horizontalOffset > 0 {
		scrollInfo := fmt.Sprintf(" ← Scrolled %d chars | use ←/→ or h/l to scroll ", m.horizontalOffset)
		header = helpStyle.Render(scrollInfo) + "\n\n"
	}

	if header != "" {
		return header + activePanelStyle.Render(m.valuesView.View())
	}

	return activePanelStyle.Render(m.valuesView.View())
}

func (m model) renderDiffViewer() string {
	return activePanelStyle.Render(m.diffView.View())
}

func (m model) renderDiffContent(diffLines []ui.DiffLine, version1, version2 string) string {
	header := fmt.Sprintf("Comparing v%s (old) → v%s (new)\n", version1, version2)
	header += fmt.Sprintf("Showing only changes (%d lines)\n\n", len(diffLines))

	var content strings.Builder
	content.WriteString(header)

	for _, line := range diffLines {
		switch line.Type {
		case "added":
			content.WriteString(addedStyle.Render("+ " + line.Line))
		case "removed":
			content.WriteString(removedStyle.Render("- " + line.Line))
		case "unchanged":
			content.WriteString("  " + line.Line)
		}
		content.WriteString("\n")
	}

	return content.String()
}

func (m model) renderHelp() string {
	help := "\n  LazyHelm - Help\n\n"
	help += "  Navigation:\n"
	help += "    ↑/k, ↓/j    Move up/down\n"
	help += "    ←/h, →/l    Scroll left/right (in values view)\n"
	help += "    enter       Select item / Go deeper\n"
	help += "    esc         Go back / Cancel\n"
	help += "    q           Quit\n\n"
	help += "  Actions:\n"
	help += "    /           Search in current view\n"
	help += "    n           Next search result (in values)\n"
	help += "    N           Previous search result (in values)\n"
	help += "    a           Add repository (in repo list)\n"
	help += "    v           View versions (in chart list)\n"
	help += "    w           Write/export values (in chart detail/values)\n"
	help += "    e           Edit in external editor (in values view)\n"
	help += "    t           Generate template (in chart detail/values)\n"
	help += "    y           Copy YAML path to clipboard (in values)\n"
	help += "    d           Diff two versions (in chart detail)\n"
	help += "    ?           Toggle help\n\n"
	help += "  Values View:\n"
	help += "    - Use ←/→ or h/l to scroll horizontally\n"
	help += "    - Lines ending with → continue beyond screen\n"
	help += "    - Press / to search, then type your query\n"
	help += "    - Shows match count and YAML path\n"
	help += "    - Use n/N to navigate between matches\n\n"
	help += "  External Editor:\n"
	help += "    - Press e in values view to open in editor\n"
	help += "    - Uses $EDITOR or $VISUAL environment variable\n"
	help += "    - Falls back to nvim, vim, or vi (in that order)\n"
	help += "    - Full editor features (syntax highlight, search, etc.)\n"
	help += "    - Save and quit editor to continue (:wq in vim/nvim)\n"
	help += "    - YAML is validated automatically\n"
	help += "    - You'll be prompted where to save the file\n"
	help += "    - Press ESC to cancel without saving\n\n"
	help += "  Diff Mode:\n"
	help += "    - Press d on a version (first version selected)\n"
	help += "    - Press enter on another version to compare\n"
	help += "    - Shows only differences with context\n\n"
	help += "  Press ? or esc to close this help\n"
	return help
}

func (m model) renderInputPrompt() string {
	var prompt string
	switch m.mode {
	case searchMode:
		prompt = "Search: " + m.searchInput.View()
	case addRepoMode:
		if m.addRepoStep == 0 {
			prompt = "Repository name: " + m.searchInput.View()
		} else {
			prompt = "Repository URL: " + m.searchInput.View()
		}
	case exportValuesMode:
		prompt = "Export to: " + m.searchInput.View()
	case templatePathMode:
		prompt = "Output directory: " + m.searchInput.View()
	case templateValuesMode:
		prompt = "Values file (optional): " + m.searchInput.View()
	case saveEditMode:
		prompt = "Save to: " + m.searchInput.View()
	default:
		return ""
	}
	return searchInputStyle.Render(" " + prompt + " ")
}

func openEditorCmd(content string) tea.Cmd {
	// Get editor from environment, fallback to nvim/vim/vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try to find nvim, vim, then vi in that order
		for _, cmd := range []string{"nvim", "vim", "vi"} {
			if path, err := exec.LookPath(cmd); err == nil {
				editor = path
				break
			}
		}
	}
	if editor == "" {
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("no editor found (tried nvim, vim, vi)")}
		}
	}

	// Parse editor command (might have flags like "code --wait")
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("invalid editor command")}
		}
	}

	// Create temp file with .yaml extension for proper syntax highlighting
	tmpfile, err := os.CreateTemp("", "lazyhelm-values-*.yaml")
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}
	tmpPath := tmpfile.Name()

	// Write content to temp file
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpPath)
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("failed to write temp file: %w", err)}
		}
	}
	tmpfile.Close()

	// Build command with editor and its args plus the temp file
	args := append(editorParts[1:], tmpPath)
	c := exec.Command(editorParts[0], args...)

	// Return tea.ExecProcess directly to properly handle terminal control
	return tea.ExecProcess(c, func(err error) tea.Msg {
		// This callback runs after the editor exits
		if err != nil {
			os.Remove(tmpPath)
			return editorFinishedMsg{err: fmt.Errorf("editor failed: %w", err), filePath: tmpPath}
		}

		// Read edited content
		editedContent, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			os.Remove(tmpPath)
			return editorFinishedMsg{err: fmt.Errorf("failed to read edited file: %w", readErr), filePath: tmpPath}
		}

		// Don't remove the file yet - we'll do it after saving
		return editorFinishedMsg{content: string(editedContent), filePath: tmpPath, err: nil}
	})
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
