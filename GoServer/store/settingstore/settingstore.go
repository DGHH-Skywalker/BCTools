package settingstore

import (
	"broadcast-tool/internal/repo"
	"broadcast-tool/internal/version"
	"broadcast-tool/models"
	"fmt"
)

// SettingsStore handles application settings.
type SettingsStore struct {
	repo *repo.Repository
}

// New creates a new SettingsStore backed by the given repository.
func New(repo *repo.Repository) *SettingsStore {
	return &SettingsStore{repo: repo}
}

func ensureBroadcastColumnMap(m map[string]string) map[string]string {
	defaults := map[string]string{"1": "新闻", "2": "文学", "3": "音乐", "4": "电影", "5": "新闻", "6": "文学", "7": "音乐"}
	if m == nil {
		m = map[string]string{}
	}
	for k, v := range defaults {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}

// GetSettings returns public settings (without password hash).
func (s *SettingsStore) GetSettings() models.PublicSettings {
	var public models.PublicSettings
	s.repo.Read(func(data *repo.Data) {
		public = models.PublicSettings{
			TimeSlots:          data.Settings.TimeSlots,
			AllowTemplateJS:    data.Settings.AllowTemplateJS,
			AdminPasswordHint:  data.Settings.AdminPasswordHint,
			Locale:             data.Settings.Locale,
			Version:            data.Settings.Version,
			DownloadURL:        data.Settings.DownloadURL,
			BroadcastColumnMap: ensureBroadcastColumnMap(data.Settings.BroadcastColumnMap),
			DuplicateCheckDays: data.Settings.DuplicateCheckDays,
		}
	})
	if public.DuplicateCheckDays <= 0 {
		public.DuplicateCheckDays = 30
	}
	public.Version = version.Version
	return public
}

// GetAdminPasswordHash returns the stored bcrypt hash.
func (s *SettingsStore) GetAdminPasswordHash() string {
	var hash string
	s.repo.Read(func(data *repo.Data) {
		hash = data.Settings.AdminPasswordHash
	})
	return hash
}

// GetSettingsRaw returns full settings including password hash.
func (s *SettingsStore) GetSettingsRaw() models.Settings {
	var settings models.Settings
	s.repo.Read(func(data *repo.Data) {
		settings = data.Settings
	})
	settings.BroadcastColumnMap = ensureBroadcastColumnMap(settings.BroadcastColumnMap)
	if settings.DuplicateCheckDays <= 0 {
		settings.DuplicateCheckDays = 30
	}
	settings.Version = version.Version
	return settings
}

// GetTimeSlots returns a copy of all time slots.
func (s *SettingsStore) GetTimeSlots() []models.TimeSlot {
	var slots []models.TimeSlot
	s.repo.Read(func(data *repo.Data) {
		slots = make([]models.TimeSlot, len(data.Settings.TimeSlots))
		copy(slots, data.Settings.TimeSlots)
	})
	return slots
}

// UpdateSettings updates settings fields from request (partial update).
func (s *SettingsStore) UpdateSettings(req models.UpdateSettingsRequest) error {
	_, err := s.UpdateSettingsAtomically(req, nil)
	return err
}

// Settings and affected song assignments share one durable commit. A failed
// settings write must not leave songs detached from their original slots.
func (s *SettingsStore) UpdateSettingsAtomically(req models.UpdateSettingsRequest, passwordHash *string) ([]models.Song, error) {
	removed := []models.Song{}
	err := s.repo.Write(func(data *repo.Data) error {
		if req.TimeSlots != nil {
			kept := map[string]bool{}
			for _, slot := range *req.TimeSlots {
				kept[slot.ID] = true
			}
			old := map[string]bool{}
			for _, slot := range data.Settings.TimeSlots {
				old[slot.ID] = true
			}
			for _, list := range []*[]models.Song{&data.DormSongs, &data.BroadcastSongs} {
				for i := range *list {
					song := &(*list)[i]
					if song.TimeSlotID != nil && old[*song.TimeSlotID] && !kept[*song.TimeSlotID] {
						if !req.Confirmed {
							return fmt.Errorf("时段关联在比对后发生变化，请重新确认删除")
						}
						removed = append(removed, *song)
						song.TimeSlotID = nil
					}
				}
			}
		}
		if passwordHash != nil {
			data.Settings.AdminPasswordHash = *passwordHash
		}
		if req.AllowTemplateJS != nil {
			data.Settings.AllowTemplateJS = *req.AllowTemplateJS
		}
		if req.AdminPasswordHint != nil {
			data.Settings.AdminPasswordHint = *req.AdminPasswordHint
		}
		if req.Locale != nil {
			if *req.Locale == "zh-CN" || *req.Locale == "en" {
				data.Settings.Locale = *req.Locale
			}
		}
		if req.TimeSlots != nil {
			data.Settings.TimeSlots = *req.TimeSlots
		}
		if req.BroadcastColumnMap != nil {
			data.Settings.BroadcastColumnMap = ensureBroadcastColumnMap(*req.BroadcastColumnMap)
		}
		if req.DuplicateCheckDays != nil {
			v := *req.DuplicateCheckDays
			if v < 1 {
				v = 1
			}
			if v > 365 {
				v = 365
			}
			data.Settings.DuplicateCheckDays = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return removed, nil
}

// SetAdminPassword sets the bcrypt password hash.
func (s *SettingsStore) SetAdminPassword(hash string) error {
	return s.repo.Write(func(data *repo.Data) error {
		data.Settings.AdminPasswordHash = hash
		return nil
	})
}
