package models

import (
	"testing"
	"time"
)

func TestBroadcastPeriodShanghai(t *testing.T) {
	for _, tc := range []struct{ timestamp, want string }{
		{"2026-09-18T03:59:59Z", "noon"},
		{"2026-09-18T04:00:00Z", "afternoon"},
		{"2026-09-18T15:00:00+08:00", "afternoon"},
		{"2026-09-17T16:00:00Z", "noon"},
	} {
		ts, _ := time.Parse(time.RFC3339, tc.timestamp)
		if got := BroadcastPeriod(ts); got != tc.want {
			t.Fatalf("%s: %s", tc.timestamp, got)
		}
	}
	if BroadcastPeriod(time.Time{}) != "afternoon" {
		t.Fatal("zero timestamp")
	}
}
