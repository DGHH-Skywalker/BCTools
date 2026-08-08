package validation

import "testing"

func TestIsValidDate(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"2026-07-25", true},
		{"2026-02-29", false},
		{"2024-02-29", true},
		{"2026-13-01", false},
		{"2026-07-32", false},
		{"not-a-date", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidDate(c.input); got != c.want {
			t.Errorf("IsValidDate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestIsValidTime(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"06:20", true},
		{"23:59", true},
		{"00:00", true},
		{"24:00", false},
		{"12:60", false},
		{"6:20", false},
	}
	for _, c := range cases {
		if got := IsValidTime(c.input); got != c.want {
			t.Errorf("IsValidTime(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestCalculateWeekday(t *testing.T) {
	cases := []struct {
		date string
		want string
	}{
		{"2026-07-20", "周一"},
		{"2026-07-25", "周六"},
		{"2026-07-26", "周日"},
	}
	for _, c := range cases {
		if got := CalculateWeekday(c.date); got != c.want {
			t.Errorf("CalculateWeekday(%q) = %q, want %q", c.date, got, c.want)
		}
	}
}

func TestCalculateDayIndex(t *testing.T) {
	cases := []struct {
		date string
		want int
	}{
		{"2026-07-20", 1},
		{"2026-07-25", 6},
		{"2026-07-26", 7},
	}
	for _, c := range cases {
		if got := CalculateDayIndex(c.date); got != c.want {
			t.Errorf("CalculateDayIndex(%q) = %d, want %d", c.date, got, c.want)
		}
	}
}

func TestIsValidSongType(t *testing.T) {
	if !IsValidSongType("dorm") {
		t.Error("dorm should be valid")
	}
	if !IsValidSongType("broadcast") {
		t.Error("broadcast should be valid")
	}
	if IsValidSongType("other") {
		t.Error("other should be invalid")
	}
}
