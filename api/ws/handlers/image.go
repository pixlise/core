package wsHandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"net/http"
	"path"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleImageListReq(req *protos.ImageListReq, hctx wsHelpers.HandlerContext) (*protos.ImageListResp, error) {
	if err := wsHelpers.CheckFieldLength(req.ScanIds, "ScanIds", 1, 10); err != nil {
		return nil, err
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range req.ScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	// OK all scans are accessible, so read images
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	criteria := "$in"
	if req.MustIncludeAll {
		criteria = "$all"
	}

	filter := bson.M{"associatedscanids": bson.M{criteria: req.ScanIds}}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ScanImage{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	return &protos.ImageListResp{
		Images: items,
	}, nil
}

func HandleImageGetReq(req *protos.ImageGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.ImageName, "ImageName", 1, 255); err != nil {
		return nil, err
	}

	// Look up the image in DB to determine scan IDs, then determine ownership
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	filter := bson.M{"_id": req.ImageName}
	result := coll.FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.ImageName)
		}
		return nil, result.Err()
	}
	img := protos.ScanImage{}
	err := result.Decode(&img)
	if err != nil {
		return nil, err
	}

	// Now look up any associated ids
	if len(img.AssociatedScanIds) <= 0 {
		return nil, fmt.Errorf("Failed to find scan associated with image: %v", req.ImageName)
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range img.AssociatedScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			handled := false
			switch e := err.(type) {
			case errorwithstatus.Error:
				if e.Status() == http.StatusNotFound {
					// Log the error instead
					hctx.Svcs.Log.Errorf("ImageGetReq: Scan %v doesn't exist when checking for user access, allowing in case of scan not existing")
					handled = true
				}
			}

			if !handled {
				return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("User cannot access scan %v associated with image %v. Error: %v", scanId, req.ImageName, err))
			}
		}
	}

	return &protos.ImageGetResp{
		Image: &img,
	}, nil
}

func HandleImageGetDefaultReq(req *protos.ImageGetDefaultReq, hctx wsHelpers.HandlerContext) (*protos.ImageGetDefaultResp, error) {
	/*if err := wsHelpers.CheckFieldLength(req.ScanIds, "ScanIds", 1, 10); err != nil {
		return nil, err
	}*/
	if len(req.ScanIds) <= 0 {
		return &protos.ImageGetDefaultResp{
			DefaultImagesPerScanId: map[string]string{},
		}, nil
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range req.ScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanDefaultImagesName)

	filter := bson.M{"_id": bson.M{"$in": req.ScanIds}}
	opts := options.Find()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ScanImageDefaultDB{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, item := range items {
		result[item.ScanId] = item.DefaultImageFileName
	}

	return &protos.ImageGetDefaultResp{
		DefaultImagesPerScanId: result,
	}, nil
}

func HandleImageSetDefaultReq(req *protos.ImageSetDefaultReq, hctx wsHelpers.HandlerContext) (*protos.ImageSetDefaultResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	if err := wsHelpers.CheckStringField(&req.DefaultImageFileName, "DefaultImageFileName", 1, 255); err != nil {
		return nil, err
	}

	// Write to DB
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanDefaultImagesName)

	filter := bson.D{{Key: "_id", Value: req.ScanId}}
	opt := options.Update().SetUpsert(true)

	data := bson.D{{Key: "$set", Value: bson.D{{Key: "defaultimagefilename", Value: req.DefaultImageFileName}}}}

	result, err := coll.UpdateOne(ctx, filter, data, opt)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 && result.UpsertedCount != 1 {
		hctx.Svcs.Log.Errorf("ImageSetDefaultReq UpdateOne result had unexpected counts %+v id: %v", result, req.ScanId)
	}

	// Send out notifications so caches can be cleared
	hctx.Svcs.Notifier.SysNotifyScanImagesChanged([]string{req.ScanId})

	return &protos.ImageSetDefaultResp{}, nil
}

func HandleImageDeleteReq(req *protos.ImageDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ImageDeleteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 255); err != nil {
		return nil, err
	}

	ctx := context.TODO()

	// Get image meta so we have all info we need
	filterImg := bson.M{"_id": req.Name}
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	result := coll.FindOne(ctx, filterImg)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Name)
		}
		return nil, result.Err()
	}

	img := protos.ScanImage{}
	err := result.Decode(&img)
	if err != nil {
		return nil, err
	}

	// If it's the default image in any scan, we can't delete it
	filter := bson.D{{Key: "defaultimagefilename", Value: img.Path}}
	opt := options.Find()
	coll = hctx.Svcs.MongoDB.Collection(dbCollections.ScanDefaultImagesName)

	cursor, err := coll.Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}

	items := []*protos.ScanImageDefaultDB{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// If we have any, it's an error
	if len(items) > 0 {
		list := []string{}
		for _, item := range items {
			list = append(list, item.ScanId)
		}
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Cannot delete image: \"%v\" because it is the default image for scans: [%v]", req.Name, strings.Join(list, ",")))
	}

	// Delete anything related to this image
	s3Path := filepaths.GetImageFilePath(img.Path)
	err = hctx.Svcs.FS.DeleteObject(hctx.Svcs.Config.DatasetsBucket, s3Path)
	if err != nil {
		// Just log, but continue
		hctx.Svcs.Log.Errorf("Delete image %v - failed to delete s3://%v/%v: %v", req.Name, hctx.Svcs.Config.DatasetsBucket, s3Path, err)
	}

	// And the cached files
	files, err := hctx.Svcs.FS.ListObjects(hctx.Svcs.Config.DatasetsBucket, path.Join(filepaths.DatasetImageCacheRoot, img.Path))
	for _, fileName := range files {
		err = hctx.Svcs.FS.DeleteObject(hctx.Svcs.Config.DatasetsBucket, fileName)
		if err != nil {
			// Just log, but continue
			hctx.Svcs.Log.Errorf("Delete cached image %v - failed to delete s3://%v/%v: %v", req.Name, hctx.Svcs.Config.DatasetsBucket, fileName, err)
		}
	}

	// Delete from Images collection
	coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	filter = bson.D{{Key: "_id", Value: req.Name}}
	delOpt := options.Delete()
	_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
	if err != nil {
		return nil, err
	}

	//Verify delImgResult.DeletedCount == 1 ???

	// For any associated scans or origin scans, we send notify out
	scanIds := []string{}
	for _, assocScanId := range img.AssociatedScanIds {
		scanIds = append(scanIds, assocScanId)
	}

	if !utils.ItemInSlice(img.OriginScanId, img.AssociatedScanIds) {
		scanIds = append(scanIds, img.OriginScanId)
	}

	hctx.Svcs.Notifier.SysNotifyScanImagesChanged(scanIds)

	return &protos.ImageDeleteResp{}, nil
}

func HandleImageUploadReq(req *protos.ImageUploadReq, hctx wsHelpers.HandlerContext) (*protos.ImageUploadResp, error) {
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 255); err != nil {
		return nil, err
	}

	// We only allow a few formats:
	nameLowerCase := strings.ToLower(req.Name)
	if !strings.HasSuffix(nameLowerCase, ".png") && !strings.HasSuffix(nameLowerCase, ".jpg") && !strings.HasSuffix(nameLowerCase, ".tif") {
		return nil, fmt.Errorf("Unexpected format: %v. Must be either PNG, JPG or 32bit float 4-channel TIF file", req.Name)
	}

	if err := wsHelpers.CheckStringField(&req.OriginScanId, "OriginScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	if err := wsHelpers.CheckFieldLength(req.AssociatedScanIds, "AssociatedScanIds", 0, 10); err != nil {
		return nil, err
	}

	// Check that user has access to this scan
	_, err := wsHelpers.CheckObjectAccess(false, req.OriginScanId, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	db := hctx.Svcs.MongoDB

	// Save image meta in collection
	img, _, err := image.Decode(bytes.NewReader(req.ImageData))
	if err != nil {
		return nil, err
	}

	purpose := protos.ScanImagePurpose_SIP_VIEWING
	if strings.HasSuffix(nameLowerCase, ".tif") {
		purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
	}

	associatedScanIds := []string{req.OriginScanId}
	if len(req.AssociatedScanIds) > 0 {
		associatedScanIds = req.AssociatedScanIds
	}

	// We make the names more unique this way...
	saveName := req.OriginScanId + "-" + req.Name
	savePath := path.Join(req.OriginScanId, saveName)
	scanImage := utils.MakeScanImage(
		saveName,
		savePath,
		uint32(len(req.ImageData)),
		protos.ScanImageSource_SI_UPLOAD,
		purpose,
		associatedScanIds,
		req.OriginScanId,
		"",
		req.GetBeamImageRef(),
		img)

	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagesName)

	// If this is the first image added to a dataset that has no images (and hence no beam location ij's), generate ij's here so the image can be
	// aligned to them. The image will refer to itself as the owner of the ij's it's matching and will be able to have a transform too
	foundItems, err := coll.Find(ctx, bson.M{"originscanid": req.OriginScanId}, options.Find())
	// If there was an error, stop here
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("Error while querying for other images for scan %v. Error was: %v", req.OriginScanId, err)
	}

	generateCoords := err == mongo.ErrNoDocuments // This won't really happen... Find() doesn't return an error for none!
	if !generateCoords {
		// Check if the count is 0
		generateCoords = !foundItems.Next(ctx)
	}

	if generateCoords && scanImage.MatchInfo == nil {
		// Set a beam transform
		scanImage.MatchInfo = &protos.ImageMatchTransform{
			BeamImageFileName: saveName,
			XOffset:           0,
			YOffset:           0,
			XScale:            1,
			YScale:            1,
		}
	}

	result, err := coll.InsertOne(ctx, scanImage, options.InsertOne())
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("%v already exists", scanImage.Name))
		}
		return nil, err
	}

	if result.InsertedID != scanImage.Name {
		return nil, fmt.Errorf("HandleImageUploadReq wrote id %v, got back %v", scanImage.Name, result.InsertedID)
	}

	// Save the image to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.DatasetsBucket, s3Path, req.ImageData)
	if err != nil {
		// Failed to upload image data, so no point in having a DB entry now either...
		coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
		filter := bson.D{{Key: "_id", Value: saveName}}
		delOpt := options.Delete()
		_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
		return nil, err
	}

	if generateCoords {
		err = generateIJs(saveName, req.OriginScanId, hctx)
		if err != nil {
			return nil, err
		}
	}

	// Notify of our successful image addition
	hctx.Svcs.Notifier.NotifyNewScanImage(req.OriginScanId, req.OriginScanId, saveName)
	hctx.Svcs.Notifier.SysNotifyScanImagesChanged([]string{req.OriginScanId})

	return &protos.ImageUploadResp{}, nil
}

func HandleImageSetMatchTransformReq(req *protos.ImageSetMatchTransformReq, hctx wsHelpers.HandlerContext) (*protos.ImageSetMatchTransformResp, error) {
	if err := wsHelpers.CheckStringField(&req.ImageName, "ImageName", 1, 255); err != nil {
		return nil, err
	}
	if req.Transform == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Transform must be set"))
	}

	if req.Transform.XScale <= 0 || req.Transform.YScale <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Transform must have positive scale values"))
	}

	// Read image
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	filter := bson.M{"_id": req.ImageName}
	result := coll.FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.ImageName)
		}
		return nil, result.Err()
	}

	img := protos.ScanImage{}
	err := result.Decode(&img)
	if err != nil {
		return nil, err
	}

	// Now look up any associated ids
	if len(img.AssociatedScanIds) <= 0 {
		return nil, fmt.Errorf("Failed to find scan associated with image: %v", req.ImageName)
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range img.AssociatedScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("User cannot access scan %v associated with image %v. Error: %v", scanId, req.ImageName, err))
		}
	}

	// Check that this is a matched image!
	if img.MatchInfo == nil {
		return nil, fmt.Errorf("Failed edit transform for image %v - it is not a matched image", req.ImageName)
	}

	// Make the change
	img.MatchInfo.XOffset = req.Transform.XOffset
	img.MatchInfo.YOffset = req.Transform.YOffset
	img.MatchInfo.XScale = req.Transform.XScale
	img.MatchInfo.YScale = req.Transform.YScale

	// Write it back
	opt := options.Update()
	data := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "matchinfo", Value: img.MatchInfo},
		}},
	}

	updResult, err := coll.UpdateOne(ctx, filter, data, opt)
	if err != nil {
		return nil, err
	}

	if updResult.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("ImageSetMatchTransformReq update result had unexpected match count %+v imageName: %v", updResult, req.ImageName)
	}

	return &protos.ImageSetMatchTransformResp{}, nil
}
