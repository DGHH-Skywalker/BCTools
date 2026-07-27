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
	"time"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/store"
	"broadcast-tool/validation"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
)

type SongHandler struct {
	Store *store.Store
}

func NewSongHandler(s *store.Store) *SongHandler {
	return &SongHandler{Store: s}
}

func (h *SongHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	songType := r.URL.Query().Get("type")
	if !validation.IsValidSongType(songType) {
		response.WriteValidationError(w, "type 参数必须为 dorm 或 broadcast")
		return
	}
	var dates []string
	if datesParam := r.URL.Query().Get("dates"); datesParam != "" {
		dates = strings.Split(datesParam, ",")
		for _, d := range dates {
			if !validation.IsValidDate(strings.TrimSpace(d)) {
				response.WriteValidationError(w, "日期格式错误，应为 YYYY-MM-DD")
				return
			}
		}
	}
	songs := h.Store.GetSongsByType(songType, dates)
	response.WriteJSON(w, http.StatusOK, songs)
}

func (h *SongHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteValidationError(w, "无效的歌曲 ID")
		return
	}
	song, _, err := h.Store.GetSongByID(id)
	if err != nil {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	}
	response.WriteJSON(w, http.StatusOK, song)
}

func (h *SongHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if !validation.IsValidSongType(req.Type) {
		response.WriteValidationError(w, "type 必须为 dorm 或 broadcast")
		return
	}
	if !validation.IsValidDate(req.Date) {
		response.WriteValidationError(w, "日期格式错误，应为 YYYY-MM-DD")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		response.WriteValidationError(w, "歌名不能为空")
		return
	}
	song, err := h.Store.AddSong(req)
	if err == store.ErrSlotConflict {
		response.WriteValidationError(w, "该时段已存在歌曲（同一天同一时段只能有一首宿舍歌）")
		return
	} else if err != nil {
		response.WriteInternalError(w, "保存歌曲失败")
		return
	}
	response.WriteCreated(w, song)
}

func (h *SongHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteValidationError(w, "无效的歌曲 ID")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.WriteValidationError(w, "读取请求体失败")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	var req models.UpdateSongRequest
	for key, val := range raw {
		switch key {
		case "date":
			var date string
			json.Unmarshal(val, &date)
			if !validation.IsValidDate(date) {
				response.WriteValidationError(w, "日期格式错误")
				return
			}
			req.Date = &date
		case "title":
			var title string
			json.Unmarshal(val, &title)
			if strings.TrimSpace(title) == "" {
				response.WriteValidationError(w, "歌名不能为空")
				return
			}
			req.Title = &title
		case "artist":
			var artist string
			json.Unmarshal(val, &artist)
			req.Artist = &artist
		case "remark":
			var remark string
			json.Unmarshal(val, &remark)
			req.Remark = &remark
		case "filePath":
			var fp string
			json.Unmarshal(val, &fp)
			req.FilePath = &fp
		case "timeSlotId":
			if string(val) == "null" {
				req.TimeSlotID = new(*string)
			} else {
				var slotID string
				json.Unmarshal(val, &slotID)
				s := slotID
				p := &s
				req.TimeSlotID = &p
			}
		}
	}
	song, err := h.Store.UpdateSong(id, req)
	if err == store.ErrNotFound {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	} else if err == store.ErrSlotConflict {
		response.WriteValidationError(w, "该时段已存在歌曲（同一天同一时段只能有一首宿舍歌）")
		return
	} else if err != nil {
		response.WriteInternalError(w, "更新失败")
		return
	}
	response.WriteJSON(w, http.StatusOK, song)
}

func (h *SongHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteValidationError(w, "无效的歌曲 ID")
		return
	}
	if err := h.Store.DeleteSong(id); err == store.ErrNotFound {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	} else if err != nil {
		response.WriteInternalError(w, "删除失败")
		return
	}
	response.WriteNoContent(w)
}

func (h *SongHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	songType := r.URL.Query().Get("type")
	if !validation.IsValidSongType(songType) {
		response.WriteValidationError(w, "type 必须为 dorm 或 broadcast")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.WriteValidationError(w, "文件过大或格式错误")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteValidationError(w, "请上传文件")
		return
	}
	defer file.Close()
	if ext := filepath.Ext(header.Filename); !strings.EqualFold(ext, ".xlsx") {
		response.WriteValidationError(w, "仅支持 xlsx 文件")
		return
	}
	tempFile := filepath.Join(h.Store.GetAppDataDir(), "temp", fmt.Sprintf("import-%d.xlsx", time.Now().UnixNano()))
	out, err := os.Create(tempFile)
	if err != nil {
		response.WriteInternalError(w, "保存临时文件失败")
		return
	}
	io.Copy(out, file)
	out.Close()
	defer os.Remove(tempFile)

	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		response.WriteValidationError(w, "无法解析 xlsx 文件")
		return
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		response.WriteValidationError(w, "xlsx 文件没有工作表")
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		response.WriteValidationError(w, "读取工作表失败")
		return
	}

	type vr struct{ date, title, remark string }
	var valid []vr
	var errors []models.ImportError
	var skipped int
	seen := make(map[string]bool)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rn := i + 1
		date, title, remark := "", "", ""
		if len(row) > 0 {
			date = strings.TrimSpace(row[0])
		}
		if len(row) > 2 {
			title = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			remark = strings.TrimSpace(row[3])
		}
		if date == "" && title == "" {
			skipped++
			continue
		}
		if !validation.IsValidDate(date) {
			errors = append(errors, models.ImportError{Row: rn, Message: "日期格式错误"})
			continue
		}
		if title == "" {
			errors = append(errors, models.ImportError{Row: rn, Message: "歌名不能为空"})
			continue
		}
		key := date + "|" + strings.ToLower(strings.TrimSpace(title))
		if seen[key] {
			errors = append(errors, models.ImportError{Row: rn, Message: "xlsx 内重复"})
			continue
		}
		if h.Store.CheckDuplicate(songType, date, title) {
			errors = append(errors, models.ImportError{Row: rn, Message: "与已有数据重复"})
			continue
		}
		seen[key] = true
		valid = append(valid, vr{date, title, remark})
	}
	h.Store.CreatePreImportSnapshot()
	now := time.Now().UTC()
	var ns []models.Song
	for _, v := range valid {
		ns = append(ns, models.Song{ID: h.Store.GetNextID(), Date: v.date, Title: v.title, Remark: v.remark, CreatedAt: now})
	}
	if err := h.Store.ReplaceSongsByType(songType, ns); err != nil {
		response.WriteInternalError(w, "导入写入失败")
		return
	}
	response.WriteJSON(w, http.StatusOK, models.ImportResult{Inserted: len(valid), Skipped: skipped, Errors: errors})
}

func (h *SongHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	songType := r.URL.Query().Get("type")
	if !validation.IsValidSongType(songType) {
		response.WriteValidationError(w, "type 必须为 dorm 或 broadcast")
		return
	}
	var dates []string
	if dp := r.URL.Query().Get("dates"); dp != "" {
		dates = strings.Split(dp, ",")
	}
	songs := h.Store.GetSongsByType(songType, dates)

	f := excelize.NewFile()
	defer f.Close()
	sheetName := "宿舍歌单"
	if songType == "broadcast" {
		sheetName = "播音歌单"
	}
	f.SetSheetName("Sheet1", sheetName)
	for i, hdr := range []string{"日期", "星期", "歌名", "备注"} {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, c, hdr)
	}
	for i, song := range songs {
		r := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", r), song.Date)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", r), validation.CalculateWeekday(song.Date))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", r), song.Title)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", r), song.Remark)
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="playlist.xlsx"`)
	f.Write(w)
}
