package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shlomiuziel/sonosh/internal/macoshelper"
	"github.com/shlomiuziel/sonosh/internal/sonos"
)

func TestModelLoadsRoomsAndStatus(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		status: Status{
			State:  "PLAYING",
			Title:  "A Track",
			Volume: 25,
		},
	}
	model := NewModel(backend, testConfig())

	updated, cmd := model.Update(roomsMsg{rooms: backend.rooms})
	model = updated.(Model)
	if len(model.rooms) != 1 {
		t.Fatalf("rooms = %d, want 1", len(model.rooms))
	}
	if cmd == nil {
		t.Fatal("expected status command")
	}

	updated, _ = model.Update(runCmd(cmd).(statusMsg))
	model = updated.(Model)
	if model.status.Title != "A Track" {
		t.Fatalf("status title = %q, want A Track", model.status.Title)
	}
}

func TestRoomsMessageKeepsLoadingUntilStatusArrives(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
	}
	model := NewModel(backend, testConfig())

	updated, cmd := model.Update(roomsMsg{rooms: backend.rooms})
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected status command")
	}
	if !model.loading {
		t.Fatal("expected loading to remain true while status loads")
	}

	updated, _ = model.Update(statusMsg{status: Status{Title: "Loaded"}})
	model = updated.(Model)
	if model.loading {
		t.Fatal("expected loading to clear after status arrives")
	}
}

func TestSpinnerAdvancesOnlyWhileLoading(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.loading = true

	updated, cmd := model.Update(spinnerMsg(time.Now()))
	model = updated.(Model)
	if model.spinnerFrame != 1 {
		t.Fatalf("spinner frame = %d, want 1", model.spinnerFrame)
	}
	if cmd == nil {
		t.Fatal("expected next spinner command while loading")
	}

	model.loading = false
	updated, cmd = model.Update(spinnerMsg(time.Now()))
	model = updated.(Model)
	if model.spinnerFrame != 1 {
		t.Fatalf("spinner frame advanced while not loading: %d", model.spinnerFrame)
	}
	if cmd != nil {
		t.Fatal("did not expect spinner command while not loading")
	}
}

func TestDashboardKeysDispatchActions(t *testing.T) {
	backend := &fakeBackend{
		rooms:  []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.11"}},
		status: Status{State: "PLAYING", Volume: 25},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.status = backend.status

	updated, cmd := model.Update(key(" "))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected transport command")
	}
	_ = runCmd(cmd)
	if backend.transportAction != "pause" {
		t.Fatalf("transport action = %q, want pause", backend.transportAction)
	}

	updated, cmd = model.Update(key("+"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected volume command")
	}
	_ = runCmd(cmd)
	if backend.volume != 30 {
		t.Fatalf("volume = %d, want 30", backend.volume)
	}
}

func TestThemeShortcutCyclesThemes(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
	}
	dir := t.TempDir()
	cfg := testConfig()
	cfg.ThemeConfigPath = filepath.Join(dir, "theme.json")
	model := NewModel(backend, cfg)
	start := model.themeName

	updated, cmd := model.Update(key("ctrl+v"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect a command when cycling theme")
	}
	if model.themeName == start {
		t.Fatalf("theme did not change from %q", start)
	}
	if model.message != "theme: "+model.themeName {
		t.Fatalf("theme message = %q, want theme announcement", model.message)
	}
	if _, err := os.Stat(cfg.ThemeConfigPath); err != nil {
		t.Fatalf("theme config not written: %v", err)
	}
	stored, err := LoadThemeName(cfg.ThemeConfigPath)
	if err != nil {
		t.Fatalf("LoadThemeName: %v", err)
	}
	if stored != model.themeName {
		t.Fatalf("stored theme = %q, want %q", stored, model.themeName)
	}

	for i := 1; i < len(visualThemes); i++ {
		updated, _ = model.Update(key("ctrl+v"))
		model = updated.(Model)
	}
	if model.themeName != start {
		t.Fatalf("theme did not wrap back to %q, got %q", start, model.themeName)
	}
}

func TestSearchKeysSearchAndPlay(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		searchFn: func(query string) []SearchResult {
			return []SearchResult{{
				Item: sonos.SMAPIItem{ID: query, ItemType: "track", Title: "Result One"},
			}}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch

	for _, r := range "daft" {
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(Model)
		if cmd == nil {
			t.Fatal("expected live search command")
		}
		updated, _ = model.Update(runCmd(cmd))
		model = updated.(Model)
	}
	if model.searchPreviewQuery != "daft" {
		t.Fatalf("preview query = %q, want daft", model.searchPreviewQuery)
	}

	updated, cmd := model.Update(key("enter"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected search command")
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if len(model.searchItems) != 1 {
		t.Fatalf("search items = %d, want 1", len(model.searchItems))
	}

	updated, cmd = model.Update(key("enter"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected play command")
	}
	_ = runCmd(cmd)
	if backend.played.ID != "daft" {
		t.Fatalf("played ID = %q, want daft", backend.played.ID)
	}
}

func TestSearchModeAllowsRAndQTyping(t *testing.T) {
	backend := &fakeBackend{
		rooms:  []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		status: Status{State: "PLAYING"},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch

	updated, cmd := model.Update(key("r"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected live search command for r")
	}
	if model.searchQuery != "r" {
		t.Fatalf("search query = %q, want r", model.searchQuery)
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.searchPreviewQuery != "r" {
		t.Fatalf("preview query = %q, want r", model.searchPreviewQuery)
	}

	updated, cmd = model.Update(key("q"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected live search command for q")
	}
	if model.searchQuery != "rq" {
		t.Fatalf("search query = %q, want rq", model.searchQuery)
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.searchPreviewQuery != "rq" {
		t.Fatalf("preview query = %q, want rq", model.searchPreviewQuery)
	}
}

func TestSearchModeAllowsSpaceTyping(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		searchFn: func(query string) []SearchResult {
			return []SearchResult{{
				Item: sonos.SMAPIItem{ID: query, ItemType: "track", Title: query},
			}}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch

	updated, cmd := model.Update(key("a"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected search command")
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)

	updated, cmd = model.Update(key(" "))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected search command after space")
	}
	if model.searchQuery != "a " {
		t.Fatalf("search query = %q, want %q", model.searchQuery, "a ")
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.searchPreviewQuery != "a " {
		t.Fatalf("preview query = %q, want %q", model.searchPreviewQuery, "a ")
	}
}

func TestSearchIgnoresStaleResults(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		searchFn: func(query string) []SearchResult {
			return []SearchResult{{
				Item: sonos.SMAPIItem{ID: query, ItemType: "track", Title: query},
			}}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch

	updated, cmd := model.Update(key("a"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected search command")
	}
	stale := cmd

	updated, cmd = model.Update(key("b"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected second search command")
	}
	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.searchPreviewQuery != "ab" {
		t.Fatalf("preview query = %q, want ab", model.searchPreviewQuery)
	}

	updated, _ = model.Update(runCmd(stale))
	model = updated.(Model)
	if model.searchPreviewQuery != "ab" {
		t.Fatalf("stale result changed preview query to %q", model.searchPreviewQuery)
	}
}

func TestSearchModeSwitchesToPlaylists(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		searchFn: func(query string) []SearchResult {
			return []SearchResult{{
				Item: sonos.SMAPIItem{ID: "spotify:playlist:" + query, ItemType: "playlist", Title: "Playlist " + query},
			}}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch
	model.searchQuery = "party"
	model.searchPreviewQuery = "party"
	model.searchItems = []SearchResult{{Item: sonos.SMAPIItem{ID: "spotify:track:old", ItemType: "track", Title: "Old"}}}

	updated, cmd := model.Update(key("ctrl+p"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected playlist search command")
	}
	if model.searchCategory != "playlists" {
		t.Fatalf("search category = %q, want playlists", model.searchCategory)
	}
	if len(model.searchItems) != 0 {
		t.Fatalf("search items were not cleared on category switch")
	}

	updated, _ = model.Update(runCmd(cmd))
	model = updated.(Model)
	if got := backend.searchCategories[len(backend.searchCategories)-1]; got != "playlists" {
		t.Fatalf("search category sent to backend = %q, want playlists", got)
	}
	if len(model.searchItems) != 1 || model.searchItems[0].Item.ItemType != "playlist" {
		t.Fatalf("playlist results not applied: %#v", model.searchItems)
	}
}

func TestMacHelperCommandDispatchesTransport(t *testing.T) {
	backend := &fakeBackend{
		rooms:  []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		status: Status{State: "PLAYING"},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.status = backend.status

	updated, cmd := model.handleMacHelperCommand("togglePlayPause")
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected transport command")
	}
	_ = runCmd(cmd)
	if backend.transportAction != "pause" {
		t.Fatalf("transport action = %q, want pause", backend.transportAction)
	}

	model.status.State = "PAUSED_PLAYBACK"
	updated, cmd = model.handleMacHelperCommand("togglePlayPause")
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected transport command")
	}
	_ = runCmd(cmd)
	if backend.transportAction != "play" {
		t.Fatalf("transport action = %q, want play", backend.transportAction)
	}
}

func TestMacHelperUnavailableIsVisible(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())

	updated, _ := model.Update(macHelperStartedMsg{err: macoshelper.ErrUnavailable})
	model = updated.(Model)
	if model.message != "mac helper unavailable" {
		t.Fatalf("message = %q, want helper unavailable message", model.message)
	}
}

func TestNowPlayingMessage(t *testing.T) {
	msg := nowPlayingMessage(Room{Name: "Kitchen"}, Status{
		State:    "PLAYING",
		Title:    "Track",
		Artist:   "Artist",
		Album:    "Album",
		AlbumArt: "http://example.test/art.jpg",
		Position: "00:01:15",
		Duration: "00:03:20",
		Volume:   42,
		Muted:    true,
	})

	if msg.Type != "nowPlaying" || msg.Room != "Kitchen" || msg.State != "playing" {
		t.Fatalf("unexpected now playing identity fields: %#v", msg)
	}
	if msg.PositionSeconds == nil || *msg.PositionSeconds != 75 {
		t.Fatalf("position seconds = %#v, want 75", msg.PositionSeconds)
	}
	if msg.DurationSeconds == nil || *msg.DurationSeconds != 200 {
		t.Fatalf("duration seconds = %#v, want 200", msg.DurationSeconds)
	}
	if msg.Volume == nil || *msg.Volume != 42 {
		t.Fatalf("volume = %#v, want 42", msg.Volume)
	}
	if msg.Muted == nil || !*msg.Muted {
		t.Fatalf("muted = %#v, want true", msg.Muted)
	}
}

func TestViewRendersPlayerSurface(t *testing.T) {
	backend := &fakeBackend{}
	model := NewModel(backend, testConfig())
	model.width = 110
	model.loading = false
	model.rooms = []Room{{
		Name:          "Kitchen",
		IP:            "192.0.2.10",
		CoordinatorIP: "192.0.2.10",
		GroupMembers:  []string{"Kitchen", "Living Room"},
	}}
	model.status = Status{
		State:    "PLAYING",
		Title:    "Whenever Wherever",
		Artist:   "Shakira",
		Album:    "Laundry Service",
		AlbumArt: "http://192.0.2.10:1400/getaa?u=x",
		Position: "00:01:15",
		Duration: "00:03:20",
		Volume:   42,
	}
	model.artURL = model.status.AlbumArt
	model.artView = "▀▀▀▀\n▀▀▀▀"

	view := model.View()
	for _, want := range []string{
		"sonosh",
		"ROOMS",
		"Kitchen",
		"NOW PLAYING",
		"Whenever Wherever",
		"Shakira",
		"Laundry Service",
		"▀▀▀▀",
		"42%",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "╭") || !strings.Contains(view, "━") {
		t.Fatalf("view missing styled panel/progress glyphs:\n%s", view)
	}
}

func TestViewRendersLoadingSpinner(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 100
	model.loading = true
	model.spinnerFrame = 1

	view := model.View()
	if !strings.Contains(view, "⠙") {
		t.Fatalf("view missing spinner frame:\n%s", view)
	}
	if !strings.Contains(view, "loading") {
		t.Fatalf("view missing loading text:\n%s", view)
	}
}

func TestViewRendersSearchSurface(t *testing.T) {
	backend := &fakeBackend{}
	model := NewModel(backend, testConfig())
	model.width = 110
	model.loading = false
	model.mode = modeSearch
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}}
	model.status = Status{State: "PAUSED_PLAYBACK", Title: "Current Track", Artist: "Artist"}
	model.searchQuery = "mas que nada"
	model.searchPreviewQuery = "mas que nada"
	model.searchItems = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:track:1", ItemType: "track", Title: "Mas Que Nada"}},
		{Item: sonos.SMAPIItem{ID: "spotify:track:2", ItemType: "track", Title: "Mas Que Nada - Live"}},
	}

	view := model.View()
	for _, want := range []string{
		"SPOTIFY / TRACKS",
		"> mas que nada",
		"results for mas que nada",
		"Mas Que Nada",
		"theme aurora",
		"ctrl+v",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("search view missing %q:\n%s", want, view)
		}
	}
}

func TestFooterFitsNarrowWidth(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	footer := model.renderFooter(32)
	if strings.Contains(footer, "\n") {
		t.Fatalf("footer wrapped unexpectedly:\n%s", footer)
	}
	if got := lipgloss.Width(footer); got > 32 {
		t.Fatalf("footer width = %d, want <= 32:\n%s", got, footer)
	}
}

func testConfig() Config {
	return Config{Timeout: time.Second, SearchService: "Spotify", SearchCategory: "tracks", SearchLimit: 10}
}

func key(value string) tea.KeyMsg {
	switch value {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	case "ctrl+v":
		return tea.KeyMsg{Type: tea.KeyCtrlV}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		if len(batch) == 0 {
			return nil
		}
		return runCmd(batch[0])
	}
	return msg
}

type fakeBackend struct {
	rooms            []Room
	status           Status
	results          []SearchResult
	searchFn         func(query string) []SearchResult
	transportAction  string
	volume           int
	muted            bool
	played           sonos.SMAPIItem
	searchQueries    []string
	searchCategories []string
}

func (f *fakeBackend) Discover(context.Context) ([]Room, error) {
	return f.rooms, nil
}

func (f *fakeBackend) Status(context.Context, Room) (Status, error) {
	return f.status, nil
}

func (f *fakeBackend) Transport(_ context.Context, _ Room, action string) error {
	f.transportAction = action
	return nil
}

func (f *fakeBackend) SetVolume(_ context.Context, _ Room, volume int) error {
	f.volume = volume
	return nil
}

func (f *fakeBackend) ToggleMute(context.Context, Room) error {
	f.muted = !f.muted
	return nil
}

func (f *fakeBackend) Search(_ context.Context, _ Room, _, category, query string, _ int) ([]SearchResult, error) {
	f.searchQueries = append(f.searchQueries, query)
	f.searchCategories = append(f.searchCategories, category)
	if f.searchFn != nil {
		return f.searchFn(query), nil
	}
	return f.results, nil
}

func (f *fakeBackend) PlaySearchResult(_ context.Context, _ Room, _ string, result SearchResult) error {
	f.played = result.Item
	return nil
}
