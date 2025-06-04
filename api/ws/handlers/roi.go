package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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

	if req.IsMIST && dbItem.IsMIST {
		// Fetch from MIST table and add to dbItem
		mistItem := &protos.MistROIItem{}
		err = hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).FindOne(context.TODO(), bson.D{{Key: "_id", Value: req.Id}}).Decode(&mistItem)
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
		_, err := hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).DeleteOne(context.TODO(), bson.D{{Key: "_id", Value: req.Id}})
		if err != nil {
			return nil, err
		}
	}

	// If the specified id is to be treated as an associatedROIId, we have to delete all ROIs that contain that id!
	if req.IsAssociatedROIId {
		// Use the delete version that allows specifying field and allows multiple items to be deleted
		delCount, err := wsHelpers.DeleteUserObjectByIdField("associatedroiid", req.Id, protos.ObjectType_OT_ROI, false, dbCollections.RegionsOfInterestName, hctx)
		if err != nil {
			return nil, err
		}

		return &protos.RegionOfInterestDeleteResp{
			DeleteCount: uint32(delCount),
		}, nil
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
		{Key: "_id", Value: true},
		{Key: "scanid", Value: true},
		{Key: "name", Value: true},
		{Key: "description", Value: true},
		{Key: "imagename", Value: true},
		{Key: "tags", Value: true},
		{Key: "modifiedunixsec", Value: true},
		{Key: "ismist", Value: true},
		{Key: "associatedroiid", Value: true},
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
			err = hctx.Svcs.MongoDB.Collection(dbCollections.MistROIsName).FindOne(context.TODO(), bson.D{{Key: "_id", Value: item.Id}}).Decode(&mistItem)
			if err != nil {
				fmt.Printf("Error decoding MIST ROI item (%v) during listing: %v\n", item.Id, err)
				sentry.CaptureMessage(fmt.Sprintf("Error decoding MIST ROI item (%v) during listing: %v\n", item.Id, err))
			} else {
				item.MistROIItem = mistItem
			}
		}

		// Look up display settings and add to item if found (otherwise leave nil)
		userROIId := formROIUserConfigID(hctx.SessUser.User, item.Id)
		displaySettings := &protos.ROIItemDisplaySettings{}
		err = hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).FindOne(context.TODO(), bson.D{{Key: "_id", Value: userROIId}}).Decode(&displaySettings)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}

		if err != mongo.ErrNoDocuments && displaySettings != nil {
			item.DisplaySettings = displaySettings
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

// id can be empty, then one is generated otherwise specify one!
func createROI(roi *protos.ROIItem, id string, hctx wsHelpers.HandlerContext, needMistEntry bool, editors *protos.UserGroupList, viewers *protos.UserGroupList) (*protos.ROIItem, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateROI(roi)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id if needed
	if len(id) <= 0 {
		id = hctx.Svcs.IDGen.GenObjectID()
	}
	roi.Id = id

	// We need to create an ownership item along with it
	ownerItem := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_ROI, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())
	if editors != nil {
		if ownerItem.Editors == nil {
			ownerItem.Editors = &protos.UserGroupList{}
		}

		ownerItem.Editors.UserIds = editors.UserIds
		ownerItem.Editors.GroupIds = editors.GroupIds
	}

	if viewers != nil {
		if ownerItem.Viewers == nil {
			ownerItem.Viewers = &protos.UserGroupList{}
		}
		ownerItem.Viewers.UserIds = viewers.UserIds
		ownerItem.Viewers.GroupIds = viewers.GroupIds
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
				PmcConfidenceMap:    mistROIItem.PmcConfidenceMap,
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

func updateROI(roi *protos.ROIItem, hctx wsHelpers.HandlerContext, editors *protos.UserGroupList, viewers *protos.UserGroupList) (*protos.ROIItem, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ROIItem](true, roi.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
	if err != nil {
		return nil, err
	}

	// Check if we need to update the ownership
	if editors != nil || viewers != nil {
		if editors != nil {
			if owner.Editors == nil {
				owner.Editors = &protos.UserGroupList{}
			}

			owner.Editors.UserIds = editors.UserIds
			owner.Editors.GroupIds = editors.GroupIds
		}

		if viewers != nil {
			if owner.Viewers == nil {
				owner.Viewers = &protos.UserGroupList{}
			}

			owner.Viewers.UserIds = viewers.UserIds
			owner.Viewers.GroupIds = viewers.GroupIds
		}

		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).UpdateByID(ctx, roi.Id, bson.D{{Key: "$set", Value: owner}})
		if err != nil {
			return nil, err
		}

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
		update = append(update, bson.E{Key: "scanentryindexesencoded", Value: roi.ScanEntryIndexesEncoded})
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
		hctx.Svcs.Log.Errorf("ROI UpdateByID result had unexpected counts %v id: %v", result, roi.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	// Notify out that this ROI has changed, in case any UIs are interested
	hctx.Svcs.Notifier.SysNotifyROIChanged(roi.Id)

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
		item, err = createROI(req.RegionOfInterest, "", hctx, req.IsMIST, nil, nil)
		if err != nil {
			return nil, err
		}
	} else {
		item, err = updateROI(req.RegionOfInterest, hctx, nil, nil)
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

		// Delete the ownership items for the MIST ROIs
		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).DeleteMany(context.TODO(), bson.M{"_id": bson.M{"$in": mistIdList}})
		if err != nil {
			return nil, err
		}

		// Delete the ROI display settings for the MIST ROIs
		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).DeleteMany(context.TODO(), bson.M{"_id": bson.M{"$in": mistIdList}})
		if err != nil {
			return nil, err
		}
	}

	editors := &protos.UserGroupList{
		UserIds:  []string{},
		GroupIds: []string{},
	}
	viewers := &protos.UserGroupList{
		UserIds:  []string{},
		GroupIds: []string{},
	}

	if req.Editors != nil {
		editors.UserIds = req.Editors.UserIds
		editors.GroupIds = req.Editors.GroupIds
	}

	if req.Viewers != nil {
		viewers.UserIds = req.Viewers.UserIds
		viewers.GroupIds = req.Viewers.GroupIds
	}

	writtenROIs := []*protos.ROIItem{}
	associatedROIId := ""
	for c, item := range req.RegionsOfInterest {
		item.Owner = nil
		item.IsMIST = req.IsMIST

		var err error
		if len(item.Id) > 0 && req.Overwrite && (req.MistROIScanIdsToDelete == nil || len(req.MistROIScanIdsToDelete) == 0) {
			// Overwrite existing ROI
			item, err = updateROI(item, hctx, editors, viewers)
			if err != nil {
				return nil, err
			}
		} else if req.Overwrite && len(item.Id) <= 0 && req.IsMIST && item.MistROIItem != nil {
			// We're overwriting by name, so we need to find the existing ROI
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

			if len(ids) > 0 {
				// Overwrite existing ROI
				item.Id = ids[0].Id
				item, err = updateROI(item, hctx, editors, viewers)
				if err != nil {
					return nil, err
				}
			} else {
				// Create new ROI
				item, err = createROI(item, "", hctx, req.IsMIST, editors, viewers)
				if err != nil {
					return nil, err
				}
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
					item, err = createROI(item, "", hctx, req.IsMIST, editors, viewers)
					if err != nil {
						return nil, err
					}
				}
			}

		} else {
			// Create new ROI
			// NOTE: if writing multiple and they need to be associated in future, we specify the first id
			//       and set that as the associated id in all items written too
			id := ""
			if len(req.RegionsOfInterest) > 1 {
				if c == 0 {
					associatedROIId = hctx.Svcs.IDGen.GenObjectID()
					id = associatedROIId
				}

				item.AssociatedROIId = associatedROIId
			}
			item, err = createROI(item, id, hctx, req.IsMIST, editors, viewers)
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
		newROI, err := createROI(item, hctx, req.IsMIST, nil, nil)
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
			AssociatedROIId: newROI.AssociatedROIId,
		}
	}

	return &protos.RegionOfInterestBulkDuplicateResp{
		RegionsOfInterest: roiSummaries,
	}, nil
}

func formROIUserConfigID(user *protos.UserInfo, roiId string) string {
	return user.Id + "-" + roiId
}

func HandleRegionOfInterestDisplaySettingsWriteReq(req *protos.RegionOfInterestDisplaySettingsWriteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestDisplaySettingsWriteResp, error) {
	// Check that we have an id, current user, and display settings
	if len(req.Id) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("ROI ID must be specified"))
	}

	if req.DisplaySettings == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("DisplaySettings must be specified"))
	}

	if hctx.SessUser.User.Id == "" {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("User must be logged in"))
	}

	userROIId := formROIUserConfigID(hctx.SessUser.User, req.Id)

	// Check if the ROI display settings already exist
	filter := bson.M{"_id": userROIId}
	opts := options.Find().SetProjection(bson.M{"_id": true})
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	existingIds := []*IdOnly{}
	err = cursor.All(context.TODO(), &existingIds)
	if err != nil {
		return nil, err
	}

	// Update the display settings
	ctx := context.TODO()
	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var err error

		if len(existingIds) > 0 {
			_, err = hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).UpdateByID(sessCtx, userROIId, bson.D{{
				Key: "$set",
				Value: bson.D{
					{Key: "colour", Value: req.DisplaySettings.Colour},
					{Key: "shape", Value: req.DisplaySettings.Shape},
				},
			}})
		} else {
			_, err = hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).InsertOne(sessCtx, &protos.ROIItemDisplaySettings{
				Id:     userROIId,
				Colour: req.DisplaySettings.Colour,
				Shape:  req.DisplaySettings.Shape,
			})
		}

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return nil, err
	}

	return &protos.RegionOfInterestDisplaySettingsWriteResp{
		DisplaySettings: req.DisplaySettings,
	}, nil
}

func HandleRegionOfInterestDisplaySettingsGetReq(req *protos.RegionOfInterestDisplaySettingsGetReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestDisplaySettingsGetResp, error) {
	// Check that we have an id, current user, and display settings
	if len(req.Id) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("ROI ID must be specified"))
	}

	if hctx.SessUser.User.Id == "" {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("User must be logged in"))
	}

	// Get the display settings
	userROIId := formROIUserConfigID(hctx.SessUser.User, req.Id)
	displaySettings := &protos.ROIItemDisplaySettings{}
	err := hctx.Svcs.MongoDB.Collection(dbCollections.UserROIDisplaySettings).FindOne(context.Background(), bson.M{"_id": userROIId}).Decode(&displaySettings)
	if err != nil {
		return nil, err
	}

	return &protos.RegionOfInterestDisplaySettingsGetResp{
		DisplaySettings: displaySettings,
	}, nil
}
