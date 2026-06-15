package tui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderAlbumArtThumb(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	thumb, err := renderAlbumArtThumb(buf.Bytes(), 4, 2)
	if err != nil {
		t.Fatalf("render thumbnail: %v", err)
	}
	if !strings.Contains(thumb, "▀") {
		t.Fatalf("thumbnail missing block glyph:\n%s", thumb)
	}
	if lines := strings.Count(thumb, "\n") + 1; lines != 2 {
		t.Fatalf("thumbnail lines = %d, want 2:\n%s", lines, thumb)
	}
}

func TestStatusMessageLoadsAlbumArt(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(srv.Close)

	model := NewModel(&fakeBackend{}, testConfig())
	updated, cmd := model.Update(statusMsg{status: Status{AlbumArt: srv.URL}})
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected album art command")
	}

	updated, _ = model.Update(runCmd(cmd).(albumArtMsg))
	model = updated.(Model)
	if model.artView == "" {
		t.Fatal("expected album art view to load")
	}
	if strings.Contains(model.artView, "\x1b_G") {
		return
	}
	if !strings.Contains(model.artView, "▀") {
		t.Fatalf("expected kitty image or block fallback, got:\n%s", model.artView)
	}
}

func TestRenderAlbumArtKitty(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	view, err := renderKittyAlbumArt(buf.Bytes(), 4, 2)
	if err != nil {
		t.Fatalf("render kitty art: %v", err)
	}
	if !strings.Contains(view, "\x1b_Ga=T") || !strings.Contains(view, "c=4") || !strings.Contains(view, "r=2") {
		t.Fatalf("unexpected kitty sequence:\n%s", view)
	}
}

func TestClearKittyGraphicsSequence(t *testing.T) {
	seq := clearKittyGraphics()
	if !strings.Contains(seq, "\x1b_Ga=d") {
		t.Fatalf("clear sequence missing delete action: %q", seq)
	}
	if !strings.Contains(seq, "q=1") {
		t.Fatalf("clear sequence should suppress OK responses: %q", seq)
	}
}

func TestGhosttyDashboardClearsTerminalGraphics(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "Ghostty")

	model := NewModel(&fakeBackend{}, testConfig())
	model.width = 110
	model.loading = false
	model.rooms = []Room{{Name: "Kitchen", IP: "192.0.2.10"}}
	model.status = Status{Title: "Track", Artist: "Artist"}
	model.artURL = "http://example.test/art.jpg"
	model.artView = "\x1b_Ga=T,C=1,f=100,c=16,r=8;AAAA\x1b\\"

	view := model.View()
	if !strings.HasPrefix(view, clearKittyGraphics()) {
		t.Fatalf("dashboard should clear terminal graphics before rendering album art:\n%s", view)
	}
}

func TestRenderCoverArtUsesProvidedInnerWidth(t *testing.T) {
	art := strings.Repeat("x", albumArtColumns)
	got := renderCoverArt(albumArtColumns, art)
	lines := strings.Split(got, "\n")
	if len(lines) == 0 {
		t.Fatal("expected cover art lines")
	}
	if strings.HasPrefix(lines[1], " ") {
		t.Fatalf("cover art was padded despite exact inner width:\n%s", got)
	}
}
