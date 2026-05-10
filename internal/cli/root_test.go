package cli

import (
	"testing"
	"time"

	"github.com/steipete/sonoscli/internal/appconfig"
	"github.com/steipete/sonoscli/internal/sonos"
)

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
