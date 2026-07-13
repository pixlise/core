package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/indexcompression"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetDetectedDiffractionPeaks(requestedIndexes []int32, exprPB *protos.Experiment, diffRawData *protos.Diffraction) ([]*protos.DetectedDiffractionPerLocation, error) {
	// Form a PMC->diffraction peaks lookup
	diffLookup := map[string]*protos.Diffraction_Location{}
	for _, loc := range diffRawData.Locations {
		diffLookup[loc.Id] = loc
	}

	// Decode the range
	diffPerLoc := []*protos.DetectedDiffractionPerLocation{}

	if len(requestedIndexes) > 0 {
		indexes, err := indexcompression.DecodeIndexList(requestedIndexes, len(exprPB.Locations))
		if err != nil {
			return nil, err
		}

		for _, c := range indexes {
			exprLoc := exprPB.Locations[c]

			if loc, ok := diffLookup[exprLoc.Id]; ok {
				peaks := readDiffractionLocPeaks(loc)

				diffPerLoc = append(diffPerLoc, &protos.DetectedDiffractionPerLocation{
					Id:    loc.Id,
					Peaks: peaks,
				})
			}
		}
	} else {
		// No indexes specified, so assume they want them all
		for _, loc := range diffLookup {
			peaks := readDiffractionLocPeaks(loc)

			diffPerLoc = append(diffPerLoc, &protos.DetectedDiffractionPerLocation{
				Id:    loc.Id,
				Peaks: peaks,
			})
		}
	}

	return diffPerLoc, nil
}

func readDiffractionLocPeaks(loc *protos.Diffraction_Location) []*protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak {
	peaks := []*protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{}

	for _, locPeak := range loc.Peaks {
		peaks = append(peaks, &protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{
			PeakChannel:       locPeak.PeakChannel,
			EffectSize:        locPeak.EffectSize,
			BaselineVariation: locPeak.BaselineVariation,
			GlobalDifference:  locPeak.GlobalDifference,
			DifferenceSigma:   locPeak.DifferenceSigma,
			PeakHeight:        locPeak.PeakHeight,
			Detector:          locPeak.Detector,
		})
	}

	return peaks
}

func GetDiffractionPeakManualList(scanId string, svcs *services.APIServices) (map[string]*protos.ManualDiffractionPeak, error) {
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.DiffractionManualPeaksName)

	filter := bson.M{"scanid": scanId}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Silent error, just return empty
			return map[string]*protos.ManualDiffractionPeak{}, nil
		}

		return nil, err
	}

	result := []*protos.ManualDiffractionPeak{}
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	resultMap := map[string]*protos.ManualDiffractionPeak{}
	for _, item := range result {
		resultMap[item.Id] = item
		item.Id = ""     // Clear it, no point doubling up info, the map key contains the id already
		item.ScanId = "" // Also no point keeping this around, it was part of the request params
	}

	return resultMap, nil
}

func GetDiffractionPeakStatusList(scanId string, svcs *services.APIServices) (*protos.DetectedDiffractionPeakStatuses, error) {
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.DiffractionDetectedPeakStatusesName)

	filter := bson.M{"_id": scanId}
	dbResult := coll.FindOne(ctx, filter)
	if dbResult.Err() != nil {
		if dbResult.Err() == mongo.ErrNoDocuments {
			// Silent error, just return empty
			return &protos.DetectedDiffractionPeakStatuses{
				Id:       scanId,
				ScanId:   scanId,
				Statuses: map[string]*protos.DetectedDiffractionPeakStatuses_PeakStatus{},
			}, nil
		}
		return nil, dbResult.Err()
	}

	result := &protos.DetectedDiffractionPeakStatuses{}
	err := dbResult.Decode(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
