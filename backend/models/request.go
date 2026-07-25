package models

// OrganizeEntry represents one file to organize
type OrganizeEntry struct {
	Source     string `json:"source"`
	TargetName string `json:"targetName"`
}

// OrganizeRequest is the request body for POST /api/files/organize
type OrganizeRequest struct {
	Entries   []OrganizeEntry `json:"entries"`
	TargetDir string          `json:"targetDir"`
	Mode      string          `json:"mode"` // "copy" or "move"
	Confirm   bool            `json:"confirm,omitempty"`
}

// OrganizeResult is a single file operation result
type OrganizeResult struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// OrganizeResponse is the response for POST /api/files/organize
type OrganizeResponse struct {
	Successful    []OrganizeResult `json:"successful"`
	Failed        []FailedItem     `json:"failed"`
	Warning       string           `json:"warning,omitempty"`
	ConfirmNeeded bool             `json:"confirmNeeded,omitempty"`
	ExistingFiles []string         `json:"existingFiles,omitempty"`
}

// FailedItem represents a failed operation
type FailedItem struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// AuthRequest is the request body for POST /api/auth/verify
type AuthRequest struct {
	Password string `json:"password"`
}

// AuthResponse is the response for POST /api/auth/verify
type AuthResponse struct {
	Success bool   `json:"success"`
	Hint    string `json:"hint,omitempty"`
}

// BackupResponse is the response for backup operations
type BackupResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
}

// SelectDirResponse is the response for POST /api/files/select-dir
type SelectDirResponse struct {
	Path string `json:"path"`
}

// BrowseResponse is the response for GET /api/files/browse
type BrowseResponse struct {
	Path  string   `json:"path"`
	Dirs  []string `json:"dirs"`
	Files []string `json:"files"`
}

// SettingsConfirmResponse is returned when timeSlot deletion might affect songs
type SettingsConfirmResponse struct {
	ConfirmNeeded bool   `json:"confirmNeeded"`
	AffectedCount int    `json:"affectedCount"`
	SnapshotFile  string `json:"snapshotFilename,omitempty"`
	Success       bool   `json:"success,omitempty"`
}

// ImportResult is the response for POST /api/songs/import
type ImportResult struct {
	Inserted int           `json:"inserted"`
	Skipped  int           `json:"skipped"`
	Errors   []ImportError `json:"errors"`
}

// ImportError represents a single import error
type ImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// UpdateCheckResponse is the response for GET /api/check-update
type UpdateCheckResponse struct {
	HasUpdate     bool   `json:"hasUpdate"`
	LatestVersion string `json:"latestVersion,omitempty"`
	DownloadURL   string `json:"downloadUrl,omitempty"`
}

// SnapshotInfo represents a snapshot file
type SnapshotInfo struct {
	Filename string `json:"filename"`
	Time     string `json:"time"`
	Size     int64  `json:"size"`
}
