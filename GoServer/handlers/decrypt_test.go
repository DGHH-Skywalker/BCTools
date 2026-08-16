package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"broadcast-tool/converter"
	"broadcast-tool/paths"

	"github.com/go-chi/chi/v5"
)

// newStagedFile 在暂存区放一个文件并写好 sidecar meta，返回 stageID。
func newStagedFile(t *testing.T, appDataDir, stageID, filename, ext string, content []byte) {
	t.Helper()
	dir := paths.GetDecryptStagingDir(appDataDir)
	if err := paths.EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stageID+ext), content, 0644); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{
		"filename": filename,
		"size":     len(content),
		"ext":      ext,
		"imported": false,
	})
	if err := os.WriteFile(filepath.Join(dir, stageID+".meta"), meta, 0644); err != nil {
		t.Fatal(err)
	}
}

func doStageImport(h *DecryptHandler, stageID string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Post("/api/decrypt/stage/{id}/import", h.HandleStageImport)
	req := httptest.NewRequest(http.MethodPost, "/api/decrypt/stage/"+stageID+"/import", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 已是 MP3 的暂存文件必须就地搬进歌库，且不调用 ffmpeg。
// ffmpeg 路径故意设为不存在：若实现走了转码就会失败，从而证明「没转码」。
func TestStageImportMovesMP3InPlace(t *testing.T) {
	dir := t.TempDir()
	conv := converter.New(filepath.Join(dir, "no-ffmpeg.exe"), "")
	h := NewDecryptHandler(dir, conv)

	content := []byte("ID3\x04\x00\x00\x00\x00\x00\x00real-audio-bytes")
	newStagedFile(t, dir, "aaaaaaaa-1111-2222-3333-444444444444", "红日.mp3", ".mp3", content)

	w := doStageImport(h, "aaaaaaaa-1111-2222-3333-444444444444")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		TempFileName string `json:"tempFileName"`
		Title        string `json:"title"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.TempFileName == "" {
		t.Fatal("tempFileName must be set so the frontend can create the song")
	}
	// 文件必须真的落在 songs 目录，且字节未被改写（没重编码）。
	out := filepath.Join(paths.GetSongsDir(dir), resp.TempFileName)
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output not in songs dir: %v", err)
	}
	if string(got) != string(content) {
		t.Fatal("bytes changed; audio was re-encoded instead of moved")
	}
}

// 无内嵌元数据时，标题退化为原始文件名去扩展名——否则用户会看到空标题。
func TestStageImportFallsBackToFilenameTitle(t *testing.T) {
	dir := t.TempDir()
	conv := converter.New(filepath.Join(dir, "no-ffmpeg.exe"), "")
	h := NewDecryptHandler(dir, conv)

	newStagedFile(t, dir, "bbbbbbbb-1111-2222-3333-444444444444", "李克勤 - 红日.mp3", ".mp3",
		[]byte("ID3\x04\x00\x00\x00\x00\x00\x00no-tags-here"))

	w := doStageImport(h, "bbbbbbbb-1111-2222-3333-444444444444")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Title string `json:"title"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Title != "李克勤 - 红日" {
		t.Fatalf("expected filename-derived title, got %q", resp.Title)
	}
}

func TestStageImportRejectsInvalidID(t *testing.T) {
	dir := t.TempDir()
	h := NewDecryptHandler(dir, converter.New("", ""))
	// 路径穿越形态的 id 必须在触碰文件系统之前就被拒绝。
	w := doStageImport(h, "..%2f..%2fevil")
	if w.Code == http.StatusOK {
		t.Fatalf("a malformed stage id must not succeed, got %d", w.Code)
	}
}

func TestStageImportMissingStageReturns404(t *testing.T) {
	dir := t.TempDir()
	h := NewDecryptHandler(dir, converter.New("", ""))
	w := doStageImport(h, "cccccccc-1111-2222-3333-444444444444")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown stage, got %d", w.Code)
	}
}

// meta 存在但音频文件已被清理时，也必须是 404 而不是 500。
func TestStageImportMetaWithoutFileReturns404(t *testing.T) {
	dir := t.TempDir()
	h := NewDecryptHandler(dir, converter.New("", ""))
	stageDir := paths.GetDecryptStagingDir(dir)
	paths.EnsureDir(stageDir)
	meta, _ := json.Marshal(map[string]any{"filename": "x.mp3", "size": 1, "ext": ".mp3", "imported": false})
	os.WriteFile(filepath.Join(stageDir, "dddddddd-1111-2222-3333-444444444444.meta"), meta, 0644)

	w := doStageImport(h, "dddddddd-1111-2222-3333-444444444444")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when the staged audio is gone, got %d", w.Code)
	}
}
