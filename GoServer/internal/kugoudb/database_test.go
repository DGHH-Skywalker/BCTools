package kugoudb

import (
	"bytes"
	"context"
	"crypto/cipher"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"unlock-music.dev/cli/algo/kgm/pc_kugou_db"
)

func syntheticDatabase(t *testing.T, path, key string, encrypted bool) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		"PRAGMA page_size=1024",
		"CREATE TABLE ShareFileItems(EncryptionKeyId TEXT, EncryptionKey TEXT)",
		"CREATE TABLE DownloadItem(EnHash TEXT, EnKey TEXT)",
		"INSERT INTO ShareFileItems VALUES ('shared-hash', 'shared-key')",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO DownloadItem VALUES ('download-hash', ?)", key); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if !encrypted {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const pageSize = pc_kugou_db.PAGE_SIZE
	header := [8]byte(data[16:24])
	for start := 0; start < len(data); start += pageSize {
		begin := start
		if start == 0 {
			begin += 16
		}
		block, iv := pageCipher(uint32(start/pageSize + 1))
		page := data[begin : start+pageSize]
		cipher.NewCBCEncrypter(block, iv[:]).CryptBlocks(page, page)
	}
	copy(data[8:16], data[16:24])
	copy(data[16:24], header[:])
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCopiesRefreshesAndRetainsLastUsableDatabase(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	source := filepath.Join(t.TempDir(), FileName)
	syntheticDatabase(t, source, "key-one", true)

	// Verify the encrypted fixture with the upstream parser independently.
	original, err := pc_kugou_db.CachedDumpEKey(source)
	if err != nil || original["shared-hash"] != "shared-key" {
		t.Fatalf("upstream fixture: %v", err)
	}

	db := New(dataDir)
	db.discover = func() string { return source }
	if err := db.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if custom, count := db.Status(); custom || count != 2 || db.Key("download-hash") != "key-one" {
		t.Fatalf("unexpected automatic source status: custom=%v count=%d", custom, count)
	}
	rawSource, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.ReadFile(db.snapshot(db.state.Snapshot))
	if err != nil || !bytes.Equal(rawSource, snapshot) {
		t.Fatalf("snapshot differs from the original database: %v", err)
	}

	selected := filepath.Join(t.TempDir(), "selected.db")
	syntheticDatabase(t, selected, "key-two", false)
	if err := db.Select(ctx, selected); err != nil {
		t.Fatal(err)
	}
	if custom, _ := db.Status(); !custom || db.Key("download-hash") != "key-two" {
		t.Fatal("the custom source did not take effect")
	}
	before := db.state.Snapshot
	if err := os.Remove(selected); err != nil {
		t.Fatal(err)
	}
	syntheticDatabase(t, selected, "key-three", true)
	nextTime := db.state.Modified.Add(time.Second)
	if err := os.Chtimes(selected, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}

	restarted := New(dataDir)
	if err := restarted.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if restarted.Key("download-hash") != "key-three" || restarted.state.Snapshot == before {
		t.Fatal("the selected database was not refreshed on restart")
	}
	if _, err := os.Stat(db.snapshot(before)); !os.IsNotExist(err) {
		t.Fatalf("obsolete snapshot remains: %v", err)
	}

	invalid := filepath.Join(t.TempDir(), "invalid.db")
	if err := os.WriteFile(invalid, make([]byte, 1024), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Select(ctx, invalid); err == nil || restarted.Key("download-hash") != "key-three" {
		t.Fatal("invalid selection replaced the working snapshot")
	}
	if err := os.Remove(selected); err != nil {
		t.Fatal(err)
	}
	missing := New(dataDir)
	if err := missing.Initialize(ctx); err != nil {
		t.Fatalf("missing sources should be silent: %v", err)
	}
	if missing.Key("download-hash") != "key-three" {
		t.Fatal("missing source should preserve the copied database")
	}
}

func TestMissingDefaultDatabaseCreatesNoFiles(t *testing.T) {
	dir := t.TempDir()
	db := New(dir)
	db.discover = func() string { return "" }
	if err := db.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(db.dir); !os.IsNotExist(err) {
		t.Fatalf("unexpected data directory: %v", err)
	}
}
