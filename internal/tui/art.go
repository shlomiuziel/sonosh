package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	albumArtColumns = 16
	albumArtRows    = 8
)

func fetchAlbumArtCmd(url string, kitty bool) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return albumArtMsg{url: url, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return albumArtMsg{url: url, err: fmt.Errorf("album art fetch returned %s", resp.Status)}
		}

		raw, err := imageDataFromReader(resp.Body)
		if err != nil {
			return albumArtMsg{url: url, err: err}
		}
		view, err := renderAlbumArtView(raw, kitty)
		if err != nil {
			return albumArtMsg{url: url, err: err}
		}
		return albumArtMsg{url: url, view: view}
	}
}

func renderAlbumArtView(data []byte, kitty bool) (string, error) {
	if kitty {
		return renderKittyAlbumArt(data, albumArtColumns, albumArtRows)
	}
	return renderAlbumArtBlocks(data, albumArtColumns, albumArtRows)
}

func imageDataFromReader(r io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderKittyAlbumArt(data []byte, cols, rows int) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return "", err
	}

	payload := base64.StdEncoding.EncodeToString(encoded.Bytes())
	return fmt.Sprintf("\x1b_Ga=T,C=1,f=100,c=%d,r=%d;%s\x1b\\", cols, rows, payload), nil
}

func renderAlbumArtBlocks(data []byte, cols, rows int) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return "", fmt.Errorf("empty album art image")
	}

	lines := make([]string, 0, rows)
	for y := 0; y < rows; y++ {
		var cells []string
		for x := 0; x < cols; x++ {
			top := sampleRGBA(img, bounds, x, y*2, cols, rows*2)
			bottom := sampleRGBA(img, bounds, x, y*2+1, cols, rows*2)
			cell := lipgloss.NewStyle().
				Foreground(rgbColor(top)).
				Background(rgbColor(bottom)).
				Render("▀")
			cells = append(cells, cell)
		}
		lines = append(lines, strings.Join(cells, ""))
	}
	return strings.Join(lines, "\n"), nil
}

func renderAlbumArtThumb(data []byte, cols, rows int) (string, error) {
	return renderAlbumArtBlocks(data, cols, rows)
}

func supportsKittyGraphics() bool {
	if strings.EqualFold(os.Getenv("TERM_PROGRAM"), "Ghostty") {
		return true
	}
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "kitty") {
		return true
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	return false
}

func sampleRGBA(img image.Image, bounds image.Rectangle, x, y, width, height int) color.RGBA {
	srcX := bounds.Min.X
	srcY := bounds.Min.Y
	if width > 0 {
		srcX += x * bounds.Dx() / width
	}
	if height > 0 {
		srcY += y * bounds.Dy() / height
	}
	return rgbaAt(img, srcX, srcY)
}

func rgbaAt(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

func rgbColor(c color.RGBA) lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
}
