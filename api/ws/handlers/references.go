package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func validateReferenceData(ref *protos.ReferenceData) error {
	if err := wsHelpers.CheckStringField(&ref.Category, "Category", 0, 100); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&ref.Group, "Group", 0, 100); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&ref.MineralSampleName, "MineralSampleName", 1, 255); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&ref.SourceCitation, "SourceCitation", 0, 1000); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&ref.SourceLink, "SourceLink", 0, 500); err != nil {
		return err
	}

	// Allow empty expression value pairs
	if len(ref.ExpressionValuePairs) == 0 {
		return nil
	}

	for i, pair := range ref.ExpressionValuePairs {
		if err := wsHelpers.CheckStringField(&pair.ExpressionId, "ExpressionId", 1, wsHelpers.IdFieldMaxLength); err != nil {
			return errorwithstatus.MakeBadRequestError(errors.New("ExpressionValuePairs[" + string(rune(i)) + "].ExpressionId: " + err.Error()))
		}
		if err := wsHelpers.CheckStringField(&pair.ExpressionName, "ExpressionName", 1, 255); err != nil {
			return errorwithstatus.MakeBadRequestError(errors.New("ExpressionValuePairs[" + string(rune(i)) + "].ExpressionName: " + err.Error()))
		}
	}

	return nil
}

func HandleReferenceDataListReq(req *protos.ReferenceDataListReq, hctx wsHelpers.HandlerContext) (*protos.ReferenceDataListResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ReferencesName)

	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "category", Value: 1}, {Key: "group", Value: 1}, {Key: "mineralsamplename", Value: 1}}))
	if err != nil {
		return nil, err
	}

	items := []*protos.ReferenceData{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	return &protos.ReferenceDataListResp{
		ReferenceData: items,
	}, nil
}

func HandleReferenceDataGetReq(req *protos.ReferenceDataGetReq, hctx wsHelpers.HandlerContext) (*protos.ReferenceDataGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ReferencesName)

	item := &protos.ReferenceData{}
	err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: req.Id}}).Decode(item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Id)
		}
		return nil, err
	}

	return &protos.ReferenceDataGetResp{
		ReferenceData: item,
	}, nil
}

func HandleReferenceDataDeleteReq(req *protos.ReferenceDataDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ReferenceDataDeleteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ReferencesName)

	result, err := coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: req.Id}})
	if err != nil {
		return nil, err
	}

	if result.DeletedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.Id)
	}

	return &protos.ReferenceDataDeleteResp{}, nil
}

func HandleReferenceDataWriteReq(req *protos.ReferenceDataWriteReq, hctx wsHelpers.HandlerContext) (*protos.ReferenceDataWriteResp, error) {
	if req.ReferenceData == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("ReferenceData must be specified"))
	}

	// Validate the reference data
	err := validateReferenceData(req.ReferenceData)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ReferencesName)

	// Check if this is a create or update operation
	isUpdate := len(req.ReferenceData.Id) > 0

	if isUpdate {
		// Update existing reference data
		if err := wsHelpers.CheckStringField(&req.ReferenceData.Id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
			return nil, err
		}

		// Check if the item exists
		existsResult := coll.FindOne(ctx, bson.D{{Key: "_id", Value: req.ReferenceData.Id}})
		if existsResult.Err() != nil {
			if existsResult.Err() == mongo.ErrNoDocuments {
				return nil, errorwithstatus.MakeNotFoundError(req.ReferenceData.Id)
			}
			return nil, existsResult.Err()
		}

		// Update the document
		update := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "category", Value: req.ReferenceData.Category},
				{Key: "group", Value: req.ReferenceData.Group},
				{Key: "mineralsamplename", Value: req.ReferenceData.MineralSampleName},
				{Key: "sourcecitation", Value: req.ReferenceData.SourceCitation},
				{Key: "sourcelink", Value: req.ReferenceData.SourceLink},
				{Key: "expressionvaluepairs", Value: req.ReferenceData.ExpressionValuePairs},
			}},
		}

		result, err := coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: req.ReferenceData.Id}}, update)
		if err != nil {
			return nil, err
		}

		if result.MatchedCount != 1 {
			hctx.Svcs.Log.Errorf("ReferenceData UpdateOne result had unexpected counts %v id: %v", result, req.ReferenceData.Id)
		}
	} else {
		// Create new reference data
		id := hctx.Svcs.IDGen.GenObjectID()
		req.ReferenceData.Id = id

		_, err := coll.InsertOne(ctx, req.ReferenceData)
		if err != nil {
			return nil, err
		}
	}

	return &protos.ReferenceDataWriteResp{
		ReferenceData: req.ReferenceData,
	}, nil
}

func HandleReferenceDataBulkWriteReq(req *protos.ReferenceDataBulkWriteReq, hctx wsHelpers.HandlerContext) (*protos.ReferenceDataBulkWriteResp, error) {
	if req.ReferenceData == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("ReferenceData must be specified"))
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ReferencesName)

	if req.MatchByFields {
		// Check if the items exist
		for _, item := range req.ReferenceData {
			existsResult := coll.FindOne(ctx, bson.D{{Key: "mineralsamplename", Value: item.MineralSampleName}, {Key: "category", Value: item.Category}, {Key: "group", Value: item.Group}})
			if existsResult.Err() != nil {
				// If the item doesn't exist, we'll just insert it
				if existsResult.Err() == mongo.ErrNoDocuments {
					continue
				}
				return nil, existsResult.Err()
			}
			// Update the id with the existing id so we can use it to update the document
			var decodedItem protos.ReferenceData
			err := existsResult.Decode(&decodedItem)
			if err != nil {
				return nil, err
			}
			item.Id = decodedItem.Id
		}
	}

	// Insert or update the items
	for _, item := range req.ReferenceData {
		if _, err := coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: item.Id}}, bson.D{{Key: "$set", Value: item}}, options.Update().SetUpsert(true)); err != nil {
			return nil, err
		}
	}

	return &protos.ReferenceDataBulkWriteResp{ReferenceData: req.ReferenceData}, nil
}
