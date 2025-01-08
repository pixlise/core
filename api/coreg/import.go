package coreg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/imageedit"
	"github.com/pixlise/core/v4/core/imgFormat"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func StartCoregImport(triggerUrl string, hctx wsHelpers.HandlerContext) (string, error) {
	if len(triggerUrl) <= 0 {
		return "", errorwithstatus.MakeBadRequestError(errors.New("MarsViewerExport trigger Url is empty"))
	}

	i := coregUpdater{hctx}

	// Start an image coreg import job (this is a Lambda function)
	// Once it completes, we have the data we need, so we can treat it as a "normal" image importing task
	jobStatus, err := job.AddJob("coreg", hctx.SessUser.User.Id, protos.JobStatus_JT_IMPORT_IMAGE, triggerUrl, fmt.Sprintf("Import: %v", path.Base(triggerUrl)), []string{}, uint32(hctx.Svcs.Config.ImportJobMaxTimeSec), hctx.Svcs.MongoDB, hctx.Svcs.IDGen, hctx.Svcs.TimeStamper, hctx.Svcs.Log, i.sendUpdate)
	jobId := ""
	if jobStatus != nil {
		jobId = jobStatus.JobId
	}

	if err != nil || len(jobId) < 0 {
		returnErr := fmt.Errorf("Failed to add job watcher for coreg import Job ID: %v. Error was: %v", jobId, err)
		hctx.Svcs.Log.Errorf("%v", returnErr)
		return "", returnErr
	}

	// completeMarsViewerImportJob("coreg-9un1y0fv2gszftw3", hctx)
	// completeMarsViewerImportJob("coreg-bjy6epclzutur900", hctx)
	// completeMarsViewerImportJob("coreg-dy6kmc5r9rg5t2l7", hctx)
	// completeMarsViewerImportJob("coreg-282g38pxwybbboqk", hctx)
	// return "", nil

	// We can now trigger the lambda
	// NOTE: here we build the same structure that triggered us, but we exclude the points data so we don't exceed
	// the SQS 256kb limit. The lambda doesn't care about the points anyway, only we do once the lambda has completed!
	coregReq := CoregJobRequest{jobId, hctx.Svcs.Config.EnvironmentName, triggerUrl}
	msg, err := json.Marshal(coregReq)
	if err != nil {
		returnErr := fmt.Errorf("Failed to create coreg job trigger message for job ID: %v", jobId)
		job.CompleteJob(jobId, false, returnErr.Error(), "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return "", returnErr
	}

	_, err = hctx.Svcs.SQS.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(msg)),
		QueueUrl:    aws.String(hctx.Svcs.Config.CoregSqsQueueUrl),
	})

	if err != nil {
		returnErr := fmt.Errorf("Failed to trigger coreg job. ID: %v. Error: %v", jobId, err)
		job.CompleteJob(jobId, false, returnErr.Error(), "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return "", returnErr
	}

	return jobId, nil
}

type coregUpdater struct {
	hctx wsHelpers.HandlerContext
}

func (i *coregUpdater) sendUpdate(status *protos.JobStatus) {
	// NOTE: The coreg image import job sets state GATHERING_RESULTS when it has downloaded everything
	// so here we trigger off that to do our part, after which we can mark the job as COMPLETE or ERROR
	if status.Status == protos.JobStatus_GATHERING_RESULTS {
		// NOTE: If this fails, it will set the job status to ERROR and we'll
		// get another call to update...
		completeMarsViewerImportJob(status.JobId, i.hctx)
		return
	}

	wsUpd := protos.WSMessage{
		Contents: &protos.WSMessage_ImportMarsViewerImageUpd{
			ImportMarsViewerImageUpd: &protos.ImportMarsViewerImageUpd{
				Status: status,
			},
		},
	}

	wsHelpers.SendForSession(i.hctx.Session, &wsUpd)
}

// Should be called after Coreg Import Lambda has completed successfully
func completeMarsViewerImportJob(jobId string, hctx wsHelpers.HandlerContext) {
	// Read the job completion entry from DB
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.CoregJobCollection)
	dbResult := coll.FindOne(ctx, bson.M{"_id": jobId}, options.FindOne())
	if dbResult.Err() != nil {
		failJob(fmt.Sprintf("Failed to find Coreg Job completion record for: %v. Error: %v", jobId, dbResult.Err()), jobId, hctx)
		return
	}

	coregResult := CoregJobResult{}
	err := dbResult.Decode(&coregResult)
	if err != nil {
		failJob(fmt.Sprintf("Failed to decode Coreg Job completion record for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	hctx.Svcs.Log.Infof("marsViewer import job %v importing from %v", jobId, coregResult.MarsViewerExportUrl)

	// At this point we should have everything ready to go - our own bucket should contain all images
	// and we have the mars viewer export msg containing any points we require so lets import the warped images we received!
	// Firstly, read the export from MV
	marsViewerExport := &protos.MarsViewerExport{}
	mvBucket, err := fileaccess.GetBucketFromS3Url(coregResult.MarsViewerExportUrl)
	if err != nil {
		failJob(fmt.Sprintf("Failed to read Coreg Job files for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	mvPath, err := fileaccess.GetPathFromS3Url(coregResult.MarsViewerExportUrl)
	if err != nil {
		failJob(fmt.Sprintf("Failed to read Coreg Job files for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	err = hctx.Svcs.FS.ReadJSON(mvBucket, mvPath, marsViewerExport, false)
	if err != nil {
		failJob(fmt.Sprintf("Failed to read MarsViewer export json for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	// How MV data is structured:
	// "baseImageUrl" represents the image that everything was warped TO.
	// Observations contain arrays of points (and CSV file references) for coordinates that were warped to the base image. It seems that if
	// a set of coordinates were already relative to that image, we don't get the "translatedPoints" list, which makes sense...
	// Therefore, if we find an observation with "translatedPoints", we can look at what image that applies to and store those beam locations
	// for the RTT identified by the observation's own "contextImageUrl"
	//
	// Warped images are also supplied, which take another image and warp it to match the base image. These are able to be rendered relative
	// the base image, much like our "matched" images, and so should probably be imported as such.

	// First, lets check that we have this base image stored. NOTE: It may not be the same file name as the one in PIXLISE! There are many
	// versions and file formats of images that get generated by the pipeline, someone may have picked another version as the base image, so
	// we have to search/match it unfortunately :(
	// For example:
	// PCW_0920_0748651385_000RCM_N04500003226342450005075J01.png <-- PIXLISE contains this
	// PCW_0920_0748651385_000FDR_N04500003226342450005075J01.IMG <-- Base image for warping in MV
	//
	// All matches except the product type AND the extension!

	// Get the meta associated with base image
	baseRTT, _, err := getRTTAndMeta(marsViewerExport.BaseImageUrl)
	if err != nil {
		failJob(fmt.Sprintf("Failed to parse baseImageUrl: %v for job %v. Error: %v", marsViewerExport.BaseImageUrl, jobId, err), jobId, hctx)
		return
	}

	// Now we can try to find the corresponding image
	ourBaseImage, ourBaseImageItem, err := findImage(marsViewerExport.BaseImageUrl, baseRTT, hctx)
	baseImageLocs := &protos.ImageLocations{}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// We don't have this image. This means we're importing a new image with the observations transformed to it
			if len(coregResult.BaseImageUrl.NewUri) <= 0 {
				// We don't have the "new" base image to import, so fail here
				failJob(fmt.Sprintf("Coreg import job %v failed, base image not available. Original: %v", jobId, coregResult.BaseImageUrl.OriginalUri), jobId, hctx)
			}
			ourBaseImage, ourBaseImageItem, baseImageLocs, err = importNewImage(jobId, coregResult.BaseImageUrl.NewUri, baseRTT, marsViewerExport, hctx)
		}
	} else {
		// Read the locations
		baseImageLocs, err = readExistingLocationsForImage(jobId, ourBaseImage, hctx)
	}

	if err == nil {
		// We are importing images/observations warped TO the base image in question
		err = importWarpedToBase(jobId, ourBaseImage, ourBaseImageItem, baseImageLocs, baseRTT, &coregResult, marsViewerExport, hctx)
	}

	if err != nil {
		failJob(fmt.Sprintf("Coreg import job %v failed. Error: %v", jobId, err), jobId, hctx)
		return
	}

	job.CompleteJob(jobId, true, "Coreg import complete", "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
}

func importNewImage(jobId string, imageUrl string, baseRTT string, marsViewerExport *protos.MarsViewerExport, hctx wsHelpers.HandlerContext) (string, *protos.ScanImage, *protos.ImageLocations, error) {
	// We're importing an image we haven't heard about before. Once this has been imported, we should be able to import the rest as normal
	hctx.Svcs.Log.Infof("importNewImage: %v for RTT %v, job: %v...", imageUrl, baseRTT, jobId)

	// Gather all "associated images" - any translated points that we'll be importing. We need to be associated with them
	// so we can be shown on their context images
	associatedScanIds := []string{baseRTT}
	for _, obs := range marsViewerExport.Observations {
		if len(obs.TranslatedPoints) > 0 {
			contextMeta, err := gdsfilename.ParseFileName(path.Base(obs.ContextImageUrl))
			if err != nil {
				hctx.Svcs.Log.Errorf("importNewImage: Failed to read RTT from translated points context image file name: %v. Error: %v", obs.ContextImageUrl, err)
			} else {
				rtt, err := contextMeta.RTT()
				if err != nil {
					hctx.Svcs.Log.Errorf("importNewImage: Failed to decode RTT from translated points context image file name: %v. Error: %v", obs.ContextImageUrl, err)
				} else {
					associatedScanIds = append(associatedScanIds, rtt)
				}
			}
		}
	}

	// We need to:
	// Add an item to DB for this image
	// Add file to S3
	imageFileName := path.Base(imageUrl)

	// Expecting file name like this:
	// SC3_0614_0721480213_394RAS_N0301172SRLC10600_0000LMJ03.IMG
	// Firstly, check that the image type is one that we support
	imageExt := strings.ToUpper(path.Ext(imageUrl))
	if imageExt != ".PNG" && imageExt != ".JPG" && imageExt != ".IMG" {
		return "", nil, nil, fmt.Errorf("Image file type not supported: %v", imageUrl)
	}

	// Read the bytes
	imgBucket, err := fileaccess.GetBucketFromS3Url(imageUrl)
	if err != nil {
		return "", nil, nil, err
	}
	imgPath, err := fileaccess.GetPathFromS3Url(imageUrl)
	if err != nil {
		return "", nil, nil, err
	}

	imgData, err := hctx.Svcs.FS.ReadObject(imgBucket, imgPath)
	if err != nil {
		return "", nil, nil, err
	}

	imgWidth := uint32(0)
	imgHeight := uint32(0)

	var img image.Image

	if imageExt == ".IMG" {
		w, h, d, err := imgFormat.ReadIMGFile(imgData)
		imgWidth = uint32(w)
		imgHeight = uint32(h)
		if err != nil {
			return "", nil, nil, fmt.Errorf("Failed to read IMG file: %v. Error: %v", imageUrl, err)
		}

		img = imageedit.MakeImageFromRGBA(w, h, d)

		// Change the file name! We're saving as PNG
		imageFileName = imageFileName[0:len(imageFileName)-3] + "png"

		// Meanwhile, create the PNG image so the data we write is the actual PNG
		var buf bytes.Buffer

		err = png.Encode(&buf, img)
		if err != nil {
			return "", nil, nil, fmt.Errorf("Failed to write PNG file for IMG file: %v. Error: %v", imageUrl, err)
		}

		imgData = buf.Bytes()
	} else {
		imgWidth, imgHeight, err = utils.ReadImageDimensions(path.Base(imgPath), imgData)
		if err != nil {
			return "", nil, nil, fmt.Errorf("Failed to read image file: %v. Error: %v", imageUrl, err)
		}
	}

	savePath := path.Join(baseRTT, imageFileName)
	scanImage := utils.MakeScanImage(
		savePath,
		uint32(len(imgData)),
		protos.ScanImageSource_SI_UPLOAD,
		protos.ScanImagePurpose_SIP_VIEWING,
		associatedScanIds,
		"", //baseRTT,
		"",
		nil,
		imgWidth,
		imgHeight,
	)

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	opt := options.Update().SetUpsert(true)

	result, err := coll.UpdateByID(ctx, scanImage.ImagePath, bson.D{{Key: "$set", Value: scanImage}}, opt)
	if err != nil {
		return "", nil, nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("importNewImage failed to upsert DB image: %v. Result: %+v", scanImage.ImagePath, result)
	}

	// Save the image to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.DatasetsBucket, s3Path, imgData)
	if err != nil {
		// Failed to upload image data, so no point in having a DB entry now either...
		coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
		filter := bson.D{{Key: "_id", Value: scanImage.ImagePath}}
		delOpt := options.Delete()
		_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
		return "", nil, nil, err
	}

	// Also insert a blank entry for beam locations for this image, as we're expecting to import scans for it
	coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)
	beamImageName := wsHelpers.GetImageNameSansVersion(scanImage.ImagePath)
	beamLocs := &protos.ImageLocations{
		ImageName:       beamImageName,
		LocationPerScan: []*protos.ImageLocationsForScan{},
	}

	beamResult, err := coll.UpdateByID(ctx, beamImageName, bson.D{{Key: "$set", Value: beamLocs}}, opt)
	if err != nil {
		return "", nil, nil, err
	}

	if beamResult.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("importNewImage failed to upsert initial DB image beam locations: %v. Result: %+v", scanImage.ImagePath, beamResult)
	}

	return scanImage.ImagePath, scanImage, beamLocs, err
}

func readExistingLocationsForImage(jobId string, image string, hctx wsHelpers.HandlerContext) (*protos.ImageLocations, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

	// We're adding to the beam locations for the base image! First, read the base image beam locations structure as there should
	// already be one!
	filter := bson.M{"_id": wsHelpers.GetImageNameSansVersion(image)}
	baseImageBeamsResult := coll.FindOne(ctx, filter)

	if baseImageBeamsResult.Err() != nil {
		return nil, fmt.Errorf("Coreg import job %v failed to read beams for base image %v. Error: %v", jobId, image, baseImageBeamsResult.Err())
	}

	baseImageBeams := &protos.ImageLocations{}
	err := baseImageBeamsResult.Decode(baseImageBeams)

	if err != nil {
		return nil, fmt.Errorf("Coreg import job %v failed to decode beams for base image %v. Error: %v", jobId, image, err)
	}

	return baseImageBeams, nil
}

// WARNING: This may have broken since image beam locations are now stored without version info. Because JPL has put the brakes on their side of
//
//	the coreg feature, there's no real way to test this out now
func importWarpedToBase(jobId string, baseImage string, ourBaseImageItem *protos.ScanImage, baseImageBeams *protos.ImageLocations, baseRtt string, coregResult *CoregJobResult, marsViewerExport *protos.MarsViewerExport, hctx wsHelpers.HandlerContext) error {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

	locationsPerScan := map[string][]*protos.Coordinate2D{}
	for _, locForScan := range baseImageBeams.LocationPerScan {
		locationsPerScan[locForScan.ScanId] = locForScan.Locations
	}

	beamsChanged := false
	associatedScanIds := ourBaseImageItem.AssociatedScanIds

	for _, obs := range marsViewerExport.Observations {
		if len(obs.TranslatedPoints) > 0 {
			hctx.Svcs.Log.Infof("marsViewer import job %v importing observation for %v with %v points", jobId, obs.ContextImageUrl, len(obs.TranslatedPoints))

			// We have new beam locations, find out which RTT this is for
			rtt, _, err := getRTTAndMeta(obs.ContextImageUrl)
			if err != nil {
				return fmt.Errorf("Failed to parse contextImageUrl: %v for job %v. Error: %v", obs.ContextImageUrl, jobId, err)
			}

			// Ensure rtt differs from our base image one
			if rtt == baseRtt {
				return fmt.Errorf("Coreg import job %v expected observation RTT %v to differ from base RTT %v", jobId, rtt, baseRtt)
			}

			// Ensure we don't have any points already stored for this RTT
			for _, locStored := range baseImageBeams.LocationPerScan {
				if locStored.ScanId == rtt {
					// Print out the fact that we'll be replacing it...
					hctx.Svcs.Log.Infof("Coreg is replacing beam locations for scan %v, image %v in job %v", rtt, baseImage, jobId)
					//return fmt.Errorf("Coreg import job %v detected beam locations already stored for image %v and scan %v", jobId, baseImage, rtt)
				}
			}

			// Save the points! We get PMCs, but need to store an array with all PMCs, so we need to fill the gaps with nil if needed
			// so we maintain "location index" ability. To do this we'll need to read the dataset file!
			exprPB, err := wsHelpers.ReadDatasetFile(rtt, hctx.Svcs)
			if err != nil {
				// NOTE: if we FAIL TO READ IT, this isn't the end of the world. It may be for a dataset we don't yet support
				// so just skip it here and continue, while logging an error.
				hctx.Svcs.Log.Errorf("Coreg import job %v skipping dataset %v due to failure to load error: %v", jobId, rtt, err)
				continue
			}

			translatedPointsLookup := map[int]*protos.Coordinate2D{}
			for _, txPoint := range obs.TranslatedPoints {
				if _, ok := translatedPointsLookup[int(txPoint.SpectrumNumber)]; ok {
					return fmt.Errorf("Coreg import job %v encounted duplicate SpectrumNumber in translated points for contextImageUrl: %v", jobId, obs.ContextImageUrl)
				}

				translatedPointsLookup[int(txPoint.SpectrumNumber)] = &protos.Coordinate2D{I: txPoint.Sample, J: txPoint.Line}
			}

			coords := []*protos.Coordinate2D{}
			for _, loc := range exprPB.Locations {
				// If our dataset contains a beam, we look up the translated beam location, otherwise store nil
				if loc.Beam != nil {
					pmc, err := strconv.Atoi(loc.Id)
					if err != nil {
						return fmt.Errorf("Coreg import job %v failed to read PMC %v from scan: %v", jobId, loc.Id, rtt)
					}
					// Find it
					if coord, ok := translatedPointsLookup[pmc]; ok {
						coords = append(coords, coord)
						continue
					}
				}

				// Nothing to put here, so put a nil so array matches
				coords = append(coords, nil)
			}

			// Store for this RTT
			locationsPerScan[rtt] = coords
			beamsChanged = true

			// Also store an associated scan id
			if !utils.ItemInSlice(rtt, associatedScanIds) {
				associatedScanIds = append(associatedScanIds, rtt)
			}
		}
	}

	// If we have added beams, save
	if beamsChanged {
		baseImageBeams.LocationPerScan = []*protos.ImageLocationsForScan{}
		for rtt, coords := range locationsPerScan {
			baseImageBeams.LocationPerScan = append(baseImageBeams.LocationPerScan, &protos.ImageLocationsForScan{ScanId: rtt, Locations: coords})
		}

		// TODO: Transaction for these 2?
		filter := bson.M{"_id": wsHelpers.GetImageNameSansVersion(baseImage)} // See WARNING at top of function
		result, err := coll.ReplaceOne(ctx, filter, &baseImageBeams, options.Replace())
		if err != nil {
			return fmt.Errorf("Coreg import job %v failed to save new beam locations: %v", jobId, err)
		}

		if result.MatchedCount != 1 {
			hctx.Svcs.Log.Errorf("Coreg import job %v didn't insert new beam locations: %+v", jobId, result)
		}

		// We also need to modify the image to add to its list of associated images
		coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
		update := bson.D{bson.E{Key: "associatedscanids", Value: associatedScanIds}}

		updImgResult, err := coll.UpdateByID(ctx, baseImage, bson.D{{Key: "$set", Value: update}})
		if err != nil {
			return fmt.Errorf("Coreg import job %v failed to update image associated scans list: %v", jobId, baseImage)
		}

		if updImgResult.MatchedCount != 1 {
			hctx.Svcs.Log.Errorf("Coreg import job %v didn't update image associated scans list: %+v", jobId, updImgResult)
		}
	}

	// Loop through each warped image, read it, along with beam locations (in observation for it)
	for _, item := range coregResult.WarpedImageUrls {
		if item.Completed {
			if err := importWarpedImage(item.NewUri, baseRtt, baseImage, hctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// NOTE: we expect URLs like this:
// s3://m20-sstage-ids-crisp-imgcoregi/crisp_data/ICM-PCW_0920_0748651385_000RAS_N045000032263424500050-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000-0-C02-J01.VIC/67e92f8ba7cd38d07d969f910db5d1d3/crisp_data/ods/surface/sol/00921/ids/rdr/shrlc/warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png

func importWarpedImage(warpedImageUrl string, rttWarpedTo string, baseImage string, hctx wsHelpers.HandlerContext) error {
	hctx.Svcs.Log.Infof("importWarpedImage: %v for RTT %v, image: %v...", warpedImageUrl, rttWarpedTo, baseImage)

	// We need to:
	// Add an item to DB for this image
	// Add file to S3
	warpedFileName := path.Base(warpedImageUrl)

	// Expecting file name like this:
	// warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png

	// Read the bytes
	warpedSrcBucket, err := fileaccess.GetBucketFromS3Url(warpedImageUrl)
	if err != nil {
		return err
	}
	warpedSrcPath, err := fileaccess.GetPathFromS3Url(warpedImageUrl)
	if err != nil {
		return err
	}

	imgData, err := hctx.Svcs.FS.ReadObject(warpedSrcBucket, warpedSrcPath)
	if err != nil {
		return err
	}

	imgWidth, imgHeight, err := utils.ReadImageDimensions(path.Base(warpedSrcPath), imgData)
	if err != nil {
		return err
	}

	matchInfo, nicerSaveName, err := readWarpedImageTransform(warpedFileName)
	if err != nil {
		return err
	}

	matchInfo.BeamImageFileName = baseImage

	savePath := path.Join(rttWarpedTo, nicerSaveName)
	scanImage := utils.MakeScanImage(
		savePath,
		uint32(len(imgData)),
		protos.ScanImageSource_SI_UPLOAD,
		protos.ScanImagePurpose_SIP_VIEWING,
		[]string{rttWarpedTo},
		rttWarpedTo,
		"",
		matchInfo,
		imgWidth,
		imgHeight,
	)

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	opt := options.Update().SetUpsert(true)
	result, err := coll.UpdateByID(ctx, scanImage.ImagePath, bson.D{{Key: "$set", Value: scanImage}}, opt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf("%v already exists", scanImage.ImagePath))
		}
		return err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("importWarpedImage failed to upsert DB image: %v. Result: %+v", scanImage.ImagePath, result)
	}

	// Save the image to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.DatasetsBucket, s3Path, imgData)
	if err != nil {
		// Failed to upload image data, so no point in having a DB entry now either...
		coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
		filter := bson.D{{Key: "_id", Value: scanImage.ImagePath}}
		delOpt := options.Delete()
		_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
		return err
	}

	return nil
}

func failJob(errMsg string, jobId string, hctx wsHelpers.HandlerContext) {
	job.CompleteJob(jobId, false, errMsg, "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
}

func getRTTAndMeta(imageUrl string) (string, gdsfilename.FileNameMeta, error) {
	srcMeta, err := gdsfilename.ParseFileName(path.Base(imageUrl))
	if err != nil {
		return "", srcMeta, err
	}

	rtt, err := srcMeta.RTT()
	return rtt, srcMeta, err
}

func findImage(imageName string, imageRTT string, hctx wsHelpers.HandlerContext) (string, *protos.ScanImage, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	foundItems, err := coll.Find(ctx, bson.M{"originscanid": imageRTT}, options.Find())
	if err != nil {
		return "", nil, err
	}

	items := []*protos.ScanImage{}
	err = foundItems.All(context.TODO(), &items)
	if err != nil {
		return "", nil, err
	}

	// Find it within what we've got
	comparableBaseName := gdsfilename.MakeComparableName(path.Base(imageName))

	for _, item := range items {
		comparableName := gdsfilename.MakeComparableName(path.Base(item.ImagePath))

		if comparableName == comparableBaseName {
			return item.ImagePath, item, nil
		}
	}

	return "", nil, mongo.ErrNoDocuments // fmt.Errorf("Failed to find image: %v for scan %v", imageName, imageRTT)
}

// NOTE: make sure to only pass the file name, not a path like scanid/file.png
func readWarpedImageTransform(fileName string) (*protos.ImageMatchTransform, string, error) {
	ext := path.Ext(fileName)
	if len(ext) > 0 {
		fileName = fileName[0 : len(fileName)-len(ext)]
	}

	parts := strings.Split(fileName, "-")

	// Expecting:
	// "warped"
	// "zoom_<zoom info>"
	// "win_<window info>"
	// Don't know what SN100D0 is?
	// And the original image file name
	// Don't know what -A is though?
	if len(parts) < 4 {
		// We expect warped, zoom_, win_ and the file name at a bare minimum
		return nil, "", fmt.Errorf("Warped image name does not have expected components")
	}

	// Check each bit
	if parts[0] != "warped" {
		return nil, "", fmt.Errorf("Expected warped image name to start with warped-")
	}

	zoomPrefix := "zoom_"
	if !strings.HasPrefix(parts[1], zoomPrefix) {
		return nil, "", fmt.Errorf("Expected warped image name second part to contain zoom")
	}

	winPrefix := "win_"
	if !strings.HasPrefix(parts[2], winPrefix) {
		return nil, "", fmt.Errorf("Expected warped image name second part to contain window")
	}

	// Ensure we have the gds-formatted file name, and check if we have the optional -A or whatever at the end
	endPiece := ""
	if len(parts[len(parts)-1]) == 1 { // Just looking for A, eg ends in -A.png...
		endPiece = "-" + parts[len(parts)-1]
	}

	gdsFileName := parts[len(parts)-1]
	if len(endPiece) > 0 {
		gdsFileName = parts[len(parts)-2]
	}

	// Verify that we got it
	_, err := gdsfilename.ParseFileName(gdsFileName + ext)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to find GDS file name section in image name: %v. Error: %v", gdsFileName, err)
	}

	// Read the zoom and window
	zoomStr := parts[1][len(zoomPrefix):] // Snipping number out of: zoom_4.478153138946561
	zoom, err := strconv.ParseFloat(zoomStr, 64)
	if err != nil {
		return nil, "", fmt.Errorf("Expected warped image zoom to contain a float, found: %v", zoomStr)
	}

	winPartsStr := strings.Split(parts[2][len(winPrefix):], "_") // Snipping numbers out of: win_519_40_1232_1183
	winParts := []int{}
	for c, n := range winPartsStr {
		winNum, err := strconv.Atoi(n)
		if err != nil {
			return nil, "", fmt.Errorf("Expected warped image window value %v to contain a number, found: %v", c+1, n)
		}

		winParts = append(winParts, winNum)
	}

	// win_... - this is stored as lines & samples, then width/height of the image
	// so line = y, samples = x in our world
	// Example: warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png
	// Has zoom factor 4.478153138946561 and x, y: (40, 519)

	// At this point we should have enough to reconstruct the transform as we interpret it
	return &protos.ImageMatchTransform{
			XOffset: float32(winParts[1]),
			YOffset: float32(winParts[0]),
			XScale:  float32(zoom),
			YScale:  float32(zoom),
		},
		// Provide a nicer file name to save as
		// If this image is coregistered in other ways we don't want it to clash, but we also don't want to generate random chars
		// for its name so it doesn't just proliferate (and a re-import can overwrite it). So lets try preserve what's likely to be
		// unique in the file name - the x,y!
		fmt.Sprintf("coreg-%v_%v-%v%v%v", winParts[1], winParts[0], gdsFileName, endPiece, ext),
		nil
}
