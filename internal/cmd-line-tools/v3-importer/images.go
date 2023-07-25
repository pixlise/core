package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/gdsfilename"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/proto"
)

func importImagesForDataset(datasetID string, dataBucket string, destDataBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	imagesColl := dest.Collection(dbCollections.ImagesName)

	// Load the dataset bin file
	s3Path := filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetFileName)
	fileBytes, err := fs.ReadObject(dataBucket, s3Path)
	if err != nil {
		return fmt.Errorf("Failed to load dataset for %v, from: s3://%v/%v, error was: %v.", datasetID, dataBucket, s3Path, err)
	}

	// Now decode the data & return it
	exprPB := &protos.Experiment{}
	err = proto.Unmarshal(fileBytes, exprPB)
	if err != nil {
		return fmt.Errorf("Failed to decode scan data for scan: %v. Error: %v", datasetID, err)
	}

	// Read all images and save an image record, while also saving location info if there is any...
	alignedImageSizes := map[string][]uint32{}
	for alignedIdx, img := range exprPB.AlignedContextImages {
		// Image itself
		if savedName, w, h, err := importAlignedImage(img, exprPB, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		} else {
			// Import coordinates
			if err := importImageLocations(savedName, datasetID, alignedIdx, exprPB, dest); err != nil {
				return err
			}

			alignedImageSizes[savedName] = []uint32{w, h, uint32(alignedIdx)}
		}
	}

	for _, img := range exprPB.MatchedAlignedContextImages {
		if _, err := importMatchedImage(img, alignedImageSizes, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		}
	}

	for _, img := range exprPB.UnalignedContextImages {
		if _, err := importUnalignedImage(img, exprPB, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		}
	}

	// Write the dataset file out to destination
	s3Path = path.Join(filepaths.DatasetScansRoot, datasetID, filepaths.DatasetFileName)
	return fs.WriteObject(destDataBucket, s3Path, fileBytes)
}

func importAlignedImage(
	img *protos.Experiment_ContextImageCoordinateInfo,
	exprPB *protos.Experiment,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) (string, uint32, uint32, error) {
	// Expecting only PNGs here
	imgExt := strings.ToLower(path.Ext(img.Image))
	if imgExt != ".png" && imgExt != ".jpg" {
		return "", 0, 0, fmt.Errorf("Expected only PNG or JPG image for Aligned image, got: %v", img.Image)
	}

	imgSave, imgBytes, err := getImportImage(img.Image, datasetID, dataBucket, fs, 0, 0)
	if err != nil {
		return "", 0, 0, err
	}

	// Work out all scans this image is associated with and that we have coordinates for
	var scanSource *protos.Experiment_ScanSource
	if exprPB.ScanSources != nil {
		scanSource = exprPB.ScanSources[img.ScanSource]
		if len(scanSource.Rtt) > 0 {
			imgSave.AssociatedScanIds = append(imgSave.AssociatedScanIds, scanSource.Rtt)
		}
	}
	if scanSource == nil {
		imgSave.AssociatedScanIds = []string{datasetID}
	}

	// Refine the scan source
	if nameBits, err := gdsfilename.ParseFileName(img.Image); err == nil {
		rtt, err := nameBits.RTT()
		if err == nil && rtt == datasetID {
			// Image is only from instrument if the name parses correctly and the RTT matches the dataset ID
			imgSave.Source = protos.ScanImageSource_SI_INSTRUMENT
			imgSave.OriginScanId = rtt

			// It's an image from GDS so lets point it back to Mars Viewer
			//imgSave.OriginImageURL =
			// The above should probably be dynamic, so don't store it in DB here...
		}
	}

	return imgSave.Name, imgSave.Width, imgSave.Height, saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

func importMatchedImage(
	img *protos.Experiment_MatchedContextImageInfo,
	alignedImageSizes map[string][]uint32,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) (string, error) {
	// Look up the matched image name and size info
	var alignedImageName string
	var alignedImageW uint32
	var alignedImageH uint32
	for name, info := range alignedImageSizes {
		if info[2] == uint32(img.AlignedIndex) {
			alignedImageName = name
			alignedImageW = info[0]
			alignedImageH = info[1]
			break
		}
	}

	// If reading a TIF we use the aligned image width+height because Go tif importer fails with floating point sample
	// types. Otherwise get image size from downloaded data
	var imageW, imageH uint32
	if isExt(img.Image, "tif") {
		imageW = alignedImageW
		imageH = alignedImageH
	}

	imgSave, imgBytes, err := getImportImage(img.Image, datasetID, dataBucket, fs, imageW, imageH)
	if err != nil {
		return "", err
	}

	imgSave.MatchInfo = &protos.ImageMatchTransform{
		BeamImageFileName: alignedImageName,
		XOffset:           img.XOffset,
		YOffset:           img.YOffset,
		XScale:            img.XScale,
		YScale:            img.YScale,
	}

	imgSave.AssociatedScanIds = []string{datasetID}

	// Refine the image purpose - Tif files are for RGBU analysis
	if isExt(img.Image, "tif") {
		imgSave.Purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
	}

	return imgSave.Name, saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

func isExt(fileName string, extNoDot string) bool {
	return strings.ToLower(path.Ext(fileName)) == "."+extNoDot
}

func importUnalignedImage(
	imgName string,
	exprPB *protos.Experiment,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) (string, error) {
	imgSave, imgBytes, err := getImportImage(imgName, datasetID, dataBucket, fs, 0, 0)
	if err != nil {
		return "", err
	}

	imgSave.AssociatedScanIds = []string{datasetID}

	return imgSave.Name, saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

// Downloads the image and determines the size if passed in imageW==imageH==0
func getImportImage(imageName string, datasetID string, dataBucket string, fs fileaccess.FileAccess, imageW uint32, imageH uint32) (*protos.ScanImage, []byte, error) {
	fmt.Printf("Importing scan: %v image: %v...\n", datasetID, imageName)

	// Read the image file itself
	s3Path := filepaths.GetDatasetFilePath(datasetID, imageName)
	imgBytes, err := fs.ReadObject(dataBucket, s3Path)
	if err != nil {
		return nil, imgBytes, err
	}

	// Open the image to determine the size
	if imageW == 0 && imageH == 0 {
		theImage, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			return nil, imgBytes, fmt.Errorf("Failed to read image: %v. Error: %v", imageName, err)
		}

		imageW = uint32(theImage.Bounds().Dx())
		imageH = uint32(theImage.Bounds().Dy())
	}

	// If the image name can't be parsed as a gds filename, we prepend the dataset ID to make it more unique. This is not done
	// on GDS filenames because they would already contain the RTT making them unique, and we also want to keep those
	// searchable/equivalent to names in Mars Viewer
	if _, err := gdsfilename.ParseFileName(imageName); err != nil {
		imageName = datasetID + "-" + imageName
	}

	imgSave := &protos.ScanImage{
		Name:              imageName,
		Source:            protos.ScanImageSource_SI_UPLOAD,
		Width:             imageW,
		Height:            imageH,
		FileSize:          uint32(len(imgBytes)),
		Purpose:           protos.ScanImagePurpose_SIP_VIEWING,
		AssociatedScanIds: []string{},
		//OriginScanId: ,
		//OriginImageURL: originURL,
		//Path: ,
		//MatchInfo: ,
	}

	return imgSave, imgBytes, nil
}

func saveImage(
	imgSave *protos.ScanImage,
	imagesColl *mongo.Collection,
	imgBytes []byte,
	fs fileaccess.FileAccess,
	destDataBucket string,
	datasetID string,
) error {
	// Work out where we'll save it
	savePath := path.Join(datasetID, imgSave.Name)
	imgSave.Path = savePath

	// Write the new image record to DB
	result, err := imagesColl.InsertOne(context.TODO(), imgSave)
	if err != nil {
		return err
	}
	if result.InsertedID != imgSave.Name {
		return fmt.Errorf("Image insert for %v inserted different id %v", imgSave.Name, result.InsertedID)
	}

	// Also write the image file to S3 destination
	writePath := path.Join(filepaths.DatasetImagesRoot, savePath)
	return fs.WriteObject(destDataBucket, writePath, imgBytes)
}

func importImageLocations(imgName string, scanId string, alignedImageIdx int, exprPB *protos.Experiment, dest *mongo.Database) error {
	imagesColl := dest.Collection(dbCollections.ImageBeamLocationsName)

	beams := &protos.ImageLocations{
		ImageName:       imgName,
		LocationPerScan: []*protos.ImageLocationsForScan{},
	}

	// Find the coordinates for this image
	ijs := []*protos.Coordinate2D{}

	for _, loc := range exprPB.Locations {
		var ij *protos.Coordinate2D

		if loc.Beam != nil {
			ij = &protos.Coordinate2D{}
			if alignedImageIdx == 0 {
				ij.I = loc.Beam.ImageI
				ij.J = loc.Beam.ImageJ
			} else {
				ij.I = loc.Beam.ContextLocations[alignedImageIdx-1].I
				ij.J = loc.Beam.ContextLocations[alignedImageIdx-1].J
			}
		}

		ijs = append(ijs, ij)
	}

	beams.LocationPerScan = append(beams.LocationPerScan, &protos.ImageLocationsForScan{
		ScanId:    scanId,
		Locations: ijs,
	})

	result, err := imagesColl.InsertOne(context.TODO(), beams)
	if err != nil {
		return err
	}
	if result.InsertedID != imgName {
		return fmt.Errorf("Image insert for %v inserted different id %v", imgName, result.InsertedID)
	}
	return nil
}
