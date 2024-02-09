package quantification

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type roiItemWithPMCs struct {
	PMCs []int
	*protos.ROIItem
}

var allPointsROIId = "AllPoints"

func getROIs(
	quantCommand string,
	scanId string,
	roiIds []string,
	svcs *services.APIServices,
	requestorSession *wsHelpers.SessionUser,
	locIdxToPMCLookup map[int32]int32,
	dataset *protos.Experiment) ([]roiItemWithPMCs, error) {
	result := []roiItemWithPMCs{}
	var err error

	if len(roiIds) <= 0 {
		// If we're in a map command, this is bad, as we want to have a list of ROIs to generate for
		if quantCommand == "map" {
			return result, errors.New("No ROI IDs specified for sum-then-quantify mode")
		} else {
			// For anything else, we use AllPoints
			roiIds = append(roiIds, allPointsROIId)
		}
	}

	queryROIs := []string{}
	needAllPoints := false

	if requestorSession != nil {
		// Read ROI IDs accessible to this user
		idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_ROI, svcs, *requestorSession)
		if err != nil {
			return nil, err
		}

		// Make sure all the ones we're after are in this list
		for _, roiId := range roiIds {
			if roiId != allPointsROIId {
				if _, ok := idToOwner[roiId]; !ok {
					return result, fmt.Errorf("User %v does not have permission to access ROI %v", requestorSession.User.Id, roiId)
				} else {
					queryROIs = append(queryROIs, roiId)
				}
			} else {
				needAllPoints = true
			}
		}
	} else {
		// Not requesting from the POV of a user, so we're just reading these...
		for _, roiId := range roiIds {
			queryROIs = append(queryROIs, roiId)

			if roiId == allPointsROIId {
				needAllPoints = true
			}
		}
	}

	filter := bson.M{"_id": bson.M{"$in": queryROIs}}

	coll := svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName)
	cursor, err := coll.Find(context.TODO(), filter, options.Find())
	if err != nil {
		return nil, err
	}

	items := []*protos.ROIItem{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Run through them and form output list
	for _, item := range items {
		roiWithPMCs, err := makeROIWithPMCs(item, locIdxToPMCLookup)
		if err != nil {
			return result, err
		}
		result = append(result, *roiWithPMCs)
	}

	// If we need the all points ROI, do that...
	if needAllPoints {
		roiWithPMCs := makeAllPointsROI(scanId, dataset)
		result = append(result, *roiWithPMCs)
	}

	return result, nil
}

func makeROIWithPMCs(roi *protos.ROIItem, locIdxToPMCLookup map[int32]int32) (*roiItemWithPMCs, error) {
	pmcs := []int{}
	for _, locIdx := range roi.ScanEntryIndexesEncoded {
		if pmc, ok := locIdxToPMCLookup[locIdx]; ok {
			pmcs = append(pmcs, int(pmc))
		}
		// We used to error here, but now that we're filtering out PMCs that have no normal/dwell spectra, this is a valid scenario
		// where an ROI contained a housekeeping PMC and the quant would've failed unless we filter out the bad PMC here.
		/* else {
			return nil, fmt.Errorf("Failed to find PMC for loc idx: %v in ROI: %v, ROI id: %v", locIdx, roi.Name, roiID)
		}*/
	}

	sort.Ints(pmcs)

	result := &roiItemWithPMCs{
		PMCs:    pmcs,
		ROIItem: roi,
	}

	return result, nil
}

func makeAllPointsROI(scanId string, dataset *protos.Experiment) *roiItemWithPMCs {
	locIdxs := []int32{}
	PMCs := []int{}

	for locIdx, loc := range dataset.Locations {
		// Only add if we have spectrum data!
		hasSpectra := false

		for _, det := range loc.Detectors {
			//_, _, err := getSpectrumMeta(det, dataset)

			metaType, metaVar, err := getDetectorMetaValue("READTYPE", det, dataset)

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
				locIdxs = append(locIdxs, int32(locIdx))
				PMCs = append(PMCs, int(pmc))
			}
		}
	}

	sort.Ints(PMCs)

	result := &roiItemWithPMCs{
		PMCs: PMCs,
		ROIItem: &protos.ROIItem{
			Id:                      allPointsROIId,
			ScanId:                  scanId,
			Name:                    "All Points",
			Description:             "All Points",
			ScanEntryIndexesEncoded: locIdxs,
		},
	}

	return result
}
