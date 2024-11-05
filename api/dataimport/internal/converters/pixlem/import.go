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

package pixlem

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dataimport/internal/converters/jplbreadboard"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/dataimport/internal/importerutils"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// These are EM files, which we expect to be in the same format as FM but because they come from
// manual uploads, we expect the actual files to be in a sub dir. We also override the group/detector
// when importing these

type PIXLEM struct {
}

func (p PIXLEM) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	log.Infof("PIXL EM Import started for path: %v", importPath)

	fs := fileaccess.FSAccess{}

	// PIXL EM has evolved over time from being a FM-like dataset, or a bunch of MSA files, to us importing the SDF-Peek output format. We run the beam geometry tool ourselves
	// when the dataset is imported, and here we're expecting the sdf peek output zip file, and the processed files we generate to make these ready to import.
	// Check expected files exist

	msaFiles, err := utils.ReadFileLines(filepath.Join(importPath, "msas.txt"))
	if err != nil {
		log.Errorf("%v", err)
		return nil, "", err
	}

	imageFiles, err := utils.ReadFileLines(filepath.Join(importPath, "images.txt"))
	if err != nil {
		log.Errorf("%v", err)
		return nil, "", err
	}

	// Read all beam location files
	beamLocPrefix := "beamLocation-"
	beams, err := fs.ListObjects(importPath, beamLocPrefix)

	if err != nil || len(beams) <= 0 {
		if err == nil {
			err = fmt.Errorf("Failed to find beam location file(s)")
		}

		log.Errorf("%v", err)
		return nil, "", err
	}

	// Extract RTTs from each beam location file name and import a dataset for each RTT
	zipName, err := extractZipName(append(msaFiles, imageFiles...))
	if err != nil {
		return nil, "", err
	}

	for _, beamName := range beams {
		log.Infof("Reading beam location file: %v", beamName)

		rttStr := beamName[len(beamLocPrefix) : len(beamName)-4]
		rtt, err := strconv.Atoi(rttStr)
		if err != nil {
			err = fmt.Errorf("Failed to read rtt from file name: %v. Error: %v", beamName, err)
			log.Infof("%v", err)
			return nil, "", err
		}

		// User may have specified RTT as hex or int, when we're checking which to import, check both ways
		rttHex := fmt.Sprintf("%X", rtt)
		if datasetIDExpected != rttStr && !strings.HasSuffix(datasetIDExpected, rttHex) {
			log.Infof("Skipping beam location file: %v, RTT doesn't match expected: %v", beamName, datasetIDExpected)
			continue
		}

		if len(rttHex) < 8 {
			rttHex = "0" + rttHex
		}
		rttHex = "_" + rttHex + "_"

		imageList := []string{}
		for _, img := range imageFiles {
			// Expecting image file names of the form: 0720239657_0C6E0205_000002.jpg
			// The second part is the RTT, so we convert our RTT to hex to compare

			if strings.Contains(img, rttHex) {
				fullPath := filepath.Join(importPath, zipName, img)
				imageList = append(imageList, fullPath)
			}
		}

		msaList := []string{}
		bulkMaxList := []string{}
		for _, msa := range msaFiles {
			// Expecting image file names of the form: 0720239657_0C6E0205_000002.jpg
			// The second part is the RTT, so we convert our RTT to hex to compare

			if strings.Contains(msa, rttHex) {

				fullPath := filepath.Join(importPath, zipName, msa)
				fileName := filepath.Base(msa)

				if strings.HasPrefix(fileName, "BulkSum_") || strings.HasPrefix(fileName, "MaxValue_") {
					bulkMaxList = append(bulkMaxList, fullPath)
				} else {
					msaList = append(msaList, fullPath)
				}
			}
		}

		beamPath := filepath.Join(importPath, beamName)
		data, err := importEMData(rttStr, beamPath, imageList, bulkMaxList, msaList, &fs, log)
		if err != nil {
			log.Errorf("Import failed for %v: %v", beamName, err)
			continue
		}

		log.Infof("Imported scan with RTT: %v", rtt)
		return data, filepath.Join(importPath, zipName, zipName), nil
	}

	// If we got here, nothing was imported
	return nil, "", fmt.Errorf("Expected RTT %v was not found in uploaded data", datasetIDExpected)
}

func extractZipName(files []string) (string, error) {
	zipName := ""
	pathSep := string(os.PathSeparator)

	// Unfortunately when unzipped, the zip file name ends up in the path again... so we have to add it here. Once we have the zip name,
	// verify all files in the list start with it
	for _, f := range files {
		if len(zipName) <= 0 {
			pos := strings.Index(f, pathSep)
			if pos == -1 {
				pos = strings.Index(f, fmt.Sprintf("%v", "/"))
			}
			if pos > 0 {
				zipName = f[0:pos]
			} else {
				return "", fmt.Errorf("Failed to read path root for PIXL EM importable files from: %v", f)
			}
		} else {
			if !strings.HasPrefix(f, zipName) {
				return "", fmt.Errorf("Error while reading importable files for PIXL EM: Expected path %v to start with %v", f, zipName)
			}
		}
	}

	return zipName, nil
}
func importEMData(rtt string, beamLocPath string, imagePathList []string, bulkMaxList []string, msaList []string, fs fileaccess.FileAccess, logger logger.ILogger) (*dataConvertModels.OutputData, error) {
	// Read MSAs
	locSpectraLookup, err := jplbreadboard.MakeSpectraLookup("", msaList, true, false, "", false, logger)
	if err != nil {
		return nil, err
	}

	bulkMaxSpectraLookup, err := jplbreadboard.MakeSpectraLookup("", bulkMaxList, true, false, "", false, logger)
	if err != nil {
		return nil, err
	}

	// Read Images
	contextImgsPerPMC := importerutils.GetContextImagesPerPMCFromListing(imagePathList, logger)
	minContextPMC := importerutils.GetMinimumContextPMC(contextImgsPerPMC)

	// Read Beams
	beamLookup, err := dataImportHelpers.ReadBeamLocationsFile(beamLocPath, true, minContextPMC, []string{"drift_x", "drift_y", "drift_z"}, logger)
	if err != nil {
		return nil, err
	}

	// We don't have everything a full FM dataset would have...
	var hkData dataConvertModels.HousekeepingData
	var pseudoIntensityData dataConvertModels.PseudoIntensities
	var pseudoIntensityRanges []dataConvertModels.PseudoIntensityRange
	var matchedAlignedImages []dataConvertModels.MatchedAlignedImageMeta

	site := "000"
	drive := "0000"
	product := "???"
	sol := "D000"
	ftype := "??" // PE
	producer := "J"
	version := "01"

	// Grab the SCLK from the lowest PMC image
	minPMC := int32(-1)
	minFileName := ""
	for pmc, img := range contextImgsPerPMC {
		if minPMC < 0 || pmc < minPMC {
			minPMC = pmc
			minFileName = img
		}
	}

	if len(minFileName) <= 0 {
		return nil, fmt.Errorf("Failed to find SCLK to use")
	}

	parts := strings.Split(minFileName, "_")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unexpected image name format: %v", minFileName)
	}

	sclk := parts[0]

	fakeFileName := fmt.Sprintf("%v__%v_%v_000%v_N%v%v%v_______%v%v.CSV", ftype, sol, sclk, product, site, drive, rtt, producer, version)
	housekeepingFileNameMeta, err := gdsfilename.ParseFileName(fakeFileName)
	if err != nil {
		return nil, err
	}

	return importerutils.MakeFMDatasetOutput(
		beamLookup,
		hkData,
		locSpectraLookup,
		bulkMaxSpectraLookup,
		contextImgsPerPMC,
		pseudoIntensityData,
		pseudoIntensityRanges,
		matchedAlignedImages,
		[]dataConvertModels.ImageMeta{},
		[]dataConvertModels.ImageMeta{},
		"",
		housekeepingFileNameMeta,
		rtt,
		protos.ScanInstrument_PIXL_EM,
		"",
		uint32(3),
		logger,
	)
}
