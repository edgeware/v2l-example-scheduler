package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/edgeware/v2l-example-scheduler/v2l"
)

const (
	assetDir             = "assets"
	server               = "http://localhost:8090"
	channelName          = "ch1"
	gopDurMS             = 2000                    // Note, all content must have this same GoP duration
	nrGopsPerSegment     = 1                       // Decides how long output segments will be in average
	slidingWindowNrGops  = 40                      // Should be at least a minute before removing stuff
	futureScheduleNrGops = 15                      // Threshold for when to add new entries in schedule
	contentTemplatePath  = "content_template.json" // Template describing input and output
	updatePeriodS        = 2                       // How often the schedule should be checked for updates in seconds
	apiPort              = 8888
	liveSource           = "http://localhost:8091/live/"
)

func main() {
	err := v2l.DeleteChannel(server, channelName) // Delete any old channel and schedule
	if err != nil {
		log.Fatal(err)
	}

	err = v2l.DeleteAllAssetPaths(server) // Now, when assets are not used, they can be deleted
	if err != nil {
		log.Print(err.Error())
	}

	assetPaths, err := v2l.DiscoverVodAssetPaths(assetDir)
	if err != nil {
		log.Fatal(err)
	}

	if liveSource != "" {
		assetPaths = append(assetPaths, v2l.AssetPath{ID: v2l.LIVE_NAME, Path: liveSource, IsLive: true})
	}

	err = v2l.AddAssetPaths(server, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	// Create channel with a few assets and get state back
	channel, err := v2l.CreateChannel(server, channelName, contentTemplatePath, gopDurMS, nrGopsPerSegment,
		slidingWindowNrGops, futureScheduleNrGops, assetPaths)
	if err != nil {
		log.Fatal(err)
	}

	isLive := false
	isLiveCh := make(chan bool, 1)

	apiServer := v2l.SetupServer(apiPort, isLiveCh)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", apiServer.Port),
		Handler: apiServer.Router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	ticker := time.NewTicker(updatePeriodS * time.Second)
TickerLoop:
	for {
		select {
		case newLiveState := <-isLiveCh:
			isLive = newLiveState
			fmt.Printf("isLive=%t\n", isLive)
		case <-signalCh:
			log.Printf("Stopping loop\n")
			break TickerLoop
		case t := <-ticker.C:
			err = v2l.UpdateSchedule(server, channel, assetPaths, t, liveSource, isLive)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
