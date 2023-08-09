package v2l

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"
)

const (
	LIVE_NAME   = "live"
	LIVE_LENGTH = 1_000_000 // Don't limit
)

// AssetPath - minimal information about an asset
type AssetPath struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	IsLive bool   `json:"isLive,omitempty"`
}

// DeleteAllAssetPaths - delete all asset paths from server
func DeleteAllAssetPaths(server string) error {
	respBody, err := httpRequest(server, "GET", "/api/v1/assetpaths", nil)
	if err != nil {
		return err
	}
	var assetPaths []AssetPath
	err = json.Unmarshal(respBody, &assetPaths)
	if err != nil {
		return err
	}
	assetIDs := make([]string, 0, len(assetPaths))
	for _, ap := range assetPaths {
		assetIDs = append(assetIDs, ap.ID)
	}
	reqBody, err := json.Marshal(assetIDs)
	if err != nil {
		return err
	}
	aIDsBuf := bytes.NewBuffer(reqBody)
	_, err = httpRequest(server, "DELETE", "/api/v1/assetpaths", aIDsBuf)
	return err
}

// AddAssetPaths - add all directories containing a content_info.json file
func AddAssetPaths(server string, assetPaths []AssetPath) error {
	_, err := uploadJSON(server, "POST", "/api/v1/assetpaths", assetPaths)
	return err
}

// DiscoverVodAssetPaths - add all directories containing a content_info.json file
func DiscoverVodAssetPaths(dir string) ([]AssetPath, error) {
	var aps []AssetPath
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if name == "content_info.json" {
			absAssetPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			assetPath := filepath.Dir(absAssetPath) // The dir containing content_info.json
			assetName := filepath.Base(assetPath)
			aps = append(aps, AssetPath{assetName, assetPath, false})
		}
		return nil
	})
	return aps, err
}

// randomEntry - return a random entry given kind and assetPaths. Set offset, length, sctedID
func randomEntry(assetPaths []AssetPath, kind string, offset, length, scteID int64) Entry {
	var programs []string
	var ads []string
	var fillers []string
	var slates []string
	for _, ap := range assetPaths {
		switch {
		case strings.Contains(ap.Path, "/filler"):
			fillers = append(fillers, ap.ID)
		case strings.Contains(ap.Path, "/slates/"):
			fillers = append(fillers, ap.ID)
		case strings.Contains(ap.Path, "/ads/"):
			ads = append(ads, ap.ID)
		case strings.Contains(ap.Path, "/programs"):
			//TODO. Change back. Right now sintel represents live
			if !strings.Contains(ap.Path, "sintel") {
				programs = append(programs, ap.ID)
			}

		default:
			// Just drop
		}
	}
	var assetID string
	switch kind {
	case "filler":
		idx := rand.Intn(len(fillers))
		assetID = fillers[idx]
	case "slates":
		idx := rand.Intn(len(slates))
		assetID = slates[idx]
	case "program":
		idx := rand.Intn(len(programs))
		assetID = programs[idx]
	case "ad":
		idx := rand.Intn(len(ads))
		assetID = ads[idx]
	default:
		panic("Unknown kind of asset")
	}
	return Entry{
		Name:        assetID,
		AssetID:     assetID,
		Offset:      offset,
		Len:         length,
		SCTEEventID: scteID,
	}
}
