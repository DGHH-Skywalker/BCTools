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

	"broadcast-tool/converter"
	"broadcast-tool/models"
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
	Converter  *converter.FFMpegConverter
}

func NewDecryptHandler(appDataDir string, conv *converter.FFMpegConverter) *DecryptHandler {
	return &DecryptHandler{AppDataDir: appDataDir, Converter: conv}
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

// HandleStageImport 把暂存文件就地转入歌库，返回与 /api/files/{process,stash}
// 相同的响应结构。
//
// 为什么需要它：um-react 桥接流程里，解密后的音频已经躺在后端暂存区了。
// 此前主应用还要 GET /stage/{id}/file 把它下载回浏览器，再 POST /files/stash
// 原样传回去——一首 12 MB 的歌在本机 HTTP 上要跑三趟共 35.6 MB，实测多花约
// 410 ms。这里直接在服务端从暂存区转到歌库，省掉那两趟。
func (h *DecryptHandler) HandleStageImport(w http.ResponseWriter, r *http.Request) {
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

	stagingDir := paths.GetDecryptStagingDir(h.AppDataDir)
	ext := getString(meta, "ext")
	srcPath := filepath.Join(stagingDir, stageID+ext)
	if _, err := os.Stat(srcPath); err != nil {
		response.WriteNotFoundError(w, "暂存文件不存在或已过期")
		return
	}

	// 产出先落暂存区，歌曲创建并分配时段后由 MusicLibrary 移入周文件夹
	songsStaging := paths.GetSongStagingDir(h.AppDataDir)
	if err := paths.EnsureDir(songsStaging); err != nil {
		response.WriteInternalError(w, "创建歌曲暂存目录失败")
		return
	}

	id := uuid.New().String()
	outputPath := filepath.Join(songsStaging, id+".mp3")

	// 标题优先取 um-react 交过来的原始文件名——它就是用户在 um-react 里看到的
	// 那一行。这样常见情况下完全不必启动 ffprobe（实测一次 ~88ms，比复制 12MB
	// 还贵 5 倍）。
	base := filepath.Base(getString(meta, "filename"))
	title := strings.TrimSuffix(base, filepath.Ext(base))
	artist := ""

	// 已是 MP3：直接搬运，跳过转码与探测。
	// 否则交给 StashFile 走转码（内部会顺带探测元数据）。
	if strings.EqualFold(ext, ".mp3") {
		if err := moveOrCopy(srcPath, outputPath); err != nil {
			response.WriteInternalError(w, "保存文件失败")
			return
		}
	} else {
		result, err := h.Converter.StashFile(srcPath, outputPath)
		if err != nil {
			response.WriteError(w, http.StatusInternalServerError, "CONVERSION_FAILED", "音频转换失败")
			return
		}
		if result.Title != "" {
			title = result.Title
		}
		artist = result.Artist
	}

	response.WriteJSON(w, http.StatusOK, models.FileProcessResponse{
		TempFileName: id + ".mp3",
		Title:        title,
		Artist:       artist,
	})
}

// moveOrCopy 优先用 rename（同盘几乎零成本），跨盘时退化为复制。
func moveOrCopy(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return converter.CopyFile(src, dst)
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
