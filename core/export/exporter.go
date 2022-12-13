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

package export

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/timestamper"
	"github.com/pixlise/core/v2/core/utils"
	protos "github.com/pixlise/core/v2/generated-protos"
	"google.golang.org/protobuf/proto"
)

// The actual exporter, implemented by our package. This is so we can be used as part of an interface by caller
type Exporter struct {
}

type imageDataWithMatchMeta struct {
	alignedPMC int32
	matchMeta  *protos.Experiment_MatchedContextImageInfo
	data       []byte
}

const FileIdSpectra = "raw-spectra"
const FileIdQuantMapCSV = "quant-map-csv"
const FileIdQuantMapTIF = "quant-map-tif"
const FileIdBeamLocations = "beam-locations"
const FileIdROIs = "rois"
const FileIdContextImage = "context-image"
const FileIdUnquantifiedWeightPct = "unquantified-weight"
const FileIdDiffractionPeak = "diffraction-peak"

// The above IDs specify what to download, and they get downloaded in parallel into this structure by downloadInputs()
type inputFiles struct {
	quantBin      *protos.Quantification
	dataset       *protos.Experiment
	contextImages map[string]imageDataWithMatchMeta
	userROIs      roiModel.ROILookup
	sharedROIs    roiModel.ROILookup
	quantCSVFile  []byte
	diffraction   *protos.Diffraction
}

// MakeExportFilesZip - makes a zip file containing all requested export data
func (m *Exporter) MakeExportFilesZip(svcs *services.APIServices, outfileNamePrefix string, userID string, datasetID string, quantID string, quantPath string, fileIDs []string, roiIDs []string) ([]byte, error) {
	// Start by making a temp dir to write all this to...
	outDir, err := ioutil.TempDir("", "img-export-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)

	// Clean it up
	fileNamePrefix := utils.MakeSaveableFileName(outfileNamePrefix)

	// Turn the file IDs into bools that are easier to deal with from here on. In future
	// if we have many more ids we may do something different...
	wantQuantCSV := false
	wantQuantTIF := false
	wantUnquantifiedPct := false
	wantSpectra := false
	wantContextImage := false
	wantBeamLocations := false
	wantROIs := false
	wantDiffractionPeak := false

	for _, id := range fileIDs {
		if id == FileIdQuantMapCSV {
			wantQuantCSV = true
		} else if id == FileIdQuantMapTIF {
			wantQuantTIF = true
		} else if id == FileIdUnquantifiedWeightPct {
			wantUnquantifiedPct = true
		} else if id == FileIdSpectra {
			wantSpectra = true
		} else if id == FileIdBeamLocations {
			wantBeamLocations = true
		} else if id == FileIdROIs {
			wantROIs = true
		} else if id == FileIdContextImage {
			wantContextImage = true
		} else if id == FileIdDiffractionPeak {
			wantDiffractionPeak = true
		}
	}

	if quantID == "" && (wantQuantCSV || wantQuantTIF || wantUnquantifiedPct) {
		return nil, fmt.Errorf("Cannot export quantified data, no QuantID specified")
	}

	// Work out the paths of quant files if we want to load them
	quantBINPath := ""
	quantCSVPath := ""
	//quantSummaryPath := path.Join(quantPath, quant.filepaths.MakeQuantSummaryFileName(quantID))

	if wantUnquantifiedPct || wantQuantTIF {
		quantBINPath = path.Join(quantPath, filepaths.MakeQuantDataFileName(quantID))
	}
	if wantQuantCSV {
		quantCSVPath = path.Join(quantPath, quantID+".csv")
	}

	// We download ROI files if we're looking at spectra too, because we want to export spectra files for each ROI
	files, err := downloadInputs(svcs, userID, datasetID, quantBINPath, quantCSVPath, wantDiffractionPeak, wantContextImage, wantROIs || wantSpectra || wantQuantCSV, outDir)

	if err != nil {
		return nil, err
	}

	// If ROIs are loaded, this will work, else just generate an empty structure
	rois := roiModel.GetROIsWithPMCs(files.userROIs, files.sharedROIs, files.dataset)

	// Export each file as needed
	pmcBeamLocLookup := datasetModel.MakePMCBeamLookup(files.dataset)
	pmcBeamIndexLookup := datasetModel.MakePMCBeamIndexLookup(files.dataset)

	if wantContextImage && len(files.contextImages) > 0 {
		for imgName, imgMetaData := range files.contextImages {
			// Write the image out
			f, err := os.Create(imgName)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			f.Write(imgMetaData.data)

			// If it's not a TIF file, we export a "marked" copy too
			if !strings.HasSuffix(imgName, ".tif") {
				// Non-TIF images are decoded processed further to add location points, etc
				img, _ /*imgType*/, err := image.Decode(bytes.NewReader(imgMetaData.data))
				if err != nil {
					return nil, err
				}

				err = utils.WritePNGImageFile(path.Join(outDir, imgName), img)
				if err != nil {
					return nil, err
				}

				// If we have matched info, create images showing the locations
				// NOTE: we have to find the bank of i/j coordinates to use
				ijIdx := int32(-1)
				if imgMetaData.alignedPMC >= 0 {
					ijIdx = pmcBeamIndexLookup[imgMetaData.alignedPMC]
				} else if imgMetaData.matchMeta != nil {
					ijIdx = imgMetaData.matchMeta.AlignedIndex - 1 // start from -1, because -1 references the "default" context image, aka aligned image 0's beam coordinates
				}

				markedContext, err := makeMarkupImage(img, ijIdx, files.dataset, imgMetaData.matchMeta)
				if err != nil {
					return nil, err
				}

				err = utils.WritePNGImageFile(path.Join(outDir, "marked-"+imgName), markedContext)
				if err != nil {
					return nil, err
				}
			} else {
				// TIF images are exported as-is
				f, err := os.Create(path.Join(outDir, imgName))
				if err != nil {
					return nil, err
				}
				defer f.Close()

				f.Write(imgMetaData.data)
			}
		}
	}

	if wantSpectra {
		for _, roi := range rois {
			if utils.StringInSlice(roi.ID, roiIDs) {
				svcs.Log.Debugf("  Saving spectra for %v...", roi.Name)

				err = writeSpectraCSVs(svcs.TimeStamper, outDir, fileNamePrefix, "Normal", files.dataset, roi)
				if err != nil {
					return nil, err
				}

				err = writeSpectraCSVs(svcs.TimeStamper, outDir, fileNamePrefix, "Dwell", files.dataset, roi)
				if err != nil {
					return nil, err
				}
			}
		}

		// Write out the "whole dataset" one. For this we create a special ROI
		all := roiModel.GetAllPointsROI(files.dataset)
		err = writeSpectraCSVs(svcs.TimeStamper, outDir, fileNamePrefix, "Normal", files.dataset, all)
		if err != nil {
			return nil, err
		}

		err = writeSpectraCSVs(svcs.TimeStamper, outDir, fileNamePrefix, "Dwell", files.dataset, all)
		if err != nil {
			return nil, err
		}
	}

	if wantBeamLocations {
		svcs.Log.Debugf("  Saving IJ CSV...")
		err = writeBeamCSV(outDir, fileNamePrefix, pmcBeamLocLookup, files.dataset)
		if err != nil {
			return nil, err
		}
	}

	if wantROIs {
		svcs.Log.Debugf("  Saving ROI CSV...")
		err = writeROICSV(outDir, rois, roiIDs)
		if err != nil {
			return nil, err
		}
	}
	if wantDiffractionPeak {
		svcs.Log.Debugf("  Diffraction: %v - Found %v locations with peaks", files.diffraction.Title, len(files.diffraction.Locations))
		csv, err := os.Create(path.Join(outDir, fileNamePrefix+"-diffraction-peaks.csv"))
		if err != nil {
			return nil, err
		}

		defer csv.Close()

		locations := files.diffraction.Locations
		sort.Slice(locations, func(i, j int) bool {
			firstLocID, err := strconv.Atoi(locations[i].Id)
			if err != nil {
				firstLocID = 0
			}
			secondLocID, err := strconv.Atoi(locations[j].Id)
			if err != nil {
				secondLocID = 0
			}
			return firstLocID < secondLocID
		})

		_, err = csv.WriteString("PMC, Peak Channel, Peak Height, Effect Size, Baseline Variation, Difference Sigma, Global Difference\n")
		if err != nil {
			return nil, err
		}

		for _, loc := range locations {
			for _, p := range loc.Peaks {
				line := fmt.Sprintf("%s, %v, %v, %v, %v, %v, %v\n", loc.Id, p.PeakChannel, p.PeakHeight, p.EffectSize, p.BaselineVariation, p.DifferenceSigma, p.GlobalDifference)
				_, err = csv.WriteString(line)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if wantQuantCSV {
		// Save the whole original CSV
		csvPath := path.Join(outDir, fileNamePrefix+"-map-by-PIQUANT.csv")
		ioutil.WriteFile(csvPath, files.quantCSVFile, 0644)

		// Also save one per ROI as a convenience feature
		csvLines := strings.Split(string(files.quantCSVFile), "\n")

		for _, roi := range rois {
			if utils.StringInSlice(roi.ID, roiIDs) {
				_, err = writeQuantCSVForROI(csvLines, roi, outDir, fileNamePrefix)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if wantUnquantifiedPct {
		weightPctCols := quantModel.GetWeightPercentColumnsInQuant(files.quantBin)

		unquantWeightPct := []map[int32]float32{}
		unquantWeightPctDetector := []string{}
		for detectorIdx, locSet := range files.quantBin.LocationSet {
			unquant, err := makeUnquantifiedMapValues(pmcBeamLocLookup, files.quantBin, detectorIdx, weightPctCols)
			if err != nil {
				return nil, err
			}
			unquantWeightPct = append(unquantWeightPct, unquant)
			unquantWeightPctDetector = append(unquantWeightPctDetector, locSet.Detector)
		}
		svcs.Log.Debugf("  Unquantified map(s) generated")

		svcs.Log.Debugf("  Saving Unquantified weight %% CSV...")
		err = writeUnquantifiedWeightPctCSV(outDir, fileNamePrefix, unquantWeightPctDetector, unquantWeightPct)
		if err != nil {
			return nil, err
		}
	}

	svcs.Log.Debugf("  Making zip")

	// Zip it all up!
	zipData, err := utils.ZipDirectory(outDir)
	if err != nil {
		return nil, err
	}

	svcs.Log.Debugf("  Returning zip in response")

	// Return the zip
	return zipData, nil
}

// Downloads the specified input files. Note that quant CSV is just written to the outputDir
// because we don't process this further. The rest are returned in an inputFiles struct
func downloadInputs(
	svcs *services.APIServices,
	userID string,
	datasetID string,
	quantBINPath string, // if blank, not loaded
	quantCSVPath string, // if blank, not loaded
	loadDiffractionPeak bool,
	loadContextImage bool, // if false, not loaded
	loadROIs bool, // if false, not loaded
	outputDir string, // only quant CSV is written directly. This is here in case other files are needed in future too
) (inputFiles, error) {
	result := inputFiles{
		contextImages: map[string]imageDataWithMatchMeta{},
		userROIs:      roiModel.ROILookup{},
		sharedROIs:    roiModel.ROILookup{},
	}

	// Start downloads simultaneously
	var wg sync.WaitGroup
	var datasetError error

	errors := []error{}

	if len(datasetID) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			svcs.Log.Debugf("  Downloading dataset id=%v", datasetID)
			datasetPath := filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetFileName)
			result.dataset, datasetError = datasetModel.GetDataset(svcs, datasetPath)
			svcs.Log.Debugf("  Dataset download finished, error: %v", datasetError)
		}()
	}

	if len(quantBINPath) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			svcs.Log.Debugf("  Downloading quantification %v", quantBINPath)
			var quantError error
			result.quantBin, quantError = quantModel.GetQuantification(svcs, quantBINPath)
			if quantError != nil {
				errors = append(errors, quantError)
			}
			svcs.Log.Debugf("  Quantification download finished, error: %v", quantError)
		}()
	}

	if len(quantCSVPath) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Download the quant CSV to the dir
			svcs.Log.Debugf("  Downloading quant CSV %v", quantCSVPath)

			var err error
			result.quantCSVFile, err = svcs.FS.ReadObject(svcs.Config.UsersBucket, quantCSVPath)
			if err != nil {
				// Don't fail due to CSV missing, we just won't supply the CSV...
				svcs.Log.Errorf("Failed to download map CSV for zipping. Path was: %v", quantCSVPath)
			} else {
				svcs.Log.Debugf("  Quantification CSV finished")
			}
		}()
	}
	if loadDiffractionPeak {
		wg.Add(1)
		go func() {
			defer wg.Done()

			diffraction := &protos.Diffraction{}

			s3Path := filepaths.GetDatasetFilePath(datasetID, filepaths.DiffractionDBFileName)
			diffractionData, err := svcs.FS.ReadObject(svcs.Config.DatasetsBucket, s3Path)
			if err != nil {
				errors = append(errors, err)
			}

			err = proto.Unmarshal(diffractionData, diffraction)
			if err != nil {
				errors = append(errors, err)
			} else {
				result.diffraction = diffraction
			}
		}()
	}

	if loadROIs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			s3Path := filepaths.GetROIPath(userID, datasetID)
			userROIsError := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &result.userROIs, true)
			if userROIsError != nil {
				errors = append(errors, userROIsError)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			s3Path := filepaths.GetROIPath(pixlUser.ShareUserID, datasetID)
			sharedROIsError := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &result.sharedROIs, true)
			if sharedROIsError != nil {
				errors = append(errors, sharedROIsError)
			}
		}()
	}

	// Wait for all
	wg.Wait()

	if loadContextImage && datasetError == nil {
		// We now have the dataset, we can work out the names of the context images
		errs := loadContextImages(svcs, datasetID, result)
		if len(errs) > 0 {
			for _, err := range errs {
				if err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	// If we found any errors, return the first one
	if len(errors) > 0 {
		return result, errors[0]
	}

	return result, nil
}

func loadContextImages(svcs *services.APIServices, datasetID string, result inputFiles) []error {
	var wgImg sync.WaitGroup
	var errs []error

	fileNames := []string{}
	matchMeta := []*protos.Experiment_MatchedContextImageInfo{}
	alignedPMC := []int32{}

	for _, meta := range result.dataset.AlignedContextImages {
		fileNames = append(fileNames, meta.Image)
		matchMeta = append(matchMeta, nil)
		alignedPMC = append(alignedPMC, meta.Pmc)
	}

	for _, meta := range result.dataset.MatchedAlignedContextImages {
		fileNames = append(fileNames, meta.Image)
		matchMeta = append(matchMeta, meta)
		alignedPMC = append(alignedPMC, -1)
	}

	for _, imgName := range result.dataset.UnalignedContextImages {
		fileNames = append(fileNames, imgName)
		matchMeta = append(matchMeta, nil)
		alignedPMC = append(alignedPMC, -1)
	}

	for c := range fileNames {
		wgImg.Add(1)
		go func(imgIdx int) {
			defer wgImg.Done()

			imgName := fileNames[imgIdx]

			svcs.Log.Debugf("  Downloading context image: \"%v\"", imgName)
			contextImagePath := filepaths.GetDatasetFilePath(datasetID, imgName)

			imgbytes, err := svcs.FS.ReadObject(svcs.Config.DatasetsBucket, contextImagePath)
			if err != nil {
				svcs.Log.Errorf("  Error downloading \"%v\": %v", imgName, err)
			} else {
				svcs.Log.Debugf("  Download \"%v\" finished", imgName)
			}

			errs = append(errs, err)
			result.contextImages[imgName] = imageDataWithMatchMeta{alignedPMC: alignedPMC[imgIdx], matchMeta: matchMeta[imgIdx], data: imgbytes}
		}(c)
	}

	wgImg.Wait()

	return errs
}

/*
	func makeReadme(svcs *services.APIServices, s3Path string, outPath string) error {
		var summary quantModel.JobSummaryItem
		err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &summary, false)
		if err != nil {
			return err
		}

		// Save the readme
		readme, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer readme.Close()

		readme.WriteString("=============================\n")
		readme.WriteString("= Quantification Map Export =\n")
		readme.WriteString("=============================\n\n")
		readme.WriteString(fmt.Sprintf("Name: %v\nQuantification ID: %v\nElements: %v\nDataset ID: %v\n", summary.Params.Name, summary.JobID, strings.Join(summary.Params.Elements, ","), summary.Params.DatasetID))
		readme.WriteString(fmt.Sprintf("Processing time: %v sec on %v cores\n", summary.EndUnixTime-summary.Params.StartUnixTime, summary.Params.CoresPerNode*int32(len(summary.PiquantLogList)/2)))
		_, err = readme.WriteString(fmt.Sprintf("Creator: %v\nPiquant detector config: %v\nPiquant custom params: \"%v\"\n", summary.Params.Creator.Name, summary.Params.DetectorConfig, summary.Params.Parameters))

		return err
	}
*/
func expandRect(r image.Rectangle, x, y int) image.Rectangle {
	if x > r.Max.X {
		r.Max.X = x
	}
	if y > r.Max.Y {
		r.Max.Y = y
	}
	if x < r.Min.X {
		r.Min.X = x
	}
	if y < r.Min.Y {
		r.Min.Y = y
	}

	return r
}

func makeMarkupImage(contextImage image.Image, beamIJBankIdx int32, dataset *protos.Experiment, matchMeta *protos.Experiment_MatchedContextImageInfo) (image.Image, error) {
	// Copy the image data to output image (in greyscale)
	bounds := contextImage.Bounds()

	var outImage draw.Image
	if contextImage.ColorModel() == color.GrayModel {
		outImage = image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	} else {
		outImage = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	}

	draw.Draw(outImage, outImage.Bounds(), contextImage, bounds.Min, draw.Src)

	// Run through all locations & set a white pixel where they are in the context image
	for _, loc := range dataset.Locations {
		if loc.Beam != nil && loc.Beam.ImageI > 0 && loc.Beam.ImageJ > 0 {
			i := loc.Beam.ImageI
			j := loc.Beam.ImageJ

			// If we're looking at a specific pmc...
			if beamIJBankIdx >= 0 {
				i = loc.Beam.ContextLocations[beamIJBankIdx].I
				j = loc.Beam.ContextLocations[beamIJBankIdx].J
			}

			// If we've got a matched image, we need to modify the coordinates to be relative to the image we're outputting
			// NOTE: PIXLISE does the opposite, it uses the same key
			if matchMeta != nil {
				i *= matchMeta.XScale
				i -= matchMeta.XOffset

				j *= matchMeta.YScale
				j -= matchMeta.YOffset
			}

			outImage.Set(int(i+0.5), int(j+0.5), color.White)
		}
	}

	// We're done with this image
	return outImage, nil
}

func makeUnquantifiedMapValues(
	pmcBeamLocLookup map[int32]protos.Experiment_Location_BeamLocation,
	quant *protos.Quantification,
	detectorIdx int,
	weightPctCols []string) (map[int32]float32, error) {

	result := map[int32]float32{}

	weightPctColIdxs := []int32{}
	for _, col := range weightPctCols {
		idx := quantModel.GetQuantColumnIndex(quant, col)
		if idx < 0 {
			return result, fmt.Errorf("makeUnquantifiedMapValues: Failed to get column: %v", col)
		}
		weightPctColIdxs = append(weightPctColIdxs, idx)
	}

	// Get the quant data & find its beam location info for each row
	locSet := quant.LocationSet[detectorIdx]
	for _, loc := range locSet.Location {
		pmc := int32(loc.Pmc)

		// Find its beam
		_, ok := pmcBeamLocLookup[pmc]
		if !ok {
			return result, fmt.Errorf("makeUnquantifiedMapValues: Failed to find beam location for PMC: %v", pmc)
		}

		// Calculate the value
		quantVal := float32(100.0)
		for _, idx := range weightPctColIdxs {
			quantVal -= loc.Values[idx].Fvalue
		}

		result[pmc] = quantVal
	}

	// We use the quant data to work out the colour of the pixel we're setting
	return result, nil
}

// Writes i/j coordinates for each PMC to a CSV file
func writeBeamCSV(dir string, fileNamePrefix string, pmcBeamLocLookup map[int32]protos.Experiment_Location_BeamLocation, dataset *protos.Experiment) error {
	csv, err := os.Create(path.Join(dir, fileNamePrefix+"-beam-locations.csv"))
	if err != nil {
		return err
	}
	defer csv.Close()

	header := "PMC,X,Y,Z"
	for _, img := range dataset.AlignedContextImages {
		header += fmt.Sprintf(",%v_i,%v_j", img.Image, img.Image)
	}

	_, err = csv.WriteString(header + "\n")
	if err != nil {
		return err
	}

	// Iterate through in PMC ascending order
	pmcs := []int{}
	for pmc := range pmcBeamLocLookup {
		pmcs = append(pmcs, int(pmc))
	}

	sort.Ints(pmcs)

	for _, pmc := range pmcs {
		beam := pmcBeamLocLookup[int32(pmc)]
		line := fmt.Sprintf("%v,%v,%v,%v,%v,%v", pmc, beam.X, beam.Y, beam.Z, beam.ImageI, beam.ImageJ)

		// Add any other beam locations that are stored
		for _, loc := range beam.ContextLocations {
			line += fmt.Sprintf(",%v,%v", loc.I, loc.J)
		}

		_, err = csv.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes a CSV which contains the PMCs for each ROI that the user has access to
// NOTE:  This writes multiple tables into the same CSV file, first specifying the ROI name and id, then the PMCs for that ROI
// NOTE2: Writes user and shared ROIs into the same table
func writeROICSV(dir string, rois []roiModel.ROIMembers, roiIDs []string) error {
	sharedStrippedIDs := []string{}
	for _, id := range roiIDs {
		strippedID, _ := utils.StripSharedItemIDPrefix(id)
		sharedStrippedIDs = append(sharedStrippedIDs, strippedID)
	}

	for _, roi := range rois {
		roiID, _ := utils.StripSharedItemIDPrefix(roi.ID)
		if !utils.StringInSlice(roiID, sharedStrippedIDs) {
			continue
		}

		name := roi.Name

		pathSafeName := strings.Replace(strings.Replace(name, " ", "_", -1), "/", "_", -1)
		csv, err := os.Create(path.Join(dir, pathSafeName+"-roi-pmcs.csv"))
		if err != nil {
			return err
		}
		defer csv.Close()

		if len(roi.SharedByName) > 0 {
			name = name + "(shared by " + roi.SharedByName + ")"
		}

		csvSafeName := strings.Replace(name, ",", "", -1)
		_, err = csv.WriteString(csvSafeName + "\n")
		if err != nil {
			return err
		}

		for _, pmc := range roi.PMCs {
			_, err = csv.WriteString(fmt.Sprintf("%v\n", pmc))
			if err != nil {
				return err
			}
		}

		csv.WriteString("\n")
	}

	return nil
}

type spectrumData struct {
	PMC int32

	x float32
	y float32
	z float32

	//yellowPieceTemp float32

	metaA datasetModel.SpectrumMetaValues
	metaB datasetModel.SpectrumMetaValues

	countsA []int32
	countsB []int32
}

func writeSpectraCSVs(timeStamper timestamper.ITimeStamper, outDir string, fileNamePrefix string, readType string, dataset *protos.Experiment, roi roiModel.ROIMembers) error {
	// Write out a CSV in the format we receive them from iSDS (pixlise-data-converter FM format CSV for RFS product type subdirectory)
	// First we read the spectra we're interested in...
	toWrite := []spectrumData{}
	for _, locIdx := range roi.LocationIdxs {
		loc := dataset.Locations[locIdx]

		// NOTE: we only support zero run encoded spectra...
		if loc.SpectrumCompression != protos.Experiment_Location_ZERO_RUN {
			return errors.New("writeSpectraCSVs failed, we only support spectrums with zero-run encoding")
		}

		// Get the PMC
		pmc, err := strconv.ParseInt(loc.GetId(), 10, 32)
		if err != nil {
			return fmt.Errorf("Unexpected PMC: %v", loc.GetId())
		}

		spectrumItem := spectrumData{
			PMC: int32(pmc),
		}

		if loc.Beam != nil {
			spectrumItem.x = loc.Beam.X
			spectrumItem.y = loc.Beam.Y
			spectrumItem.z = loc.Beam.Z
		}

		for _, det := range loc.Detectors {
			detectorMeta, err := datasetModel.GetSpectrumMeta(det, dataset)

			// We may fail to read some stuff, there may be no spectrum or metadata in this PMC, that's OK
			if err == nil && len(det.Spectrum) > 0 && detectorMeta.ReadType == readType {
				// Now that we have the detector ID, read in the rest of the values
				spectrumValues := datasetModel.DecodeZeroRun(det.Spectrum)

				if detectorMeta.Detector == "A" {
					spectrumItem.metaA = detectorMeta
					spectrumItem.countsA = spectrumValues
				} else if detectorMeta.Detector == "B" {
					spectrumItem.metaB = detectorMeta
					spectrumItem.countsB = spectrumValues
				} else {
					return fmt.Errorf("Unexpected Detector: %v", detectorMeta.Detector)
				}
			}
		}

		// Only save it if we actually read spectrum values!
		if len(spectrumItem.countsA) > 0 && len(spectrumItem.countsB) > 0 {
			if len(spectrumItem.countsA) != len(spectrumItem.countsB) {
				return fmt.Errorf("PMC: %v had %v A spectra channels, %v B spectra channels", loc.GetId(), len(spectrumItem.countsA), len(spectrumItem.countsB))
			}
			toWrite = append(toWrite, spectrumItem)
		}
	}

	// Possible we won't have spectra to export, eg dataset is a test old one with only A or B or partial dataset, etc. Also dataset may just
	// not have spectra data (RGBU disco dataset), but "all points" ROI still exists. Just skip files in this situation instead of erroring out
	if len(toWrite) <= 0 {
		return nil
	}

	writePath := path.Join(outDir, makeFileNameWithROI(fileNamePrefix, readType, roi.Name, roi.SharedByName, "csv"))
	{
		csv, err := os.Create(writePath)
		if err != nil {
			return err
		}
		defer csv.Close()

		err = writeSpectraCSV(writePath, toWrite, csv)
		if err != nil {
			return err
		}
	}

	// Now add all spectra to form a bulk-sum (separate for A and B) and output these
	bulkSum := spectrumData{}
	for c, spectrum := range toWrite {
		if c == 0 {
			bulkSum = spectrum
		} else {
			// We're summing what we can...
			for i := range spectrum.countsA {
				bulkSum.countsA[i] += spectrum.countsA[i]
				bulkSum.countsB[i] += spectrum.countsB[i]
			}

			bulkSum.metaA.LiveTime += spectrum.metaA.LiveTime
			bulkSum.metaB.LiveTime += spectrum.metaB.LiveTime
			bulkSum.metaA.RealTime += spectrum.metaA.RealTime
			bulkSum.metaB.RealTime += spectrum.metaB.RealTime

			bulkSum.metaA.Offset += spectrum.metaA.Offset
			bulkSum.metaB.Offset += spectrum.metaB.Offset
			bulkSum.metaA.XPerChan += spectrum.metaA.XPerChan
			bulkSum.metaB.XPerChan += spectrum.metaB.XPerChan
		}
	}

	// Make the ones that are averages be averages
	bulkSum.metaA.Offset /= float32(len(toWrite))
	bulkSum.metaB.Offset /= float32(len(toWrite))
	bulkSum.metaA.XPerChan /= float32(len(toWrite))
	bulkSum.metaB.XPerChan /= float32(len(toWrite))

	writePath = path.Join(outDir, makeFileNameWithROI(fileNamePrefix, readType+"-BulkSum", roi.Name, roi.SharedByName, "csv"))
	{
		csv, err := os.Create(writePath)
		if err != nil {
			return err
		}
		defer csv.Close()

		err = writeSpectraCSV(writePath, []spectrumData{bulkSum}, csv)
		if err != nil {
			return err
		}
	}

	// We also write these in MSA format as requested by several users, this way they can run PIQUANT on it locally
	writePath = path.Join(outDir, makeFileNameWithROI(fileNamePrefix, readType+"-BulkSum", roi.Name, roi.SharedByName, "msa"))
	{
		msa, err := os.Create(writePath)
		if err != nil {
			return err
		}
		defer msa.Close()

		err = writeSpectraMSA(writePath, timeStamper, bulkSum, msa)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeFileNameWithROI(prefix string, mainName string, roiName string, roiSharer string, ext string) string {
	result := prefix + "-" + mainName + " ROI " + roiName
	if len(roiSharer) > 0 {
		result += " (shared by " + roiSharer + ")"
	}

	result += "." + ext

	result = utils.MakeSaveableFileName(result)
	return result
}

func writeSpectraCSV(path string, spectra []spectrumData, csv io.StringWriter) error {
	if len(spectra) <= 0 {
		return fmt.Errorf("No spectra for writeSpectraCSV when writing %v", path)
	}

	// Table 1: SCLK_A,SCLK_B,PMC,real_time_A,real_time_B,live_time_A,live_time_B,XPERCHAN_A,XPERCHAN_B,OFFSET_A,OFFSET_B
	// NOTE: we omit yellow_piece_temp because we don't store it in dataset bin files!
	_, err := csv.WriteString("SCLK_A,SCLK_B,PMC,real_time_A,real_time_B,live_time_A,live_time_B,XPERCHAN_A,XPERCHAN_B,OFFSET_A,OFFSET_B\n")
	if err != nil {
		return err
	}

	for _, spectrum := range spectra {
		line := fmt.Sprintf(
			"%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v\n",
			spectrum.metaA.SCLK,
			spectrum.metaB.SCLK,
			spectrum.PMC,
			spectrum.metaA.RealTime,
			spectrum.metaB.RealTime,
			spectrum.metaA.LiveTime,
			spectrum.metaB.LiveTime,
			spectrum.metaA.XPerChan,
			spectrum.metaB.XPerChan,
			spectrum.metaA.Offset,
			spectrum.metaB.Offset,
		)

		_, err = csv.WriteString(line)
		if err != nil {
			return err
		}
	}

	// Table 2: PMC,x,y,z
	_, err = csv.WriteString("PMC,x,y,z\n")
	if err != nil {
		return err
	}

	for _, spectrum := range spectra {
		line := fmt.Sprintf(
			"%v,%v,%v,%v\n",
			spectrum.PMC,
			spectrum.x,
			spectrum.y,
			spectrum.z,
		)

		_, err = csv.WriteString(line)
		if err != nil {
			return err
		}
	}

	// Table 3: A_1, ... A_4096
	var sb strings.Builder
	for c := range spectra[0].countsA {
		if c > 0 {
			sb.WriteString(",")
		}

		sb.WriteString(fmt.Sprintf("A_%v", c+1))
	}

	_, err = csv.WriteString(sb.String() + "\n")
	if err != nil {
		return err
	}

	for _, spectrum := range spectra {
		sb.Reset()

		for c, count := range spectrum.countsA {
			if c > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%v", count))
		}

		_, err = csv.WriteString(sb.String() + "\n")
		if err != nil {
			return err
		}
	}

	// Table 3: B_1, ... B_4096
	sb.Reset()
	for c := range spectra[0].countsB {
		if c > 0 {
			sb.WriteString(",")
		}

		sb.WriteString(fmt.Sprintf("B_%v", c+1))
	}

	_, err = csv.WriteString(sb.String() + "\n")
	if err != nil {
		return err
	}

	for _, spectrum := range spectra {
		sb.Reset()

		for c, count := range spectrum.countsB {
			if c > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%v", count))
		}

		_, err = csv.WriteString(sb.String() + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeSpectraMSA(path string, timeStamper timestamper.ITimeStamper, spectra spectrumData, msa io.StringWriter) error {
	if len(spectra.countsA) <= 0 && len(spectra.countsA) <= 0 {
		return fmt.Errorf("Unexpected spectrum data counts for writeSpectraMSA when writing %v", path)
	}

	// We receive spectra in MSA format but don't store it, instead using a binary file that's quicker to download/use in the browser.
	// At this point though we convert back to MSA format with the fields that we have (we don't store everything from the MSA header)

	columns := "Y"
	if len(spectra.countsB) > 0 {
		columns = "YY"
	}

	currentTime := time.Unix(timeStamper.GetTimeNowSec(), 0).UTC()
	dateNow := currentTime.Format("01-02-2006")
	timeNow := currentTime.Format("15:04:05")

	xPerChan := fmt.Sprintf("%v", spectra.metaA.XPerChan)
	offset := fmt.Sprintf("%v", spectra.metaA.Offset)
	livetime := fmt.Sprintf("%v", spectra.metaA.LiveTime)
	realtime := fmt.Sprintf("%v", spectra.metaA.RealTime)

	if len(spectra.countsB) > 0 {
		xPerChan += fmt.Sprintf(", %v", spectra.metaB.XPerChan)
		offset += fmt.Sprintf(", %v", spectra.metaB.Offset)
		livetime += fmt.Sprintf(", %v", spectra.metaB.LiveTime)
		realtime += fmt.Sprintf(", %v", spectra.metaB.RealTime)
	}

	_, err := msa.WriteString(fmt.Sprintf(`#FORMAT      : EMSA/MAS spectral data file
#VERSION     : TC202v2.0 PIXL
#TITLE       : Control Program v7
#OWNER       : JPL BREADBOARD vx
#DATE        : %v
#TIME        : %v
#NPOINTS     : %v
#NCOLUMNS    : %v
#XUNITS      :  eV
#YUNITS      :  COUNTS
#DATATYPE    :  %v
#XPERCHAN    :  %v    eV per channel
#OFFSET      :  %v    eV of first channel
#SIGNALTYPE  :  XRF
#COMMENT     :  Exported bulk sum MSA from PIXLISE
#XPOSITION   :    0.000
#YPOSITION   :    0.000
#ZPOSITION   :    0.000
#LIVETIME    :  %v
#REALTIME    :  %v
#SPECTRUM    :
`,
		dateNow,
		timeNow,
		len(spectra.countsA),
		len(columns),
		columns,
		xPerChan,
		offset,
		livetime,
		realtime,
	))

	if err != nil {
		return err
	}

	// Now we write the lines
	for c, A := range spectra.countsA {
		countLine := fmt.Sprintf("%v", A)
		if len(spectra.countsB) > 0 {
			countLine += fmt.Sprintf(", %v", spectra.countsB[c])
		}

		_, err := msa.WriteString(countLine + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeUnquantifiedWeightPctCSV(dir string, fileNamePrefix string, detectors []string, values []map[int32]float32) error {
	csv, err := os.Create(path.Join(dir, fileNamePrefix+"-unquantified-weight-pct.csv"))
	if err != nil {
		return err
	}
	defer csv.Close()

	// headers
	header := "PMC"
	for _, det := range detectors {
		header += "," + det
	}

	_, err = csv.WriteString(header + "\n")
	if err != nil {
		return err
	}

	// Get the PMCs, which we want to write in sorted order
	pmcs := []int{}
	for pmc := range values[0] {
		pmcs = append(pmcs, int(pmc))
	}

	sort.Ints(pmcs)

	// Write the values
	for _, pmc := range pmcs {
		line := fmt.Sprintf("%v", pmc)

		for detIdx := range values {
			line += fmt.Sprintf(",%v", values[detIdx][int32(pmc)])
		}

		_, err = csv.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeQuantCSVForROI(quantCSVFileLines []string, roi roiModel.ROIMembers, outDir string, fileNamePrefix string) (string, error) {
	if len(quantCSVFileLines) < 3 {
		return "", errors.New("Not enough lines in CSV")
	}

	// Check that PMC is first column
	if !strings.HasPrefix(quantCSVFileLines[1], "PMC") {
		return "", errors.New("Expected CSV column to be first")
	}

	// Make a lookup for PMCs that we want to include
	pmcLookup := map[int]bool{} // REFACTOR: TODO: Make generic version of utils.SetStringsInMap() for this
	for _, pmc := range roi.PMCs {
		pmcLookup[int(pmc)] = true
	}

	if len(pmcLookup) <= 0 {
		return "", fmt.Errorf("ROI %v contained no PMCs", roi.ID)
	}

	// Start writing file
	csvFileName := makeFileNameWithROI(fileNamePrefix, "map", roi.Name, roi.SharedByName, "csv")
	csv, err := os.Create(path.Join(outDir, csvFileName))
	if err != nil {
		return "", err
	}
	defer csv.Close()

	// Write title & column headings
	_, err = csv.WriteString(quantCSVFileLines[0] + "\n")
	if err != nil {
		return "", err
	}
	_, err = csv.WriteString(quantCSVFileLines[1] + "\n")
	if err != nil {
		return "", err
	}

	// Assumption: The PMCs are in increasing order!
	// Loop through each line and if the PMC is in this ROI, write line to file
	for lineNumber, line := range quantCSVFileLines[2:] {
		commaPos := strings.Index(line, ",")
		if commaPos > 0 {
			pmcStr := strings.Trim(line[0:commaPos], " \t")
			pmc, err := strconv.Atoi(pmcStr)
			if err != nil {
				return "", fmt.Errorf("Map CSV line %v expected PMC at start, got %v", lineNumber, pmcStr)
			}

			if _, ok := pmcLookup[pmc]; ok {
				// Exists in the lookup, write to file!
				_, err = csv.WriteString(line + "\n")
			}
		}
	}

	return csvFileName, nil
}
