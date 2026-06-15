package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type layoutConfigFile struct {
	Version int  `json:"version"`
	Compact bool `json:"compact"`
}

func DefaultLayoutConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "layout.json"), nil
}

func LoadCompactLayout(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	var ff layoutConfigFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return false, fmt.Errorf("parse layout config: %w", err)
	}
	return ff.Compact, nil
}

func SaveCompactLayout(path string, compact bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	ff := layoutConfigFile{Version: 1, Compact: compact}
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
