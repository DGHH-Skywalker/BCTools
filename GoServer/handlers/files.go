package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"broadcast-tool/converter"
	"broadcast-tool/models"
	"broadcast-tool/paths"
	"broadcast-tool/response"
	"broadcast-tool/services"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/validation"

	"github.com/google/uuid"
)

type FileHandler struct {
	AppDataDir      string
	Settings        *settingstore.SettingsStore
	Converter       *converter.FFMpegConverter
	organizeService *services.OrganizeService
}

func NewFileHandler(appDataDir string, settings *settingstore.SettingsStore, c *converter.FFMpegConverter) *FileHandler {
	h := &FileHandler{AppDataDir: appDataDir, Settings: settings, Converter: c}
	h.organizeService = services.NewOrganizeService(
		appDataDir,
		settings,
		c,
		h.validateSourcePath,
		validateTargetName,
		validateOrganizeTargetDir,
	)
	return h
}

func (h *FileHandler) HandleProcess(w http.ResponseWriter, r *http.Request) {
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

	id := uuid.New().String()
	tempDir := paths.GetTempDir(h.AppDataDir)
	songsDir := paths.GetSongsDir(h.AppDataDir)
	if err := paths.EnsureDir(songsDir); err != nil {
		response.WriteInternalError(w, "创建歌曲目录失败")
		return
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	inputExt := ".tmp"
	if converter.IsEncryptedFormat(ext) {
		inputExt = ext
	}
	inputPath := filepath.Join(tempDir, id+inputExt)
	out, err := os.Create(inputPath)
	if err != nil {
		response.WriteInternalError(w, "保存文件失败")
		return
	}
	io.Copy(out, file)
	out.Close()

	outputPath := filepath.Join(songsDir, id+".mp3")
	result, err := h.Converter.ProcessFile(inputPath, outputPath)
	if err != nil {
		os.Remove(inputPath)
		os.Remove(outputPath)
		code := classifyConversionError(err)
		response.WriteError(w, http.StatusInternalServerError, code, "音频转换失败")
		return
	}
	os.Remove(inputPath)
	response.WriteJSON(w, http.StatusOK, models.FileProcessResponse{
		TempFileName: id + ".mp3",
		Title:        result.Title,
		Artist:       result.Artist,
	})
}

func (h *FileHandler) HandleStash(w http.ResponseWriter, r *http.Request) {
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

	id := uuid.New().String()
	tempDir := paths.GetTempDir(h.AppDataDir)
	songsDir := paths.GetSongsDir(h.AppDataDir)
	if err := paths.EnsureDir(songsDir); err != nil {
		response.WriteInternalError(w, "创建歌曲目录失败")
		return
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	inputExt := ".mp3"
	if converter.IsEncryptedFormat(ext) {
		inputExt = ext
	}
	inputPath := filepath.Join(tempDir, id+inputExt)
	out, err := os.Create(inputPath)
	if err != nil {
		response.WriteInternalError(w, "保存文件失败")
		return
	}
	io.Copy(out, file)
	out.Close()

	outputPath := filepath.Join(songsDir, id+".mp3")
	result, err := h.Converter.StashFile(inputPath, outputPath)
	if err != nil {
		result = converter.ProbeResult{}
	}
	response.WriteJSON(w, http.StatusOK, models.FileProcessResponse{
		TempFileName: id + ".mp3",
		Title:        result.Title,
		Artist:       result.Artist,
	})
}

func classifyConversionError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unsupported format"):
		return "UNSUPPORTED_ENCRYPTED_FORMAT"
	case strings.Contains(msg, "validate:"):
		return "ENCRYPTED_FILE_INVALID"
	case strings.Contains(msg, "decrypt"):
		return "ENCRYPTION_DECRYPT_FAILED"
	default:
		return "CONVERSION_ERROR"
	}
}

func (h *FileHandler) HandleOrganize(w http.ResponseWriter, r *http.Request) {
	var req models.OrganizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if !validation.IsValidMode(req.Mode) {
		response.WriteValidationError(w, "mode 必须为 copy 或 move")
		return
	}

	targetDir, err := h.organizeService.PreValidate(req)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	existingFiles := h.organizeService.ListExistingFiles(req.Entries, targetDir)
	if len(existingFiles) > 0 && !req.Confirm {
		response.WriteJSON(w, http.StatusOK, models.OrganizeResponse{
			ConfirmNeeded: true,
			ExistingFiles: existingFiles,
		})
		return
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		response.WriteValidationError(w, "无法创建目标目录")
		return
	}

	resp := h.organizeService.Execute(req, targetDir)
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *FileHandler) validateSourcePath(src string) (string, error) {
	appDataDir := h.AppDataDir
	// Bare filenames are treated as songs dir files first, then temp dir (for backward compatibility)
	if src != "" && !strings.ContainsAny(src, `/\`) {
		songsPath := filepath.Join(paths.GetSongsDir(appDataDir), src)
		if _, err := os.Stat(songsPath); err == nil {
			src = songsPath
		} else {
			src = filepath.Join(paths.GetTempDir(appDataDir), src)
		}
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("path traversal")
	}
	allowedRoots := []string{
		appDataDir,
	}
	for _, root := range allowedRoots {
		if strings.HasPrefix(strings.ToLower(clean), strings.ToLower(root)+`\`) || strings.EqualFold(clean, root) {
			return clean, nil
		}
	}
	return "", fmt.Errorf("source not allowed")
}

func validateTargetName(name string) (string, error) {
	base := filepath.Base(name)
	if base == "" || base == "." || strings.Contains(base, "..") {
		return "", fmt.Errorf("invalid target name")
	}
	if strings.ContainsAny(base, `\/:*?"\u003c\u003e|`) {
		return "", fmt.Errorf("invalid characters in target name")
	}
	return base, nil
}

func validateOrganizeTargetDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("targetDir 不能为空")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("无效路径")
	}
	clean := filepath.Clean(abs)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("路径包含非法字符")
	}
	return clean, nil
}

func (h *FileHandler) HandleSelectDir(w http.ResponseWriter, r *http.Request) {
	path, err := selectDirectory("选择文件夹")
	if err != nil {
		response.WriteJSON(w, http.StatusOK, models.SelectDirResponse{Path: ""})
		return
	}
	response.WriteJSON(w, http.StatusOK, models.SelectDirResponse{Path: path})
}

func (h *FileHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		response.WriteValidationError(w, "file 参数不能为空")
		return
	}
	srcPath, err := h.validateSourcePath(filename)
	if err != nil {
		response.WriteValidationError(w, "非法文件路径")
		return
	}
	info, err := os.Stat(srcPath)
	if err != nil || info.IsDir() {
		response.WriteNotFoundError(w, "文件不存在")
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeFile(w, r, srcPath)
}

func (h *FileHandler) HandleBrowse(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		response.WriteValidationError(w, "dir 参数不能为空")
		return
	}
	cleanDir, err := validateBrowsePath(dir)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	entries, err := os.ReadDir(cleanDir)
	if err != nil {
		response.WriteValidationError(w, "无法读取目录")
		return
	}
	var resp models.BrowseResponse
	resp.Path = cleanDir
	const maxEntries = 1000
	const maxNameLen = 256
	for i, e := range entries {
		if i >= maxEntries {
			break
		}
		name := e.Name()
		if len(name) > maxNameLen {
			continue
		}
		// Hide hidden/system files
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if e.IsDir() {
			resp.Dirs = append(resp.Dirs, name)
		} else {
			resp.Files = append(resp.Files, name)
		}
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

// validateBrowsePath normalizes the path and rejects traversal / sensitive roots.
func validateBrowsePath(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("无效路径")
	}
	clean := filepath.Clean(abs)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("路径包含非法字符")
	}
	// Disallow sensitive system directories
	sensitive := []string{
		`C:\Windows`, `C:\Program Files`, `C:\Program Files (x86)`,
		`C:\ProgramData`, `C:\$Recycle.Bin`, `C:\Users\Default`,
	}
	for _, s := range sensitive {
		if strings.EqualFold(clean, s) || strings.HasPrefix(strings.ToLower(clean), strings.ToLower(s)+`\`) {
			return "", fmt.Errorf("不允许访问系统目录")
		}
	}
	return clean, nil
}

func (h *FileHandler) HandleSilent(w http.ResponseWriter, r *http.Request) {
	durStr := r.URL.Query().Get("duration")
	if durStr == "" {
		durStr = "30"
	}
	dur, err := strconv.Atoi(durStr)
	if err != nil || dur <= 0 {
		response.WriteValidationError(w, "静音时长无效")
		return
	}

	tempDir := paths.GetTempDir(h.AppDataDir)
	if err := paths.EnsureDir(tempDir); err != nil {
		response.WriteInternalError(w, "创建临时目录失败")
		return
	}

	tempPath := filepath.Join(tempDir, fmt.Sprintf("silent-%d.mp3", dur))
	if err := h.Converter.GenerateSilentMP3(tempPath, dur); err != nil {
		response.WriteInternalError(w, "生成静音文件失败")
		return
	}
	defer os.Remove(tempPath)

	w.Header().Set("Content-Type", "audio/mpeg")
	http.ServeFile(w, r, tempPath)
}

func (h *FileHandler) HandleDeleteSource(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		response.WriteValidationError(w, "file 参数不能为空")
		return
	}

	srcPath, err := h.validateSourcePath(filename)
	if err != nil {
		response.WriteValidationError(w, "非法文件路径")
		return
	}

	if err := os.Remove(srcPath); err != nil {
		response.WriteInternalError(w, "删除源文件失败")
		return
	}

	response.WriteNoContent(w)
}
