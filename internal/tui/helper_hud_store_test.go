package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadHelperHUDConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper_hud.json")
	want := HelperHUDConfig{Enabled: false, Position: "top-right"}

	if err := SaveHelperHUDConfig(path, want); err != nil {
		t.Fatalf("SaveHelperHUDConfig: %v", err)
	}

	got, err := LoadHelperHUDConfig(path)
	if err != nil {
		t.Fatalf("LoadHelperHUDConfig: %v", err)
	}
	if got != want {
		t.Fatalf("config = %+v, want %+v", got, want)
	}
}

func TestLoadHelperHUDConfigDefaultsForMissingFile(t *testing.T) {
	got, err := LoadHelperHUDConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadHelperHUDConfig: %v", err)
	}
	want := DefaultHelperHUDConfig()
	if got != want {
		t.Fatalf("config = %+v, want %+v", got, want)
	}
}

func TestLoadHelperHUDConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper_hud.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := LoadHelperHUDConfig(path); err == nil {
		t.Fatal("LoadHelperHUDConfig: expected error")
	}
}

func TestLoadHelperHUDConfigNormalizesUnknownPosition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper_hud.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"enabled":true,"position":"weird"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := LoadHelperHUDConfig(path)
	if err != nil {
		t.Fatalf("LoadHelperHUDConfig: %v", err)
	}
	if got.Position != defaultHelperHUDPosition {
		t.Fatalf("position = %q, want %q", got.Position, defaultHelperHUDPosition)
	}
}
