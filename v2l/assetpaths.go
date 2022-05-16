package v2l

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgeware/sbgo/pkg/esf"
)

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

// DiscoverAssetPaths - add all directories containing a content_info.json file
func DiscoverAssetPaths(dir string) ([]AssetPath, error) {
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
			assetLen, err := getAssetLen(absAssetPath)
			if err != nil {
				return err
			}
			aps = append(aps, AssetPath{assetName, assetPath, assetLen})
		}
		return nil
	})
	return aps, err
}

// randomEntry - return a random entry given kind and assetPaths. Set offset, length, sctedID
func randomEntry(assetPaths []AssetPath, kind string, scteID int64) Entry {
	var selectedAssets []AssetPath
	var subPath = "/" + kind + "s/"
	for _, ap := range assetPaths {
		if strings.Contains(ap.Path, subPath) {
			selectedAssets = append(selectedAssets, ap)
		}
	}

	if len(selectedAssets) == 0 {
		panic("No  such asset kind: " + kind)
	}

	asset := selectedAssets[rand.Intn(len(selectedAssets))]

	return Entry{
		Name:        asset.ID,
		AssetID:     asset.ID,
		Offset:      0,
		Len:         asset.len,
		SCTEEventID: scteID,
	}
}

// getAssetLen -- get length in number of GoPs
func getAssetLen(assetPath string) (int64, error) {
	bytes, err := os.ReadFile(assetPath)
	if err != nil {
		return 0, err
	}

	ci, err := esf.ParseContentInfo(bytes)
	if err != nil {
		return 0, err
	}

	cd := ci.ContentDurationMS
	if cd == 0 {
		return 0, fmt.Errorf("ContentDurationMS not found")
	}

	gd := ci.GOPDurationMS
	if gd == 0 {
		return 0, fmt.Errorf("GOPDurationMS not found")
	}

	return cd / gd, nil
}
