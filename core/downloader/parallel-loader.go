// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package downloader

import (
	"fmt"
	"path"
	"sync"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/utils"
	protos "github.com/pixlise/core/v2/generated-protos"
)

// DownloadFiles - Downloads multiple files in parallel as needed. This centralises some logic
// we had in multiple places for larger operations that require several files to be loaded from
// S3 before they can be done.
func DownloadFiles(
	svcs *services.APIServices,
	datasetID string,
	userID string,
	loadUserROIs bool,
	loadSharedROIs bool,
	quantIDs []string,
	loadQuantSummaries bool,
) (
	*protos.Experiment,
	roiModel.ROILookup,
	map[string]*protos.Quantification,
	map[string]quantModel.JobSummaryItem,
	error,
) {
	var wg sync.WaitGroup

	var datasetFile *protos.Experiment
	var datasetError error

	rois := roiModel.ROILookup{}
	var roiError error

	quantFiles := map[string]*protos.Quantification{}
	quantSummaries := map[string]quantModel.JobSummaryItem{}
	quantErrors := []error{}

	// Set up to wait for the right amount of items
	wg.Add(1 + len(quantIDs)) // Always loading dataset file + quant files

	if loadUserROIs {
		wg.Add(1) // Optional load
	}
	if loadSharedROIs {
		wg.Add(1) // Optional load
	}
	if loadQuantSummaries {
		wg.Add(len(quantIDs)) // Optional loading quant summaries
	}

	// Mutex for accessing the "result" maps and arrays above
	mu := sync.Mutex{}

	// Download Dataset file
	go func(datasetID string) {
		defer wg.Done()

		datasetS3Path := filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetFileName)

		svcs.Log.Debugf("  Downloading dataset: %v", datasetS3Path)

		dsFile, err := datasetModel.GetDataset(svcs, datasetS3Path)

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			datasetError = fmt.Errorf("Failed to download dataset: %v", err)
		} else {
			svcs.Log.Debugf("  Finished dataset %v", datasetS3Path)
			datasetFile = dsFile
		}
	}(datasetID)

	// Define ROI loader function
	var roiLoadFunc = func(datasetID string, roiLoadUserID string, isShared bool) {
		defer wg.Done()

		svcs.Log.Debugf("  Downloading ROIs for datasetID: %v, roiID: %v", datasetID, roiLoadUserID)

		roisLoaded := roiModel.ROILookup{}
		err := roiModel.GetROIs(svcs, roiLoadUserID, datasetID, &roisLoaded)

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			roiError = fmt.Errorf("Failed to download user ROIs: %v", err)
		} else {
			svcs.Log.Debugf("  Finished loading ROIs for datasetID: %v, roiID: %v", datasetID, roiLoadUserID)

			// Write them to the ROI array
			for k, v := range roisLoaded {
				rois[k] = v
			}
		}
	}

	if loadUserROIs {
		// Download user ROIs
		go roiLoadFunc(datasetID, userID, false)
	}

	if loadSharedROIs {
		// Download shared ROIs
		go roiLoadFunc(datasetID, pixlUser.ShareUserID, true)
	}

	// Download each quant file
	var quantBinLoadFunc = func(quantID string, s3Path string) {
		defer wg.Done()

		svcs.Log.Debugf("  Downloading quant: %v", quantID)
		quantFileLoaded, err := quantModel.GetQuantification(svcs, s3Path)

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			err = fmt.Errorf("Failed to download quant %v: %v", quantID, err)
			quantErrors = append(quantErrors, err)
		} else {
			svcs.Log.Debugf("  Finished quant %v", quantID)
			quantFiles[quantID] = quantFileLoaded
		}
	}

	var quantSummaryLoadFunc = func(quantID string, s3Path string) {
		defer wg.Done()

		svcs.Log.Debugf("  Downloading quant summary: %v", quantID)

		summaryFileLoaded := quantModel.JobSummaryItem{}
		err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &summaryFileLoaded, false)

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			err = fmt.Errorf("Failed to download quant summary %v: %v", quantID, err)
			quantErrors = append(quantErrors, err)
		} else {
			svcs.Log.Debugf("  Finished quant summary %v", quantID)
			quantSummaries[quantID] = summaryFileLoaded
		}
	}

	for _, quantID := range quantIDs {
		quantS3PathRoot, quantIDToLoad, _ := getQuantificationRootPathAndID(userID, datasetID, quantID)
		quantS3Path := path.Join(quantS3PathRoot, filepaths.MakeQuantDataFileName(quantIDToLoad))

		// Load quant file
		go quantBinLoadFunc(quantID, quantS3Path)

		// Load quant summary file if requested
		if loadQuantSummaries {
			quantSummaryS3Path := path.Join(quantS3PathRoot, filepaths.MakeQuantSummaryFileName(quantIDToLoad))
			go quantSummaryLoadFunc(quantID, quantSummaryS3Path)
		}
	}

	// Wait for all
	wg.Wait()

	// Return them in this order...
	var errorToReturn error

	errorToReturn = roiError
	if errorToReturn == nil {
		errorToReturn = datasetError
		if errorToReturn == nil && len(quantErrors) > 0 {
			errorToReturn = quantErrors[0]
		}
	}

	return datasetFile, rois, quantFiles, quantSummaries, errorToReturn
}

func getQuantificationRootPathAndID(userID string, datasetID string, quantID string) (string, string, bool) {
	quantIDToLoad := quantID
	quantS3PathRoot := filepaths.GetUserQuantPath(userID, datasetID, "")
	strippedQuantID, isSharedQuant := utils.StripSharedItemIDPrefix(quantIDToLoad)
	if isSharedQuant {
		quantS3PathRoot = filepaths.GetSharedQuantPath(datasetID, "")
		quantIDToLoad = strippedQuantID
	}

	return quantS3PathRoot, quantIDToLoad, isSharedQuant
}
