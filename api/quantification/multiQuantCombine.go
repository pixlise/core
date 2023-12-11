package quantification

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/indexcompression"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type quantItem struct {
	RTT      int32
	PMC      int32
	SCLK     int32
	Filename string
	LiveTime int32
	RoiID    string
	Columns  map[string]float64
	ROIName  string
}

type combinedQuantData struct {
	DataPerDetectorPerPMC map[string]map[int32]quantItem
	AllColumns            map[string]bool
	PMCCount              int
	Detectors             []string
	QuantIds              []string
}

func MultiQuantCombinedCSV(
	name string,
	scanId string,
	roiZStack []*protos.QuantCombineItem,
	exprPB *protos.Experiment,
	hctx wsHelpers.HandlerContext) (combinedQuantData, error) {
	result := combinedQuantData{
		map[string]map[int32]quantItem{},
		map[string]bool{},
		0,
		[]string{},
		[]string{},
	}

	if checkQuantificationNameExists(name, scanId, hctx) {
		return result, errorwithstatus.MakeBadRequestError(fmt.Errorf("Name already used: %v", name))
	}

	// Read ROIs
	// NOTE: here we split out the "remaining points" roi, and pull in ALL points (pmcs) for the quantification specified
	// because that's the "base" layer, and the refinements happen on top of that. It should be the first item in our ROI ID
	// array.
	roiIDs := []string{}

	for _, zItem := range roiZStack {
		// Ensure unique ROI
		if utils.ItemInSlice(zItem.RoiId, roiIDs) {
			return result, errorwithstatus.MakeBadRequestError(fmt.Errorf("Duplicate ROI ID: %v", zItem.RoiId))
		}
		roiIDs = append(roiIDs, zItem.RoiId)
	}

	// Make unique list of quant IDs
	for _, zItem := range roiZStack {
		if len(zItem.QuantificationId) <= 0 {
			return result, errorwithstatus.MakeBadRequestError(fmt.Errorf("Quantification not specified for ROI ID: %v", zItem.RoiId))
		}

		if !utils.ItemInSlice(zItem.QuantificationId, result.QuantIds) {
			result.QuantIds = append(result.QuantIds, zItem.QuantificationId)
		}
	}

	// Check that loaded quants all have the same detector counts, etc
	// We also init the detector list and build a list of all columns here
	quantSummaryLookup := map[string]*protos.QuantificationSummary{}
	quantDataLookup := map[string]*protos.Quantification{}

	for _, quantId := range result.QuantIds {
		// Read the DB item to verify we have access and to get the quant path
		quantDBItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, quantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
		if err != nil {
			return result, err
		}

		quantSummaryLookup[quantId] = quantDBItem

		// Load the quant
		quantPath := path.Join(quantDBItem.Status.OutputFilePath, quantId+".bin")
		quantData, err := wsHelpers.ReadQuantificationFile(quantId, quantPath, hctx.Svcs)
		if err != nil {
			return result, err
		}

		quantDataLookup[quantId] = quantData

		if len(result.DataPerDetectorPerPMC) <= 0 {
			// First one, save the detectors
			for _, detectorLocation := range quantData.LocationSet {
				result.DataPerDetectorPerPMC[detectorLocation.Detector] = map[int32]quantItem{}
				result.Detectors = append(result.Detectors, detectorLocation.Detector)
			}
		} else {
			// Ensure detectors match what we're already storing from prev quants
			matchCount := 0
			for _, detectorLocation := range quantData.LocationSet {
				if utils.ItemInSlice(detectorLocation.Detector, result.Detectors) {
					matchCount++
				}
			}

			if matchCount != len(result.Detectors) {
				return result, errorwithstatus.MakeBadRequestError(fmt.Errorf("Detectors don't match other quantifications: %v", quantId))
			}
		}

		// Store in all columns
		for _, col := range quantData.Labels {
			if isRequiredColumnForQuant(col) {
				result.AllColumns[col] = true
			}
		}
	}

	// Get all points so we can use it later AND so we can calculate our ratios of PMC count to all PMCs
	// for each element total
	allPoints := makeAllPointsROI(scanId, exprPB)

	// Build the multi-quant by traversing the z-stack
	for c := range roiZStack {
		zItem := roiZStack[len(roiZStack)-c-1]
		roiName := ""

		// Get location indexes from the ROI and convert them to PMCs
		roiPMCs := []int32{}
		if zItem.RoiId == "RemainingPoints" {
			// RemainingPoints should be at the end (first in reverse order!)
			if c != 0 {
				return result, errorwithstatus.MakeBadRequestError(errors.New("RemainingPoints ROI must be last in z-stack"))
			}

			// We actually use all points, it's the bottom of the z-stack, anything in z-stack items above
			// overrides these values, so therefore it will end up covering the "remaining points"
			for _, pmc := range allPoints.PMCs {
				roiPMCs = append(roiPMCs, int32(pmc))
			}

			roiName = zItem.RoiId
		} else {
			// Read from DB
			coll := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName)
			roiResult := coll.FindOne(context.TODO(), bson.D{{"_id", zItem.RoiId}}, options.FindOne())
			if roiResult.Err() != nil {
				return result, roiResult.Err()
			}

			roiItem := &protos.ROIItem{}
			err := roiResult.Decode(&roiItem)
			if err != nil {
				return result, err
			}

			// Get location indexes from the ROI and convert them to PMCs
			locIdxs, err := indexcompression.DecodeIndexList(roiItem.ScanEntryIndexesEncoded, -1)
			if err != nil {
				return result, err
			}

			roiPMCs, err = getPMCsForLocationIndexes(locIdxs, exprPB)
			if err != nil {
				return result, errorwithstatus.MakeStatusError(http.StatusNotFound, err)
			}

			roiName = roiItem.Name
		}

		// Process ROI
		quant := quantDataLookup[zItem.QuantificationId]
		for _, detectorLocation := range quant.LocationSet {
			// Build a quick lookup for PMC->idx within this detector, because we'll be referering to them by PMC
			// not their location index
			pmcToIdx := map[int32]int{}
			for idx, loc := range detectorLocation.Location {
				pmcToIdx[loc.Pmc] = idx
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
						zItem.RoiId,
						roiName,
					)

					result.DataPerDetectorPerPMC[detectorLocation.Detector][pmc] = toWrite
				}
				// else: ROI specified a PMC that we didn't have data for in this quant. This can easily happen if users created the ROI
				// by circling a PMC that has no spectra (eg a housekeeping PMC)
			}
		}
	}

	result.PMCCount = len(allPoints.PMCs)
	return result, nil
}

func FormCombinedCSV(quantIDs []string, dataPerDetectorPerPMC map[string]map[int32]quantItem, allColumns map[string]bool) string {
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
			pmcs = append(pmcs, int(pmc))
		}
		sort.Ints(pmcs)

		for _, pmc := range pmcs {
			quantItem := detectorData[int32(pmc)]
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

func getDataForCombine(locationData *protos.Quantification_QuantLocation, colNames []string, detector string, roiID string, roiName string) quantItem {
	result := quantItem{
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

func FormMultiQuantSummary(dataPerDetectorPerPMC map[string]map[int32]quantItem, allColumns map[string]bool, totalPMCCount int) *protos.QuantCombineSummary {
	resp := &protos.QuantCombineSummary{}
	resp.Detectors = []string{}

	for det := range dataPerDetectorPerPMC {
		resp.Detectors = append(resp.Detectors, det)
	}

	resp.WeightPercents = map[string]*protos.QuantCombineSummaryRow{}

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

			resp.WeightPercents[colName] = &protos.QuantCombineSummaryRow{
				Values:   values,
				RoiNames: roiNames,
				RoiIds:   roiIDs,
			}
		}
	}

	return resp
}
