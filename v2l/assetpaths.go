package v2l

import (
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"
)

// addAssetPaths - add all directories containing a content_info.json file
func AddAssetPaths(server string, assetPaths []AssetPath) error {
	_, err := uploadJSON(server, "POST", "/api/v1/assetpaths", assetPaths)
	return err
}

func GetAssetPaths(dir string) ([]AssetPath, error) {
	var aps []AssetPath
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if name == "content_info.json" {
			assetPath := filepath.Dir(path) // The dir containing content_info.json
			assetName := filepath.Base(assetPath)
			aps = append(aps, AssetPath{assetName, assetPath})
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
		if strings.Contains(ap.Path, "/filler") {
			fillers = append(fillers, ap.ID)
			continue
		}
		if strings.Contains(ap.Path, "/slates/") {
			fillers = append(fillers, ap.ID)
			continue
		}
		if strings.Contains(ap.Path, "/ads/") {
			ads = append(ads, ap.ID)
			continue
		}
		programs = append(programs, ap.ID)
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
