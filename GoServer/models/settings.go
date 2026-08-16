package models

// TimeSlot represents a broadcast time slot
type TimeSlot struct {
	ID       string `json:"id"`       // UUID, immutable
	DayIndex int    `json:"dayIndex"` // 1=Monday..7=Sunday
	Order    int    `json:"order"`    // sequence number, >=1
	Time     string `json:"time"`     // "HH:mm"
}

// Settings holds application settings.
//
// 注：静音占位时长是 converter.DefaultSilentPlaceholder（17.43s）写死的常量，
// 不再作为设置项暴露——见 converter/silent.go。旧数据文件里的
// silentPlaceholderDuration 字段会被忽略。
type Settings struct {
	TimeSlots          []TimeSlot        `json:"timeSlots"`
	AllowTemplateJS    bool              `json:"allowTemplateJS"`
	AdminPasswordHash  string            `json:"adminPasswordHash,omitempty"`
	AdminPasswordHint  string            `json:"adminPasswordHint,omitempty"`
	Locale             string            `json:"locale"`
	Version            string            `json:"version"`
	DownloadURL        string            `json:"downloadUrl"`
	BroadcastColumnMap map[string]string `json:"broadcastColumnMap"`
	DuplicateCheckDays int               `json:"duplicateCheckDays"`
}

// PublicSettings is the settings object returned to clients (without password hash)
type PublicSettings struct {
	TimeSlots          []TimeSlot        `json:"timeSlots"`
	AllowTemplateJS    bool              `json:"allowTemplateJS"`
	AdminPasswordHint  string            `json:"adminPasswordHint"`
	Locale             string            `json:"locale"`
	Version            string            `json:"version"`
	DownloadURL        string            `json:"downloadUrl"`
	BroadcastColumnMap map[string]string `json:"broadcastColumnMap"`
	DuplicateCheckDays int               `json:"duplicateCheckDays"`
}

// UpdateSettingsRequest contains fields to update in settings
type UpdateSettingsRequest struct {
	TimeSlots          *[]TimeSlot        `json:"timeSlots,omitempty"`
	AllowTemplateJS    *bool              `json:"allowTemplateJS,omitempty"`
	AdminPassword      *string            `json:"adminPassword,omitempty"`
	AdminPasswordHint  *string            `json:"adminPasswordHint,omitempty"`
	Locale             *string            `json:"locale,omitempty"`
	Confirmed          bool               `json:"confirmed,omitempty"`
	BroadcastColumnMap *map[string]string `json:"broadcastColumnMap,omitempty"`
	DuplicateCheckDays *int               `json:"duplicateCheckDays,omitempty"`
}
