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

package pixlfm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	gdsfilename "github.com/pixlise/core/v2/data-import/gds-filename"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	"github.com/pixlise/core/v2/data-import/internal/importerutils"
)

// These are structured differently to test data-sets in our test-data repo. One can be found in test-data called FM-cal-target-crosshair
// Beam location (now 1 per context image PMC):
// drift_corr_x_ray_beam_location/PE__D077T0637741109_000RXL_N001003600098356100660__J01.CSV
//
// MSAs are now all in 1 big CSV file:
// localized_full_spectra/PS__D077T0637741109_000RFS_N001003600098356100640__J01.CSV
//
// Bulk sum spectra MSA (A & B):
// bulk_histogram_inputs/PS__D077T0637746318_000RBS_N001003600098356103760__J01.MSA
//
// Max value spectra MSA (A & B):
// bulk_histogram_inputs/PS__D077T0637746319_000RMS_N001003600098356103760__J01.MSA
//
// Context images are now PNGs, thrown in a directory:
// context_images/*.PNG
//
// Housekeeping details:
// spatial_inputs/PE__D077T0637741109_000RSI_N001003600098356100660__J01.CSV
// also in the root there is PE__D077T0637741109_000E08_N001003600098356100640__J01.CSV with more columns (?)
//
// Pseudo-intensity data:
// pseudointensity_maps/PS__D077T0637741109_000RPM_N001003600098356100640__J01.CSV
//
// Metadata about the dataset (can be derived from file names, see in docs/PIXL_filename.docx)

type PIXLFM struct {
}

type fileStructure struct {
	directories       []string
	extensions        string
	expectedFileCount int
}

var log logger.ILogger

func (p PIXLFM) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	log = jobLog
	localFS := &fileaccess.FSAccess{}

	beamDir := fileStructure{}
	spectraDir := fileStructure{}
	bulkSpectraDir := fileStructure{}
	contexImgDir := fileStructure{}
	housekeepingDir := fileStructure{}
	pseudoIntensityDir := fileStructure{}
	rgbuImgDir := fileStructure{}
	discoImgDir := fileStructure{}

	pathType, err := DetectPIXLFMStructure(importPath)
	if err != nil {
		return nil, "", err
	}

	if pathType == "DataDrive" {
		// This is the official way we receive PIXL FM data from Mars
		// We expect these directories to exist...
		beamDir = fileStructure{[]string{"RXL"}, "csv", 1} // The BGT file contains the positions of the commanded X-ray shots and the actual X-ray shots (in x,y,z). We need RXL (containing image i/js)
		spectraDir = fileStructure{[]string{"RFS"}, "csv", 1}
		bulkSpectraDir = fileStructure{[]string{"RBS", "RMS"}, "msa", 2}
		contexImgDir = fileStructure{[]string{"RCM"}, "tif", -1}
		housekeepingDir = fileStructure{[]string{"RSI"}, "csv", 1}
		pseudoIntensityDir = fileStructure{[]string{"RPM"}, "csv", 1} //EPN
		rgbuImgDir = fileStructure{[]string{"RGBU"}, "tif", -1}
		discoImgDir = fileStructure{[]string{"DISCO"}, "png", -1}
		// These all contain pseudointensity data, but they're just PNG maps so we ignore them:
		// "PAL", "PAS", "PBA", "PCA", "PCB", "PCE", "PCF", "PCL", "PCR", "PCS", "PFE", "PGE", "PKC", "PKX", "PMF", "PMG", "PMN", "PNA", "PNI", "PPX", "PSB", "PSC", "PSI", "PSR", "PST", "PSX", "PSZ", "PTF", "PTI", "PYX", "PZN", "PZR"
	} else if pathType == "PreDataDriveFormat" {
		// This was an early version of the dir structure of PIXL FM data. We had a few datasets from the EM given to us
		// in this form, with these subdirs
		beamDir = fileStructure{[]string{"drift_corr_x_ray_beam_location"}, "csv", 2}
		spectraDir = fileStructure{[]string{"localized_full_spectra"}, "csv", 1}
		bulkSpectraDir = fileStructure{[]string{"bulk_histogram_inputs"}, "msa", 2}
		contexImgDir = fileStructure{[]string{"image_mark_up"}, "tif", -1}
		housekeepingDir = fileStructure{[]string{"spatial_inputs"}, "csv", 1}
		pseudoIntensityDir = fileStructure{[]string{"pseudointensity_maps"}, "csv", 1}
	}

	// Allocate everything needed (empty, if we find & load stuff, great, but we still need the data struct for the last step)
	beamLookup := dataConvertModels.BeamLocationByPMC{}
	hkData := dataConvertModels.HousekeepingData{}
	locSpectraLookup := dataConvertModels.DetectorSampleByPMC{}
	bulkMaxSpectraLookup := dataConvertModels.DetectorSampleByPMC{}
	contextImgsPerPMC := map[int32]string{}
	pseudoIntensityData := dataConvertModels.PseudoIntensities{}
	pseudoIntensityRanges := []dataConvertModels.PseudoIntensityRange{}
	rgbuImages := []dataConvertModels.ImageMeta{}
	discoImages := []dataConvertModels.ImageMeta{}
	whiteDiscoImage := ""

	//spectraFileNameMeta := FileNameMeta{}
	housekeepingFileNameMeta := gdsfilename.FileNameMeta{}

	// Get a path for each file
	//pathsToRead := [][]string{beamDir, spectraDir, bulkSpectraDir, contexImgDir, housekeepingDir, pseudoIntensityDir}
	pathsToRead := map[string]fileStructure{"beamDir": beamDir, "spectraDir": spectraDir, "bulkSpectraDir": bulkSpectraDir, "contextImgDir": contexImgDir, "housekeepingDir": housekeepingDir, "pseudoIntensityDir": pseudoIntensityDir, "rgbuImgDir": rgbuImgDir, "discoImgDir": discoImgDir}
	for dirType, subdir := range pathsToRead {
		pathToSubdir := importPath
		log.Infof("READING %v from \"%v\", subdirs: \"%v\"...", dirType, pathToSubdir, strings.Join(subdir.directories, ","))

		extUpper := strings.ToUpper(subdir.extensions)
		if extUpper[0:1] != "." {
			extUpper = "." + extUpper
		}

		var allFoundPaths []string
		for _, d := range subdir.directories {
			paths, err := localFS.ListObjects(path.Join(pathToSubdir, d), "")

			for _, p := range paths {
				if strings.HasSuffix(strings.ToUpper(path.Ext(p)), extUpper) {
					allFoundPaths = append(allFoundPaths, path.Join(d, p))
				}
			}

			if err != nil {
				log.Infof("  WARNING: Failed to read dir \"%v\". SKIPPING. Error was: \"%v\"", pathToSubdir, err)
				err = nil
			} else if len(paths) <= 0 {
				log.Infof("  WARNING: No files read from dir \"%v\". SKIPPING", pathToSubdir)
				err = nil
			}
		}

		latestVersionFoundPaths := gdsfilename.GetLatestFileVersions(allFoundPaths, log)
		numFoundPaths := len(latestVersionFoundPaths)

		if numFoundPaths < len(allFoundPaths) {
			// Print out all the files ignored due to being old versions...
			for _, allFoundItem := range allFoundPaths {
				if _, ok := latestVersionFoundPaths[allFoundItem]; !ok {
					log.Infof("  IGNORED: \"%v\", due to being older version", allFoundItem)
				}
			}
		}

		// Check we got the right amount of files
		if subdir.expectedFileCount > 0 {
			expVsFoundPathsCount := subdir.expectedFileCount - numFoundPaths
			if expVsFoundPathsCount > 0 {
				log.Infof("  WARNING: Not enough %v files found in %v, only found %v!", subdir.extensions, strings.Join(subdir.directories, ","), numFoundPaths)
			} else if expVsFoundPathsCount < 0 {
				log.Infof("  WARNING: Unexpected %v file count %v in %v. Check that we read the right one!", subdir.extensions, numFoundPaths, strings.Join(subdir.directories, ","))
			}
		}

		// To aid debugging, print out the file names we DO consider current version, and will present to be processed
		for file := range latestVersionFoundPaths {
			log.Infof("  FOUND: \"%v\"", file)
		}

		// OK we have the paths, now read this type
		switch dirType {
		case "beamDir":
			for file, beamCsvMeta := range latestVersionFoundPaths {
				if beamCsvMeta.ProdType == "RXL" {
					// If files don't conform, don't read...
					beamLookup, err = importerutils.ReadBeamLocationsFile(path.Join(pathToSubdir, file), true, 1, log)
					if err != nil {
						return nil, "", err
					} else {
						// Found it, why keep looping?
						break
					}
				}
			}
			// Check that we did end up with beam data...
			if len(beamLookup) <= 0 {
				//return nil, "", errors.New("Failed to find beam location CSV")
				log.Infof("No beam location found for this dataset. Continuing in case it's a \"disco\" dataset")
			}
		case "spectraDir":
			file := ""
			for file /*, spectraFileNameMeta*/ = range latestVersionFoundPaths {
				locSpectraLookup, err = readSpectraCSV(path.Join(pathToSubdir, file), log)
				if err != nil {
					return nil, "", err
				}
				// Stop after first file
				break
			}
		case "bulkSpectraDir":
			filesOnly := []string{}

			for file := range latestVersionFoundPaths {
				filesOnly = append(filesOnly, file)
			}

			if len(filesOnly) > 0 {
				bulkMaxSpectraLookup, err = readBulkMaxSpectra(pathToSubdir, filesOnly, log)
				if err != nil {
					return nil, "", err
				}
			}
		case "contextImgDir":
			for file, meta := range latestVersionFoundPaths {
				// Markup dir contains PNG (with markup) but also TIFF (with 1st layer being image, 2nd layer being markup) so
				// here we are reading only the TIFs so we can separate out the first layer from the TIFF and save as PNG for
				// our own web purposes

				pmc, err := meta.PMC()
				if err != nil {
					log.Infof("  WARNING: No PMC in context image file name: \"%v\"", file)
				} else {
					contextImgsPerPMC[pmc] = file
				}
			}
		case "housekeepingDir":
			file := ""
			for file, housekeepingFileNameMeta = range latestVersionFoundPaths {
				hkData, err = importerutils.ReadHousekeepingFile(path.Join(pathToSubdir, file), 1, log)
				if err != nil {
					return nil, "", err
				}
				// Stop after first file
				break
			}
		case "pseudoIntensityDir":
			for file := range latestVersionFoundPaths {
				// If we have pseudointensity data, make sure a range file was specified
				if len(pseudoIntensityRangesPath) <= 0 {
					return nil, "", errors.New("Dataset contains pseudo-intensity CSV file, but no pseudo-intensity ranges file specified")
				}

				pseudoIntensityRanges, err = importerutils.ReadPseudoIntensityRangesFile(pseudoIntensityRangesPath, log)
				if err != nil {
					return nil, "", err
				}

				pseudoIntensityData, err = importerutils.ReadPseudoIntensityFile(path.Join(pathToSubdir, file), false, log)
				if err != nil {
					return nil, "", err
				}
				// Stop after first file
				break
			}
		case "rgbuImgDir":
			for file, meta := range latestVersionFoundPaths {

				info := dataConvertModels.ImageMeta{FileName: file}

				pmc, err := meta.PMC()
				if err != nil {
					//return result, nil
					log.Infof("RGBU image file name \"%v\" did not contain PMC: %v", file, err)
				}

				info.PMC = pmc
				info.LEDs = meta.ColourFilter
				info.ProdType = meta.ProdType

				// RGBU data sits in its own directory, TIFF files which must be output unchanged
				rgbuImages = append(rgbuImages, info)
			}
		case "discoImgDir":
			for file, meta := range latestVersionFoundPaths {
				discoInfo := dataConvertModels.ImageMeta{FileName: file}

				pmc, err := meta.PMC()
				if err != nil {
					//return result, nil
					log.Infof("DISCO image file name \"%v\" did not contain PMC: %v", file, err)
				}

				discoInfo.PMC = pmc
				discoInfo.LEDs = meta.ColourFilter

				if meta.ColourFilter == "W" {
					whiteDiscoImage = file
				}

				discoImages = append(discoImages, discoInfo)
			}
		default:
			return nil, "", errors.New("Error searching paths")
		}
	}

	// Now that all have been read, combine the bulk/max spectra into our lookup
	for pmc := range bulkMaxSpectraLookup {
		locSpectraLookup[pmc] = append(locSpectraLookup[pmc], bulkMaxSpectraLookup[pmc]...)
	}

	importerutils.LogIfMoreFoundMSA(locSpectraLookup, "MSA/spectrum", 2, jobLog)
	// Not really relevant, what would we show? It's a list of meta, how many is too many?
	//importer.LogIfMoreFoundHousekeeping(hkData, "Housekeeping", 1)

	matchedAlignedImages, err := importerutils.ReadMatchedImages(path.Join(importPath, "MATCHED"), beamLookup, log, localFS)

	if err != nil {
		return nil, "", err
	}

	// Build internal representation of the data that we can pass to the output code
	// We now read the metadata from the housekeeping file name, as it's the only file we expect to always exist!
	meta, err := makeDatasetFileMeta(housekeepingFileNameMeta, log)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to parse file metadata: %v", err)
	}

	// Expecting RTT to be read
	if len(meta.RTT) <= 0 {
		return nil, "", errors.New("Failed to determine dataset RTT")
	}

	// Ensure it matches what we're expecting
	// We allow for missing 0's at the start because for a while we imported RTTs as ints, so older dataset RTTs
	// were coming in as eg 76481028, while we now read them as 076481028
	if meta.RTT != datasetIDExpected && meta.RTT != "0"+datasetIDExpected {
		return nil, "", fmt.Errorf("Expected dataset ID %v, read %v", datasetIDExpected, meta.RTT)
	}

	detectorConfig := "PIXL"
	group := "PIXL-FM"

	// Depending on the SOL we may override the group and detector, as we have some test datasets that came
	// from the EM and have special characters as first part of SOL
	if len(meta.SOL) > 0 && (meta.SOL[0] == 'D' || meta.SOL[0] == 'C') {
		detectorConfig = "PIXL-EM-E2E"
		group = "PIXL-EM"
	}

	data := &dataConvertModels.OutputData{
		DatasetID:      meta.RTT,
		Group:          group,
		Meta:           meta,
		DetectorConfig: detectorConfig,
		//BulkQuantFile: "", <-- no bulk quant for tactical... TODO: what do we do here, does a scientist do it and we publish it back through PDS?
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		RGBUImages:           rgbuImages,
		DISCOImages:          discoImages,
		MatchedAlignedImages: matchedAlignedImages,
	}

	data.SetPMCData(beamLookup, hkData, locSpectraLookup, contextImgsPerPMC, pseudoIntensityData)
	if data.DefaultContextImage != "" {
		log.Infof("Setting context image to: " + data.DefaultContextImage)
	}
	// If we have no default context image at this point, see if we can use one of the DISCO images
	if len(data.DefaultContextImage) <= 0 && len(discoImages) > 0 {
		if len(whiteDiscoImage) > 0 {
			data.DefaultContextImage = whiteDiscoImage
			log.Infof("White Disco Image Found. Setting Context Image: " + whiteDiscoImage)
		} else {
			log.Infof("Setting Context to first Disco Image: " + discoImages[0].FileName)
			data.DefaultContextImage = discoImages[0].FileName
		}
	}

	return data, importPath, nil
}

func DetectPIXLFMStructure(importPath string) (string, error) {
	c, _ := ioutil.ReadDir(importPath)
	for _, entry := range c {
		if entry.IsDir() && entry.Name() == "drift_corr_x_ray_beam_location" {
			return "PreDataDriveFormat", nil
		}

		// All datasets (even ones without PIXL scans) have housekeeping files
		if entry.IsDir() && entry.Name() == "RSI" {
			return "DataDrive", nil
		}
	}
	return "", errors.New("unknown data source type")
}
func validatePaths(importPath string, validpaths []string) error {
	validated := []string{}
	c, _ := ioutil.ReadDir(importPath)
	for _, entry := range c {
		for _, p := range validpaths {
			if p == entry.Name() && entry.IsDir() {
				validated = append(validated, p)
			}
		}
	}

	if len(validated) != len(importPath) {
		return errors.New("not all directories located")
	}
	return nil
}

func makeDatasetFileMeta(fMeta gdsfilename.FileNameMeta, jobLog logger.ILogger) (dataConvertModels.FileMetaData, error) {
	result := dataConvertModels.FileMetaData{}

	sol, err := fMeta.SOL()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SOL: %v", err)
	}

	rtt, err := fMeta.RTT()
	if err != nil {
		return result, nil
	}

	sclk, err := fMeta.SCLK()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SCLK: %v", err)
	}

	site, err := fMeta.Site()
	if err != nil {
		return result, nil
	}

	drive, err := fMeta.Drive()
	if err != nil {
		return result, nil
	}

	result.SOL = sol
	result.RTT = rtt
	result.SCLK = sclk
	result.SiteID = site
	result.DriveID = drive
	result.TargetID = "?"
	result.Title = rtt
	return result, nil
}

func readBulkMaxSpectra(inPath string, files []string, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	result := dataConvertModels.DetectorSampleByPMC{}

	for _, file := range files {
		// Parse metadata for file
		csvMeta, err := gdsfilename.ParseFileName(file)
		if err != nil {
			return nil, err
		}

		// Make sure it's one of the products we're expecting
		readType := ""
		if csvMeta.ProdType == "RBS" {
			readType = "BulkSum"
		} else if csvMeta.ProdType == "RMS" {
			readType = "MaxValue"
		} else {
			return nil, fmt.Errorf("Unexpected bulk/max MSA product type: %v", csvMeta.ProdType)
		}

		pmc, err := csvMeta.PMC()
		if err != nil {
			return nil, err
		}

		csvPath := path.Join(inPath, file)
		jobLog.Infof("  Reading %v MSA: %v", readType, csvPath)
		lines, err := importerutils.ReadFileLines(csvPath, jobLog)
		if err != nil {
			return nil, fmt.Errorf("Failed to load %v: %v", csvPath, err)
		}

		// Parse the MSA data
		spectrumList, err := importerutils.ReadMSAFileLines(lines, false, false, false)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse %v: %v", csvPath, err)
		}

		// Set the read type, detector & PMC
		for c := range spectrumList {
			detector := "A"
			if c > 0 {
				detector = "B"
			}
			spectrumList[c].Meta["READTYPE"] = dataConvertModels.StringMetaValue(readType)
			spectrumList[c].Meta["DETECTOR_ID"] = dataConvertModels.StringMetaValue(detector)
			spectrumList[c].Meta["PMC"] = dataConvertModels.IntMetaValue(pmc)
		}

		if _, ok := result[pmc]; !ok {
			result[pmc] = []dataConvertModels.DetectorSample{}
		}
		result[pmc] = append(result[pmc], spectrumList...)
	}

	return result, nil
}
