package handlers

import (
	"net/http"

	"broadcast-tool/launcher"
	"broadcast-tool/paths"
	"broadcast-tool/response"
)

// SystemHandler exposes runtime environment status for frontend feasibility checks.
type SystemHandler struct {
	appDataDir string
	version    string
}

// NewSystemHandler creates a SystemHandler.
func NewSystemHandler(appDataDir, version string) *SystemHandler {
	return &SystemHandler{appDataDir: appDataDir, version: version}
}

// HandleStatus returns version and ffmpeg/ffprobe availability.
func (h *SystemHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	binDir := paths.GetBinDir(h.appDataDir)
	exeDir := paths.GetExecutableDir()
	ffmpegPath := launcher.ResolveBinary("ffmpeg.exe", binDir, exeDir)
	ffprobePath := launcher.ResolveBinary("ffprobe.exe", binDir, exeDir)

	response.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"version":          h.version,
		"ffmpegAvailable":  ffmpegPath != "",
		"ffprobeAvailable": ffprobePath != "",
	})
}
