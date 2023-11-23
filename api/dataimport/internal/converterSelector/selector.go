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

package converterSelector

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dataimport/internal/converters/converter"
	"github.com/pixlise/core/v3/api/dataimport/internal/converters/jplbreadboard"
	"github.com/pixlise/core/v3/api/dataimport/internal/converters/pixlem"
	"github.com/pixlise/core/v3/api/dataimport/internal/converters/pixlfm"
	"github.com/pixlise/core/v3/api/dataimport/internal/converters/soff"
	dataimportModel "github.com/pixlise/core/v3/api/dataimport/models"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
)

// SelectDataConverter - Looks in specified path and determines what importer to use.
func SelectDataConverter(localFS fileaccess.FileAccess, remoteFS fileaccess.FileAccess, datasetBucket string, importPath string, log logger.ILogger) (converter.DataConverter, error) {
	items, err := localFS.ListObjects(importPath, "")
	if err != nil {
		return nil, errors.New("Failed to list files in import path when determining dataset type")
	}

	log.Infof("SelectDataConverter: Path contains %v files...", len(items))
	/*for c, item := range items {
		log.Infof("  %v. %v", c+1, item)
	}*/
	/*
		// Check if it's a combined dataset
		combinedFiles, _ /*imageFileNames* /, _ /*combinedFile1Meta* /, _ /*combinedFile2Meta* /, err := combined.GetCombinedBeamFiles(importPath, log)
		if len(combinedFiles) > 0 && err == nil {
			// It's a combined dataset, interpret it as such
			return combined.MakeCombinedDatasetImporter(SelectDataConverter, remoteFS, datasetBucket), nil
		}
	*/
	// Check if it's a PIXL FM style dataset
	pathType, err := pixlfm.DetectPIXLFMStructure(importPath)
	if len(pathType) > 0 && err == nil {
		// We know it's a PIXL FM type dataset... it'll later be determined which one
		return pixlfm.PIXLFM{}, nil
	}

	// Check if it's SOFF
	soffFile, err := soff.GetSOFFDescriptionFile(importPath)
	if err != nil {
		return nil, err
	}

	if len(soffFile) > 0 {
		return &soff.SOFFImport{}, nil
	}

	// Try to read a detector.json - manually uploaded datasets will contain this to direct our operation...
	detPath := filepath.Join(importPath, "detector.json")
	var detectorFile dataimportModel.DetectorChoice
	err = localFS.ReadJSON(detPath, "", &detectorFile, false)
	if err == nil {
		// We found it, work out based on what's in there
		if strings.HasSuffix(detectorFile.Detector, "-breadboard") {
			return jplbreadboard.MSATestData{}, nil
		} else if detectorFile.Detector == "pixl-em" {
			return pixlem.PIXLEM{}, nil
		}
	} else {
		return nil, errors.New("Failed to open detector.json when determining dataset type")
	}

	// Unknown
	return nil, errors.New("Failed to determine dataset type to import.")
}
