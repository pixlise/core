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

package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/core/api"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/downloader"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/utils"
	protos "github.com/pixlise/core/generated-protos"
)

type MultiQuantificationComparisonRequest struct {
	QuantIDs            []string `json:"quantIDs"`
	RemainingPointsPMCs []int    `json:"remainingPointsPMCs"`
}

type QuantTable struct {
	QuantID   string `json:"quantID"`
	QuantName string `json:"quantName"`

	ElementWeights map[string]float32 `json:"elementWeights"`
}

type MultiQuantificationComparisonResponse struct {
	RoiID       string       `json:"roiID"`
	QuantTables []QuantTable `json:"quantTables"`
}

func multiQuantificationComparisonPost(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	reqRoiID := params.PathParams[idIdentifier]

	reqBody, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, api.MakeBadRequestError(errors.New("Failed to get request body"))
	}

	req := MultiQuantificationComparisonRequest{QuantIDs: []string{}}
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(errors.New("Request body invalid"))
	}

	if len(req.QuantIDs) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("Requested with 0 quant IDs"))
	}

	// If we're requesting for RemainingPoints ROI, mandate that the PMC list is not empty, otherwise it should be
	if reqRoiID == "RemainingPoints" && len(req.RemainingPointsPMCs) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("No PMCs supplied for RemainingPoints ROI"))
	} else if reqRoiID != "RemainingPoints" && len(req.RemainingPointsPMCs) > 0 {
		return nil, api.MakeBadRequestError(errors.New("Unexpected PMCs supplied for ROI: " + reqRoiID))
	}

	// Work out ROI details
	loadUserROIs := false
	loadSharedROIs := false

	// Only loading ROIs if we are NOT using RemainingPoints
	if reqRoiID != "RemainingPoints" {
		/*strippedRoiID*/ _, isSharedRoi := utils.StripSharedItemIDPrefix(reqRoiID)

		if isSharedRoi {
			loadSharedROIs = true
		} else {
			loadUserROIs = true
		}
	}

	// Load files so we can calculate the comparison
	dataset, roisLoaded, quantLookup, quantSummaryLookup, err := downloader.DownloadFiles(params.Svcs, datasetID, params.UserInfo.UserID, loadUserROIs, loadSharedROIs, req.QuantIDs, true)

	if err != nil {
		return nil, api.MakeStatusError(http.StatusNotFound, err)
	}

	// Find the ROI in question
	roiName := reqRoiID // for RemianingPoints really...
	roiPMCs := req.RemainingPointsPMCs

	if reqRoiID != "RemainingPoints" {
		roiItem, ok := roisLoaded[reqRoiID]
		if !ok {
			return nil, api.MakeStatusError(http.StatusNotFound, errors.New("ROI ID "+reqRoiID+" not found"))
		}

		roiName = roiItem.Name

		// Get location indexes from the ROI and convert them to PMCs
		roiPMCs, err = datasetModel.GetPMCsForLocationIndexes(roiItem.LocationIndexes, dataset)

		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}
	}

	// Load relevant info from each quantification
	result := MultiQuantificationComparisonResponse{RoiID: reqRoiID, QuantTables: []QuantTable{}}
	for _, quantID := range req.QuantIDs {
		quantFile, quantOK := quantLookup[quantID]
		summaryFile, summaryOK := quantSummaryLookup[quantID]
		if !quantOK {
			return nil, api.MakeStatusError(http.StatusNotFound, errors.New("Missing quant file: "+quantID))
		}
		if !summaryOK {
			return nil, api.MakeStatusError(http.StatusNotFound, errors.New("Missing quant summary file: "+quantID))
		}

		// Work out the totals, filtering only to PMCs we are interested in
		totals, err := calculateTotals(quantFile, roiPMCs)

		if err != nil {
			return nil, api.MakeBadRequestError(fmt.Errorf("Failed to calculate totals for quantification: \"%v\" (%v) and ROI: \"%v\" (%v). Error was: %v", summaryFile.Params.Name, quantID, roiName, reqRoiID, err))
		}

		table := QuantTable{QuantID: quantID, QuantName: summaryFile.Params.Name, ElementWeights: totals}
		result.QuantTables = append(result.QuantTables, table)
	}

	return result, nil
}

func calculateTotals(quantFile *protos.Quantification, roiPMCs []int) (map[string]float32, error) {
	columns := []string{}
	totals := map[string]float32{}

	// Ensure we're dealing with a Combined quant
	if len(quantFile.LocationSet) != 1 || quantFile.LocationSet[0].Detector != "Combined" {
		return totals, errors.New("Quantification must be for Combined detectors")
	}

	// Make a quick lookup for PMCs
	roiPMCSet := map[int]bool{} // REFACTOR: TODO: Make generic version of utils.SetStringsInMap() for this
	for _, pmc := range roiPMCs {
		roiPMCSet[pmc] = true
	}

	// Decide on columns we're working with
	columns = quantModel.GetWeightPercentColumnsInQuant(quantFile)
	columnIdxs := []int{}
	for _, col := range columns {
		columnIdxs = append(columnIdxs, int(quantModel.GetQuantColumnIndex(quantFile, col)))
	}

	if len(columns) <= 0 {
		return totals, errors.New("Quantification has no weight %% columns")
	}

	// Read the combined locations for the PMCs in the ROI
	foundPMCCount := 0
	for _, loc := range quantFile.LocationSet[0].Location {
		// If PMC is in ROI
		if roiPMCSet[int(loc.Pmc)] {
			// Run through all columns and add them to our totals
			for c := 0; c < len(columnIdxs); c++ {
				column := columns[c]
				value := loc.Values[columnIdxs[c]]

				totals[column] += value.Fvalue
			}

			foundPMCCount++
		}
	}

	// Here we finally turn these into averages
	result := map[string]float32{}
	for k, v := range totals {
		if foundPMCCount > 0 {
			result[k] = v / float32(foundPMCCount)
		}
	}

	if len(result) <= 0 {
		return result, errors.New("Quantification had no valid data for ROI PMCs")
	}

	return result, nil
}
