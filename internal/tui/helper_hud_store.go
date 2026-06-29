package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultHelperHUDPosition = "default"
)

type HelperHUDConfig struct {
	Enabled  bool
	Position string
}

type helperHUDConfigFile struct {
	Version  int    `json:"version"`
	Enabled  bool   `json:"enabled"`
	Position string `json:"position,omitempty"`
}

func DefaultHelperHUDConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sonosh", "helper_hud.json"), nil
}

func LoadHelperHUDConfig(path string) (HelperHUDConfig, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return DefaultHelperHUDConfig(), nil
	}
	// #nosec G304 -- path comes from app-controlled config location.
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultHelperHUDConfig(), nil
		}
		return DefaultHelperHUDConfig(), err
	}
	var ff helperHUDConfigFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return DefaultHelperHUDConfig(), fmt.Errorf("parse helper HUD config: %w", err)
	}
	return HelperHUDConfig{
		Enabled:  ff.Enabled,
		Position: normalizeHelperHUDPosition(ff.Position),
	}, nil
}

func SaveHelperHUDConfig(path string, cfg HelperHUDConfig) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	ff := helperHUDConfigFile{
		Version:  1,
		Enabled:  cfg.Enabled,
		Position: normalizeHelperHUDPosition(cfg.Position),
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

func LoadHelperHUDEnabled(path string) (bool, error) {
	cfg, err := LoadHelperHUDConfig(path)
	return cfg.Enabled, err
}

func SaveHelperHUDEnabled(path string, enabled bool) error {
	cfg, err := LoadHelperHUDConfig(path)
	if err != nil {
		cfg = DefaultHelperHUDConfig()
	}
	cfg.Enabled = enabled
	return SaveHelperHUDConfig(path, cfg)
}

func DefaultHelperHUDConfig() HelperHUDConfig {
	return HelperHUDConfig{
		Enabled:  true,
		Position: defaultHelperHUDPosition,
	}
}

func normalizeHelperHUDPosition(position string) string {
	position = strings.TrimSpace(strings.ToLower(position))
	switch position {
	case "", defaultHelperHUDPosition:
		return defaultHelperHUDPosition
	case "top-left", "top-right", "bottom-left", "bottom-right":
		return position
	default:
		return defaultHelperHUDPosition
	}
}

func nextHelperHUDPosition(position string) string {
	switch normalizeHelperHUDPosition(position) {
	case defaultHelperHUDPosition:
		return "top-left"
	case "top-left":
		return "top-right"
	case "top-right":
		return "bottom-left"
	case "bottom-left":
		return "bottom-right"
	default:
		return defaultHelperHUDPosition
	}
}

func helperHUDPositionLabel(position string) string {
	switch normalizeHelperHUDPosition(position) {
	case "top-left":
		return "Top L"
	case "top-right":
		return "Top R"
	case "bottom-left":
		return "Bottom L"
	case "bottom-right":
		return "Bottom R"
	default:
		return "Default"
	}
}
