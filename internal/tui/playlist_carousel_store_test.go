package tui

import (
	"path/filepath"
	"testing"
)

func TestPlaylistCarouselStoreDefaultsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "playlist_carousel.json")

	store, err := LoadPlaylistCarouselStore(path)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore missing: %v", err)
	}
	if len(store.Pins) != 1 || store.Pins[0].Title != "Release Radar" {
		t.Fatalf("default pins = %#v", store.Pins)
	}
	if store.DefaultPinsSeeded {
		t.Fatalf("default pins should not be marked seeded: %#v", store)
	}

	store.Pins[0].ID = "spotify:playlist:release-radar"
	store.Recent = []playlistCarouselStoreItem{{ID: "spotify:playlist:recent", Title: "Recent One", ItemType: "playlist"}}
	store.DefaultPinsSeeded = true
	if err := SavePlaylistCarouselStore(path, store); err != nil {
		t.Fatalf("SavePlaylistCarouselStore: %v", err)
	}
	loaded, err := LoadPlaylistCarouselStore(path)
	if err != nil {
		t.Fatalf("LoadPlaylistCarouselStore saved: %v", err)
	}
	if loaded.Pins[0].ID != "spotify:playlist:release-radar" {
		t.Fatalf("loaded pin ID = %q", loaded.Pins[0].ID)
	}
	if len(loaded.Recent) != 1 || loaded.Recent[0].Title != "Recent One" {
		t.Fatalf("loaded recent = %#v", loaded.Recent)
	}
	if !loaded.DefaultPinsSeeded {
		t.Fatalf("loaded defaultPinsSeeded = %#v", loaded)
	}
}
