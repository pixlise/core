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

package output

import (
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func makeSummaryFileContent(
	exp *protos.Experiment,
	prevSavedScan *protos.ScanItem,
	datasetID string,
	sourceInstrument protos.ScanInstrument,
	meta dataConvertModels.FileMetaData,
	//fileSize int,
	creationUnixTimeSec int64,
	creatorUserId string,
	jobLog logger.ILogger) *protos.ScanItem {
	contextImgCount := len(exp.AlignedContextImages) + len(exp.UnalignedContextImages) + len(exp.MatchedAlignedContextImages)
	tiffContextImgCount := 0

	// Count the number of TIFF context images so that we can quickly determine if the dataset is RGBU
	for _, img := range exp.AlignedContextImages {
		if strings.HasSuffix(img.Image, ".tif") {
			tiffContextImgCount++
		}
	}
	for _, img := range exp.UnalignedContextImages {
		if strings.HasSuffix(img, ".tif") {
			tiffContextImgCount++
		}
	}
	for _, img := range exp.MatchedAlignedContextImages {
		if strings.HasSuffix(img.Image, ".tif") {
			tiffContextImgCount++
		}
	}

	dataTypes := []*protos.ScanItem_ScanTypeCount{}

	// Add up what we have
	if contextImgCount > 0 {
		dataTypes = append(dataTypes, &protos.ScanItem_ScanTypeCount{DataType: protos.ScanDataType_SD_IMAGE, Count: uint32(contextImgCount)})
	}
	if tiffContextImgCount > 0 {
		dataTypes = append(dataTypes, &protos.ScanItem_ScanTypeCount{DataType: protos.ScanDataType_SD_RGBU, Count: uint32(tiffContextImgCount)})
	}
	if exp.NormalSpectra > 0 {
		dataTypes = append(dataTypes, &protos.ScanItem_ScanTypeCount{DataType: protos.ScanDataType_SD_XRF, Count: uint32(exp.NormalSpectra)})
	}

	saveMeta := map[string]string{
		"TargetId": exp.TargetId,
		"SiteId":   fmt.Sprintf("%v", meta.SiteID),
		"DriveId":  fmt.Sprintf("%v", meta.DriveID),
		"Target":   meta.Target,
		"Site":     meta.Site,
		"Sol":      meta.SOL,
		"RTT":      meta.RTT,
		"SCLK":     fmt.Sprintf("%v", meta.SCLK),
	}

	contentCounts := map[string]int32{
		"NormalSpectra":     int32(exp.NormalSpectra),
		"DwellSpectra":      int32(exp.DwellSpectra),
		"BulkSpectra":       int32(exp.BulkSpectra),
		"MaxSpectra":        int32(exp.MaxSpectra),
		"PseudoIntensities": int32(exp.PseudoIntensities),
	}

	// Values that went missing during the v4 rebuild:
	// LocationCount:       len(exp.Locations),
	// DataFileSize:        fileSize,
	// But these didn't seem important anyway, UI shows spectra counts not locations, and file
	// size shouldn't be relevant because new UI doesn't download the file as one thing anyway, only
	// what it's displaying

	s := &protos.ScanItem{
		Id:                         datasetID,
		Title:                      meta.Title,
		Description:                "",
		DataTypes:                  dataTypes,
		Instrument:                 sourceInstrument,
		InstrumentConfig:           exp.DetectorConfig,
		TimestampUnixSec:           uint32(creationUnixTimeSec),
		Meta:                       saveMeta,
		ContentCounts:              contentCounts,
		CreatorUserId:              creatorUserId,
		PreviousImportTimesUnixSec: []uint32{},
	}

	// If we've got a previously stored ScanItem, we are updating it, so read its time stamp into the array of previous time stamps
	isComplete := exp.PseudoIntensities > 0 && exp.NormalSpectra == exp.PseudoIntensities*2

	if prevSavedScan != nil {
		// Build the list of previous import times
		// Preserve the previous list...
		s.PreviousImportTimesUnixSec = append(s.PreviousImportTimesUnixSec, prevSavedScan.PreviousImportTimesUnixSec...)

		// Add new time at the end
		s.PreviousImportTimesUnixSec = append(s.PreviousImportTimesUnixSec, prevSavedScan.TimestampUnixSec)
		jobLog.Infof(" Added new previous time stamp entry: %v, total previous timestamps: %v", prevSavedScan.TimestampUnixSec, len(s.PreviousImportTimesUnixSec))
	}

	// Save a time stamp for completion
	if isComplete {
		jobLog.Infof(" Detected completed dataset...")

		// If previous scan was also complete, DON'T update the time stamp, just preserve it
		if prevSavedScan != nil && prevSavedScan.CompleteTimeStampUnixSec > 0 {
			s.CompleteTimeStampUnixSec = prevSavedScan.CompleteTimeStampUnixSec
			jobLog.Infof(" Preserved previous CompleteTimeStampUnixSec of %v", s.CompleteTimeStampUnixSec)
		} else {
			// We must've just completed now, so save the time
			s.CompleteTimeStampUnixSec = uint32(creationUnixTimeSec)
			jobLog.Infof(" Setting CompleteTimeStampUnixSec=%v", creationUnixTimeSec)
		}
	}

	// Preserve user-editable fields
	if prevSavedScan != nil {
		s.Tags = prevSavedScan.Tags
		s.Description = prevSavedScan.Description

		descSnippet := s.Description
		if len(descSnippet) > 30 {
			descSnippet = descSnippet[0:30] + "..."
		}
		jobLog.Infof(" Preserved previous description=\"%v\"...", descSnippet)

		if len(prevSavedScan.Description) > 0 {
			// User has entered a description so perserve the title too
			s.Title = prevSavedScan.Title
			jobLog.Infof(" Preserved previous title=\"%v\"", s.Title)
		}
	}

	return s
}
