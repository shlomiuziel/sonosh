package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_SaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	cfg := Config{DefaultRoom: " Office ", DefaultTimeout: "15000ms", Format: "JSON"}
	if err := s.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DefaultRoom != "Office" {
		t.Fatalf("defaultRoom: %q", got.DefaultRoom)
	}
	if got.Format != "json" {
		t.Fatalf("format: %q", got.Format)
	}
	if got.DefaultTimeout != "15s" {
		t.Fatalf("defaultTimeout: %q", got.DefaultTimeout)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("expected perms 0600, got %o", fi.Mode().Perm())
	}
}

func TestConfigNormalize_InvalidDefaultTimeout(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"nope", "0s", "-1s"} {
		cfg := Config{DefaultTimeout: value}
		got := cfg.Normalize()
		if got.DefaultTimeout != "" {
			t.Fatalf("DefaultTimeout for %q = %q, want empty", value, got.DefaultTimeout)
		}
	}
}

func TestConfigNormalize_InvalidFormat(t *testing.T) {
	t.Parallel()

	cfg := Config{Format: "nope"}
	got := cfg.Normalize()
	if got.Format != "plain" {
		t.Fatalf("expected plain fallback, got %q", got.Format)
	}
}
