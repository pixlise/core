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

// Exposes the interface of the dataset importer aka converter and selecting one automatically based on what
// files are in the folder being imported. The converter supports various formats as delivered by GDS or test
// instruments and this is inteded to be extendable further to other lab instruments and devices in future.
package dataimport

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	dataimportModel "github.com/pixlise/core/v4/api/dataimport/models"
	"github.com/pixlise/core/v4/api/dataimport/sdfToRSI"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func readFromZip(fileInZip *zip.File, outPath string) (string, error) {
	outFullPath := filepath.Join(outPath, path.Base(fileInZip.Name))
	outFile, err := os.OpenFile(outFullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInZip.Mode())
	if err != nil {
		return "", fmt.Errorf("Failed to create output file: %v. Error: %v", outFullPath, err)
	}

	rc, err := fileInZip.Open()
	if err != nil {
		return "", fmt.Errorf("Failed to read %v from zip. Error: %v", fileInZip.Name, err)
	}

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return "", fmt.Errorf("Failed to write zip %v. Error: %v", outFullPath, err)
	}

	// Close the file without defer to close before next iteration of loop
	outFile.Close()
	rc.Close()
	return outFullPath, nil
}

func ProcessEM(importId string, zipReader *zip.Reader, zippedData []byte, destBucket string, s3PathStart string, fs fileaccess.FileAccess, logger logger.ILogger) error {
	// EM data comes not via the FM pipeline, but more direct from the instrument via SDF peeks output format.
	// We search the zip dir tree for:
	// - sdf_raw.txt
	// - All *.msa fils
	// - All *.jpg files (and based on file name decide if they're needed)
	// SDF raw file may contain data from multiple scans, so we have to generate one scan per RTT found in sdf_raw.

	localTemp, sdfLocalPath, msas, images, err := startEMProcess(importId, zipReader, zippedData, logger)
	if err != nil {
		return err
	}

	lowerId := strings.ToLower(importId)
	isCalTarget := strings.Contains(lowerId, "cal-target") || strings.Contains(lowerId, "cal_target") || strings.Contains(lowerId, "caltarget")

	// Create an RSI file from the sdf_raw file
	genFiles, rtts, err := sdfToRSI.ConvertSDFtoRSIs(sdfLocalPath, localTemp)

	if err != nil {
		return fmt.Errorf("Failed to scan %v for RSI creation: %v", sdfLocalPath, err)
	}

	if len(rtts) > 0 && len(genFiles) != len(rtts)*2 {
		return fmt.Errorf("Unexpected file generation count for RSI creation from: %v", sdfLocalPath)
	}

	logger.Infof("Generated RSI & HK files:")
	for _, f := range genFiles {
		logger.Infof("  %v", f)
	}

	rsiUploaded := 0
	for c := 0; c < len(genFiles); c += 2 {
		f := genFiles[c]
		hkFile := genFiles[c+1]

		// Every second file is a HK file not an actual RSI file... make sure we have the right prefix here
		if !strings.HasPrefix(f, "RSI-") || !strings.HasPrefix(hkFile, "HK-") {
			logger.Errorf("ConvertSDFtoRSIs generated : %v. Error: %v", f, err)
			continue
		}

		rxlPath, logPath, surfPath, err := createBeamLocation(isCalTarget, filepath.Join(localTemp, f), rtts[c/2], localTemp, logger)
		if err != nil {
			// Don't fail on errors for these - we may have run beam location tool on some incomplete scan, so failure isn't terrible!
			logger.Errorf("Beam location generation failed for RSI: %v. Error: %v", f, err)
			continue
		}

		// Upoad the output files (beam locations, log and surface)
		files := []string{filepath.Join(localTemp, hkFile), rxlPath, logPath, surfPath}
		name := []string{"housekeeping", "beam location", "log", "surface"}
		for i, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				// Don't fail on errors for these - we may have run beam location tool on some incomplete scan, so failure isn't terrible!
				logger.Errorf("Failed to read generated %v file: %v. Error: %v", name[i], file, err)
				continue
			}

			// Upload
			savePath := path.Join(s3PathStart, path.Base(file))
			err = fs.WriteObject(destBucket, savePath, data)
			if err != nil {
				// We do want to fail here, this isn't an error related to input data - if we have the file, we should be able to upload it
				return err
			}

			logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)
		}

		rsiUploaded++
	}

	if rsiUploaded <= 0 {
		return fmt.Errorf("Failed to generate beam locations from uploaded data")
	}

	// Write the list of images and MSAs we found
	savePath := path.Join(s3PathStart, "images.txt")
	err = fs.WriteObject(destBucket, savePath, []byte(strings.Join(images, "\n")))
	if err != nil {
		return fmt.Errorf("Failed to write image list: %v", err)
	}

	savePath = path.Join(s3PathStart, "msas.txt")
	err = fs.WriteObject(destBucket, savePath, []byte(strings.Join(msas, "\n")))
	if err != nil {
		return fmt.Errorf("Failed to write MSA list: %v", err)
	}

	return nil
}

// Just broken out to make it testable
func startEMProcess(importId string, zipReader *zip.Reader, zippedData []byte, logger logger.ILogger) (string, string, []string, []string, error) {
	// We also have to run the beam location tool ourselves - there isn't one coming from sdf_raw.txt
	localTemp := filepath.Join(os.TempDir(), importId)
	localMSAPath := filepath.Join(localTemp, "msa")
	if err := os.MkdirAll(localMSAPath, 0777); err != nil { // other than 0777 fails in unit tests :(
		return localTemp, "", []string{}, []string{}, fmt.Errorf("Failed to create output MSA path: %v. Error: %v", localMSAPath, err)
	}
	localImagesPath := filepath.Join(localTemp, "images")
	if err := os.MkdirAll(localImagesPath, 0777); err != nil { // other than 0777 fails in unit tests :(
		return localTemp, "", []string{}, []string{}, fmt.Errorf("Failed to create output images path: %v. Error: %v", localImagesPath, err)
	}

	msas := []string{}
	images := []string{}
	sdf_raw_zipPath := ""
	sdfLocalPath := ""

	for _, f := range zipReader.File {
		if strings.Contains(f.Name, "..") {
			return localTemp, sdfLocalPath, msas, images, fmt.Errorf("Found invalid path in zip that references ..: %v", f.Name)
		}

		if !f.FileInfo().IsDir() {
			// Add to list of files we're interested in
			if strings.HasSuffix(f.Name, "sdf_raw.txt") {
				sdf_raw_zipPath = path.Base(f.Name)
				if p, err := readFromZip(f, localTemp); err != nil {
					return localTemp, sdfLocalPath, msas, images, err
				} else {
					sdfLocalPath = p
				}
			} else if strings.HasSuffix(f.Name, ".msa") {
				/*if p, err := readFromZip(f, localMSAPath); err != nil {
					return err
				} else {
					msas = append(msas, path.Base(p))
				}*/
				msas = append(msas, f.Name)
			} else if strings.HasSuffix(f.Name, ".jpg") {
				/*if p, err := readFromZip(f, localImagesPath); err != nil {
					return err
				} else {
					images = append(images, path.Base(p))
				}*/
				images = append(images, f.Name)
			}
		}
	}

	logger.Infof("Found sdf_raw: %v", sdf_raw_zipPath)
	logger.Infof("Found %v images", len(images))
	logger.Infof("Found %v histograms (MSA files)", len(msas))

	// Reject any scans that don't have histograms from the EM
	if len(msas) <= 0 {
		return localTemp, sdfLocalPath, msas, images, fmt.Errorf("No histograms found")
	}
	if len(sdf_raw_zipPath) <= 0 {
		return localTemp, sdfLocalPath, msas, images, fmt.Errorf("No sdf_raw.txt found")
	}

	return localTemp, sdfLocalPath, msas, images, nil
}

func createBeamLocation(isCalTarget bool, rsiPath string, rtt int64, outputBeamLocationPath string, logger logger.ILogger) (string, string, string, error) {
	outSurfaceTop := filepath.Join(outputBeamLocationPath, fmt.Sprintf("surfaceTop-%v.txt", rtt))
	outRXL := filepath.Join(outputBeamLocationPath, fmt.Sprintf("beamLocation-%v.csv", rtt))
	outLog := filepath.Join(outputBeamLocationPath, fmt.Sprintf("log-%v.txt", rtt))

	logger.Infof("Generating beam location CSV from: %v. Is Cal target: %v", rsiPath, isCalTarget)

	bgtPath := "." + string(os.PathSeparator)

	if _, err := os.Stat(bgtPath + "BGT"); err != nil {
		// Try the path used in local testing
		bgtPath = ".." + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "beam-tool" + string(os.PathSeparator)
	}

	if _, err := os.Stat(bgtPath + "Geometry_PIXL_EM_Landing_25Jan2021.csv"); err != nil {
		return "", "", "", errors.New("Calibration file not found")
	}
	if _, err := os.Stat(rsiPath); err != nil {
		return "", "", "", errors.New("RSI not found")
	}

	args := []string{bgtPath + "Geometry_PIXL_EM_Landing_25Jan2021.csv", rsiPath, outSurfaceTop, outRXL}
	if isCalTarget {
		args = append(args, "-t")
	}
	args = append(args, outLog)

	fmt.Printf("Executing: %v %v", bgtPath+"BGT", strings.Join(args, " "))

	cmd := exec.Command(bgtPath+"BGT", args...)

	// var out bytes.Buffer
	// var stderr bytes.Buffer
	// cmd.Stdout = &out
	// cmd.Stderr = &stderr

	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Dir = bgtPath

	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Infof("CombinedOutput:\n%s", out)
		//if err := cmd.Run(); err != nil {
		// Dump std out
		// logger.Infof("BGT stdout:\n" + out.String())
		// logger.Errorf("BGT stderr:\n" + stderr.String())
		return "", "", "", fmt.Errorf("BGT tool error: %v", err)
	} else {
		logger.Infof("CombinedOutput:\n%s", out)
	}

	/*
		procAttr := new(os.ProcAttr)
		procAttr.Files = []*os.File{nil, nil, nil}
		if _, err := os.StartProcess(bgtPath+"BGT", []string{bgtPath + "Geometry_PIXL_EM_Landing_25Jan2021.csv", rsiPath, outSurfaceTop, outRXL, outLog}, procAttr); err != nil {
			return fmt.Errorf("BGT tool error: %v", err)
		}
	*/
	// Make sure we have all output files
	outputs := []string{outSurfaceTop, outRXL, outLog}
	for _, out := range outputs {
		if _, err := os.Stat(out); err != nil {
			return "", "", "", fmt.Errorf("%v not found after BGT tool ran: %v", out, err)
		}
	}

	return outRXL, outLog, outSurfaceTop, nil
}

func ProcessBreadboard(format string, creatorUserId string, datasetID string, zipReader *zip.Reader, zippedData []byte, destBucket string, s3PathStart string, fs fileaccess.FileAccess, logger logger.ILogger) error {
	var err error

	// Expecting flat zip of MSA files
	count := 0
	for _, f := range zipReader.File {
		// If the zip path starts with __MACOSX, ignore it, it's garbage that a mac laptop has included...
		//if strings.HasPrefix(f.Name, "__MACOSX") {
		//	continue
		//}

		if f.FileInfo().IsDir() {
			return fmt.Errorf("Zip file must not contain sub-directories. Found: %v", f.Name)
		}

		if !strings.HasSuffix(f.Name, ".msa") {
			return fmt.Errorf("Zip file must only contain MSA files. Found: %v", f.Name)
		}
		count++
	}

	// Make sure it has at least one msa!
	if count <= 0 {
		return errors.New("Zip file did not contain any MSA files")
	}

	// Save the contents as a zip file in the uploads area
	savePath := path.Join(s3PathStart, "spectra.zip")
	err = fs.WriteObject(destBucket, savePath, zippedData)
	if err != nil {
		return err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)

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

		CreatorUserId: creatorUserId,

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

	if format == "sbu-breadboard" {
		importerFile.Group = "Stony Brook Breadboard"
		importerFile.DetectorConfig = "StonyBrookBreadboard"
	}

	err = fs.WriteJSON(destBucket, savePath, importerFile)
	if err != nil {
		return err
	}
	logger.Infof("  Uploaded: s3://%v/%v", destBucket, savePath)
	return nil
}
