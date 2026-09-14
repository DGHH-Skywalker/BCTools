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

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/services"
	"broadcast-tool/store"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"
	"broadcast-tool/validation"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
)

type SongHandler struct {
	Songs         *songstore.SongStore
	Snapshots     *snapshotstore.SnapshotStore
	Library       *services.MusicLibrary
	importService *services.ImportService
}

func NewSongHandler(songs *songstore.SongStore, snapshots *snapshotstore.SnapshotStore, library *services.MusicLibrary) *SongHandler {
	return &SongHandler{
		Songs:         songs,
		Snapshots:     snapshots,
		Library:       library,
		importService: services.NewImportService(songs, snapshots),
	}
}

// syncAfterChange 在歌曲增删改之后收敛周文件夹布局。失败只记日志不阻断响应：
// data.json 已经更新成功，布局会在下一次同步/导出时自愈。
func (h *SongHandler) syncAfterChange(songs ...models.Song) {
	if h.Library == nil {
		return
	}
	for _, s := range songs {
		if s.Date == "" {
			continue
		}
		if err := h.Library.SyncSongWeek(s); err != nil {
			log.Printf("musiclibrary: sync week for song %d failed: %v", s.ID, err)
		}
	}
}

// scheduleSyncAfterChange keeps editing responsive. The playlist is already
// committed; a later export also performs a full synchronization.
func (h *SongHandler) scheduleSyncAfterChange(songs ...models.Song) {
	go h.syncAfterChange(songs...)
}

func (h *SongHandler) HandleSort(w http.ResponseWriter, r *http.Request) {
	songType := r.URL.Query().Get("type")
	if songType != "dorm" {
		response.WriteValidationError(w, "仅支持 dorm 类型排序")
		return
	}
	if err := h.Songs.SortDormSongs(); err != nil {
		response.WriteInternalError(w, "排序失败")
		return
	}
	h.Snapshots.EnsureDailySnapshot()
	songs := h.Songs.GetSongsByType(songType, nil)
	response.WriteJSON(w, http.StatusOK, songs)
}

func (h *SongHandler) HandleReorder(w http.ResponseWriter, r *http.Request) {
	var req models.ReorderSongsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if len(req.SongIDs) == 0 {
		response.WriteValidationError(w, "songIds 不能为空")
		return
	}
	if err := h.Songs.ReorderSongs(req.SongIDs); err != nil {
		response.WriteInternalError(w, "排序失败")
		return
	}
	h.Snapshots.EnsureDailySnapshot()
	response.WriteNoContent(w)
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
	songs := h.Songs.GetSongsByType(songType, dates)
	response.WriteJSON(w, http.StatusOK, songs)
}

func (h *SongHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteValidationError(w, "无效的歌曲 ID")
		return
	}
	song, _, err := h.Songs.GetSongByID(id)
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
	song, err := h.Songs.AddSong(req)
	if err == store.ErrSlotConflict {
		response.WriteValidationError(w, "该时段已存在歌曲（同一天同一时段只能有一首宿舍歌）")
		return
	} else if err != nil {
		response.WriteInternalError(w, "保存歌曲失败")
		return
	}
	h.scheduleSyncAfterChange(song)
	h.Snapshots.EnsureDailySnapshot()
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
		case "period":
			var period string
			json.Unmarshal(val, &period)
			req.Period = &period
		}
	}
	// 先取旧值：日期/时段变更后需要同步旧布局（把音频搬离原编号）
	oldSong, _, getErr := h.Songs.GetSongByID(id)
	if getErr != nil {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	}
	song, err := h.Songs.UpdateSong(id, req)
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
	h.scheduleSyncAfterChange(oldSong, song)
	h.Snapshots.EnsureDailySnapshot()
	response.WriteJSON(w, http.StatusOK, song)
}

func (h *SongHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteValidationError(w, "无效的歌曲 ID")
		return
	}
	// 删除前取歌曲信息，用于同步周文件夹（时段可能从有歌变空、需要静音占位）
	deletedSong, _, getErr := h.Songs.GetSongByID(id)
	if getErr != nil {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	}
	if err := h.Songs.DeleteSong(id); err == store.ErrNotFound {
		response.WriteNotFoundError(w, "歌曲不存在")
		return
	} else if err != nil {
		response.WriteInternalError(w, "删除失败")
		return
	}
	h.scheduleSyncAfterChange(deletedSong)
	h.Snapshots.EnsureDailySnapshot()
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

	tempFile, err := h.importService.SaveUploadedFile(file, h.Songs.GetAppDataDir())
	if err != nil {
		response.WriteInternalError(w, "保存临时文件失败")
		return
	}
	defer os.Remove(tempFile)

	result, err := h.importService.ImportXlsx(tempFile, songType)
	if err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
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
	songs := h.Songs.GetSongsByType(songType, dates)

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
