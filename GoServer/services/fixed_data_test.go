package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFixedMigrationConflictAndCopy(t *testing.T) {
	a, b, root := t.TempDir(), t.TempDir(), t.TempDir()
	first := []byte(`{"dormSongs":[],"broadcastSongs":[{"title":"本周","date":"2026-09-18"}],"settings":{"timeSlots":[]}}`)
	second := []byte(`{"dormSongs":[],"broadcastSongs":[],"settings":{"timeSlots":[]}}`)
	os.WriteFile(filepath.Join(a, "data.json"), first, 0600)
	os.WriteFile(filepath.Join(b, "data.json"), second, 0600)
	if err := PrepareFixedData(root, []string{a, b}, ""); err == nil {
		t.Fatal("must not guess between databases")
	}
	if _, err := os.Stat(filepath.Join(root, "data.json")); !os.IsNotExist(err) {
		t.Fatal("conflict wrote database")
	}
	os.Mkdir(filepath.Join(a, "songs"), 0700)
	os.WriteFile(filepath.Join(a, "songs", "track.mp3"), []byte("audio"), 0600)
	if err := PrepareFixedData(root, []string{a, b}, a); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "data.json"))
	if !bytes.Equal(got, first) {
		t.Fatal("data changed")
	}
	if _, err := os.Stat(filepath.Join(root, "MusicFiles", "_staging", "track.mp3")); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(a, "data.json"))
	if !bytes.Equal(got, first) {
		t.Fatal("source changed")
	}
	if err := PrepareFixedData(root, []string{b}, ""); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(root, "data.json"))
	if !bytes.Equal(got, first) {
		t.Fatal("existing data replaced")
	}
}

func TestIdenticalSourcesAndInterruptedMigration(t *testing.T) {
	a, b, root := t.TempDir(), t.TempDir(), t.TempDir()
	data := []byte(`{"dormSongs":[],"broadcastSongs":[]}`)
	os.WriteFile(filepath.Join(a, "data.json"), data, 0600)
	os.WriteFile(filepath.Join(b, "data.json"), data, 0600)
	os.WriteFile(filepath.Join(root, "interrupted-copy"), []byte("keep"), 0600)
	if err := PrepareFixedData(root, []string{a, b}, ""); err != nil {
		t.Fatal(err)
	}
}
