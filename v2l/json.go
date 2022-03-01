package v2l

import (
	"encoding/json"
	"fmt"
)

// AssetPath - minimal information about an asset
type AssetPath struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type Channel struct {
	// Unique name
	Name string `json:"name"`
	// Exact millisecond duration of all video GoPs in all assets
	GopDurMS int64 `json:"gopDurMS"`
	// How many GoPs to include in an average output segment
	NrGopsPerSeg int64 `json:"nrGopsPerSegment"`
	// ID of asset defining the valid tracks, bitrates etc
	MasterAssetID string `json:"masterAssetID"`
	// Start time relative epoch (1970-01-01) in seconds
	StartTimeS int64 `json:"startTimeS"`
	// Loop schedule or not
	DoLoop bool `json:"doLoop"`
	// The current scedule of the channel
	Schedule *Schedule `json:"schedule"`
	// LastSCTEEventID - internal book keeping for incrementing SCTE Event ID
	LastSCTEEventID int64 `json:"-"`
	// SlidingWindowNrGops - internal constant for how long sliding window to use
	SlidingWindowNrGops int64 `json:"-"`
	// FutureScheduleNrGops - threshold for when to add future entries to schedule
	FutureScheduleNrGops int64 `json:"-"`
}

type Schedule struct {
	GopNrAtScheduleStart int64   `json:"gopNrAtScheduleStart"`
	GopNrAfterLastAd     int64   `json:"gopNrAfterLastAd"`
	Entries              []Entry `json:"entries"` // list of programs or other entries
}

// Entry is a specific entry based on an asset.
// Negative offsets (counting from end) are replaced with positive during config parsing
// Zero Len is replaced with length in Gops until end of asset during config parsing.
type Entry struct {
	// Name to include in EPG
	Name string `json:"name"`
	// Asset identifier
	AssetID string `json:"assetID"`
	// Zero-based GoP nr to start in asset. Negative value means from end
	Offset int64 `json:"offset"`
	// How many GoPs to play in asset. 0 is until end of asset. Beyond end results in wrap to start
	Len int64 `json:"length"`
	// SCTE-35 Event ID in SCTE message. A non-zero value signals an ad
	SCTEEventID int64 `json:"scteEventID"`
}

func printJSON(name string, data interface{}) {
	raw, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("%s: %s\n", name, string(raw))
}
