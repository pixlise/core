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

package dataimport

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/encoding/protojson"
)

func initTest(testDir string, autoShareCreatorId string, autoShareCreatorGroupEditor string) (fileaccess.FileAccess, *logger.StdOutLoggerForTest, string, string, string, string, *mongo.Database) {
	remoteFS := &fileaccess.FSAccess{}
	log := &logger.StdOutLoggerForTest{}
	envName := "unit-test"
	configBucket := "./test-data/" + testDir + "/config-bucket"
	datasetBucket := "./test-data/" + testDir + "/dataset-bucket"
	manualBucket := "./test-data/" + testDir + "/manual-bucket"

	db := wstestlib.GetDB()
	ctx := context.TODO()

	// Clear relevant collections
	db.Collection(dbCollections.ImagesName).Drop(ctx)
	db.Collection(dbCollections.ScansName).Drop(ctx)
	db.Collection(dbCollections.ScanDefaultImagesName).Drop(ctx)
	db.Collection(dbCollections.ScanAutoShareName).Drop(ctx)

	// Insert an item if configured to
	if len(autoShareCreatorId) > 0 {
		item := protos.ScanAutoShareEntry{
			Id: autoShareCreatorId,
			Editors: &protos.UserGroupList{
				GroupIds: []string{autoShareCreatorGroupEditor},
			},
		}

		db.Collection(dbCollections.ScanAutoShareName).InsertOne(ctx, &item)
	}

	return remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db
}

/*
func startTestWithMockMongo(name string, t *testing.T, testFunc func(mt *mtest.T)) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run(name, testFunc)
}
*/
// Import unknown dataset (simulate trigger by OCS pipeline), file goes to archive, then all files downloaded from archive, dataset create fails due to unknown data type
func Example_importForTrigger_OCS_Archive_BadData() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Archive_BadData", specialUserIds.PIXLISESystemUserId, "PIXLFMGroupId")

	// In case it ran before, delete the file from dataset bucket, otherwise we will end for the wrong reason
	os.Remove(datasetBucket + "/Archive/70000_069-02-09-2021-06-25-13.zip")

	trigger := `{
	"Records": [
		{
			"eventVersion": "2.1",
			"eventSource": "aws:s3",
			"awsRegion": "us-east-1",
			"eventTime": "2022-10-16T22:07:40.929Z",
			"eventName": "ObjectCreated:CompleteMultipartUpload",
			"userIdentity": {
				"principalId": "AWS:123"
			},
			"requestParameters": {
				"sourceIPAddress": "3.213.168.4"
			},
			"responseElements": {
				"x-amz-request-id": "234",
				"x-amz-id-2": "345+678"
			},
			"s3": {
				"s3SchemaVersion": "1.0",
				"configurationId": "id1234",
				"bucket": {
					"name": "./test-data/Archive_BadData/raw-data-bucket",
					"ownerIdentity": {
						"principalId": "AP902Y0PI20DF"
					},
					"arn": "arn:aws:s3:::raw-data-bucket"
				},
				"object": {
					"key": "70000_069-02-09-2021-06-25-13.zip",
					"size": 602000,
					"eTag": "1234567890",
					"sequencer": "00112233445566"
				}
			}
		}
	]
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	// Ensure these log msgs appeared...
	requiredLogs := []string{
		"Downloading archived zip files...",
		"Downloaded 2 zip files, unzipped 6 files",
		"Downloading pseudo-intensity ranges...",
		"Downloading user customisation files...",
		"SelectDataConverter: Path contains 3 files...",
		"Failed to open detector.json when determining dataset type",
	}

	for _, msg := range requiredLogs {
		fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))
	}

	// Output:
	// Errors: Failed to determine dataset type to import., changes: , isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 2 zip files, unzipped 6 files": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "SelectDataConverter: Path contains 3 files...": true
	// Logged "Failed to open detector.json when determining dataset type": true
}

// Import FM-style (simulate trigger by OCS pipeline), file already in archive, so should do nothing
func Example_importForTrigger_OCS_Archive_Exists() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Archive_Exists", specialUserIds.PIXLISESystemUserId, "PIXLFMGroupId")
	trigger := `{
	"Records": [
		{
			"eventVersion": "2.1",
			"eventSource": "aws:s3",
			"awsRegion": "us-east-1",
			"eventTime": "2022-10-16T22:07:40.929Z",
			"eventName": "ObjectCreated:CompleteMultipartUpload",
			"userIdentity": {
				"principalId": "AWS:123"
			},
			"requestParameters": {
				"sourceIPAddress": "3.213.168.4"
			},
			"responseElements": {
				"x-amz-request-id": "234",
				"x-amz-id-2": "345+678"
			},
			"s3": {
				"s3SchemaVersion": "1.0",
				"configurationId": "id1234",
				"bucket": {
					"name": "./test-data/Archive_Exists/raw-data-bucket",
					"ownerIdentity": {
						"principalId": "AP902Y0PI20DF"
					},
					"arn": "arn:aws:s3:::raw-data-bucket"
				},
				"object": {
					"key": "70000_069-02-09-2021-06-25-13.zip",
					"size": 602000,
					"eTag": "1234567890",
					"sequencer": "00112233445566"
				}
			}
		}
	]
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	msg := "Archiving source file: \"s3://./test-data/Archive_Exists/raw-data-bucket/70000_069-02-09-2021-06-25-13.zip\""
	fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))

	fmt.Printf("Log shows exists in archive: %v\n", strings.Contains(log.LastLogLine(), "File already exists in archive, processing stopped. File was: \"70000_069-02-09-2021-06-25-13.zip\""))

	// Output:
	// Errors: <nil>, changes: , isUpdate: false
	// Logged "Archiving source file: "s3://./test-data/Archive_Exists/raw-data-bucket/70000_069-02-09-2021-06-25-13.zip"": true
	// Log shows exists in archive: true
}

func printArchiveOKLogOutput(logger *logger.StdOutLoggerForTest, db *mongo.Database) {
	// Ensure these log msgs appeared...
	requiredLogs := []string{
		"Downloading archived zip files...",
		"Downloaded 20 zip files, unzipped 364 files",
		"Downloading pseudo-intensity ranges...",
		"Downloading user customisation files...",
		"This dataset's detector config is PIXL",
		"PMC 218 has 4 MSA/spectrum entries",
		"Main context image: PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
		"Diffraction db saved successfully",
		"Applying custom title: Naltsos",
		"Matched aligned image: PCCR0577_0718181212_000MSA_N029000020073728500030LUD01.tif, offset(0, 0), scale(1, 1). Match for aligned index: 0",
	}

	for _, msg := range requiredLogs {
		fmt.Printf("Logged \"%v\": %v\n", msg, logger.LogContains(msg))
	}

	// Dump contents of summary file, this verifies most things imported as expected
	summary, err := scan.ReadScanItem("048300551", db)
	if err != nil {
		fmt.Println("Failed to read dataset summary file")
		return
	}
	// Clear the time stamp so it doesn't change next time we run test
	summary.TimestampUnixSec = 0
	summary.CompleteTimeStampUnixSec = 0

	b, err := protojson.Marshal(summary)
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("%v|%v\n", err, utils.MakeDeterministicJSON(b, true))
}

// Import FM-style (simulate trigger by OCS pipeline), file goes to archive, then all files downloaded from archive and dataset created
func Example_importForTrigger_OCS_Archive_OK() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Archive_OK", specialUserIds.PIXLISESystemUserId, "PIXLFMGroupId")
	// In case it ran before, delete the file from dataset bucket, otherwise we will end for the wrong reason
	os.Remove(datasetBucket + "/Archive/048300551-27-06-2021-09-52-25.zip")

	trigger := `{
	"Records": [
		{
			"eventVersion": "2.1",
			"eventSource": "aws:s3",
			"awsRegion": "us-east-1",
			"eventTime": "2022-10-16T22:07:40.929Z",
			"eventName": "ObjectCreated:CompleteMultipartUpload",
			"userIdentity": {
				"principalId": "AWS:123"
			},
			"requestParameters": {
				"sourceIPAddress": "3.213.168.4"
			},
			"responseElements": {
				"x-amz-request-id": "234",
				"x-amz-id-2": "345+678"
			},
			"s3": {
				"s3SchemaVersion": "1.0",
				"configurationId": "id1234",
				"bucket": {
					"name": "./test-data/Archive_OK/raw-data-bucket",
					"ownerIdentity": {
						"principalId": "AP902Y0PI20DF"
					},
					"arn": "arn:aws:s3:::raw-data-bucket"
				},
				"object": {
					"key": "048300551-27-06-2021-09-52-25.zip",
					"size": 602000,
					"eTag": "1234567890",
					"sequencer": "00112233445566"
				}
			}
		}
	]
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printArchiveOKLogOutput(log, db)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 20 zip files, unzipped 364 files": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "This dataset's detector config is PIXL": true
	// Logged "PMC 218 has 4 MSA/spectrum entries": true
	// Logged "Main context image: PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Applying custom title: Naltsos": true
	// Logged "Matched aligned image: PCCR0577_0718181212_000MSA_N029000020073728500030LUD01.tif, offset(0, 0), scale(1, 1). Match for aligned index: 0": true
	// <nil>|{"contentCounts": {"BulkSpectra": 2,"DwellSpectra": 0,"MaxSpectra": 2,"NormalSpectra": 242,"PseudoIntensities": 121},"creatorUserId": "PIXLISEImport","dataTypes": [{"count": 5,"dataType": "SD_IMAGE"},{"count": 1,"dataType": "SD_RGBU"},{"count": 242,"dataType": "SD_XRF"}],"id": "048300551","instrument": "PIXL_FM","instrumentConfig": "PIXL","meta": {"DriveId": "1712","RTT": "048300551","SCLK": "678031418","Site": "","SiteId": "4","Sol": "0125","Target": "","TargetId": "?"},"title": "Naltsos"}
}

// Import FM-style (simulate trigger by dataset edit screen), should create dataset with custom name+image
func Example_importForTrigger_OCS_DatasetEdit() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Archive_OK", specialUserIds.PIXLISESystemUserId, "PIXLFMGroupId")

	// To save from checking in 2 sets of the same zip files for this and Example_ImportForTrigger_OCS_Archive_OK, here we copy
	// the archive files from the Archive_OK test to here.
	// NOTE: This test doesn't get triggered by the arrival of a new archive file, so we have to copy the "new" file from
	// the Archive_OK raw bucket separately
	err := fileaccess.CopyFileLocally("./test-data/Archive_OK/raw-data-bucket/048300551-27-06-2021-09-52-25.zip", datasetBucket+"/Archive/048300551-27-06-2021-09-52-25.zip")
	if err != nil {
		fmt.Println("Failed to copy new archive file")
	}

	localFS := fileaccess.FSAccess{}
	archiveFiles, err := localFS.ListObjects("./test-data/Archive_OK/dataset-bucket/Archive/", "")
	if err != nil {
		fmt.Println("Failed to copy archive from OK test to Edit test")
	}
	for _, fileName := range archiveFiles {
		if strings.HasSuffix(fileName, ".zip") { // Guard from .DS_Store and other garbage
			err = fileaccess.CopyFileLocally("./test-data/Archive_OK/dataset-bucket/Archive/"+fileName, datasetBucket+"/Archive/"+fileName)
			if err != nil {
				fmt.Println("Failed to copy archive from OK test to Edit test")
			}
		}
	}

	trigger := `{
	"datasetID": "048300551",
	"jobID": "dataimport-unittest123"
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printArchiveOKLogOutput(log, db)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: true
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 20 zip files, unzipped 364 files": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "This dataset's detector config is PIXL": true
	// Logged "PMC 218 has 4 MSA/spectrum entries": true
	// Logged "Main context image: PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Applying custom title: Naltsos": true
	// Logged "Matched aligned image: PCCR0577_0718181212_000MSA_N029000020073728500030LUD01.tif, offset(0, 0), scale(1, 1). Match for aligned index: 0": true
	// <nil>|{"contentCounts": {"BulkSpectra": 2,"DwellSpectra": 0,"MaxSpectra": 2,"NormalSpectra": 242,"PseudoIntensities": 121},"creatorUserId": "PIXLISEImport","dataTypes": [{"count": 5,"dataType": "SD_IMAGE"},{"count": 1,"dataType": "SD_RGBU"},{"count": 242,"dataType": "SD_XRF"}],"id": "048300551","instrument": "PIXL_FM","instrumentConfig": "PIXL","meta": {"DriveId": "1712","RTT": "048300551","SCLK": "678031418","Site": "","SiteId": "4","Sol": "0125","Target": "","TargetId": "?"},"title": "Naltsos"}
}

func printManualOKLogOutput(log *logger.StdOutLoggerForTest, db *mongo.Database, datasetId string, fileCount uint32) {
	// Ensure these log msgs appeared...
	requiredLogs := []string{
		"Downloading archived zip files...",
		"Downloaded 0 zip files, unzipped 0 files",
		"No zip files found in archive, dataset may have been manually uploaded. Trying to download...",
		fmt.Sprintf("Dataset %v downloaded %v files from manual upload area", datasetId, fileCount),
		"Downloading pseudo-intensity ranges...",
		"Downloading user customisation files...",
		"Reading 1261 files from spectrum directory...",
		"Reading spectrum [1135/1260] 90%",
		"PMC 1261 has 4 MSA/spectrum entries",
		"WARNING: No main context image determined",
		"Diffraction db saved successfully",
		"Warning: No import.json found, defaults will be used",
		"No auto-share destination found, so only importing user will be able to access this dataset.",
	}

	for _, msg := range requiredLogs {
		fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))
	}

	// Dump contents of summary file, this verifies most things imported as expected
	summary, err := scan.ReadScanItem(datasetId, db)
	if err != nil {
		fmt.Println("Failed to read dataset summary file")
	} else {
		// Clear the time stamp so it doesn't change next time we run test
		summary.TimestampUnixSec = 0
		summary.CompleteTimeStampUnixSec = 0
	}

	b, err := protojson.Marshal(summary)
	s := strings.ReplaceAll(string(b), " ", "")
	fmt.Printf("%v|%v\n", err, s)
}

// Import a breadboard dataset from manual uploaded zip file
func Example_importForTrigger_Manual_JPL() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Manual_OK", specialUserIds.JPLImport, "JPLTestUserGroupId")

	trigger := `{
	"datasetID": "test1234",
	"jobID": "dataimport-unittest123"
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printManualOKLogOutput(log, db, "test1234", 3)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 0 zip files, unzipped 0 files": true
	// Logged "No zip files found in archive, dataset may have been manually uploaded. Trying to download...": true
	// Logged "Dataset test1234 downloaded 3 files from manual upload area": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Reading 1261 files from spectrum directory...": true
	// Logged "Reading spectrum [1135/1260] 90%": true
	// Logged "PMC 1261 has 4 MSA/spectrum entries": true
	// Logged "WARNING: No main context image determined": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Warning: No import.json found, defaults will be used": true
	// Logged "No auto-share destination found, so only importing user will be able to access this dataset.": false
	// <nil>|{"id":"test1234","title":"test1234","dataTypes":[{"dataType":"SD_XRF","count":2520}],"instrument":"JPL_BREADBOARD","instrumentConfig":"Breadboard","meta":{"DriveId":"0","RTT":"","SCLK":"0","Site":"","SiteId":"0","Sol":"","Target":"","TargetId":"0"},"contentCounts":{"BulkSpectra":2,"DwellSpectra":0,"MaxSpectra":2,"NormalSpectra":2520,"PseudoIntensities":0},"creatorUserId":"JPLImport"}
}

// Import a breadboard dataset from manual uploaded zip file
func Example_importForTrigger_Manual_SBU() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Manual_OK2", specialUserIds.SBUImport, "SBUTestUserGroupId")

	trigger := `{
	"datasetID": "test1234sbu",
	"jobID": "dataimport-unittest123sbu"
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printManualOKLogOutput(log, db, "test1234sbu", 4)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 0 zip files, unzipped 0 files": true
	// Logged "No zip files found in archive, dataset may have been manually uploaded. Trying to download...": true
	// Logged "Dataset test1234sbu downloaded 4 files from manual upload area": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Reading 1261 files from spectrum directory...": true
	// Logged "Reading spectrum [1135/1260] 90%": true
	// Logged "PMC 1261 has 4 MSA/spectrum entries": true
	// Logged "WARNING: No main context image determined": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Warning: No import.json found, defaults will be used": false
	// Logged "No auto-share destination found, so only importing user will be able to access this dataset.": false
	// <nil>|{"id":"test1234sbu","title":"test1234sbu","dataTypes":[{"dataType":"SD_XRF","count":2520}],"instrument":"SBU_BREADBOARD","instrumentConfig":"StonyBrookBreadboard","meta":{"DriveId":"0","RTT":"","SCLK":"0","Site":"","SiteId":"0","Sol":"","Target":"","TargetId":"0"},"contentCounts":{"BulkSpectra":2,"DwellSpectra":0,"MaxSpectra":2,"NormalSpectra":2520,"PseudoIntensities":0},"creatorUserId":"SBUImport"}
}

// Import a breadboard dataset from manual uploaded zip file
func Example_ImportForTrigger_Manual_SBU_NoAutoShare() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Manual_OK2", specialUserIds.JPLImport, "JPLTestUserGroupId")

	trigger := `{
	"datasetID": "test1234sbu",
	"jobID": "dataimport-unittest123sbu"
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printManualOKLogOutput(log, db, "test1234sbu", 4)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 0 zip files, unzipped 0 files": true
	// Logged "No zip files found in archive, dataset may have been manually uploaded. Trying to download...": true
	// Logged "Dataset test1234sbu downloaded 4 files from manual upload area": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Reading 1261 files from spectrum directory...": true
	// Logged "Reading spectrum [1135/1260] 90%": true
	// Logged "PMC 1261 has 4 MSA/spectrum entries": true
	// Logged "WARNING: No main context image determined": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Warning: No import.json found, defaults will be used": false
	// Logged "No auto-share destination found, so only importing user will be able to access this dataset.": true
	// <nil>|{"id":"test1234sbu","title":"test1234sbu","dataTypes":[{"dataType":"SD_XRF","count":2520}],"instrument":"SBU_BREADBOARD","instrumentConfig":"StonyBrookBreadboard","meta":{"DriveId":"0","RTT":"","SCLK":"0","Site":"","SiteId":"0","Sol":"","Target":"","TargetId":"0"},"contentCounts":{"BulkSpectra":2,"DwellSpectra":0,"MaxSpectra":2,"NormalSpectra":2520,"PseudoIntensities":0},"creatorUserId":"SBUImport"}
}

/* Didnt get this working when the above was changed. Problem is this still generates the user name: SBUImport, so the
   premise of the test fails because it doesn't end up with no user id at that point!
func Test_ImportForTrigger_Manual_SBU_NoAutoShare_FailForPipeline(t *testing.T) {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("Manual_OK2", "", "")

	trigger := `{
	"datasetID": "test1234sbu",
	"jobID": "dataimport-unittest123sbu"
}`

	_, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	// Make sure we got the error
	if !strings.HasSuffix(err.Error(), "Cannot work out groups to auto-share imported dataset with") {
		t.Errorf("ImportForTrigger didnt return expected error")
	}
}
*/
// Import a breadboard dataset from manual uploaded zip file
func Example_importForTrigger_Manual_EM() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket, db := initTest("ManualEM_OK", specialUserIds.PIXLISESystemUserId, "PIXLFMGroupId")

	trigger := `{
	"datasetID": "048300551",
	"jobID": "dataimport-unittest048300551"
}`

	result, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, db, log, remoteFS)

	fmt.Printf("Errors: %v, changes: %v, isUpdate: %v\n", err, result.WhatChanged, result.IsUpdate)

	printManualOKLogOutput(log, db, "048300551", 3)

	// Output:
	// Errors: <nil>, changes: unknown, isUpdate: false
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 0 zip files, unzipped 0 files": true
	// Logged "No zip files found in archive, dataset may have been manually uploaded. Trying to download...": true
	// Logged "Dataset 048300551 downloaded 3 files from manual upload area": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Reading 1261 files from spectrum directory...": false
	// Logged "Reading spectrum [1135/1260] 90%": false
	// Logged "PMC 1261 has 4 MSA/spectrum entries": false
	// Logged "WARNING: No main context image determined": false
	// Logged "Diffraction db saved successfully": true
	// Logged "Warning: No import.json found, defaults will be used": false
	// Logged "No auto-share destination found, so only importing user will be able to access this dataset.": false
	// <nil>|{"id":"048300551","title":"048300551","dataTypes":[{"dataType":"SD_IMAGE","count":4},{"dataType":"SD_XRF","count":242}],"instrument":"PIXL_EM","instrumentConfig":"PIXL-EM-E2E","meta":{"DriveId":"1712","RTT":"048300551","SCLK":"678031418","Site":"","SiteId":"4","Sol":"0125","Target":"","TargetId":"?"},"contentCounts":{"BulkSpectra":2,"DwellSpectra":0,"MaxSpectra":2,"NormalSpectra":242,"PseudoIntensities":121},"creatorUserId":"PIXLISEImport"}
}

/* NOT TESTED YET, because it's not done yet!

// Import a breadboard dataset from manual uploaded zip file, including custom name+image
func Example_importForTrigger_Manual_DatasetEdit() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Manual_Edit")

	trigger := `{
	"datasetID": "test1234",
	"logID": "dataimport-unittest123"
}`

	err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	printManualOKLogOutput(log, datasetBucket, remoteFS)

	// Output:
	// Errors: <nil>
}
*/
