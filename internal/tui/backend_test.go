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

func TestScrubTargetClampsAndOffsets(t *testing.T) {
	tests := []struct {
		name     string
		position string
		duration string
		delta    int
		want     string
	}{
		{name: "forward within duration", position: "0:01:10", duration: "0:03:00", delta: 5, want: "0:01:15"},
		{name: "backward clamps at zero", position: "0:00:03", duration: "0:03:00", delta: -5, want: "0:00:00"},
		{name: "forward clamps at duration", position: "0:02:58", duration: "0:03:00", delta: 5, want: "0:03:00"},
		{name: "invalid duration still seeks", position: "0:00:10", duration: "bad", delta: 5, want: "0:00:15"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scrubTarget(tt.position, tt.duration, tt.delta)
			if err != nil {
				t.Fatalf("scrubTarget returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("scrubTarget(%q, %q, %d) = %q, want %q", tt.position, tt.duration, tt.delta, got, tt.want)
			}
		})
	}
}
