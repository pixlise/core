package wsHelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/vips"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ImportUploadedImage(req *protos.ImageImportReq, svcs *services.APIServices) error {
	ctx := context.TODO()

	saveName := req.Name

	// Read back the multi-part image to disk
	localFilePath, imgFileSize, err := svcs.MultiPartUpload.ReadCompletedUpload(saveName)
	if err != nil {
		return err
	}
	if imgFileSize == 0 {
		return fmt.Errorf("Uploaded image %v is empty", saveName)
	}

	// Check if it exists already as an image
	// TODO: Could this be moved to happen sooner? Hard to do this in a generic file upload step but still
	//       would be nice if the user uploading knew it will fail earlier??!!
	coll := svcs.MongoDB.Collection(dbCollections.ImagesName)
	findOneResult := coll.FindOne(ctx, bson.M{"_id": saveName}, options.FindOne())
	if findOneResult.Err() == nil {
		// If we don't have an error, we have this already
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Image \"%v\" already exists", saveName))
	} else {
		// We found an error, only stop for non-no-document errors. What we DO want is a no document error!
		if findOneResult.Err() != mongo.ErrNoDocuments {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to check if image \"%v\" already exists: %v", saveName, findOneResult.Err()))
		}
	}

	// Read the scan to confirm it's valid, and also so we have the instrument to save for beams (if needed)
	coll = svcs.MongoDB.Collection(dbCollections.ScansName)
	findOneResult = coll.FindOne(ctx, bson.M{"_id": req.OriginScanId}, options.FindOne())
	if findOneResult.Err() != nil {
		return errorwithstatus.MakeNotFoundError(req.OriginScanId)
	}

	scan := &protos.ScanItem{}
	err = findOneResult.Decode(scan)
	if err != nil {
		return fmt.Errorf("Failed to decode scan: %v. Error: %v", req.OriginScanId, err)
	}

	// Save image meta in collection
	purpose := protos.ScanImagePurpose_SIP_VIEWING
	if strings.HasSuffix(strings.ToLower(saveName), ".tif") {
		meta, err := gdsfilename.ParseFileName(saveName)
		if err == nil && meta.ProdType == "MSA" || meta.ProdType == "VIS" {
			// It's only considered an RGBU image based on strict file naming!
			purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
		}
	}

	associatedScanIds := []string{req.OriginScanId}
	if len(req.AssociatedScanIds) > 0 {
		associatedScanIds = req.AssociatedScanIds
	}

	// Read image dimensions using vips, this covers all cases of pngs/jpgs and large multi-layer/float32 tif images
	img, err := vips.NewImageFromFile(localFilePath, nil)
	if err != nil {
		return err
	}

	imgWidth := uint32(img.Width())
	imgHeight := uint32(img.Height())

	img.Close()

	// We make the names more unique this way...
	scanImage := utils.MakeScanImage(
		saveName,
		imgFileSize,
		protos.ScanImageSource_SI_UPLOAD,
		purpose,
		associatedScanIds,
		req.OriginScanId,
		"",
		req.GetBeamImageRef(),
		"",
		"",
		imgWidth,
		imgHeight,
	)

	asPyramid := isPyramidImport(scanImage.Width, scanImage.Height, scanImage.FileSize64)

	pyramidTileSize := uint32(1024)
	pyramidQuality := uint32(90)

	if asPyramid {
		// Set the id to be the image id
		scanImage.PyramidId = scanImage.ImagePath
		// Add other pyramid related fields

		scanImage.PyramidTileFormat = "jpg"
		if pyramidQuality >= 100 {
			scanImage.PyramidTileFormat = "png"
		}

		pyramidLevels := BuildImagePyramidLevels(scanImage.Width, scanImage.Height, pyramidTileSize)

		svcs.Log.Infof("Importing image %v (%vx%v) as pyramid with tile size: %v, levels: %v. Pyramid ID generated: %v", req.Name, scanImage.Width, scanImage.Height, pyramidLevels.TileSize, len(pyramidLevels.Pyramid), scanImage.PyramidId)

		err = savePyramidEntry(ctx, svcs.MongoDB, scanImage.PyramidId, pyramidLevels, svcs.Log)
		if err != nil {
			return err
		}
	}

	generateCoords, isDuplicate, err := saveScanImage(ctx, svcs.MongoDB, scan, scanImage)
	if err != nil {
		// If this isn't an "already exists" error, we delete what we've saved already. We don't want
		// to delete otherwise though, or we'd wipe out the one that already existed!
		// TODO: Probably should do this with transactions
		if !isDuplicate {
			undoScanImagePUT(ctx, svcs.MongoDB, req.Name, scanImage.ImagePath, scanImage.PyramidId, localFilePath, imgFileSize, svcs.Log)
		}
		return err
	}

	if asPyramid {
		err = saveImageDataAsPyramid(svcs.FS, svcs.Config.DatasetsBucket, saveName, localFilePath, pyramidTileSize, pyramidQuality, svcs.Log)
	} else {
		err = saveImageDataFromFile(svcs.FS, svcs.Config.DatasetsBucket, saveName, localFilePath)
	}

	if err != nil {
		undoScanImagePUT(ctx, svcs.MongoDB, req.Name, scanImage.ImagePath, scanImage.PyramidId, localFilePath, imgFileSize, svcs.Log)
		return err
	}

	// Delete the temp file now!
	err = os.Remove(localFilePath)
	if err != nil {
		svcs.Log.Errorf("Failed to delete uploaded image file cache for: %v. Error: %v", localFilePath, err)
		// No need to stop here though!
	}

	// From this point on, we consider the image created. If IJ generation fails or anything else, we don't roll back and delete

	if generateCoords {
		_, err = GenerateIJs(scanImage.ImagePath, req.OriginScanId, scan.Instrument, svcs)
		if err != nil {
			return err
		}
	}

	// Finally, update the scan if needed
	err = UpdateScanImageDataTypes(req.OriginScanId, svcs.MongoDB, svcs.Log)
	if err != nil {
		svcs.Log.Errorf("UpdateScanImageDataTypes Failed for scan: %v, when uploading image: %v. DataType counts may not be accurate on Scan Item, RGBU icon may not show correctly.", req.OriginScanId, scanImage.ImagePath)
	}

	// Notify of our successful image addition
	svcs.Notifier.NotifyNewScanImage(req.OriginScanId, req.OriginScanId, scanImage.ImagePath)
	svcs.Notifier.SysNotifyScanImagesChanged(scanImage.ImagePath, scanImage.AssociatedScanIds)

	return nil
}

func savePyramidEntry(ctx context.Context, db *mongo.Database, pyramidId string, pyramidInfo *protos.ImagePyramid, l logger.ILogger) error {
	entry := &protos.ImagePyramidDBEntry{
		Id:      pyramidId,
		Pyramid: pyramidInfo,
	}

	coll := db.Collection(dbCollections.ImagePyramidsName)
	opt := options.InsertOne()
	result, err := coll.InsertOne(ctx, entry, opt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Don't overwrite, so we're OK with this
			//return nil
			l.Errorf("insertImagePyramid writing pyramid structure to DB skipped due to duplicate pyramid id: %v", pyramidId)
		} else {
			// A real error happened!
			return err
		}
	} else if result.InsertedID != entry.Id {
		l.Errorf("insertImagePyramid wrote id %v, got back %v", entry.Id, result.InsertedID)
		// Not the end of the world... don't error out here
	}

	return nil
}

// Returns generate flag, is duplicate flag, and error
func saveScanImage(ctx context.Context, db *mongo.Database, scan *protos.ScanItem, scanImage *protos.ScanImage) (bool, bool, error) {
	coll := db.Collection(dbCollections.ImagesName)

	// If this is the first image added to a dataset that has no images (and hence no beam location ij's), generate ij's here so the image can be
	// aligned to them. The image will refer to itself as the owner of the ij's it's matching and will be able to have a transform too
	foundItems, err := coll.Find(ctx, bson.M{"originscanid": scanImage.OriginScanId}, options.Find())
	// If there was an error, stop here
	if err != nil && err != mongo.ErrNoDocuments {
		return false, false, fmt.Errorf("Error while querying for other images for scan %v. Error was: %v", scanImage.OriginScanId, err)
	}

	generateCoords := err == mongo.ErrNoDocuments // This won't really happen... Find() doesn't return an error for none!
	if !generateCoords {
		// Check if the count is 0
		generateCoords = !foundItems.Next(ctx)
	}

	hasXRF := false
	for _, dt := range scan.DataTypes {
		if dt.DataType == protos.ScanDataType_SD_XRF && dt.Count > 0 {
			hasXRF = true
			break
		}
	}

	if !hasXRF {
		generateCoords = false
	}

	if generateCoords && scanImage.MatchInfo == nil {
		// Set a beam transform
		scanImage.MatchInfo = &protos.ImageMatchTransform{
			BeamImageFileName: scanImage.ImagePath,
			XOffset:           0,
			YOffset:           0,
			XScale:            1,
			YScale:            1,
		}
	}

	result, err := coll.InsertOne(ctx, scanImage, options.InsertOne())
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, true, errorwithstatus.MakeBadRequestError(fmt.Errorf("%v already exists", scanImage.ImagePath))
		}
		return generateCoords, false, err
	}

	if result.InsertedID != scanImage.ImagePath {
		return generateCoords, false, fmt.Errorf("HandleImageUploadReq wrote id %v, got back %v", scanImage.ImagePath, result.InsertedID)
	}

	return generateCoords, false, nil
}

// Call to abort an image upload operation - if anything happens here we delete all the things that the
// image upload may have affected/saved/created
func undoScanImagePUT(ctx context.Context, db *mongo.Database, reqName string, scanImageId string, pyramidId string, localFilePath string, imageByteSize uint64, l logger.ILogger) {
	// DB Image
	coll := db.Collection(dbCollections.ImagesName)
	filter := bson.D{{Key: "_id", Value: scanImageId}}
	delOpt := options.Delete()
	_ /*delImgResult*/, err := coll.DeleteOne(ctx, filter, delOpt)
	if err != nil {
		l.Errorf("Failed to delete scan image %v while handling image upload error. Error was: %v", scanImageId, err)
	} else {
		l.Infof("Deleted imported scan image %v due to image upload error", scanImageId)
	}

	// DB ImagePyramid
	coll = db.Collection(dbCollections.ImagePyramidsName)
	filter = bson.D{{Key: "_id", Value: pyramidId}}
	delOpt = options.Delete()
	_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
	if err != nil {
		l.Errorf("Failed to delete scan image pyramid %v while handling image upload error. Error was: %v", pyramidId, err)
	} else {
		l.Infof("Deleted imported scan image pyramid %v due to image upload error", pyramidId)
	}

	// Local image file (containing uploaded chunks)
	if err2 := os.Remove(localFilePath); err2 != nil {
		l.Errorf("Failed to delete local copy of downloaded image %v, error: %v", localFilePath, err2)
	} else {
		l.Infof("Deleted temp download image %v, size: %v", localFilePath, imageByteSize)
	}
}

func saveImageDataAsPyramid(
	fs fileaccess.FileAccess,
	bucket string,
	savePath string,
	localImageDataPath string,
	tileSize uint32,
	tileQuality uint32,
	l logger.ILogger) error {
	// This is the most complex situation. We take the locally saved file (constructed from chunks sent) and
	// break it up into tiles. We write these to S3 as separate images in a tree structure

	// Read the image
	img, err := vips.NewImageFromFile(localImageDataPath, &vips.LoadOptions{ /*Page: , N: */ })
	if err != nil {
		return err
	}

	// Use vipsgen Dzsave
	// IMPORTANT: These options are carefully chosen:
	// - Imagename: The base name for generated files
	// - Suffix: .jpg for JPEG tiles
	// - Q: JPEG quality (85 = good balance of quality/size)
	// - Depth: DzDepthOnetile means generate tiles at all zoom levels
	// - Overlap: 0 means no pixel overlap between tiles (can be changed if needed)
	// - TileSize: Size of each tile (254 is default, accounts for overlap)
	// If quality is 100% we do PNG output
	suffix := ".jpg"
	if tileQuality == 100 {
		suffix = ".png"
	}

	l.Infof("  Generating tiles of size %v, quality: %v, format: %v...", tileSize, tileQuality, suffix)

	err = img.Dzsave(localImageDataPath, &vips.DzsaveOptions{
		Imagename: filepath.Base(localImageDataPath),
		Suffix:    suffix,
		Q:         int(tileQuality),
		Depth:     vips.DzDepthOnetile,
		Overlap:   0,
		TileSize:  int(tileSize),
	})

	if err != nil {
		return err
	}

	// Write to S3 (need to use a multithreaded approach as it'd take way too long otherwise)
	// Save the image we saved to local storage in chunks as one file to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	localImagePyramidPath := localImageDataPath + "_files"
	err = fileaccess.CopyToBucket(fs, localImagePyramidPath, bucket, s3Path, true, l)

	// If we're done or not, delete the locally generated files
	if remErr := os.RemoveAll(localImagePyramidPath); remErr != nil {
		// Warn about this
		l.Errorf("Failed to delete image pyramid tiles at %v after they were uploaded: %v", localImagePyramidPath, remErr)
	}

	// Also remove the .dzi file
	if remErr := os.RemoveAll(localImageDataPath + ".dzi"); remErr != nil {
		l.Errorf("Failed to delete image pyramid tiles at %v after they were uploaded: %v", localImageDataPath+".dzi", remErr)
	}

	return err
}

func saveImageDataFromFile(fs fileaccess.FileAccess, bucket string, savePath string, localImageDataPath string) error {
	// Save the image we saved to local storage in chunks as one file to S3
	s3Path := filepaths.GetImageFilePath(savePath)

	f, err := os.Open(localImageDataPath)
	if err != nil {
		return err
	}
	return fs.WriteObjectStream(bucket, s3Path, f)
}

func isPyramidImport(imageWidth uint32, imageHeight uint32, imageFileSize uint64) bool {
	return imageWidth > 4000 || imageHeight > 4000 || imageFileSize > 35*1024*1024
}
