//go:build darwin

package macoshelper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHelperPathFindsExplicitPath(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "sonosh-macos-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	got, err := resolveHelperPath(helper)
	if err != nil {
		t.Fatalf("resolveHelperPath: %v", err)
	}
	if got != helper {
		t.Fatalf("helper path = %q, want %q", got, helper)
	}
}

func TestResolveHelperPathReportsMissingExplicitPath(t *testing.T) {
	_, err := resolveHelperPath(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected missing helper error")
	}
}
