package cli

import (
	"context"
	"testing"
	"time"

	"github.com/shlomiuziel/sonosh/internal/appconfig"
	"github.com/shlomiuziel/sonosh/internal/sonos"
)

func TestRootCmdUsesSonoshCommandName(t *testing.T) {
	origLoad := loadAppConfig
	t.Cleanup(func() { loadAppConfig = origLoad })
	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{}, nil
	}

	cmd, _, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}
	if got := cmd.Use; got != "sonosh" {
		t.Fatalf("Use = %q, want sonosh", got)
	}
	if got := cmd.VersionTemplate(); got != "sonosh {{.Version}}\n" {
		t.Fatalf("version template = %q", got)
	}
}

func TestRootCmdWithoutSubcommandRunsTUI(t *testing.T) {
	origLoad := loadAppConfig
	origRunTUI := runTUIApp
	t.Cleanup(func() {
		loadAppConfig = origLoad
		runTUIApp = origRunTUI
	})
	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{}, nil
	}

	called := false
	runTUIApp = func(ctx context.Context, flags *rootFlags) error {
		called = true
		if flags.SearchService != "Spotify" {
			t.Fatalf("SearchService = %q, want Spotify", flags.SearchService)
		}
		if flags.SearchCategory != "tracks" {
			t.Fatalf("SearchCategory = %q, want tracks", flags.SearchCategory)
		}
		if flags.SearchLimit != 10 {
			t.Fatalf("SearchLimit = %d, want 10", flags.SearchLimit)
		}
		return nil
	}

	cmd, _, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}
	cmd.SetArgs(nil)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext: %v", err)
	}
	if !called {
		t.Fatal("root command did not run TUI")
	}
}

func TestRootCmdFindsAuthSMAPIBeginSubcommand(t *testing.T) {
	origLoad := loadAppConfig
	origRunTUI := runTUIApp
	t.Cleanup(func() {
		loadAppConfig = origLoad
		runTUIApp = origRunTUI
	})
	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{}, nil
	}
	runTUIApp = func(ctx context.Context, flags *rootFlags) error {
		t.Fatal("auth smapi begin dispatched to TUI")
		return nil
	}

	cmd, _, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}
	found, _, err := cmd.Find([]string{"auth", "smapi", "begin", "--service", "Spotify"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got := found.CommandPath(); got != "sonosh auth smapi begin" {
		t.Fatalf("command path = %q, want sonosh auth smapi begin", got)
	}
}

func TestRootCmdAppliesConfigDefaultsToFlags(t *testing.T) {
	orig := loadAppConfig
	t.Cleanup(func() { loadAppConfig = orig })

	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{
			DefaultRoom: "Office",
			Format:      "json",
		}, nil
	}

	cmd, flags, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}

	if got := cmd.PersistentFlags().Lookup("name").DefValue; got != "Office" {
		t.Fatalf("name default mismatch: %q", got)
	}
	if got := cmd.PersistentFlags().Lookup("format").DefValue; got != "json" {
		t.Fatalf("format default mismatch: %q", got)
	}
	if got := cmd.PersistentFlags().Lookup("timeout").DefValue; got != sonos.DefaultTimeout.String() {
		t.Fatalf("timeout default mismatch: %q", got)
	}

	if flags.Name != "Office" {
		t.Fatalf("flags.Name mismatch: %q", flags.Name)
	}
	if flags.Format != "json" {
		t.Fatalf("flags.Format mismatch: %q", flags.Format)
	}
	if flags.Timeout != sonos.DefaultTimeout {
		t.Fatalf("flags.Timeout mismatch: %s", flags.Timeout)
	}
}

func TestRootCmdAppliesConfigDefaultTimeout(t *testing.T) {
	orig := loadAppConfig
	t.Cleanup(func() { loadAppConfig = orig })

	loadAppConfig = func() (appconfig.Config, error) {
		return appconfig.Config{DefaultTimeout: "10s"}, nil
	}

	cmd, flags, err := newRootCmd()
	if err != nil {
		t.Fatalf("newRootCmd: %v", err)
	}

	if got := cmd.PersistentFlags().Lookup("timeout").DefValue; got != "10s" {
		t.Fatalf("timeout default mismatch: %q", got)
	}
	if flags.Timeout != 10*time.Second {
		t.Fatalf("flags.Timeout mismatch: %s", flags.Timeout)
	}
}
