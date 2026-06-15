package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shlomiuziel/sonosh/internal/macoshelper"
)

const (
	statusRefreshEvery   = 1 * time.Second
	queueRefreshEvery    = 5 * time.Second
	spinnerEvery         = 120 * time.Millisecond
	playlistPreviewLimit = 6
	queuePageSize        = 50
)

type Config struct {
	Timeout          time.Duration
	SearchService    string
	SearchCategory   string
	SearchLimit      int
	MacHelperPath    string
	Theme            string
	ThemeConfigPath  string
	Compact          bool
	LayoutConfigPath string
}

type Model struct {
	backend Backend
	config  Config
	helper  *macoshelper.Controller

	rooms            []Room
	roomIndex        int
	status           Status
	artURL           string
	artView          string
	artFallbackView  string
	artViews         map[string]string
	artFallbackViews map[string]string

	mode                    mode
	loading                 bool
	spinnerFrame            int
	err                     error
	message                 string
	searchQuery             string
	searchPreviewQuery      string
	searchCategory          string
	searchGeneration        int
	searchItems             []SearchResult
	searchIndex             int
	themeName               string
	searchPreviewItemID     string
	searchPreviewLoading    bool
	searchPreviewGeneration int
	searchPreviewItems      []SearchResult
	queueItems              []QueueItem
	queueTotal              int
	queueIndex              int
	queueOffset             int
	queueLoading            bool
	queueErr                error
	queueRefreshAt          time.Time
	playbackConfigIndex     int
	dashboardFocus          dashboardFocus
	compactLayout           bool

	width  int
	height int
}

type mode int
type dashboardFocus int

const (
	modeDashboard mode = iota
	modeSearch
	modePlaybackConfig
)

const (
	focusMain dashboardFocus = iota
	focusQueue
)

type roomsMsg struct {
	rooms []Room
	err   error
}

type statusMsg struct {
	status Status
	err    error
}

type albumArtMsg struct {
	url          string
	view         string
	fallbackView string
	err          error
}

type actionMsg struct {
	message string
	err     error
}

type crossfadeMsg struct {
	enabled bool
	err     error
}

type shuffleMsg struct {
	enabled bool
	err     error
}

type repeatMsg struct {
	mode string
	err  error
}

type queueMsg struct {
	items []QueueItem
	total int
	err   error
}

type queueActionMsg struct {
	message string
	err     error
}

type seekMsg struct {
	message string
	err     error
}

type searchMsg struct {
	query      string
	category   string
	generation int
	items      []SearchResult
	err        error
}

type playlistPreviewMsg struct {
	itemID     string
	generation int
	items      []SearchResult
	err        error
}

type tickMsg time.Time

type spinnerMsg time.Time

type macHelperStartedMsg struct {
	err error
}

type macHelperCommandMsg struct {
	command string
}

type macHelperErrorMsg struct {
	err error
}

func NewModel(backend Backend, cfg Config) Model {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	if strings.TrimSpace(cfg.SearchService) == "" {
		cfg.SearchService = "Spotify"
	}
	if strings.TrimSpace(cfg.SearchCategory) == "" {
		cfg.SearchCategory = "tracks"
	}
	if cfg.SearchLimit <= 0 {
		cfg.SearchLimit = 10
	}
	themeName := applyTheme(cfg.Theme)
	return Model{
		backend:        backend,
		config:         cfg,
		helper:         macoshelper.New(cfg.MacHelperPath),
		mode:           modeDashboard,
		loading:        true,
		searchCategory: cfg.SearchCategory,
		themeName:      themeName,
		compactLayout:  cfg.Compact,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(discoverCmd(m.backend, m.config.Timeout), tickCmd(), spinnerCmd(), macHelperStartCmd(m.helper))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen
	case tea.KeyMsg:
		return m.updateKey(msg)
	case roomsMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}
		m.rooms = msg.rooms
		if m.roomIndex >= len(m.rooms) {
			m.roomIndex = max(0, len(m.rooms)-1)
		}
		if len(m.rooms) == 0 {
			m.message = "No Sonos rooms found"
			return m, nil
		}
		m.loading = true
		m.queueLoading = true
		m.queueErr = nil
		m.queueItems = nil
		m.queueTotal = 0
		m.queueIndex = 0
		m.queueOffset = 0
		return m, tea.Batch(
			statusCmd(m.backend, m.config.Timeout, m.selectedRoom()),
			queueCmd(m.backend, m.config.Timeout, m.selectedRoom()),
		)
	case statusMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status = msg.status
			if url := strings.TrimSpace(msg.status.AlbumArt); url != "" {
				if url != m.artURL {
					m.artURL = url
					m.artView = ""
					m.artFallbackView = ""
					if view, ok := m.artViews[url]; ok {
						m.artView = view
						m.artFallbackView = m.artFallbackViews[url]
					} else {
						return m, fetchAlbumArtCmd(url, supportsKittyGraphics())
					}
				}
			} else {
				m.artURL = ""
				m.artView = ""
				m.artFallbackView = ""
			}
			m.publishNowPlaying()
		}
		return m, nil
	case albumArtMsg:
		if msg.err != nil {
			return m, nil
		}
		if msg.url != m.artURL {
			return m, nil
		}
		m.artView = msg.view
		m.artFallbackView = msg.fallbackView
		if m.artViews == nil {
			m.artViews = make(map[string]string)
		}
		if m.artFallbackViews == nil {
			m.artFallbackViews = make(map[string]string)
		}
		m.artViews[msg.url] = msg.view
		m.artFallbackViews[msg.url] = msg.fallbackView
		return m, nil
	case actionMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.message = msg.message
			if len(m.rooms) > 0 {
				m.loading = true
				return m, statusCmd(m.backend, m.config.Timeout, m.selectedRoom())
			}
		}
		return m, nil
	case crossfadeMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status.CrossfadeEnabled = msg.enabled
			m.status.CrossfadeKnown = true
			m.message = "crossfade " + onOff(msg.enabled)
			if len(m.rooms) > 0 {
				m.loading = true
				return m, statusCmd(m.backend, m.config.Timeout, m.selectedRoom())
			}
		}
		return m, nil
	case shuffleMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status.ShuffleEnabled = msg.enabled
			m.status.ShuffleKnown = true
			m.message = "shuffle " + onOff(msg.enabled)
			if len(m.rooms) > 0 {
				m.loading = true
				return m, statusCmd(m.backend, m.config.Timeout, m.selectedRoom())
			}
		}
		return m, nil
	case repeatMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status.RepeatMode = msg.mode
			m.status.RepeatKnown = true
			m.message = "repeat " + msg.mode
			if len(m.rooms) > 0 {
				m.loading = true
				return m, statusCmd(m.backend, m.config.Timeout, m.selectedRoom())
			}
		}
		return m, nil
	case queueMsg:
		m.queueLoading = false
		m.queueErr = msg.err
		m.queueRefreshAt = time.Now()
		if msg.err == nil {
			m.queueItems = msg.items
			m.queueTotal = msg.total
			m.clampQueueSelection()
		}
		return m, nil
	case queueActionMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.message = msg.message
			if len(m.rooms) > 0 {
				m.loading = true
				m.queueLoading = true
				m.queueErr = nil
				return m, tea.Batch(
					statusCmd(m.backend, m.config.Timeout, m.selectedRoom()),
					queueCmd(m.backend, m.config.Timeout, m.selectedRoom()),
					spinnerCmd(),
				)
			}
		}
		return m, nil
	case seekMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.message = msg.message
			if len(m.rooms) > 0 {
				m.loading = true
				return m, statusCmd(m.backend, m.config.Timeout, m.selectedRoom())
			}
		}
		return m, nil
	case searchMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil && msg.generation == m.searchGeneration && msg.query == m.searchQuery && msg.category == m.searchCategory {
			m.searchPreviewQuery = msg.query
			m.searchItems = msg.items
			m.searchIndex = 0
			m.message = fmt.Sprintf("%d search results", len(msg.items))
			m.resetPlaylistPreview()
			updated, cmd := m.previewSelectedSearchResult()
			return updated, cmd
		}
		return m, nil
	case playlistPreviewMsg:
		if msg.generation != m.searchPreviewGeneration || msg.itemID != m.searchPreviewItemID {
			return m, nil
		}
		m.searchPreviewLoading = false
		if msg.err != nil {
			m.searchPreviewItems = nil
			return m, nil
		}
		m.searchPreviewItems = msg.items
		return m, nil
	case tickMsg:
		now := time.Time(msg)
		if len(m.rooms) > 0 && !m.loading {
			cmds := []tea.Cmd{
				statusCmd(m.backend, m.config.Timeout, m.selectedRoom()),
				tickCmd(),
			}
			if !m.queueLoading && (m.queueRefreshAt.IsZero() || now.Sub(m.queueRefreshAt) >= queueRefreshEvery) {
				m.queueLoading = true
				m.queueErr = nil
				m.queueRefreshAt = now
				cmds = append(cmds, queueCmd(m.backend, m.config.Timeout, m.selectedRoom()))
			}
			return m, tea.Batch(cmds...)
		}
		return m, tickCmd()
	case spinnerMsg:
		if !m.loading {
			return m, nil
		}
		m.spinnerFrame++
		return m, spinnerCmd()
	case macHelperStartedMsg:
		if msg.err != nil {
			if errors.Is(msg.err, macoshelper.ErrUnavailable) {
				m.message = "mac helper unavailable"
			} else {
				m.message = "mac helper start failed: " + msg.err.Error()
			}
			return m, nil
		}
		m.message = "mac helper started"
		m.publishNowPlaying()
		return m, macHelperWaitCmd(m.helper)
	case macHelperCommandMsg:
		updated, cmd := m.handleMacHelperCommand(msg.command)
		return updated, tea.Batch(cmd, macHelperWaitCmd(m.helper))
	case macHelperErrorMsg:
		if msg.err != nil {
			m.message = "mac helper: " + msg.err.Error()
		}
		return m, macHelperWaitCmd(m.helper)
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+l":
		m.compactLayout = !m.compactLayout
		m.dashboardFocus = focusMain
		if err := SaveCompactLayout(m.config.LayoutConfigPath, m.compactLayout); err != nil {
			m.message = "layout save failed: " + err.Error()
		} else if m.compactLayout {
			m.message = "layout: compact"
		} else {
			m.message = "layout: full"
		}
		m.err = nil
		return m, nil
	case "ctrl+v":
		m.themeName = cycleTheme()
		if err := SaveThemeName(m.config.ThemeConfigPath, m.themeName); err != nil {
			m.message = "theme save failed: " + err.Error()
		} else {
			m.message = "theme: " + m.themeName
		}
		m.err = nil
		return m, nil
	}

	if m.mode == modeSearch {
		return m.updateSearchKey(msg)
	}
	if m.mode == modePlaybackConfig {
		return m.updatePlaybackConfigKey(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab":
		m.toggleDashboardFocus()
		m.err = nil
		return m, nil
	case "/":
		m.mode = modeSearch
		m.err = nil
		return m, tea.ClearScreen
	case "o":
		m.mode = modePlaybackConfig
		m.playbackConfigIndex = 0
		m.err = nil
		return m, tea.ClearScreen
	case "esc":
		m.dashboardFocus = focusMain
		m.mode = modeDashboard
		m.err = nil
		return m, nil
	case "r":
		m.loading = true
		m.dashboardFocus = focusMain
		m.err = nil
		return m, tea.Batch(discoverCmd(m.backend, m.config.Timeout), spinnerCmd())
	}

	return m.updateDashboardKey(msg)
}

func (m Model) updateDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dashboardFocus == focusQueue {
		return m.updateQueueKey(msg)
	}
	if len(m.rooms) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.roomIndex > 0 {
			m.roomIndex--
			m.loading = true
			m.queueLoading = true
			m.queueErr = nil
			return m, tea.Batch(
				statusCmd(m.backend, m.config.Timeout, m.selectedRoom()),
				queueCmd(m.backend, m.config.Timeout, m.selectedRoom()),
				spinnerCmd(),
			)
		}
	case "down", "j":
		if m.roomIndex < len(m.rooms)-1 {
			m.roomIndex++
			m.loading = true
			m.queueLoading = true
			m.queueErr = nil
			return m, tea.Batch(
				statusCmd(m.backend, m.config.Timeout, m.selectedRoom()),
				queueCmd(m.backend, m.config.Timeout, m.selectedRoom()),
				spinnerCmd(),
			)
		}
	case " ", "enter":
		m.loading = true
		action := "play"
		if strings.EqualFold(m.status.State, "PLAYING") {
			action = "pause"
		}
		return m, tea.Batch(transportCmd(m.backend, m.config.Timeout, m.selectedRoom(), action), spinnerCmd())
	case "s":
		m.loading = true
		return m, tea.Batch(transportCmd(m.backend, m.config.Timeout, m.selectedRoom(), "stop"), spinnerCmd())
	case "n":
		m.loading = true
		return m, tea.Batch(transportCmd(m.backend, m.config.Timeout, m.selectedRoom(), "next"), spinnerCmd())
	case "p":
		m.loading = true
		return m, tea.Batch(transportCmd(m.backend, m.config.Timeout, m.selectedRoom(), "previous"), spinnerCmd())
	case "left":
		m.loading = true
		return m, tea.Batch(seekCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Position, m.status.Duration, -5), spinnerCmd())
	case "right":
		m.loading = true
		return m, tea.Batch(seekCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Position, m.status.Duration, 5), spinnerCmd())
	case "+", "=":
		m.loading = true
		return m, tea.Batch(volumeCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Volume+5), spinnerCmd())
	case "-", "_":
		m.loading = true
		return m, tea.Batch(volumeCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Volume-5), spinnerCmd())
	case "m":
		m.loading = true
		return m, tea.Batch(muteCmd(m.backend, m.config.Timeout, m.selectedRoom()), spinnerCmd())
	}
	return m, nil
}

func (m Model) updateQueueKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dashboardFocus = focusMain
		return m, nil
	case "up", "k":
		if m.queueIndex > 0 {
			m.queueIndex--
			m.ensureQueueSelectionVisible()
		}
		return m, nil
	case "down", "j":
		if m.queueIndex < len(m.queueItems)-1 {
			m.queueIndex++
			m.ensureQueueSelectionVisible()
		}
		return m, nil
	case "enter":
		item, ok := m.selectedQueueItem()
		if !ok {
			return m, nil
		}
		m.loading = true
		m.err = nil
		return m, tea.Batch(queuePlayCmd(m.backend, m.config.Timeout, m.selectedRoom(), item.Position), spinnerCmd())
	case "x":
		item, ok := m.selectedQueueItem()
		if !ok {
			return m, nil
		}
		m.loading = true
		m.err = nil
		return m, tea.Batch(queueRemoveCmd(m.backend, m.config.Timeout, m.selectedRoom(), item.Position), spinnerCmd())
	case "X":
		if len(m.queueItems) == 0 {
			return m, nil
		}
		m.loading = true
		m.err = nil
		return m, tea.Batch(queueClearCmd(m.backend, m.config.Timeout, m.selectedRoom()), spinnerCmd())
	case "[":
		item, ok := m.selectedQueueItem()
		if !ok || item.Position <= 1 || m.queueIndex <= 0 {
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.queueIndex--
		m.ensureQueueSelectionVisible()
		return m, tea.Batch(queueMoveCmd(m.backend, m.config.Timeout, m.selectedRoom(), item.Position, item.Position-1), spinnerCmd())
	case "]":
		item, ok := m.selectedQueueItem()
		if !ok || m.queueIndex >= len(m.queueItems)-1 {
			return m, nil
		}
		m.loading = true
		m.err = nil
		m.queueIndex++
		m.ensureQueueSelectionVisible()
		return m, tea.Batch(queueMoveCmd(m.backend, m.config.Timeout, m.selectedRoom(), item.Position, item.Position+1), spinnerCmd())
	default:
		return m, nil
	}
}

func (m Model) updatePlaybackConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "esc", "q":
		m.mode = modeDashboard
		m.err = nil
		m.playbackConfigIndex = 0
		return m, nil
	case "up", "k":
		if m.playbackConfigIndex > 0 {
			m.playbackConfigIndex--
		}
		return m, nil
	case "down", "j":
		if m.playbackConfigIndex < 2 {
			m.playbackConfigIndex++
		}
		return m, nil
	case " ", "enter":
		if len(m.rooms) == 0 {
			return m, nil
		}
		m.loading = true
		m.err = nil
		if m.playbackConfigIndex == 1 {
			return m, tea.Batch(shuffleCmd(m.backend, m.config.Timeout, m.selectedRoom()), spinnerCmd())
		}
		if m.playbackConfigIndex == 2 {
			return m, tea.Batch(repeatCmd(m.backend, m.config.Timeout, m.selectedRoom()), spinnerCmd())
		}
		return m, tea.Batch(crossfadeCmd(m.backend, m.config.Timeout, m.selectedRoom()), spinnerCmd())
	default:
		return m, nil
	}
}

func (m Model) updateSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.searchIndex > 0 {
			m.searchIndex--
		}
		updated, cmd := m.previewSelectedSearchResult()
		return updated, cmd
	case "down", "j":
		if m.searchIndex < len(m.searchItems)-1 {
			m.searchIndex++
		}
		updated, cmd := m.previewSelectedSearchResult()
		return updated, cmd
	case "backspace", "ctrl+h":
		m.searchQuery = trimLastRune(m.searchQuery)
		m.searchIndex = 0
		m.searchGeneration++
		m.loading = true
		m.resetPlaylistPreview()
		return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
	case "ctrl+u":
		m.searchQuery = ""
		m.searchIndex = 0
		m.searchGeneration++
		m.loading = false
		m.searchPreviewQuery = ""
		m.searchItems = nil
		m.resetPlaylistPreview()
		return m, nil
	case "tab", "esc":
		m.mode = modeDashboard
		m.err = nil
		return m, nil
	case "ctrl+t":
		return m.setSearchCategory("tracks")
	case "ctrl+p":
		return m.setSearchCategory("playlists")
	case "enter":
		if len(m.rooms) == 0 {
			return m, nil
		}
		if len(m.searchItems) > 0 && m.searchPreviewQuery == m.searchQuery {
			m.loading = true
			return m, tea.Batch(playSearchCmd(m.backend, m.config, m.selectedRoom(), m.searchItems[m.searchIndex]), spinnerCmd())
		}
		if strings.TrimSpace(m.searchQuery) != "" {
			m.loading = true
			m.searchGeneration++
			return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
		}
		return m, nil
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.searchQuery += msg.String()
			m.searchIndex = 0
			m.searchGeneration++
			m.loading = true
			m.resetPlaylistPreview()
			return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
		}
		return m, nil
	}
}

func (m Model) setSearchCategory(category string) (tea.Model, tea.Cmd) {
	if m.searchCategory == category {
		return m, nil
	}
	m.searchCategory = category
	m.searchIndex = 0
	m.searchPreviewQuery = ""
	m.searchItems = nil
	m.resetPlaylistPreview()
	m.message = "searching " + category
	m.err = nil
	m.searchGeneration++
	if strings.TrimSpace(m.searchQuery) == "" || len(m.rooms) == 0 {
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
}

func (m Model) previewSelectedSearchResult() (tea.Model, tea.Cmd) {
	m.searchPreviewGeneration++
	m.searchPreviewItemID = ""
	m.searchPreviewLoading = false
	m.searchPreviewItems = nil
	if len(m.rooms) == 0 || len(m.searchItems) == 0 || m.searchIndex < 0 || m.searchIndex >= len(m.searchItems) {
		return m, nil
	}
	selected := m.searchItems[m.searchIndex]
	if !strings.EqualFold(strings.TrimSpace(selected.Item.ItemType), "playlist") {
		return m, nil
	}
	id := strings.TrimSpace(selected.Item.ID)
	if id == "" {
		return m, nil
	}
	m.searchPreviewItemID = id
	m.searchPreviewLoading = true
	gen := m.searchPreviewGeneration
	return m, browsePlaylistCmd(m.backend, m.config, m.selectedRoom(), selected, gen)
}

func (m *Model) resetPlaylistPreview() {
	m.searchPreviewGeneration++
	m.searchPreviewItemID = ""
	m.searchPreviewLoading = false
	m.searchPreviewItems = nil
}

func (m *Model) toggleDashboardFocus() {
	if !m.queuePaneVisible() || m.compactLayout {
		m.dashboardFocus = focusMain
		return
	}
	if m.dashboardFocus == focusQueue {
		m.dashboardFocus = focusMain
		return
	}
	m.dashboardFocus = focusQueue
}

func (m Model) queuePaneVisible() bool {
	if m.compactLayout {
		return false
	}
	width := m.width
	if width <= 0 {
		width = 100
	}
	return width-2 >= queueAtWidth
}

func (m *Model) clampQueueSelection() {
	if len(m.queueItems) == 0 {
		m.queueIndex = 0
		m.queueOffset = 0
		return
	}
	if m.queueIndex < 0 {
		m.queueIndex = 0
	}
	if m.queueIndex >= len(m.queueItems) {
		m.queueIndex = len(m.queueItems) - 1
	}
	m.ensureQueueSelectionVisible()
}

func (m *Model) ensureQueueSelectionVisible() {
	if len(m.queueItems) == 0 {
		m.queueOffset = 0
		return
	}
	visible := m.queueVisibleRows()
	if m.queueIndex < m.queueOffset {
		m.queueOffset = m.queueIndex
	}
	if m.queueIndex >= m.queueOffset+visible {
		m.queueOffset = m.queueIndex - visible + 1
	}
	maxOffset := max(0, len(m.queueItems)-visible)
	if m.queueOffset > maxOffset {
		m.queueOffset = maxOffset
	}
	if m.queueOffset < 0 {
		m.queueOffset = 0
	}
}

func (m Model) queueVisibleRows() int {
	height := m.height
	if height <= 0 {
		height = 24
	}
	return max(3, height-8)
}

func (m Model) selectedQueueItem() (QueueItem, bool) {
	if len(m.queueItems) == 0 || m.queueIndex < 0 || m.queueIndex >= len(m.queueItems) {
		return QueueItem{}, false
	}
	return m.queueItems[m.queueIndex], true
}

func (m Model) View() string {
	return m.renderApp()
}

func (m Model) Close() error {
	if m.helper == nil {
		return nil
	}
	return m.helper.Close()
}

func (m Model) selectedRoom() Room {
	if len(m.rooms) == 0 {
		return Room{}
	}
	return m.rooms[m.roomIndex]
}

func (m Model) handleMacHelperCommand(command string) (tea.Model, tea.Cmd) {
	if len(m.rooms) == 0 {
		m.message = "No Sonos room selected"
		return m, nil
	}
	action := ""
	switch strings.TrimSpace(command) {
	case "play":
		action = "play"
	case "pause":
		action = "pause"
	case "togglePlayPause":
		action = "play"
		if strings.EqualFold(m.status.State, "PLAYING") {
			action = "pause"
		}
	case "next":
		action = "next"
	case "previous":
		action = "previous"
	default:
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(transportCmd(m.backend, m.config.Timeout, m.selectedRoom(), action), spinnerCmd())
}

func (m Model) publishNowPlaying() {
	if m.helper == nil {
		return
	}
	if len(m.rooms) == 0 {
		m.helper.Clear()
		return
	}
	m.helper.Publish(nowPlayingMessage(m.selectedRoom(), m.status))
}

func discoverCmd(backend Backend, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		rooms, err := backend.Discover(ctx)
		return roomsMsg{rooms: rooms, err: err}
	}
}

func statusCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		status, err := backend.Status(ctx, room)
		return statusMsg{status: status, err: err}
	}
}

func queueCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		page, err := backend.Queue(ctx, room, 0, queuePageSize)
		return queueMsg{items: page.Items, total: page.TotalMatches, err: err}
	}
}

func transportCmd(backend Backend, timeout time.Duration, room Room, action string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.Transport(ctx, room, action)
		return actionMsg{message: action, err: err}
	}
}

func volumeCmd(backend Backend, timeout time.Duration, room Room, volume int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.SetVolume(ctx, room, volume)
		return actionMsg{message: fmt.Sprintf("volume %d", clamp(volume, 0, 100)), err: err}
	}
}

func muteCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.ToggleMute(ctx, room)
		return actionMsg{message: "mute toggled", err: err}
	}
}

func queuePlayCmd(backend Backend, timeout time.Duration, room Room, position int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.PlayQueuePosition(ctx, room, position)
		return queueActionMsg{message: fmt.Sprintf("queue play %d", position), err: err}
	}
}

func queueRemoveCmd(backend Backend, timeout time.Duration, room Room, position int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.RemoveQueuePosition(ctx, room, position)
		return queueActionMsg{message: fmt.Sprintf("removed queue %d", position), err: err}
	}
}

func queueClearCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.ClearQueue(ctx, room)
		return queueActionMsg{message: "queue cleared", err: err}
	}
}

func queueMoveCmd(backend Backend, timeout time.Duration, room Room, fromPosition, toPosition int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := backend.MoveQueuePosition(ctx, room, fromPosition, toPosition)
		return queueActionMsg{message: fmt.Sprintf("moved queue %d to %d", fromPosition, toPosition), err: err}
	}
}

func seekCmd(backend Backend, timeout time.Duration, room Room, position, duration string, deltaSeconds int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		target, err := backend.Scrub(ctx, room, position, duration, deltaSeconds)
		if err != nil {
			return seekMsg{message: "seek failed", err: err}
		}
		suffix := "+"
		if deltaSeconds < 0 {
			suffix = ""
		}
		return seekMsg{message: fmt.Sprintf("seek %s%ds -> %s", suffix, deltaSeconds, target), err: nil}
	}
}

func crossfadeCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		enabled, err := backend.ToggleCrossfade(ctx, room)
		return crossfadeMsg{enabled: enabled, err: err}
	}
}

func shuffleCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		enabled, err := backend.ToggleShuffle(ctx, room)
		return shuffleMsg{enabled: enabled, err: err}
	}
}

func repeatCmd(backend Backend, timeout time.Duration, room Room) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		mode, err := backend.ToggleRepeat(ctx, room)
		return repeatMsg{mode: mode, err: err}
	}
}

func searchCmd(backend Backend, cfg Config, room Room, category, query string, generation int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		items, err := backend.Search(ctx, room, cfg.SearchService, category, query, cfg.SearchLimit)
		return searchMsg{query: query, category: category, generation: generation, items: items, err: err}
	}
}

func playSearchCmd(backend Backend, cfg Config, room Room, result SearchResult) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		err := backend.PlaySearchResult(ctx, room, cfg.SearchService, result)
		return actionMsg{message: "playing " + result.Title(), err: err}
	}
}

func browsePlaylistCmd(backend Backend, cfg Config, room Room, result SearchResult, generation int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		items, err := backend.BrowsePlaylist(ctx, room, cfg.SearchService, result, playlistPreviewLimit)
		return playlistPreviewMsg{itemID: strings.TrimSpace(result.Item.ID), generation: generation, items: items, err: err}
	}
}

func macHelperStartCmd(helper *macoshelper.Controller) tea.Cmd {
	return func() tea.Msg {
		if helper == nil {
			return macHelperStartedMsg{err: macoshelper.ErrUnavailable}
		}
		return macHelperStartedMsg{err: helper.Start()}
	}
}

func macHelperWaitCmd(helper *macoshelper.Controller) tea.Cmd {
	return func() tea.Msg {
		if helper == nil {
			return macHelperErrorMsg{err: macoshelper.ErrUnavailable}
		}
		select {
		case command := <-helper.Commands():
			return macHelperCommandMsg{command: command}
		case err := <-helper.Errors():
			return macHelperErrorMsg{err: err}
		}
	}
}

func nowPlayingMessage(room Room, status Status) macoshelper.Message {
	msg := macoshelper.Message{
		Type:        "nowPlaying",
		Room:        room.Name,
		State:       helperState(status.State),
		Title:       strings.TrimSpace(status.Title),
		Artist:      strings.TrimSpace(status.Artist),
		Album:       strings.TrimSpace(status.Album),
		AlbumArtURL: strings.TrimSpace(status.AlbumArt),
	}
	if seconds, ok := parseClock(status.Position); ok {
		v := float64(seconds)
		msg.PositionSeconds = &v
	}
	if seconds, ok := parseClock(status.Duration); ok {
		v := float64(seconds)
		msg.DurationSeconds = &v
	}
	volume := clamp(status.Volume, 0, 100)
	msg.Volume = &volume
	muted := status.Muted
	msg.Muted = &muted
	return msg
}

func helperState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "PLAYING":
		return "playing"
	case "PAUSED_PLAYBACK":
		return "paused"
	case "STOPPED":
		return "stopped"
	default:
		return "idle"
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(statusRefreshEvery, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func spinnerCmd() tea.Cmd {
	return tea.Tick(spinnerEvery, func(t time.Time) tea.Msg {
		return spinnerMsg(t)
	})
}

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(s)
	if size <= 0 {
		return ""
	}
	return s[:len(s)-size]
}

func empty(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
