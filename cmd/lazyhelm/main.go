// Copyright 2025 Alessandro Pitocchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alessandropitocchi/lazyhelm/internal/artifacthub"
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
	// Set via ldflags during build
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var (
	// Stile fzf-like con sfondi per massima leggibilità
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero/Bianco (adaptive)
			Background(lipgloss.Color("105")). // Purple medio
			Bold(true).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")). // Grigio medio
			Padding(1, 2)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("141")). // Violet chiaro
				Padding(1, 2)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero/Bianco
			Background(lipgloss.Color("73")).  // Cyan/Teal
			Bold(true).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero
			Background(lipgloss.Color("120")). // Verde chiaro
			Bold(true).
			Padding(0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")). // Bianco
			Background(lipgloss.Color("196")). // Rosso brillante
			Bold(true).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")) // Grigio medio

	addedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero
			Background(lipgloss.Color("120")). // Verde chiaro
			Bold(true)

	removedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")). // Bianco
			Background(lipgloss.Color("160")). // Rosso medio
			Bold(true)

	modifiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero
			Background(lipgloss.Color("228")). // Giallo chiaro
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero
			Background(lipgloss.Color("141")). // Violet
			Bold(true).
			Padding(0, 2)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Nero
			Background(lipgloss.Color("228")). // Giallo chiaro
			Bold(true).
			Padding(0, 2)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("228")). // Giallo chiaro
			Foreground(lipgloss.Color("0")).   // Nero
			Bold(true)

	searchInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).   // Nero
				Background(lipgloss.Color("141")). // Violet
				Padding(0, 1).
				Bold(true)
)

type navigationState int

const (
	stateMainMenu navigationState = iota
	stateBrowseMenu
	stateRepoList
	stateChartList
	stateChartDetail
	stateValueViewer
	stateDiffViewer
	stateHelp
	stateArtifactHubSearch
	stateArtifactHubPackageDetail
	stateArtifactHubVersions
	stateClusterReleasesMenu
	stateNamespaceList
	stateReleaseList
	stateReleaseDetail
	stateReleaseHistory
	stateReleaseValues
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
	confirmRemoveRepoMode
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

	// Artifact Hub
	artifactHubClient  *artifacthub.Client
	ahPackages         []artifacthub.Package
	ahSelectedPackage  *artifacthub.Package
	ahPackageList      list.Model
	ahVersionList      list.Model
	ahSelectedPkg      int
	ahSelectedVersion  int
	ahLoading          bool

	// Cluster Releases
	releases           []helm.Release
	namespaces         []string
	selectedRelease    int
	selectedRevision   int
	selectedNamespace  string
	releaseHistory     []helm.ReleaseRevision
	releaseValues      string
	releaseValuesLines []string
	releaseStatus      *helm.ReleaseStatus
	kubeContext        string

	mainMenu              list.Model
	browseMenu            list.Model
	clusterReleasesMenu   list.Model
	namespaceList         list.Model
	releaseList           list.Model
	releaseHistoryList    list.Model
	releaseValuesView     viewport.Model
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
	ArtifactHub key.Binding
	RemoveRepo  key.Binding
	UpdateRepo  key.Binding
	ClearFilter key.Binding
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
	ArtifactHub: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "search artifact hub"),
	),
	RemoveRepo: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove repository"),
	),
	UpdateRepo: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update repository"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear filter"),
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

type repoRemovedMsg struct {
	repos    []helm.Repository
	repoName string
	err      error
}

type editorFinishedMsg struct {
	content  string
	filePath string
	err      error
}

type releasesLoadedMsg struct {
	releases []helm.Release
	err      error
}

type namespacesLoadedMsg struct {
	namespaces []string
	err        error
}

type releaseHistoryLoadedMsg struct {
	history []helm.ReleaseRevision
	err     error
}

type releaseValuesLoadedMsg struct {
	values string
	err    error
}

type releaseStatusLoadedMsg struct {
	status *helm.ReleaseStatus
	err    error
}

type kubeContextLoadedMsg struct {
	context string
	err     error
}

type artifactHubSearchMsg struct {
	packages []artifacthub.Package
	err      error
}

type artifactHubPackageMsg struct {
	pkg *artifacthub.Package
	err error
}

type clearSuccessMsgMsg struct{}

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

func loadReleases(client *helm.Client, namespace string) tea.Cmd {
	return func() tea.Msg {
		releases, err := client.ListReleases(namespace)
		return releasesLoadedMsg{releases: releases, err: err}
	}
}

func loadNamespaces(client *helm.Client) tea.Cmd {
	return func() tea.Msg {
		namespaces, err := client.ListNamespaces()
		return namespacesLoadedMsg{namespaces: namespaces, err: err}
	}
}

func loadReleaseHistory(client *helm.Client, releaseName, namespace string) tea.Cmd {
	return func() tea.Msg {
		history, err := client.GetReleaseHistory(releaseName, namespace)
		return releaseHistoryLoadedMsg{history: history, err: err}
	}
}

func loadReleaseValues(client *helm.Client, releaseName, namespace string) tea.Cmd {
	return func() tea.Msg {
		values, err := client.GetReleaseValues(releaseName, namespace)
		return releaseValuesLoadedMsg{values: values, err: err}
	}
}

func loadReleaseStatus(client *helm.Client, releaseName, namespace string) tea.Cmd {
	return func() tea.Msg {
		status, err := client.GetReleaseStatus(releaseName, namespace)
		return releaseStatusLoadedMsg{status: status, err: err}
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

func searchArtifactHub(client *artifacthub.Client, query string) tea.Cmd {
	return func() tea.Msg {
		packages, err := client.SearchPackages(query, 50)
		if err != nil {
			return artifactHubSearchMsg{err: err}
		}
		return artifactHubSearchMsg{packages: packages}
	}
}

func loadArtifactHubPackage(client *artifacthub.Client, repoName, packageName string) tea.Cmd {
	return func() tea.Msg {
		pkg, err := client.GetPackageDetails(repoName, packageName)
		if err != nil {
			return artifactHubPackageMsg{err: err}
		}
		return artifactHubPackageMsg{pkg: pkg}
	}
}

func clearSuccessMsgAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearSuccessMsgMsg{}
	})
}

// Helper to set success message and auto-clear after 3 seconds
func (m *model) setSuccessMsg(msg string) tea.Cmd {
	m.successMsg = msg
	return clearSuccessMsgAfter(3 * time.Second)
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

	// Create custom delegate with fzf-like colors (background for selected items)
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("0")).    // Nero/Bianco (adaptive)
		Background(lipgloss.Color("141")).  // Violet - stile fzf
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("0")).    // Nero/Bianco
		Background(lipgloss.Color("141"))   // Violet
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "255"})   // Grigio scuro su chiaro, bianco su scuro
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "250"})   // Grigio medio

	repoList := list.New(repoItems, delegate, 0, 0)
	repoList.Title = "Repositories"
	repoList.SetShowStatusBar(false)
	repoList.SetFilteringEnabled(true)
	repoList.Styles.Title = titleStyle
	repoList.Styles.FilterPrompt = searchInputStyle
	repoList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	chartDelegate := list.NewDefaultDelegate()
	chartDelegate.Styles = delegate.Styles
	chartList := list.New([]list.Item{}, chartDelegate, 0, 0)
	chartList.Title = "Charts"
	chartList.SetShowStatusBar(false)
	chartList.SetFilteringEnabled(true)
	chartList.Styles.Title = titleStyle
	chartList.Styles.FilterPrompt = searchInputStyle
	chartList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	versionDelegate := list.NewDefaultDelegate()
	versionDelegate.Styles = delegate.Styles
	versionList := list.New([]list.Item{}, versionDelegate, 0, 0)
	versionList.Title = "Versions"
	versionList.SetShowStatusBar(false)
	versionList.SetFilteringEnabled(true)
	versionList.Styles.Title = titleStyle
	versionList.Styles.FilterPrompt = searchInputStyle
	versionList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	valuesView := viewport.New(0, 0)
	diffView := viewport.New(0, 0)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."

	helpView := help.New()

	// Artifact Hub lists
	ahPackageDelegate := list.NewDefaultDelegate()
	ahPackageDelegate.Styles = delegate.Styles
	ahPackageList := list.New([]list.Item{}, ahPackageDelegate, 0, 0)
	ahPackageList.Title = "Artifact Hub"
	ahPackageList.SetShowStatusBar(false)
	ahPackageList.SetFilteringEnabled(true)
	ahPackageList.Styles.Title = titleStyle
	ahPackageList.Styles.FilterPrompt = searchInputStyle
	ahPackageList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	ahVersionDelegate := list.NewDefaultDelegate()
	ahVersionDelegate.Styles = delegate.Styles
	ahVersionList := list.New([]list.Item{}, ahVersionDelegate, 0, 0)
	ahVersionList.Title = "Versions"
	ahVersionList.SetShowStatusBar(false)
	ahVersionList.SetFilteringEnabled(true)
	ahVersionList.Styles.Title = titleStyle
	ahVersionList.Styles.FilterPrompt = searchInputStyle
	ahVersionList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	// Main Menu
	menuItems := []list.Item{
		listItem{title: "Browse Repositories", description: "Browse Helm repositories and charts"},
		listItem{title: "Cluster Releases", description: "View deployed Helm releases"},
		listItem{title: "Settings", description: "Configure LazyHelm settings (Coming Soon)"},
	}
	mainMenuDelegate := list.NewDefaultDelegate()
	mainMenuDelegate.Styles = delegate.Styles
	mainMenu := list.New(menuItems, mainMenuDelegate, 0, 0)
	mainMenu.Title = "LazyHelm"
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.Styles.Title = titleStyle

	// Browse Menu (submenu for Browse Repositories)
	browseMenuItems := []list.Item{
		listItem{title: "Local Repositories", description: "Browse your configured Helm repositories"},
		listItem{title: "Search Artifact Hub", description: "Search charts on Artifact Hub"},
	}
	browseMenuDelegate := list.NewDefaultDelegate()
	browseMenuDelegate.Styles = delegate.Styles
	browseMenu := list.New(browseMenuItems, browseMenuDelegate, 0, 0)
	browseMenu.Title = "Browse Repositories"
	browseMenu.SetShowStatusBar(false)
	browseMenu.SetFilteringEnabled(false)
	browseMenu.Styles.Title = titleStyle

	// Cluster Releases Menu
	clusterReleasesMenuItems := []list.Item{
		listItem{title: "All Namespaces", description: "View releases from all namespaces"},
		listItem{title: "Select Namespace", description: "Choose a specific namespace"},
	}
	clusterReleasesMenuDelegate := list.NewDefaultDelegate()
	clusterReleasesMenuDelegate.Styles = delegate.Styles
	clusterReleasesMenu := list.New(clusterReleasesMenuItems, clusterReleasesMenuDelegate, 0, 0)
	clusterReleasesMenu.Title = "Cluster Releases"
	clusterReleasesMenu.SetShowStatusBar(false)
	clusterReleasesMenu.SetFilteringEnabled(false)
	clusterReleasesMenu.Styles.Title = titleStyle

	// Namespace List
	namespaceDelegate := list.NewDefaultDelegate()
	namespaceDelegate.Styles = delegate.Styles
	namespaceList := list.New([]list.Item{}, namespaceDelegate, 0, 0)
	namespaceList.Title = "Namespaces"
	namespaceList.SetShowStatusBar(false)
	namespaceList.SetFilteringEnabled(true)
	namespaceList.Styles.Title = titleStyle
	namespaceList.Styles.FilterPrompt = searchInputStyle
	namespaceList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	// Release List
	releaseDelegate := list.NewDefaultDelegate()
	releaseDelegate.Styles = delegate.Styles
	releaseList := list.New([]list.Item{}, releaseDelegate, 0, 0)
	releaseList.Title = "Releases"
	releaseList.SetShowStatusBar(false)
	releaseList.SetFilteringEnabled(true)
	releaseList.Styles.Title = titleStyle
	releaseList.Styles.FilterPrompt = searchInputStyle
	releaseList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))

	// Release History List
	releaseHistoryDelegate := list.NewDefaultDelegate()
	releaseHistoryDelegate.Styles = delegate.Styles
	releaseHistoryList := list.New([]list.Item{}, releaseHistoryDelegate, 0, 0)
	releaseHistoryList.Title = "Release History"
	releaseHistoryList.SetShowStatusBar(false)
	releaseHistoryList.SetFilteringEnabled(false)
	releaseHistoryList.Styles.Title = titleStyle

	// Release Values View
	releaseValuesView := viewport.New(0, 0)

	return model{
		helmClient:        client,
		cache:             cache,
		chartCache:        make(map[string]chartCacheEntry),
		versionCache:      make(map[string]versionCacheEntry),
		state:             stateMainMenu,
		mode:              normalMode,
		repos:             repos,
		artifactHubClient:     artifacthub.NewClient(),
		ahPackageList:         ahPackageList,
		ahVersionList:         ahVersionList,
		mainMenu:              mainMenu,
		browseMenu:            browseMenu,
		clusterReleasesMenu:   clusterReleasesMenu,
		namespaceList:         namespaceList,
		releaseList:           releaseList,
		releaseHistoryList:    releaseHistoryList,
		releaseValuesView:     releaseValuesView,
		repoList:              repoList,
		chartList:         chartList,
		versionList:       versionList,
		valuesView:        valuesView,
		diffView:          diffView,
		searchInput:       searchInput,
		helpView:          helpView,
		keys:              defaultKeys,
		err:               err,
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

		m.mainMenu.SetSize(w/2, h)
		m.browseMenu.SetSize(w/2, h)
		m.repoList.SetSize(w/3, h)
		m.chartList.SetSize(w/2, h)
		m.versionList.SetSize(w/3, h)

		// Artifact Hub lists
		m.ahPackageList.SetSize(w-4, h)
		m.ahVersionList.SetSize(w/3, h)

		// Cluster Releases lists
		m.clusterReleasesMenu.SetSize(w/2, h)
		m.namespaceList.SetSize(w/3, h)
		m.releaseList.SetSize(w-4, h)
		m.releaseHistoryList.SetSize(w/3, h)

		// Values view takes full screen
		m.valuesView.Width = msg.Width - 6  // Full width minus border padding
		m.valuesView.Height = msg.Height - 8 // Full height minus header/footer

		m.diffView.Width = msg.Width - 6
		m.diffView.Height = msg.Height - 8

		m.releaseValuesView.Width = msg.Width - 6
		m.releaseValuesView.Height = msg.Height - 8

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
			if (m.state == stateArtifactHubPackageDetail || m.state == stateArtifactHubVersions) && m.ahSelectedPackage != nil {
				// Add repo from Artifact Hub - only ask for name, URL is auto-filled
				m.mode = addRepoMode
				m.addRepoStep = 0
				m.newRepoURL = m.ahSelectedPackage.Repository.URL // Pre-fill URL
				m.searchInput.Reset()
				m.searchInput.Placeholder = fmt.Sprintf("Repository name (default: %s)...", m.ahSelectedPackage.Repository.Name)
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keys.RemoveRepo):
			if m.state == stateRepoList && len(m.repos) > 0 {
				// Enter confirmation mode - use selected item to handle filtered lists
				selectedItem := m.repoList.SelectedItem()
				if selectedItem != nil {
					item := selectedItem.(listItem)
					m.mode = confirmRemoveRepoMode
					m.searchInput.Reset()
					m.searchInput.Placeholder = fmt.Sprintf("Remove '%s'? (y/n)", item.title)
					m.searchInput.Focus()
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.UpdateRepo):
			if m.state == stateRepoList && len(m.repos) > 0 {
				selectedItem := m.repoList.SelectedItem()
				if selectedItem != nil {
					item := selectedItem.(listItem)
					repoName := item.title
					return m, func() tea.Msg {
						err := m.helmClient.UpdateRepository(repoName)
						if err != nil {
							return operationDoneMsg{err: err}
						}
						return operationDoneMsg{success: fmt.Sprintf("Repository '%s' updated successfully", repoName)}
					}
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Export):
			if m.state == stateChartDetail || m.state == stateValueViewer || m.state == stateReleaseValues {
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

		case key.Matches(msg, m.keys.ArtifactHub):
			if m.state == stateRepoList {
				m.mode = searchMode
				m.searchInput.Reset()
				m.searchInput.Placeholder = "Search Artifact Hub..."
				m.searchInput.Focus()
				m.state = stateArtifactHubSearch
			}
			return m, nil

		case key.Matches(msg, m.keys.ClearFilter):
			// Clear filters and restore full lists
			var clearCmd tea.Cmd
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
				clearCmd = m.setSuccessMsg("Filter cleared")

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
				clearCmd = m.setSuccessMsg("Filter cleared")

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
				clearCmd = m.setSuccessMsg("Filter cleared")

			case stateArtifactHubSearch:
				items := make([]list.Item, len(m.ahPackages))
				for i, pkg := range m.ahPackages {
					badges := pkg.GetBadges()
					stars := fmt.Sprintf("⭐%d", pkg.Stars)
					security := pkg.SecurityReport.GetSecurityBadge()

					desc := fmt.Sprintf("%s | %s %s | %s", pkg.Repository.DisplayName, stars, badges, security)
					items[i] = listItem{
						title:       pkg.Name,
						description: desc,
					}
				}
				m.ahPackageList.SetItems(items)
				clearCmd = m.setSuccessMsg("Filter cleared")

			case stateReleaseList:
				items := make([]list.Item, len(m.releases))
				for i, release := range m.releases {
					desc := fmt.Sprintf("%s | %s | %s", release.Namespace, release.Chart, release.Status)
					items[i] = listItem{
						title:       release.Name,
						description: desc,
					}
				}
				m.releaseList.SetItems(items)
				clearCmd = m.setSuccessMsg("Filter cleared")
			}
			return m, clearCmd

		case key.Matches(msg, m.keys.Versions):
			if m.state == stateChartList && len(m.charts) > 0 {
				m.state = stateChartDetail
				m.loading = true
				idx := m.chartList.Index()
				if idx < len(m.charts) {
					return m, loadVersions(m.helmClient, m.versionCache, m.charts[idx].Name)
				}
			}
			if m.state == stateArtifactHubPackageDetail && m.ahSelectedPackage != nil {
				m.state = stateArtifactHubVersions
			}
			if m.state == stateReleaseDetail && m.selectedRelease < len(m.releases) {
				release := m.releases[m.selectedRelease]
				m.state = stateReleaseValues
				m.loadingVals = true
				return m, loadReleaseValues(m.helmClient, release.Name, release.Namespace)
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
				var copyCmd tea.Cmd
				if yamlPath != "" {
					err := clipboard.WriteAll(yamlPath)
					if err != nil {
						copyCmd = m.setSuccessMsg("Failed to copy to clipboard")
					} else {
						copyCmd = m.setSuccessMsg("Copied: " + yamlPath)
					}
				} else {
					copyCmd = m.setSuccessMsg("No YAML path found for current line")
				}
				return m, copyCmd
			}
			if m.state == stateReleaseValues && len(m.releaseValuesLines) > 0 {
				var lineNum int
				// If we have search matches, use the current match line
				if len(m.searchMatches) > 0 && m.currentMatchIndex < len(m.searchMatches) {
					lineNum = m.searchMatches[m.currentMatchIndex]
				} else {
					// Otherwise use the current viewport position (center of visible area)
					lineNum = m.releaseValuesView.YOffset + m.releaseValuesView.Height/2
					if lineNum >= len(m.releaseValuesLines) {
						lineNum = len(m.releaseValuesLines) - 1
					}
				}

				yamlPath := ui.GetYAMLPath(m.releaseValuesLines, lineNum)
				var copyCmd tea.Cmd
				if yamlPath != "" {
					err := clipboard.WriteAll(yamlPath)
					if err != nil {
						copyCmd = m.setSuccessMsg("Failed to copy to clipboard")
					} else {
						copyCmd = m.setSuccessMsg("Copied: " + yamlPath)
					}
				} else {
					copyCmd = m.setSuccessMsg("No YAML path found for current line")
				}
				return m, copyCmd
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
					return m, m.setSuccessMsg("No values to edit")
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
				editorCmd := m.setSuccessMsg(fmt.Sprintf("Opening %s...", editor))
				return m, tea.Batch(editorCmd, openEditorCmd(m.values))
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
			// 'h' for history in release detail view
			if m.state == stateReleaseDetail && m.selectedRelease < len(m.releases) {
				m.state = stateReleaseHistory
				return m, nil
			}
			// Horizontal scroll in value viewers
			if m.state == stateValueViewer {
				if m.horizontalOffset > 0 {
					m.horizontalOffset -= 5 // Scroll by 5 characters
					if m.horizontalOffset < 0 {
						m.horizontalOffset = 0
					}
					m.updateValuesViewWithSearch()
				}
			} else if m.state == stateReleaseValues {
				if m.horizontalOffset > 0 {
					m.horizontalOffset -= 5 // Scroll by 5 characters
					if m.horizontalOffset < 0 {
						m.horizontalOffset = 0
					}
					m.updateReleaseValuesViewWithSearch()
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Right):
			if m.state == stateValueViewer {
				m.horizontalOffset += 5 // Scroll by 5 characters
				m.updateValuesViewWithSearch()
			} else if m.state == stateReleaseValues {
				m.horizontalOffset += 5 // Scroll by 5 characters
				m.updateReleaseValuesViewWithSearch()
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
			return m, nil
		} else {
			return m, m.setSuccessMsg(msg.success)
		}


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
			m.mode = normalMode
			return m, m.setSuccessMsg(fmt.Sprintf("Repository '%s' added successfully", m.newRepoName))
		}
		return m, nil

	case repoRemovedMsg:
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
			m.mode = normalMode
			return m, m.setSuccessMsg(fmt.Sprintf("Repository '%s' removed successfully", msg.repoName))
		}
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			return m, m.setSuccessMsg(fmt.Sprintf("Editor error: %v", msg.err))
		}

		// Validate YAML
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(msg.content), &yamlData); err != nil {
			// Clean up temp file
			if msg.filePath != "" {
				os.Remove(msg.filePath)
			}
			return m, m.setSuccessMsg(fmt.Sprintf("Invalid YAML: %v", err))
		}

		// Save edited content and temp file path, then ask where to save
		m.editedContent = msg.content
		m.editTempFile = msg.filePath
		m.mode = saveEditMode
		m.searchInput.Reset()
		m.searchInput.Placeholder = "./custom-values.yaml"
		m.searchInput.Focus()
		return m, nil

	case artifactHubSearchMsg:
		m.ahLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.ahPackages = msg.packages
		items := make([]list.Item, len(msg.packages))
		for i, pkg := range msg.packages {
			badges := pkg.GetBadges()
			stars := fmt.Sprintf("⭐%d", pkg.Stars)
			security := pkg.SecurityReport.GetSecurityBadge()

			desc := fmt.Sprintf("%s | %s %s | %s", pkg.Repository.DisplayName, stars, badges, security)
			items[i] = listItem{
				title:       pkg.Name,
				description: desc,
			}
		}
		m.ahPackageList.SetItems(items)
		return m, nil

	case artifactHubPackageMsg:
		m.ahLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.ahSelectedPackage = msg.pkg

		// Populate version list
		if len(msg.pkg.AvailableVersions) > 0 {
			items := make([]list.Item, len(msg.pkg.AvailableVersions))
			for i, ver := range msg.pkg.AvailableVersions {
				desc := ""
				if ver.ContainsSecurityUpdates {
					desc = "🛡️ Security update"
				}
				if ver.Prerelease {
					desc += " [Pre-release]"
				}
				items[i] = listItem{
					title:       "v" + ver.Version,
					description: desc,
				}
			}
			m.ahVersionList.SetItems(items)
		}
		return m, nil

	case clearSuccessMsgMsg:
		m.successMsg = ""
		return m, nil

	case releasesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.releases = msg.releases
		items := make([]list.Item, len(msg.releases))
		for i, release := range msg.releases {
			desc := fmt.Sprintf("%s | %s | %s", release.Namespace, release.Chart, release.Status)
			items[i] = listItem{
				title:       release.Name,
				description: desc,
			}
		}
		m.releaseList.SetItems(items)
		return m, nil

	case namespacesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.namespaces = msg.namespaces
		items := make([]list.Item, len(msg.namespaces))
		for i, ns := range msg.namespaces {
			items[i] = listItem{
				title:       ns,
				description: "Kubernetes namespace",
			}
		}
		m.namespaceList.SetItems(items)
		return m, nil

	case releaseHistoryLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.releaseHistory = msg.history
		items := make([]list.Item, len(msg.history))
		for i, rev := range msg.history {
			desc := fmt.Sprintf("%s | %s | %s", rev.Status, rev.Chart, rev.Updated)
			items[i] = listItem{
				title:       fmt.Sprintf("Revision %d", rev.Revision),
				description: desc,
			}
		}
		m.releaseHistoryList.SetItems(items)
		return m, nil

	case releaseValuesLoadedMsg:
		m.loadingVals = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.releaseValues = msg.values
		m.releaseValuesLines = strings.Split(msg.values, "\n")
		highlighted := ui.HighlightYAMLContent(msg.values)
		m.releaseValuesView.SetContent(highlighted)
		return m, nil

	case releaseStatusLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.releaseStatus = msg.status
		return m, nil

	case kubeContextLoadedMsg:
		if msg.err != nil {
			// Context error is not fatal, just don't show it
			m.kubeContext = "unknown"
		} else {
			m.kubeContext = msg.context
		}
		return m, nil
	}

	switch m.state {
	case stateMainMenu:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
		cmds = append(cmds, cmd)
	case stateBrowseMenu:
		m.browseMenu, cmd = m.browseMenu.Update(msg)
		cmds = append(cmds, cmd)
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
	case stateArtifactHubSearch:
		m.ahPackageList, cmd = m.ahPackageList.Update(msg)
		cmds = append(cmds, cmd)
	case stateArtifactHubPackageDetail:
		m.ahVersionList, cmd = m.ahVersionList.Update(msg)
		cmds = append(cmds, cmd)
	case stateArtifactHubVersions:
		m.ahVersionList, cmd = m.ahVersionList.Update(msg)
		cmds = append(cmds, cmd)
	case stateClusterReleasesMenu:
		m.clusterReleasesMenu, cmd = m.clusterReleasesMenu.Update(msg)
		cmds = append(cmds, cmd)
	case stateNamespaceList:
		m.namespaceList, cmd = m.namespaceList.Update(msg)
		cmds = append(cmds, cmd)
	case stateReleaseList:
		m.releaseList, cmd = m.releaseList.Update(msg)
		cmds = append(cmds, cmd)
	case stateReleaseDetail:
		m.releaseHistoryList, cmd = m.releaseHistoryList.Update(msg)
		cmds = append(cmds, cmd)
	case stateReleaseHistory:
		m.releaseHistoryList, cmd = m.releaseHistoryList.Update(msg)
		cmds = append(cmds, cmd)
	case stateReleaseValues:
		m.releaseValuesView, cmd = m.releaseValuesView.Update(msg)
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
	case stateBrowseMenu:
		m.state = stateMainMenu
	case stateRepoList:
		m.state = stateBrowseMenu
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
	case stateArtifactHubSearch:
		m.state = stateBrowseMenu
		m.ahPackages = nil
		m.ahPackageList.SetItems([]list.Item{})
	case stateArtifactHubPackageDetail:
		m.state = stateArtifactHubSearch
		m.ahSelectedPackage = nil
		m.ahVersionList.SetItems([]list.Item{})
	case stateArtifactHubVersions:
		m.state = stateArtifactHubPackageDetail
	case stateClusterReleasesMenu:
		m.state = stateMainMenu
	case stateNamespaceList:
		m.state = stateClusterReleasesMenu
		m.namespaces = nil
		m.namespaceList.SetItems([]list.Item{})
	case stateReleaseList:
		if m.selectedNamespace == "" {
			// Came from "All Namespaces"
			m.state = stateClusterReleasesMenu
		} else {
			// Came from "Select Namespace"
			m.state = stateNamespaceList
		}
		m.releases = nil
		m.releaseList.SetItems([]list.Item{})
	case stateReleaseDetail:
		m.state = stateReleaseList
	case stateReleaseHistory:
		m.state = stateReleaseDetail
	case stateReleaseValues:
		m.state = stateReleaseHistory
		m.releaseValues = ""
		m.releaseValuesLines = nil
		m.selectedRevision = 0
		m.horizontalOffset = 0
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	// Clear success message
	m.successMsg = ""

	switch m.state {
	case stateMainMenu:
		selectedItem := m.mainMenu.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			switch item.title {
			case "Browse Repositories":
				m.state = stateBrowseMenu
				return m, nil
			case "Cluster Releases":
				m.state = stateClusterReleasesMenu
				// Load kubectl context
				return m, func() tea.Msg {
					ctx, err := m.helmClient.GetCurrentContext()
					if err != nil {
						return kubeContextLoadedMsg{err: err}
					}
					return kubeContextLoadedMsg{context: ctx}
				}
			case "Settings":
				return m, m.setSuccessMsg("Feature coming soon!")
			}
		}

	case stateBrowseMenu:
		selectedItem := m.browseMenu.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			switch item.title {
			case "Local Repositories":
				m.state = stateRepoList
				return m, nil
			case "Search Artifact Hub":
				m.mode = searchMode
				m.searchInput.Reset()
				m.searchInput.Placeholder = "Search Artifact Hub..."
				m.searchInput.Focus()
				m.state = stateArtifactHubSearch
				return m, nil
			}
		}

	case stateClusterReleasesMenu:
		selectedItem := m.clusterReleasesMenu.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			switch item.title {
			case "All Namespaces":
				m.state = stateReleaseList
				m.selectedNamespace = "" // Empty means all namespaces
				m.loading = true
				return m, loadReleases(m.helmClient, "")
			case "Select Namespace":
				m.state = stateNamespaceList
				m.loading = true
				return m, loadNamespaces(m.helmClient)
			}
		}

	case stateNamespaceList:
		selectedItem := m.namespaceList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			m.selectedNamespace = item.title
			m.state = stateReleaseList
			m.loading = true
			return m, loadReleases(m.helmClient, item.title)
		}

	case stateReleaseList:
		selectedItem := m.releaseList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			// Find the release by name
			for i, release := range m.releases {
				if release.Name == item.title {
					m.selectedRelease = i
					m.state = stateReleaseDetail
					m.loading = true
					// Load both history and status for the detail view
					return m, tea.Batch(
						loadReleaseHistory(m.helmClient, release.Name, release.Namespace),
						loadReleaseStatus(m.helmClient, release.Name, release.Namespace),
					)
				}
			}
		}

	case stateReleaseHistory:
		selectedItem := m.releaseHistoryList.SelectedItem()
		if selectedItem != nil && m.selectedRelease < len(m.releases) {
			item := selectedItem.(listItem)
			release := m.releases[m.selectedRelease]
			// Find the revision by matching the title "Revision X"
			for _, rev := range m.releaseHistory {
				revTitle := fmt.Sprintf("Revision %d", rev.Revision)
				if revTitle == item.title {
					m.selectedRevision = rev.Revision
					m.state = stateReleaseValues
					m.loadingVals = true
					return m, func() tea.Msg {
						values, err := m.helmClient.GetReleaseValuesByRevision(release.Name, release.Namespace, rev.Revision)
						if err != nil {
							return releaseValuesLoadedMsg{err: err}
						}
						return releaseValuesLoadedMsg{values: values}
					}
				}
			}
		}

	case stateRepoList:
		selectedItem := m.repoList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			// Find the correct index in the full list by matching the title
			for i, repo := range m.repos {
				if repo.Name == item.title {
					m.selectedRepo = i
					m.state = stateChartList
					m.loading = true
					return m, loadCharts(m.helmClient, m.chartCache, repo.Name)
				}
			}
		}

	case stateChartList:
		selectedItem := m.chartList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			// Find the correct index in the full list by matching the title
			for i, chart := range m.charts {
				chartName := chart.Name
				if m.selectedRepo < len(m.repos) {
					chartName = strings.TrimPrefix(chartName, m.repos[m.selectedRepo].Name+"/")
				}
				if chartName == item.title {
					m.selectedChart = i
					m.state = stateChartDetail
					m.loading = true
					return m, loadVersions(m.helmClient, m.versionCache, m.charts[i].Name)
				}
			}
		}

	case stateChartDetail:
		selectedItem := m.versionList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			// Find the correct index in the full list by matching the title
			var selectedIdx int = -1
			for i, ver := range m.versions {
				if "v"+ver.Version == item.title {
					selectedIdx = i
					break
				}
			}

			if selectedIdx >= 0 {
				if m.diffMode {
					if selectedIdx == m.compareVersion {
						return m, m.setSuccessMsg("Please select a different version to compare")
					}

					chartName := m.charts[m.selectedChart].Name
					version1 := m.versions[m.compareVersion].Version
					version2 := m.versions[selectedIdx].Version

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

				m.selectedVersion = selectedIdx
				m.state = stateValueViewer
				m.loadingVals = true
				chartName := m.charts[m.selectedChart].Name
				version := m.versions[selectedIdx].Version
				return m, loadValuesByVersion(m.helmClient, m.cache, chartName, version)
			}
		}

	case stateArtifactHubSearch:
		selectedItem := m.ahPackageList.SelectedItem()
		if selectedItem != nil {
			item := selectedItem.(listItem)
			// Find the correct package in the full list by matching the name
			for i, pkg := range m.ahPackages {
				if pkg.Name == item.title {
					m.ahSelectedPkg = i
					m.state = stateArtifactHubPackageDetail
					m.ahLoading = true
					return m, loadArtifactHubPackage(m.artifactHubClient, pkg.Repository.Name, pkg.Name)
				}
			}
		}

	case stateArtifactHubVersions:
		// Can't view values from Artifact Hub - need to add repo first
		return m, m.setSuccessMsg("Add the repository first (press 'a'), then browse it from the main menu to view values")
	}

	return m, nil
}

func (m model) handleSearch() (tea.Model, tea.Cmd) {
	if m.state == stateRepoList || m.state == stateChartList || m.state == stateChartDetail || m.state == stateValueViewer || m.state == stateDiffViewer || m.state == stateReleaseValues || m.state == stateReleaseList {
		m.successMsg = "" // Clear success message
		m.mode = searchMode
		m.searchInput.Reset()
		m.searchInput.Placeholder = "Search..."
		m.searchInput.Focus()
	}
	if m.state == stateArtifactHubSearch {
		// Allow searching again in Artifact Hub
		m.successMsg = ""
		m.mode = searchMode
		m.searchInput.Reset()
		m.searchInput.Placeholder = "Search Artifact Hub..."
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

			case stateReleaseValues:
				// Clear search results
				m.searchMatches = []int{}
				m.lastSearchQuery = ""

			case stateReleaseList:
				// Restore full release list
				items := make([]list.Item, len(m.releases))
				for i, release := range m.releases {
					desc := fmt.Sprintf("%s | %s | %s", release.Namespace, release.Chart, release.Status)
					items[i] = listItem{
						title:       release.Name,
						description: desc,
					}
				}
				m.releaseList.SetItems(items)

			case stateDiffViewer:
				// Clear search results and restore original content
				m.searchMatches = []int{}
				m.lastSearchQuery = ""
				m.updateDiffViewWithSearch() // Restore original without highlights

			case stateArtifactHubSearch:
				// Return to repo list
				m.state = stateRepoList
				m.ahPackages = nil
				m.ahPackageList.SetItems([]list.Item{})
			}
		}

		m.mode = normalMode
		m.searchInput.Blur()
		m.addRepoStep = 0
		m.newRepoURL = "" // Reset pre-filled URL
		return m, nil

	case "enter":
		switch m.mode {
		case searchMode:
			// Check if we're in Artifact Hub search state
			if m.state == stateArtifactHubSearch {
				query := m.searchInput.Value()
				if query != "" {
					m.mode = normalMode
					m.searchInput.Blur()
					m.ahLoading = true
					return m, searchArtifactHub(m.artifactHubClient, query)
				}
			}
			m.mode = normalMode
			m.searchInput.Blur()

		case addRepoMode:
			if m.addRepoStep == 0 {
				inputName := m.searchInput.Value()
				// If coming from Artifact Hub and no name provided, use default
				if inputName == "" && m.newRepoURL != "" && m.ahSelectedPackage != nil {
					inputName = m.ahSelectedPackage.Repository.Name
				}
				m.newRepoName = inputName

				// If URL is already set (from Artifact Hub), skip asking for URL
				if m.newRepoURL != "" {
					m.mode = normalMode
					m.searchInput.Blur()
					return m, addRepository(m.helmClient, m.newRepoName, m.newRepoURL)
				}

				// Otherwise ask for URL (normal flow)
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

			if m.state == stateReleaseValues {
				return m, func() tea.Msg {
					err := os.WriteFile(path, []byte(m.releaseValues), 0644)
					if err != nil {
						return operationDoneMsg{err: err}
					}
					if m.selectedRevision > 0 {
						return operationDoneMsg{success: fmt.Sprintf("Values (revision %d) exported to %s", m.selectedRevision, path)}
					}
					return operationDoneMsg{success: fmt.Sprintf("Values exported to %s", path)}
				}
			}

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

			m.editedContent = "" // Clear edited content
			if err != nil {
				return m, m.setSuccessMsg(fmt.Sprintf("Error saving: %v", err))
			} else {
				return m, m.setSuccessMsg(fmt.Sprintf("✓ Values saved to %s", path))
			}

		case confirmRemoveRepoMode:
			response := strings.ToLower(m.searchInput.Value())
			m.mode = normalMode
			m.searchInput.Blur()

			if response == "y" || response == "yes" {
				// Remove the repository - use selected item to handle filtered lists
				selectedItem := m.repoList.SelectedItem()
				if selectedItem != nil {
					item := selectedItem.(listItem)
					repoName := item.title
					return m, func() tea.Msg {
						err := m.helmClient.RemoveRepository(repoName)
						if err != nil {
							return operationDoneMsg{err: err}
						}

						// Reload repositories
						repos, repoErr := m.helmClient.ListRepositories()
						if repoErr != nil {
							return operationDoneMsg{success: fmt.Sprintf("Repository '%s' removed, but failed to reload list", repoName)}
						}

						return repoRemovedMsg{repos: repos, repoName: repoName}
					}
				}
			}
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

		case stateReleaseList:
			matches := fuzzy.Find(query, releasesToStrings(m.releases))
			items := make([]list.Item, len(matches))
			for i, match := range matches {
				release := m.releases[match.Index]
				desc := fmt.Sprintf("%s | %s | %s", release.Namespace, release.Chart, release.Status)
				items[i] = listItem{
					title:       release.Name,
					description: desc,
				}
			}
			m.releaseList.SetItems(items)

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

		case stateReleaseValues:
			// Find all matches in release values
			m.searchMatches = []int{}
			m.lastSearchQuery = query
			for i, line := range m.releaseValuesLines {
				if strings.Contains(strings.ToLower(line), query) {
					m.searchMatches = append(m.searchMatches, i)
				}
			}

			// Update the view with highlighted search terms
			m.updateReleaseValuesViewWithSearch()

			// Jump to first match
			if len(m.searchMatches) > 0 {
				m.currentMatchIndex = 0
				targetLine := m.searchMatches[0]
				if targetLine > m.releaseValuesView.Height/2 {
					targetLine = targetLine - m.releaseValuesView.Height/2
				} else {
					targetLine = 0
				}
				m.releaseValuesView.YOffset = targetLine
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

func releasesToStrings(releases []helm.Release) []string {
	result := make([]string, len(releases))
	for i, r := range releases {
		result[i] = r.Name
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
	} else if m.state == stateReleaseValues {
		if targetLine > m.releaseValuesView.Height/2 {
			m.releaseValuesView.YOffset = targetLine - m.releaseValuesView.Height/2
		} else {
			m.releaseValuesView.YOffset = 0
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
			arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
			highlighted += arrowStyle.Render(" →")
		}

		highlightedLines[i] = highlighted
	}

	m.valuesView.SetContent(strings.Join(highlightedLines, "\n"))
}

func (m *model) updateReleaseValuesViewWithSearch() {
	lines := strings.Split(m.releaseValues, "\n")
	viewportWidth := m.releaseValuesView.Width
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
			arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
			highlighted += arrowStyle.Render(" →")
		}

		highlightedLines[i] = highlighted
	}

	m.releaseValuesView.SetContent(strings.Join(highlightedLines, "\n"))
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
	if (m.state == stateValueViewer || m.state == stateReleaseValues || m.state == stateDiffViewer) && len(m.searchMatches) > 0 {
		content += m.renderSearchHeader() + "\n"
	}

	switch m.state {
	case stateMainMenu:
		content += m.renderMainMenu()
	case stateBrowseMenu:
		content += m.renderBrowseMenu()
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
	case stateArtifactHubSearch:
		content += m.renderArtifactHubSearch()
	case stateArtifactHubPackageDetail:
		content += m.renderArtifactHubPackageDetail()
	case stateArtifactHubVersions:
		content += m.renderArtifactHubVersions()
	case stateClusterReleasesMenu:
		content += m.renderClusterReleasesMenu()
	case stateNamespaceList:
		content += m.renderNamespaceList()
	case stateReleaseList:
		content += m.renderReleaseList()
	case stateReleaseDetail:
		content += m.renderReleaseDetail()
	case stateReleaseHistory:
		content += m.renderReleaseHistory()
	case stateReleaseValues:
		content += m.renderReleaseValues()
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
	} else if m.state == stateReleaseValues {
		matchLine := m.searchMatches[m.currentMatchIndex]
		yamlPath := ui.GetYAMLPath(m.releaseValuesLines, matchLine)

		if yamlPath != "" {
			header += pathStyle.Render(" " + yamlPath + " ")
		} else if matchLine < len(m.releaseValuesLines) {
			lineContent := strings.TrimSpace(m.releaseValuesLines[matchLine])
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

	// Cluster Releases navigation
	if m.state >= stateClusterReleasesMenu && m.state <= stateReleaseValues {
		parts = append(parts, "Cluster Releases")

		if m.state == stateNamespaceList {
			parts = append(parts, "Select Namespace")
		}

		if m.state >= stateReleaseList && m.selectedNamespace != "" {
			parts = append(parts, m.selectedNamespace)
		}

		if m.state >= stateReleaseList && m.selectedRelease < len(m.releases) {
			parts = append(parts, m.releases[m.selectedRelease].Name)
		}

		if m.state == stateReleaseHistory {
			parts = append(parts, "history")
		}

		if m.state == stateReleaseValues {
			if m.selectedRevision > 0 {
				parts = append(parts, fmt.Sprintf("revision %d", m.selectedRevision))
			}
			parts = append(parts, "values")
		}

		return strings.Join(parts, " > ")
	}

	// Artifact Hub navigation
	if m.state == stateArtifactHubSearch {
		parts = append(parts, "Artifact Hub")
		return strings.Join(parts, " > ")
	}

	if m.state == stateArtifactHubPackageDetail && m.ahSelectedPackage != nil {
		parts = append(parts, "Artifact Hub", m.ahSelectedPackage.Name)
		return strings.Join(parts, " > ")
	}

	if m.state == stateArtifactHubVersions && m.ahSelectedPackage != nil {
		parts = append(parts, "Artifact Hub", m.ahSelectedPackage.Name, "Versions")
		return strings.Join(parts, " > ")
	}

	// Regular Helm navigation
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

func (m model) renderMainMenu() string {
	return activePanelStyle.Render(m.mainMenu.View())
}

func (m model) renderBrowseMenu() string {
	return activePanelStyle.Render(m.browseMenu.View())
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
	help += "    esc         Go back to previous screen\n"
	help += "    q           Quit application\n"
	help += "    ?           Toggle this help screen\n\n"

	help += "  Search & Filter:\n"
	help += "    /           Search/filter in current view\n"
	help += "    c           Clear search filter\n"
	help += "    n           Next search result\n"
	help += "    N           Previous search result\n\n"

	help += "  Repository Management:\n"
	help += "    a           Add new repository\n"
	help += "    r           Remove selected repository\n"
	help += "    u           Update repository index (helm repo update)\n"
	help += "    s           Search Artifact Hub\n\n"

	help += "  Chart & Version Actions:\n"
	help += "    v           View all versions (in chart list)\n"
	help += "    d           Diff two versions (select first, then second)\n\n"

	help += "  Values View:\n"
	help += "    e           Edit values in external editor ($EDITOR)\n"
	help += "    w           Write/export values to file\n"
	help += "    t           Generate Helm template\n"
	help += "    y           Copy YAML path to clipboard\n"
	help += "    ←/→, h/l    Scroll horizontally for long lines\n\n"

	help += "  Tips:\n"
	help += "    • Horizontal scroll: Lines ending with → continue beyond screen\n"
	help += "    • Search shows match count and current YAML path\n"
	help += "    • Editor: Uses $EDITOR/$VISUAL, falls back to nvim→vim→vi\n"
	help += "    • Diff: Press d on first version, enter on second to compare\n"
	help += "    • YAML validation happens automatically when editing\n\n"

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
	case confirmRemoveRepoMode:
		prompt = m.searchInput.Placeholder + " " + m.searchInput.View()
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

func (m model) renderArtifactHubSearch() string {
	if m.ahLoading {
		return activePanelStyle.Render("Searching Artifact Hub...")
	}
	if len(m.ahPackages) == 0 {
		return activePanelStyle.Render("No packages found.\nTry a different search query.\n\nPress 'esc' to go back")
	}

	hint := "\n" + helpStyle.Render("  enter: view details | a: add repository | esc: back  ")
	return activePanelStyle.Render(m.ahPackageList.View()) + hint
}

func (m model) renderArtifactHubPackageDetail() string {
	if m.ahLoading {
		return activePanelStyle.Render("Loading package details...")
	}

	if m.ahSelectedPackage == nil {
		return activePanelStyle.Render("No package selected")
	}

	pkg := m.ahSelectedPackage

	// Build info panel - full screen
	info := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")).
		Width(m.termWidth - 8).
		Render(fmt.Sprintf(
			"%s %s\n\n"+
				"Repository: %s\n"+
				"URL: %s\n"+
				"Latest Version: %s\n"+
				"App Version: %s\n"+
				"Stars: ⭐%d\n"+
				"Security: %s\n"+
				"Signed: %s\n\n"+
				"%s\n\n"+
				"Available versions: %d",
			pkg.Name,
			pkg.GetBadges(),
			pkg.Repository.DisplayName,
			pkg.Repository.URL,
			pkg.Version,
			pkg.AppVersion,
			pkg.Stars,
			pkg.SecurityReport.GetSecurityBadge(),
			func() string {
				if pkg.Signed {
					return "🔒 Yes"
				}
				return "No"
			}(),
			pkg.Description,
			len(pkg.AvailableVersions),
		))

	hint := "\n" + helpStyle.Render("  a: add repository | v: view versions | esc: back  ")

	return info + hint
}

func (m model) renderArtifactHubVersions() string {
	if len(m.ahSelectedPackage.AvailableVersions) == 0 {
		return activePanelStyle.Render("No versions available")
	}

	hint := "\n" + helpStyle.Render("  a: add repository to view values | esc: back  ")
	return activePanelStyle.Render(m.ahVersionList.View()) + hint
}

func (m model) renderClusterReleasesMenu() string {
	var header string
	if m.kubeContext != "" {
		contextInfo := fmt.Sprintf(" Kubectl Context: %s ", m.kubeContext)
		header = infoStyle.Render(contextInfo) + "\n\n"
	}
	return header + activePanelStyle.Render(m.clusterReleasesMenu.View())
}

func (m model) renderNamespaceList() string {
	if m.loading {
		return "Loading namespaces..."
	}
	if len(m.namespaces) == 0 {
		return "No namespaces with Helm releases found."
	}
	return activePanelStyle.Render(m.namespaceList.View())
}

func (m model) renderReleaseList() string {
	if m.loading {
		return "Loading releases..."
	}
	if len(m.releases) == 0 {
		return "No releases found."
	}

	var header string
	if m.selectedNamespace == "" {
		header = infoStyle.Render(" Showing releases from all namespaces ") + "\n\n"
	} else {
		header = infoStyle.Render(fmt.Sprintf(" Namespace: %s ", m.selectedNamespace)) + "\n\n"
	}

	return header + activePanelStyle.Render(m.releaseList.View())
}

func (m model) renderReleaseDetail() string {
	if m.loading {
		return activePanelStyle.Render("Loading release details...")
	}

	if m.selectedRelease >= len(m.releases) {
		return activePanelStyle.Render("No release selected.")
	}

	release := m.releases[m.selectedRelease]

	var content strings.Builder

	// Release header
	content.WriteString(infoStyle.Render(fmt.Sprintf(" Release: %s ", release.Name)) + "\n\n")

	// Status section
	if m.releaseStatus != nil {
		content.WriteString("Status: " + m.releaseStatus.Status + "\n")
		if m.releaseStatus.Description != "" {
			content.WriteString("Description: " + m.releaseStatus.Description + "\n")
		}
		content.WriteString("\n")
	}

	// Release info
	content.WriteString(fmt.Sprintf("Namespace:  %s\n", release.Namespace))
	content.WriteString(fmt.Sprintf("Chart:      %s\n", release.Chart))
	content.WriteString(fmt.Sprintf("App Version: %s\n", release.AppVersion))
	content.WriteString(fmt.Sprintf("Updated:    %s\n", release.Updated))
	content.WriteString("\n")

	// History section
	content.WriteString("Revision History:\n")
	if len(m.releaseHistory) > 0 {
		for _, rev := range m.releaseHistory {
			revStr := fmt.Sprintf("  Revision %d - %s (%s) - %s\n",
				rev.Revision, rev.Status, rev.Chart, rev.Updated)
			content.WriteString(revStr)
		}
	} else {
		content.WriteString("  Loading...\n")
	}
	content.WriteString("\n")

	// Notes section
	if m.releaseStatus != nil && m.releaseStatus.Notes != "" {
		content.WriteString("Notes:\n")
		// Indent each line of notes
		noteLines := strings.Split(m.releaseStatus.Notes, "\n")
		for _, line := range noteLines {
			content.WriteString("  " + line + "\n")
		}
		content.WriteString("\n")
	}

	content.WriteString(helpStyle.Render("  v: view current values | h: interactive history | esc: back  "))

	return activePanelStyle.Render(content.String())
}

func (m model) renderReleaseHistory() string {
	if m.loading {
		return activePanelStyle.Render("Loading revision history...")
	}
	if len(m.releaseHistory) == 0 {
		return activePanelStyle.Render("No revision history found.")
	}

	hint := "\n" + helpStyle.Render("  Select a revision to view its values | esc: back  ")
	return activePanelStyle.Render(m.releaseHistoryList.View()) + hint
}

func (m model) renderReleaseValues() string {
	if m.loadingVals {
		return activePanelStyle.Render("Loading values...")
	}
	if m.releaseValues == "" {
		return activePanelStyle.Render("No values available.")
	}

	var header string
	// Show which revision we're viewing
	if m.selectedRevision > 0 {
		header = infoStyle.Render(fmt.Sprintf(" Revision %d Values ", m.selectedRevision)) + "\n\n"
	}

	// Show horizontal scroll indicator if scrolled
	if m.horizontalOffset > 0 {
		scrollInfo := fmt.Sprintf(" ← Scrolled %d chars | use ←/→ or h/l to scroll ", m.horizontalOffset)
		header += helpStyle.Render(scrollInfo) + "\n\n"
	}

	if header != "" {
		return header + activePanelStyle.Render(m.releaseValuesView.View())
	}

	return activePanelStyle.Render(m.releaseValuesView.View())
}

func main() {
	// Check for version flag
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" || arg == "version" {
			fmt.Printf("lazyhelm version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built: %s\n", date)
			os.Exit(0)
		}
		if arg == "--help" || arg == "-h" || arg == "help" {
			fmt.Println("LazyHelm - A fast, intuitive Terminal User Interface (TUI) for managing Helm charts")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  lazyhelm           Start the TUI")
			fmt.Println("  lazyhelm --version Show version information")
			fmt.Println("  lazyhelm --help    Show this help message")
			fmt.Println()
			fmt.Println("For more information, visit: https://github.com/alessandropitocchi/lazyhelm")
			os.Exit(0)
		}
	}

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
