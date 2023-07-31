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

func HandleRegionOfInterestGetReq(req *protos.RegionOfInterestGetReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ROIItem](false, req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.RegionOfInterestGetResp{
		RegionOfInterest: dbItem,
	}, nil
}

func HandleRegionOfInterestDeleteReq(req *protos.RegionOfInterestDeleteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.RegionOfInterestDeleteResp](req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
}

func HandleRegionOfInterestListReq(req *protos.RegionOfInterestListReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_ROI, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}

	// Since we want only summary data, specify less fields to retrieve
	opts := options.Find().SetProjection(bson.D{
		{"_id", true},
		{"scanid", true},
		{"name", true},
		{"description", true},
		{"imagename", true},
		{"tags", true},
		{"modifiedunixsec", true},
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
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}

		// Add to map
		rois[item.Id] = item
	}

	return &protos.RegionOfInterestListResp{
		RegionsOfInterest: rois,
	}, nil
}

func validateROI(roi *protos.ROIItem) error {
	if err := wsHelpers.CheckStringField(&roi.Name, "Name", 1, 50); err != nil {
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

	if err := wsHelpers.CheckFieldLength(roi.Tags, "Tags", 0, 10); err != nil {
		return err
	}

	return nil
}

func createROI(roi *protos.ROIItem, hctx wsHelpers.HandlerContext) (*protos.ROIItem, error) {
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

	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).InsertOne(sessCtx, roi)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem)
		if _err != nil {
			return nil, _err
		}
		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)

	if err != nil {
		return nil, err
	}

	roi.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
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

	if len(roi.Description) > 0 {
		dbItem.Description = roi.Description
		update = append(update, bson.E{Key: "description", Value: roi.Description})
	}

	if len(roi.ImageName) > 0 {
		dbItem.ImageName = roi.ImageName
		update = append(update, bson.E{Key: "imagename", Value: roi.ImageName})
	}

	if len(roi.ScanEntryIndexesEncoded) > 0 {
		dbItem.ScanEntryIndexesEncoded = roi.ScanEntryIndexesEncoded
		update = append(update, bson.E{Key: "ScanEntryIndexesEncoded", Value: roi.ScanEntryIndexesEncoded})
	}

	if len(roi.PixelIndexesEncoded) > 0 {
		dbItem.PixelIndexesEncoded = roi.PixelIndexesEncoded
		update = append(update, bson.E{Key: "pixelindexesencoded", Value: roi.PixelIndexesEncoded})
	}

	if len(roi.Tags) > 0 {
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
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return dbItem, nil
}

func HandleRegionOfInterestWriteReq(req *protos.RegionOfInterestWriteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestWriteResp, error) {
	// Owner should never be accepted from API
	if req.RegionOfInterest.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	var item *protos.ROIItem
	var err error

	if len(req.RegionOfInterest.Id) <= 0 {
		item, err = createROI(req.RegionOfInterest, hctx)
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
