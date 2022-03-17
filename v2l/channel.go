package v2l

import (
	"encoding/json"
	"log"
	"time"
)

// CreateChannel - create a channel with two assets and an ad in between
func CreateChannel(server, chName, contentTemplatePath string, gopDurMS, nrGopsPerSegment, slidingWindowNrGops, futureScheduleNrGops int64,
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
	channel := Channel{
		Name:                 chName,
		GopDurMS:             gopDurMS,
		NrGopsPerSeg:         nrGopsPerSegment,
		ContentTemplatePath:  contentTemplatePath,
		StartTimeS:           0,     // All times are counted from 1970-01-01
		DoLoop:               false, // Do not loop
		Schedule:             &schedule,
		SlidingWindowNrGops:  slidingWindowNrGops,
		FutureScheduleNrGops: futureScheduleNrGops,
	}
	_, err := uploadJSON(server, "POST", "/api/v1/channels", channel)
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
	channel.SlidingWindowNrGops = 1
	return &channel, nil
}

// nowToSegNr - calculate what segment was last produced
func nowToSegNr(gopDurMS, nrGopsPerSegment int64) int64 {
	nowMS := time.Now().UnixNano() / 1_000_000
	return nowMS/(gopDurMS*nrGopsPerSegment) - 1
}
