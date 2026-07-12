package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type lastRoomSelection struct {
	Version int    `json:"version"`
	IP      string `json:"ip,omitempty"`
	Name    string `json:"name,omitempty"`
}

func DefaultLastRoomConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "last_room.json"), nil
}

func LoadLastRoomSelection(path string) (lastRoomSelection, error) {
	if path = strings.TrimSpace(path); path == "" {
		return normalizeLastRoomSelection(lastRoomSelection{}), nil
	}
	raw, err := os.ReadFile(path) //nolint:gosec // Config path resolves under the user config directory.
	if errors.Is(err, os.ErrNotExist) {
		return normalizeLastRoomSelection(lastRoomSelection{}), nil
	}
	if err != nil {
		return lastRoomSelection{}, fmt.Errorf("read last room selection: %w", err)
	}
	var stored lastRoomSelection
	if err := json.Unmarshal(raw, &stored); err != nil {
		return lastRoomSelection{}, fmt.Errorf("parse last room selection: %w", err)
	}
	return normalizeLastRoomSelection(stored), nil
}

func SaveLastRoomSelection(path string, selection lastRoomSelection) error {
	if path = strings.TrimSpace(path); path == "" {
		return nil
	}
	selection = normalizeLastRoomSelection(selection)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir last room config dir: %w", err)
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal last room selection: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write last room selection temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename last room selection temp file: %w", err)
	}
	return nil
}

func normalizeLastRoomSelection(selection lastRoomSelection) lastRoomSelection {
	selection.Version = 1
	selection.IP = strings.TrimSpace(selection.IP)
	selection.Name = strings.TrimSpace(selection.Name)
	return selection
}
