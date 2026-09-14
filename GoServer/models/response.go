package models

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error code and message
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse is a simple success response
type SuccessResponse struct {
	Success bool `json:"success"`
}

// FileProcessResponse is the response for file processing
type FileProcessResponse struct {
	TempFileName string `json:"tempFileName"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
}

// DeletedSongLog represents a deleted song entry in the log
type DeletedSongLog struct {
	OriginalID    int64  `json:"originalId"`
	Date          string `json:"date"`
	Title         string `json:"title"`
	TimeSlotID    string `json:"timeSlotId"`
	TimeSlotLabel string `json:"timeSlotLabel"`
	DeletedAt     string `json:"deletedAt"`
}

// ExportWeekResponse is the response for POST /api/files/export-week
type ExportWeekResponse struct {
	ConfirmNeeded bool     `json:"confirmNeeded,omitempty"`
	ExistingFiles []string `json:"existingFiles,omitempty"`
	TargetDir     string   `json:"targetDir,omitempty"`
	FileCount     int      `json:"fileCount,omitempty"`
}
