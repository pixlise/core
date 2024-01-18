package quantification

import (
	"context"
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/indexcompression"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MultiQuantCompare(reqRoiId string, roiPMCs []int32, quantIds []string, exprPB *protos.Experiment, hctx wsHelpers.HandlerContext) ([]*protos.QuantComparisonTable, error) {
	roiName := reqRoiId

	if reqRoiId != "RemainingPoints" {
		coll := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName)
		roiResult := coll.FindOne(context.TODO(), bson.D{{"_id", reqRoiId}}, options.FindOne())
		if roiResult.Err() != nil {
			return nil, roiResult.Err()
		}

		roiItem := &protos.ROIItem{}
		err := roiResult.Decode(&roiItem)
		if err != nil {
			return nil, err
		}

		roiName = roiItem.Name

		// Get location indexes from the ROI and convert them to PMCs
		locIdxs, err := indexcompression.DecodeIndexList(roiItem.ScanEntryIndexesEncoded, -1)
		if err != nil {
			return nil, err
		}

		roiPMCs, err = getPMCsForLocationIndexes(locIdxs, exprPB)
		if err != nil {
			return nil, err
		}
	}

	// Load relevant info from each quantification
	tables := []*protos.QuantComparisonTable{}
	for _, quantId := range quantIds {
		quantDBItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, quantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
		if err != nil {
			return nil, err
		}

		// We have access, so read the file too
		quantPath := path.Join(quantDBItem.Status.OutputFilePath, quantId+".bin")
		quantFile, err := wsHelpers.ReadQuantificationFile(quantId, quantPath, hctx.Svcs)
		if err != nil {
			return nil, err
		}

		// Work out the totals, filtering only to PMCs we are interested in
		totals, err := calculateTotals(quantFile, roiPMCs)

		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to calculate totals for quantification: \"%v\" (%v) and ROI: \"%v\" (%v). Error was: %v", quantDBItem.Params.UserParams.Name, quantId, roiName, reqRoiId, err))
		}

		table := &protos.QuantComparisonTable{QuantId: quantId, QuantName: quantDBItem.Params.UserParams.Name, ElementWeights: totals}
		tables = append(tables, table)
	}

	return tables, nil
}
