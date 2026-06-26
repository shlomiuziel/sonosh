package tui

import (
	"context"
	"fmt"
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

func TestWindowResizeClearsScreen(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	model = updated.(Model)
	if model.width != 120 || model.height != 32 {
		t.Fatalf("window size = %dx%d, want 120x32", model.width, model.height)
	}
	if cmd == nil {
		t.Fatal("expected clear screen command after resize")
	}
	if got := fmt.Sprintf("%T", runCmd(cmd)); got != "tea.clearScreenMsg" {
		t.Fatalf("resize command = %s, want tea.clearScreenMsg", got)
	}
}

func TestDashboardKeysDispatchActions(t *testing.T) {
	backend := &fakeBackend{
		rooms:  []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.11"}},
		status: Status{State: "PLAYING", Volume: 25, Position: "0:01:10", Duration: "0:03:00"},
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

	updated, cmd = model.Update(key("right"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected seek command")
	}
	_ = runCmd(cmd)
	if backend.seekDelta != 5 {
		t.Fatalf("seek delta = %d, want 5", backend.seekDelta)
	}
	if backend.seekPosition != "0:01:10" || backend.seekDuration != "0:03:00" {
		t.Fatalf("seek inputs = %q / %q", backend.seekPosition, backend.seekDuration)
	}

	_, cmd = model.Update(key("left"))
	if cmd == nil {
		t.Fatal("expected seek command")
	}
	_ = runCmd(cmd)
	if backend.seekDelta != -5 {
		t.Fatalf("seek delta = %d, want -5", backend.seekDelta)
	}
}

func TestPlaybackConfigModalTogglesPlaybackSettings(t *testing.T) {
	backend := &fakeBackend{
		rooms:     []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		status:    Status{State: "PLAYING", CrossfadeKnown: true, CrossfadeEnabled: false, ShuffleKnown: true, ShuffleEnabled: false, RepeatKnown: true, RepeatMode: "off"},
		crossfade: false,
		shuffle:   false,
		repeat:    "off",
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.status = backend.status

	updated, cmd := model.Update(key("o"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected clear screen command when opening playback config")
	}
	if got := fmt.Sprintf("%T", runCmd(cmd)); got != "tea.clearScreenMsg" {
		t.Fatalf("playback config open command = %s, want tea.clearScreenMsg", got)
	}
	if model.mode != modePlaybackConfig {
		t.Fatalf("mode = %v, want playback config", model.mode)
	}

	view := model.View()
	for _, want := range []string{"PLAYBACK", "Crossfade", "Shuffle", "Repeat", "off", "UP/DOWN MOVE"} {
		if !strings.Contains(strings.ToUpper(view), strings.ToUpper(want)) {
			t.Fatalf("playback config view missing %q:\n%s", want, view)
		}
	}

	updated, cmd = model.Update(key(" "))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected crossfade toggle command")
	}
	updated, refreshCmd := model.Update(runCmd(cmd))
	model = updated.(Model)
	if !backend.crossfade {
		t.Fatal("backend crossfade was not toggled")
	}
	if !model.status.CrossfadeKnown || !model.status.CrossfadeEnabled {
		t.Fatalf("model crossfade = known %v enabled %v, want on", model.status.CrossfadeKnown, model.status.CrossfadeEnabled)
	}
	if model.message != "crossfade on" {
		t.Fatalf("message = %q, want crossfade on", model.message)
	}
	if refreshCmd == nil {
		t.Fatal("expected status refresh after crossfade toggle")
	}

	updated, cmd = model.Update(key("down"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect command on playback config move")
	}
	if model.playbackConfigIndex != 1 {
		t.Fatalf("playbackConfigIndex = %d, want 1", model.playbackConfigIndex)
	}

	_, cmd = model.Update(key("enter"))
	if cmd == nil {
		t.Fatal("expected shuffle toggle command")
	}
	updated, refreshCmd = model.Update(runCmd(cmd))
	model = updated.(Model)
	if !backend.shuffle {
		t.Fatal("backend shuffle was not toggled")
	}
	if !model.status.ShuffleKnown || !model.status.ShuffleEnabled {
		t.Fatalf("model shuffle = known %v enabled %v, want on", model.status.ShuffleKnown, model.status.ShuffleEnabled)
	}
	if model.message != "shuffle on" {
		t.Fatalf("message = %q, want shuffle on", model.message)
	}
	if refreshCmd == nil {
		t.Fatal("expected status refresh after shuffle toggle")
	}

	updated, cmd = model.Update(key("down"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect command on playback config move")
	}
	if model.playbackConfigIndex != 2 {
		t.Fatalf("playbackConfigIndex = %d, want 2", model.playbackConfigIndex)
	}

	updated, cmd = model.Update(key(" "))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected repeat toggle command")
	}
	updated, refreshCmd = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.status.RepeatMode != "all" {
		t.Fatalf("repeat mode = %q, want all", model.status.RepeatMode)
	}
	if !model.status.ShuffleEnabled {
		t.Fatal("shuffle should remain enabled when repeat all is enabled")
	}
	if model.message != "repeat all" {
		t.Fatalf("message = %q, want repeat all", model.message)
	}
	if refreshCmd == nil {
		t.Fatal("expected status refresh after repeat toggle")
	}

	_, cmd = model.Update(key("enter"))
	if cmd == nil {
		t.Fatal("expected repeat toggle command")
	}
	updated, refreshCmd = model.Update(runCmd(cmd))
	model = updated.(Model)
	if model.status.RepeatMode != "once" {
		t.Fatalf("repeat mode = %q, want once", model.status.RepeatMode)
	}
	if model.message != "repeat once" {
		t.Fatalf("message = %q, want repeat once", model.message)
	}
	if refreshCmd == nil {
		t.Fatal("expected status refresh after repeat toggle")
	}
}

func TestPlaybackConfigModalCloses(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.mode = modePlaybackConfig

	updated, cmd := model.Update(key("esc"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect close command")
	}
	if model.mode != modeDashboard {
		t.Fatalf("mode = %v, want dashboard", model.mode)
	}
}

func TestPlaybackConfigRendersUnknownCrossfade(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 110
	model.mode = modePlaybackConfig
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}}
	model.status = Status{State: "PLAYING", AlbumArt: "http://example.test/art.jpg"}
	model.artURL = model.status.AlbumArt
	model.artView = "\x1b_Ga=T,C=1,f=100,c=16,r=8,z=-1;AAAA\x1b\\"
	model.artFallbackView = "▀▀▀▀\n▀▀▀▀"

	view := model.View()
	for _, want := range []string{"PLAYBACK", "Crossfade", "Shuffle", "Repeat", "unknown"} {
		if !strings.Contains(strings.ToUpper(view), strings.ToUpper(want)) {
			t.Fatalf("playback config view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "\x1b_Ga=T") {
		t.Fatalf("playback config should use block album art while the modal is active:\n%s", view)
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

	_, cmd = model.Update(key("enter"))
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

func TestPlaylistSearchShowsPreviewTracks(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		searchFn: func(query string) []SearchResult {
			return []SearchResult{{
				Item: sonos.SMAPIItem{ID: "spotify:playlist:" + query, ItemType: "playlist", Title: "Playlist " + query},
			}}
		},
		browseFn: func(id string) []SearchResult {
			return []SearchResult{
				{Item: sonos.SMAPIItem{ID: id + ":1", ItemType: "track", Title: "Track One"}},
				{Item: sonos.SMAPIItem{ID: id + ":2", ItemType: "track", Title: "Track Two"}},
			}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch
	model.searchCategory = "playlists"

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected search command")
	}
	updated, previewCmd := model.Update(runCmd(cmd))
	model = updated.(Model)
	if previewCmd == nil {
		t.Fatal("expected playlist preview command")
	}
	updated, _ = model.Update(runCmd(previewCmd))
	model = updated.(Model)
	if len(model.searchPreviewItems) != 2 {
		t.Fatalf("preview items = %d, want 2", len(model.searchPreviewItems))
	}

	view := model.View()
	for _, want := range []string{
		"PLAYLIST PREVIEW",
		"Track One",
		"Track Two",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("playlist preview missing %q:\n%s", want, view)
		}
	}
}

func TestPlaylistCarouselRendersOnlyInPlaylistMode(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.mode = modeSearch
	tracksView := model.renderSearchContent(76)
	if strings.Contains(tracksView, "Pinned") {
		t.Fatalf("track search rendered playlist carousel:\n%s", tracksView)
	}

	model.searchCategory = "playlists"
	playlistView := model.renderSearchContent(76)
	fieldAt := strings.Index(playlistView, "type to search")
	pinnedAt := strings.Index(playlistView, "Pinned")
	placeholderAt := strings.Index(playlistView, "Search results will appear here")
	if pinnedAt < 0 {
		t.Fatalf("playlist carousel missing:\n%s", playlistView)
	}
	if fieldAt < 0 || pinnedAt <= fieldAt || placeholderAt <= pinnedAt {
		t.Fatalf("carousel not between search field and result placeholder:\n%s", playlistView)
	}
}

func TestPlaylistCarouselRecentFitsSearchPane(t *testing.T) {
	model := NewModel(&fakeBackend{}, Config{})
	model.searchCategory = "playlists"
	model.carouselRecent = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:one", ItemType: "playlist", Title: "Playlist One"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:two", ItemType: "playlist", Title: "Playlist Two"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:three", ItemType: "playlist", Title: "Playlist Three"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:four", ItemType: "playlist", Title: "Playlist Four"}},
	}

	const width = 48
	view := model.renderPlaylistCarousel(width)
	if strings.Contains(view, "Playlist Four") {
		t.Fatalf("recent carousel rendered more than three visible cards:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("playlist carousel line width = %d, want <= %d:\n%s", got, width, view)
		}
	}
}

func TestPlaylistCarouselResolvesDefaultPinsWithFallback(t *testing.T) {
	backend := &fakeBackend{}
	model := NewModel(backend, testConfig())
	msg := runCmd(playlistCarouselCmd(backend, testConfig(), Room{Name: "Kitchen", IP: "192.0.2.10"}, model.carouselStore, 1)).(playlistCarouselMsg)

	if got := strings.Join(backend.resolvedTitles, ","); got != "Release Radar" {
		t.Fatalf("resolved titles = %q", got)
	}
	if len(msg.pinned) != 0 {
		t.Fatalf("unresolved Release Radar should be removed, got pins %#v", msg.pinned)
	}
}

func TestPlaylistCarouselKeepsOnlySpotifyReleaseRadar(t *testing.T) {
	backend := &fakeBackend{
		resolveFn: func(title string) (SearchResult, bool) {
			switch title {
			case "Release Radar":
				return SearchResult{Item: sonos.SMAPIItem{
					ID:         "spotify:playlist:37i9dQZF1DX0s5kDXi1oC5",
					ItemType:   "playlist",
					Title:      title,
					Creator:    "Spotify",
					ArtworkURI: "https://example.test/release-radar.jpg",
				}}, true
			default:
				return SearchResult{}, false
			}
		},
	}
	model := NewModel(backend, testConfig())
	msg := runCmd(playlistCarouselCmd(backend, testConfig(), Room{Name: "Kitchen", IP: "192.0.2.10"}, model.carouselStore, 1)).(playlistCarouselMsg)

	if len(msg.pinned) != 1 {
		t.Fatalf("pins = %#v, want release radar only", msg.pinned)
	}
	releaseRadar := msg.pinned[0]
	if releaseRadar.Title() != "Release Radar" || releaseRadar.Item.Creator != "Spotify" || releaseRadar.Item.ArtworkURI == "" {
		t.Fatalf("Release Radar was not preserved with Spotify metadata: %#v", releaseRadar)
	}
}

func TestPlaylistCarouselSeedsDefaultPopularPlaylistsAndLikedSongs(t *testing.T) {
	backend := &fakeBackend{
		shelfFn: func(shelfID string) []SearchResult {
			switch shelfID {
			case "root":
				return []SearchResult{
					{Item: sonos.SMAPIItem{ID: "spotify:shelf:popular", ItemType: "container", Title: "Popular Playlists"}},
					{Item: sonos.SMAPIItem{ID: "spotify:shelf:yourmusic", ItemType: "container", Title: "Your Music"}},
				}
			case "spotify:shelf:popular":
				return []SearchResult{
					{Item: sonos.SMAPIItem{ID: "spotify:playlist:popular-one", ItemType: "playlist", Title: "Popular One", Creator: "Spotify", ArtworkURI: "https://example.test/popular.jpg"}},
				}
			case "spotify:shelf:yourmusic":
				return []SearchResult{
					{Item: sonos.SMAPIItem{ID: "your_songs", ItemType: "trackList", Title: "Songs", Creator: "Spotify", ArtworkURI: "https://example.test/yourmusic.jpg"}},
				}
			default:
				return nil
			}
		},
		resolveFn: func(title string) (SearchResult, bool) {
			if title != "Release Radar" {
				return SearchResult{}, false
			}
			return SearchResult{Item: sonos.SMAPIItem{
				ID:         "spotify:playlist:release-radar",
				ItemType:   "playlist",
				Title:      "Release Radar",
				Creator:    "Spotify",
				ArtworkURI: "https://example.test/release-radar.jpg",
			}}, true
		},
	}
	model := NewModel(backend, testConfig())

	msg := runCmd(playlistCarouselCmd(backend, testConfig(), Room{Name: "Kitchen", IP: "192.0.2.10"}, model.carouselStore, 1)).(playlistCarouselMsg)
	if !msg.store.DefaultPinsSeeded {
		t.Fatalf("default pins were not marked seeded: %#v", msg.store)
	}
	if len(msg.pinned) != 3 {
		t.Fatalf("pins = %#v, want release radar + popular playlists + liked songs", msg.pinned)
	}
	if got := []string{msg.pinned[0].Title(), msg.pinned[1].Title(), msg.pinned[2].Title()}; strings.Join(got, ",") != "Release Radar,Popular One,Liked Songs" {
		t.Fatalf("seeded pin order = %v", got)
	}
}

func TestPlaylistCarouselDoesNotSeedDefaultsAfterUserCustomization(t *testing.T) {
	store := playlistCarouselStore{
		Pins: []playlistCarouselStoreItem{
			{ID: "spotify:playlist:custom", ItemType: "playlist", Title: "My Custom Playlist"},
		},
	}
	backend := &fakeBackend{
		shelfFn: func(shelfID string) []SearchResult {
			if shelfID == "root" {
				return []SearchResult{
					{Item: sonos.SMAPIItem{ID: "spotify:shelf:popular", ItemType: "container", Title: "Popular Playlists"}},
				}
			}
			if shelfID == "spotify:shelf:popular" {
				return []SearchResult{
					{Item: sonos.SMAPIItem{ID: "spotify:playlist:popular-one", ItemType: "playlist", Title: "Popular One", Creator: "Spotify"}},
				}
			}
			return nil
		},
	}

	msg := runCmd(playlistCarouselCmd(backend, testConfig(), Room{Name: "Kitchen", IP: "192.0.2.10"}, store, 1)).(playlistCarouselMsg)
	if len(msg.pinned) != 1 || msg.pinned[0].Title() != "My Custom Playlist" {
		t.Fatalf("custom pins were modified: %#v", msg.pinned)
	}
	if !msg.store.DefaultPinsSeeded {
		t.Fatalf("customized store should still be marked seeded: %#v", msg.store)
	}
}

func TestPlaylistCarouselDropsRandomReleaseRadar(t *testing.T) {
	store := defaultPlaylistCarouselStore()
	store.Pins[0] = playlistCarouselStoreItem{
		ID:       "spotify:playlist:random",
		ItemType: "playlist",
		Title:    "Release Radar",
		Creator:  "Someone Else",
	}
	backend := &fakeBackend{
		resolveFn: func(title string) (SearchResult, bool) {
			if title == "Release Radar" {
				return SearchResult{Item: sonos.SMAPIItem{ID: "spotify:playlist:random", ItemType: "playlist", Title: title, Creator: "Someone Else"}}, true
			}
			return SearchResult{}, false
		},
	}
	msg := runCmd(playlistCarouselCmd(backend, testConfig(), Room{Name: "Kitchen", IP: "192.0.2.10"}, store, 1)).(playlistCarouselMsg)

	for _, pin := range msg.pinned {
		if pin.Title() == "Release Radar" {
			t.Fatalf("random Release Radar should be removed: %#v", msg.pinned)
		}
	}
}

func TestPlaylistCarouselPinAndRecentPersistence(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig()
	cfg.CarouselPath = filepath.Join(dir, "playlist_carousel.json")
	backend := &fakeBackend{}
	model := NewModel(backend, cfg)
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusCarousel
	recent := SearchResult{Item: sonos.SMAPIItem{ID: "spotify:playlist:recent", ItemType: "playlist", Title: "Recent Playlist"}}
	model.carouselRecent = []SearchResult{recent}
	model.carouselTab = 1

	updated, _ := model.Update(key(" "))
	model = updated.(Model)
	if model.carouselTab != 0 || len(model.carouselPinned) == 0 || model.carouselPinned[model.carouselIndex].Item.ID != "spotify:playlist:recent" {
		t.Fatalf("pinning did not switch to pinned tab: tab=%d index=%d pins=%#v", model.carouselTab, model.carouselIndex, model.carouselPinned)
	}
	store, err := LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore: %v", err)
	}
	if !storeHasPlaylist(store.Pins, "spotify:playlist:recent") {
		t.Fatalf("pinned playlist was not persisted: %#v", store.Pins)
	}

	model.carouselTab = 0
	model.carouselIndex = len(model.carouselPinned) - 1
	updated, _ = model.Update(key(" "))
	model = updated.(Model)
	store, err = LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore after unpin: %v", err)
	}
	if storeHasPlaylist(store.Pins, "spotify:playlist:recent") {
		t.Fatalf("unpinned playlist still persisted: %#v", store.Pins)
	}

	playlistA := SearchResult{Item: sonos.SMAPIItem{ID: "spotify:playlist:a", ItemType: "playlist", Title: "Playlist A"}}
	playlistB := SearchResult{Item: sonos.SMAPIItem{ID: "spotify:playlist:b", ItemType: "playlist", Title: "Playlist B"}}
	_, _ = model.Update(actionMsg{playlistPlayed: true, playedPlaylist: playlistA})
	model = updated.(Model)
	updated, _ = model.Update(actionMsg{playlistPlayed: true, playedPlaylist: playlistB})
	model = updated.(Model)
	_, _ = model.Update(actionMsg{playlistPlayed: true, playedPlaylist: playlistA})
	store, err = LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore recent: %v", err)
	}
	if len(store.Recent) < 2 || store.Recent[0].ID != "spotify:playlist:a" || store.Recent[1].ID != "spotify:playlist:b" {
		t.Fatalf("recent MRU order = %#v", store.Recent)
	}
}

func TestPlaylistCarouselLikedSongsPinAndRecentPersistence(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig()
	cfg.CarouselPath = filepath.Join(dir, "playlist_carousel.json")
	model := NewModel(&fakeBackend{}, cfg)
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusCarousel
	likedSongs := SearchResult{Item: sonos.SMAPIItem{ID: "your_songs", ItemType: "trackList", Title: "Songs", Creator: "Spotify"}}
	model.carouselRecent = []SearchResult{likedSongs}
	model.carouselTab = 1

	updated, _ := model.Update(key(" "))
	model = updated.(Model)
	if model.carouselTab != 0 || len(model.carouselPinned) == 0 || model.carouselPinned[model.carouselIndex].Item.ID != "your_songs" {
		t.Fatalf("pinning liked songs did not switch to pinned tab: tab=%d index=%d pins=%#v", model.carouselTab, model.carouselIndex, model.carouselPinned)
	}
	store, err := LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore: %v", err)
	}
	if !storeHasPlaylist(store.Pins, "your_songs") {
		t.Fatalf("liked songs was not persisted: %#v", store.Pins)
	}
	normalizedLikedSongs := false
	for _, pin := range store.Pins {
		if pin.ID == "your_songs" && pin.Title == "Liked Songs" {
			normalizedLikedSongs = true
			break
		}
	}
	if !normalizedLikedSongs {
		t.Fatalf("liked songs title was not normalized: %#v", store.Pins)
	}

	_, _ = model.Update(actionMsg{playlistPlayed: true, playedPlaylist: likedSongs})
	store, err = LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore recent: %v", err)
	}
	if len(store.Recent) == 0 || store.Recent[0].ID != "your_songs" || store.Recent[0].Title != "Liked Songs" {
		t.Fatalf("liked songs recent persistence = %#v", store.Recent)
	}
}

func TestPlaylistCarouselPinSurvivesStaleRefresh(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig()
	cfg.CarouselPath = filepath.Join(dir, "playlist_carousel.json")
	backend := &fakeBackend{}
	model := NewModel(backend, cfg)
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusCarousel
	model.carouselPinned = nil
	model.carouselStore.Pins = nil
	model.carouselRecent = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:recent", ItemType: "playlist", Title: "Recent Playlist"}},
	}
	model.startPlaylistCarouselLoad()
	staleGeneration := model.carouselGeneration
	staleStore := model.carouselStore

	updated, _ := model.Update(key(" "))
	model = updated.(Model)
	if !storeHasPlaylist(model.carouselStore.Pins, "spotify:playlist:recent") {
		t.Fatalf("pin was not applied locally: %#v", model.carouselStore.Pins)
	}

	updated, _ = model.Update(playlistCarouselMsg{
		generation: staleGeneration,
		store:      staleStore,
		pinned:     playlistCarouselPinnedResults(staleStore),
		recent:     playlistCarouselRecentResults(staleStore),
	})
	model = updated.(Model)
	if !storeHasPlaylist(model.carouselStore.Pins, "spotify:playlist:recent") {
		t.Fatalf("stale refresh overwrote local pin: %#v", model.carouselStore.Pins)
	}
}

func TestPlaylistSearchResultPinAndUnpinWithP(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig()
	cfg.CarouselPath = filepath.Join(dir, "playlist_carousel.json")
	model := NewModel(&fakeBackend{}, cfg)
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.carouselStore.Pins = nil
	model.carouselPinned = nil
	model.searchFocus = searchFocusResults
	model.searchQuery = "mix"
	model.searchPreviewQuery = "mix"
	model.searchItems = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:mix", ItemType: "playlist", Title: "Mix Playlist"}},
	}

	updated, _ := model.Update(key("ctrl+f"))
	model = updated.(Model)
	if model.carouselTab != 0 || len(model.carouselPinned) != 1 || model.carouselPinned[0].Item.ID != "spotify:playlist:mix" {
		t.Fatalf("result pin did not update pinned carousel: tab=%d index=%d pins=%#v", model.carouselTab, model.carouselIndex, model.carouselPinned)
	}
	store, err := LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore: %v", err)
	}
	if !storeHasPlaylist(store.Pins, "spotify:playlist:mix") {
		t.Fatalf("pinned result not persisted: %#v", store.Pins)
	}

	model.searchFocus = searchFocusCarousel
	model.carouselIndex = 0
	_, _ = model.Update(key("ctrl+f"))
	store, err = LoadPlaylistCarouselStore(cfg.CarouselPath)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore after unpin: %v", err)
	}
	if storeHasPlaylist(store.Pins, "spotify:playlist:mix") {
		t.Fatalf("unpinned result still persisted: %#v", store.Pins)
	}
}

func TestPlaylistSearchResultPinStartsThumbFetchImmediately(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig()
	cfg.CarouselPath = filepath.Join(dir, "playlist_carousel.json")
	model := NewModel(&fakeBackend{}, cfg)
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.carouselStore.Pins = nil
	model.carouselPinned = nil
	model.searchFocus = searchFocusResults
	model.searchItems = []SearchResult{
		{Item: sonos.SMAPIItem{
			ID:         "spotify:playlist:mix",
			ItemType:   "playlist",
			Title:      "Mix Playlist",
			ArtworkURI: "https://example.test/mix.jpg",
		}},
	}

	_, cmd := model.Update(key("ctrl+f"))
	if cmd == nil {
		t.Fatal("expected pin command batch")
	}
	msg := runCmd(cmd)
	if _, ok := msg.(playlistThumbMsg); !ok {
		t.Fatalf("expected playlist thumb fetch after pin, got %T", msg)
	}
}

func TestPlaylistCarouselPreviewIgnoresStaleResponses(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
		browseFn: func(id string) []SearchResult {
			return []SearchResult{{Item: sonos.SMAPIItem{ID: id + ":track", ItemType: "track", Title: "Track for " + id}}}
		},
	}
	model := NewModel(backend, testConfig())
	model.rooms = backend.rooms
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusCarousel
	model.carouselPinned = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:one", ItemType: "playlist", Title: "One"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:two", ItemType: "playlist", Title: "Two"}},
	}

	updated, staleCmd := model.previewSelectedCarouselResult()
	model = updated.(Model)
	if staleCmd == nil {
		t.Fatal("expected first preview command")
	}
	updated, freshCmd := model.Update(key("right"))
	model = updated.(Model)
	if freshCmd == nil {
		t.Fatal("expected preview command after moving carousel")
	}
	updated, _ = model.Update(runCmd(freshCmd))
	model = updated.(Model)
	if model.searchPreviewItemID != "spotify:playlist:two" || len(model.searchPreviewItems) != 1 {
		t.Fatalf("fresh preview not applied: id=%q items=%#v", model.searchPreviewItemID, model.searchPreviewItems)
	}
	updated, _ = model.Update(runCmd(staleCmd))
	model = updated.(Model)
	if model.searchPreviewItemID != "spotify:playlist:two" || model.searchPreviewItems[0].Item.ID != "spotify:playlist:two:track" {
		t.Fatalf("stale preview overwrote fresh preview: id=%q items=%#v", model.searchPreviewItemID, model.searchPreviewItems)
	}
}

func TestPlaylistCarouselAndSearchNavigationDoNotConflict(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusResults
	model.searchQuery = "mix"
	model.searchItems = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:one", ItemType: "playlist", Title: "One"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:two", ItemType: "playlist", Title: "Two"}},
	}
	model.carouselPinned = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:pinned-one", ItemType: "playlist", Title: "Pinned One"}},
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:pinned-two", ItemType: "playlist", Title: "Pinned Two"}},
	}

	updated, _ := model.Update(key("down"))
	model = updated.(Model)
	if model.searchFocus != searchFocusResults || model.searchIndex != 1 {
		t.Fatalf("down in results changed focus/index to %v/%d", model.searchFocus, model.searchIndex)
	}
	updated, _ = model.Update(key("up"))
	model = updated.(Model)
	if model.searchFocus != searchFocusResults || model.searchIndex != 0 {
		t.Fatalf("up within results changed focus/index to %v/%d", model.searchFocus, model.searchIndex)
	}
	updated, _ = model.Update(key("up"))
	model = updated.(Model)
	if model.searchFocus != searchFocusCarousel {
		t.Fatalf("up at first result did not move to carousel: %v", model.searchFocus)
	}
	updated, _ = model.Update(key("right"))
	model = updated.(Model)
	if model.carouselIndex != 1 || model.searchIndex != 0 {
		t.Fatalf("right in carousel changed carousel/search indexes to %d/%d", model.carouselIndex, model.searchIndex)
	}
	updated, _ = model.Update(key("down"))
	model = updated.(Model)
	if model.searchFocus != searchFocusResults || model.searchIndex != 0 {
		t.Fatalf("down from carousel did not return to results: %v/%d", model.searchFocus, model.searchIndex)
	}
}

func TestSearchResultsScrollSelectionIntoView(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.mode = modeSearch
	model.searchItems = make([]SearchResult, 12)
	for i := range model.searchItems {
		model.searchItems[i] = SearchResult{Item: sonos.SMAPIItem{ID: fmt.Sprintf("spotify:track:%d", i), ItemType: "track", Title: fmt.Sprintf("Track %d", i+1)}}
	}
	model.searchFocus = searchFocusResults

	for i := 0; i < 9; i++ {
		updated, _ := model.Update(key("down"))
		model = updated.(Model)
	}
	if model.searchIndex != 9 {
		t.Fatalf("searchIndex = %d, want 9", model.searchIndex)
	}
	if model.searchOffset == 0 {
		t.Fatal("searchOffset did not advance")
	}

	view := model.renderSearchContent(76)
	if !strings.Contains(view, "Track 10") {
		t.Fatalf("selected track not visible after scroll:\n%s", view)
	}
	if !strings.Contains(view, "+") {
		t.Fatalf("expected scroll indicators in view:\n%s", view)
	}
}

func TestPlaylistCarouselArtworkFallbackRendering(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.mode = modeSearch
	model.searchCategory = "playlists"
	model.searchFocus = searchFocusCarousel
	model.carouselPinned = []SearchResult{
		{Item: sonos.SMAPIItem{ID: "spotify:playlist:release-radar", ItemType: "playlist", Title: "Release Radar"}},
	}
	view := model.renderPlaylistCarousel(76)
	if !strings.Contains(view, "RR") {
		t.Fatalf("fallback initials missing:\n%s", view)
	}

	model.carouselPinned[0].Item.ArtworkURI = "https://example.test/cover.jpg"
	model.carouselThumbViews = map[string]string{"https://example.test/cover.jpg": "▓▓▓▓▓▓▓▓\n▓▓▓▓▓▓▓▓\n▓▓▓▓▓▓▓▓\n▓▓▓▓▓▓▓▓"}
	view = model.renderPlaylistCarousel(76)
	if strings.Contains(view, "RR") || !strings.Contains(view, "▓▓▓▓▓▓▓▓") {
		t.Fatalf("thumbnail artwork missing:\n%s", view)
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
	_, cmd = model.handleMacHelperCommand("togglePlayPause")
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
		"▸",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "╭") || !strings.Contains(view, "━") {
		t.Fatalf("view missing styled panel/progress glyphs:\n%s", view)
	}
	if strings.Contains(strings.ToUpper(view), "SPOTIFY / TRACKS") {
		t.Fatalf("dashboard unexpectedly rendered search UI:\n%s", view)
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
	t.Setenv("TERM_PROGRAM", "Ghostty")

	backend := &fakeBackend{}
	model := NewModel(backend, testConfig())
	model.width = 110
	model.loading = false
	model.mode = modeSearch
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}}
	model.status = Status{State: "PAUSED_PLAYBACK", Title: "Current Track", Artist: "Artist", AlbumArt: "http://example.test/art.jpg"}
	model.artURL = model.status.AlbumArt
	model.artView = "\x1b_Ga=T,C=1,f=100,c=16,r=8,z=-1;AAAA\x1b\\"
	model.artFallbackView = "▀▀▀▀\n▀▀▀▀"
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
		"ENTER PLAY  CTRL+T TRACKS  CTRL+P PLAYLISTS  ESC CLOSE",
		"▸",
		"█",
	} {
		if !strings.Contains(strings.ToUpper(view), strings.ToUpper(want)) {
			t.Fatalf("search view missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "╭") || !strings.Contains(strings.ToUpper(view), "SPOTIFY / TRACKS") {
		t.Fatalf("search modal missing expected framing or label:\n%s", view)
	}
	if !strings.HasPrefix(view, clearKittyGraphics()) {
		t.Fatalf("search surface should clear terminal graphics before drawing modal:\n%s", view)
	}
	if strings.Contains(view, "\x1b_Ga=T") {
		t.Fatalf("search surface should use block album art while the modal is active:\n%s", view)
	}
	if strings.Contains(view, "\x1b[s") || strings.Contains(view, "\x1b[u") {
		t.Fatalf("search surface should not use cursor-position overlays:\n%s", view)
	}
	if !strings.Contains(view, "Kitchen") || !strings.Contains(view, "Current Track") {
		t.Fatalf("search surface should keep the dashboard rendered behind the modal:\n%s", view)
	}
}

func TestViewKeepsKittyArtWhenSearchSurfaceDoesNotOverlapCover(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "Ghostty")

	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 110
	model.height = 60
	model.loading = false
	model.mode = modeSearch
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}}
	model.status = Status{State: "PAUSED_PLAYBACK", Title: "Current Track", Artist: "Artist", AlbumArt: "http://example.test/art.jpg"}
	model.artURL = model.status.AlbumArt
	model.artView = "\x1b_Ga=T,C=1,f=100,c=16,r=8,z=-1;AAAA\x1b\\"
	model.artFallbackView = "▀▀▀▀\n▀▀▀▀"

	view := model.View()
	if !strings.Contains(strings.ToUpper(view), "SPOTIFY / TRACKS") {
		t.Fatalf("search modal missing expected label:\n%s", view)
	}
	if !strings.Contains(view, "\x1b_Ga=T") || !strings.Contains(view, "z=-1") {
		t.Fatalf("search surface should keep high-quality album art when the modal does not overlap it:\n%s", view)
	}
}

func TestViewRendersSearchSurfaceFallsBackForTopLayerKittyArt(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "Ghostty")

	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 110
	model.loading = false
	model.mode = modeSearch
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}}
	model.status = Status{State: "PAUSED_PLAYBACK", Title: "Current Track", Artist: "Artist", AlbumArt: "http://example.test/art.jpg"}
	model.artURL = model.status.AlbumArt
	model.artView = "\x1b_Ga=T,C=1,f=100,c=16,r=8;AAAA\x1b\\"
	model.artFallbackView = "▀▀▀▀\n▀▀▀▀"

	view := model.View()
	if strings.Contains(view, "\x1b_Ga=T") {
		t.Fatalf("search surface should not redraw top-layer kitty art behind modal:\n%s", view)
	}
	if !strings.Contains(view, "▀▀▀▀") {
		t.Fatalf("search surface should keep fallback album art behind the modal:\n%s", view)
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

func TestWideFooterStartsUnderRightPane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	footer := model.renderFooterRow(108)
	gutter := strings.Repeat(" ", sidebarWidth+paneGapWidth)
	if !strings.HasPrefix(footer, gutter) {
		t.Fatalf("footer did not start after sidebar gutter:\n%q", footer)
	}
	if got := lipgloss.Width(footer); got > 108 {
		t.Fatalf("footer width = %d, want <= 108:\n%s", got, footer)
	}
}

func TestWideFooterCentersInRightPane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	footer := model.renderFooterRow(108)
	rightWidth := 108 - sidebarWidth - paneGapWidth
	right := footer[sidebarWidth+paneGapWidth:]
	if got := lipgloss.Width(right); got != rightWidth {
		t.Fatalf("right footer width = %d, want %d:\n%s", got, rightWidth, footer)
	}
	if strings.HasPrefix(right, " ") {
		return
	}
	t.Fatalf("footer was not centered in right pane:\n%q", right)
}

func TestWideFooterCentersInMainPaneWithQueuePane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.queueItems = []QueueItem{{Position: 1, Title: "Track One"}}

	footer := model.renderFooterRow(130)
	gutter := strings.Repeat(" ", sidebarWidth+paneGapWidth)
	if !strings.HasPrefix(footer, gutter) {
		t.Fatalf("footer did not start after sidebar gutter:\n%q", footer)
	}

	rightWidth := 130 - sidebarWidth - queuePaneWidth - 2*paneGapWidth
	right := strings.TrimPrefix(footer, gutter)
	if got := lipgloss.Width(right); got != rightWidth {
		t.Fatalf("footer width in main pane = %d, want %d:\n%s", got, rightWidth, footer)
	}
	if strings.HasPrefix(right, " ") {
		return
	}
	t.Fatalf("footer was not centered in main pane:\n%q", right)
}

func TestWideBodyKeepsPaneWidthsAligned(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}
	body := model.renderBody(108)
	if got := lipgloss.Width(body); got != 108 {
		t.Fatalf("body width = %d, want 108:\n%s", got, body)
	}
	if got := lipgloss.Width(model.renderRooms(sidebarWidth)); got != sidebarWidth {
		t.Fatalf("rooms pane width = %d, want %d", got, sidebarWidth)
	}
	rightWidth := 108 - sidebarWidth - paneGapWidth
	if got := lipgloss.Width(model.renderRightPane(rightWidth)); got != rightWidth {
		t.Fatalf("right pane width = %d, want %d", got, rightWidth)
	}
}

func TestWideBodyRendersQueuePane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 132
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}
	model.queueItems = []QueueItem{
		{Position: 1, Title: "Track One", Artist: "Artist One"},
		{Position: 2, Title: "Track Two", Artist: "Artist Two"},
	}
	model.queueTotal = 2
	model.status.QueuePosition = 2

	body := model.renderBody(130)
	if got := lipgloss.Width(body); got != 130 {
		t.Fatalf("body width = %d, want 130:\n%s", got, body)
	}
	for _, want := range []string{"QUEUE", "Track One", "Track Two"} {
		if !strings.Contains(strings.ToUpper(body), strings.ToUpper(want)) {
			t.Fatalf("queue body missing %q:\n%s", want, body)
		}
	}
}

func TestQueueFooterSticksToCenterPane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 132
	model.loading = false
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}
	model.message = "footer"
	for i := 1; i <= 24; i++ {
		model.queueItems = append(model.queueItems, QueueItem{Position: i, Title: fmt.Sprintf("Track %02d", i)})
	}
	model.queueTotal = len(model.queueItems)

	body := model.renderAppContent(130)
	if !strings.Contains(body, "footer") {
		t.Fatalf("footer message missing from body:\n%s", body)
	}
	if !strings.Contains(body, "Track 01") {
		t.Fatalf("queue content missing from body:\n%s", body)
	}
}

func TestCompactBodyHidesQueuePane(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 100
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}
	model.queueItems = []QueueItem{{Position: 1, Title: "Track One"}}
	body := model.renderBody(98)
	if strings.Contains(strings.ToUpper(body), "QUEUE") || strings.Contains(body, "Track One") {
		t.Fatalf("compact body unexpectedly rendered queue:\n%s", body)
	}
}

func TestCompactRoomsUseSingleLineRows(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.rooms = []Room{
		{Name: "Living Room", IP: "192.0.2.10", GroupMembers: []string{"Living Room"}},
		{Name: "Office", IP: "192.0.2.11", GroupMembers: []string{"Office"}},
	}

	compact := model.renderCompactRooms(70)
	if count := strings.Count(compact, "Living Room"); count != 1 {
		t.Fatalf("compact room should render once, got %d:\n%s", count, compact)
	}
	if strings.Contains(compact, "192.0.2.10") {
		t.Fatalf("compact room unexpectedly rendered secondary IP line:\n%s", compact)
	}
	if strings.Contains(compact, "\nLiving Room") {
		t.Fatalf("compact room should keep selection marker on the room row:\n%s", compact)
	}
}

func TestLayoutShortcutTogglesCompactMode(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Kitchen", IP: "192.0.2.10", CoordinatorIP: "192.0.2.10"}},
	}
	dir := t.TempDir()
	cfg := testConfig()
	cfg.LayoutConfigPath = filepath.Join(dir, "layout.json")
	model := NewModel(backend, cfg)

	updated, cmd := model.Update(key("ctrl+l"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect a command when toggling layout")
	}
	if !model.compactLayout {
		t.Fatal("expected compact layout to toggle on")
	}
	if model.message != "layout: compact" {
		t.Fatalf("layout message = %q, want compact announcement", model.message)
	}
	if _, err := os.Stat(cfg.LayoutConfigPath); err != nil {
		t.Fatalf("layout config not written: %v", err)
	}
	stored, err := LoadCompactLayout(cfg.LayoutConfigPath)
	if err != nil {
		t.Fatalf("LoadCompactLayout: %v", err)
	}
	if !stored {
		t.Fatal("stored compact layout = false, want true")
	}
}

func TestCompactLayoutHidesSidePanes(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.compactLayout = true
	model.width = 130
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}
	model.queueItems = []QueueItem{{Position: 1, Title: "Track One"}}
	model.status = Status{Title: "Track One", Artist: "Artist"}

	body := model.renderAppContent(128)
	if strings.Contains(strings.ToUpper(body), "ROOMS") {
		t.Fatalf("compact layout unexpectedly rendered rooms pane:\n%s", body)
	}
	if strings.Contains(strings.ToUpper(body), "QUEUE") {
		t.Fatalf("compact layout unexpectedly rendered queue pane:\n%s", body)
	}
	if !strings.Contains(strings.ToUpper(body), "NOW PLAYING") {
		t.Fatalf("compact layout missing main pane:\n%s", body)
	}
}

func TestCompactCoverCentersArtwork(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.compactLayout = true
	model.status = Status{
		Title:    "Track",
		Artist:   "Artist",
		Album:    "Album",
		AlbumArt: "http://example.test/art.jpg",
	}
	model.artURL = model.status.AlbumArt
	model.artView = "▀▀▀▀\n▀▀▀▀"

	cover := model.renderCover(28)
	lines := strings.Split(cover, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "▀▀▀▀") {
			found = true
			if !strings.Contains(line, "     ▀▀▀▀") {
				t.Fatalf("compact cover art was not centered:\n%s", cover)
			}
		}
	}
	if !found {
		t.Fatalf("compact cover did not render artwork:\n%s", cover)
	}
}

func TestDashboardTabFocusesQueueWhenVisible(t *testing.T) {
	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 132
	model.rooms = []Room{{Name: "Living Room", IP: "192.0.2.10"}}

	updated, cmd := model.Update(key("tab"))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("did not expect command when focusing queue")
	}
	if model.dashboardFocus != focusQueue {
		t.Fatalf("focus = %v, want queue", model.dashboardFocus)
	}

	updated, cmd = model.Update(key("/"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected clear screen command when opening search")
	}
	if got := fmt.Sprintf("%T", runCmd(cmd)); got != "tea.clearScreenMsg" {
		t.Fatalf("search open command = %s, want tea.clearScreenMsg", got)
	}
	if model.mode != modeSearch {
		t.Fatalf("mode = %v, want search", model.mode)
	}
}

func TestQueueFocusDispatchesActions(t *testing.T) {
	backend := &fakeBackend{
		rooms: []Room{{Name: "Living Room", IP: "192.0.2.10"}},
		queuePage: QueuePage{Items: []QueueItem{
			{Position: 1, Title: "Track One"},
			{Position: 2, Title: "Track Two"},
			{Position: 3, Title: "Track Three"},
		}, TotalMatches: 3},
	}
	model := NewModel(backend, testConfig())
	model.width = 132
	model.rooms = backend.rooms
	model.queueItems = append([]QueueItem{}, backend.queuePage.Items...)
	model.queueTotal = 3
	model.dashboardFocus = focusQueue
	model.queueIndex = 1

	updated, cmd := model.Update(key("enter"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected queue play command")
	}
	_ = runCmd(cmd)
	if backend.playQueuePosition != 2 {
		t.Fatalf("play queue position = %d, want 2", backend.playQueuePosition)
	}

	updated, cmd = model.Update(key("x"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected queue remove command")
	}
	_ = runCmd(cmd)
	if backend.removeQueuePosition != 2 {
		t.Fatalf("remove queue position = %d, want 2", backend.removeQueuePosition)
	}

	updated, cmd = model.Update(key("["))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected queue move-up command")
	}
	_ = runCmd(cmd)
	if backend.moveFrom != 2 || backend.moveTo != 1 {
		t.Fatalf("move = %d -> %d, want 2 -> 1", backend.moveFrom, backend.moveTo)
	}

	updated, cmd = model.Update(key("]"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected queue move-down command")
	}
	_ = runCmd(cmd)
	if backend.moveFrom != 1 || backend.moveTo != 2 {
		t.Fatalf("move = %d -> %d, want 1 -> 2", backend.moveFrom, backend.moveTo)
	}

	_, cmd = model.Update(key("X"))
	if cmd == nil {
		t.Fatal("expected queue clear command")
	}
	_ = runCmd(cmd)
	if !backend.clearedQueue {
		t.Fatal("expected queue clear")
	}
}

func testConfig() Config {
	return Config{Timeout: time.Second, SearchService: "Spotify", SearchCategory: "tracks", SearchLimit: 10}
}

func storeHasPlaylist(items []playlistCarouselStoreItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func key(value string) tea.KeyMsg {
	switch value {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "ctrl+l":
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	case "ctrl+f":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
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
	rooms               []Room
	status              Status
	results             []SearchResult
	searchFn            func(query string) []SearchResult
	browseFn            func(id string) []SearchResult
	shelfFn             func(id string) []SearchResult
	resolveFn           func(title string) (SearchResult, bool)
	transportAction     string
	volume              int
	muted               bool
	crossfade           bool
	shuffle             bool
	repeat              string
	seekPosition        string
	seekDuration        string
	seekDelta           int
	queuePage           QueuePage
	playQueuePosition   int
	removeQueuePosition int
	clearedQueue        bool
	moveFrom            int
	moveTo              int
	played              sonos.SMAPIItem
	searchQueries       []string
	searchCategories    []string
	shelfIDs            []string
	resolvedTitles      []string
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

func (f *fakeBackend) ToggleCrossfade(context.Context, Room) (bool, error) {
	f.crossfade = !f.crossfade
	return f.crossfade, nil
}

func (f *fakeBackend) ToggleShuffle(context.Context, Room) (bool, error) {
	f.shuffle = !f.shuffle
	return f.shuffle, nil
}

func (f *fakeBackend) ToggleRepeat(context.Context, Room) (string, error) {
	switch strings.TrimSpace(strings.ToLower(f.repeat)) {
	case "all":
		f.repeat = "once"
	case "once":
		f.repeat = "off"
	default:
		f.repeat = "all"
	}
	return f.repeat, nil
}

func (f *fakeBackend) Scrub(_ context.Context, _ Room, position, duration string, deltaSeconds int) (string, error) {
	f.seekPosition = position
	f.seekDuration = duration
	f.seekDelta = deltaSeconds
	return "0:01:05", nil
}

func (f *fakeBackend) Queue(context.Context, Room, int, int) (QueuePage, error) {
	return f.queuePage, nil
}

func (f *fakeBackend) PlayQueuePosition(_ context.Context, _ Room, position int) error {
	f.playQueuePosition = position
	return nil
}

func (f *fakeBackend) RemoveQueuePosition(_ context.Context, _ Room, position int) error {
	f.removeQueuePosition = position
	return nil
}

func (f *fakeBackend) ClearQueue(context.Context, Room) error {
	f.clearedQueue = true
	return nil
}

func (f *fakeBackend) MoveQueuePosition(_ context.Context, _ Room, fromPosition, toPosition int) error {
	f.moveFrom = fromPosition
	f.moveTo = toPosition
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

func (f *fakeBackend) PlaylistShelf(_ context.Context, _ Room, _, shelfID string, _ int) ([]SearchResult, error) {
	f.shelfIDs = append(f.shelfIDs, shelfID)
	if f.shelfFn != nil {
		return f.shelfFn(shelfID), nil
	}
	return nil, nil
}

func (f *fakeBackend) ResolvePinnedPlaylist(_ context.Context, _ Room, _, title string) (SearchResult, bool, error) {
	f.resolvedTitles = append(f.resolvedTitles, title)
	if f.resolveFn != nil {
		result, ok := f.resolveFn(title)
		return result, ok, nil
	}
	return SearchResult{}, false, nil
}

func (f *fakeBackend) BrowsePlaylist(_ context.Context, _ Room, _ string, result SearchResult, _ int) ([]SearchResult, error) {
	if f.browseFn != nil {
		return f.browseFn(result.Item.ID), nil
	}
	return nil, nil
}

func (f *fakeBackend) PlaySearchResult(_ context.Context, _ Room, _ string, result SearchResult) error {
	f.played = result.Item
	return nil
}
