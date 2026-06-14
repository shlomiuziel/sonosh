package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type themeConfigFile struct {
	Version int    `json:"version"`
	Theme   string `json:"theme"`
}

func DefaultThemeConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "theme.json"), nil
}

func LoadThemeName(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var ff themeConfigFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return "", fmt.Errorf("parse theme config: %w", err)
	}
	name := strings.TrimSpace(ff.Theme)
	if themeIndex(name) < 0 {
		return "", nil
	}
	return strings.ToLower(name), nil
}

func SaveThemeName(path, theme string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	theme = strings.ToLower(strings.TrimSpace(theme))
	if themeIndex(theme) < 0 {
		return fmt.Errorf("invalid theme %q", theme)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	ff := themeConfigFile{Version: 1, Theme: theme}
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
