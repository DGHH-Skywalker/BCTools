package handlers

import (
	"net/http"

	"broadcast-tool/internal/repo"
	"broadcast-tool/launcher"
	"broadcast-tool/paths"
	"broadcast-tool/response"
)

// SystemHandler exposes runtime environment status for frontend feasibility checks.
type SystemHandler struct {
	appDataDir string
	version    string
	Repo       *repo.Repository
}

// NewSystemHandler creates a SystemHandler.
func NewSystemHandler(appDataDir, version string, repositories ...*repo.Repository) *SystemHandler {
	h := &SystemHandler{appDataDir: appDataDir, version: version}
	if len(repositories) > 0 {
		h.Repo = repositories[0]
	}
	return h
}

// HandleStatus returns version and ffmpeg/ffprobe availability.
func (h *SystemHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	binDir := paths.GetBinDir(h.appDataDir)
	exeDir := paths.GetExecutableDir()
	ffmpegPath := launcher.ResolveBinary("ffmpeg.exe", binDir, exeDir)
	ffprobePath := launcher.ResolveBinary("ffprobe.exe", binDir, exeDir)

	response.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"version":             h.version,
		"dataDirectory":       h.appDataDir,
		"executableDirectory": exeDir,
		"ffmpegAvailable":     ffmpegPath != "",
		"ffprobeAvailable":    ffprobePath != "",
	})
}
