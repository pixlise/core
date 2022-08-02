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

package downloader

import (
	"fmt"
	"path"
	"sync"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/roiModel"
	"github.com/pixlise/core/core/utils"
	protos "github.com/pixlise/core/generated-protos"
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
