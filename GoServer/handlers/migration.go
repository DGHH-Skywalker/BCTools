package handlers

import (
	"encoding/json"
	"net/http"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/services"
)

// MigrationHandler exposes two endpoints:
//
//	POST /api/migration/preview  : 选完目录后立即问「下一个候选子目录是否已存在」，
//	                               把 existingFiles 一起带回去给前端弹框用。
//	POST /api/migration/export   : 真正写文件。json 模式直接回 JSON 给浏览器下载；
//	                               files 模式走两阶段：先 confirmNeeded，再带 confirm=true。
//
// 跟现有 /api/files/organize 的两阶段交互对齐，让前端可以复用同一种 modal 模式。
type MigrationHandler struct {
	Service *services.MigrationService
}

func NewMigrationHandler(svc *services.MigrationService) *MigrationHandler {
	return &MigrationHandler{Service: svc}
}

func (h *MigrationHandler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetDir string `json:"targetDir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if req.TargetDir == "" {
		response.WriteValidationError(w, "targetDir 不能为空")
		return
	}
	preview, err := h.Service.Preview(req.TargetDir)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, preview)
}

func (h *MigrationHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	var req models.MigrationExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if req.Scope != models.MigrationScopeJSON && req.Scope != models.MigrationScopeFiles {
		response.WriteValidationError(w, "scope 必须为 json 或 files")
		return
	}

	// json 模式：直接回 JSON 字节流，浏览器用 a.download 触发下载。
	// 走这条路径就不需要「子目录」概念，所以不走 Export 的目录流程，
	// 也不要求 targetDir。
	if req.Scope == models.MigrationScopeJSON {
		data, err := h.Service.BuildPayloadJSON()
		if err != nil {
			response.WriteInternalError(w, "生成数据失败: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition",
			`attachment; filename="broadcast-tool-data.json"`)
		_, _ = w.Write(data)
		return
	}

	// files 模式：需要 targetDir；先校验。
	if req.TargetDir == "" {
		response.WriteValidationError(w, "targetDir 不能为空")
		return
	}
	result, err := h.Service.Export(req)
	if err != nil {
		if services.IsConfirmNeeded(err) {
			// 把 existingFiles 也带回去，让前端在弹框里列出。
			preview, _ := h.Service.Preview(req.TargetDir)
			writeConfirmResponse(w, preview)
			return
		}
		response.WriteValidationError(w, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

// migrationConfirmResponse 给前端发「需要确认」信号，沿用 OrganizeResponse
// 的字段名风格。existingFiles 复用 models.MigrationPreview 的列表。
type migrationConfirmResponse struct {
	ConfirmNeeded bool     `json:"confirmNeeded"`
	BackupDir     string   `json:"backupDir"`
	ExistingFiles []string `json:"existingFiles,omitempty"`
}

func writeConfirmResponse(w http.ResponseWriter, preview models.MigrationPreview) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(migrationConfirmResponse{
		ConfirmNeeded: true,
		BackupDir:     preview.BackupDir,
		ExistingFiles: preview.ExistingFiles,
	})
}

// HandleImport 从用户选定的备份目录还原数据。前端调 POST /api/migration/import。
//
// 流程与 Export 共享：handler 解析 body 校验 mode，service.Import 做实际工作。
// service 已经把"复制歌曲 + 写主数据 + 合并设置"封装成一步；失败不会留下半成品。
func (h *MigrationHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	var req models.MigrationImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	// backupDir 与 jsonData 二选一；handler 层不重复校验，service.Import 会兜底
	if req.BackupDir == "" && req.JSONData == nil {
		response.WriteValidationError(w, "backupDir 与 jsonData 至少填一个")
		return
	}
	if req.Mode != "" && req.Mode != "merge" && req.Mode != "replace" {
		response.WriteValidationError(w, "mode 必须为 merge 或 replace")
		return
	}

	result, err := h.Service.Import(req)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}
