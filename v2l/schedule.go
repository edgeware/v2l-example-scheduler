package v2l

import (
	"encoding/json"
	"fmt"
	"time"
)

// updateSchedule - update schedule by removing old entries and adding new
// for old ones, the limit is now - sliding window
// for new ones, a new asset or ad will be added if within 30s of end
// of schedule.
// GopNrAtScheduleStart and GopNrAfterLastAd must have consistent values.
func UpdateSchedule(server string, channel *Channel, assetPaths []AssetPath, now time.Time) error {
	scheduleChanged := false
	lastSch := channel.Schedule
	nowGopNr := nowToGopNr(channel.GopDurMS, now)
	newSch := Schedule{
		GopNrAtScheduleStart: lastSch.GopNrAtScheduleStart, // Starting point
		GopNrAfterLastAd:     lastSch.GopNrAfterLastAd,
		Entries:              nil,
	}
	nextEntryStart := lastSch.GopNrAtScheduleStart
	var nextDrop int64
	for i, e := range lastSch.Entries {
		entryEnd := nextEntryStart + (e.Len - e.Offset)
		if i == 0 {
			nextDrop = entryEnd - (nowGopNr - channel.SlidingWindowNrGops) + 1
		}
		if entryEnd < nowGopNr-channel.SlidingWindowNrGops {
			// Drop this entry
			newSch.GopNrAtScheduleStart += e.Len
			if e.SCTEEventID > 0 {
				newSch.GopNrAfterLastAd = entryEnd
			}
			nextEntryStart = entryEnd
			scheduleChanged = true
			fmt.Printf("Removed %s from schedule\n", e.AssetID)
			continue
		}
		newSch.Entries = append(newSch.Entries, e)
		nextEntryStart += e.Len
	}
	nextAdd := nextEntryStart - channel.FutureScheduleNrGops - nowGopNr + 1
	if nowGopNr > nextEntryStart-channel.FutureScheduleNrGops {
		// Time to add an add, program etc
		lastEntry := newSch.Entries[len(newSch.Entries)-1]
		if lastEntry.SCTEEventID > 0 { // Last entry is an ad
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "program", 0, 0, 0))
		} else {
			channel.LastSCTEEventID++
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "ad", 0, 0, channel.LastSCTEEventID))
		}
		newEntry := newSch.Entries[len(newSch.Entries)-1]
		fmt.Printf("Added %s with SCTE id %d to schedule\n", newEntry.AssetID, newEntry.SCTEEventID)
		scheduleChanged = true
	}
	if scheduleChanged {
		printJSON("schedule to upload", newSch)
		respBody, err := uploadJSON(server, "PUT", "/api/v1/schedule/"+channel.Name, &newSch)
		if err != nil {
			return fmt.Errorf("problem uploading schedule: %s\n", err)
		}
		respSchedule := Schedule{}
		err = json.Unmarshal(respBody, &respSchedule)
		if err != nil {
			return err
		}
		printJSON("responded schedule", &respSchedule)
		channel.Schedule = &respSchedule
	} else {
		fmt.Printf("No schedule change at %s. Next add/drop in %d/%d GoPs\n", now, nextAdd, nextDrop)
	}
	return nil
}

// nowToGopNr - what GoP is currently being produced
func nowToGopNr(gopDurMS int64, now time.Time) int64 {
	nowMS := now.UnixNano() / 1_000_000
	return nowMS / gopDurMS
}
