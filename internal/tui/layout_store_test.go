package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLayoutStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "layout.json")

	if err := SaveCompactLayout(path, true); err != nil {
		t.Fatalf("SaveCompactLayout: %v", err)
	}

	got, err := LoadCompactLayout(path)
	if err != nil {
		t.Fatalf("LoadCompactLayout: %v", err)
	}
	if !got {
		t.Fatal("loaded compact = false, want true")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected layout config to be written")
	}
}

func TestLoadCompactLayoutMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	got, err := LoadCompactLayout(path)
	if err != nil {
		t.Fatalf("LoadCompactLayout: %v", err)
	}
	if got {
		t.Fatal("loaded compact = true, want false")
	}
}
