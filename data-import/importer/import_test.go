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

package importer

import (
	"fmt"
	"os"
	"strings"

	"github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
)

func initTest(testDir string) (fileaccess.FileAccess, *logger.StdOutLoggerForTest, string, string, string, string) {
	remoteFS := &fileaccess.FSAccess{}
	log := &logger.StdOutLoggerForTest{}
	envName := "unit-test"
	configBucket := "./test-data/" + testDir + "/config-bucket"
	datasetBucket := "./test-data/" + testDir + "/dataset-bucket"
	manualBucket := "./test-data/" + testDir + "/manual-bucket"

	return remoteFS, log, envName, configBucket, datasetBucket, manualBucket
}

// Import unknown dataset (simulate trigger by OCS pipeline), file goes to archive, then all files downloaded from archive, dataset create fails due to unknown data type
func Example_ImportForTrigger_OCS_Archive_BadData() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Archive_BadData")

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

	_, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	// Ensure these log msgs appeared...
	requiredLogs := []string{
		"Downloading archived zip files...",
		"Downloaded 2 zip files, unzipped 6 files",
		"Downloading pseudo-intensity ranges...",
		"Downloading user customisation files...",
		"Failed to determine dataset type to import.",
	}

	for _, msg := range requiredLogs {
		fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))
	}

	// Output:
	// Errors: Failed to determine dataset type to import.
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 2 zip files, unzipped 6 files": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Failed to determine dataset type to import.": true
}

// Import FM-style (simulate trigger by OCS pipeline), file already in archive, so should do nothing
func Example_ImportForTrigger_OCS_Archive_Exists() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Archive_Exists")
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

	_, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	msg := "Archiving source file: \"s3://./test-data/Archive_Exists/raw-data-bucket/70000_069-02-09-2021-06-25-13.zip\""
	fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))

	fmt.Printf("Log shows exists in archive: %v\n", strings.Contains(log.LastLogLine(), "File already exists in archive, processing stopped. File was: \"70000_069-02-09-2021-06-25-13.zip\""))

	// Output:
	// Errors: <nil>
	// Logged "Archiving source file: "s3://./test-data/Archive_Exists/raw-data-bucket/70000_069-02-09-2021-06-25-13.zip"": true
	// Log shows exists in archive: true
}

func printArchiveOKLogOutput(log *logger.StdOutLoggerForTest, datasetBucket string, remoteFS fileaccess.FileAccess) {
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
		fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))
	}

	// Dump contents of summary file, this verifies most things imported as expected
	summary, err := dataset.ReadDataSetSummary(remoteFS, datasetBucket, "048300551")
	if err != nil {
		fmt.Println("Failed to read dataset summary file")
	}
	// Clear the time stamp so it doesn't change next time we run test
	summary.CreationUnixTimeSec = 0
	fmt.Printf("%+v\n", summary)
}

// Import FM-style (simulate trigger by OCS pipeline), file goes to archive, then all files downloaded from archive and dataset created
func Example_ImportForTrigger_OCS_Archive_OK() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Archive_OK")
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

	_, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	printArchiveOKLogOutput(log, datasetBucket, remoteFS)

	// Output:
	// Errors: <nil>
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
	// {DatasetID:048300551 Group:PIXL-FM DriveID:1712 SiteID:4 TargetID:? Site: Target: Title:Naltsos SOL:0125 RTT:048300551 SCLK:678031418 ContextImage:PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png LocationCount:133 DataFileSize:843541 ContextImages:5 TIFFContextImages:1 NormalSpectra:242 DwellSpectra:0 BulkSpectra:2 MaxSpectra:2 PseudoIntensities:121 DetectorConfig:PIXL CreationUnixTimeSec:0}
}

// Import FM-style (simulate trigger by dataset edit screen), should create dataset with custom name+image
func Example_ImportForTrigger_OCS_DatasetEdit() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Archive_OK")

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
	"logID": "dataimport-unittest123"
}`

	_, err = ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	printArchiveOKLogOutput(log, datasetBucket, remoteFS)

	// Output:
	// Errors: <nil>
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
	// {DatasetID:048300551 Group:PIXL-FM DriveID:1712 SiteID:4 TargetID:? Site: Target: Title:Naltsos SOL:0125 RTT:048300551 SCLK:678031418 ContextImage:PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png LocationCount:133 DataFileSize:843541 ContextImages:5 TIFFContextImages:1 NormalSpectra:242 DwellSpectra:0 BulkSpectra:2 MaxSpectra:2 PseudoIntensities:121 DetectorConfig:PIXL CreationUnixTimeSec:0}
}

func printManualOKLogOutput(log *logger.StdOutLoggerForTest, datasetBucket string, remoteFS fileaccess.FileAccess) {
	// Ensure these log msgs appeared...
	requiredLogs := []string{
		"Downloading archived zip files...",
		"Downloaded 0 zip files, unzipped 0 files",
		"No zip files found in archive, dataset may have been manually uploaded. Trying to download...",
		"Dataset test1234 downloaded from manual upload area",
		"Downloading pseudo-intensity ranges...",
		"Downloading user customisation files...",
		"Reading 1261 files from spectrum directory...",
		"Reading spectrum [1135/1260] 90%",
		"PMC 1261 has 4 MSA/spectrum entries",
		"WARNING: No main context image determined",
		"Diffraction db saved successfully",
		"Warning: No import.json found, defaults will be used",
	}

	for _, msg := range requiredLogs {
		fmt.Printf("Logged \"%v\": %v\n", msg, log.LogContains(msg))
	}

	// Dump contents of summary file, this verifies most things imported as expected
	summary, err := dataset.ReadDataSetSummary(remoteFS, datasetBucket, "test1234")
	if err != nil {
		fmt.Println("Failed to read dataset summary file")
	}
	// Clear the time stamp so it doesn't change next time we run test
	summary.CreationUnixTimeSec = 0
	fmt.Printf("%+v\n", summary)
}

// Import a breadboard dataset from manual uploaded zip file
func Example_ImportForTrigger_Manual() {
	remoteFS, log, envName, configBucket, datasetBucket, manualBucket := initTest("Manual_OK")

	trigger := `{
	"datasetID": "test1234",
	"logID": "dataimport-unittest123"
}`

	_, err := ImportForTrigger([]byte(trigger), envName, configBucket, datasetBucket, manualBucket, log, remoteFS)

	fmt.Printf("Errors: %v\n", err)

	printManualOKLogOutput(log, datasetBucket, remoteFS)

	// Output:
	// Errors: <nil>
	// Logged "Downloading archived zip files...": true
	// Logged "Downloaded 0 zip files, unzipped 0 files": true
	// Logged "No zip files found in archive, dataset may have been manually uploaded. Trying to download...": true
	// Logged "Dataset test1234 downloaded from manual upload area": true
	// Logged "Downloading pseudo-intensity ranges...": true
	// Logged "Downloading user customisation files...": true
	// Logged "Reading 1261 files from spectrum directory...": true
	// Logged "Reading spectrum [1135/1260] 90%": true
	// Logged "PMC 1261 has 4 MSA/spectrum entries": true
	// Logged "WARNING: No main context image determined": true
	// Logged "Diffraction db saved successfully": true
	// Logged "Warning: No import.json found, defaults will be used": true
	// {DatasetID:test1234 Group:JPL Breadboard DriveID:0 SiteID:0 TargetID:0 Site: Target: Title:test1234 SOL: RTT: SCLK:0 ContextImage: LocationCount:1261 DataFileSize:6786781 ContextImages:0 TIFFContextImages:0 NormalSpectra:2520 DwellSpectra:0 BulkSpectra:2 MaxSpectra:2 PseudoIntensities:0 DetectorConfig:Breadboard CreationUnixTimeSec:0}
}

/* NOT TESTED YET, because it's not done yet!

// Import a breadboard dataset from manual uploaded zip file, including custom name+image
func Example_ImportForTrigger_Manual_DatasetEdit() {
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
