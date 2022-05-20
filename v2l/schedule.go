package v2l

import (
	"time"
)

func (schedule *Schedule) LastSCTEEventID() int64 {
	entries := schedule.Entries
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].SCTEEventID != 0 {
			return entries[i].SCTEEventID
		}
	}
	return 0
}

// nowToGopNr - what GoP is currently being produced
func nowToGopNr(gopDurMS int64, now time.Time) int64 {
	nowMS := now.UnixNano() / 1_000_000
	return nowMS / gopDurMS
}
