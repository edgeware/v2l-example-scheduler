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
	assetDir                   = "assets"
	server                     = "http://localhost:8090"
	channelName                = "ch1"
	gopDurMS                   = 2000                    // Note, all content must have this same GoP duration
	nrGopsPerSegment           = 2                       // Decides how long output segments will be in average
	slidingWindowNrGopsDefault = 40                      // Should be at least a minute before removing stuff
	futureScheduleNrGops       = 15                      // Threshold for when to add new entries in schedule
	contentTemplatePath        = "content_template.json" // Template describing input and output
	updatePeriodS              = 2                       // How often the schedule should be checked for updates in seconds
)

func main() {
	nrChannels := 1
	var slidingWindowNrGops int64 = slidingWindowNrGopsDefault
	var err error
	if len(os.Args) > 1 {
		nrChannels, err = strconv.Atoi(os.Args[1])
		if err != nil {
			printUsage()
		}
	}

	if len(os.Args) > 2 {
		slidingWindowS, err := strconv.Atoi(os.Args[2])
		if err != nil {
			printUsage()
		}
		slidingWindowNrGops = int64(slidingWindowS) * 1000 / gopDurMS
	}

	err = v2l.DeleteChannels(nrChannels, server) // Delete any old channel and schedule
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
	channels, err := v2l.CreateChannels(nrChannels, server, contentTemplatePath, gopDurMS, nrGopsPerSegment,
		slidingWindowNrGops, futureScheduleNrGops, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	ticker := time.NewTicker(updatePeriodS * time.Second / time.Duration(nrChannels))
	chIndex := 0
TickerLoop:
	for {
		select {
		case <-signalCh:
			log.Printf("Stopping loop\n")
			break TickerLoop
		case t := <-ticker.C:
			err = channels[chIndex].UpdateSchedule(server, assetPaths, t)
			if err != nil {
				log.Fatal(err)
			}
		}
		chIndex++
		if chIndex == nrChannels {
			chIndex = 0
		}
	}
}

func printUsage() {
	println("Usage: ", os.Args[0], "[<number-of-channels>] [<sliding-window-S>] defaults:1 channel, 80 sec ")
	os.Exit(1)
}
