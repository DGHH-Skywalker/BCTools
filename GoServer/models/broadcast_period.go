package models

import "time"

// Historical creation timestamps are only a fallback. Explicit periods always win.
// This school's schedule uses China Standard Time, independent of OS/client zones.
func BroadcastPeriod(created time.Time) string {
	if !created.IsZero() && created.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Hour() < 12 {
		return "noon"
	}
	return "afternoon"
}
