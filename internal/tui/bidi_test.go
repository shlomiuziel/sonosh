package tui

import "testing"

func TestDisplayTextLeavesLTRTextUnchanged(t *testing.T) {
	got := displayText("Daft Punk", 20)
	if got != "Daft Punk" {
		t.Fatalf("displayText() = %q, want %q", got, "Daft Punk")
	}
}

func TestDisplayTextReordersRTLText(t *testing.T) {
	got := displayText("שלום", 20)
	if got != "םולש" {
		t.Fatalf("displayText() = %q, want %q", got, "םולש")
	}
}

func TestDisplayTextReordersMixedText(t *testing.T) {
	got := displayText("ABC שלום", 20)
	if got != "ABC םולש" {
		t.Fatalf("displayText() = %q, want %q", got, "ABC םולש")
	}
}

func TestDisplayTextTruncatesBeforeReordering(t *testing.T) {
	got := displayText("abcdef", 4)
	if got != "abc…" {
		t.Fatalf("displayText() = %q, want %q", got, "abc…")
	}
}
