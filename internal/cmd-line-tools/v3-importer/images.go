package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
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
	for alignedIdx, img := range exprPB.AlignedContextImages {
		// Image itself
		if err := importAlignedImage(img, exprPB, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		}

		// Import coordinates
		if err := importImageLocations(img.Image, datasetID, alignedIdx, exprPB, dest); err != nil {
			return err
		}
	}

	for _, img := range exprPB.MatchedAlignedContextImages {
		if err := importMatchedImage(img, exprPB, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		}
	}

	for _, img := range exprPB.UnalignedContextImages {
		if err := importUnalignedImage(img, exprPB, datasetID, dataBucket, destDataBucket, fs, imagesColl); err != nil {
			return err
		}
	}

	return nil
}

func importAlignedImage(
	img *protos.Experiment_ContextImageCoordinateInfo,
	exprPB *protos.Experiment,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) error {
	// Expecting only PNGs here
	imgExt := strings.ToLower(path.Ext(img.Image))
	if imgExt != ".png" && imgExt != ".jpg" {
		return fmt.Errorf("Expected only PNG or JPG image for Aligned image, got: %v", img.Image)
	}

	imgSave, imgBytes, err := getImportImage(img.Image, datasetID, dataBucket, fs)
	if err != nil {
		return err
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

	return saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

func importMatchedImage(
	img *protos.Experiment_MatchedContextImageInfo,
	exprPB *protos.Experiment,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) error {
	imgSave, imgBytes, err := getImportImage(img.Image, datasetID, dataBucket, fs)
	if err != nil {
		return err
	}

	// Save match info
	imgSave.MatchInfo = &protos.ImageMatchTransform{
		BeamImageFileName: img.Image,
		XOffset:           img.XOffset,
		YOffset:           img.YOffset,
		XScale:            img.XScale,
		YScale:            img.YScale,
	}

	// Refine the image purpose - Tif files are for RGBU analysis
	if strings.ToLower(path.Ext(img.Image)) == ".tif" {
		imgSave.Purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
	}

	return saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

func importUnalignedImage(
	imgName string,
	exprPB *protos.Experiment,
	datasetID string,
	dataBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	imagesColl *mongo.Collection,
) error {
	imgSave, imgBytes, err := getImportImage(imgName, datasetID, dataBucket, fs)
	if err != nil {
		return err
	}

	// Nothing to customise

	return saveImage(imgSave, imagesColl, imgBytes, fs, destDataBucket, datasetID)
}

func getImportImage(imageName string, datasetID string, dataBucket string, fs fileaccess.FileAccess) (*protos.ScanImage, []byte, error) {
	// Read the image file itself
	s3Path := filepaths.GetDatasetFilePath(datasetID, imageName)
	imgBytes, err := fs.ReadObject(dataBucket, s3Path)
	if err != nil {
		return nil, imgBytes, err
	}

	// Open the image to determine the size
	theImage, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, imgBytes, err
	}

	imgSave := &protos.ScanImage{
		Name:              imageName,
		Source:            protos.ScanImageSource_SI_UPLOAD,
		Width:             uint32(theImage.Bounds().Dx()),
		Height:            uint32(theImage.Bounds().Dy()),
		FileSize:          uint32(len(imgBytes)),
		Purpose:           protos.ScanImagePurpose_SIP_VIEWING,
		AssociatedScanIds: []string{},
		//OriginScanId: ,
		//OriginImageURL: originURL,
		//Url: imgGetURL,
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
	// Write the new image record to DB
	result, err := imagesColl.InsertOne(context.TODO(), imgSave)
	if err != nil {
		return err
	}
	if result.InsertedID != imgSave.Name {
		return fmt.Errorf("Image insert for %v inserted different id %v", imgSave.Name, result.InsertedID)
	}

	// Also write the image file to S3 destination
	writePath := path.Join("images", datasetID, imgSave.Name)
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
