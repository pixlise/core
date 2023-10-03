package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type IdOnly struct {
	Id string `bson:"_id"`
}

func HandleRegionOfInterestGetReq(req *protos.RegionOfInterestGetReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ROIItem](false, req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	if req.IsMIST {
		// Fetch from MIST table and add to dbItem
		mistItem := &protos.MistROIItem{}
		err = hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).FindOne(context.TODO(), bson.D{{"_id", req.Id}}).Decode(&mistItem)
		if err != nil {
			return nil, err
		}

		dbItem.MistROIItem = mistItem
	}

	return &protos.RegionOfInterestGetResp{
		RegionOfInterest: dbItem,
	}, nil
}

func HandleRegionOfInterestDeleteReq(req *protos.RegionOfInterestDeleteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestDeleteResp, error) {
	if req.IsMIST {
		// Delete from MIST table
		_, err := hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).DeleteOne(context.TODO(), bson.D{{"_id", req.Id}})
		if err != nil {
			return nil, err
		}
	}

	return wsHelpers.DeleteUserObject[protos.RegionOfInterestDeleteResp](req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
}

func HandleRegionOfInterestListReq(req *protos.RegionOfInterestListReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestListResp, error) {
	filter, idToOwner, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_ROI, hctx)
	if err != nil {
		return nil, err
	}

	// Add MIST ROI filter
	filter["ismist"] = req.IsMIST

	// Since we want only summary data, specify less fields to retrieve
	opts := options.Find().SetProjection(bson.D{
		{"_id", true},
		{"scanid", true},
		{"name", true},
		{"description", true},
		{"imagename", true},
		{"tags", true},
		{"modifiedunixsec", true},
		{"ismist", true},
	})

	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ROIItemSummary{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Need to return this as a map
	rois := map[string]*protos.ROIItemSummary{}
	for _, item := range items {

		// Look up owner info
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}

		if req.IsMIST {
			// Fetch from MIST table and add to item
			mistItem := &protos.MistROIItem{}
			err = hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).FindOne(context.TODO(), bson.D{{"_id", item.Id}}).Decode(&mistItem)
			if err != nil {
				return nil, err
			}

			item.MistROIItem = mistItem
		}

		// Add to map
		rois[item.Id] = item
	}

	return &protos.RegionOfInterestListResp{
		RegionsOfInterest: rois,
	}, nil
}

func validateROI(roi *protos.ROIItem) error {
	if err := wsHelpers.CheckStringField(&roi.Name, "Name", 1, 100); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&roi.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&roi.Description, "Description", 0, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&roi.ImageName, "ImageName", 0, 255); err != nil {
		return err
	}
	if err := wsHelpers.CheckFieldLength(roi.ScanEntryIndexesEncoded, "ScanEntryIndexesEncoded", 0, 100000); err != nil {
		return err
	}
	if err := wsHelpers.CheckFieldLength(roi.PixelIndexesEncoded, "PixelIndexesEncoded", 0, 100000); err != nil {
		return err
	}

	// Can't both be empty!
	if len(roi.ScanEntryIndexesEncoded) <= 0 && len(roi.PixelIndexesEncoded) <= 0 {
		return errors.New("ROI must have location or pixel indexes defined")
	}

	// Can't have image without indexes
	if (len(roi.ImageName) > 0) != (len(roi.PixelIndexesEncoded) > 0) {
		return errors.New("ROI image and pixel indexes must both be defined")
	}

	if err := wsHelpers.CheckFieldLength(roi.Tags, "Tags", 0, wsHelpers.TagListMaxLength); err != nil {
		return err
	}

	return nil
}

func createROI(roi *protos.ROIItem, hctx wsHelpers.HandlerContext, needMistEntry bool) (*protos.ROIItem, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateROI(roi)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	id := hctx.Svcs.IDGen.GenObjectID()
	roi.Id = id

	// We need to create an ownership item along with it
	ownerItem, err := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_ROI, hctx)
	if err != nil {
		return nil, err
	}

	roi.ModifiedUnixSec = ownerItem.CreatedUnixSec

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	var mistROIItem *protos.MistROIItem
	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// We don't want to write this to the ROI table
		mistROIItem = roi.MistROIItem
		roi.MistROIItem = nil

		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).InsertOne(sessCtx, roi)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem)
		if _err != nil {
			return nil, _err
		}

		if needMistEntry {
			mistROIItem = &protos.MistROIItem{
				Id:                  roi.Id,
				ScanId:              roi.ScanId,
				Species:             mistROIItem.Species,
				MineralGroupID:      mistROIItem.MineralGroupID,
				IdDepth:             mistROIItem.IdDepth,
				ClassificationTrail: mistROIItem.ClassificationTrail,
				Formula:             mistROIItem.Formula,
			}
			// Add an entry into the MIST ROI table
			_, err := hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).InsertOne(context.TODO(), mistROIItem)
			if err != nil {
				return nil, err
			}

			roi.MistROIItem = mistROIItem
		}

		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return nil, err
	}

	roi.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	return roi, nil
}

func updateROI(roi *protos.ROIItem, hctx wsHelpers.HandlerContext) (*protos.ROIItem, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ROIItem](true, roi.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
	if err != nil {
		return nil, err
	}

	// Some fields can't change
	if len(roi.ScanId) > 0 && dbItem.ScanId != roi.ScanId {
		return nil, errors.New("ScanId cannot be changed")
	}

	// Update fields
	update := bson.D{}
	if len(roi.Name) > 0 {
		dbItem.Name = roi.Name
		update = append(update, bson.E{Key: "name", Value: roi.Name})
	}

	if dbItem.Description != roi.Description {
		dbItem.Description = roi.Description
		update = append(update, bson.E{Key: "description", Value: roi.Description})
	}

	if len(roi.ImageName) > 0 {
		dbItem.ImageName = roi.ImageName
		update = append(update, bson.E{Key: "imagename", Value: roi.ImageName})
	}

	// Once created, these can't be set to empty
	if roi.ScanEntryIndexesEncoded != nil && !utils.SlicesEqual(dbItem.ScanEntryIndexesEncoded, roi.ScanEntryIndexesEncoded) {
		dbItem.ScanEntryIndexesEncoded = roi.ScanEntryIndexesEncoded
		update = append(update, bson.E{Key: "ScanEntryIndexesEncoded", Value: roi.ScanEntryIndexesEncoded})
	}

	// Once created, these can't be set to empty
	if roi.PixelIndexesEncoded != nil && !utils.SlicesEqual(dbItem.PixelIndexesEncoded, roi.PixelIndexesEncoded) {
		dbItem.PixelIndexesEncoded = roi.PixelIndexesEncoded
		update = append(update, bson.E{Key: "pixelindexesencoded", Value: roi.PixelIndexesEncoded})
	}

	// Tags are a summary field, so are expected to be passed with every request
	if !utils.SlicesEqual(dbItem.Tags, roi.Tags) {
		dbItem.Tags = roi.Tags
		update = append(update, bson.E{Key: "tags", Value: roi.Tags})
	}

	// Validate it
	err = validateROI(dbItem)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Update modified time
	dbItem.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update = append(update, bson.E{Key: "modifiedunixsec", Value: dbItem.ModifiedUnixSec})

	// It's valid, update the DB
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).UpdateByID(ctx, roi.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("ROI UpdateByID result had unexpected counts %+v id: %v", result, roi.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return dbItem, nil
}

func HandleRegionOfInterestWriteReq(req *protos.RegionOfInterestWriteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestWriteResp, error) {
	// Owner should never be accepted from API
	if req.RegionOfInterest.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	if req.IsMIST {
		req.RegionOfInterest.IsMIST = true
	}

	var item *protos.ROIItem

	var err error
	if len(req.RegionOfInterest.Id) <= 0 {
		item, err = createROI(req.RegionOfInterest, hctx, req.IsMIST)
		if err != nil {
			return nil, err
		}
	} else {
		item, err = updateROI(req.RegionOfInterest, hctx)
	}
	if err != nil {
		return nil, err
	}

	return &protos.RegionOfInterestWriteResp{
		RegionOfInterest: item,
	}, nil
}

func HandleRegionOfInterestBulkWriteReq(req *protos.RegionOfInterestBulkWriteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestBulkWriteResp, error) {
	if req.IsMIST && req.MistROIScanIdsToDelete != nil && len(req.MistROIScanIdsToDelete) > 0 {
		ctx := context.TODO()
		coll := hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName)
		filter := bson.M{"scanid": bson.M{"$in": req.MistROIScanIdsToDelete}}
		opts := options.Find().SetProjection(bson.M{"_id": true})

		cursor, err := coll.Find(ctx, filter, opts)
		if err != nil {
			return nil, err
		}

		mistIds := []*IdOnly{}
		err = cursor.All(ctx, &mistIds)
		if err != nil {
			return nil, err
		}

		mistIdList := []string{}
		for _, item := range mistIds {
			mistIdList = append(mistIdList, item.Id)
		}

		// Delete all the MIST ROIs for this scan
		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": mistIdList}})
		if err != nil {
			return nil, err
		}

		// Delete all the ROIs associated with the MIST ROIs for this scan
		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": mistIdList}})
		if err != nil {
			return nil, err
		}
	}

	writtenROIs := []*protos.ROIItem{}
	for _, item := range req.RegionsOfInterest {
		item.Owner = nil
		item.IsMIST = req.IsMIST

		var err error
		if len(item.Id) > 0 && req.Overwrite {
			// Overwrite existing ROI
			item, err = updateROI(item, hctx)
			if err != nil {
				return nil, err
			}
		} else if req.SkipDuplicates {
			// If id is not empty, but we're not overwriting, so skip this ROI
			// If id is empty and this is a MIST ROI, we need to check if this ROI already exists
			if req.IsMIST && len(item.MistROIItem.ClassificationTrail) > 0 {
				// Skip ROIs with same classification trail for the same scan
				filter := bson.M{"scanid": item.ScanId, "classificationtrail": item.MistROIItem.ClassificationTrail}
				opts := options.Find().SetProjection(bson.M{"_id": true})
				cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).Find(context.TODO(), filter, opts)
				if err != nil {
					return nil, err
				}

				ids := []*IdOnly{}
				err = cursor.All(context.TODO(), &ids)
				if err != nil {
					return nil, err
				}

				// If we found an ROI with the same classification trail, then don't create a new one
				if len(ids) > 0 {
					continue
				} else {
					// Create new ROI
					item, err = createROI(item, hctx, req.IsMIST)
					if err != nil {
						return nil, err
					}
				}
			}

		} else {
			// Create new ROI
			item, err = createROI(item, hctx, req.IsMIST)
			if err != nil {
				return nil, err
			}
		}

		writtenROIs = append(writtenROIs, item)
	}

	return &protos.RegionOfInterestBulkWriteResp{
		RegionsOfInterest: writtenROIs,
	}, nil
}

func HandleRegionOfInterestBulkDuplicateReq(req *protos.RegionOfInterestBulkDuplicateReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestBulkDuplicateResp, error) {
	// Get the ROIs to duplicate
	filter := bson.M{"_id": bson.M{"$in": req.Ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ROIItem{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	roiSummaries := map[string]*protos.ROIItemSummary{}
	// Create new ROIs
	for _, item := range items {
		item.Id = ""
		item.Owner = nil
		item.IsMIST = req.IsMIST

		// Create new ROI
		newROI, err := createROI(item, hctx, req.IsMIST)
		if err != nil {
			return nil, err
		}

		// Add summary to list
		roiSummaries[newROI.Id] = &protos.ROIItemSummary{
			Id:              newROI.Id,
			Name:            newROI.Name,
			ScanId:          newROI.ScanId,
			Description:     newROI.Description,
			ImageName:       newROI.ImageName,
			Tags:            newROI.Tags,
			ModifiedUnixSec: newROI.ModifiedUnixSec,
			IsMIST:          newROI.IsMIST,
		}
	}

	return &protos.RegionOfInterestBulkDuplicateResp{
		RegionsOfInterest: roiSummaries,
	}, nil
}
