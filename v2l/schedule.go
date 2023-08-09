package v2l

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// updateSchedule - update schedule by removing old entries and adding new
// for old ones, the limit is now - sliding window
// for new ones, a new asset or ad will be added if within 30s of end
// of schedule.
// GopNrAtScheduleStart and GopNrAfterLastAd must have consistent values.
func UpdateSchedule(server string, channel *Channel, assetPaths []AssetPath, now time.Time, liveSource string, isLive bool) error {
	scheduleChanged := false
	inputSch := channel.Schedule
	nowGopNr := nowToGopNr(channel.GopDurMS, now)
	lastIsLive := false
	if len(inputSch.Entries) > 0 {
		lastEntry := inputSch.Entries[len(inputSch.Entries)-1]
		if lastEntry.Name == LIVE_NAME {
			lastIsLive = true
		}
	}
	switch {
	case isLive:
		if lastIsLive {
			return nil // Just continue with live
		}
		inputSch = switchToLive(inputSch, nowGopNr)
		scheduleChanged = true
	case !isLive && lastIsLive:
		// Turn off live
		inputSch = stopLive(inputSch, nowGopNr)
		scheduleChanged = true
	default:
		// No live specific stuff
	}

	nextEntryStart := inputSch.GopNrAtScheduleStart
	var gopsUntilNextDrop int64
	newSch := Schedule{
		GopNrAtScheduleStart: inputSch.GopNrAtScheduleStart, // Starting point
		GopNrAfterLastAd:     inputSch.GopNrAfterLastAd,
		Entries:              nil,
	}
	for i, e := range inputSch.Entries {
		nextEntryStart += e.Len // Now start of the one after this
		if i == 0 {
			gopsUntilNextDrop = nextEntryStart - (nowGopNr - channel.SlidingWindowNrGops)
			if gopsUntilNextDrop <= 0 { // Drop this entry
				newSch.GopNrAtScheduleStart += e.Len
				if e.SCTEEventID > 0 {
					newSch.GopNrAfterLastAd = nextEntryStart
				}
				scheduleChanged = true
				log.Printf("Removed %s from schedule\n", e.AssetID)
				continue
			}
		}
		newSch.Entries = append(newSch.Entries, e)
	}
	gopsUntilNextAdd := nextEntryStart - channel.FutureScheduleNrGops - nowGopNr
	if gopsUntilNextAdd <= 0 { // Time to add an add, program etc
		lastEntry := newSch.Entries[len(newSch.Entries)-1]
		if lastEntry.SCTEEventID > 0 { // Last entry is an ad
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "program", 0, 0, 0))
		} else {
			channel.LastSCTEEventID++
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "ad", 0, 0, channel.LastSCTEEventID))
		}
		newEntry := newSch.Entries[len(newSch.Entries)-1]
		log.Printf("Added %s with SCTE id %d to schedule\n", newEntry.AssetID, newEntry.SCTEEventID)
		scheduleChanged = true
	}
	if scheduleChanged {
		//printJSON("schedule to upload", newSch)
		respBody, err := uploadJSON(server, "PUT", "/api/v1/schedule/"+channel.Name, &newSch)
		if err != nil {
			return fmt.Errorf("problem uploading schedule: %s", err)
		}
		respSchedule := Schedule{}
		err = json.Unmarshal(respBody, &respSchedule)
		if err != nil {
			return err
		}
		printJSON("responded schedule", &respSchedule)
		channel.Schedule = &respSchedule
	} /*else {
		log.Printf("No schedule change. Next add/drop in %d/%d GoPs\n", gopsUntilNextAdd, gopsUntilNextDrop)
	}*/
	return nil
}

// nowToGopNr - what GoP is currently being produced
func nowToGopNr(gopDurMS int64, now time.Time) int64 {
	nowMS := now.UnixNano() / 1_000_000
	return nowMS / gopDurMS
}

func switchToLive(sch *Schedule, nowGopNr int64) *Schedule {
	// Start live as soon as possible
	currStart := sch.GopNrAtScheduleStart
	currEnd := currStart
	switchGopNr := nowGopNr + 3
	newSch := Schedule{
		GopNrAtScheduleStart: sch.GopNrAtScheduleStart,
		GopNrAfterLastAd:     sch.GopNrAfterLastAd,
		Entries:              nil,
	}
	for _, e := range sch.Entries {
		currEnd = currStart + e.Len
		if currEnd <= switchGopNr {
			newSch.Entries = append(newSch.Entries, e)
			currStart = currEnd
			continue
		}
		if currStart < switchGopNr {
			// Shorten current asset
			e.Len = switchGopNr - currStart
			newSch.Entries = append(newSch.Entries, e)
		}
		currEnd = switchGopNr
		currStart = currEnd
	}
	// Add live
	liveEntry := Entry{LIVE_NAME, LIVE_NAME, 0, LIVE_LENGTH, 0}
	newSch.Entries = append(newSch.Entries, liveEntry)
	return &newSch
}

func stopLive(sch *Schedule, nowGopNr int64) *Schedule {
	last := len(sch.Entries) - 1
	switchGopNr := nowGopNr + 3
	currStart := sch.GopNrAtScheduleStart
	for _, e := range sch.Entries[:last] {
		currStart += e.Len
	}
	liveLen := switchGopNr - currStart
	sch.Entries[last].Len = liveLen
	return sch
}
