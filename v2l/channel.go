package v2l

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

// UpdateSchedule - update schedule by removing old entries and adding new
//
// for old ones, the limit is now - sliding window
// for new ones, a new asset or ad will be added if within 30s of end
// of schedule.
// GopNrAtScheduleStart and GopNrAfterLastAd must have consistent values.
func (channel *Channel) UpdateSchedule(server string, assetPaths []AssetPath, now time.Time) error {
	shouldLog := false
	if channel.Name == "ch1" {
		shouldLog = true
	}

	scheduleChanged := false
	lastSch := channel.Schedule
	nowGopNr := nowToGopNr(channel.GopDurMS, now)
	newSch := Schedule{
		GopNrAtScheduleStart: lastSch.GopNrAtScheduleStart, // Starting point
		GopNrAfterLastAd:     lastSch.GopNrAfterLastAd,
		Entries:              nil,
	}
	nextEntryStart := lastSch.GopNrAtScheduleStart
	var gopsUntilNextDrop int64
	for i, e := range lastSch.Entries {
		nextEntryStart += e.Len // Now start of the one after this
		if i == 0 {
			gopsUntilNextDrop = nextEntryStart - (nowGopNr - channel.SlidingWindowNrGops)
			if gopsUntilNextDrop <= 0 { // Drop this entry
				newSch.GopNrAtScheduleStart += e.Len
				if e.SCTEEventID > 0 {
					newSch.GopNrAfterLastAd = nextEntryStart
				}
				scheduleChanged = true
				if shouldLog {
					log.Printf("Removed %s from schedule\n", e.AssetID)
				}
				continue
			}
		}
		newSch.Entries = append(newSch.Entries, e)
	}
	gopsUntilNextAdd := nextEntryStart - channel.FutureScheduleNrGops - nowGopNr
	if gopsUntilNextAdd <= 0 { // Time to add an add, program etc
		lastEntry := newSch.Entries[len(newSch.Entries)-1]
		if lastEntry.SCTEEventID > 0 { // Last entry is an ad
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "program", 0))
		} else {
			channel.LastSCTEEventID++
			newSch.Entries = append(newSch.Entries, randomEntry(assetPaths, "ad", channel.LastSCTEEventID))
		}
		newEntry := newSch.Entries[len(newSch.Entries)-1]
		if shouldLog {
			log.Printf("Added %s with SCTE id %d to schedule\n", newEntry.AssetID, newEntry.SCTEEventID)
		}
		scheduleChanged = true
	}
	if scheduleChanged {
		if shouldLog {
			printJSON("schedule to upload", newSch)
		}

		respBody, err := uploadJSON(server, "PUT", "/api/v1/schedule/"+channel.Name, &newSch)
		if err != nil {
			return fmt.Errorf("problem uploading schedule: %s", err)
		}
		respSchedule := Schedule{}
		err = json.Unmarshal(respBody, &respSchedule)
		if err != nil {
			return err
		}
		if shouldLog {
			printJSON("responded schedule", &respSchedule)
		}
		channel.Schedule = &respSchedule
	} else {
		if shouldLog {
			log.Printf("No schedule change. Next add/drop in %d/%d GoPs\n", gopsUntilNextAdd, gopsUntilNextDrop)
		}
	}
	return nil
}

// CreateSchedule -- create a complete schedule to fill the entire live window
// the schedule is filled with asset pairs of types: "program" & "ad"
func (channel *Channel) CreateSchedule(slidingWindowNrGops, futureScheduleNrGops, gopDurMS int64, assetPaths []AssetPath) {
	nowGopNr := nowToGopNr(gopDurMS, time.Now())
	startGopNr := nowGopNr - slidingWindowNrGops - 1
	latestGopNr := nowGopNr + futureScheduleNrGops + 1

	startTime := time.Unix(int64(startGopNr*gopDurMS/1000), 0)
	log.Printf("New schedule\n    gopNrAtScheduleStart: %d\n    Time: %s", startGopNr, startTime)
	channel.Schedule = &Schedule{
		GopNrAtScheduleStart: startGopNr,
		GopNrAfterLastAd:     0,
		Entries:              []Entry{},
	}
	currGopNr := startGopNr
	channel.LastSCTEEventID = 0
	for {
		progEntry := randomEntry(assetPaths, "program", 0)
		channel.Schedule.Entries = append(channel.Schedule.Entries, progEntry)
		currGopNr += progEntry.Len

		channel.LastSCTEEventID++
		adEntry := randomEntry(assetPaths, "ad", channel.LastSCTEEventID)
		channel.Schedule.Entries = append(channel.Schedule.Entries, adEntry)
		currGopNr += adEntry.Len

		if currGopNr >= latestGopNr {
			break
		}
	}
}

// CreateChannel - create a channel with a fully populated schedule
func CreateChannel(server, chName, contentTemplatePath string,
	gopDurMS, nrGopsPerSegment, slidingWindowNrGops, futureScheduleNrGops int64,
	assetPaths []AssetPath) (*Channel, error) {
	startGopNr := nowToGopNr(gopDurMS, time.Now())

	absContentTemplatePath, err := filepath.Abs(contentTemplatePath)
	if err != nil {
		return nil, err
	}
	channel := Channel{
		Name:                 chName,
		GopDurMS:             gopDurMS,
		NrGopsPerSeg:         nrGopsPerSegment,
		ContentTemplatePath:  absContentTemplatePath,
		StartTimeS:           0,     // All times are counted from 1970-01-01
		DoLoop:               false, // Do not loop
		SlidingWindowNrGops:  slidingWindowNrGops,
		FutureScheduleNrGops: futureScheduleNrGops,
	}

	channel.CreateSchedule(slidingWindowNrGops, futureScheduleNrGops, gopDurMS, assetPaths)

	_, err = uploadJSON(server, "POST", "/api/v1/channels", channel)
	if err != nil {
		return nil, err
	}
	// Next ask for the schedule to get it back with filled in lengths of the assets
	respBody, err := httpRequest(server, "GET", "/api/v1/schedule/"+chName, nil)
	if err != nil {
		return nil, err
	}

	var serverSchedule Schedule
	err = json.Unmarshal(respBody, &serverSchedule)
	if err != nil {
		return nil, err
	}
	log.Printf("Channel: %s, Schedule entries: %d", chName, len(serverSchedule.Entries))
	channel.Schedule = &serverSchedule
	if channel.LastSCTEEventID != serverSchedule.LastSCTEEventID() {
		log.Printf("Expected LastSCTEEventID %d, vod2cbm returned: %d [channel: %s]",
			channel.LastSCTEEventID,
			serverSchedule.LastSCTEEventID(),
			chName)
	}
	return &channel, nil
}

// CreateChannels - create a slice filled with channels
func CreateChannels(nrChannels int,
	server, contentTemplatePath string,
	gopDurMS, nrGopsPerSegment, slidingWindowNrGops, futureScheduleNrGops int64,
	assetPaths []AssetPath) ([]*Channel, error) {
	var channels []*Channel
	for ch := 1; ch <= nrChannels; ch++ {
		channelName := fmt.Sprintf("ch%d", ch)
		// Create channel with a few assets and get state back
		channel, err := CreateChannel(server, channelName, contentTemplatePath,
			gopDurMS, nrGopsPerSegment,
			slidingWindowNrGops, futureScheduleNrGops,
			assetPaths)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

// DeleteChannel - delete a channel
func DeleteChannel(server, channelName string) error {
	_, err := httpRequest(server, "DELETE", "/api/v1/channels/"+channelName, nil)
	return err
}

// DeleteChannels - delete a number of channels: "ch1"..."chn"
func DeleteChannels(nrChannels int, server string) error {
	for i := 1; i <= nrChannels; i++ {
		channelName := fmt.Sprintf("ch%d", i)
		err := DeleteChannel(server, channelName)
		if err != nil {
			return err
		}
	}
	return nil
}
