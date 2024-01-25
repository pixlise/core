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
	"sync"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/beamLocation"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/proto"

	"golang.org/x/image/tiff"
)

func importImagesForDataset(datasetID string, srcBucket string, destDataBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	imagesColl := dest.Collection(dbCollections.ImagesName)

	// Load the dataset bin file
	s3Path := SrcGetDatasetFilePath(datasetID, filepaths.DatasetFileName)
	fileBytes, err := fs.ReadObject(srcBucket, s3Path)
	if err != nil {
		return fmt.Errorf("Failed to load dataset for %v, from: s3://%v/%v, error was: %v.", datasetID, srcBucket, s3Path, err)
	}

	// Now decode the data & return it
	exprPB := &protos.Experiment{}
	err = proto.Unmarshal(fileBytes, exprPB)
	if err != nil {
		return fmt.Errorf("Failed to decode scan data for scan: %v. Error: %v", datasetID, err)
	}

	var wg sync.WaitGroup

	// Read all images and save an image record, while also saving location info if there is any...
	alignedImageSizes := map[string][]uint32{}
	var alignedImgMutex sync.Mutex
	for alignedIdx, img := range exprPB.AlignedContextImages {
		wg.Add(1)
		go func(alignedIdx int, img *protos.Experiment_ContextImageCoordinateInfo) {
			defer wg.Done()

			taskId := addImportTask(fmt.Sprintf("importAlignedImage datasetID: %v, image: %v", datasetID, img.Image))

			// Image itself
			savedName, w, h, err := importAlignedImage(img, exprPB, datasetID, srcBucket, destDataBucket, fs, imagesColl)
			/*if err != nil {
				fatalError(err)
			}*/
			finishImportTask(taskId, err)

			if err == nil {
				taskId = addImportTask(fmt.Sprintf("ImportBeamLocationToDB datasetID: %v, image: %v, alignedIdx: %v", datasetID, savedName, alignedIdx))
				// Import coordinates
				err = beamLocation.ImportBeamLocationToDB(savedName, datasetID, alignedIdx, exprPB, dest, &logger.StdOutLogger{})
				/*if err != nil {
					fatalError(err)
				}*/
				finishImportTask(taskId, err)

				alignedImgMutex.Lock()
				alignedImageSizes[savedName] = []uint32{w, h, uint32(alignedIdx)}
				alignedImgMutex.Unlock()
			}
		}(alignedIdx, img)
	}

	// Wait for all aligned images (we need the sizes to import tifs next)
	wg.Wait()

	var wg2 sync.WaitGroup

	for _, img := range exprPB.MatchedAlignedContextImages {
		wg2.Add(1)
		go func(img *protos.Experiment_MatchedContextImageInfo) {
			defer wg2.Done()
			taskId := addImportTask(fmt.Sprintf("importMatchedImage datasetID: %v, img: %v", datasetID, img.Image))
			_, err := importMatchedImage(img, alignedImageSizes, datasetID, srcBucket, destDataBucket, fs, imagesColl)
			finishImportTask(taskId, err)
			/*if err != nil {
				fatalError(err)
			}*/
		}(img)
	}

	for _, img := range exprPB.UnalignedContextImages {
		wg2.Add(1)
		go func(img string) {
			defer wg2.Done()
			taskId := addImportTask(fmt.Sprintf("importUnalignedImage datasetID: %v, img: %v", datasetID, img))
			_, err := importUnalignedImage(img, exprPB, datasetID, srcBucket, destDataBucket, fs, imagesColl)
			finishImportTask(taskId, err)
			/*if err != nil {
				fatalError(err)
			}*/
		}(img)
	}

	// Wait for all
	wg2.Wait()

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
) (string, uint32, uint32, error) {
	// Expecting only PNGs here
	imgExt := strings.ToLower(path.Ext(img.Image))
	if imgExt != ".png" && imgExt != ".jpg" {
		return "", 0, 0, fmt.Errorf("Expected only PNG or JPG image for Aligned image, got: %v", img.Image)
	}

	imgSave, imgBytes, srcS3Path, err := getImportImage("aligned", img.Image, datasetID, dataBucket, fs, 0, 0)
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

	return imgSave.Name, imgSave.Width, imgSave.Height, saveImage(imgSave, imagesColl, imgBytes, fs, dataBucket, srcS3Path, destDataBucket, datasetID)
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

	imgSave, imgBytes, srcS3Path, err := getImportImage("matched", img.Image, datasetID, dataBucket, fs, imageW, imageH)
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

	return imgSave.Name, saveImage(imgSave, imagesColl, imgBytes, fs, dataBucket, srcS3Path, destDataBucket, datasetID)
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
	imgSave, imgBytes, srcS3Path, err := getImportImage("unaligned", imgName, datasetID, dataBucket, fs, 0, 0)
	if err != nil {
		return "", err
	}

	imgSave.AssociatedScanIds = []string{datasetID}

	return imgSave.Name, saveImage(imgSave, imagesColl, imgBytes, fs, dataBucket, srcS3Path, destDataBucket, datasetID)
}

// Downloads the image and determines the size if passed in imageW==imageH==0
func getImportImage(imgType string, imageName string, datasetID string, dataBucket string, fs fileaccess.FileAccess, imageW uint32, imageH uint32) (*protos.ScanImage, []byte, string, error) {
	fmt.Printf("Importing scan: %v %v image: %v...\n", datasetID, imgType, imageName)

	// Read the image file itself
	s3Path := SrcGetDatasetFilePath(datasetID, imageName)
	imgBytes, err := fs.ReadObject(dataBucket, s3Path)
	if err != nil {
		return nil, imgBytes, s3Path, err
	}

	// Open the image to determine the size
	// NOTE: it may be a tif file!
	if imageW == 0 && imageH == 0 {
		if isExt(imageName, "tif") {
			// First detected with: PCCR0095_0667226570_000MSA_N0010052000004530000075CD01.tif.
			theImage, err := tiff.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				// If we still can't open it, maybe it's one of our floating point RGBU TIF images. The tiff libary doesn't support these
				// but we know the size already, so use that here
				if err.Error() == "tiff: unsupported feature: sample format" {
					imageW = 752
					imageH = 580
				} else {
					return nil, imgBytes, s3Path, fmt.Errorf("Failed to decode TIF image: %v. Error: %v", imageName, err)
				}
			} else {
				imageW = uint32(theImage.Bounds().Dx())
				imageH = uint32(theImage.Bounds().Dy())
			}
		} else {
			theImage, _, err := image.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				return nil, imgBytes, s3Path, fmt.Errorf("Failed to read image: %v. Error: %v", imageName, err)
			}

			imageW = uint32(theImage.Bounds().Dx())
			imageH = uint32(theImage.Bounds().Dy())
		}
	}

	imageName = getImageSaveName(datasetID, imageName)

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

	return imgSave, imgBytes, s3Path, nil
}

func getImageSaveName(scanId string, imageName string) string {
	// If the image name can't be parsed as a gds filename, we prepend the dataset ID to make it more unique. This is not done
	// on GDS filenames because they would already contain the RTT making them unique, and we also want to keep those
	// searchable/equivalent to names in Mars Viewer
	if fields, err := gdsfilename.ParseFileName(imageName); err != nil || fields.Producer == "D" || fields.ProdType == "MSA" || fields.ProdType == "VIS" {
		imageName = scanId + "-" + imageName
	}
	return imageName
}

func saveImage(
	imgSave *protos.ScanImage,
	imagesColl *mongo.Collection,
	imgBytes []byte,
	fs fileaccess.FileAccess,
	srcDataBucket string,
	srcS3Path string,
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
	writePath := filepaths.GetImageFilePath(savePath)
	// TODO: copy within AWS for speed
	//return fs.WriteObject(destDataBucket, writePath, imgBytes)
	return fs.CopyObject(srcDataBucket, srcS3Path, destDataBucket, writePath)
}
