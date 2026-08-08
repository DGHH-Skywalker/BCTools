package store

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrValidation    = errors.New("validation error")
	ErrDataCorrupted = errors.New("data corrupted")
	ErrSlotConflict  = errors.New("time slot already has a song")
)
