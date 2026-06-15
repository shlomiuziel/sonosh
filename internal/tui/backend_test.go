package tui

import "testing"

func TestSetShufflePlayModePreservesRepeat(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		enabled bool
		want    string
	}{
		{name: "enable from normal", mode: "NORMAL", enabled: true, want: "SHUFFLE_NOREPEAT"},
		{name: "enable from repeat all", mode: "REPEAT_ALL", enabled: true, want: "SHUFFLE"},
		{name: "enable from repeat once", mode: "REPEAT_ONE", enabled: true, want: "SHUFFLE_REPEAT_ONE"},
		{name: "disable from shuffle repeat all", mode: "SHUFFLE", enabled: false, want: "REPEAT_ALL"},
		{name: "disable from shuffle no repeat", mode: "SHUFFLE_NOREPEAT", enabled: false, want: "NORMAL"},
		{name: "disable from shuffle repeat once", mode: "SHUFFLE_REPEAT_ONE", enabled: false, want: "REPEAT_ONE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setShufflePlayMode(tt.mode, tt.enabled); got != tt.want {
				t.Fatalf("setShufflePlayMode(%q, %v) = %q, want %q", tt.mode, tt.enabled, got, tt.want)
			}
		})
	}
}

func TestSetRepeatPlayModePreservesShuffle(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		repeat string
		want   string
	}{
		{name: "shuffle off repeat all", mode: "NORMAL", repeat: "all", want: "REPEAT_ALL"},
		{name: "shuffle on repeat all", mode: "SHUFFLE_NOREPEAT", repeat: "all", want: "SHUFFLE"},
		{name: "shuffle on repeat once", mode: "SHUFFLE_NOREPEAT", repeat: "once", want: "SHUFFLE_REPEAT_ONE"},
		{name: "shuffle on repeat off", mode: "SHUFFLE", repeat: "off", want: "SHUFFLE_NOREPEAT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setRepeatPlayMode(tt.mode, tt.repeat); got != tt.want {
				t.Fatalf("setRepeatPlayMode(%q, %q) = %q, want %q", tt.mode, tt.repeat, got, tt.want)
			}
		})
	}
}

func TestRepeatModeFromPlayModeReadsCombinedModes(t *testing.T) {
	tests := map[string]string{
		"NORMAL":             "off",
		"SHUFFLE_NOREPEAT":   "off",
		"REPEAT_ALL":         "all",
		"SHUFFLE":            "all",
		"REPEAT_ONE":         "once",
		"SHUFFLE_REPEAT_ONE": "once",
	}
	for mode, want := range tests {
		if got := repeatModeFromPlayMode(mode); got != want {
			t.Fatalf("repeatModeFromPlayMode(%q) = %q, want %q", mode, got, want)
		}
	}
}
