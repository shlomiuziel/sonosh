package macoshelper

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	pos := 75.0
	dur := 200.0
	vol := 42
	muted := false
	in := Message{
		Type:            "nowPlaying",
		Room:            "Kitchen",
		State:           "playing",
		Title:           "Track",
		Artist:          "Artist",
		Album:           "Album",
		AlbumArtURL:     "http://example.test/art.jpg",
		PositionSeconds: &pos,
		DurationSeconds: &dur,
		Volume:          &vol,
		Muted:           &muted,
	}

	var buf bytes.Buffer
	if err := Encode(&buf, in); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("encoded message is not newline-delimited: %q", buf.String())
	}

	var out Message
	if err := Decode(&buf, &out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.Type != in.Type || out.Room != in.Room || out.Title != in.Title {
		t.Fatalf("decoded message mismatch: %#v", out)
	}
	if out.PositionSeconds == nil || *out.PositionSeconds != pos {
		t.Fatalf("PositionSeconds = %#v, want %v", out.PositionSeconds, pos)
	}
	if out.Volume == nil || *out.Volume != vol {
		t.Fatalf("Volume = %#v, want %v", out.Volume, vol)
	}
}
