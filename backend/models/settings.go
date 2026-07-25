package models

// TimeSlot represents a broadcast time slot
type TimeSlot struct {
	ID       string `json:"id"`       // UUID, immutable
	DayIndex int    `json:"dayIndex"` // 1=Monday..7=Sunday
	Order    int    `json:"order"`    // sequence number, >=1
	Time     string `json:"time"`     // "HH:mm"
}

// Settings holds application settings
type Settings struct {
	TimeSlots                 []TimeSlot `json:"timeSlots"`
	AllowTemplateJS           bool       `json:"allowTemplateJS"`
	AutoBackupPath            string     `json:"autoBackupPath"`
	AutoBackupEnabled         bool       `json:"autoBackupEnabled"`
	SilentPlaceholderDuration int        `json:"silentPlaceholderDuration"`
	AdminPasswordHash         string     `json:"adminPasswordHash,omitempty"`
	AdminPasswordHint         string     `json:"adminPasswordHint,omitempty"`
	Locale                    string     `json:"locale"`
	Version                   string     `json:"version"`
	DownloadURL               string     `json:"downloadUrl"`
}

// PublicSettings is the settings object returned to clients (without password hash)
type PublicSettings struct {
	TimeSlots                 []TimeSlot `json:"timeSlots"`
	AllowTemplateJS           bool       `json:"allowTemplateJS"`
	AutoBackupPath            string     `json:"autoBackupPath"`
	AutoBackupEnabled         bool       `json:"autoBackupEnabled"`
	SilentPlaceholderDuration int        `json:"silentPlaceholderDuration"`
	AdminPasswordHint         string     `json:"adminPasswordHint"`
	Locale                    string     `json:"locale"`
	Version                   string     `json:"version"`
	DownloadURL               string     `json:"downloadUrl"`
}

// UpdateSettingsRequest contains fields to update in settings
type UpdateSettingsRequest struct {
	TimeSlots                 *[]TimeSlot `json:"timeSlots,omitempty"`
	AllowTemplateJS           *bool       `json:"allowTemplateJS,omitempty"`
	AutoBackupPath            *string     `json:"autoBackupPath,omitempty"`
	AutoBackupEnabled         *bool       `json:"autoBackupEnabled,omitempty"`
	SilentPlaceholderDuration *int        `json:"silentPlaceholderDuration,omitempty"`
	AdminPassword             *string     `json:"adminPassword,omitempty"`
	AdminPasswordHint         *string     `json:"adminPasswordHint,omitempty"`
	Locale                    *string     `json:"locale,omitempty"`
	Confirmed                 bool        `json:"confirmed,omitempty"`
}
