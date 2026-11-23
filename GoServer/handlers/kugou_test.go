package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"broadcast-tool/converter"
	"broadcast-tool/internal/kugoudb"
	"broadcast-tool/models"
	"broadcast-tool/paths"

	"unlock-music.dev/cli/algo/qmc"
)

func TestProcessKggWithLocalDatabase(t *testing.T) {
	dir := t.TempDir()
	const hash = "synthetic-audio-id"
	// EKey for a synthetic test key, independent of any music service account.
	const ekey = "a3Vnb3VrZXls9jNASxMyyh/jAgTdqWIH"
	source := filepath.Join(t.TempDir(), kugoudb.FileName)
	sqlite, err := sql.Open("sqlite", source)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.Close()
	if _, err := sqlite.Exec("CREATE TABLE DownloadItem(EnHash TEXT, EnKey TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlite.Exec("INSERT INTO DownloadItem VALUES (?, ?)", hash, ekey); err != nil {
		t.Fatal(err)
	}
	if err := sqlite.Close(); err != nil {
		t.Fatal(err)
	}
	db := kugoudb.New(dir)
	if err := db.Select(context.Background(), source); err != nil {
		t.Fatal(err)
	}

	const audioOffset = 1024
	audio := []byte("ID3\x04\x00\x00\x00\x00\x00\x00synthetic MP3 fixture")
	encrypted := make([]byte, audioOffset+len(audio))
	copy(encrypted[:16], []byte{0x7c, 0xd5, 0x32, 0xeb, 0x86, 0x02, 0x7f, 0x4b, 0xa8, 0xaf, 0xa6, 0x8e, 0x0f, 0xff, 0x99, 0x14})
	binary.LittleEndian.PutUint32(encrypted[16:20], audioOffset)
	binary.LittleEndian.PutUint32(encrypted[20:24], 5)
	binary.LittleEndian.PutUint32(encrypted[68:72], uint32(len(hash)))
	copy(encrypted[72:], hash)
	copy(encrypted[audioOffset:], audio)
	cipher, err := qmc.NewQmcCipherDecoderFromEKey([]byte(ekey))
	if err != nil {
		t.Fatal(err)
	}
	cipher.Decrypt(encrypted[audioOffset:], 0)

	if err := os.MkdirAll(paths.GetTempDir(dir), 0700); err != nil {
		t.Fatal(err)
	}
	conv := converter.New("", "")
	conv.KugouKey = db.Key
	h := NewFileHandler(dir, conv, nil)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "SYNTHETIC.KGG")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(encrypted); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/files/process", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	w := httptest.NewRecorder()
	h.HandleProcess(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %d %s", w.Code, w.Body.String())
	}
	var result models.FileProcessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	decoded, err := os.ReadFile(filepath.Join(paths.GetSongStagingDir(dir), result.TempFileName))
	if err != nil || !bytes.Equal(audio, decoded) {
		t.Fatalf("decoded bytes differ: %v", err)
	}
}
