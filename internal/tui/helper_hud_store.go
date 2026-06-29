package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type helperHUDConfigFile struct {
	Version int  `json:"version"`
	Enabled bool `json:"enabled"`
}

func DefaultHelperHUDConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "helper_hud.json"), nil
}

func LoadHelperHUDEnabled(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return true, nil
	}
	// #nosec G304 -- path comes from app-controlled config location.
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return true, err
	}
	var ff helperHUDConfigFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return true, fmt.Errorf("parse helper HUD config: %w", err)
	}
	return ff.Enabled, nil
}

func SaveHelperHUDEnabled(path string, enabled bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	ff := helperHUDConfigFile{Version: 1, Enabled: enabled}
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
