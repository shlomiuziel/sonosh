package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestThemeStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.json")

	if err := SaveThemeName(path, "sunset"); err != nil {
		t.Fatalf("SaveThemeName: %v", err)
	}

	got, err := LoadThemeName(path)
	if err != nil {
		t.Fatalf("LoadThemeName: %v", err)
	}
	if got != "sunset" {
		t.Fatalf("loaded theme = %q, want sunset", got)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected theme config to be written")
	}
}

func TestLoadThemeNameMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	got, err := LoadThemeName(path)
	if err != nil {
		t.Fatalf("LoadThemeName: %v", err)
	}
	if got != "" {
		t.Fatalf("loaded theme = %q, want empty", got)
	}
}
