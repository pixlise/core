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

package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/core/api"
	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/downloader"
	"github.com/pixlise/core/v3/core/quantModel"
	"github.com/pixlise/core/v3/core/roiModel"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

// Users specify a range of ROIs, with a quant for each. Order matters, this is how they will be combined
type QuantCombineItem struct {
	RoiID            string `json:"roiID"`
	QuantificationID string `json:"quantificationID"`
}

type QuantCombineRequest struct {
	RoiZStack   []QuantCombineItem `json:"roiZStack"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	SummaryOnly bool               `json:"summaryOnly"`
}

type QuantCombineList struct {
	RoiZStack []QuantCombineItem `json:"roiZStack"`
}

type QuantItem struct {
	RTT      int32
	PMC      int32
	SCLK     int32
	Filename string
	LiveTime int32
	RoiID    string
	Columns  map[string]float64
	ROIName  string
}

type SummaryRow struct {
	Values   []float32 `json:"values"`
	ROIIDs   []string  `json:"roiIDs"`
	ROINames []string  `json:"roiNames"`
}

type QuantCombineSummaryResponse struct {
	Detectors      []string              `json:"detectors"`
	WeightPercents map[string]SummaryRow `json:"weightPercents"`
}

func quantificationCombine(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req QuantCombineRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	// Simple validation

	// NOTE: if only asking for a summary, we don't care about name being empty
	if !req.SummaryOnly && len(req.Name) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("Name cannot be empty"))
	}

	if len(req.RoiZStack) <= 1 {
		return nil, api.MakeBadRequestError(errors.New("Must reference more than 1 ROI"))
	}

	if quantModel.CheckQuantificationNameExists(req.Name, params.PathParams[datasetIdentifier], params.UserInfo.UserID, params.Svcs) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Name already used: %v", req.Name))
	}

	// Read ROIs
	// NOTE: here we split out the "remaining points" roi, and pull in ALL points (pmcs) for the quantification specified
	// because that's the "base" layer, and the refinements happen on top of that. It should be the first item in our ROI ID
	// array.
	roiIDs := []string{}
	//baseQuant := ""

	for _, zItem := range req.RoiZStack {
		/*if c == 0 {
			if zItem.RoiID != "RemainingPoints" {
				return nil, api.MakeBadRequestError(errors.New("First ROI must be RemainingPoints"))
			}

			baseQuant = zItem.QuantificationID
		} else {*/
		// Ensure unique ROI
		if utils.StringInSlice(zItem.RoiID, roiIDs) {
			return nil, api.MakeBadRequestError(fmt.Errorf("Duplicate ROI ID: %v", zItem.RoiID))
		}
		roiIDs = append(roiIDs, zItem.RoiID)
		//}
	}

	// Make unique list of quant IDs
	quantIDs := []string{}

	for _, zItem := range req.RoiZStack {
		if len(zItem.QuantificationID) <= 0 {
			return nil, api.MakeBadRequestError(fmt.Errorf("Quantification not specified for ROI ID: %v", zItem.RoiID))
		}

		if !utils.StringInSlice(zItem.QuantificationID, quantIDs) {
			quantIDs = append(quantIDs, zItem.QuantificationID)
		}
	}

	// Seems pointless, but breaks tests if not done
	sort.Strings(quantIDs)

	// Download all files needed. This does it in parallel, we then process quickly using ready items
	datasetProto, roisLoaded, quantLookup, _, err := downloader.DownloadFiles(params.Svcs, datasetID, params.UserInfo.UserID, true, true, quantIDs, false)

	if err != nil {
		return nil, api.MakeStatusError(http.StatusNotFound, err)
	}

	// Filter ROIs down to the ones we're dealing with
	roisById, err := filterROIsByID(roisLoaded, roiIDs)

	if err != nil {
		return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("Failed to get all ROIs: %v", err))
	}

	// Check that loaded quants all have the same detector counts, etc
	// We also init the detector list and build a list of all columns here
	dataPerDetectorPerPMC := map[string]map[int]QuantItem{}
	detectors := []string{}
	allColumns := map[string]bool{}

	// Traverse the map in quant ID order, so unit tests don't fail :(
	for _, quantID := range quantIDs {
		quant, ok := quantLookup[quantID]
		if !ok {
			return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("Quantification lookup failed for: %v", quantID))
		}

		if len(dataPerDetectorPerPMC) <= 0 {
			// First one, save the detectors
			for _, detectorLocation := range quant.LocationSet {
				dataPerDetectorPerPMC[detectorLocation.Detector] = map[int]QuantItem{}
				detectors = append(detectors, detectorLocation.Detector)
			}
		} else {
			// Ensure detectors match what we're already storing from prev quants
			matchCount := 0
			for _, detectorLocation := range quant.LocationSet {
				if utils.StringInSlice(detectorLocation.Detector, detectors) {
					matchCount++
				}
			}

			if matchCount != len(detectors) {
				return nil, api.MakeBadRequestError(fmt.Errorf("Detectors don't match other quantifications: %v", quantID))
			}
		}

		// Store in all columns
		for _, col := range quant.Labels {
			if isRequiredColumnForQuant(col) {
				allColumns[col] = true
			}
		}
	}

	// Get all points so we can use it later AND so we can calculate our ratios of PMC count to all PMCs
	// for each element total
	allPoints := roiModel.GetAllPointsROI(datasetProto)

	// Build the multi-quant by traversing the z-stack
	for c := range req.RoiZStack {
		zItem := req.RoiZStack[len(req.RoiZStack)-c-1]
		roiName := ""

		// Get location indexes from the ROI and convert them to PMCs
		roiPMCs := []int{}
		if zItem.RoiID == "RemainingPoints" {
			// RemainingPoints should be at the end (first in reverse order!)
			if c != 0 {
				return nil, api.MakeBadRequestError(errors.New("RemainingPoints ROI must be last in z-stack"))
			}

			// We actually use all points, it's the bottom of the z-stack, anything in z-stack items above
			// overrides these values, so therefore it will end up covering the "remaining points"
			for _, pmc := range allPoints.PMCs {
				roiPMCs = append(roiPMCs, int(pmc))
			}

			roiName = zItem.RoiID
		} else {
			roiItem, ok := roisById[zItem.RoiID]
			if !ok {
				return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("Failed to retrieve ROI: %v", zItem.RoiID))
			}

			var err error

			roiPMCs, err = datasetModel.GetPMCsForLocationIndexes(roiItem.LocationIndexes, datasetProto)
			if err != nil {
				return nil, api.MakeStatusError(http.StatusNotFound, err)
			}

			roiName = roiItem.Name
		}

		// Process ROI
		quant := quantLookup[zItem.QuantificationID]
		for _, detectorLocation := range quant.LocationSet {
			// Build a quick lookup for PMC->idx within this detector, because we'll be referering to them by PMC
			// not their location index
			pmcToIdx := map[int]int{}
			for idx, loc := range detectorLocation.Location {
				pmcToIdx[int(loc.Pmc)] = idx
			}

			// Run through PMCs in the ROI and grab column data
			for _, pmc := range roiPMCs {
				// NOTE: This should overwrite an existing entry for a pmc, we're going up the z-stack and
				// as we find pmcs, they're more "important" than past entries
				idx, ok := pmcToIdx[pmc]
				if ok {
					toWrite := getDataForCombine(
						detectorLocation.Location[idx],
						quant.Labels,
						detectorLocation.Detector,
						zItem.RoiID,
						roiName,
					)

					dataPerDetectorPerPMC[detectorLocation.Detector][pmc] = toWrite
				}
				// else: ROI specified a PMC that we didn't have data for in this quant. This can easily happen if users created the ROI
				// by circling a PMC that has no spectra (eg a housekeeping PMC)
			}
		}
	}

	if req.SummaryOnly {
		// We return a summary instead of forming a CSV
		return formMultiQuantSummary(dataPerDetectorPerPMC, allColumns, len(allPoints.PMCs))
	}

	// Form a CSV
	csv := formCombinedCSV(quantIDs, dataPerDetectorPerPMC, allColumns)

	quantMode := quantModel.QuantModeCombinedMultiQuant
	if len(detectors) > 1 {
		quantMode = quantModel.QuantModeABMultiQuant
	}

	return quantModel.ImportQuantCSV(params.Svcs, datasetID, params.UserInfo, csv, "combined-multi", "multi", req.Name, quantMode, req.Description)
}

func formCombinedCSV(quantIDs []string, dataPerDetectorPerPMC map[string]map[int]QuantItem, allColumns map[string]bool) string {
	var csv strings.Builder

	// Header
	csv.WriteString("Combined multi-quantification from " + strings.Join(quantIDs, ", ") + "\n")
	csv.WriteString("PMC, RTT, SCLK, filename, livetime")

	columnsInOrderOfPrint := []string{}
	for col := range allColumns {
		if col != "livetime" {
			columnsInOrderOfPrint = append(columnsInOrderOfPrint, col)
		}
	}

	sort.Strings(columnsInOrderOfPrint)

	for _, col := range columnsInOrderOfPrint {
		csv.WriteString(", ")
		csv.WriteString(col)
	}

	csv.WriteString("\n")

	// We loop through all detectors, all PMCs, try to read for all columns, if a column doesn't exist save with a -1
	for _ /*detector*/, detectorData := range dataPerDetectorPerPMC {
		//for _ /*pmc*/, quantItem := range detectorData {

		// Read PMCs in ascending order
		pmcs := []int{}
		for pmc := range detectorData {
			pmcs = append(pmcs, pmc)
		}
		sort.Ints(pmcs)

		for _, pmc := range pmcs {
			quantItem := detectorData[pmc]
			csv.WriteString(fmt.Sprintf("%v, %v, %v, %v, %v", quantItem.PMC, quantItem.RTT, quantItem.SCLK, quantItem.Filename+"_"+quantItem.RoiID, quantItem.LiveTime))

			// Append each column
			for _, colName := range columnsInOrderOfPrint {
				csv.WriteString(", ")

				val, ok := quantItem.Columns[colName]
				if !ok {
					val = -1
				}

				csv.WriteString(fmt.Sprint(val))
			}

			csv.WriteString("\n")
		}
	}

	return csv.String()
}

func isRequiredColumnForQuant(colName string) bool {
	return strings.HasSuffix(colName, "_%") || strings.HasSuffix(colName, "_err") || colName == "filename" || colName == "livetime"
}

func getDataForCombine(locationData *protos.Quantification_QuantLocation, colNames []string, detector string, roiID string, roiName string) QuantItem {
	result := QuantItem{
		RTT:      locationData.Rtt,
		PMC:      locationData.Pmc,
		SCLK:     locationData.Sclk,
		Filename: "Normal_" + detector,
		RoiID:    roiID,
		Columns:  map[string]float64{},
		ROIName:  roiName,
	}

	// Grab all columns we're interested in...
	for c, val := range locationData.Values {
		if isRequiredColumnForQuant(colNames[c]) {
			if colNames[c] == "livetime" {
				result.LiveTime = val.Ivalue
			} else if colNames[c] != "filename" { // filename is handled separately
				result.Columns[colNames[c]] = float64(val.Fvalue)
			}
		}
	}

	return result
}

func filterROIsByID(rois roiModel.ROILookup, roiIDs []string) (roiModel.ROILookup, error) {
	result := roiModel.ROILookup{}

	for _, roiID := range roiIDs {
		// Allow RemainingPoints through
		if roiID != "RemainingPoints" {
			roiItem, ok := rois[roiID]
			if !ok {
				return nil, fmt.Errorf("Failed to find ROI ID: %v", roiID)
			}
			result[roiID] = roiItem
		}
	}

	return result, nil
}

func formMultiQuantSummary(dataPerDetectorPerPMC map[string]map[int]QuantItem, allColumns map[string]bool, totalPMCCount int) (interface{}, error) {
	resp := QuantCombineSummaryResponse{}
	resp.Detectors = []string{}

	for det := range dataPerDetectorPerPMC {
		resp.Detectors = append(resp.Detectors, det)
	}

	resp.WeightPercents = map[string]SummaryRow{}

	// Add all elements
	for col := range allColumns {
		// Only add _% ones (but remove _%!)
		if strings.HasSuffix(col, "_%") {
			colName := col[0 : len(col)-2]
			values := []float32{}
			roisInvolved := map[string]string{}

			// Total up the column and then multiply by percentage of PMCs that were included
			for _, detData := range dataPerDetectorPerPMC {
				total := float64(0)
				count := 0

				for _, item := range detData {
					val, ok := item.Columns[col]
					if ok {
						total += val
						count++

						roisInvolved[item.RoiID] = item.ROIName
					}
				}

				// Save for this detector
				if count > 0 {
					// NOTE: we are taking the average, but want to multiply by the ratio of points
					// in sample, so it would be
					// avg = total/count * count/totalPMCCount
					// which simplifies to total/totalPMCCount
					values = append(values, float32(total/float64(totalPMCCount)))
				}
			}

			roiNames := []string{}
			roiIDs := []string{}

			for roiID := range roisInvolved {
				roiIDs = append(roiIDs, roiID)
			}

			// Read them in order (helps unit tests)
			sort.Strings(roiIDs)

			for _, roiID := range roiIDs {
				roiNames = append(roiNames, roisInvolved[roiID])
			}

			resp.WeightPercents[colName] = SummaryRow{
				Values:   values,
				ROINames: roiNames,
				ROIIDs:   roiIDs,
			}
		}
	}

	return resp, nil
}
