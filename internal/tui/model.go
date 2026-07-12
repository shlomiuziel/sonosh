package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	playlistShelfLimit   = 6
	queuePageSize        = 50
)

type Config struct {
	Timeout             time.Duration
	SearchService       string
	SearchCategory      string
	SearchLimit         int
	MacHelperPath       string
	HelperHUDEnabled    bool
	HelperHUDPosition   string
	HelperHUDConfigPath string
	Theme               string
	ThemeConfigPath     string
	Compact             bool
	LayoutConfigPath    string
	CarouselPath        string
	LastRoomConfigPath  string
}

type Model struct {
	backend Backend
	config  Config
	helper  *macoshelper.Controller

	rooms            []Room
	roomIndex        int
	lastRoom         lastRoomSelection
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
	searchOffset            int
	searchFocus             searchFocus
	themeName               string
	searchPreviewItemID     string
	searchPreviewLoading    bool
	searchPreviewGeneration int
	searchPreviewItems      []SearchResult
	carouselStore           playlistCarouselStore
	carouselLoading         bool
	carouselGeneration      int
	carouselPinned          []SearchResult
	carouselRecent          []SearchResult
	carouselTab             int
	carouselIndex           int
	carouselThumbViews      map[string]string
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
	helperHUDEnabled        bool
	helperHUDPosition       string

	width  int
	height int
}

type (
	mode           int
	dashboardFocus int
	searchFocus    int
)

const (
	modeDashboard mode = iota
	modeSearch
	modePlaybackConfig
)

const (
	focusMain dashboardFocus = iota
	focusQueue
)

const (
	searchFocusResults searchFocus = iota
	searchFocusCarousel
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
	message        string
	err            error
	playlistPlayed bool
	playedPlaylist SearchResult
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

type playlistCarouselMsg struct {
	generation int
	store      playlistCarouselStore
	pinned     []SearchResult
	recent     []SearchResult
	err        error
}

type playlistThumbMsg struct {
	url  string
	view string
	err  error
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
	carouselStore, err := LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		carouselStore = defaultPlaylistCarouselStore()
	}
	lastRoom, err := LoadLastRoomSelection(cfg.LastRoomConfigPath)
	if err != nil {
		slog.Debug("tui: load last room selection failed", "err", err)
	}
	return Model{
		backend:           backend,
		config:            cfg,
		helper:            macoshelper.New(cfg.MacHelperPath),
		mode:              modeDashboard,
		loading:           true,
		searchCategory:    cfg.SearchCategory,
		themeName:         themeName,
		compactLayout:     cfg.Compact,
		helperHUDEnabled:  cfg.HelperHUDEnabled,
		helperHUDPosition: normalizeHelperHUDPosition(cfg.HelperHUDPosition),
		lastRoom:          lastRoom,
		carouselStore:     carouselStore,
		carouselPinned:    playlistCarouselPinnedResults(carouselStore),
		carouselRecent:    playlistCarouselRecentResults(carouselStore),
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
		previousRoom := m.selectedRoom()
		m.rooms = msg.rooms
		if idx, ok := roomIndexForSelection(m.rooms, currentLastRoomSelection(previousRoom)); ok {
			m.roomIndex = idx
		} else if idx, ok := roomIndexForSelection(m.rooms, m.lastRoom); ok {
			m.roomIndex = idx
		} else if m.roomIndex >= len(m.rooms) {
			m.roomIndex = max(0, len(m.rooms)-1)
		}
		if len(m.rooms) == 0 {
			m.message = "No Sonos rooms found"
			return m, nil
		}
		m.rememberSelectedRoom()
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
			if msg.playlistPlayed {
				m.recordRecentPlaylist(msg.playedPlaylist)
			}
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
			m.searchOffset = 0
			m.searchFocus = searchFocusResults
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
	case playlistCarouselMsg:
		if msg.generation != m.carouselGeneration {
			return m, nil
		}
		var cmds []tea.Cmd
		m.carouselLoading = false
		m.carouselStore = normalizePlaylistCarouselStore(msg.store)
		m.carouselPinned = msg.pinned
		m.carouselRecent = msg.recent
		m.clampCarouselSelection()
		cmds = append(cmds, m.playlistThumbCmds()...)
		if err := SavePlaylistCarouselStore(m.config.CarouselPath, m.carouselStore); err != nil {
			m.message = "playlist carousel save failed: " + err.Error()
		} else if msg.err != nil {
			m.message = "playlist carousel unavailable: " + msg.err.Error()
		}
		if m.mode == modeSearch && m.playlistCarouselVisible() && strings.TrimSpace(m.searchQuery) == "" {
			m.searchFocus = searchFocusCarousel
			updated, cmd := m.previewSelectedCarouselResult()
			m = updated.(Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	case playlistThumbMsg:
		if msg.err != nil || strings.TrimSpace(msg.url) == "" || msg.view == "" {
			return m, nil
		}
		if m.carouselThumbViews == nil {
			m.carouselThumbViews = map[string]string{}
		}
		m.carouselThumbViews[msg.url] = msg.view
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
		m.publishHelperSettings()
		m.publishNowPlaying()
		return m, macHelperWaitCmd(m.helper)
	case macHelperCommandMsg:
		switch strings.TrimSpace(msg.command) {
		case "volumeUp":
			m.loading = true
			cmd := tea.Batch(volumeCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Volume+5), spinnerCmd())
			return m, tea.Batch(cmd, macHelperWaitCmd(m.helper))
		case "volumeDown":
			m.loading = true
			cmd := tea.Batch(volumeCmd(m.backend, m.config.Timeout, m.selectedRoom(), m.status.Volume-5), spinnerCmd())
			return m, tea.Batch(cmd, macHelperWaitCmd(m.helper))
		}
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
		return m.openSearch()
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
			m.rememberSelectedRoom()
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
			m.rememberSelectedRoom()
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
		if m.playbackConfigIndex < 4 {
			m.playbackConfigIndex++
		}
		return m, nil
	case " ", "enter":
		if m.playbackConfigIndex == 3 {
			m.helperHUDEnabled = !m.helperHUDEnabled
			if err := SaveHelperHUDConfig(m.config.HelperHUDConfigPath, m.currentHelperHUDConfig()); err != nil {
				m.message = "helper HUD save failed: " + err.Error()
			} else {
				m.message = "media HUD " + onOff(m.helperHUDEnabled)
			}
			m.publishHelperSettings()
			m.err = nil
			return m, nil
		}
		if m.playbackConfigIndex == 4 {
			m.helperHUDPosition = nextHelperHUDPosition(m.helperHUDPosition)
			if err := SaveHelperHUDConfig(m.config.HelperHUDConfigPath, m.currentHelperHUDConfig()); err != nil {
				m.message = "helper HUD save failed: " + err.Error()
			} else {
				m.message = "media HUD position " + strings.ToLower(helperHUDPositionLabel(m.helperHUDPosition))
			}
			m.publishHelperSettings()
			m.err = nil
			return m, nil
		}
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
		if m.playlistCarouselVisible() && m.searchFocus == searchFocusResults && m.searchIndex == 0 && m.hasCarouselItems() {
			m.searchFocus = searchFocusCarousel
			updated, cmd := m.previewSelectedCarouselResult()
			return updated, cmd
		}
		if m.searchFocus == searchFocusCarousel {
			return m, nil
		}
		if m.searchIndex > 0 {
			m.searchIndex--
			m.ensureSearchSelectionVisible()
		}
		updated, cmd := m.previewSelectedSearchResult()
		return updated, cmd
	case "down", "j":
		if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
			if len(m.searchItems) > 0 {
				m.searchFocus = searchFocusResults
				updated, cmd := m.previewSelectedSearchResult()
				return updated, cmd
			}
			return m, nil
		}
		if m.searchIndex < len(m.searchItems)-1 {
			m.searchIndex++
			m.ensureSearchSelectionVisible()
		}
		updated, cmd := m.previewSelectedSearchResult()
		return updated, cmd
	case "left", "right":
		if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
			m.moveCarouselSelection(msg.String())
			updated, cmd := m.previewSelectedCarouselResult()
			return updated, cmd
		}
		return m, nil
	case "backspace", "ctrl+h":
		m.searchQuery = trimLastRune(m.searchQuery)
		m.searchIndex = 0
		m.searchOffset = 0
		m.searchGeneration++
		m.searchFocus = searchFocusResults
		m.resetPlaylistPreview()
		if strings.TrimSpace(m.searchQuery) == "" {
			m.searchPreviewQuery = ""
			m.searchItems = nil
			if m.playlistCarouselVisible() {
				m.searchFocus = searchFocusCarousel
				updated, cmd := m.previewSelectedCarouselResult()
				return updated, cmd
			}
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
	case "ctrl+u":
		m.searchQuery = ""
		m.searchIndex = 0
		m.searchOffset = 0
		m.searchGeneration++
		m.loading = false
		m.searchPreviewQuery = ""
		m.searchItems = nil
		m.resetPlaylistPreview()
		if m.playlistCarouselVisible() {
			m.searchFocus = searchFocusCarousel
			updated, cmd := m.previewSelectedCarouselResult()
			return updated, cmd
		}
		return m, nil
	case "tab", "esc":
		m.mode = modeDashboard
		m.err = nil
		return m, nil
	case "ctrl+t":
		return m.setSearchCategory("tracks")
	case "ctrl+p":
		return m.setSearchCategory("playlists")
	case "ctrl+f":
		if toggled, ok := m.toggleSelectedPlaylistPin(); ok {
			if toggled {
				return m.finishPlaylistPinToggle()
			}
			return m, nil
		}
		return m, nil
	case "enter":
		if len(m.rooms) == 0 {
			return m, nil
		}
		if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
			result, ok := m.selectedCarouselResult()
			if !ok {
				return m, nil
			}
			if strings.TrimSpace(result.Item.ID) == "" {
				m.message = "playlist not resolved yet"
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(playSearchCmd(m.backend, m.config, m.selectedRoom(), result), spinnerCmd())
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
	case " ":
		if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
			if toggled, ok := m.toggleSelectedPlaylistPin(); ok {
				if toggled {
					return m.finishPlaylistPinToggle()
				}
				return m, nil
			}
		}
		m.searchQuery += msg.String()
		m.searchIndex = 0
		m.searchOffset = 0
		m.searchFocus = searchFocusResults
		m.searchGeneration++
		m.loading = true
		m.resetPlaylistPreview()
		return m, tea.Batch(searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration), spinnerCmd())
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.searchQuery += msg.String()
			m.searchIndex = 0
			m.searchOffset = 0
			m.searchFocus = searchFocusResults
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
	m.searchOffset = 0
	m.searchFocus = searchFocusResults
	m.searchPreviewQuery = ""
	m.searchItems = nil
	m.resetPlaylistPreview()
	m.message = "searching " + category
	m.err = nil
	m.searchGeneration++
	var cmds []tea.Cmd
	if strings.TrimSpace(m.searchQuery) == "" || len(m.rooms) == 0 {
		if m.playlistCarouselVisible() {
			m.searchFocus = searchFocusCarousel
			if len(m.rooms) > 0 {
				m.startPlaylistCarouselLoad()
				cmds = append(cmds, playlistCarouselCmd(m.backend, m.config, m.selectedRoom(), m.carouselStore, m.carouselGeneration))
			}
			updated, cmd := m.previewSelectedCarouselResult()
			m = updated.(Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
	m.loading = true
	cmds = append(cmds, searchCmd(m.backend, m.config, m.selectedRoom(), m.searchCategory, m.searchQuery, m.searchGeneration))
	if m.playlistCarouselVisible() {
		m.startPlaylistCarouselLoad()
		cmds = append(cmds, playlistCarouselCmd(m.backend, m.config, m.selectedRoom(), m.carouselStore, m.carouselGeneration))
	}
	cmds = append(cmds, spinnerCmd())
	return m, tea.Batch(cmds...)
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
	if !isPlaylistResult(selected) {
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

func (m Model) previewSelectedCarouselResult() (tea.Model, tea.Cmd) {
	m.searchPreviewGeneration++
	m.searchPreviewItemID = ""
	m.searchPreviewLoading = false
	m.searchPreviewItems = nil
	if len(m.rooms) == 0 {
		return m, nil
	}
	selected, ok := m.selectedCarouselResult()
	if !ok {
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

func (m Model) previewSelectedCarouselOrSearchResult() (tea.Model, tea.Cmd) {
	if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
		return m.previewSelectedCarouselResult()
	}
	return m.previewSelectedSearchResult()
}

func (m Model) finishPlaylistPinToggle() (tea.Model, tea.Cmd) {
	updated, previewCmd := m.previewSelectedCarouselOrSearchResult()
	m = updated.(Model)
	cmds := m.playlistThumbCmds()
	if previewCmd != nil {
		cmds = append(cmds, previewCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) resetPlaylistPreview() {
	m.searchPreviewGeneration++
	m.searchPreviewItemID = ""
	m.searchPreviewLoading = false
	m.searchPreviewItems = nil
}

func (m Model) openSearch() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{tea.ClearScreen}
	if m.playlistCarouselVisible() && len(m.rooms) > 0 {
		m.startPlaylistCarouselLoad()
		cmds = append(cmds, playlistCarouselCmd(m.backend, m.config, m.selectedRoom(), m.carouselStore, m.carouselGeneration))
		if strings.TrimSpace(m.searchQuery) == "" {
			m.searchFocus = searchFocusCarousel
			updated, cmd := m.previewSelectedCarouselResult()
			m = updated.(Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) startPlaylistCarouselLoad() {
	m.carouselLoading = true
	m.carouselGeneration++
}

func (m Model) playlistCarouselVisible() bool {
	return strings.EqualFold(strings.TrimSpace(m.searchCategory), "playlists")
}

type playlistCarouselTab struct {
	ID    string
	Label string
	Items []SearchResult
}

func (m Model) playlistCarouselTabs() []playlistCarouselTab {
	if !m.playlistCarouselVisible() {
		return nil
	}
	tabs := []playlistCarouselTab{}
	if len(m.carouselPinned) > 0 {
		tabs = append(tabs, playlistCarouselTab{ID: "pinned", Label: "Pinned", Items: m.carouselPinned})
	}
	if len(m.carouselRecent) > 0 {
		tabs = append(tabs, playlistCarouselTab{ID: "recent", Label: "Recent", Items: m.carouselRecent})
	}
	return tabs
}

func (m Model) hasCarouselItems() bool {
	return len(m.playlistCarouselTabs()) > 0
}

func (m *Model) clampCarouselSelection() {
	tabs := m.playlistCarouselTabs()
	if len(tabs) == 0 {
		m.carouselTab = 0
		m.carouselIndex = 0
		return
	}
	if m.carouselTab < 0 {
		m.carouselTab = 0
	}
	if m.carouselTab >= len(tabs) {
		m.carouselTab = len(tabs) - 1
	}
	if m.carouselIndex < 0 {
		m.carouselIndex = 0
	}
	if m.carouselIndex >= len(tabs[m.carouselTab].Items) {
		m.carouselIndex = len(tabs[m.carouselTab].Items) - 1
	}
}

func (m Model) selectedCarouselResult() (SearchResult, bool) {
	tabs := m.playlistCarouselTabs()
	if len(tabs) == 0 {
		return SearchResult{}, false
	}
	tabIndex := clamp(m.carouselTab, 0, len(tabs)-1)
	items := tabs[tabIndex].Items
	if len(items) == 0 {
		return SearchResult{}, false
	}
	itemIndex := clamp(m.carouselIndex, 0, len(items)-1)
	return items[itemIndex], true
}

func (m *Model) moveCarouselSelection(direction string) {
	tabs := m.playlistCarouselTabs()
	if len(tabs) == 0 {
		m.carouselTab = 0
		m.carouselIndex = 0
		return
	}
	m.clampCarouselSelection()
	switch direction {
	case "left":
		if m.carouselIndex > 0 {
			m.carouselIndex--
			return
		}
		if m.carouselTab > 0 {
			m.carouselTab--
			m.carouselIndex = len(tabs[m.carouselTab].Items) - 1
		}
	case "right":
		if m.carouselIndex < len(tabs[m.carouselTab].Items)-1 {
			m.carouselIndex++
			return
		}
		if m.carouselTab < len(tabs)-1 {
			m.carouselTab++
			m.carouselIndex = 0
		}
	}
	m.clampCarouselSelection()
}

func (m *Model) toggleSelectedPlaylistPin() (bool, bool) {
	result, ok := m.selectedPinnablePlaylistResult()
	if !ok {
		return false, false
	}
	// Invalidate any in-flight carousel refresh so it can't overwrite local pin changes.
	m.carouselGeneration++
	m.carouselLoading = false
	item := playlistCarouselStoreItemFromResult(result)
	if item.Title == "" && item.ID == "" {
		return false, false
	}
	key := playlistCarouselStoreKey(item)
	pins := normalizePlaylistCarouselItems(m.carouselStore.Pins, 0)
	next := make([]playlistCarouselStoreItem, 0, len(pins)+1)
	removed := false
	for _, pin := range pins {
		if playlistCarouselStoreKey(pin) == key {
			removed = true
			continue
		}
		next = append(next, pin)
	}
	if removed {
		m.message = "unpinned " + result.Title()
	} else {
		next = append(next, item)
		m.message = "pinned " + result.Title()
	}
	m.carouselStore.Pins = next
	m.carouselStore = normalizePlaylistCarouselStore(m.carouselStore)
	m.carouselPinned = playlistCarouselPinnedResults(m.carouselStore)
	if !removed && len(m.carouselPinned) > 0 {
		m.carouselTab = 0
		m.carouselIndex = 0
		for i, pinned := range m.carouselPinned {
			if sameSearchResultID(pinned, result) || strings.EqualFold(strings.TrimSpace(pinned.Title()), strings.TrimSpace(result.Title())) {
				m.carouselIndex = i
				break
			}
		}
	}
	m.clampCarouselSelection()
	if m.carouselThumbViews == nil {
		m.carouselThumbViews = map[string]string{}
	}
	if err := SavePlaylistCarouselStore(m.config.CarouselPath, m.carouselStore); err != nil {
		m.message = "playlist carousel save failed: " + err.Error()
	}
	return true, true
}

func (m *Model) recordRecentPlaylist(result SearchResult) {
	if !isRecentEligibleCarouselResult(result) {
		return
	}
	// Invalidate any in-flight carousel refresh so it can't overwrite local recent updates.
	m.carouselGeneration++
	m.carouselStore = addRecentPlaylistToStore(m.carouselStore, result)
	m.carouselRecent = playlistCarouselRecentResults(m.carouselStore)
	m.clampCarouselSelection()
	if err := SavePlaylistCarouselStore(m.config.CarouselPath, m.carouselStore); err != nil {
		m.message = "playlist carousel save failed: " + err.Error()
	}
}

func (m Model) playlistThumbCmds() []tea.Cmd {
	var cmds []tea.Cmd
	seen := map[string]bool{}
	for _, tab := range m.playlistCarouselTabs() {
		for _, item := range tab.Items {
			url := playlistArtworkURL(item)
			if !fetchableArtworkURL(url) || seen[url] {
				continue
			}
			seen[url] = true
			if m.carouselThumbViews != nil {
				if _, ok := m.carouselThumbViews[url]; ok {
					continue
				}
			}
			cmds = append(cmds, fetchPlaylistThumbCmd(url))
		}
	}
	return cmds
}

func (m Model) selectedPinnablePlaylistResult() (SearchResult, bool) {
	if m.playlistCarouselVisible() && m.searchFocus == searchFocusCarousel {
		result, ok := m.selectedCarouselResult()
		if !ok || !isPinnableCarouselResult(result) {
			return SearchResult{}, false
		}
		return result, true
	}
	if m.searchFocus != searchFocusResults || len(m.searchItems) == 0 || m.searchIndex < 0 || m.searchIndex >= len(m.searchItems) {
		return SearchResult{}, false
	}
	result := m.searchItems[m.searchIndex]
	if !isPinnableCarouselResult(result) {
		return SearchResult{}, false
	}
	return result, true
}

func (m *Model) ensureSearchSelectionVisible() {
	if len(m.searchItems) == 0 {
		m.searchOffset = 0
		return
	}
	visible := m.searchVisibleRows()
	if m.searchIndex < m.searchOffset {
		m.searchOffset = m.searchIndex
	}
	if m.searchIndex >= m.searchOffset+visible {
		m.searchOffset = m.searchIndex - visible + 1
	}
	maxOffset := max(0, len(m.searchItems)-visible)
	if m.searchOffset > maxOffset {
		m.searchOffset = maxOffset
	}
	if m.searchOffset < 0 {
		m.searchOffset = 0
	}
}

func (m Model) searchVisibleRows() int {
	return 8
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

func (m *Model) rememberSelectedRoom() {
	selection := currentLastRoomSelection(m.selectedRoom())
	if selection == (lastRoomSelection{Version: 1}) {
		return
	}
	m.lastRoom = selection
	if err := SaveLastRoomSelection(m.config.LastRoomConfigPath, selection); err != nil {
		slog.Debug("tui: save last room selection failed", "err", err)
	}
}

func currentLastRoomSelection(room Room) lastRoomSelection {
	return normalizeLastRoomSelection(lastRoomSelection{
		IP:   room.IP,
		Name: room.Name,
	})
}

func roomIndexForSelection(rooms []Room, selection lastRoomSelection) (int, bool) {
	for i, room := range rooms {
		if selection.IP != "" && strings.TrimSpace(room.IP) == selection.IP {
			return i, true
		}
	}
	for i, room := range rooms {
		if selection.Name != "" && strings.TrimSpace(room.Name) == selection.Name {
			return i, true
		}
	}
	return 0, false
}

func (m Model) handleMacHelperCommand(command string) (tea.Model, tea.Cmd) {
	if len(m.rooms) == 0 {
		m.message = "No Sonos room selected"
		return m, nil
	}
	var action string
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

func (m Model) publishHelperSettings() {
	if m.helper == nil {
		return
	}
	enabled := m.helperHUDEnabled
	position := normalizeHelperHUDPosition(m.helperHUDPosition)
	m.helper.Publish(macoshelper.Message{
		Type:        "settings",
		HUDEnabled:  &enabled,
		HUDPosition: &position,
	})
}

func (m Model) currentHelperHUDConfig() HelperHUDConfig {
	return HelperHUDConfig{
		Enabled:  m.helperHUDEnabled,
		Position: normalizeHelperHUDPosition(m.helperHUDPosition),
	}
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
		target := clamp(volume, 0, 100)
		err := backend.SetVolume(ctx, room, target)
		return actionMsg{message: fmt.Sprintf("volume %d", target), err: err}
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
		return actionMsg{
			message:        "playing " + result.Title(),
			err:            err,
			playlistPlayed: isRecentEligibleCarouselResult(result),
			playedPlaylist: result,
		}
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

func playlistCarouselCmd(backend Backend, cfg Config, room Room, store playlistCarouselStore, generation int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		store = normalizePlaylistCarouselStore(store)
		shouldSeedDefaults := !store.DefaultPinsSeeded && !hasCustomizedPlaylistPins(store)
		slog.Debug("playlist carousel: load start",
			"generation", generation,
			"room", room.Name,
			"service", cfg.SearchService,
			"pins", len(store.Pins),
			"recent", len(store.Recent),
			"defaultPinsSeeded", store.DefaultPinsSeeded,
			"shouldSeedDefaults", shouldSeedDefaults,
		)
		pinned := playlistCarouselPinnedResults(store)
		nextPins := make([]playlistCarouselStoreItem, 0, len(store.Pins))
		var firstErr error
		for i, pin := range store.Pins {
			title := strings.TrimSpace(pin.Title)
			if title == "" {
				continue
			}
			current := playlistCarouselResultFromStoreItem(pin)
			needsResolve := strings.TrimSpace(pin.ID) == "" || playlistArtworkURL(current) == "" || isReleaseRadarTitle(title)
			slog.Debug("playlist carousel: inspect pin",
				"title", title,
				"id", pin.ID,
				"creator", pin.Creator,
				"needsResolve", needsResolve,
			)
			if !needsResolve {
				nextPins = append(nextPins, pin)
				continue
			}

			result, ok, err := backend.ResolvePinnedPlaylist(ctx, room, cfg.SearchService, title)
			if err == nil && ok && strings.TrimSpace(pin.ID) != "" && !sameSearchResultID(current, result) && !isReleaseRadarTitle(title) {
				ok = false
			}
			if err == nil && ok && isReleaseRadarTitle(title) && !isSpotifyReleaseRadarResult(result) {
				ok = false
			}
			if err == nil && ok {
				slog.Debug("playlist carousel: resolved pin",
					"title", title,
					"resolvedTitle", result.Title(),
					"resolvedID", result.Item.ID,
					"resolvedCreator", result.Item.Creator,
				)
				pin = playlistCarouselStoreItemFromResult(result)
				pinned[i] = result
				nextPins = append(nextPins, pin)
				continue
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if err != nil {
				slog.Debug("playlist carousel: pin resolve failed",
					"title", title,
					"id", pin.ID,
					"err", err.Error(),
				)
				if isReleaseRadarTitle(title) && strings.TrimSpace(pin.ID) != "" && !isSpotifyReleaseRadarResult(current) {
					continue
				}
				nextPins = append(nextPins, pin)
				continue
			}
			if isReleaseRadarTitle(title) {
				slog.Debug("playlist carousel: dropping unresolved release radar")
				continue
			}
			slog.Debug("playlist carousel: keeping unresolved pin",
				"title", title,
				"id", pin.ID,
				"creator", pin.Creator,
			)
			if strings.TrimSpace(pin.ID) != "" {
				if strings.TrimSpace(pin.ArtworkURI) == "" && strings.TrimSpace(pin.AlbumArtURI) != "" {
					pin.ArtworkURI = pin.AlbumArtURI
				}
			}
			nextPins = append(nextPins, pin)
		}
		store.Pins = nextPins
		pinned = playlistCarouselPinnedResults(store)

		defaultPins, seedErr := discoverDefaultPinnedPlaylists(ctx, backend, cfg, room)
		if seedErr != nil && firstErr == nil {
			firstErr = seedErr
		}
		if shouldSeedDefaults && seedErr == nil {
			store = seedDefaultPlaylistPins(store, defaultPins)
			store.DefaultPinsSeeded = true
			pinned = playlistCarouselPinnedResults(store)
			slog.Debug("playlist carousel: seeded default pins",
				"pinnedCount", len(pinned),
				"popularCount", len(defaultPins.PopularPlaylists),
				"likedSongsFound", isSpotifyLikedSongsResult(defaultPins.LikedSongs),
			)
		} else if !store.DefaultPinsSeeded && seedErr == nil {
			store.DefaultPinsSeeded = true
			slog.Debug("playlist carousel: marked defaults seeded without changes")
		}
		slog.Debug("playlist carousel: load complete",
			"generation", generation,
			"pinnedCount", len(pinned),
			"recentCount", len(store.Recent),
			"err", debugErrString(firstErr),
		)

		return playlistCarouselMsg{
			generation: generation,
			store:      store,
			pinned:     pinned,
			recent:     playlistCarouselRecentResults(store),
			err:        firstErr,
		}
	}
}

type defaultPinnedPlaylists struct {
	PopularPlaylists []SearchResult
	LikedSongs       SearchResult
}

func discoverDefaultPinnedPlaylists(ctx context.Context, backend Backend, cfg Config, room Room) (defaultPinnedPlaylists, error) {
	root, err := backend.PlaylistShelf(ctx, room, cfg.SearchService, "root", 24)
	if err != nil {
		slog.Debug("playlist carousel: default pin root shelf fetch failed",
			"room", room.Name,
			"service", cfg.SearchService,
			"err", err.Error(),
		)
		return defaultPinnedPlaylists{}, err
	}
	var out defaultPinnedPlaylists
	var firstErr error
	for _, shelf := range root {
		kind := defaultPinShelfKind(shelf.Title())
		slog.Debug("playlist carousel: inspect default pin shelf",
			"title", shelf.Title(),
			"id", shelf.Item.ID,
			"itemType", shelf.Item.ItemType,
			"kind", kind,
		)
		if kind == "" {
			continue
		}
		items, err := backend.PlaylistShelf(ctx, room, cfg.SearchService, strings.TrimSpace(shelf.Item.ID), playlistShelfLimit)
		if err != nil {
			slog.Debug("playlist carousel: default pin child shelf fetch failed",
				"title", shelf.Title(),
				"id", shelf.Item.ID,
				"kind", kind,
				"err", err.Error(),
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rawItems := items
		items = filterDefaultPinShelfItems(kind, rawItems, playlistShelfLimit)
		slog.Debug("playlist carousel: default pin child shelf items",
			"title", shelf.Title(),
			"id", shelf.Item.ID,
			"kind", kind,
			"count", len(items),
			"titles", searchResultTitles(items),
			"rawCount", len(rawItems),
			"rawTitles", searchResultTitles(rawItems),
		)
		switch kind {
		case "popular-playlists":
			out.PopularPlaylists = items
		case "your-music":
			if len(items) > 0 {
				out.LikedSongs = items[0]
			}
		}
	}
	return out, firstErr
}

func defaultPinShelfKind(title string) string {
	normalized := strings.ToLower(strings.TrimSpace(title))
	switch normalized {
	case "popular playlists":
		return "popular-playlists"
	case "your music":
		return "your-music"
	default:
		return ""
	}
}

func seedDefaultPlaylistPins(store playlistCarouselStore, shelves defaultPinnedPlaylists) playlistCarouselStore {
	addDefaultPin := func(result SearchResult) {
		item := playlistCarouselStoreItemFromResult(result)
		if item.Title == "" && item.ID == "" {
			return
		}
		store.Pins = append(store.Pins, item)
		store = normalizePlaylistCarouselStore(store)
		slog.Debug("playlist carousel: seeded pin",
			"title", result.Title(),
			"id", result.Item.ID,
			"creator", result.Item.Creator,
		)
	}
	if len(shelves.PopularPlaylists) > 0 {
		addDefaultPin(shelves.PopularPlaylists[0])
	} else {
		slog.Debug("playlist carousel: no popular playlists candidates found")
	}
	if isSpotifyLikedSongsResult(shelves.LikedSongs) {
		addDefaultPin(shelves.LikedSongs)
	} else {
		slog.Debug("playlist carousel: no liked songs candidate found")
	}
	return normalizePlaylistCarouselStore(store)
}

func hasCustomizedPlaylistPins(store playlistCarouselStore) bool {
	pins := normalizePlaylistCarouselItems(store.Pins, 0)
	for _, pin := range pins {
		title := strings.TrimSpace(pin.Title)
		if !isReleaseRadarTitle(title) {
			return true
		}
		if strings.TrimSpace(pin.ID) != "" {
			return true
		}
	}
	return false
}

func searchResultTitles(items []SearchResult) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Title())
	}
	return out
}

func filterPlaylistResults(items []SearchResult, limit int) []SearchResult {
	out := make([]SearchResult, 0, len(items))
	for _, item := range items {
		if !isPlaylistResult(item) {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func filterDefaultPinShelfItems(kind string, items []SearchResult, limit int) []SearchResult {
	switch kind {
	case "popular-playlists":
		return filterPlaylistResults(items, limit)
	case "your-music":
		for _, item := range items {
			if isSpotifyLikedSongsResult(item) {
				return []SearchResult{item}
			}
		}
	}
	return nil
}

func debugErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

//nolint:unused
func dedupePlaylistResults(items []SearchResult, limit int) []SearchResult {
	out := make([]SearchResult, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		key := searchResultKey(item)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

//nolint:unused
func searchResultKey(result SearchResult) string {
	if id := strings.ToLower(strings.TrimSpace(result.Item.ID)); id != "" {
		return "id:" + id
	}
	if title := strings.ToLower(strings.TrimSpace(result.Title())); title != "" {
		return "title:" + title
	}
	return ""
}

func isPlaylistResult(result SearchResult) bool {
	itemType := strings.ToLower(strings.TrimSpace(result.Item.ItemType))
	return itemType == "playlist" || strings.Contains(itemType, "playlist")
}

func isSpotifyLikedSongsResult(result SearchResult) bool {
	if strings.EqualFold(strings.TrimSpace(result.Item.ID), "your_songs") {
		return true
	}
	title := strings.ToLower(strings.TrimSpace(result.Item.Title))
	itemType := strings.ToLower(strings.TrimSpace(result.Item.ItemType))
	return title == "songs" && itemType == "tracklist"
}

func isPinnableCarouselResult(result SearchResult) bool {
	return isPlaylistResult(result) || isSpotifyLikedSongsResult(result)
}

func isBrowsableCarouselResult(result SearchResult) bool {
	return isPlaylistResult(result) || isSpotifyLikedSongsResult(result)
}

func isRecentEligibleCarouselResult(result SearchResult) bool {
	return isPlaylistResult(result) || isSpotifyLikedSongsResult(result)
}

func sameSearchResultID(a, b SearchResult) bool {
	aID := strings.TrimSpace(a.Item.ID)
	bID := strings.TrimSpace(b.Item.ID)
	return aID != "" && bID != "" && strings.EqualFold(aID, bID)
}

func isReleaseRadarTitle(title string) bool {
	return strings.EqualFold(strings.TrimSpace(title), "Release Radar")
}

//nolint:unused
func isLikedSongsTitle(title string) bool {
	return strings.EqualFold(strings.TrimSpace(title), "Liked Songs")
}

func isSpotifyReleaseRadarResult(result SearchResult) bool {
	if !isPlaylistResult(result) || !isReleaseRadarTitle(result.Title()) {
		return false
	}
	creator := strings.ToLower(strings.TrimSpace(result.Item.Creator))
	if creator == "spotify" {
		return true
	}
	id := strings.ToLower(strings.TrimSpace(result.Item.ID))
	return strings.Contains(id, "spotify:user:spotify:playlist:") ||
		strings.HasPrefix(id, "spotify:playlist:37i9dq")
}

func playlistArtworkURL(result SearchResult) string {
	if url := strings.TrimSpace(result.Item.ArtworkURI); url != "" {
		return url
	}
	return strings.TrimSpace(result.Item.AlbumArtURI)
}

func fetchableArtworkURL(url string) bool {
	url = strings.TrimSpace(strings.ToLower(url))
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
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
