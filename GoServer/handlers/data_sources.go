package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"broadcast-tool/internal/repo"
	"broadcast-tool/network"
	"broadcast-tool/response"
	"broadcast-tool/services"
	"github.com/go-chi/chi/v5"
)

func localDataRequest(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !network.IsLocalhost(host) {
		response.WriteError(w, 403, "FORBIDDEN", "数据来源操作仅限本机")
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
		response.WriteError(w, 403, "FORBIDDEN", "来源不匹配")
		return false
	}
	return true
}

func (h *SystemHandler) HandleDataSources(w http.ResponseWriter, r *http.Request) {
	if !localDataRequest(w, r) {
		return
	}
	sources, err := services.ListDataSources(h.appDataDir)
	if err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}
	response.WriteJSON(w, 200, map[string]interface{}{"directory": h.appDataDir, "sources": sources})
}
func (h *SystemHandler) HandleSourcePreview(w http.ResponseWriter, r *http.Request) {
	if !localDataRequest(w, r) {
		return
	}
	if h.Repo == nil {
		response.WriteInternalError(w, "未连接数据库")
		return
	}
	source, data, err := services.LoadDataSource(h.appDataDir, chi.URLParam(r, "sourceID"), "")
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	var current repo.Data
	h.Repo.Read(func(d *repo.Data) { b, _ := json.Marshal(d); _ = json.Unmarshal(b, &current) })
	response.WriteJSON(w, 200, map[string]interface{}{"source": source, "records": services.RecoveryPreview(current, data)})
}
func (h *SystemHandler) HandleSourceImport(w http.ResponseWriter, r *http.Request) {
	if !localDataRequest(w, r) {
		return
	}
	if h.Repo == nil {
		response.WriteInternalError(w, "未连接数据库")
		return
	}
	var req struct {
		SourceID  string                       `json:"sourceId"`
		Digest    string                       `json:"digest"`
		Records   []services.RecoverySelection `json:"records"`
		Confirmed bool                         `json:"confirmed"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024*1024)).Decode(&req) != nil || !req.Confirmed || req.Digest == "" {
		response.WriteValidationError(w, "请先比对并确认来源记录")
		return
	}
	source, data, err := services.LoadDataSource(h.appDataDir, req.SourceID, req.Digest)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	count, err := services.RecoverSelected(h.Repo, source, data, req.Records)
	if err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}
	response.WriteJSON(w, 200, map[string]int{"imported": count})
}
func (h *SystemHandler) HandleSearchDirectory(w http.ResponseWriter, r *http.Request) {
	if !localDataRequest(w, r) {
		return
	}
	var req struct {
		Directory string `json:"directory"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || !filepath.IsAbs(req.Directory) {
		response.WriteValidationError(w, "请选择完整目录")
		return
	}
	info, err := os.Lstat(req.Directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		response.WriteValidationError(w, "目录不存在或不可访问")
		return
	}
	file := filepath.Join(filepath.Dir(h.appDataDir), "search-directories.json")
	var dirs []string
	if b, err := os.ReadFile(file); err == nil {
		if err = json.Unmarshal(b, &dirs); err != nil {
			response.WriteInternalError(w, err.Error())
			return
		}
	} else if !os.IsNotExist(err) {
		response.WriteInternalError(w, err.Error())
		return
	}
	found := false
	for _, dir := range dirs {
		if strings.EqualFold(filepath.Clean(dir), filepath.Clean(req.Directory)) {
			found = true
		}
	}
	if !found {
		dirs = append(dirs, req.Directory)
	}
	b, err := json.MarshalIndent(dirs, "", "  ")
	if err == nil {
		err = os.WriteFile(file+".tmp", b, 0600)
	}
	if err == nil {
		err = os.Rename(file+".tmp", file)
	}
	if err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}
	if err := services.RecordDataSources(h.appDataDir, []string{req.Directory, filepath.Join(req.Directory, "BctoolData")}); err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}
	response.WriteJSON(w, 200, map[string]string{"status": "ok"})
}
