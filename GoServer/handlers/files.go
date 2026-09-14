package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	"broadcast-tool/validation"

	"github.com/google/uuid"
)

type FileHandler struct {
	AppDataDir      string
	Converter       *converter.FFMpegConverter
	Library         *services.MusicLibrary
	organizeService *services.OrganizeService
}

func NewFileHandler(appDataDir string, c *converter.FFMpegConverter, library *services.MusicLibrary) *FileHandler {
	h := &FileHandler{AppDataDir: appDataDir, Converter: c, Library: library}
	h.organizeService = services.NewOrganizeService(
		appDataDir,
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
	// 产出先落在 MusicFiles/_staging 暂存区；歌曲创建并分配时段后，
	// 由 MusicLibrary 移入对应的周文件夹（2026年第N周/NN.mp3）。
	stagingDir := paths.GetSongStagingDir(h.AppDataDir)
	if err := paths.EnsureDir(stagingDir); err != nil {
		response.WriteInternalError(w, "创建歌曲暂存目录失败")
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

	outputPath := filepath.Join(stagingDir, id+".mp3")
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
	stagingDir := paths.GetSongStagingDir(h.AppDataDir)
	if err := paths.EnsureDir(stagingDir); err != nil {
		response.WriteInternalError(w, "创建歌曲暂存目录失败")
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

	outputPath := filepath.Join(stagingDir, id+".mp3")
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

// HandleMerge 把多个歌库文件按顺序合并成一个 MP3 并直接回传。
//
// 供前端「文件系统访问 API」那条导出路径使用：那条路径由浏览器自己写 SD 卡，
// 拿不到后端的合并结果，所以需要一个能直接取到合并后字节流的端点。
// 后端直连目录的那条路径不走这里（它在服务端内部合并，见 organize_service）。
func (h *FileHandler) HandleMerge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sources []string `json:"sources"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if len(req.Sources) == 0 {
		response.WriteValidationError(w, "sources 不能为空")
		return
	}

	absSources := make([]string, 0, len(req.Sources))
	for _, src := range req.Sources {
		abs, err := h.validateSourcePath(src)
		if err != nil {
			response.WriteValidationError(w, "非法文件路径")
			return
		}
		if _, err := os.Stat(abs); err != nil {
			response.WriteNotFoundError(w, "源文件不存在")
			return
		}
		absSources = append(absSources, abs)
	}

	tempDir := paths.GetTempDir(h.AppDataDir)
	if err := paths.EnsureDir(tempDir); err != nil {
		response.WriteInternalError(w, "创建临时目录失败")
		return
	}
	outPath := filepath.Join(tempDir, "merged-"+uuid.New().String()+".mp3")
	defer os.Remove(outPath)

	if err := converter.MergeMP3s(absSources, outPath); err != nil {
		response.WriteInternalError(w, "合并音频失败")
		return
	}

	info, err := os.Stat(outPath)
	if err != nil {
		response.WriteInternalError(w, "合并音频失败")
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeFile(w, r, outPath)
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

// HandleExportWeek 把某一周的文件夹（MusicFiles\<YYYY年第N周>，同步到最终
// 形态后）整目录复制到桌面。桌面已有同名文件夹时先返回 confirmNeeded。
//
// 这是「导出歌曲文件」页面（原换卡工具）的主入口：不再需要选 SD 卡目录，
// 导出产物就是存储布局本身。
func (h *FileHandler) HandleExportWeek(w http.ResponseWriter, r *http.Request) {
	var req models.ExportWeekRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if req.Year < 1970 || req.Year > 2999 || req.Week < 1 || req.Week > 53 {
		response.WriteValidationError(w, "year/week 参数非法")
		return
	}
	weekName := fmt.Sprintf("%d年第%d周", req.Year, req.Week)
	res, err := h.Library.ExportWeekToDesktop(weekName, req.Confirm)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, models.ExportWeekResponse{
		ConfirmNeeded: res.ConfirmNeeded,
		ExistingFiles: res.ExistingFiles,
		TargetDir:     res.TargetDir,
		FileCount:     res.FileCount,
	})
}

func (h *FileHandler) validateSourcePath(src string) (string, error) {
	appDataDir := h.AppDataDir
	// Bare filenames are treated as song staging files first, then temp dir
	// (for backward compatibility)
	if src != "" && !strings.ContainsAny(src, `/\`) {
		stagingPath := filepath.Join(paths.GetSongStagingDir(appDataDir), src)
		if _, err := os.Stat(stagingPath); err == nil {
			src = stagingPath
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
	if strings.ContainsAny(base, `\/:*?"<>|`) {
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
		// 之前完全静默吞掉错误，导致 Win10 上对话框不显示时无法排查。
		// 现在至少在日志里留痕，前端仍按"用户取消"处理（path=""），
		// 兜底走浏览器 File System Access API。
		log.Printf("selectDirectory 失败: %v", err)
		response.WriteJSON(w, http.StatusOK, models.SelectDirResponse{Path: ""})
		return
	}
	response.WriteJSON(w, http.StatusOK, models.SelectDirResponse{Path: path})
}

func (h *FileHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	// 优先按歌曲 ID 解析：合并文件的成员会抽出自己那一段单独播放；
	// 没传 songId 时退回旧的 file 路径行为（organize 旧导出路径兼容）。
	var srcPath string
	var cleanup func()
	if songIDStr := r.URL.Query().Get("songId"); songIDStr != "" {
		id, err := strconv.ParseInt(songIDStr, 10, 64)
		if err != nil {
			response.WriteValidationError(w, "songId 参数非法")
			return
		}
		path, cl, err := h.Library.SongAudioPath(id)
		if err != nil {
			response.WriteNotFoundError(w, "歌曲音频不存在")
			return
		}
		srcPath, cleanup = path, cl
		if cleanup != nil {
			defer cleanup()
		}
	} else {
		filename := r.URL.Query().Get("file")
		if filename == "" {
			response.WriteValidationError(w, "file 参数不能为空")
			return
		}
		resolved, err := h.validateSourcePath(filename)
		if err != nil {
			response.WriteValidationError(w, "非法文件路径")
			return
		}
		srcPath = resolved
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

// HandleSilent 生成一个固定时长的静音 MP3 并回传，供前端 File System Access API
// 那条导出路径在浏览器里直接写 SD 卡。
//
// 时长是硬编码的 17.43s（converter.DefaultSilentPlaceholder），与后端内部
// organizeService 走的是同一个常量——绝不能按 query 参数让前端随便传，否则
// 整天空着时生成的占位时长会跟内部路径不一致，SD 卡上同一序号文件的时长
// 就会对不上。
func (h *FileHandler) HandleSilent(w http.ResponseWriter, r *http.Request) {
	tempDir := paths.GetTempDir(h.AppDataDir)
	if err := paths.EnsureDir(tempDir); err != nil {
		response.WriteInternalError(w, "创建临时目录失败")
		return
	}

	tempPath := filepath.Join(tempDir, "silent.mp3")
	if err := h.Converter.GenerateSilentMP3(tempPath, converter.DefaultSilentPlaceholder); err != nil {
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
