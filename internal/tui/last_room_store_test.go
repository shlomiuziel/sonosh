package tui

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadLastRoomSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last_room.json")
	want := lastRoomSelection{IP: "192.0.2.11", Name: "Living Room"}
	if err := SaveLastRoomSelection(path, want); err != nil {
		t.Fatalf("SaveLastRoomSelection: %v", err)
	}

	got, err := LoadLastRoomSelection(path)
	if err != nil {
		t.Fatalf("LoadLastRoomSelection: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("version = %d, want 1", got.Version)
	}
	if got.IP != want.IP {
		t.Fatalf("ip = %q, want %q", got.IP, want.IP)
	}
	if got.Name != want.Name {
		t.Fatalf("name = %q, want %q", got.Name, want.Name)
	}
}

func TestLoadLastRoomSelectionMissingFile(t *testing.T) {
	got, err := LoadLastRoomSelection(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadLastRoomSelection: %v", err)
	}
	if got != (lastRoomSelection{Version: 1}) {
		t.Fatalf("got = %#v, want empty normalized selection", got)
	}
}
