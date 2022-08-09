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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"gitlab.com/pixlise/pixlise-go-api/api/filepaths"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/data-converter/converterModels"

	"gitlab.com/pixlise/pixlise-go-api/data-converter/importer"
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

func (p PIXLFM) Import(importPath string, pseudoIntensityRangesPath string, jobLog logger.ILogger) (*converterModels.OutputData, string, error) {
	// For now this is hard-coded, we may need to parse metadata from a file name to work this out, or it may need to be a param eventually
	const detectorConfig = "PIXL-EM-E2E"
	const group = "PIXL-FM"
	const targetID = "Insert-Target-ID-Here"
	log = jobLog
	beamDir := fileStructure{}
	spectraDir := fileStructure{}
	bulkSpectraDir := fileStructure{}
	contexImgDir := fileStructure{}
	housekeepingDir := fileStructure{}
	pseudoIntensityDir := fileStructure{}
	rgbuImgDir := fileStructure{}
	discoImgDir := fileStructure{}

	pathType, err := detectPaths(importPath)
	if err != nil {
		return nil, "", err
	}

	if pathType == "PIXL-FM" {
		beamDir = fileStructure{[]string{"drift_corr_x_ray_beam_location"}, "csv", 2}
		spectraDir = fileStructure{[]string{"localized_full_spectra"}, "csv", 1}
		bulkSpectraDir = fileStructure{[]string{"bulk_histogram_inputs"}, "msa", 2}
		contexImgDir = fileStructure{[]string{"image_mark_up"}, "tif", -1}
		housekeepingDir = fileStructure{[]string{"spatial_inputs"}, "csv", 1}
		pseudoIntensityDir = fileStructure{[]string{"pseudointensity_maps"}, "csv", 1}
	} else if pathType == "DataDrive" {
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
	}

	// Allocate everything needed (empty, if we find & load stuff, great, but we still need the data struct for the last step)
	beamLookup := converterModels.BeamLocationByPMC{}
	hkData := converterModels.HousekeepingData{}
	locSpectraLookup := converterModels.DetectorSampleByPMC{}
	bulkMaxSpectraLookup := converterModels.DetectorSampleByPMC{}
	contextImgsPerPMC := map[int32]string{}
	pseudoIntensityData := converterModels.PseudoIntensities{}
	pseudoIntensityRanges := []converterModels.PseudoIntensityRange{}
	rgbuImages := []converterModels.ImageMeta{}
	discoImages := []converterModels.ImageMeta{}
	whiteDiscoImage := ""

	//spectraFileNameMeta := FileNameMeta{}
	housekeepingFileNameMeta := FileNameMeta{}

	// Get a path for each file
	//pathsToRead := [][]string{beamDir, spectraDir, bulkSpectraDir, contexImgDir, housekeepingDir, pseudoIntensityDir}
	pathsToRead := map[string]fileStructure{"beamDir": beamDir, "spectraDir": spectraDir, "bulkSpectraDir": bulkSpectraDir, "contextImgDir": contexImgDir, "housekeepingDir": housekeepingDir, "pseudoIntensityDir": pseudoIntensityDir, "rgbuImgDir": rgbuImgDir, "discoImgDir": discoImgDir}
	for dirType, subdir := range pathsToRead {
		pathToSubdir := importPath
		log.Infof("READING %v from \"%v\", subdirs: \"%v\"...\n", dirType, pathToSubdir, strings.Join(subdir.directories, ","))

		var allFoundPaths []string
		for _, d := range subdir.directories {
			paths, err := importer.GetDirListing(path.Join(pathToSubdir, d), subdir.extensions, log)

			for _, p := range paths {
				allFoundPaths = append(allFoundPaths, path.Join(d, p))
			}

			if err != nil {
				log.Infof("  WARNING: Failed to read dir \"%v\". SKIPPING. Error was: \"%v\"\n", pathToSubdir, err)
				err = nil
			}
		}

		latestVersionFoundPaths := getLatestFileVersions(allFoundPaths, log)
		numFoundPaths := len(latestVersionFoundPaths)

		if numFoundPaths < len(allFoundPaths) {
			// Print out all the files ignored due to being old versions...
			for _, allFoundItem := range allFoundPaths {
				if _, ok := latestVersionFoundPaths[allFoundItem]; !ok {
					log.Infof("  IGNORED: \"%v\", due to being older version\n", allFoundItem)
				}
			}
		}

		// Check we got the right amount of files
		if subdir.expectedFileCount > 0 {
			expVsFoundPathsCount := subdir.expectedFileCount - numFoundPaths
			if expVsFoundPathsCount > 0 {
				log.Infof("  WARNING: Not enough %v files found in %v, only found %v!\n", subdir.extensions, strings.Join(subdir.directories, ","), numFoundPaths)
			} else if expVsFoundPathsCount < 0 {
				log.Infof("  WARNING: Unexpected %v file count %v in %v. Check that we read the right one!\n", subdir.extensions, numFoundPaths, strings.Join(subdir.directories, ","))
			}
		}

		// OK we have the paths, now read this type
		switch dirType {
		case "beamDir":
			for file, beamCsvMeta := range latestVersionFoundPaths {
				if beamCsvMeta.prodType == "RXL" {
					// If files don't conform, don't read...
					beamLookup, err = importer.ReadBeamLocationsFile(path.Join(pathToSubdir, file), true, 1, log)
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
				fmt.Println("No beam location found for this dataset. Continuing in case it's a \"disco\" dataset")
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
					log.Infof("  WARNING: No PMC in context image file name: \"%v\"\n", file)
				} else {
					contextImgsPerPMC[pmc] = file
				}
			}
		case "housekeepingDir":
			file := ""
			for file, housekeepingFileNameMeta = range latestVersionFoundPaths {
				hkData, err = importer.ReadHousekeepingFile(path.Join(pathToSubdir, file), 1, log)
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

				pseudoIntensityRanges, err = importer.ReadPseudoIntensityRangesFile(pseudoIntensityRangesPath, log)
				if err != nil {
					return nil, "", err
				}

				pseudoIntensityData, err = importer.ReadPseudoIntensityFile(path.Join(pathToSubdir, file), false, log)
				if err != nil {
					return nil, "", err
				}
				// Stop after first file
				break
			}
		case "rgbuImgDir":
			for file, meta := range latestVersionFoundPaths {

				info := converterModels.ImageMeta{FileName: file}

				pmc, err := meta.PMC()
				if err != nil {
					//return result, nil
					log.Infof("RGBU image file name \"%v\" did not contain PMC: %v\n", file, err)
				}

				info.PMC = pmc
				info.LEDs = meta.colourFilter
				info.ProdType = meta.prodType

				// RGBU data sits in its own directory, TIFF files which must be output unchanged
				rgbuImages = append(rgbuImages, info)
			}
		case "discoImgDir":
			for file, meta := range latestVersionFoundPaths {
				discoInfo := converterModels.ImageMeta{FileName: file}

				pmc, err := meta.PMC()
				if err != nil {
					//return result, nil
					log.Infof("DISCO image file name \"%v\" did not contain PMC: %v\n", file, err)
				}

				discoInfo.PMC = pmc
				discoInfo.LEDs = meta.colourFilter

				if meta.colourFilter == "W" {
					whiteDiscoImage = file
				}

				discoImages = append(discoImages, discoInfo)
			}
		default:
			return nil, "", errors.New("Error searching paths")
		}
	}

	cmeta, err := readCustomMeta(log, importPath)
	if err != nil {
		return nil, "", err
	}
	matchedAlignedImages, err := readMatchedImages(path.Join(importPath, "MATCHED"), beamLookup, log)

	if err != nil {
		return nil, "", err
	}

	// Now that all have been read, combine the bulk/max spectra into our lookup
	for pmc := range bulkMaxSpectraLookup {
		locSpectraLookup[pmc] = append(locSpectraLookup[pmc], bulkMaxSpectraLookup[pmc]...)
	}

	importer.LogIfMoreFoundMSA(locSpectraLookup, "MSA/spectrum", 2)
	// Not really relevant, what would we show? It's a list of meta, how many is too many?
	//importer.LogIfMoreFoundHousekeeping(hkData, "Housekeeping", 1)

	// Build internal representation of the data that we can pass to the output code
	// We now read the metadata from the housekeeping file name, as it's the only file we expect to always exist!
	meta, err := makeDatasetFileMeta(housekeepingFileNameMeta, cmeta, log)
	//meta, err := makeDatasetFileMeta(spectraFileNameMeta)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to parse file metadata: %v", err)
	}

	// Expecting RTT to be read
	if meta.RTT <= 0 {
		return nil, "", errors.New("Failed to determine dataset RTT")
	}

	data := &converterModels.OutputData{
		DatasetID:      strconv.Itoa(int(meta.RTT)),
		Group:          group,
		Meta:           meta,
		DetectorConfig: detectorConfig,
		//BulkQuantFile: "", <-- no bulk quant for tactical... TODO: what do we do here, does a scientist do it and we publish it back through PDS?
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*converterModels.PMCData{},
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

func readCustomMeta(jobLog logger.ILogger, importPath string) (map[string]interface{}, error) {
	var result map[string]interface{}

	metapath := path.Join(importPath, filepaths.DatasetCustomMetaFileName)
	jobLog.Infof("Checking for custom meta: %v\n", metapath)

	if _, err := os.Stat(metapath); os.IsNotExist(err) {
		jobLog.Infof("Can't find custom meta file\n")
		return result, nil
	}

	localFS := fileaccess.FSAccess{}
	err := localFS.ReadJSON("", metapath, &result, false)
	if err != nil {
		jobLog.Infof("Can't read custom meta file\n")
		return result, err
	}
	fmt.Println("Successfully Opened custom-meta")
	return result, err
}

func detectPaths(importPath string) (string, error) {
	c, _ := ioutil.ReadDir(importPath)
	for _, entry := range c {
		if entry.IsDir() && entry.Name() == "drift_corr_x_ray_beam_location" {
			return "PIXL-FM", nil
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
func makeDatasetFileMeta(fMeta FileNameMeta, cmeta map[string]interface{}, jobLog logger.ILogger) (converterModels.FileMetaData, error) {
	result := converterModels.FileMetaData{}

	sol, err := fMeta.SOL()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SOL: %v\n", err)
	}

	rtt, err := fMeta.RTT()
	if err != nil {
		return result, nil
	}

	sclk, err := fMeta.SCLK()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SCLK: %v\n", err)
	}

	site, err := fMeta.site()
	if err != nil {
		return result, nil
	}

	drive, err := fMeta.drive()
	if err != nil {
		return result, nil
	}

	title := strconv.Itoa(int(rtt))
	if val, ok := cmeta["title"]; ok {
		jobLog.Infof("Found custom title")
		v := fmt.Sprintf("%v", val)
		jobLog.Infof("Setting title to: %v", v)
		if len(v) > 0 && val != " " {
			title = v
		}
	}

	result.SOL = sol
	result.RTT = rtt
	result.SCLK = sclk
	result.SiteID = site
	result.DriveID = drive
	result.TargetID = "?"
	result.Title = title
	return result, nil
}

func readBulkMaxSpectra(inPath string, files []string, jobLog logger.ILogger) (converterModels.DetectorSampleByPMC, error) {
	result := converterModels.DetectorSampleByPMC{}

	for _, file := range files {
		// Parse metadata for file
		csvMeta, err := ParseFileName(file)
		if err != nil {
			return nil, err
		}

		// Make sure it's one of the products we're expecting
		readType := ""
		if csvMeta.prodType == "RBS" {
			readType = "BulkSum"
		} else if csvMeta.prodType == "RMS" {
			readType = "MaxValue"
		} else {
			return nil, fmt.Errorf("Unexpected bulk/max MSA product type: %v", csvMeta.prodType)
		}

		pmc, err := csvMeta.PMC()
		if err != nil {
			return nil, err
		}

		csvPath := path.Join(inPath, file)
		jobLog.Infof("  Reading %v MSA: %v\n", readType, csvPath)
		lines, err := importer.ReadFileLines(csvPath, jobLog)
		if err != nil {
			return nil, fmt.Errorf("Failed to load %v: %v", csvPath, err)
		}

		// Parse the MSA data
		spectrumList, err := importer.ReadMSAFileLines(lines, false, false, false)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse %v: %v", csvPath, err)
		}

		// Set the read type, detector & PMC
		for c := range spectrumList {
			detector := "A"
			if c > 0 {
				detector = "B"
			}
			spectrumList[c].Meta["READTYPE"] = converterModels.StringMetaValue(readType)
			spectrumList[c].Meta["DETECTOR_ID"] = converterModels.StringMetaValue(detector)
			spectrumList[c].Meta["PMC"] = converterModels.IntMetaValue(pmc)
		}

		if _, ok := result[pmc]; !ok {
			result[pmc] = []converterModels.DetectorSample{}
		}
		result[pmc] = append(result[pmc], spectrumList...)
	}

	return result, nil
}

func readMatchedImages(matchedPath string, beamLookup converterModels.BeamLocationByPMC, jobLog logger.ILogger) ([]converterModels.MatchedAlignedImageMeta, error) {
	result := []converterModels.MatchedAlignedImageMeta{}

	// Read all JSON files in the directory, if they reference a context image by file name great, otherwise error
	files, err := importer.GetDirListing(matchedPath, "json", jobLog)

	if err != nil {
		fmt.Println("readMatchedImages: directory not found, SKIPPING")
		return result, nil
	}

	for _, jsonFile := range files {
		jsonPath := path.Join(matchedPath, jsonFile)
		// Read JSON file
		jsonBytes, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			return result, err
		}

		var meta converterModels.MatchedAlignedImageMeta
		err = json.Unmarshal(jsonBytes, &meta)
		if err != nil {
			return result, err
		}

		// Verify the images exist
		if _, ok := beamLookup[meta.AlignedBeamPMC]; !ok {
			return result, fmt.Errorf("Matched image %v references beam locations for PMC which cannot be found: %v", jsonPath, meta.AlignedBeamPMC)
		}

		// Work out the full path, will be needed when copying to output dir
		meta.MatchedImageFullPath = path.Join(matchedPath, meta.MatchedImageName)

		_, err = os.Stat(meta.MatchedImageFullPath)
		if err != nil {
			return result, fmt.Errorf("Matched image %v references image which cannot be found: %v", jsonPath, meta.MatchedImageName)
		}

		// And the offsets are valid. I doubt we'll be loading images much larger than maxSize:
		const maxSize = 10000.0
		if meta.XOffset < -maxSize || meta.XOffset > maxSize || meta.YOffset < -maxSize || meta.YOffset > maxSize {
			return result, fmt.Errorf("%v x/y offsets invalid", jsonPath)
		}

		// And the scale values are valid
		const maxScale = 100.0 // 100x greater/less resolution... not likely!
		if meta.XScale < 1/maxScale || meta.XScale > maxScale || meta.YScale < 1/maxScale || meta.YScale > maxScale {
			return result, fmt.Errorf("%v x/y scales invalid", jsonPath)
		}

		result = append(result, meta)
	}

	return result, nil
}
