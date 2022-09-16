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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	apiNotifications "github.com/pixlise/core/v2/core/notifications"
	"github.com/pixlise/core/v2/core/utils"
	"github.com/pixlise/core/v2/data-converter/importer"
	"github.com/pixlise/core/v2/data-converter/importer/msatestdata"
	"github.com/pixlise/core/v2/data-converter/importer/pixlfm"
)

// Processes any trigger that we have, and calls import dataset with the right parameters
//
// Triggers Possible:
//
// SNS Message from Dataset Edit page Save button (API endpoint: PUT dataset/meta/datasetID):
//   {"datasetaddons":{"dir": "dataset-addons/datasetID/custom-meta.json", "log": "dataimport-a1b2c3d4e5f6g7h8"}}
// Used when user clicks save button
//
// SNS Message from Dataset reprocess (API endpoint: POST dataset/reprocess/datasetID):
//   datasetID
// NOT USED?
//
// S3 Trigger for object key containing "dataset-addons":
//   S3.Object.Key path
// NOT USED?
//
// S3 Trigger for a zip file in archive? bucket
//   After downloading and unzipping the file it is deleted
// NOT SURE IF THIS IS USED?

/////////////////
// Asking Tom, it seems S3 triggers have gone unused, and we're only triggered 2 ways:
// 1. SNS due to user clicking "save" in dataset edit page
/*
{
    "datasetaddons": {
        "dir": "dataset-addons/189137412/custom-meta.json",
        "log": "dataimport-zmzddoytch2krd7n"
    }
}
*/
// 2. OCS triggering SNS, which has S3 record wrapped inside it:
/*
{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-09-16T09:10:28.417Z",
            "eventName": "ObjectCreated:CompleteMultipartUpload",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "81.154.57.137"
            },
            "responseElements": {
                "x-amz-request-id": "G3QWWT0BAYKP81QK",
                "x-amz-id-2": "qExUWHHDE1nL+UP3zim1XA7FIXRUoKxlIrJt/7ULAtn08/+EvRCt4sChLhCGEqMo7ny4CU/KufMNmOcyZsDPKGWHT2ukMbo+"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "OTBjMjZmYzAtYThlOC00OWRmLWIwMzUtODkyZDk0YmRhNzkz",
                "bucket": {
                    "name": "prodpipeline-rawdata202c7bd0-o40ktu17o2oj",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
                },
                "object": {
                    "key": "189137412-07-09-2022-10-07-57.zip",
                    "size": 54237908,
                    "eTag": "b21ebca14f67255be1cd28c01d494508-7",
                    "sequencer": "0063243D6858D568F0"
                }
            }
        }
    ]
}
*/

func processImportTrigger(trigger awsutil.Record, log logger.ILogger) error {
	isS3 := trigger.EventSource == "aws:s3" // Otherwise assume SNS

	// Work out if it's a "reprocess"
	isReprocess := false
	if isS3 && strings.Contains(trigger.S3.Object.Key, "dataset-addons") ||
		strings.HasPrefix(trigger.SNS.Message, `{"datasetaddons":`) {
		isReprocess = true
	}

	triggerer := "S3"
	if !isS3 {
		triggerer = "SNS"
	}

	op := "process"
	if isReprocess {
		op = "re-process"
	}

	// Get bucket, file path and dataset ID
	datasetID := ""
	sourcePath := ""
	sourceBucket := ""

	// Looks like we can get rid of S3 processing in future...
	if isS3 {
		if isReprocess {
			// Example:
			// NOTE: dataset ID is the first part of the path
			sourcePath = trigger.S3.Object.Key
			splits := strings.Split(trigger.S3.Object.Key, "/")
			datasetID = splits[1]

			// REPROCESS
		} else {
			// Example:
			// NOTE: dataset ID is before the first - in the path
			sourcePath = trigger.S3.Object.Key
			sourceBucket = trigger.S3.Bucket.Name

			splits := strings.Split(trigger.S3.Object.Key, "-")
			datasetID = splits[1]

			// FULL PROCESS

			// Delete the object after pipeline runs
		}
	} else {
		if isReprocess {
			// Example:
			// NOTE: dataset ID is the first part of the path
			var snsMsg APISnsMessage
			err := json.Unmarshal([]byte(trigger.SNS.Message), &snsMsg)
			if err != nil {
				log.Errorf("Failed to read SNS message: %v", err)
			}

			sourcePath = snsMsg.Key.Dir
			splits := strings.Split(snsMsg.Key.Dir, "/")
			datasetID = splits[1]

			// REPROCESS
		} else {
			var e awsutil.Event
			err := e.UnmarshalJSON([]byte(trigger.SNS.Message))
			if err != nil {
				log.Errorf("Issue decoding message: %v", err)
			}
			if e.Records[0].EventSource == "aws:s3" {
				// Example:
				// NOTE: dataset ID is before the first - in the path
				sourcePath = e.Records[0].S3.Object.Key
				sourceBucket = e.Records[0].S3.Bucket.Name

				// FULL PROCESS
			} else if strings.HasPrefix(trigger.SNS.Message, "datasource:") {
				// DO NOTHING???
			} else {
				datasetID = trigger.SNS.Message

				// REPROCESS
			}
		}
	}

	log.Infof("Triggered PIXLISE dataset importer by %v to %v dataset \"%v\"", triggerer, op, datasetID)

	// Initialise stuff
	sess, err := awsutil.GetSession()
	svc, err := awsutil.GetS3(sess)
	if err != nil {
		return err
	}

	tmpprefix, err = ioutil.TempDir("", "archive")
	if err != nil {
		return fmt.Errorf("Failed to create temp directory: %v", err)
	}

	localUnzipPath = path.Join(tmpprefix, "unzippath")
	localInputPath = path.Join(tmpprefix, "inputfiles")
	localArchivePath = path.Join(tmpprefix, "archive")
	localRangesCSVPath = path.Join(tmpprefix, "ranges.csv")

	importer := datasetImporter{
		fs:             fileaccess.MakeS3Access(svc),
		ns:             makeNotificationStack(nil, log),
		rangesPath:     "DatasetConfig/StandardPseudoIntensities.csv",
		sourcePath:     sourcePath,
		sourceBucket:   sourceBucket,
		outPath:        "",
		detectorConfig: "", // Something like: PIXL or breadboard
		datasetID:      datasetID,
		datasetBucket:  os.Getenv("DATASETS_BUCKET"),
	}

	return importer.importData(isReprocess)
}

const s3ArchivePath = "/Archive"

type datasetImporter struct {
	// File access (S3 or local file system)
	fs fileaccess.FileAccess
	// For notifications to be sent when needed
	ns apiNotifications.NotificationManager

	// Source bucket
	sourceBucket string

	// Source path that triggered us
	sourcePath string

	// Dataset bucket (where we output to)
	datasetBucket string

	// Config bucket
	configBucket string

	// Paths on local machine for processing:

	// Download path - where we download zips and other files
	downloadPath string

	// Input path - unzipped, ready to go files which are sent to the dataset converter and other tools
	importPath string

	// Dataset conversion settings:

	// Detector config we're working with. Determines which importer we'll use
	// as in FM vs breadboard
	detectorConfig string

	// Dataset ID
	datasetID string
}

func (i *datasetImporter) importData(isReprocess bool) error {
	// Download the ranges file
	localRangesPath := path.Join(i.downloadPath, "StandardPseudoIntensities.csv")
	err := i.fetchFile(i.configBucket, "DatasetConfig/StandardPseudoIntensities.csv", localRangesPath)
	if err != nil {
		return err
	}

	// If we've been triggered due to a new zip file arriving from OCS, we put this into our archive first
	err = i.archiveFile()
	if err != nil {
		return err
	}

	updateExisting := false
	allthefiles := []string{}

	if !isReprocess {
		err := os.MkdirAll(localUnzipPath, os.ModePerm)
		if err != nil {
			return err
		}
		localFS := fileaccess.FSAccess{}
		err = localFS.EmptyObjects(localUnzipPath)
		if err != nil {
			return err
		}

		inpath := localUnzipPath
		//allthefiles = append(allthefiles, inpath)
		// As this datasource is now in the process flow, copy to the archive folder for re-processing and historical purposes
		jobLog.Infof("----- Copying file %v %v to archive: %v %v -----\n", sourcebucket, name.Inpath, getConfigBucket(), "Datasets/archive/"+name.Inpath)
		err = fs.CopyObject(sourcebucket, name.Inpath, getDatasourceBucket(), "Datasets/archive/"+name.Inpath)
		if err != nil {
			return err
		}
	}

	importers := map[string]importer.Importer{"test-msa": msatestdata.MSATestData{}, "pixl-fm": pixlfm.PIXLFM{}}

	jobLog.Infof("----- Importing pseudo-intensity ranges -----\n")
	err = fetchRanges(getConfigBucket(), name.Rangespath, fs)

	if err != nil {
		return err
	}

	updateExisting = false

	files, err := checkExistingArchive(allthefiles, rtt, &updateExisting, fs, jobLog)

	if isReprocess {
		for _, p := range allthefiles {
			if strings.HasSuffix(p, ".zip") || strings.Contains(p, "zip") {
				jobLog.Infof("Preparing to unzip %v\n", p)
				_, err := utils.UnzipDirectory(p, localUnzipPath)
				if err != nil {
					return err
				}
				inpath := localUnzipPath
				name.Inpath = inpath
			}
		}
	} else {

		if len(files) > 0 {
			updateExisting = true
		}

		// Download the input file from the preprocess bucket -- Should be with the existing archive to ensure order
		jobLog.Infof("----- Importing file %v -----\n", name.Inpath)
		_, err = downloadDirectoryZip(sourcebucket, name.Inpath, fs)
		if err != nil {
			return err
		}

		//Unzip the files into the same folder
		jobLog.Infof("Unzipping all archives...")
		for _, p := range allthefiles {
			if strings.HasSuffix(p, ".zip") || strings.Contains(p, "zip") {
				//fmt.Printf("Preparing to unzip %v\n", p)
				_, err := utils.UnzipDirectory(p, localUnzipPath)
				if err != nil {
					jobLog.Errorf("Unzip failed for \"%v\". Error: \"%v\"\n", p, err)
					return err
				}
				name.Inpath = inpath
			}
		}
	}

	// Get the extra manual stuff
	err = downloadExtraFiles(rtt, fs)

	if err != nil {
		return err
	}

	r, err := processFiles(inpath, name, importers, creationUnixTimeSec, updateExisting, fs, ns, targetbucket, jobLog)
	return err
}

func (i *datasetImporter) archiveFile(bucketFrom string, pathFrom string) error {
	// Work out the file name
	fileName := path.Base(pathFrom)

	return i.fs.CopyObject(bucketFrom, pathFrom, i.datasetBucket, path.Join(s3ArchivePath, fileName))
}

func (i *datasetImporter) fetchFile(bucketFrom string, pathFrom string, savePath string) error {
	bytes, err := i.fs.ReadObject(bucketFrom, pathFrom)
	if err != nil {
		return err
	}

	localFS := fileaccess.FSAccess{}
	return localFS.WriteObject(savePath, "", bytes)
}
