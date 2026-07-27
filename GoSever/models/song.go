package models

import "time"

// Song represents a song entry in the playlist
type Song struct {
	ID         int64     `json:"id"`
	Type       string    `json:"type,omitempty"` // not persisted, determined by array
	Date       string    `json:"date"`
	Weekday    string    `json:"weekday,omitempty"` // computed, not persisted
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Remark     string    `json:"remark"`
	FilePath   string    `json:"filePath"`
	TimeSlotID *string   `json:"timeSlotId"` // null = unassigned
	CreatedAt  time.Time `json:"createdAt"`
}

// UpdateSongRequest contains only the fields to update
type UpdateSongRequest struct {
	Date       *string  `json:"date,omitempty"`
	Title      *string  `json:"title,omitempty"`
	Artist     *string  `json:"artist,omitempty"`
	Remark     *string  `json:"remark,omitempty"`
	FilePath   *string  `json:"filePath,omitempty"`
	TimeSlotID **string `json:"timeSlotId,omitempty"` // double pointer: nil=not set, *nil=set to null
}

// CreateSongRequest contains fields for creating a new song
type CreateSongRequest struct {
	Type       string  `json:"type"`
	Date       string  `json:"date"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist,omitempty"`
	Remark     string  `json:"remark,omitempty"`
	FilePath   string  `json:"filePath,omitempty"`
	TimeSlotID *string `json:"timeSlotId,omitempty"`
}
