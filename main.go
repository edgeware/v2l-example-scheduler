package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/edgeware/v2l-example-scheduler/v2l"
)

const (
	assetDir             = "assets"
	server               = "http://localhost:8090"
	channelName          = "ch1"
	gopDurMS             = 2000      // Note, all content must have this same GoP duration
	nrGopsPerSegment     = 2         // Decides how long output segments will be in average
	slidingWindowNrGops  = 40        // Should be at least a minute before removing stuff
	futureScheduleNrGops = 15        // Threshold for when to add new entries in schedule
	masterAssetID        = "bbb_40s" // Asset from which content_info is fetched
	updatePeriodS        = 2         // How often the schedule should be checked for updates in seconds
)

func main() {
	assetPaths, err := v2l.GetAssetPaths(assetDir)
	if err != nil {
		log.Fatal(err)
	}

	err = v2l.AddAssetPaths(server, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	err = v2l.DeleteChannel(server, channelName) // Delete any old channel and schedule
	if err != nil {
		log.Fatal(err)
	}

	// Create channel with a few assets and get state back
	channel, err := v2l.CreateChannel(server, channelName, masterAssetID, gopDurMS, nrGopsPerSegment,
		slidingWindowNrGops, futureScheduleNrGops, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	ticker := time.NewTicker(updatePeriodS * time.Second)
TickerLoop:
	for {
		select {
		case <-signalCh:
			fmt.Printf("Stopping loop\n")
			break TickerLoop
		case t := <-ticker.C:
			err = v2l.UpdateSchedule(server, channel, assetPaths, t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
