// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
	ccopy "github.com/otiai10/copy"
	"github.com/pixlise/core/core/awsutil"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
	apiNotifications "github.com/pixlise/core/core/notifications"
	"github.com/pixlise/core/core/utils"
	"github.com/pixlise/core/data-converter/output"
	diffractionDetection "github.com/pixlise/core/diffraction-detector"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// setupLocalPaths - Setup the local paths for the files required for datasource processing
func setupLocalPaths() {
	var err error
	tmpprefix, err = ioutil.TempDir("", "archive")
	if err != nil {
		log.Fatal(err)
	}
	localUnzipPath = tmpprefix + "/unzippath"
	localInputPath = tmpprefix + "/inputfiles"
	localArchivePath = tmpprefix + "/archive"
	localRangesCSVPath = tmpprefix + "/ranges.csv"
}

// generatePrefix - Generate the prefix requried for storage in the archive and retrieval
func generatePrefix(name string) string {
	filename := strings.Split(name, ".")
	splits := strings.Split(filename[0], "-")
	return splits[0]
}

// checkExistingArchive - Check the existing archive for any older files already processed for this dataset
func checkExistingArchive(allthefiles []string, name string, updateExisting *bool, fs fileaccess.FileAccess, jobLog logger.ILogger) ([]string, error) {
	prefix := generatePrefix(name)
	paths, err := checkExisting(getDatasourceBucket(), prefix, fs, jobLog)
	if err != nil {
		return allthefiles, err
	}

	for _, p := range paths {
		//set update flag
		*updateExisting = true
		//Download the other parts found
		jobLog.Infof("----- Importing file %v -----\n", p)
		_, err := downloadDirectoryZip(getDatasourceBucket(), p, fs)
		if err != nil {
			return allthefiles, err
		}
		//allthefiles = append(allthefiles, addpath)
	}

	return allthefiles, err

}

// getUpdateNotificationType - Get the notificationtype for a dataset update
func getUpdateNotificationType(datasetID string, bucket string, fs fileaccess.FileAccess) (string, error) {
	datasetSummary, err := datasetModel.ReadDataSetSummary(fs, bucket, datasetID)
	if err != nil {
		return "", err
	}

	diff, err := output.SummaryDiff(datasetSummary, bucket, fs)
	if err != nil {
		return "unknown", err
	}
	if diff.MaxSpectra > 0 || diff.BulkSpectra > 0 || diff.DwellSpectra > 0 || diff.NormalSpectra > 0 {
		return "spectra", nil
	} else if diff.ContextImages > 0 {
		return "image", nil
	} else if diff.DriveID > 0 || diff.Site != "" || diff.Target != "" || diff.Title != "" {
		return "housekeeping", nil
	}
	return "unknown", nil
}

// copyAdditionalDirectories - Copy in additional directories
func copyAdditionalDirectories(outpath string, jobLog logger.ILogger) error {
	dirs := []string{"RGBU", "DISCO", "MATCHED"}
	for _, d := range dirs {
		jobLog.Infof("CHECKING %v EXISTS \n", d)
		if _, err := os.Stat(localInputPath + "/" + d); !os.IsNotExist(err) {
			jobLog.Infof("%v EXISTS COPYING TO ARCHIVE\n", d)
			err := ccopy.Copy(localInputPath+"/"+d, outpath+"/"+d)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// createPeakDiffractoinDB - Use the diffraction engine to calculate the diffraction peaks
func createPeakDiffractionDB(path string, savepath string, jobLog logger.ILogger) error {
	protoParsed, err := datasetModel.ReadDatasetFile(path)
	if err != nil {
		jobLog.Errorf("Failed to open dataset \"%v\": \"%v\"\n", path, err)
		return err
	}

	jobLog.Infof("  Opened %v, got RTT: %v, title %v\n", path, protoParsed.Rtt, protoParsed.Title)

	fmt.Println("  Scanning dataset for diffraction peaks...")
	datasetPeaks, err := diffractionDetection.ScanDataset(protoParsed)
	if err != nil {
		jobLog.Errorf("Error Encoundered During Scanning: %v\n", err)
		return err
	}

	fmt.Println("  Completed scan successfully")

	if savepath != "" {
		jobLog.Infof("  Saving diffraction db file: %v\n", savepath)
		diffractionPB := diffractionDetection.BuildDiffractionProtobuf(protoParsed, datasetPeaks)
		err := diffractionDetection.SaveDiffractionProtobuf(diffractionPB, savepath)
		if err != nil {
			jobLog.Errorf("Error Encoundered During Saving: %v\n", err)
			return err
		}

		fmt.Println("  Diffraction db saved successfully")
	}

	return nil
}

// checkExisting - Check existing files in S3
func checkExisting(bucket string, prefix string, fs fileaccess.FileAccess, jobLog logger.ILogger) ([]string, error) {
	jobLog.Infof("----- Checking for other files -----\n")
	files, err := fs.ListObjects(bucket, "archive/"+prefix)
	if err != nil {
		return nil, err
	}
	m := make(map[int]string)
	var keys []string
	if files != nil && len(files) > 0 {
		for _, f := range files {
			splits := strings.SplitN(f, "-", 2)
			timestamp := strings.Split(splits[1], ".")[0]

			layout := "02-01-2006-15-04-05"
			t, err := time.Parse(layout, timestamp)
			if err != nil {
			}
			m[int(utils.AbsI64(t.Unix()))] = f
		}
		key := make([]int, 0, len(m))
		for k := range m {
			key = append(key, k)
		}
		sort.Ints(key)

		for _, k := range key {
			fmt.Println(k, m[k])
			keys = append(keys, m[k])
		}
	}
	jobLog.Infof("Number of other files found: %v\n", len(keys))
	jobLog.Infof("Found file names: \n")
	for _, j := range keys {
		jobLog.Infof("Filename: %v \n", j)
	}
	jobLog.Infof("End of filenames \n")
	return keys, nil
}

// importAutoQuickLook - Import the quicklook files.
func importAutoQuickLook(path string) {
	files, err := checkLocalExisting("APIX", localUnzipPath)
	if err != nil {
		// REFACTOR: found this empty, shouldn't we error check something?
	}

	for _, i := range files {
		filename := filepath.Base(i)
		os.Rename(path, path+"/"+filename)
	}
}

// checkLocalExisting - Check for local existing files
func checkLocalExisting(prefix string, path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		fn := filepath.Base(path)
		if strings.HasPrefix(fn, prefix) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// makeNotificationStack - Create a notification stack
func makeNotificationStack(fs fileaccess.FileAccess) apiNotifications.NotificationManager {
	return &apiNotifications.NotificationStack{
		Notifications: []apiNotifications.UINotificationObj{},
		FS:            fs,
		Track:         cmap.New(), //make(map[string]bool),
		Bucket:        os.Getenv("notificationBucket"),
		Environment:   "prod",
		Logger:        logger.NullLogger{},
	}
}

// createLogger - Create a logger
func createLogger(makeLog bool) logger.ILogger {
	var jobLog logger.ILogger
	jobID := utils.RandStringBytesMaskImpr(16)
	if !makeLog {
		// Creator doesn't want it logged - used for unit tests so we don't have to set up AWS credentials
		jobLog = logger.NullLogger{}
	} else {
		var err error
		var loglevel = logger.LogDebug
		sess, _ := awsutil.GetSession()
		fmt.Printf("Creating CloudwatchLogger\n")
		t := time.Now()
		ti := fmt.Sprintf(t.Format("20060102150405"))
		jobLog, err = logger.Init("dataimport-"+ti+jobID, loglevel, "prod", sess)
		if err != nil {
			fmt.Printf("Failed to create logger for Job ID: %v\n %v\n", jobID, err)
		}
	}
	jobLog.Infof("==============================")
	jobLog.Infof("=  PIXLISE dataset importer  =")
	jobLog.Infof("==============================")
	return jobLog
}

// downloadExtraFile - Download addon files
func downloadExtraFiles(rtt string, fs fileaccess.FileAccess) error {
	fmt.Printf("Downloading addons\n")
	a, err := fs.ListObjects(getManualBucket(), "dataset-addons/"+rtt)
	if err != nil {
		return err
	}
	if a != nil {
		for _, obj := range a {
			fmt.Printf("Processing addon: %v\n", obj)
			bytes, err := fs.ReadObject(getManualBucket(), obj)
			if err != nil {
				return err
			}
			objpath := obj
			splits := strings.Split(objpath, "/")
			filename := splits[len(splits)-1]
			splits = splits[:len(splits)-1]
			splits = splits[2:]
			newpath := strings.Join(splits, "/")
			newpath = localUnzipPath + "/" + newpath
			os.MkdirAll(newpath, 0755)
			writepath := newpath + "/" + filename
			fmt.Printf("Writing to path: %v\n", writepath)
			err = ioutil.WriteFile(writepath, bytes, 0644)
			if err != nil {
				fmt.Printf("Couldn't write custom meta")
			}
		}
	}
	return nil
}

// fetchRanges - Fetch the ranges files
func fetchRanges(s3bucket string, s3path string, fs fileaccess.FileAccess) error {
	bytes, err := fs.ReadObject(s3bucket, s3path)
	if err != nil {
		return err
	}

	f, err := os.Create(localRangesCSVPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}
	f.Sync()

	return nil
}
