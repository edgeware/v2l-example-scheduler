package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/edgeware/v2l-example-scheduler/v2l"
)

const (
	assetDir             = "assets"
	server               = "http://localhost:8090"
	channelName          = "ch1"
	gopDurMS             = 2000                    // Note, all content must have this same GoP duration
	nrGopsPerSegment     = 2                       // Decides how long output segments will be in average
	slidingWindowNrGops  = 40                      // Should be at least a minute before removing stuff
	futureScheduleNrGops = 15                      // Threshold for when to add new entries in schedule
	contentTemplatePath  = "content_template.json" // Template describing input and output
	updatePeriodS        = 2                       // How often the schedule should be checked for updates in seconds
)

func main() {
	nofChannels := 1
	var err error
	if len(os.Args) > 1 {
		nofChannels, err = strconv.Atoi(os.Args[1])
		if err != nil {
			println("Usage: ", os.Args[0], "[<number-of-channels>] default:1")
			os.Exit(1)
		}
	}

	err = v2l.DeleteChannels(nofChannels, server) // Delete any old channel and schedule
	if err != nil {
		log.Fatal(err)
	}

	err = v2l.DeleteAllAssetPaths(server) // Now, when assets are not used, they can be deleted
	if err != nil {
		log.Print(err.Error())
	}

	assetPaths, err := v2l.DiscoverAssetPaths(assetDir)
	if err != nil {
		log.Fatal(err)
	}

	err = v2l.AddAssetPaths(server, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	// Create channel with a few assets and get state back
	channels, err := v2l.CreateChannels(nofChannels, server, contentTemplatePath, gopDurMS, nrGopsPerSegment,
		slidingWindowNrGops, futureScheduleNrGops, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	ticker := time.NewTicker(updatePeriodS * time.Second / time.Duration(nofChannels))
	chIndex := 0
TickerLoop:
	for {
		select {
		case <-signalCh:
			log.Printf("Stopping loop\n")
			break TickerLoop
		case t := <-ticker.C:
			err = v2l.UpdateSchedule(server, channels[chIndex%nofChannels], assetPaths, t)
			if err != nil {
				log.Fatal(err)
			}
		}
		chIndex++
	}
}
