package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"broadcast-tool/paths"
	"broadcast-tool/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// DecryptHandler stages decrypted audio files produced by the um-react tool so
// they can be imported into the main app from any device on the LAN. The um-react
// tab and the main app may run on different devices (e.g. a phone decrypting and
// a PC importing), so cross-page communication goes through the backend instead
// of window.opener/postMessage, which only works within a single browser on one
// device.
type DecryptHandler struct {
	AppDataDir string
}

func NewDecryptHandler(appDataDir string) *DecryptHandler {
	return &DecryptHandler{AppDataDir: appDataDir}
}

type stageResponse struct {
	StageID  string `json:"stageId"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type stageMetaResponse struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Ext      string `json:"ext"`
	Imported bool   `json:"imported"`
}

// HandleStage receives a decrypted audio file and stores it temporarily so the
// main app can fetch and import it. Returns a stageId the main app uses to
// retrieve the file. A sidecar ".meta" file records the original filename.
func (h *DecryptHandler) HandleStage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 512<<20)
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		response.WriteValidationError(w, "文件过大")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteValidationError(w, "请上传文件")
		return
	}
	defer file.Close()

	stageID := uuid.New().String()
	stagingDir := paths.GetDecryptStagingDir(h.AppDataDir)
	if err := paths.EnsureDir(stagingDir); err != nil {
		response.WriteInternalError(w, "创建暂存目录失败")
		return
	}
	// Drop abandoned stages from previous (failed/cancelled) imports.
	h.cleanupStale()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	outPath := filepath.Join(stagingDir, stageID+ext)
	out, err := os.Create(outPath)
	if err != nil {
		response.WriteInternalError(w, "保存文件失败")
		return
	}
	written, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(outPath)
		response.WriteInternalError(w, "保存文件失败")
		return
	}

	metaBytes, _ := json.Marshal(map[string]any{
		"filename": header.Filename,
		"size":     written,
		"ext":      ext,
		"imported": false,
	})
	_ = os.WriteFile(filepath.Join(stagingDir, stageID+".meta"), metaBytes, 0644)

	response.WriteJSON(w, http.StatusOK, stageResponse{StageID: stageID, Filename: header.Filename, Size: written})
}

// HandleStageMeta returns metadata for a staged file (used by the import modal).
func (h *DecryptHandler) HandleStageMeta(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if !isValidStageID(stageID) {
		response.WriteValidationError(w, "无效的暂存ID")
		return
	}
	meta := h.readMeta(stageID)
	if meta == nil {
		response.WriteNotFoundError(w, "暂存文件不存在或已过期")
		return
	}
	response.WriteJSON(w, http.StatusOK, stageMetaResponse{
		Filename: getString(meta, "filename"),
		Size:     getInt64(meta, "size"),
		Ext:      getString(meta, "ext"),
		Imported: getBool(meta, "imported"),
	})
}

// HandleStageFile streams the raw staged audio bytes back to the main app.
func (h *DecryptHandler) HandleStageFile(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if !isValidStageID(stageID) {
		response.WriteValidationError(w, "无效的暂存ID")
		return
	}
	meta := h.readMeta(stageID)
	if meta == nil {
		response.WriteNotFoundError(w, "暂存文件不存在或已过期")
		return
	}
	ext := getString(meta, "ext")
	path := filepath.Join(paths.GetDecryptStagingDir(h.AppDataDir), stageID+ext)
	if _, err := os.Stat(path); err != nil {
		response.WriteNotFoundError(w, "暂存文件不存在或已过期")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, getString(meta, "filename")))
	http.ServeFile(w, r, path)
}

// HandleStageDelete removes a staged file (called after a successful import or
// when the user cancels).
func (h *DecryptHandler) HandleStageDelete(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if !isValidStageID(stageID) {
		response.WriteValidationError(w, "无效的暂存ID")
		return
	}
	h.removeStage(stageID)
	response.WriteNoContent(w)
}

// HandleStageImported marks a staged file as successfully imported. The um-react
// tab polls the stage meta; once it sees imported=true it removes the decrypted
// card via its own delete logic and then deletes the stage. This is how the main
// app signals import success back to um-react through the backend without either
// side navigating the other.
func (h *DecryptHandler) HandleStageImported(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "id")
	if !isValidStageID(stageID) {
		response.WriteValidationError(w, "无效的暂存ID")
		return
	}
	meta := h.readMeta(stageID)
	if meta == nil {
		response.WriteNotFoundError(w, "暂存文件不存在或已过期")
		return
	}
	meta["imported"] = true
	metaBytes, _ := json.Marshal(meta)
	_ = os.WriteFile(filepath.Join(paths.GetDecryptStagingDir(h.AppDataDir), stageID+".meta"), metaBytes, 0644)
	response.WriteNoContent(w)
}

func (h *DecryptHandler) readMeta(stageID string) map[string]any {
	metaPath := filepath.Join(paths.GetDecryptStagingDir(h.AppDataDir), stageID+".meta")
	bytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(bytes, &m) != nil {
		return nil
	}
	return m
}

func (h *DecryptHandler) removeStage(stageID string) {
	stagingDir := paths.GetDecryptStagingDir(h.AppDataDir)
	entries, _ := os.ReadDir(stagingDir)
	for _, e := range entries {
		name := e.Name()
		// Remove both "<stageID>.<ext>" and "<stageID>.meta".
		if strings.HasPrefix(name, stageID+".") {
			os.Remove(filepath.Join(stagingDir, name))
		}
	}
}

// cleanupStale removes staging files older than 24h so abandoned imports do not
// accumulate indefinitely.
func (h *DecryptHandler) cleanupStale() {
	dir := paths.GetDecryptStagingDir(h.AppDataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > 24*time.Hour {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// isValidStageID ensures the id is a bare UUID (hex + hyphens) with no path
// separators, so a crafted id can never reach files outside the staging dir.
func isValidStageID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if (c >= 'a' && c <= 'f') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
