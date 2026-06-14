package cli

import "github.com/shlomiuziel/sonosh/internal/sonos"

// Dependency injection points for tests.
var newSMAPITokenStore = func() (sonos.SMAPITokenStore, error) {
	return sonos.NewDefaultSMAPITokenStore()
}
