package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/appconfig"
)

func TestConfigSetGetUnset(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	// set defaultRoom
	{
		cmd := newConfigSetCmd(flags)
		cmd.SetOut(newDiscardWriter())
		cmd.SetErr(newDiscardWriter())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"defaultRoom", "Office"})
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("set defaultRoom: %v", err)
		}
	}

	// set defaultTimeout through the documented snake_case alias
	{
		cmd := newConfigSetCmd(flags)
		cmd.SetOut(newDiscardWriter())
		cmd.SetErr(newDiscardWriter())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"default_timeout", "15000ms"})
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("set default_timeout: %v", err)
		}
	}

	// get
	{
		cmd := newConfigGetCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("get: %v", err)
		}
		if !strings.Contains(out.String(), "defaultRoom=Office") {
			t.Fatalf("unexpected get output: %s", out.String())
		}
		if !strings.Contains(out.String(), "defaultTimeout=15s") {
			t.Fatalf("unexpected get output: %s", out.String())
		}
	}

	// unset
	{
		cmd := newConfigUnsetCmd(flags)
		cmd.SetOut(newDiscardWriter())
		cmd.SetErr(newDiscardWriter())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"defaultRoom"})
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("unset: %v", err)
		}
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultRoom != "" {
		t.Fatalf("expected defaultRoom to be empty, got %q", cfg.DefaultRoom)
	}
}

func TestConfigSetRejectsInvalidFormat(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	cmd := newConfigSetCmd(flags)
	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"format", "banana"})
	if err := cmd.ExecuteContext(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfigSetRejectsInvalidDefaultTimeout(t *testing.T) {
	flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}

	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	cmd := newConfigSetCmd(flags)
	cmd.SetOut(newDiscardWriter())
	cmd.SetErr(newDiscardWriter())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"defaultTimeout", "nope"})
	if err := cmd.ExecuteContext(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConfigPathPlainAndJSON(t *testing.T) {
	dir := t.TempDir()
	store, err := appconfig.NewFileStore(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	orig := newConfigStore
	t.Cleanup(func() { newConfigStore = orig })
	newConfigStore = func() (appconfig.Store, error) { return store, nil }

	{
		flags := &rootFlags{Timeout: 2 * time.Second, Format: formatPlain}
		cmd := newConfigPathCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("path plain: %v", err)
		}
		if strings.TrimSpace(out.String()) != store.Path() {
			t.Fatalf("unexpected path output: %q", out.String())
		}
	}

	{
		flags := &rootFlags{Timeout: 2 * time.Second, Format: formatJSON}
		cmd := newConfigPathCmd(flags)
		var out captureWriter
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("path json: %v", err)
		}
		if !strings.Contains(out.String(), store.Path()) || !strings.Contains(out.String(), "\"path\"") {
			t.Fatalf("unexpected json output: %q", out.String())
		}
	}
}

func TestGetConfigKey(t *testing.T) {
	cfg := appconfig.Config{DefaultRoom: "Office", DefaultTimeout: "15s", Format: "json"}
	if v, ok := getConfigKey(cfg, "defaultRoom"); !ok || v != "Office" {
		t.Fatalf("defaultRoom: ok=%v v=%q", ok, v)
	}
	if v, ok := getConfigKey(cfg, "default_timeout"); !ok || v != "15s" {
		t.Fatalf("default_timeout: ok=%v v=%q", ok, v)
	}
	if v, ok := getConfigKey(cfg, "format"); !ok || v != "json" {
		t.Fatalf("format: ok=%v v=%q", ok, v)
	}
	if _, ok := getConfigKey(cfg, "nope"); ok {
		t.Fatalf("expected ok=false")
	}
}
