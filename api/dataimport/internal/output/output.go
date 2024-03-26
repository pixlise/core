// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Allows outputting (in PIXLISE protobuf dataset format) of in-memory representation of PIXL data that importer has read. Also
// outputs summary JSON files for the dataset.
package output

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/beamLocation"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

// PIXLISEDataSaver - module to save the internal representation of a dataset
type metaInfo struct {
	label    string
	index    int32
	dataType protos.Experiment_MetaDataType
}

type PIXLISEDataSaver struct {
	metaLookup map[string]metaInfo
}

// Save - saves internal representation of dataset (outputData)
func (s *PIXLISEDataSaver) Save(
	data dataConvertModels.OutputData,
	contextImageSrcPath string,
	outputDatasetPath string,
	outputImagesPath string,
	db *mongo.Database,
	creationUnixTimeSec int64,
	jobLog logger.ILogger) error {
	jobLog.Infof("Serializing dataset...")

	outPrefix := ""

	// Prepare to receive meta values
	s.metaLookup = map[string]metaInfo{}

	exp := protos.Experiment{}

	// Set all dataset targetting information/metadata, this is what Dataset Summary will be generated from...
	exp.TargetId = data.Meta.TargetID
	exp.SiteId = data.Meta.SiteID
	exp.DriveId = data.Meta.DriveID
	exp.Target = data.Meta.Target
	exp.Site = data.Meta.Site
	exp.Title = data.Meta.Title
	exp.Sol = data.Meta.SOL

	rtt, err := strconv.Atoi(data.Meta.RTT)
	if err != nil {
		jobLog.Infof("Warning: Failed to convert RTT %v to number. Error: %v. This can be ignored for non-FM datasets", data.Meta.RTT, err)
	} else {
		exp.Rtt = int32(rtt)
	}

	exp.Sclk = data.Meta.SCLK

	jobLog.Infof("This dataset's detector config is %v", data.DetectorConfig)
	exp.DetectorConfig = data.DetectorConfig

	// NOTE: count values are saved after we saved locations, see: saveSpectrumTypeCounts

	if len(data.DefaultContextImage) <= 0 {
		jobLog.Infof("WARNING: No main context image determined")
	} else {
		jobLog.Infof("Main context image: %v", data.DefaultContextImage)
	}

	if len(data.BulkQuantFile) > 0 {
		exp.BulkSumQuantFile = data.BulkQuantFile
	}

	// If we're combining from multiple sources, save metadata for each source
	for _, src := range data.Sources {
		exp.ScanSources = append(exp.ScanSources, &protos.Experiment_ScanSource{
			Instrument:       "PIXL", // FIXME combine
			TargetId:         src.TargetID,
			DriveId:          src.DriveID,
			SiteId:           src.SiteID,
			Target:           src.Target,
			Site:             src.Site,
			Title:            src.Title,
			Sol:              src.SOL,
			Rtt:              src.RTT,
			Sclk:             src.SCLK,
			BulkSumQuantFile: "",
			DetectorConfig:   data.DetectorConfig, // FIXME combine
			IdOffset:         src.PMCOffset,
		})
	}

	// Get a sorted list of PMCs, so we save them in order
	// It's not mandatory, but nicer!
	pmcs := []int{}
	for pmc := range data.PerPMCData {
		pmcs = append(pmcs, int(pmc))
	}
	sort.Ints(pmcs)

	// Check that the min PMC is valid
	if pmcs[0] < 0 {
		return fmt.Errorf("Lowest PMC detected was %v", pmcs[0])
	}
	if pmcs[len(pmcs)-1] < pmcs[0] {
		return fmt.Errorf("Highest PMC detected (%v) should be higher than lowest pmc %v", pmcs[len(pmcs)-1], pmcs[0])
	}

	// Now fill in the other context image entries
	exp.AlignedContextImages = []*protos.Experiment_ContextImageCoordinateInfo{}
	exp.UnalignedContextImages = []string{}

	// Run through all locations and build the list of all context images by PMC. We then get a list of PMCs that have beam
	// data, and break this list into aligned and unaligned images
	contextImagesByPMC := map[int]string{}
	pmcsWithBeamIJs := []int{}

	for _, pmcI := range pmcs {
		pmc := int32(pmcI)
		dataForPMC := data.PerPMCData[pmc]

		// Store in PMC->context image lookup
		if len(dataForPMC.ContextImageDst) > 0 {
			contextImagesByPMC[pmcI] = dataForPMC.ContextImageDst
		}

		// If this PMC is the first one with beam locations, store the list of PMCs that need them
		if len(pmcsWithBeamIJs) <= 0 && dataForPMC.Beam != nil {
			for beamPMC := range dataForPMC.Beam.IJ {
				pmcsWithBeamIJs = append(pmcsWithBeamIJs, int(beamPMC))
			}
		}
	}

	// Store these PMCs in order because we want lowest PMC coordinates to end up in ImageI/ImageJ, see saveExperimentLocationItem()
	sort.Ints(pmcsWithBeamIJs)

	// Now partition into the 2 lists. Any associated with our PMCs are aligned, and we remove from the map
	// therefore anything remaining is unaligned
	// Also note that one of the images should be set in MainContextImage - generally the one with the lowest PMC, but not required.
	// We want to ensure that the image in MainContextImage is not repeated in the other 2 arrays.
	/*if len(exp.MainContextImage) <= 0 {
		return errors.New("MainContextImage not set")
	}*/

	mainContextMatched := false
	for _, pmc := range pmcsWithBeamIJs {
		img, ok := contextImagesByPMC[pmc]
		if !ok {
			// Looks like we had IJ's defined in beam location for a PMC, but we don't have a context image for it.
			// Just print a warning...
			jobLog.Infof("WARNING: Context image not found for PMC: %v", pmc)
		} else {
			// If it's the main context image, note that we found one...
			if exp.MainContextImage == img {
				mainContextMatched = true
			}

			item := &protos.Experiment_ContextImageCoordinateInfo{
				Image:              img,
				Pmc:                int32(pmc),
				TrapezoidCorrected: false,
			}
			exp.AlignedContextImages = append(exp.AlignedContextImages, item)

			// Remove from map
			delete(contextImagesByPMC, pmc)
		}
	}

	// Verify that main has this set...
	if len(exp.MainContextImage) > 0 && !mainContextMatched {
		return fmt.Errorf("Main context image inconsistant: \"%v\" does not match any context images defined for PMCs", exp.MainContextImage)
	}

	// Remainder are unaligned
	for _, img := range contextImagesByPMC {
		exp.UnalignedContextImages = append(exp.UnalignedContextImages, img)
	}

	// RGBU images are also unaligned for now
	for _, meta := range data.RGBUImages {
		//exp.UnalignedContextImages = append(exp.UnalignedContextImages, img)
		exp.UnalignedContextImages = append(exp.UnalignedContextImages, makeRGBUFileName(meta))
	}

	// As are other context images for visual spectroscopy taken with the "disco" setup - different coloured LEDs
	for _, meta := range data.DISCOImages {
		//exp.UnalignedContextImages = append(exp.UnalignedContextImages, img)
		exp.UnalignedContextImages = append(exp.UnalignedContextImages, makeDiscoFileName(meta))
	}

	// Now loop through them, saving in this order...
	jobLog.Infof("Saving images by PMC...")
	for _, pmcI := range pmcs {
		pmc := int32(pmcI)
		dataForPMC := data.PerPMCData[pmc]
		// If there is a source RTT, we need to look up the index of which saved source item this corresponds to
		srcIdx := int32(0)
		if len(dataForPMC.SourceRTT) > 0 {
			for c, src := range exp.ScanSources {
				if src.Rtt == dataForPMC.SourceRTT {
					srcIdx = int32(c)
					break
				}
			}
		}

		err := s.saveExperimentLocationItem(&exp, pmc, srcIdx, *dataForPMC, data.HousekeepingHeaders, pmcsWithBeamIJs, jobLog)
		if err != nil {
			return fmt.Errorf("Error saving pmc %v: %v", pmc, err)
		}
	}

	jobLog.Infof("Saving %v field names...", len(s.metaLookup))
	err = s.saveMetaData(&exp)
	if err != nil {
		return err
	}

	if len(data.PseudoRanges) > 0 {
		jobLog.Infof("Saving %v pseudo-intensity ranges...", len(data.PseudoRanges))
		savePseudoIntensityRanges(&exp, data.PseudoRanges)
	}

	// Now save the counts
	saveSpectrumTypeCounts(&exp, data)
	whatDir := []string{"dataset", "images"}
	dirs := []string{outputDatasetPath, outputImagesPath}

	for c, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			jobLog.Infof("Creating %v output directory: \"%v\"", whatDir[c], dir)
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("Failed to create %v output directory: %v", whatDir[c], dir)
			}
		}
	}

	// Look up who to auto-share with based on creator ID
	coll := db.Collection(dbCollections.ScanAutoShareName)
	optFind := options.FindOne()

	autoShare := &protos.ScanAutoShareEntry{}
	sharer := data.CreatorUserId

	if len(sharer) <= 0 {
		// we dont have a creator, so probably started as an automated process. Here we
		// try to look up the auto-share destination by instrument type
		sharer = data.Instrument.String()
	}

	jobLog.Infof("Looking up auto-share group(s) for: \"%v\"", sharer)

	autoShareResult := coll.FindOne(context.TODO(), bson.D{{Key: "_id", Value: sharer}}, optFind)
	if autoShareResult.Err() != nil {
		// We couldn't find someone to auto-share it with, if we don't have anyone to share with, just fail here
		if autoShareResult.Err() == mongo.ErrNoDocuments {
			// If the user has no auto-share destination configured, share with just the user - BUT if we're
			// not dealing with a user here, we must be importing via the pipeline, in which case it should've
			// been configured to share already...
			if len(data.CreatorUserId) > 0 {
				jobLog.Infof("No auto-share destination found, so only importing user will be able to access this dataset.")
				autoShare.Id = data.CreatorUserId
				autoShare.Viewers = &protos.UserGroupList{UserIds: []string{}, GroupIds: []string{}}
				autoShare.Editors = &protos.UserGroupList{UserIds: []string{data.CreatorUserId}, GroupIds: []string{}}
			} else {
				return fmt.Errorf("Cannot work out groups to auto-share imported dataset with")
			}
		}
		return autoShareResult.Err()
	} else {
		err := autoShareResult.Decode(autoShare)

		if err != nil {
			return fmt.Errorf("Failed to decode auto share configuration: %v", err)
		}
	}

	// We work out the default file name when copying output images now... because if there isn't one, we may pick one during that process.
	defaultContextImage, err := copyImagesToOutput(contextImageSrcPath, []string{data.DatasetID}, data.DatasetID, outputImagesPath, data, db, jobLog)
	exp.MainContextImage = defaultContextImage

	// Set any matched aligned images - this happens after copyImagesToOutput because file names may be modified by it depending on formats
	err = setMatchedImageInfo(data, &exp, jobLog)
	if err != nil {
		return err
	}

	// Save the image beam locations to DB. They are already in the experiment file (dataset.bin) but those aren't read any more
	// as we switched to storing them in DB (to allow import of other images with a corresponding set of beam locations)
	// Redundant, but this is how it evolved...
	for idx, imgItem := range exp.AlignedContextImages {
		beamVer := data.BeamVersion
		if beamVer < 1 {
			beamVer = 1
		}
		err := beamLocation.ImportBeamLocationToDB(imgItem.Image, data.Instrument, data.DatasetID, beamVer, idx, &exp, db, jobLog)
		if err != nil {
			return fmt.Errorf("Failed to import beam locations for image %v into DB. Error: %v", imgItem.Image, err)
		}
	}

	outfileName := outPrefix + filepaths.DatasetFileName
	outFilePath := filepath.Join(outputDatasetPath, outfileName)

	jobLog.Infof("Writing binary file: %v", outFilePath)
	out, err := proto.Marshal(&exp)
	if err != nil {
		return fmt.Errorf("Failed to encode dataset: %v", err)
	}
	if err := os.WriteFile(outFilePath, out, 0644); err != nil {
		return fmt.Errorf("Failed to write dataset file: %v", err)
	}

	fi, err := os.Stat(outFilePath)
	if err != nil || fi == nil {
		return fmt.Errorf("Failed to get dataset file size for: %v", outFilePath)
	}

	summaryData := makeSummaryFileContent(&exp, data.DatasetID, data.Instrument, data.Meta, int(fi.Size()), creationUnixTimeSec, data.CreatorUserId)

	jobLog.Infof("Writing summary to DB...")

	coll = db.Collection(dbCollections.ScansName)
	opt := options.Update().SetUpsert(true)

	result, err := coll.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: summaryData.Id}}, bson.D{{Key: "$set", Value: summaryData}}, opt)

	if err != nil {
		jobLog.Errorf("Failed to write summary to DB: %v", err)
		return err
	} else if result.UpsertedCount != 1 {
		jobLog.Errorf("Expected summary write to create 1 upsert, got: %v", result.UpsertedCount)
	}

	// Set ownership
	ownerItem, err := wsHelpers.MakeOwnerForWrite(summaryData.Id, protos.ObjectType_OT_SCAN, summaryData.CreatorUserId, creationUnixTimeSec)

	ownerItem.Viewers = autoShare.Viewers
	ownerItem.Editors = autoShare.Editors

	coll = db.Collection(dbCollections.OwnershipName)
	opt = options.Update().SetUpsert(true)

	result, err = coll.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: ownerItem.Id}}, bson.D{{Key: "$set", Value: ownerItem}}, opt)
	if err != nil {
		jobLog.Errorf("Failed to write ownership item to DB: %v", err)
		return err
	}

	bulkSpectraCount := summaryData.ContentCounts["BulkSpectra"]
	maxSpectraCount := summaryData.ContentCounts["MaxSpectra"]

	if bulkSpectraCount < 2 || maxSpectraCount < 2 {
		jobLog.Infof("WARNING: NOT ENOUGH BULK/MAX SPECTRA DEFINED! Bulk: %v, Max: %v", bulkSpectraCount, maxSpectraCount)
	}

	// If we don't have a default image set for this dataset, set one (otherwise user may have changed it, don't want to overwrite)
	err = insertDefaultImage(db, summaryData.Id, data.DefaultContextImage, jobLog)
	return err
}

func insertDefaultImage(db *mongo.Database, scanId string, defaultImage string, jobLog logger.ILogger) error {
	if len(scanId) <= 0 || len(defaultImage) <= 0 {
		return nil // Don't write empty stuff
	}

	coll := db.Collection(dbCollections.ScanDefaultImagesName)
	opt := options.InsertOne()

	defaultImageResult, err := coll.InsertOne(context.TODO(), &protos.ScanImageDefaultDB{ScanId: scanId, DefaultImageFileName: defaultImage}, opt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Don't overwrite, so we're OK with this
			return nil
		}
		return err
	}

	if defaultImageResult.InsertedID != scanId {
		jobLog.Errorf("insertDefaultImage wrote id %v, got back %v", scanId, defaultImageResult.InsertedID)
		// Not the end of the world... don't error out here
	}

	return nil
}

func makeRGBUFileName(meta dataConvertModels.ImageMeta) string {
	//return fmt.Sprintf("RGBU_PMC_%v_%v.tif", meta.PMC, meta.ProdType)
	return path.Base(meta.FileName)
}

// Must be called after experiment locations are set, because this reads from them to count...
func saveSpectrumTypeCounts(exp *protos.Experiment, data dataConvertModels.OutputData) {
	readTypeIdx := 0
	for _, l := range exp.MetaLabels {
		if l == "READTYPE" {
			break
		}
		readTypeIdx = readTypeIdx + 1
	}

	normalSpectraCount := int32(0)
	dwellSpectraCount := int32(0)
	maxSpectraCount := int32(0)
	bulkSpectraCount := int32(0)
	pseudoIntensityCount := int32(0)
	//contextImgCount := int32(0)

	for _, loc := range exp.Locations {
		for _, det := range loc.Detectors {
			for _, meta := range det.Meta {
				if meta.LabelIdx == int32(readTypeIdx) {
					if meta.Svalue == "Normal" {
						normalSpectraCount = normalSpectraCount + 1
					}
					if meta.Svalue == "Dwell" {
						dwellSpectraCount = dwellSpectraCount + 1
					}
					if meta.Svalue == "BulkSum" {
						bulkSpectraCount = bulkSpectraCount + 1
					}
					if meta.Svalue == "MaxValue" {
						maxSpectraCount = maxSpectraCount + 1
					}
				}
			}
		}
		if len(loc.PseudoIntensities) > 0 {
			pseudoIntensityCount = pseudoIntensityCount + 1
		}
		/*if len(loc.ContextImage) > 0 {
			contextImgCount++
		}*/
	}

	// Set on the experiment
	exp.BulkSpectra = bulkSpectraCount
	exp.DwellSpectra = dwellSpectraCount
	exp.MaxSpectra = maxSpectraCount
	exp.NormalSpectra = normalSpectraCount
	exp.PseudoIntensities = pseudoIntensityCount
}

func copyImagesToOutput(
	contextImgDir string,
	associatedScanIds []string,
	originScanId string,
	outPath string,
	data dataConvertModels.OutputData,
	db *mongo.Database, jobLog logger.ILogger) (string, error) {
	defaultContextImage := ""

	// Copy the context images into the output dir
	// Also making sure that one of them matches what we have set as the default image
	defaultMatched := false

	pmcToImage := map[int32]string{}

	for pmc, item := range data.PerPMCData {
		if len(item.ContextImageSrc) > 0 {
			fromImgFile := filepath.Join(contextImgDir, item.ContextImageSrc)
			outImgFile := filepath.Join(outPath, item.ContextImageDst)

			// Make sure output format is PNG
			if strings.ToUpper(filepath.Ext(fromImgFile)) == ".TIF" {
				outImgFile = outImgFile[0:len(outImgFile)-3] + "png"
				jobLog.Infof("  Convert img PMC[%v] %v -> %v", pmc, fromImgFile, outImgFile)

				err := convertTiffToPNG(fromImgFile, outImgFile)
				if err != nil {
					return "", err
				}
			} else {
				jobLog.Infof("  Copy img PMC[%v] %v -> %v", pmc, fromImgFile, outImgFile)

				err := fileaccess.CopyFileLocally(fromImgFile, outImgFile)
				if err != nil {
					return "", err
				}
			}

			// If it definitely came from FM, we store the name of the file to search for in mars viewer to see this
			// Otherwise it's a blank string, so this also helps any code that wants to open this in mars viewer...
			originURL := ""
			fileName := filepath.Base(outImgFile)
			fileNameMeta, err := gdsfilename.ParseFileName(fileName)
			if err == nil {
				sol, err := fileNameMeta.SOL()
				if err == nil {
					iSol, err := strconv.Atoi(sol)
					if err == nil && iSol > 1 {
						// It's an FM dataset with a real SOL so lets save this name in our URL
						originURL = fileName
					}
				}
			}

			err = insertImageDBEntryForImage(outImgFile, db, protos.ScanImageSource_SI_INSTRUMENT, protos.ScanImagePurpose_SIP_VIEWING, associatedScanIds, originScanId, originURL, nil, jobLog)
			if err != nil {
				return "", err
			}

			// Remember this PMC->file name mapping for any potential "matched" images we import
			pmcToImage[pmc] = fileName

			if data.DefaultContextImage == item.ContextImageDst {
				defaultMatched = true
				defaultContextImage = item.ContextImageDst
			}
		}
	}

	// Copying RGBU images untouched
	for _, img := range data.RGBUImages {
		fromImgFile := filepath.Join(contextImgDir, img.FileName)

		// These paths come in with their product type prefix, eg DTU/something.tif
		// Here we want an output path that doesn't include the extra product type
		// NOTE: THIS MUST MATCH WHAT WAS WRITTEN INTO UnalignedContextImages!!!
		outImgFile := filepath.Join(outPath, makeRGBUFileName(img)) //path.Base(rgbuPath))

		jobLog.Infof("  Copy RGBU img %v -> %v", fromImgFile, outImgFile)

		err := fileaccess.CopyFileLocally(fromImgFile, outImgFile)
		if err != nil {
			return "", err
		}

		err = insertImageDBEntryForImage(outImgFile, db, protos.ScanImageSource_SI_UPLOAD, protos.ScanImagePurpose_SIP_MULTICHANNEL, associatedScanIds, originScanId, "", nil, jobLog)
		if err != nil {
			return "", err
		}
	}

	// Also copy DISCO images
	for _, meta := range data.DISCOImages {
		fromImgFile := filepath.Join(contextImgDir, meta.FileName)

		// These paths come in with their product type prefix, eg DTU/something.tif
		// Here we want an output path that doesn't include the extra product type
		// NOTE: THIS MUST MATCH WHAT WAS WRITTEN INTO UnalignedContextImages!!!
		outFileName := makeDiscoFileName(meta)
		outImgFile := filepath.Join(outPath, outFileName)

		jobLog.Infof("  Copy MCC multispectral img %v -> %v", fromImgFile, outImgFile)

		err := fileaccess.CopyFileLocally(fromImgFile, outImgFile)
		if err != nil {
			return "", err
		}

		err = insertImageDBEntryForImage(outImgFile, db, protos.ScanImageSource_SI_UPLOAD, protos.ScanImagePurpose_SIP_MULTICHANNEL, associatedScanIds, originScanId, "", nil, jobLog)
		if err != nil {
			return "", err
		}

		// This image could be our default context image - this is only for DISCO datasets
		if data.DefaultContextImage == meta.FileName {
			defaultMatched = true
			defaultContextImage = outFileName
		}
	}

	// Matched-aligned context images, ie WATSON images that are transformed to match MCC images
	for _, matchedMeta := range data.MatchedAlignedImages {
		// We assume here that we're reading FULL paths, get just the file name
		fromImgFile := matchedMeta.MatchedImageFullPath
		matchedFileName := path.Base(matchedMeta.MatchedImageName)

		outImgFile := filepath.Join(outPath, matchedFileName)

		jobLog.Infof("  Copy matched aligned img %v -> %v", fromImgFile, outImgFile)

		err := fileaccess.CopyFileLocally(fromImgFile, outImgFile)
		if err != nil {
			return "", err
		}

		// Find the beam image file name that we are matched against
		beamImgName := pmcToImage[matchedMeta.AlignedBeamPMC]

		if len(beamImgName) <= 0 {
			return "", fmt.Errorf("Matched image for PMC: %v - image not found", matchedMeta.AlignedBeamPMC)
		}

		matchInfo := &protos.ImageMatchTransform{
			BeamImageFileName: beamImgName,
			XOffset:           matchedMeta.XOffset,
			YOffset:           matchedMeta.YOffset,
			XScale:            matchedMeta.XScale,
			YScale:            matchedMeta.YScale,
		}

		err = insertImageDBEntryForImage(outImgFile, db, protos.ScanImageSource_SI_UPLOAD, protos.ScanImagePurpose_SIP_VIEWING, associatedScanIds, originScanId, "", matchInfo, jobLog)
		if err != nil {
			return "", err
		}
	}

	if len(data.DefaultContextImage) > 0 && !defaultMatched {
		return "", fmt.Errorf("Main context image \"%v\" was not found when copying to output directory", data.DefaultContextImage)
	}

	return defaultContextImage, nil
}

func insertImageDBEntryForImage(
	imagePath string,
	db *mongo.Database,
	source protos.ScanImageSource,
	purpose protos.ScanImagePurpose,
	associatedScanIds []string,
	originScanId string,
	originImageURL string,
	matchInfo *protos.ImageMatchTransform,
	jobLog logger.ILogger) error {
	// Read the image - we used to only copy files around but here we need to open it for meta data
	imgFile, err := utils.ReadImageFile(imagePath)
	if err != nil {
		return err
	}

	//defer imgFile.Close()

	stats, err := os.Stat(imagePath)

	saveName := filepath.Base(imagePath)
	savePath := path.Join(originScanId, saveName)

	img := utils.MakeScanImage(savePath, uint32(stats.Size()), source, purpose, associatedScanIds, originScanId, originImageURL, matchInfo, imgFile)
	return insertImageDBEntry(db, img, jobLog)
}

func insertImageDBEntry(db *mongo.Database, image *protos.ScanImage, jobLog logger.ILogger) error {
	if image == nil {
		return nil // Don't write empty stuff
	}

	coll := db.Collection(dbCollections.ImagesName)
	opt := options.InsertOne()

	result, err := coll.InsertOne(context.TODO(), image, opt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Don't overwrite, so we're OK with this
			return nil
		}
		return err
	}

	if result.InsertedID != image.ImagePath {
		jobLog.Errorf("insertImageDBEntry wrote id %v, got back %v", image.ImagePath, result.InsertedID)
		// Not the end of the world... don't error out here
	}

	return nil
}

func setMatchedImageInfo(fromData dataConvertModels.OutputData, toExperiment *protos.Experiment, jobLog logger.ILogger) error {
	for _, matchedImg := range fromData.MatchedAlignedImages {
		matchItem := &protos.Experiment_MatchedContextImageInfo{}

		// Search for the index to set for the referenced aligned image
		// NOTE: if we don't have aligned images, we just have to ignore this, as this dataset will be created with ONLY 1 or more matched
		// images - matched to the beam location PMC...
		if len(toExperiment.AlignedContextImages) > 0 {
			found := false

			// look up the saved name of the image
			for c, aligned := range toExperiment.AlignedContextImages {
				if aligned.Pmc == matchedImg.AlignedBeamPMC {
					matchItem.AlignedIndex = int32(c)
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("Failed to find index of aligned image %v for PMC %v", matchedImg.MatchedImageFullPath, matchedImg.AlignedBeamPMC)
			}
		}

		matchItem.Image = matchedImg.MatchedImageName
		matchItem.XOffset = matchedImg.XOffset
		matchItem.YOffset = matchedImg.YOffset
		matchItem.XScale = matchedImg.XScale
		matchItem.YScale = matchedImg.YScale

		toExperiment.MatchedAlignedContextImages = append(toExperiment.MatchedAlignedContextImages, matchItem)
		jobLog.Infof("Matched aligned image: %v, offset(%v, %v), scale(%v, %v). Match for aligned index: %v", matchItem.Image, matchItem.XOffset, matchItem.YOffset, matchItem.XScale, matchItem.YScale, matchItem.AlignedIndex)
	}

	return nil
}

func makeDiscoFileName(meta dataConvertModels.ImageMeta) string {
	//return fmt.Sprintf("MCC_MultiSpectral_%v_%v.png", meta.PMC, meta.LEDs)
	return path.Base(meta.FileName)
}

func (s *PIXLISEDataSaver) saveSpectrumMeta(meta dataConvertModels.MetaData, detector *protos.Experiment_Location_DetectorSpectrum) error {
	// NOTE: Here we read from the map in alphabetical order. This is purely because Go map ordering is undefined (and changes by
	// definition run-to-run, you're not meant to rely on it). Therefore, if we regenerate the same dataset, we output different files
	// unless this order is specified
	keys := []string{}
	for label := range meta {
		keys = append(keys, label)
	}

	sort.Strings(keys)

	//for label, metaValue := range meta {
	for _, label := range keys {
		metaValue := meta[label]

		idx, err := s.getMetaIndex(label, metaValue.DataType)
		if err != nil {
			return err
		}

		saveMeta := &protos.Experiment_Location_MetaDataItem{}
		s.convertToOutputMeta(metaValue, idx, saveMeta)
		detector.Meta = append(detector.Meta, saveMeta)
	}

	return nil
}

func (s *PIXLISEDataSaver) convertToOutputMeta(meta dataConvertModels.MetaValue, labelIdx int32, saveMeta *protos.Experiment_Location_MetaDataItem) {
	saveMeta.LabelIdx = labelIdx

	// Depending on the type...
	switch meta.DataType {
	case protos.Experiment_MT_STRING:
		saveMeta.Svalue = meta.SValue
	case protos.Experiment_MT_INT:
		saveMeta.Ivalue = meta.IValue
	case protos.Experiment_MT_FLOAT:
		saveMeta.Fvalue = meta.FValue
	}
}

func (s *PIXLISEDataSaver) saveExperimentLocationItem(saveToExperiment *protos.Experiment, pmc int32, srcIdx int32, data dataConvertModels.PMCData, hkHeaders []string, beamIJPMCAscending []int, jobLog logger.ILogger) error {
	location := &protos.Experiment_Location{}
	location.Id = strconv.Itoa(int(pmc))
	location.ScanSource = srcIdx

	if len(data.ContextImageDst) > 0 {
		location.ContextImage = data.ContextImageDst
	}

	// The only way we save spectrum data, compressing runs of 0's
	location.SpectrumCompression = protos.Experiment_Location_ZERO_RUN

	// Ensure we save them in a robust, predictable order. Go map ordering is not deterministic, so we don't really know what order they ended up
	// here, but we can scan for READTYPE and DETECTOR_ID and ensure we write those as alphabetical order
	// First, lets make a lookup for the combination of those values
	detectorSpectraLookup := map[string]dataConvertModels.DetectorSample{}
	detectorSpectraLookupKeys := []string{}

	for _, det := range data.DetectorSpectra {
		readType, ok := det.Meta["READTYPE"]
		if !ok {
			jobLog.Infof("WARNING: Not saving spectrum for PMC %v, READTYPE not found", pmc)
			continue
		}

		spectraReadTypeValid := readType.SValue == "BulkSum" || readType.SValue == "MaxValue" || readType.SValue == "Normal" || readType.SValue == "Dwell"
		if !spectraReadTypeValid {
			jobLog.Infof("WARNING: Not saving spectrum for PMC %v, READTYPE \"%v\" is not valid", pmc, readType)
			continue
		}

		detectorID, ok := det.Meta["DETECTOR_ID"]
		if !ok {
			jobLog.Infof("WARNING: Not saving spectrum for PMC %v, DETECTOR_ID not found", pmc)
			continue
		}

		// Form a key
		key := readType.SValue + "|" + detectorID.SValue
		if _, ok := detectorSpectraLookup[key]; ok {
			jobLog.Infof("WARNING: Found duplicate spectrum for PMC %v: DETECTOR_ID=\"%v\", READTYPE=\"%v\"", pmc, detectorID.SValue, readType.SValue)
			continue
		}

		detectorSpectraLookup[key] = det
		detectorSpectraLookupKeys = append(detectorSpectraLookupKeys, key)
	}

	sort.Strings(detectorSpectraLookupKeys)

	for _, key := range detectorSpectraLookupKeys {
		det := detectorSpectraLookup[key]

		detector := &protos.Experiment_Location_DetectorSpectrum{}
		err := s.saveSpectrumMeta(det.Meta, detector)
		if err != nil {
			return err
		}

		max := int64(0)
		for i, e := range det.Spectrum {
			if i == 0 || e > max {
				max = e
			}
		}
		detector.SpectrumMax = int32(max)
		zero := zeroRunEncode(det.Spectrum)
		detector.Spectrum = append(detector.Spectrum, zero...)
		location.Detectors = append(location.Detectors, detector)
	}

	if data.Beam != nil {
		// The order we store our IJ coordinates is defined by beamIJPMCAscending. We store the lowest PMCs coordinates in ImageI/ImageJ
		// and the rest are stored in the ContextLocations array
		if len(beamIJPMCAscending) != len(data.Beam.IJ) {
			return errors.New("PMC order for beam locations mismatched with beam IJs stored")
		}

		alignedBeamCoords := []*protos.Experiment_Location_BeamLocation_Coordinate2D{}
		for c := 1; c < len(beamIJPMCAscending); c++ {
			ij := &protos.Experiment_Location_BeamLocation_Coordinate2D{
				I: data.Beam.IJ[int32(beamIJPMCAscending[c])].I,
				J: data.Beam.IJ[int32(beamIJPMCAscending[c])].J,
			}

			alignedBeamCoords = append(alignedBeamCoords, ij)
		}

		beamLoc := &protos.Experiment_Location_BeamLocation{
			X:                data.Beam.X,
			Y:                data.Beam.Y,
			Z:                data.Beam.Z,
			ImageI:           data.Beam.IJ[int32(beamIJPMCAscending[0])].I,
			ImageJ:           data.Beam.IJ[int32(beamIJPMCAscending[0])].J,
			ContextLocations: alignedBeamCoords,
		}

		// geom_corr is optional, so only set it if it's non-0
		if data.Beam.GeomCorr > 0 {
			beamLoc.GeomCorr = data.Beam.GeomCorr
		}

		location.Beam = beamLoc
	}

	if len(data.PseudoIntensities) > 0 {
		// Save the array
		// NOTE: detector ID might not be required, for now we just save a single set
		// so we save blank
		ps := &protos.Experiment_Location_PseudoIntensityData{}
		ps.DetectorId = ""
		ps.ElementIntensities = append(ps.ElementIntensities, data.PseudoIntensities...)

		location.PseudoIntensities = append(location.PseudoIntensities, ps)
	}

	// If has housekeeping data, save it
	for colIdx, hkMeta := range data.Housekeeping {
		// Get an index
		idx, err := s.getMetaIndex(hkHeaders[colIdx], hkMeta.DataType)
		if err != nil {
			return err
		}

		saveMeta := &protos.Experiment_Location_MetaDataItem{}
		s.convertToOutputMeta(hkMeta, idx, saveMeta)
		location.Meta = append(location.Meta, saveMeta)
	}

	saveToExperiment.Locations = append(saveToExperiment.Locations, location)
	return nil
}

func savePseudoIntensityRanges(exp *protos.Experiment, items []dataConvertModels.PseudoIntensityRange) {
	for _, item := range items {
		var toSave protos.Experiment_PseudoIntensityRange //exp.PseudoIntensityRanges

		toSave.Name = item.Name
		toSave.ChannelStart = int32(item.Start)
		toSave.ChannelEnd = int32(item.End)
		exp.PseudoIntensityRanges = append(exp.PseudoIntensityRanges, &toSave)
	}
}

// Anything saving metadata to the file needs to call this. This will store the meta label & type, and return an
// index into the label/type lookup. If it already exists, it returns the existing index
func (s *PIXLISEDataSaver) getMetaIndex(label string, dataType protos.Experiment_MetaDataType) (int32, error) {
	item, ok := s.metaLookup[label]
	if ok {
		// Verify that the datatype matches
		if dataType != item.dataType {
			return -1, fmt.Errorf("Metadata \"%v\" already stored as type \"%v\", got \"%v\"", label, item.dataType, dataType)
		}

		// Just return the index to use
		return item.index, nil
	}

	// Store it as a new one
	idx := int32(len(s.metaLookup))
	s.metaLookup[label] = metaInfo{
		label:    label,
		index:    idx,
		dataType: dataType,
	}

	return idx, nil
}

func (s *PIXLISEDataSaver) saveMetaData(exp *protos.Experiment) error {
	// We have to save the meta values ordered by the index that the map values contain
	toSave := []metaInfo{}
	for _, info := range s.metaLookup {
		toSave = append(toSave, info)
	}

	sort.Slice(toSave, func(i, j int) bool { return toSave[i].index < toSave[j].index })

	// Now write them
	// Meanwhile, do a final check on them so we're sure that certain data types are saved in the right way
	asInt := []string{"PMC", "SCLK", "RTT"}
	asFloat := []string{"XPERCHAN", "OFFSET", "LIVETIME", "REALTIME", "XPOSITION", "YPOSITION", "ZPOSITION"}

	for _, item := range toSave {
		if utils.ItemInSlice(item.label, asInt) && item.dataType != protos.Experiment_MT_INT {
			return fmt.Errorf("Failed to save metadata. %v expected as int, got: %v", item.label, item.dataType)
		} else if utils.ItemInSlice(item.label, asFloat) && item.dataType != protos.Experiment_MT_FLOAT {
			return fmt.Errorf("Failed to save metadata. %v expected as float, got: %v", item.label, item.dataType)
		}

		exp.MetaLabels = append(exp.MetaLabels, item.label)
		exp.MetaTypes = append(exp.MetaTypes, item.dataType)
	}

	return nil
}
