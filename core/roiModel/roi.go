// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package roiModel

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/utils"
	protos "github.com/pixlise/core/generated-protos"
)

// ROIItem - Region of interest item, only public so Go can reflect/interogate it
type ROIItem struct {
	Name            string  `json:"name"`
	LocationIndexes []int32 `json:"locationIndexes"`
	Description     string  `json:"description"`
	ImageName       string  `json:"imageName,omitempty"` // Name of image whose pixels are present in this ROI.
	// If no imageName, it's a traditional ROI consisting of PMCs
	PixelIndexes []int32 `json:"pixelIndexes,omitempty"`
}

// ROISavedItem - Region of interest item as saved to S3, only public so Go can reflect/interogate it
type ROISavedItem struct {
	*ROIItem
	*pixlUser.APIObjectItem
}

type ROILookup map[string]ROISavedItem

type ROIMembers struct {
	Name         string
	ID           string
	SharedByName string
	LocationIdxs []int32
	PMCs         []int32
}

func GetAllPointsROI(dataset *protos.Experiment) ROIMembers {
	result := ROIMembers{
		Name:         "All Points",
		ID:           "",
		SharedByName: "",
		LocationIdxs: []int32{},
		PMCs:         []int32{},
	}

	// Run through all locations, and write to our array
	for locIdx, loc := range dataset.Locations {
		// Only add if we have spectrum data!
		hasSpectra := false

		for _, det := range loc.Detectors {
			//_, _, err := getSpectrumMeta(det, dataset)

			metaType, metaVar, err := datasetModel.GetDetectorMetaValue("READTYPE", det, dataset)

			// We may fail to read some stuff, there may be no spectrum or metadata in this PMC, that's OK
			if err == nil && metaType == protos.Experiment_MT_STRING && metaVar.Svalue == "Normal" {
				hasSpectra = true
				break
			}
		}

		// Get the PMC
		if hasSpectra {
			pmc, err := strconv.ParseInt(loc.GetId(), 10, 32)
			if err == nil {
				result.LocationIdxs = append(result.LocationIdxs, int32(locIdx))
				result.PMCs = append(result.PMCs, int32(pmc))
			}
		}
	}

	return result
}

func GetROIsWithPMCs(userROIs ROILookup, sharedROIs ROILookup, dataset *protos.Experiment) []ROIMembers {
	result := []ROIMembers{}

	// put them in 1 spot to loop through
	lookups := []ROILookup{userROIs, sharedROIs}

	for c, lookup := range lookups {
		// Read keys in order so we're deterministic and can be tested
		keys := []string{} // TODO: REFACTOR: make this work instead keys := utils.GetStringMapKeys(lookup)
		for key := range lookup {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			roi := lookup[key]
			roiToSave := ROIMembers{
				ID:           key,
				Name:         roi.Name,
				SharedByName: "",
			}

			if c > 0 { // looking in that second lookup!
				roiToSave.SharedByName = roi.Creator.Name
			}

			for _, idx := range roi.LocationIndexes {
				if idx >= 0 && idx < int32(len(dataset.Locations)) {
					pmc, err := strconv.ParseInt(dataset.Locations[idx].Id, 10, 32)

					if err == nil {
						roiToSave.PMCs = append(roiToSave.PMCs, int32(pmc))
						roiToSave.LocationIdxs = append(roiToSave.LocationIdxs, idx)
					}
				}
			}

			result = append(result, roiToSave)
		}
	}

	return result
}

// TODO: make this take params: userID and datasetID instead of a path, path should probably be known only by this package?
// Currently this is not straight-forward in the case of users requesting a shared item, and needing to call utils.StripSharedItemIDPrefix ...
func ReadROIData(svcs *services.APIServices, s3Path string) (ROILookup, error) {
	itemLookup := ROILookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)
	return itemLookup, err
}

func GetROIs(svcs *services.APIServices, userID string, datasetID string, outMap *ROILookup) error {
	s3Path := filepaths.GetROIPath(userID, datasetID)

	items, err := ReadROIData(svcs, s3Path)
	if err != nil {
		return err
	}

	sharedFile := userID == pixlUser.ShareUserID

	// Run through and just return summary info
	for id, item := range items {
		// Loop through all elements and make an element set summary
		toSave := ROISavedItem{
			ROIItem: item.ROIItem,
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  sharedFile,
				Creator: item.Creator,
			},
		}

		// We modify the ids of shared items, so if passed to GET/PUT/DELETE we know this refers to something that's
		saveID := id
		if sharedFile {
			saveID = utils.SharedItemIDPrefix + id
		}
		(*outMap)[saveID] = toSave
	}

	return nil
}

// ShareROIs - Shares the given ROIs that are currently owned by userID, and a part of datasetID. Returns the new IDs generated and an error (or nil)
func ShareROIs(svcs *services.APIServices, userID string, datasetID string, roiIDs []string) ([]string, error) {
	generatedIDs := []string{}

	// User is supplying IDs of an objects to share. We should be able to find them in the users own data file
	// and put it in the shared file with a new ID, thereby implementing "share a copy"

	// Read user items
	s3Path := filepaths.GetROIPath(userID, datasetID)
	userItems, err := ReadROIData(svcs, s3Path)

	if err != nil {
		return generatedIDs, err
	}

	// Read shared items
	sharedS3Path := filepaths.GetROIPath(pixlUser.ShareUserID, datasetID)
	sharedItems, err := ReadROIData(svcs, sharedS3Path)
	if err != nil {
		return generatedIDs, err
	}

	// Run through and share each one
	for _, id := range roiIDs {
		roiItem, ok := userItems[id]
		if !ok {
			return generatedIDs, api.MakeNotFoundError(id)
		}

		// We found it, now generate id to save it to
		sharedID := svcs.IDGen.GenObjectID()
		_, ok = sharedItems[sharedID]
		if ok {
			return generatedIDs, fmt.Errorf("Failed to generate unique share ID for " + id)
		}

		// Add it to the shared file and we're done
		sharedCopy := ROISavedItem{
			ROIItem: &ROIItem{
				Name:            roiItem.Name,
				LocationIndexes: roiItem.LocationIndexes,
				Description:     roiItem.Description,
				ImageName:       roiItem.ImageName,
				PixelIndexes:    roiItem.PixelIndexes,
			},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  true,
				Creator: roiItem.Creator,
			},
		}

		sharedItems[sharedID] = sharedCopy
		generatedIDs = append(generatedIDs, sharedID)
	}

	// Save the shared file
	return generatedIDs, svcs.FS.WriteJSON(svcs.Config.UsersBucket, sharedS3Path, sharedItems)
}
