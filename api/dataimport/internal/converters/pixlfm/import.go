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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/dataimport/internal/importerutils"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
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
	overrideInstrument protos.ScanInstrument
	overrideDetector   string
}

func (p *PIXLFM) SetOverrides(instrument protos.ScanInstrument, detector string) {
	p.overrideInstrument = instrument
	p.overrideDetector = detector
}

type fileStructure struct {
	directories       []string
	extensions        string
	expectedFileCount int
}

func (p PIXLFM) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	localFS := &fileaccess.FSAccess{}

	beamDir := fileStructure{}
	spectraDir := fileStructure{}
	bulkSpectraDir := fileStructure{}
	contextImgDir := fileStructure{}
	housekeepingDir := fileStructure{}
	pseudoIntensityDir := fileStructure{}
	rgbuImgDir := fileStructure{}
	discoImgDir := fileStructure{}

	log.Infof("Checking path \"%v\" for FM dataset type", importPath)
	pathType, err := DetectPIXLFMStructure(importPath)
	if err != nil {
		return nil, "", err
	}

	log.Infof("Found path \"%v\" is of type %v", importPath, pathType)
	if pathType == "DataDrive" {
		// This is the official way we receive PIXL FM data from Mars
		// We expect these directories to exist...
		beamDir = fileStructure{[]string{"RXL"}, "csv", 1} // The BGT file contains the positions of the commanded X-ray shots and the actual X-ray shots (in x,y,z). We need RXL (containing image i/js)
		spectraDir = fileStructure{[]string{"RFS"}, "csv", 1}
		bulkSpectraDir = fileStructure{[]string{"RBS", "RMS"}, "msa", 2}
		contextImgDir = fileStructure{[]string{"RCM"}, "tif", -1}
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
		contextImgDir = fileStructure{[]string{"image_mark_up"}, "tif", -1}
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
	//pathsToRead := [][]string{beamDir, spectraDir, bulkSpectraDir, contextImgDir, housekeepingDir, pseudoIntensityDir}
	pathsToRead := map[string]fileStructure{"beamDir": beamDir, "spectraDir": spectraDir, "bulkSpectraDir": bulkSpectraDir, "contextImgDir": contextImgDir, "housekeepingDir": housekeepingDir, "pseudoIntensityDir": pseudoIntensityDir, "rgbuImgDir": rgbuImgDir, "discoImgDir": discoImgDir}
	for dirType, subdir := range pathsToRead {
		pathToSubdir := importPath
		log.Infof("READING %v from \"%v\", subdirs: \"%v\"...", dirType, pathToSubdir, strings.Join(subdir.directories, ","))

		extUpper := strings.ToUpper(subdir.extensions)
		if extUpper[0:1] != "." {
			extUpper = "." + extUpper
		}

		var allFoundPaths []string
		for _, d := range subdir.directories {
			paths, err := localFS.ListObjects(filepath.Join(pathToSubdir, d), "")

			for _, p := range paths {
				if strings.HasSuffix(strings.ToUpper(path.Ext(p)), extUpper) {
					allFoundPaths = append(allFoundPaths, filepath.Join(d, p))
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

		if subdir.expectedFileCount == 1 && numFoundPaths > 1 {
			// We expected 1 file, but somehow got more

			// NOTE: Early August 2023 we had a case where GDS provided spectrum CSV file names which changed
			//       as later downlinks happened. This meant we had 2 spectrum CSVs, and the code here didn't
			//       reliably pick one, due to reading from a Go map. So we now select the file with the
			//       lower SCLK value because if GDS receives files from later parts of a scan first they will
			//       eventually generate a file with the lower SCLK that contains all points.

			chosenSingleFile := getByLowestSCLK(latestVersionFoundPaths)
			chosenMeta := latestVersionFoundPaths[chosenSingleFile]

			log.Infof("  CHOOSING: \"%v\"", chosenSingleFile)

			// Overwrite so we only have this choice...
			latestVersionFoundPaths = map[string]gdsfilename.FileNameMeta{chosenSingleFile: chosenMeta}
		}

		// OK we have the paths, now read this type
		switch dirType {
		case "beamDir":
			for file, beamCsvMeta := range latestVersionFoundPaths {
				if beamCsvMeta.ProdType == "RXL" {
					// If files don't conform, don't read...
					beamLookup, err = importerutils.ReadBeamLocationsFile(filepath.Join(pathToSubdir, file), true, 1, log)
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
				locSpectraLookup, err = importerutils.ReadSpectraCSV(filepath.Join(pathToSubdir, file), log)
				if err != nil {
					return nil, "", err
				}
				// Stop after first file
				break
			}
		case "bulkSpectraDir":
			filePaths := []string{}

			for file := range latestVersionFoundPaths {
				filePaths = append(filePaths, filepath.Join(pathToSubdir, file))
			}

			if len(filePaths) > 0 {
				bulkMaxSpectraLookup, err = importerutils.ReadBulkMaxSpectra(filePaths, log)
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
				hkData, err = importerutils.ReadHousekeepingFile(filepath.Join(pathToSubdir, file), 1, log)
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

				pseudoIntensityData, err = importerutils.ReadPseudoIntensityFile(filepath.Join(pathToSubdir, file), false, log)
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

	matchedAlignedImages, err := importerutils.ReadMatchedImages(filepath.Join(importPath, "MATCHED"), beamLookup, log, localFS)
	if err != nil {
		return nil, "", err
	}

	data, err := importerutils.MakeFMDatasetOutput(
		beamLookup,
		hkData,
		locSpectraLookup,
		bulkMaxSpectraLookup,
		contextImgsPerPMC,
		pseudoIntensityData,
		pseudoIntensityRanges,
		matchedAlignedImages,
		rgbuImages,
		discoImages,
		whiteDiscoImage,
		housekeepingFileNameMeta,
		datasetIDExpected,
		p.overrideInstrument,
		p.overrideDetector,
		1, // TODO: Retrieve beam version and set it here!
		log,
	)

	if err != nil {
		return nil, "", err
	}

	return data, importPath, nil
}

func DetectPIXLFMStructure(importPath string) (string, error) {
	c, _ := os.ReadDir(importPath)
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
	c, _ := os.ReadDir(importPath)
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

func getByLowestSCLK(fileNames map[string]gdsfilename.FileNameMeta) string {
	chosenFile := ""
	var chosenSCLK int32
	for name, meta := range fileNames {
		sclk, err := meta.SCLK()

		if len(chosenFile) == 0 || (err == nil && sclk < chosenSCLK) {
			chosenFile = name
			chosenSCLK = sclk
		}
	}

	return chosenFile
}
