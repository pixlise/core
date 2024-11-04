package wsHandler

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dataimport"
	dataimportModel "github.com/pixlise/core/v4/api/dataimport/models"
	"github.com/pixlise/core/v4/api/dataimport/sdfToRSI"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/indexcompression"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

func HandleScanListReq(req *protos.ScanListReq, hctx wsHelpers.HandlerContext) (*protos.ScanListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_SCAN, hctx.Svcs, hctx.SessUser)
	if err != nil {
		return nil, err
	}

	// Check if the user specified a scanId in the request search filters, then we only need to use that one
	filterItems := []bson.M{}

	for field, value := range req.SearchFilters {
		if field == "scanId" {
			filterItems = []bson.M{{"_id": value}}
			break
		}
	}

	if len(filterItems) <= 0 {
		// Search through all ids accessible to our caller user
		ids := utils.GetMapKeys(idToOwner)
		filterItems = []bson.M{{"_id": bson.M{"$in": ids}}}
	}

	// It's either a meta field... one of the following known fields:
	metaFields := []string{"Target", "SiteId", "Site", "RTT", "SCLK", "Sol", "DriveId", "TargetId"}

	// Or it's a field on the struct:
	// - title
	// - description
	// - instrument
	// - instrumentConfig
	// - timeStampUnixSec

	for field, value := range req.SearchFilters {
		if field == "scanId" {
			// handled above
			continue
		}

		if utils.ItemInSlice(field, metaFields) {
			filterItems = append(filterItems, bson.M{"meta." + field: value})
		} else {
			// It must just be a struct field...
			filterItems = append(filterItems, bson.M{field: value})
		}
	}
	/*
		for field, minmax := range req.SearchMinMaxFilters {
			filterItems = append(filterItems, )
		}
	*/
	// Form the filter
	var filter bson.M
	if len(filterItems) == 1 {
		// Filter is simply the "_id" search
		filter = filterItems[0]
	} else {
		// It's an and clause of all our filter options
		ifcItems := []interface{}{}
		for _, item := range filterItems {
			ifcItems = append(ifcItems, item)
		}
		filter = bson.M{"$and": ifcItems}
	}

	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(context.TODO(), &scans)
	if err != nil {
		return nil, err
	}

	for _, scan := range scans {
		owner := idToOwner[scan.Id]
		scan.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	}

	return &protos.ScanListResp{
		Scans: scans,
	}, nil
}

func HandleScanGetReq(req *protos.ScanGetReq, hctx wsHelpers.HandlerContext) (*protos.ScanGetResp, error) {
	// Check that we have an id, current user, and display settings
	if len(req.Id) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Scan ID must be specified"))
	}

	if hctx.SessUser.User.Id == "" {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("User must be logged in"))
	}

	// Check user has access
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ScanItem](true, req.Id, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	return &protos.ScanGetResp{
		Scan: dbItem,
	}, nil
}

func HandleScanMetaLabelsAndTypesReq(req *protos.ScanMetaLabelsAndTypesReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaLabelsAndTypesResp, error) {
	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	// Form the list of types, we have the enums defined in a new spot separate to the experiment files
	types := []protos.ScanMetaDataType{}
	for _, t := range exprPB.MetaTypes {
		tScan := protos.ScanMetaDataType_MT_STRING
		if t == protos.Experiment_MT_INT {
			tScan = protos.ScanMetaDataType_MT_INT
		} else if t == protos.Experiment_MT_FLOAT {
			tScan = protos.ScanMetaDataType_MT_FLOAT
		}
		types = append(types, tScan)
	}

	return &protos.ScanMetaLabelsAndTypesResp{
		MetaLabels: exprPB.MetaLabels,
		MetaTypes:  types,
	}, nil
}

// Utility to call for any Req message that involves serving data out of a dataset.bin file
// scanId is mandatory, but startIdx and locCount may not exist in all requests, can be set to 0 if unused/not relevant
func beginDatasetFileReqForRange(scanId string, entryRange *protos.ScanEntryRange, hctx wsHelpers.HandlerContext) (*protos.Experiment, []uint32, error) {
	exprPB, err := beginDatasetFileReq(scanId, hctx)
	if err != nil {
		return nil, []uint32{}, err
	}

	indexes := []uint32{}
	if entryRange == nil {
		// Use all indexes available in the file
		for c := range exprPB.Locations {
			indexes = append(indexes, uint32(c))
		}
	} else {
		// Decode the range
		indexes, err = indexcompression.DecodeIndexList(entryRange.Indexes, len(exprPB.Locations))
		if err != nil {
			return nil, []uint32{}, err
		}
	}

	return exprPB, indexes, nil
}

func beginDatasetFileReq(scanId string, hctx wsHelpers.HandlerContext) (*protos.Experiment, error) {
	if err := wsHelpers.CheckStringField(&scanId, "ScanId", 1, 50); err != nil {
		return nil, err
	}

	_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	// We've come this far, we have access to the scan, so read it
	exprPB, err := wsHelpers.ReadDatasetFile(scanId, hctx.Svcs)
	if err != nil {
		return nil, err
	}

	return exprPB, nil
}

func HandleScanDeleteReq(req *protos.ScanDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ScanDeleteResp, error) {
	// Check user has access
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](true, req.ScanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return nil, err
	}

	// Verify they specified the right name
	if req.ScanNameForVerification != dbItem.Title {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Specified title did not match scan title of: \"%v\"", dbItem.Title))
	}

	// Check that it's not an FM dataset
	if dbItem.Instrument == protos.ScanInstrument_PIXL_FM {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Cannot delete FM datasets using this feature"))
	}

	// TODO: Should we stop deletion if images or quants reference it???

	// Delete the dataset from DB and the file from S3
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName)
	delResult, err := coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: req.ScanId}}, options.Delete())
	if err != nil {
		return nil, err
	}

	if delResult.DeletedCount != 1 {
		hctx.Svcs.Log.Errorf("ScanDelete %v - Unexpected DeletedCount %v, expected 1", req.ScanId, delResult.DeletedCount)
	}

	// Delete scan data from S3
	err = hctx.Svcs.FS.DeleteObject(hctx.Svcs.Config.DatasetsBucket, filepaths.GetScanFilePath(req.ScanId, filepaths.DatasetFileName))
	if err != nil {
		return nil, fmt.Errorf("ScanDelete %v - partially succeeded, as some files failed to delete: %v", req.ScanId, err)
	}

	err = hctx.Svcs.FS.DeleteObject(hctx.Svcs.Config.DatasetsBucket, filepaths.GetScanFilePath(req.ScanId, filepaths.DiffractionDBFileName))
	if err != nil {
		return nil, fmt.Errorf("ScanDelete %v - partially succeeded, as some files failed to delete: %v", req.ScanId, err)
	}

	// Notify of our scan change
	hctx.Svcs.Notifier.SysNotifyScanChanged(req.ScanId)

	return &protos.ScanDeleteResp{}, nil
}

func HandleScanMetaWriteReq(req *protos.ScanMetaWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaWriteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Title, "Title", 1, 100); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Description, "Description", 0, 30000); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckFieldLength(req.Tags, "Tags", 0, 10); err != nil {
		return nil, err
	}

	_, err := wsHelpers.CheckObjectAccess(true, req.ScanId, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	// Overwrites some metadata fields to allow them to be more descriptive to users. Requires permission EDIT_SCAN
	// so only admins can do this
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName)

	update := bson.D{
		bson.E{Key: "title", Value: req.Title},
		bson.E{Key: "description", Value: req.Description},
		bson.E{Key: "tags", Value: req.Tags},
	}

	result, err := coll.UpdateByID(ctx, req.ScanId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.ScanId)
	}

	// Notify of our scan change
	hctx.Svcs.Notifier.SysNotifyScanChanged(req.ScanId)

	return &protos.ScanMetaWriteResp{}, nil
}

func HandleScanTriggerReImportReq(req *protos.ScanTriggerReImportReq, hctx wsHelpers.HandlerContext) (*protos.ScanTriggerReImportResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, 50); err != nil {
		return nil, err
	}

	i := importUpdater{
		hctx.Session,
		hctx.Melody,
		hctx.Svcs.Notifier,
		req.ScanId,
		hctx.Svcs.MongoDB,
	}

	jobStatus, err := job.AddJob("reimport", hctx.SessUser.User.Id, protos.JobStatus_JT_REIMPORT_SCAN, req.ScanId, fmt.Sprintf("Reimport: %v", req.ScanId), []string{}, uint32(hctx.Svcs.Config.ImportJobMaxTimeSec), hctx.Svcs.MongoDB, hctx.Svcs.IDGen, hctx.Svcs.TimeStamper, hctx.Svcs.Log, i.sendReimportUpdate)
	jobId := ""
	if jobStatus != nil {
		jobId = jobStatus.JobId
	}

	if err != nil || len(jobId) < 0 {
		returnErr := fmt.Errorf("Failed to add job watcher for scan import trigger Job ID: %v. Error was: %v", jobId, err)
		hctx.Svcs.Log.Errorf("%v", returnErr)
		return nil, returnErr
	}

	result, err := dataimport.TriggerDatasetReprocessViaSNS(hctx.Svcs.SNS, jobId, req.ScanId, hctx.Svcs.Config.DataSourceSNSTopic)

	hctx.Svcs.Log.Infof("Triggered dataset reprocess via SNS topic. Result: %v. Job ID: %v", result, jobId)
	return &protos.ScanTriggerReImportResp{JobId: jobId}, err
}

// NOTE: before this is sent, we expect the PUT /scan endpoint to have been called and a zip uploaded already
//
//	The same parameters passed there (scan id & file name) must be used with this request otherwise we
//	won't look up the zip correctly and fail.
func HandleScanUploadReq(req *protos.ScanUploadReq, hctx wsHelpers.HandlerContext) (*protos.ScanUploadResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, 50); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Format, "Format", 1, 50); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.ZipFileName, "ZipFileName", 1, 100); err != nil {
		return nil, err
	}

	destBucket := hctx.Svcs.Config.ManualUploadBucket
	fs := hctx.Svcs.FS
	logger := hctx.Svcs.Log
	logger.Infof("Dataset create started for format: %v, id: %v", req.Format, req.Id)

	// Validate the dataset ID - can't contain funny characters because it ends up as an S3 path
	// NOTE: we also turn space to _ here! Having spaces in the path broke quants because the
	// quant summary file was written with a + instead of a space?!
	datasetID := fileaccess.MakeValidObjectName(req.Id, false)

	formats := []string{"jpl-breadboard", "sbu-breadboard", "pixl-em"}
	if !utils.ItemInSlice(req.Format, formats) {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Unexpected format: \"%v\"", req.Format))
	}

	s3PathStart := path.Join(filepaths.DatasetUploadRoot, datasetID)

	// Append a few random chars to make it more unique from this point on
	//datasetID += "_" + utils.RandStringBytesMaskImpr(6)

	// We don't need to check anything beyond does this file exist?
	/*
		// Check if this exists already...
		existingPaths, err := fs.ListObjects(destBucket, s3PathStart)
		if err != nil {
			err = fmt.Errorf("Failed to list existing files for dataset ID: %v. Error: %v", datasetID, err)
			logger.Errorf("%v", err)
			return nil, err
		}

		// If there are any existing paths, we stop here
		if len(existingPaths) > 0 {
			err = fmt.Errorf("Dataset ID already exists: %v", datasetID)
			logger.Errorf("%v", err)
			return nil, errorwithstatus.MakeBadRequestError(err)
		}
	*/

	// Read the zip that was expected to be uploaded
	zipPath := path.Join(s3PathStart, req.ZipFileName)
	zippedData, err := hctx.Svcs.FS.ReadObject(destBucket, zipPath)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to import, zip file not found: %v", zipPath))
	}

	// Validate zip contents matches the format we were given
	zipReader, err := zip.NewReader(bytes.NewReader(zippedData), int64(len(zippedData)))
	if err != nil {
		return nil, err
	}

	// Validate contents - detector dependent
	if req.Format == "pixl-em" {
		err = processEM(datasetID, zipReader, zippedData, destBucket, s3PathStart, fs, logger)
	} else {
		err = processBeadboard(req.Format, hctx.SessUser.User.Id, datasetID, zipReader, zippedData, destBucket, s3PathStart, fs, logger)
	}

	if err != nil {
		return nil, err
	}

	// Save detector info
	savePath := path.Join(s3PathStart, "detector.json")
	detectorFile := dataimportModel.DetectorChoice{
		Detector: req.Format,
	}
	err = fs.WriteJSON(destBucket, savePath, detectorFile)
	if err != nil {
		return nil, err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

	// Now save creator info
	savePath = path.Join(s3PathStart, "creator.json")
	err = fs.WriteJSON(destBucket, savePath, hctx.SessUser.User)
	if err != nil {
		return nil, err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

	i := importUpdater{
		hctx.Session,
		hctx.Melody,
		hctx.Svcs.Notifier,
		datasetID,
		hctx.Svcs.MongoDB,
	}

	// Add a job watcher for this
	jobStatus, err := job.AddJob("import", hctx.SessUser.User.Id, protos.JobStatus_JT_IMPORT_SCAN, datasetID, fmt.Sprintf("Import: %v", datasetID), []string{}, uint32(hctx.Svcs.Config.ImportJobMaxTimeSec), hctx.Svcs.MongoDB, hctx.Svcs.IDGen, hctx.Svcs.TimeStamper, hctx.Svcs.Log, i.sendImportUpdate)
	jobId := ""
	if jobStatus != nil {
		jobId = jobStatus.JobId
	}

	if err != nil || len(jobId) < 0 {
		returnErr := fmt.Errorf("Failed to add job watcher for scan upload Job ID: %v. Error was: %v", jobId, err)
		hctx.Svcs.Log.Errorf("%v", returnErr)
		return nil, returnErr
	}

	// Now we trigger a dataset conversion
	result, err := dataimport.TriggerDatasetReprocessViaSNS(hctx.Svcs.SNS, jobId, datasetID, hctx.Svcs.Config.DataSourceSNSTopic)
	if err != nil {
		return nil, err
	}

	logger.Infof("Triggered dataset reprocess via SNS topic. Result: %v. Job ID: %v", result, jobId)

	return &protos.ScanUploadResp{JobId: jobId}, nil
}

func readFromZip(fileInZip *zip.File, outPath string) (string, error) {
	outFullPath := filepath.Join(outPath, path.Base(fileInZip.Name))
	outFile, err := os.OpenFile(outFullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInZip.Mode())
	if err != nil {
		return "", fmt.Errorf("Failed to create output file: %v. Error: %v", outFullPath, err)
	}

	rc, err := fileInZip.Open()
	if err != nil {
		return "", fmt.Errorf("Failed to read %v from zip. Error: %v", fileInZip.Name, err)
	}

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return "", fmt.Errorf("Failed to write zip %v. Error: %v", outFullPath, err)
	}

	// Close the file without defer to close before next iteration of loop
	outFile.Close()
	rc.Close()
	return outFullPath, nil
}

func processEM(importId string, zipReader *zip.Reader, zippedData []byte, destBucket string, s3PathStart string, fs fileaccess.FileAccess, logger logger.ILogger) error {
	// EM data comes not via the FM pipeline, but more direct from the instrument via SDF peeks output format.
	// We search the zip dir tree for:
	// - sdf_raw.txt
	// - All *.msa fils
	// - All *.jpg files (and based on file name decide if they're needed)
	// SDF raw file may contain data from multiple scans, so we have to generate one scan per RTT found in sdf_raw.

	// We also have to run the beam location tool ourselves - there isn't one coming from sdf_raw.txt
	localTemp := filepath.Join(os.TempDir(), importId)
	localMSAPath := filepath.Join(localTemp, "msa")
	if err := os.MkdirAll(localMSAPath, 0644); err != nil {
		return fmt.Errorf("Failed to create output MSA path: %v. Error: %v", localMSAPath, err)
	}
	localImagesPath := filepath.Join(localTemp, "images")
	if err := os.MkdirAll(localImagesPath, 0644); err != nil {
		return fmt.Errorf("Failed to create output images path: %v. Error: %v", localImagesPath, err)
	}

	msas := []string{}
	images := []string{}
	sdfLocalPath := ""
	sdf_raw_zipPath := ""

	for _, f := range zipReader.File {
		if !f.FileInfo().IsDir() {
			// Add to list of files we're interested in
			if strings.HasSuffix(f.Name, "sdf_raw.txt") {
				sdf_raw_zipPath = path.Base(f.Name)
				if p, err := readFromZip(f, localTemp); err != nil {
					return err
				} else {
					sdfLocalPath = p
				}
			} else if strings.HasSuffix(f.Name, ".msa") {
				if p, err := readFromZip(f, localMSAPath); err != nil {
					return err
				} else {
					msas = append(msas, path.Base(p))
				}
			} else if strings.HasSuffix(f.Name, ".jpg") {
				if p, err := readFromZip(f, localImagesPath); err != nil {
					return err
				} else {
					images = append(images, path.Base(p))
				}
			}
		}
	}

	logger.Infof("Found sdf_raw: %v", sdf_raw_zipPath)
	logger.Infof("Found %v images", len(images))
	logger.Infof("Found %v histograms (MSA files)", len(msas))

	// Reject any scans that don't have histograms from the EM
	if len(msas) <= 0 {
		return fmt.Errorf("No histograms found")
	}
	if len(sdf_raw_zipPath) <= 0 {
		return fmt.Errorf("No sdf_raw.txt found")
	}

	// Create an RSI file from the sdf_raw file
	rsis, err := sdfToRSI.ConvertSDFtoRSIs(sdfLocalPath, localTemp)

	if err != nil {
		return fmt.Errorf("Failed to scan %v for RSI creation: %v", sdfLocalPath, err)
	}

	logger.Infof("Generated RSI files:")
	for _, rsi := range rsis {
		logger.Infof("  %v", rsi)
	}

	for _, rsi := range rsis {
		rxlPath, logPath, err := createBeamLocation(rsi, localTemp, logger)
		if err != nil {
			// Don't fail on errors for these - we may have run beam location tool on some incomplete scan, so failure isn't terrible!
			logger.Errorf("Beam location generation failed for RSI: %v. Error: %v", rsi, err)
			continue
		}

		// Read in the beam location file
		rxl, err := os.ReadFile(rxlPath)
		if err != nil {
			// Don't fail on errors for these - we may have run beam location tool on some incomplete scan, so failure isn't terrible!
			logger.Errorf("Failed to read generated beam location file: %v. Error: %v", rxlPath, err)
			continue
		}

		// Upload
		savePath := path.Join(s3PathStart, "beam-locations.csv")
		err = fs.WriteObject(destBucket, savePath, rxl)
		if err != nil {
			// We do want to fail here, this isn't an error related to input data - if we have the file, we should be able to upload it
			return err
		}

		logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

		// Also upload the log in case it's needed
		logdata, err := os.ReadFile(logPath)
		if err != nil {
			logger.Errorf("Failed to read beam geometry tool log: %v. Error: %v", logPath, err)
			continue
		}

		// Upload
		savePath = path.Join(s3PathStart, "beam-geom-log.txt")
		err = fs.WriteObject(destBucket, savePath, logdata)
		if err != nil {
			return err
		}

		logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)
	}

	// Upload the images
	for _, image := range images {
		p := filepath.Join(localImagesPath, image)
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("Failed to read image: %v. Error: %v", p, err)
		}

		savePath := path.Join(s3PathStart, image)
		err = fs.WriteObject(destBucket, savePath, b)
		if err != nil {
			return err
		}
		logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)
	}

	// Zip up the MSA's
	msaData, err := utils.ZipDirectory(localMSAPath)
	if err != nil {
		return fmt.Errorf("Failed to zip MSA files from: %v. Error: %v", localMSAPath, err)
	}

	// Upload the MSA zip
	savePath := path.Join(s3PathStart, "spectra.zip")
	err = fs.WriteObject(destBucket, savePath, msaData)
	if err != nil {
		return err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

	// Process each RSI file generated
	return nil
}

func createBeamLocation(rsiPath string, outputBeamLocationPath string, logger logger.ILogger) (string, string, error) {
	outSurfaceTop := filepath.Join(outputBeamLocationPath, "surface_top.txt")
	outRXL := filepath.Join(outputBeamLocationPath, "beam_location_RXL.csv")
	outLog := filepath.Join(outputBeamLocationPath, "log.txt")

	logger.Infof("Generating beam location CSV from: %v", rsiPath)

	bgtPath := "." + string(os.PathSeparator)

	if _, err := os.Stat(bgtPath + "BGT"); err != nil {
		// Try the path used in local testing
		bgtPath = ".." + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "beam-tool" + string(os.PathSeparator)

		/*if _, err := os.Stat(bgtPath + "BGT"); err != nil {
			// Try the path used in local testing
			fmt.Println("PATH WONT BE FOUND")

			d, _ := os.Getwd()
			fmt.Println(d)

			// Try get it backwards
			bgtPath = filepath.Dir(filepath.Dir(d)) + string(os.PathSeparator)
		}*/
	}

	cmd := exec.Command(bgtPath+"BGT", bgtPath+"Geometry_PIXL_EM_Landing_25Jan2021.csv", rsiPath, outSurfaceTop, outRXL, outLog)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = bgtPath

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("BGT tool error: %v", err)
	}

	/*
		procAttr := new(os.ProcAttr)
		procAttr.Files = []*os.File{nil, nil, nil}
		if _, err := os.StartProcess(bgtPath+"BGT", []string{bgtPath + "Geometry_PIXL_EM_Landing_25Jan2021.csv", rsiPath, outSurfaceTop, outRXL, outLog}, procAttr); err != nil {
			return fmt.Errorf("BGT tool error: %v", err)
		}
	*/
	// Make sure we have all output files
	outputs := []string{outSurfaceTop, outRXL, outLog}
	for _, out := range outputs {
		if _, err := os.Stat(out); err != nil {
			return "", "", fmt.Errorf("%v not found after BGT tool ran: %v", out, err)
		}
	}

	return outRXL, outLog, nil
}

func processBeadboard(format string, creatorUserId string, datasetID string, zipReader *zip.Reader, zippedData []byte, destBucket string, s3PathStart string, fs fileaccess.FileAccess, logger logger.ILogger) error {
	var err error

	// Expecting flat zip of MSA files
	count := 0
	for _, f := range zipReader.File {
		// If the zip path starts with __MACOSX, ignore it, it's garbage that a mac laptop has included...
		//if strings.HasPrefix(f.Name, "__MACOSX") {
		//	continue
		//}

		if f.FileInfo().IsDir() {
			return fmt.Errorf("Zip file must not contain sub-directories. Found: %v", f.Name)
		}

		if !strings.HasSuffix(f.Name, ".msa") {
			return fmt.Errorf("Zip file must only contain MSA files. Found: %v", f.Name)
		}
		count++
	}

	// Make sure it has at least one msa!
	if count <= 0 {
		return errors.New("Zip file did not contain any MSA files")
	}

	// Save the contents as a zip file in the uploads area
	savePath := path.Join(s3PathStart, "spectra.zip")
	err = fs.WriteObject(destBucket, savePath, zippedData)
	if err != nil {
		return err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

	// Now save detector info
	savePath = path.Join(s3PathStart, "import.json")
	importerFile := dataimportModel.BreadboardImportParams{
		MsaDir:           "spectra", // We now assume we will have a spectra.zip extracted into a spectra dir!
		MsaBeamParams:    "10,0,10,0",
		GenBulkMax:       true,
		GenPMCs:          true,
		ReadTypeOverride: "Normal",
		DetectorConfig:   "Breadboard",
		Group:            "JPL Breadboard",
		TargetID:         "0",
		SiteID:           0,

		CreatorUserId: creatorUserId,

		// The rest we set to the dataset ID
		DatasetID: datasetID,
		//Site: datasetID,
		//Target: datasetID,
		Title: datasetID,
		/*
			BeamFile // Beam location CSV path
			HousekeepingFile // Housekeeping CSV path
			ContextImgDir // Dir to find context images in
			PseudoIntensityCSVPath // Pseudointensity CSV path
			IgnoreMSAFiles // MSA files to ignore
			SingleDetectorMSAs // Expecting single detector (1 column) MSA files
			DetectorADuplicate // Duplication of detector A to B, because test MSA only had 1 set of spectra
			BulkQuantFile // Bulk quantification file (for tactical datasets)
			XPerChanA // eV calibration eV/channel (detector A)
			OffsetA // eV calibration eV start offset (detector A)
			XPerChanB // eV calibration eV/channel (detector B)
			OffsetB // eV calibration eV start offset (detector B)
			ExcludeNormalDwellSpectra // Hack for tactical datasets - load all MSAs to gen bulk sum, but dont save them in output
			SOL // Might as well be able to specify SOL. Needed for first spectrum dataset on SOL13
		*/
	}

	if format == "sbu-breadboard" {
		importerFile.Group = "Stony Brook Breadboard"
		importerFile.DetectorConfig = "StonyBrookBreadboard"
	}

	err = fs.WriteJSON(destBucket, savePath, importerFile)
	if err != nil {
		return err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)
	return nil
}

type importUpdater struct {
	session        *melody.Session
	melody         *melody.Melody
	notifier       services.INotifier
	scanIdImported string
	db             *mongo.Database
}

func (i *importUpdater) sendReimportUpdate(status *protos.JobStatus) {
	wsUpd := protos.WSMessage{
		Contents: &protos.WSMessage_ScanTriggerReImportUpd{
			ScanTriggerReImportUpd: &protos.ScanTriggerReImportUpd{
				Status: status,
			},
		},
	}

	wsHelpers.SendForSession(i.session, &wsUpd)

	if status.Status == protos.JobStatus_COMPLETE && status.EndUnixTimeSec > 0 {
		// Notify of our scan change
		i.notifier.SysNotifyScanChanged(i.scanIdImported)

		// Notify users
		scan, err := scan.ReadScanItem(status.JobItemId, i.db)
		if err != nil {
			fmt.Errorf("sendImportUpdate failed to read scan for id: %v, job id: %v", status.JobItemId, status.JobId)
			return
		}

		i.notifier.NotifyUpdatedScan(scan.Title, scan.Id)
	}
}

func (i *importUpdater) sendImportUpdate(status *protos.JobStatus) {
	wsUpd := protos.WSMessage{
		Contents: &protos.WSMessage_ScanUploadUpd{
			ScanUploadUpd: &protos.ScanUploadUpd{
				Status: status,
			},
		},
	}

	wsHelpers.SendForSession(i.session, &wsUpd)

	// If this is the final complete success message of a scan import, fire off a ScanListUpd to trigger
	// anyone who is connected to do a listing of scans
	// NOTE: IDEALLY this should happen when the scan notification happens. That process is not yet
	// implemented in the "new" way - Lambda completes but still needs to notify all instances of API
	// of the notification... For now this should work though
	if status.Status == protos.JobStatus_COMPLETE && status.EndUnixTimeSec > 0 {
		wsScanListUpd := protos.WSMessage{
			Contents: &protos.WSMessage_ScanListUpd{
				ScanListUpd: &protos.ScanListUpd{},
			},
		}

		bytes, err := proto.Marshal(&wsScanListUpd)
		if err == nil {
			i.melody.BroadcastBinary(bytes)
		}

		// Notify of our scan change
		i.notifier.SysNotifyScanChanged(i.scanIdImported)

		scan, err := scan.ReadScanItem(status.JobItemId, i.db)
		if err != nil {
			fmt.Errorf("sendImportUpdate failed to read scan for id: %v, job id: %v", status.JobItemId, status.JobId)
			return
		}

		i.notifier.NotifyNewScan(scan.Title, scan.Id)
	}
}

func HandleScanAutoShareReq(req *protos.ScanAutoShareReq, hctx wsHelpers.HandlerContext) (*protos.ScanAutoShareResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, 50); err != nil {
		return nil, err
	}

	// We don't check for permissions here...
	filter := bson.M{"_id": req.Id}

	opts := options.FindOne()
	ctx := context.TODO()

	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanAutoShareName)
	result := coll.FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Id)
		}
		return nil, result.Err()
	}

	item := &protos.ScanAutoShareEntry{}
	err := result.Decode(item)
	if err != nil {
		return nil, err
	}

	return &protos.ScanAutoShareResp{
		Entry: item,
	}, nil
}

func HandleScanAutoShareWriteReq(req *protos.ScanAutoShareWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanAutoShareWriteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Entry.Id, "Id", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanAutoShareName)

	// We don't check for permissions here...

	// If no permissions to assign, delete it
	if req.Entry.Editors == nil && req.Entry.Viewers == nil {
		// Just delete here
		filter := bson.M{"_id": req.Entry.Id}
		delResult, err := coll.DeleteOne(ctx, filter, options.Delete())
		if err != nil {
			return nil, err
		}

		if delResult.DeletedCount != 1 {
			hctx.Svcs.Log.Errorf("HandleScanAutoShareWriteReq: delete for %v failed: %+v", req.Entry.Id, delResult)
		}
	} else {
		opts := options.Update().SetUpsert(true)
		result, err := coll.UpdateByID(ctx, req.Entry.Id, bson.D{{Key: "$set", Value: req.Entry}}, opts)
		if err != nil {
			return nil, err
		}

		if result.MatchedCount != 1 {
			hctx.Svcs.Log.Errorf("HandleScanAutoShareWriteReq: write for %v failed: %+v", req.Entry.Id, result)
		}
	}

	return &protos.ScanAutoShareWriteResp{}, nil
}

func HandleScanTriggerAutoQuantReq(req *protos.ScanTriggerAutoQuantReq, hctx wsHelpers.HandlerContext) (*protos.ScanTriggerAutoQuantResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Run auto-quants no matter if they have run already
	quantification.RunAutoQuantifications(req.ScanId, hctx.Svcs, false)

	return &protos.ScanTriggerAutoQuantResp{}, nil
}
