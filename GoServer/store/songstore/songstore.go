package songstore

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/store"
)

// SongStore handles song CRUD and related queries.
type SongStore struct {
	repo *repo.Repository
}

// New creates a new SongStore backed by the given repository.
func New(repo *repo.Repository) *SongStore {
	return &SongStore{repo: repo}
}

// GetDormSongs returns all dorm songs with computed weekday.
func (s *SongStore) GetDormSongs() []models.Song {
	var result []models.Song
	s.repo.Read(func(data *repo.Data) {
		result = cloneSongsWithWeekday(data.DormSongs)
	})
	return result
}

// GetBroadcastSongs returns all broadcast songs with computed weekday.
func (s *SongStore) GetBroadcastSongs() []models.Song {
	var result []models.Song
	s.repo.Read(func(data *repo.Data) {
		result = cloneSongsWithWeekday(data.BroadcastSongs)
	})
	return result
}

// GetSongByID returns a song by its ID from either list.
func (s *SongStore) GetSongByID(id int64) (models.Song, string, error) {
	var result models.Song
	var songType string
	var found bool
	s.repo.Read(func(data *repo.Data) {
		for _, song := range data.DormSongs {
			if song.ID == id {
				result = song
				songType = "dorm"
				found = true
				return
			}
		}
		for _, song := range data.BroadcastSongs {
			if song.ID == id {
				result = song
				songType = "broadcast"
				found = true
				return
			}
		}
	})
	if !found {
		return models.Song{}, "", store.ErrNotFound
	}
	result.Weekday = repo.CalcWeekday(result.Date)
	return result, songType, nil
}

// GetSongsByType returns songs filtered by type and optional dates.
// Dorm songs are returned in canonical order (date → slot time → within-slot
// drag order → id) so every consumer—API list, xlsx export, image export—sees
// the same sequence without having to re-sort.
func (s *SongStore) GetSongsByType(songType string, dates []string) []models.Song {
	var result []models.Song
	var slotTime map[string]int
	s.repo.Read(func(data *repo.Data) {
		switch songType {
		case "dorm":
			result = cloneSongsWithWeekday(data.DormSongs)
			slotTime = buildSlotTimeMap(data.Settings.TimeSlots)
		case "broadcast":
			result = cloneSongsWithWeekday(data.BroadcastSongs)
		default:
			result = nil
		}
	})
	if result == nil {
		return nil
	}
	if songType == "dorm" {
		sort.SliceStable(result, func(i, j int) bool {
			return dormLess(result[i], result[j], slotTime)
		})
	}
	if len(dates) == 0 {
		return result
	}
	dateSet := make(map[string]bool, len(dates))
	for _, d := range dates {
		dateSet[d] = true
	}
	filtered := make([]models.Song, 0, len(result))
	for _, song := range result {
		if dateSet[song.Date] {
			filtered = append(filtered, song)
		}
	}
	return filtered
}

// IsSongsEmpty checks if both song lists are empty.
func (s *SongStore) IsSongsEmpty() bool {
	var empty bool
	s.repo.Read(func(data *repo.Data) {
		empty = len(data.DormSongs) == 0 && len(data.BroadcastSongs) == 0
	})
	return empty
}

// SortDormSongs sorts dorm songs by date, then by the time slot's actual time
// (chronological), then by the within-slot drag order, then by ID.
func (s *SongStore) SortDormSongs() error {
	return s.repo.Write(func(data *repo.Data) error {
		slotTime := buildSlotTimeMap(data.Settings.TimeSlots)
		sort.SliceStable(data.DormSongs, func(i, j int) bool {
			return dormLess(data.DormSongs[i], data.DormSongs[j], slotTime)
		})
		return nil
	})
}

// buildSlotTimeMap maps each time slot ID to its minute-of-day value.
func buildSlotTimeMap(slots []models.TimeSlot) map[string]int {
	m := make(map[string]int, len(slots))
	for _, slot := range slots {
		m[slot.ID] = parseTimeMinutes(slot.Time)
	}
	return m
}

// dormLess is the canonical dorm ordering: date, then slot time, then within-slot
// drag order, then ID. It is shared by SortDormSongs (mutating) and GetSongsByType
// (read-only) so the stored order and the served order never diverge.
func dormLess(a, b models.Song, slotTime map[string]int) bool {
	if a.Date != b.Date {
		return a.Date < b.Date
	}
	ta, tb := slotTimeMinutes(a.TimeSlotID, slotTime), slotTimeMinutes(b.TimeSlotID, slotTime)
	if ta != tb {
		return ta < tb
	}
	if a.Order != b.Order {
		return a.Order < b.Order
	}
	return a.ID < b.ID
}

// parseTimeMinutes turns an "HH:MM" string into minutes since midnight.
func parseTimeMinutes(t string) int {
	parts := strings.Split(t, ":")
	if len(parts) < 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	if h < 0 || h > 23 {
		h = 0
	}
	if m < 0 || m > 59 {
		m = 0
	}
	return h*60 + m
}

func slotTimeMinutes(id *string, slotTime map[string]int) int {
	if id == nil {
		return 0
	}
	if v, ok := slotTime[*id]; ok {
		return v
	}
	return 0
}

// ReorderSongs assigns within-slot drag order by position for the given song IDs.
func (s *SongStore) ReorderSongs(ids []int64) error {
	return s.repo.Write(func(data *repo.Data) error {
		orderMap := make(map[int64]int, len(ids))
		for i, id := range ids {
			orderMap[id] = i
		}
		for i := range data.DormSongs {
			if ord, ok := orderMap[data.DormSongs[i].ID]; ok {
				data.DormSongs[i].Order = ord
			}
		}
		for i := range data.BroadcastSongs {
			if ord, ok := orderMap[data.BroadcastSongs[i].ID]; ok {
				data.BroadcastSongs[i].Order = ord
			}
		}
		return nil
	})
}

// AddSong creates a new song with auto-generated ID and timestamp.
func (s *SongStore) AddSong(req models.CreateSongRequest) (models.Song, error) {
	if req.Type != "dorm" && req.Type != "broadcast" {
		return models.Song{}, store.ErrValidation
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return models.Song{}, store.ErrValidation
	}
	sanitize := func(v string, max int, allowEmpty bool) string {
		if !allowEmpty {
			v = strings.TrimSpace(v)
		}
		if len(v) > max {
			v = v[:max]
		}
		return v
	}
	req.Title = sanitize(req.Title, 500, req.Type == "broadcast")
	req.Artist = sanitize(req.Artist, 500, true)
	req.Remark = sanitize(req.Remark, 1000, true)

	var created models.Song
	err := s.repo.Write(func(data *repo.Data) error {
		// 同一宿舍时段允许放置多首歌曲，不再做冲突校验。

		var maxID int64
		for _, song := range data.DormSongs {
			if song.ID > maxID {
				maxID = song.ID
			}
		}
		for _, song := range data.BroadcastSongs {
			if song.ID > maxID {
				maxID = song.ID
			}
		}
		now := time.Now().UTC()
		period := ""
		if req.Period != nil {
			period = strings.TrimSpace(*req.Period)
		}
		if req.Type == "broadcast" && period == "" {
			if now.Hour() < 12 {
				period = "noon"
			} else {
				period = "afternoon"
			}
		}
		// 新宿舍歌曲追加到所在时段末尾：取同日同时段已有歌曲的最大 Order + 1。
		order := 0
		if req.Type == "dorm" && req.TimeSlotID != nil && strings.TrimSpace(*req.TimeSlotID) != "" {
			slotID := *req.TimeSlotID
			maxOrder := -1
			for _, song := range data.DormSongs {
				if song.Date == req.Date && song.TimeSlotID != nil && *song.TimeSlotID == slotID {
					if song.Order > maxOrder {
						maxOrder = song.Order
					}
				}
			}
			order = maxOrder + 1
		}
		song := models.Song{
			ID:         maxID + 1,
			Date:       req.Date,
			Title:      req.Title,
			Artist:     req.Artist,
			Remark:     req.Remark,
			FilePath:   req.FilePath,
			TimeSlotID: req.TimeSlotID,
			CreatedAt:  now,
			Period:     period,
			Order:      order,
		}
		switch req.Type {
		case "dorm":
			data.DormSongs = append(data.DormSongs, song)
		case "broadcast":
			data.BroadcastSongs = append(data.BroadcastSongs, song)
		}
		created = song
		return nil
	})
	if err != nil {
		return models.Song{}, err
	}
	created.Weekday = repo.CalcWeekday(created.Date)
	return created, nil
}

// UpdateSong updates specific fields of a song.
func (s *SongStore) UpdateSong(id int64, req models.UpdateSongRequest) (models.Song, error) {
	var updated models.Song
	err := s.repo.Write(func(data *repo.Data) error {
		idx, songType := findSongIndex(data, id)
		if idx < 0 {
			return store.ErrNotFound
		}
		var songs *[]models.Song
		switch songType {
		case "dorm":
			songs = &data.DormSongs
		case "broadcast":
			songs = &data.BroadcastSongs
		default:
			return store.ErrNotFound
		}
		song := &(*songs)[idx]
		if req.Title != nil {
			song.Title = strings.TrimSpace(*req.Title)
		}
		if req.Artist != nil {
			song.Artist = strings.TrimSpace(*req.Artist)
		}
		if req.Remark != nil {
			song.Remark = strings.TrimSpace(*req.Remark)
		}
		if req.FilePath != nil {
			song.FilePath = *req.FilePath
		}
		if req.Date != nil {
			song.Date = *req.Date
		}
		if req.TimeSlotID != nil {
			song.TimeSlotID = *req.TimeSlotID
		}
		if req.Period != nil {
			song.Period = strings.TrimSpace(*req.Period)
		}
		// 同一宿舍时段允许放置多首歌曲，不再做冲突校验。
		updated = *song
		return nil
	})
	if err != nil {
		return models.Song{}, err
	}
	updated.Weekday = repo.CalcWeekday(updated.Date)
	return updated, nil
}

// DeleteSong removes a song by ID.
func (s *SongStore) DeleteSong(id int64) error {
	return s.repo.Write(func(data *repo.Data) error {
		idx, songType := findSongIndex(data, id)
		if idx < 0 {
			return store.ErrNotFound
		}
		switch songType {
		case "dorm":
			data.DormSongs = append(data.DormSongs[:idx], data.DormSongs[idx+1:]...)
		case "broadcast":
			data.BroadcastSongs = append(data.BroadcastSongs[:idx], data.BroadcastSongs[idx+1:]...)
		}
		return nil
	})
}

// ReplaceSongsByType replaces all songs of a given type (for full import).
func (s *SongStore) ReplaceSongsByType(songType string, songs []models.Song) error {
	if songs == nil {
		songs = []models.Song{}
	}
	return s.repo.Write(func(data *repo.Data) error {
		switch songType {
		case "dorm":
			data.DormSongs = songs
		case "broadcast":
			data.BroadcastSongs = songs
		default:
			return store.ErrValidation
		}
		return nil
	})
}

// UnassignTimeSlotIDs nullifies timeSlotId for all songs matching given IDs.
func (s *SongStore) UnassignTimeSlotIDs(timeSlotIDs map[string]bool) ([]models.Song, error) {
	var removed []models.Song
	err := s.repo.Write(func(data *repo.Data) error {
		removed = nil
		for i := range data.DormSongs {
			if data.DormSongs[i].TimeSlotID != nil && timeSlotIDs[*data.DormSongs[i].TimeSlotID] {
				removed = append(removed, data.DormSongs[i])
				data.DormSongs[i].TimeSlotID = nil
			}
		}
		for i := range data.BroadcastSongs {
			if data.BroadcastSongs[i].TimeSlotID != nil && timeSlotIDs[*data.BroadcastSongs[i].TimeSlotID] {
				removed = append(removed, data.BroadcastSongs[i])
				data.BroadcastSongs[i].TimeSlotID = nil
			}
		}
		return nil
	})
	return removed, err
}

// GetSongsAssignedToTimeSlotIDs returns songs assigned to given time slot IDs.
func (s *SongStore) GetSongsAssignedToTimeSlotIDs(ids map[string]bool) []models.Song {
	var affected []models.Song
	s.repo.Read(func(data *repo.Data) {
		check := func(songs []models.Song) {
			for _, song := range songs {
				if song.TimeSlotID != nil && ids[*song.TimeSlotID] {
					song.Weekday = repo.CalcWeekday(song.Date)
					affected = append(affected, song)
				}
			}
		}
		check(data.DormSongs)
		check(data.BroadcastSongs)
	})
	return affected
}

// GetNextID returns the next available song ID.
func (s *SongStore) GetNextID() int64 {
	var maxID int64
	s.repo.Read(func(data *repo.Data) {
		for _, song := range data.DormSongs {
			if song.ID > maxID {
				maxID = song.ID
			}
		}
		for _, song := range data.BroadcastSongs {
			if song.ID > maxID {
				maxID = song.ID
			}
		}
	})
	return maxID + 1
}

// CheckDormSlotConflict checks if the given dorm date+timeSlot already has a song.
func (s *SongStore) CheckDormSlotConflict(songType, date, timeSlotID string, excludeID int64) bool {
	if songType != "dorm" || strings.TrimSpace(timeSlotID) == "" {
		return false
	}
	var conflict bool
	s.repo.Read(func(data *repo.Data) {
		conflict = checkDormSlotConflict(data.DormSongs, date, timeSlotID, excludeID)
	})
	return conflict
}

// CheckDuplicate checks if a song with same date+title+type already exists.
func (s *SongStore) CheckDuplicate(songType, date, title string) bool {
	var duplicate bool
	title = strings.TrimSpace(strings.ToLower(title))
	s.repo.Read(func(data *repo.Data) {
		var songs []models.Song
		switch songType {
		case "dorm":
			songs = data.DormSongs
		case "broadcast":
			songs = data.BroadcastSongs
		default:
			return
		}
		for _, song := range songs {
			if song.Date == date && strings.EqualFold(strings.TrimSpace(song.Title), title) {
				duplicate = true
				return
			}
		}
	})
	return duplicate
}

func findSongIndex(data *repo.Data, id int64) (int, string) {
	for i, song := range data.DormSongs {
		if song.ID == id {
			return i, "dorm"
		}
	}
	for i, song := range data.BroadcastSongs {
		if song.ID == id {
			return i, "broadcast"
		}
	}
	return -1, ""
}

func cloneSongsWithWeekday(songs []models.Song) []models.Song {
	result := make([]models.Song, len(songs))
	for i, song := range songs {
		song.Weekday = repo.CalcWeekday(song.Date)
		result[i] = song
	}
	return result
}

func checkDormSlotConflict(songs []models.Song, date, timeSlotID string, excludeID int64) bool {
	for _, song := range songs {
		if song.ID == excludeID {
			continue
		}
		if song.Date == date && song.TimeSlotID != nil && strings.EqualFold(strings.TrimSpace(*song.TimeSlotID), timeSlotID) {
			return true
		}
	}
	return false
}

// GetAppDataDir returns the app data directory from the underlying repository.
func (s *SongStore) GetAppDataDir() string {
	return s.repo.AppDataDir()
}
