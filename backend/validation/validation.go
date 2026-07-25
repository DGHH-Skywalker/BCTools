package validation

import (
	"regexp"
	"strings"
	"time"
)

var (
	dateRegex    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRegex    = regexp.MustCompile(`^\d{2}:\d{2}$`)
	allowedTypes = map[string]bool{"dorm": true, "broadcast": true}
)

// IsValidDate checks if the string is a valid YYYY-MM-DD date
func IsValidDate(dateStr string) bool {
	if !dateRegex.MatchString(dateStr) {
		return false
	}
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// IsValidTime checks if the string is a valid HH:mm time
func IsValidTime(timeStr string) bool {
	if !timeRegex.MatchString(timeStr) {
		return false
	}
	parts := strings.Split(timeStr, ":")
	h, m := parts[0], parts[1]
	return h >= "00" && h <= "23" && m >= "00" && m <= "59"
}

// IsValidSongType checks if type is dorm or broadcast
func IsValidSongType(songType string) bool {
	return allowedTypes[songType]
}

// SanitizeString trims whitespace and limits length
func SanitizeString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

// IsValidMode checks if mode is copy or move
func IsValidMode(mode string) bool {
	return mode == "copy" || mode == "move"
}

// CalculateWeekday calculates weekday string from date
func CalculateWeekday(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	dayIndex := int(t.Weekday())
	if dayIndex == 0 {
		dayIndex = 7
	}
	weekdays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	return weekdays[dayIndex-1]
}

// CalculateDayIndex returns 1-7 from a date string
func CalculateDayIndex(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	dayIndex := int(t.Weekday())
	if dayIndex == 0 {
		dayIndex = 7
	}
	return dayIndex
}
