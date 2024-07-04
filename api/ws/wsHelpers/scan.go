package wsHelpers

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateScanImageDataTypes(scanId string, db *mongo.Database, logger logger.ILogger) error {
	if len(scanId) <= 0 {
		return errors.New("No scanId specified for UpdateScanImageDataTypes")
	}

	// Check how many RGBU images and normal images this scan has
	ctx := context.TODO()

	coll := db.Collection(dbCollections.ImagesName)
	filter := bson.M{"originscanid": scanId}
	opt := options.Find()
	cursor, err := coll.Find(ctx, filter, opt)
	if err != nil {
		return err
	}

	images := []*protos.ScanImage{}
	err = cursor.All(ctx, &images)
	if err != nil {
		return err
	}

	imageCount := uint32(0)
	rgbuCount := uint32(0)

	for _, img := range images {
		if img.Purpose == protos.ScanImagePurpose_SIP_MULTICHANNEL {
			rgbuCount++
		} else {
			imageCount++
		}
	}

	// Update the scan item
	summary, err := scan.ReadScanItem(scanId, db)
	if err != nil {
		return err
	}

	saveDataTypes := []*protos.ScanItem_ScanTypeCount{}
	for _, item := range summary.DataTypes {
		if item.DataType == protos.ScanDataType_SD_XRF {
			saveDataTypes = append(saveDataTypes, item)
		}
	}

	// Add these if needed
	if imageCount > 0 {
		saveDataTypes = append(saveDataTypes, &protos.ScanItem_ScanTypeCount{
			DataType: protos.ScanDataType_SD_IMAGE,
			Count:    imageCount,
		})
	}
	if rgbuCount > 0 {
		saveDataTypes = append(saveDataTypes, &protos.ScanItem_ScanTypeCount{
			DataType: protos.ScanDataType_SD_RGBU,
			Count:    rgbuCount,
		})
	}

	summary.DataTypes = saveDataTypes

	coll = db.Collection(dbCollections.ScansName)
	filter = bson.M{"_id": scanId}
	replaceResult, err := coll.ReplaceOne(ctx, filter, summary, options.Replace())
	if err != nil {
		return err
	}

	if replaceResult.ModifiedCount != 1 {
		logger.Infof("UpdateScanImageDataTypes for scan: %v expected ModifiedCount 1, got: %v", scanId, replaceResult.ModifiedCount)
	}
	return nil
}
