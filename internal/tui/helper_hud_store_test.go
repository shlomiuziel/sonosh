package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadHelperHUDEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helper_hud.json")

	if err := SaveHelperHUDEnabled(path, false); err != nil {
		t.Fatalf("SaveHelperHUDEnabled: %v", err)
	}

	got, err := LoadHelperHUDEnabled(path)
	if err != nil {
		t.Fatalf("LoadHelperHUDEnabled: %v", err)
	}
	if got {
		t.Fatalf("LoadHelperHUDEnabled = %v, want false", got)
	}
}

func TestLoadHelperHUDEnabledDefaultsTrueForMissingFile(t *testing.T) {
	got, err := LoadHelperHUDEnabled(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadHelperHUDEnabled: %v", err)
	}
	if !got {
		t.Fatalf("LoadHelperHUDEnabled = %v, want true", got)
	}
}

func TestLoadHelperHUDEnabledInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helper_hud.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := LoadHelperHUDEnabled(path); err == nil {
		t.Fatal("expected parse error")
	}
}
