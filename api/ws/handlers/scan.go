package wsHandler

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pixlise/core/v3/api/dataimport"
	dataimportModel "github.com/pixlise/core/v3/api/dataimport/models"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/indexcompression"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleScanListReq(req *protos.ScanListReq, hctx wsHelpers.HandlerContext) (*protos.ScanListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_SCAN, hctx)
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

	return &protos.ScanListResp{
		Scans: scans,
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

func HandleScanMetaWriteReq(req *protos.ScanMetaWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaWriteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Title, "Title", 1, 100); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Description, "Description", 1, 600); err != nil {
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

	update := bson.D{bson.E{Key: "title", Value: req.Title}, bson.E{Key: "description", Value: req.Description}}

	result, err := coll.UpdateByID(ctx, req.ScanId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.ScanId)
	}

	return &protos.ScanMetaWriteResp{}, nil
}

func HandleScanTriggerReImportReq(req *protos.ScanTriggerReImportReq, hctx wsHelpers.HandlerContext) (*protos.ScanTriggerReImportResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, 50); err != nil {
		return nil, err
	}

	result, logId, err := dataimport.TriggerDatasetReprocessViaSNS(hctx.Svcs.SNS, hctx.Svcs.IDGen, req.ScanId, hctx.Svcs.Config.DataSourceSNSTopic)

	hctx.Svcs.Log.Infof("Triggered dataset reprocess via SNS topic. Result: %v. Log ID: %v", result, logId)
	return &protos.ScanTriggerReImportResp{LogId: logId}, err
}

func HandleScanUploadReq(req *protos.ScanUploadReq, hctx wsHelpers.HandlerContext) (*protos.ScanUploadResp, error) {
	destBucket := hctx.Svcs.Config.ManualUploadBucket
	fs := hctx.Svcs.FS
	logger := hctx.Svcs.Log
	logger.Debugf("Dataset create started for format: %v, id: %v", req.Id, req.Format)

	// Validate the dataset ID - can't contain funny characters because it ends up as an S3 path
	// NOTE: we also turn space to _ here! Having spaces in the path broke quants because the
	// quant summary file was written with a + instead of a space?!
	datasetID := fileaccess.MakeValidObjectName(req.Id, false)

	formats := []string{"jpl-breadboard", "sbu-breadboard", "pixl-em"}
	if !utils.ItemInSlice(req.Format, formats) {
		return nil, fmt.Errorf("Unexpected format: \"%v\"", req.Format)
	}

	s3PathStart := path.Join(filepaths.DatasetUploadRoot, datasetID)

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
		return nil, err
	}

	// Validate zip contents matches the format we were given
	zipReader, err := zip.NewReader(bytes.NewReader(req.ZippedData), int64(len(req.ZippedData)))
	if err != nil {
		return nil, err
	}

	// Validate contents - detector dependent
	if req.Format == "pixl-em" {
		// Expecting certain product dirs, but don't be too prescriptive
		foundHousekeeping := false
		foundBeamLocation := false
		foundSpectra := false
		for _, f := range zipReader.File {
			if f.FileInfo().IsDir() {
				if f.Name == "RFS" {
					foundSpectra = true
				} else if f.Name == "RXL" {
					foundBeamLocation = true
				} else if f.Name == "RSI" {
					foundHousekeeping = true
				}
			} else {
				if strings.HasPrefix(f.Name, "RFS") {
					foundSpectra = true
				} else if strings.HasPrefix(f.Name, "RXL") {
					foundBeamLocation = true
				} else if strings.HasPrefix(f.Name, "RSI") {
					foundHousekeeping = true
				}
			}
		}

		if !foundHousekeeping {
			return nil, fmt.Errorf("Zip file missing RSI sub-directory")
		}
		if !foundBeamLocation {
			return nil, fmt.Errorf("Zip file missing RXL sub-directory")
		}
		if !foundSpectra {
			return nil, fmt.Errorf("Zip file missing RFS sub-directory")
		}

		// Save the contents as a zip file in the uploads area
		savePath := path.Join(s3PathStart, "data.zip")
		err = fs.WriteObject(destBucket, savePath, req.ZippedData)
		if err != nil {
			return nil, err
		}
		logger.Debugf("  Wrote: s3://%v/%v", destBucket, savePath)
	} else {
		// Expecting flat zip of MSA files
		count := 0
		for _, f := range zipReader.File {
			// If the zip path starts with __MACOSX, ignore it, it's garbage that a mac laptop has included...
			//if strings.HasPrefix(f.Name, "__MACOSX") {
			//	continue
			//}

			if f.FileInfo().IsDir() {
				return nil, fmt.Errorf("Zip file must not contain sub-directories. Found: %v", f.Name)
			}

			if !strings.HasSuffix(f.Name, ".msa") {
				return nil, fmt.Errorf("Zip file must only contain MSA files. Found: %v", f.Name)
			}
			count++
		}

		// Make sure it has at least one msa!
		if count <= 0 {
			return nil, errors.New("Zip file did not contain any MSA files")
		}

		// Save the contents as a zip file in the uploads area
		savePath := path.Join(s3PathStart, "spectra.zip")
		err = fs.WriteObject(destBucket, savePath, req.ZippedData)
		if err != nil {
			return nil, err
		}
		logger.Debugf("  Wrote: s3://%v/%v", destBucket, savePath)

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

			CreatorUserId: hctx.SessUser.User.Id,

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

		if req.Format == "sbu-breadboard" {
			importerFile.Group = "Stony Brook Breadboard"
			importerFile.DetectorConfig = "StonyBrookBreadboard"
		}

		err = fs.WriteJSON(destBucket, savePath, importerFile)
		if err != nil {
			return nil, err
		}
		logger.Debugf("  Wrote: s3://%v/%v", destBucket, savePath)
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

	// Now save creator info
	savePath = path.Join(s3PathStart, "creator.json")
	err = fs.WriteJSON(destBucket, savePath, hctx.SessUser.User)
	if err != nil {
		return nil, err
	}
	logger.Debugf("  Wrote: s3://%v/%v", destBucket, savePath)

	// Now we trigger a dataset conversion
	result, logId, err := dataimport.TriggerDatasetReprocessViaSNS(hctx.Svcs.SNS, hctx.Svcs.IDGen, datasetID, hctx.Svcs.Config.DataSourceSNSTopic)
	if err != nil {
		return nil, err
	}

	logger.Infof("Triggered dataset reprocess via SNS topic. Result: %v. Log ID: %v", result, logId)

	return &protos.ScanUploadResp{LogId: logId}, nil
}
