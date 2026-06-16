package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shlomiuziel/sonosh/internal/sonos"
	"github.com/shlomiuziel/sonosh/internal/tui"
)

func main() {
	themeConfigPath, err := tui.DefaultThemeConfigPath()
	if err != nil {
		themeConfigPath = ""
	}
	layoutConfigPath, err := tui.DefaultLayoutConfigPath()
	if err != nil {
		layoutConfigPath = ""
	}
	carouselConfigPath, err := tui.DefaultPlaylistCarouselConfigPath()
	if err != nil {
		carouselConfigPath = ""
	}
	storedTheme, err := tui.LoadThemeName(themeConfigPath)
	if err != nil {
		storedTheme = ""
	}
	storedCompact, err := tui.LoadCompactLayout(layoutConfigPath)
	if err != nil {
		storedCompact = false
	}
	defaultTheme := "aurora"
	if storedTheme != "" {
		defaultTheme = storedTheme
	}
	timeout := flag.Duration("timeout", sonos.DefaultTimeout, "network timeout")
	service := flag.String("service", "Spotify", "SMAPI music service for search")
	category := flag.String("category", "tracks", "SMAPI search category (tracks or playlists; switch in the TUI with ctrl+t/ctrl+p)")
	limit := flag.Int("limit", 10, "SMAPI search result limit")
	theme := flag.String("theme", defaultTheme, "visual theme (aurora, sunset, electric, midnight; cycle in the TUI with ctrl+v)")
	macHelperPath := flag.String("mac-helper-path", "", "path to sonosh-macos-helper (macOS only; defaults to SONOSH_MAC_HELPER or executable sibling)")
	flag.Parse()

	cfg := tui.Config{
		Timeout:          normalizeTimeout(*timeout),
		SearchService:    *service,
		SearchCategory:   *category,
		SearchLimit:      *limit,
		Theme:            *theme,
		ThemeConfigPath:  themeConfigPath,
		Compact:          storedCompact,
		LayoutConfigPath: layoutConfigPath,
		CarouselPath:     carouselConfigPath,
		MacHelperPath:    *macHelperPath,
	}
	model := tui.NewModel(tui.NewSonosBackend(cfg.Timeout), cfg)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if m, ok := finalModel.(tui.Model); ok {
		_ = m.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sonosh: %v\n", err)
		os.Exit(1)
	}
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return sonos.DefaultTimeout
	}
	return timeout
}
