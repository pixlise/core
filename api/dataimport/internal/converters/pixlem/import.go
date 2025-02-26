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
	"time"

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

	creatorId, err := readCreator(filepath.Join(importPath, "creator.json"), &fs)
	if err != nil {
		return nil, "", err
	}
	/*
		// Extract RTTs from each beam location file name and import a dataset for each RTT
		zipName, err := extractZipName(append(msaFiles, imageFiles...))
		if err != nil {
			return nil, "", err
		}
	*/

	// Assume zip name is the same as dataset id, ie the directory where the files referenced in the above text files exist
	zipName := datasetIDExpected

	// Reject silly imports - if the scan has multiple sets of beams for different RTTs, we stop here
	if len(beams) > 1 {
		rtts := []string{}
		for _, beamName := range beams {
			rttStr := beamName[len(beamLocPrefix) : len(beamName)-4]
			rtts = append(rtts, rttStr)
		}

		err = fmt.Errorf("Multiple RTTs found in SDF data. Stopping because we don't know which to import: %v", strings.Join(rtts, ","))
		log.Infof("%v", err)
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
		/*if datasetIDExpected != rttStr && !strings.HasSuffix(datasetIDExpected, rttHex) {
			log.Infof("Skipping beam location file: %v, RTT doesn't match expected: %v", beamName, datasetIDExpected)
			continue
		}*/

		if len(rttHex) < 8 {
			rttHex = "0" + rttHex
		}
		rttHex = rttHex + "_"

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
		// HK file should be here too...
		hkPath := filepath.Join(importPath, "HK-"+rttStr+".csv")
		data, err := importEMData(creatorId, rttStr, beamPath, hkPath, imageList, bulkMaxList, msaList, &fs, log)
		if err != nil {
			log.Errorf("Import failed for %v: %v", beamName, err)
			continue
		}

		log.Infof("Imported scan with RTT: %v", rtt)
		data.DatasetID += "_em" // To ensure we don't overwrite real datasets

		// Set the title if we need one
		data.Meta.Title = datasetIDExpected

		// NOTE: PIXL EM import - we clear everything before importing so we don't end up with eg images from a bad previous import
		data.ClearBeforeSave = true
		return data, filepath.Join(importPath, zipName /*, zipName*/), nil
	}

	// If we got here, nothing was imported
	return nil, "", fmt.Errorf("Expected RTT %v was not found in uploaded data", datasetIDExpected)
}

func readCreator(creatorPath string, fs fileaccess.FileAccess) (string, error) {
	var creator = protos.UserInfo{}
	err := fs.ReadJSON(creatorPath, "", &creator, false)
	if err != nil {
		return "", nil
	}

	if len(creator.Id) <= 0 {
		return "", fmt.Errorf("Failed to read creator id from %v", creatorPath)
	}
	return creator.Id, nil
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
func importEMData(creatorId string, rtt string, beamLocPath string, hkPath string, imagePathList []string, bulkMaxList []string, msaList []string, fs fileaccess.FileAccess, logger logger.ILogger) (*dataConvertModels.OutputData, error) {
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
	beamLookup, ijPMCs, err := dataImportHelpers.ReadBeamLocationsFile(beamLocPath, true, minContextPMC, []string{"drift_x", "drift_y", "drift_z"}, logger)
	if err != nil {
		return nil, err
	}

	// Remove any images which don't have beam locations
	for pmc, img := range contextImgsPerPMC {
		if !utils.ItemInSlice(pmc, ijPMCs) {
			logger.Infof("Excluding image due to not having beam locations: %v", img)
			delete(contextImgsPerPMC, pmc)
		}
	}

	hkData, err := importerutils.ReadHousekeepingFile(hkPath, 0, logger)
	if err != nil {
		return nil, err
	}

	// We don't have everything a full FM dataset would have...
	var pseudoIntensityData dataConvertModels.PseudoIntensities
	var pseudoIntensityRanges []dataConvertModels.PseudoIntensityRange
	var matchedAlignedImages []dataConvertModels.MatchedAlignedImageMeta

	site := "000"
	drive := "0000"
	product := "???"

	// Use current date encoded as a test sol
	sol := timeToTestSol(time.Now())

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

	for len(rtt) <= 8 {
		rtt = "0" + rtt
	}

	fakeFileName := fmt.Sprintf("%v__%v_%v_000%v_N%v%v%v_______%v%v.CSV", ftype, sol, sclk, product, site, drive, rtt, producer, version)
	housekeepingFileNameMeta, err := gdsfilename.ParseFileName(fakeFileName)
	if err != nil {
		return nil, err
	}

	outData, err := importerutils.MakeFMDatasetOutput(
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
		"PIXL-EM-E2E", // Specifying this and the above will allow importer to work, we want to block out weird EM data from FM pipeline normally
		uint32(3),
		logger,
	)

	if outData != nil {
		outData.CreatorUserId = creatorId
	}

	return outData, err
}

/*
The Primary timestamp of coarser granularity than the Secondary timestamp (documented later).  Value type is based on either of four scenarios:
Flight Cruise
Year-DOY (4 alphanumeric) - This field stores two metadata items in the order:
a)    One alpha character in range “A-Z” to designate Earth Year portion of the UTC-like time value, representing Years 2017 to 2042
b)    Three integers in range “001-365” representing Day-of-Year (DOY)
*/
func timeToTestSol(t time.Time) string {
	// A=2017, 'A' is 65 ascii
	var result string

	yearSinceEpoch := t.Year() - 2017
	if yearSinceEpoch < 0 || yearSinceEpoch > 25 {
		return "????"
	}

	var asciiDate = rune('A' + yearSinceEpoch)
	result += string(asciiDate)
	result += fmt.Sprintf("%03d", t.YearDay())
	return result
}
