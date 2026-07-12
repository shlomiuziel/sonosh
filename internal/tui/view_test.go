package tui

import (
	"strings"
	"testing"
)

func TestRenderSearchFieldShowsCursorBeforePlaceholder(t *testing.T) {
	field := renderSearchField("type to search...", 40, true, true)

	cursorAt := strings.Index(field, "█")
	placeholderAt := strings.Index(field, "type to search...")
	if cursorAt < 0 || placeholderAt < 0 {
		t.Fatalf("search field missing cursor or placeholder:\n%s", field)
	}
	if cursorAt >= placeholderAt {
		t.Fatalf("cursor should appear before placeholder text:\n%s", field)
	}
	if strings.Contains(field, "> type to search...") {
		t.Fatalf("search field should not prepend a prompt to the placeholder:\n%s", field)
	}
}
