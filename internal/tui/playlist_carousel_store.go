package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shlomiuziel/sonosh/internal/sonos"
)

const (
	playlistCarouselStoreVersion = 1
	maxRecentPlaylists           = 12
)

type playlistCarouselStore struct {
	Pins              []playlistCarouselStoreItem
	Recent            []playlistCarouselStoreItem
	DefaultPinsSeeded bool
}

type playlistCarouselConfigFile struct {
	Version           int                         `json:"version"`
	Pins              []playlistCarouselStoreItem `json:"pins,omitempty"`
	Recent            []playlistCarouselStoreItem `json:"recent,omitempty"`
	DefaultPinsSeeded bool                        `json:"defaultPinsSeeded,omitempty"`
}

type playlistCarouselStoreItem struct {
	ID          string `json:"id,omitempty"`
	ItemType    string `json:"itemType,omitempty"`
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Creator     string `json:"creator,omitempty"`
	ArtworkURI  string `json:"artworkURI,omitempty"`
	AlbumArtURI string `json:"albumArtURI,omitempty"`
}

func DefaultPlaylistCarouselConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "playlist_carousel.json"), nil
}

func LoadPlaylistCarouselStore(path string) (playlistCarouselStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultPlaylistCarouselStore(), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultPlaylistCarouselStore(), nil
		}
		return playlistCarouselStore{}, err
	}
	var ff playlistCarouselConfigFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return playlistCarouselStore{}, fmt.Errorf("parse playlist carousel config: %w", err)
	}
	return normalizePlaylistCarouselStore(playlistCarouselStore{
		Pins:              ff.Pins,
		Recent:            ff.Recent,
		DefaultPinsSeeded: ff.DefaultPinsSeeded,
	}), nil
}

func SavePlaylistCarouselStore(path string, store playlistCarouselStore) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	store = normalizePlaylistCarouselStore(store)
	ff := playlistCarouselConfigFile{
		Version:           playlistCarouselStoreVersion,
		Pins:              store.Pins,
		Recent:            store.Recent,
		DefaultPinsSeeded: store.DefaultPinsSeeded,
	}
	data, err := json.MarshalIndent(ff, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func defaultPlaylistCarouselStore() playlistCarouselStore {
	return playlistCarouselStore{
		Pins: []playlistCarouselStoreItem{
			{Title: "Release Radar", ItemType: "playlist"},
		},
	}
}

func normalizePlaylistCarouselStore(store playlistCarouselStore) playlistCarouselStore {
	store.Pins = normalizePlaylistCarouselItems(store.Pins, 0)
	store.Recent = normalizePlaylistCarouselItems(store.Recent, maxRecentPlaylists)
	return store
}

func normalizePlaylistCarouselItems(items []playlistCarouselStoreItem, limit int) []playlistCarouselStoreItem {
	out := make([]playlistCarouselStoreItem, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item = normalizePlaylistCarouselItem(item)
		if item.Title == "" && item.ID == "" {
			continue
		}
		key := playlistCarouselStoreKey(item)
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

func normalizePlaylistCarouselItem(item playlistCarouselStoreItem) playlistCarouselStoreItem {
	item.ID = strings.TrimSpace(item.ID)
	item.ItemType = strings.TrimSpace(item.ItemType)
	if item.ItemType == "" {
		item.ItemType = "playlist"
	}
	item.Title = strings.TrimSpace(item.Title)
	item.Summary = strings.TrimSpace(item.Summary)
	item.MimeType = strings.TrimSpace(item.MimeType)
	item.Creator = strings.TrimSpace(item.Creator)
	item.ArtworkURI = strings.TrimSpace(item.ArtworkURI)
	item.AlbumArtURI = strings.TrimSpace(item.AlbumArtURI)
	return item
}

func playlistCarouselStoreKey(item playlistCarouselStoreItem) string {
	if id := strings.ToLower(strings.TrimSpace(item.ID)); id != "" {
		return "id:" + id
	}
	if title := strings.ToLower(strings.TrimSpace(item.Title)); title != "" {
		return "title:" + title
	}
	return ""
}

func playlistCarouselResultFromStoreItem(item playlistCarouselStoreItem) SearchResult {
	item = normalizePlaylistCarouselItem(item)
	return SearchResult{Item: sonos.SMAPIItem{
		ID:          item.ID,
		ItemType:    item.ItemType,
		Title:       item.Title,
		Summary:     item.Summary,
		MimeType:    item.MimeType,
		Creator:     item.Creator,
		ArtworkURI:  item.ArtworkURI,
		AlbumArtURI: item.AlbumArtURI,
	}}
}

func playlistCarouselStoreItemFromResult(result SearchResult) playlistCarouselStoreItem {
	if isSpotifyLikedSongsResult(result) {
		result.Item.Title = "Liked Songs"
		if strings.TrimSpace(result.Item.Creator) == "" {
			result.Item.Creator = "Spotify"
		}
	}
	return normalizePlaylistCarouselItem(playlistCarouselStoreItem{
		ID:          result.Item.ID,
		ItemType:    result.Item.ItemType,
		Title:       result.Title(),
		Summary:     result.Item.Summary,
		MimeType:    result.Item.MimeType,
		Creator:     result.Item.Creator,
		ArtworkURI:  result.Item.ArtworkURI,
		AlbumArtURI: result.Item.AlbumArtURI,
	})
}

func playlistCarouselPinnedResults(store playlistCarouselStore) []SearchResult {
	out := make([]SearchResult, 0, len(store.Pins))
	for _, item := range normalizePlaylistCarouselStore(store).Pins {
		out = append(out, playlistCarouselResultFromStoreItem(item))
	}
	return out
}

func playlistCarouselRecentResults(store playlistCarouselStore) []SearchResult {
	out := make([]SearchResult, 0, len(store.Recent))
	for _, item := range normalizePlaylistCarouselStore(store).Recent {
		out = append(out, playlistCarouselResultFromStoreItem(item))
	}
	return out
}

func addRecentPlaylistToStore(store playlistCarouselStore, result SearchResult) playlistCarouselStore {
	item := playlistCarouselStoreItemFromResult(result)
	if item.Title == "" && item.ID == "" {
		return normalizePlaylistCarouselStore(store)
	}
	recent := []playlistCarouselStoreItem{item}
	key := playlistCarouselStoreKey(item)
	for _, existing := range store.Recent {
		existing = normalizePlaylistCarouselItem(existing)
		if playlistCarouselStoreKey(existing) == key {
			continue
		}
		recent = append(recent, existing)
	}
	store.Recent = recent
	return normalizePlaylistCarouselStore(store)
}
