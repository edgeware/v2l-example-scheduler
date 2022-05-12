package v2l

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

// CreateChannel - create a channel with two assets and an ad in between
func CreateChannel(server, chName, contentTemplatePath string,
	gopDurMS, nrGopsPerSegment, slidingWindowNrGops, futureScheduleNrGops int64,
	assetPaths []AssetPath) (*Channel, error) {
	startGopNr := nowToSegNr(gopDurMS, nrGopsPerSegment) * nrGopsPerSegment
	log.Printf("Start time for channels set to %s\n", time.Duration(startGopNr*gopDurMS)*(time.Millisecond))

	schedule := Schedule{
		GopNrAtScheduleStart: startGopNr,
		GopNrAfterLastAd:     0,
		Entries: []Entry{
			randomEntry(assetPaths, "program", 0, 5, 0),
			randomEntry(assetPaths, "ad", 0, 5, 1),
			//randomEntry(assetPaths, "program", 0, 5, 0),
		},
	}
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
		Schedule:             &schedule,
		SlidingWindowNrGops:  slidingWindowNrGops,
		FutureScheduleNrGops: futureScheduleNrGops,
	}
	_, err = uploadJSON(server, "POST", "/api/v1/channels", channel)
	if err != nil {
		return nil, err
	}
	// Next ask for the schedule to get it back with filled in lengths of the assets
	respBody, err := httpRequest(server, "GET", "/api/v1/schedule/"+chName, nil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respBody, &schedule)
	if err != nil {
		return nil, err
	}
	printJSON("schedule", &schedule)
	channel.Schedule = &schedule
	channel.LastSCTEEventID = 1
	return &channel, nil
}

// CreateChannels - create a slice filled with channels
func CreateChannels(nofChannels int,
	server, contentTemplatePath string,
	gopDurMS, nrGopsPerSegment, slidingWindowNrGops, futureScheduleNrGops int64,
	assetPaths []AssetPath) ([]*Channel, error) {
	var channels []*Channel
	for ch := 1; ch <= nofChannels; ch++ {
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

// DeleteChannels - delete a number of channels
func DeleteChannels(nofChannels int, server string) error {
	for i := 1; i <= nofChannels; i++ {
		channelName := fmt.Sprintf("ch%d", i)
		err := DeleteChannel(server, channelName)
		if err != nil {
			return err
		}
	}
	return nil
}

// nowToSegNr - calculate what segment was last produced
func nowToSegNr(gopDurMS, nrGopsPerSegment int64) int64 {
	nowMS := time.Now().UnixNano() / 1_000_000
	return nowMS/(gopDurMS*nrGopsPerSegment) - 1
}
