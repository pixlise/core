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
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
	apiNotifications "github.com/pixlise/core/core/notifications"
	"github.com/pixlise/core/core/utils"
)

const testFileCreationUnixTimeSec = 1234567890 // needs to match what's in test-output/summary*.json

func loadFileBytes(path string, t *testing.T) *os.File {
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("s3 Mock setup failed to read file: %v. Error: %v", path, err)
		return nil
	}
	return f
}

func makeTestNotifications(fs fileaccess.FileAccess) apiNotifications.NotificationManager {
	return &apiNotifications.DummyNotificationStack{
		Notifications: []apiNotifications.UINotificationObj{},
		FS:            fs,
		Bucket:        os.Getenv("notificationBucket"),
		Track:         make(map[string]bool),
		Environment:   "prod",
		Logger:        logger.NullLogger{},
	}
}

func TestRunFull(t *testing.T) {
	var mockS3 awsutil.MockS3Client
	// NOTE: directly calling mockS3.FinishTest() at the end to check its return value
	artifactPreProcessedBucket := "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7"
	dir, err := ioutil.TempDir("/tmp", "ds")
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer os.RemoveAll(dir)
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/063111681"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/063111681.zip"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{},
		{},
		{},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(getConfigBucket()), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("063111681.zip"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/full-test-datasource-1/summary.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/063111681.zip", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
	}

	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/063111681.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/063111681.zip"),
		},
	}

	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{}

	// Add each expected upload file operation
	expFiles := []string{
		"full-test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"full-test-datasource-1/dataset.bin",
		"full-test-datasource-1/diffraction-db.bin",
		"full-test-datasource-1/summary.json",
	}
	expExpFilePaths := []string{
		"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/dataset.bin",
		"./test-output/diffraction-db.bin",
		"./test-output/summary.json",
	}

	skipPaths := []string{
		"full-test-datasource-1/diffraction-db.bin",
		"Datasets/full-test-datasource-1/diffraction-db.bin",
	}
	mockS3.SkipPutChecks(skipPaths)
	expBuckets := []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}

	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}

	e := DatasourceEvent{
		Inpath:         "063111681.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	setupLocalPaths()
	fs := fileaccess.MakeS3Access(&mockS3)
	ns := makeTestNotifications(fs)
	str, err := executePipeline(e, fs, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf(err.Error())
	}

	fmt.Printf(str)

	localFS := fileaccess.FSAccess{}
	root := dir
	expectedFiles := []string{
		"full-test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"full-test-datasource-1/dataset.bin",
		"full-test-datasource-1/diffraction-db.bin",
		"full-test-datasource-1/summary.json",
	}

	files, err := localFS.ListObjects(root, "")
	if err != nil {
		t.Errorf("Error finding files.")
	}

	if !utils.StringSlicesEqual(expectedFiles, files) {
		t.Errorf("File list was incorrect, got: %v, want: %v.", files, expectedFiles)
	}

	// This is not an Example test, so we call it directly here and check its return value
	err = mockS3.FinishTest()
	if err != nil {
		t.Errorf("mockS3 reported errors: %v", err)
	}
}

func TestRunLocalTestMissingFilesAppend(t *testing.T) {
	var mockS3 awsutil.MockS3Client
	// NOTE: directly calling mockS3.FinishTest() at the end to check its return value
	os.Setenv("CONFIG_BUCKET", "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7")
	os.Setenv("DATASETS_BUCKET", "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7")
	os.Setenv("MANUAL_BUCKET", "artifactsstack-artifactsmanualuploaddatasourcespi-1m9y4zu1x9vud")
	dir, err := ioutil.TempDir("/tmp", "ds")
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer os.RemoveAll(dir)
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getConfigBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
	}

	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			/*Contents: []*s3.Object{
				{Key: aws.String("archive/summary.json")},
			},*/
		},
		{},
	}
	artifactPreProcessedBucket := "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7"
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-04-35.zip"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-04-35.zip"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/test-datasource-1/summary.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-04-35.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-04-35.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-output/summary1.json", t),
		},
	}

	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-04-35.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-04-35.zip"),
		},
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-05-39.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
	}

	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
		{},
	}

	skipPaths := []string{
		"test-datasource-1/diffraction-db.bin",
		"Datasets/test-datasource-1/diffraction-db.bin",
	}
	mockS3.SkipPutChecks(skipPaths)
	// Add each expected upload file operation
	expFiles := []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}
	expExpFilePaths := []string{
		"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/dataset.bin",
		"./test-output/diffraction-db.bin",
		"./test-output/summary1.json",
	}
	expBuckets := []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}

	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}

	e := DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-04-35.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	setupLocalPaths()

	fs := fileaccess.MakeS3Access(&mockS3)
	ns := makeTestNotifications(fs)
	str, err := executePipeline(e, fs, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	// Expecting an error
	if err.Error() != "Failed to determine dataset RTT" {
		t.Errorf("Unexpected error when executing pipeline")
	}
	if str != "IMPORT ERROR: Failed to determine dataset RTT\n" {
		t.Errorf("Unexpected error text when executing pipeline")
	}

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
	}

	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				//{Key: aws.String("archive/summary.json")},
				{Key: aws.String("archive/060883460-04-08-2021-09-04-35.zip")},
			},
		},
		{},
		{},
	}

	e = DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-05-39.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	setupLocalPaths()
	str, err = executePipeline(e, fs, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf(err.Error())
	}

	fmt.Printf(str)

	localFS := fileaccess.FSAccess{}
	root := dir
	actualfiles := []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}

	files, err := localFS.ListObjects(root, "")
	if err != nil {
		t.Errorf("Error finding files.")
	}

	if !utils.StringSlicesEqual(actualfiles, files) {
		t.Errorf("File list was incorrect, got: %v, want: %v.", files, actualfiles)
	}

	// This is not an Example test, so we call it directly here and check its return value
	err = mockS3.FinishTest()
	if err != nil {
		t.Errorf("mockS3 reported errors: %v", err)
	}
}

func TestRunLocalTestMissingFilesBrokenAppend(t *testing.T) {
	var mockS3 awsutil.MockS3Client
	// NOTE: directly calling mockS3.FinishTest() at the end to check its return value

	dir, err := ioutil.TempDir("/tmp", "ds")
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer os.RemoveAll(dir)
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
	}

	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			/*Contents: []*s3.Object{
				{Key: aws.String("archive/summary.json")},
			},*/
		},
		{},
		{},
		{
			Contents: []*s3.Object{
				{Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip")},
			},
		},
		{},
		{},
	}
	/*
		{
		Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/lastloaded.json"),
		},*/
	artifactPreProcessedBucket := "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7"
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/test-datasource-1/summary.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/lastloaded.json"),
		},
		{
			Bucket: aws.String(""), Key: aws.String("UserContent/notifications/123.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-04-35.zip"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/test-datasource-1/summary.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/lastloaded.json"),
		},
	}
	/*{
	Body: loadFileBytes("./test-data/configs/lastloaded.json", t),
	},*/
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{

		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/lastloaded.json", t),
		},

		{
			Body: loadFileBytes("./test-data/configs/123.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-04-35.zip", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/lastloaded.json", t),
		},
	}

	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-05-39.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-04-35.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-04-35.zip"),
		},
	}

	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
		{},
	}
	skipPaths := []string{
		"test-datasource-1/diffraction-db.bin",
		"Datasets/test-datasource-1/diffraction-db.bin",
	}
	mockS3.SkipPutChecks(skipPaths)
	// Add each expected upload file operation
	expFiles := []string{
		//"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}
	expExpFilePaths := []string{
		//"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/dataset-partial.bin",
		"./test-output/diffraction-db-partial.bin",
		"./test-output/summary2.json",
	}
	expBuckets := []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}
	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}

	e := DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-05-39.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	setupLocalPaths()

	s3access := fileaccess.MakeS3Access(&mockS3)
	ns := makeTestNotifications(s3access)
	str, err := executePipeline(e, s3access, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf("Error executing pipeline: %v", err)
	}
	if str != "" {
		t.Errorf("Unexpected return from pipeline: %v", str)
	}

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
	}

	// Add each expected upload file operation
	expFiles = []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}
	expExpFilePaths = []string{
		"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/dataset.bin",
		"./test-output/diffraction-db.bin",
		"./test-output/summary3.json",
	}
	expBuckets = []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}
	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}
	e = DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-04-35.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	str, err = executePipeline(e, s3access, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if str != "" {
		t.Errorf("Unexpected return from pipeline: %v", str)
	}

	localFS := fileaccess.FSAccess{}
	root := dir
	actualfiles := []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}

	files, err := localFS.ListObjects(root, "")
	if err != nil {
		t.Errorf("Error finding files.")
	}

	if !utils.StringSlicesEqual(actualfiles, files) {
		t.Errorf("File list was incorrect, got: %v, want: %v.", files, actualfiles)
	}

	// This is not an Example test, so we call it directly here and check its return value
	err = mockS3.FinishTest()
	if err != nil {
		t.Errorf("mockS3 reported errors: %v", err)
	}
}

func TestRunBrokenAppendWithCustomName(t *testing.T) {
	var mockS3 awsutil.MockS3Client
	// NOTE: directly calling mockS3.FinishTest() at the end to check its return value

	dir, err := ioutil.TempDir("/tmp", "ds")
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer os.RemoveAll(dir)
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("archive/060883460"),
		},
		{
			Bucket: aws.String(getManualBucket()), Prefix: aws.String("dataset-addons/060883460"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Prefix: aws.String("983561"),
		},
	}

	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{},
		{
			Contents: []*s3.Object{
				{Key: aws.String("dataset-addons/060883460/custom-meta.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String("983561/summary.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String("dataset-addons/060883460/custom-meta.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String("983561/summary.json")},
			},
		},
	}
	artifactPreProcessedBucket := "artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7"

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(getManualBucket()), Key: aws.String("dataset-addons/060883460/custom-meta.json"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/test-datasource-1/summary.json"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("983561/summary.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/lastloaded.json"),
		},
		{
			Bucket: aws.String(""), Key: aws.String("UserContent/notifications/123.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-04-35.zip"),
		},
		{
			Bucket: aws.String(getManualBucket()), Key: aws.String("dataset-addons/060883460/custom-meta.json"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("Datasets/test-datasource-1/summary.json"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("983561/summary.json"),
		},
		/*{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/lastloaded.json"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("configs/StandardPseudoIntensities.csv"),
		},
		{
			Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
		{
			Bucket: aws.String(artifactPreProcessedBucket), Key: aws.String("060883460-04-08-2021-09-04-35.zip"),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("dataset-addons/060883460/custom-meta.json"),
		},*/
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/config.json", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/lastloaded.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/123.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-04-35.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/config.json", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		{
			Body: loadFileBytes("./test-output/summary.json", t),
		},
		/*{
			Body: loadFileBytes("./test-data/configs/lastloaded.json", t),
		},
		{
			Body: loadFileBytes("./test-data/configs/StandardPseudoIntensities.csv", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-05-39.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/060883460-04-08-2021-09-04-35.zip", t),
		},
		{
			Body: loadFileBytes("./test-data/config.json", t),
		},*/
	}

	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-05-39.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-05-39.zip"),
		},
		{
			CopySource: aws.String(artifactPreProcessedBucket + "/060883460-04-08-2021-09-04-35.zip"), Bucket: aws.String(getDatasourceBucket()), Key: aws.String("archive/060883460-04-08-2021-09-04-35.zip"),
		},
	}

	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
		{},
	}
	skipPaths := []string{
		"test-datasource-1/diffraction-db.bin",
		"Datasets/test-datasource-1/diffraction-db.bin",
	}
	mockS3.SkipPutChecks(skipPaths)
	// Add each expected upload file operation
	expFiles := []string{
		//"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}
	expExpFilePaths := []string{
		//"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/dataset-partial-updatedname.bin",
		"./test-output/diffraction-partial-updatedname.bin",
		"./test-output/summary4.json",
	}
	expBuckets := []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}
	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}

	e := DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-05-39.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}
	setupLocalPaths()

	s3access := fileaccess.MakeS3Access(&mockS3)
	ns := makeTestNotifications(s3access)
	str, err := executePipeline(e, s3access, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf("Error executing pipeline: %v", err)
	}

	fmt.Printf(str)

	// Add each expected upload file operation
	expFiles = []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}
	expExpFilePaths = []string{
		"./test-output/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"./test-output/datasetupdatedname.bin",
		"./test-output/diffraction-db-updatedname.bin",
		"./test-output/summary5.json",
	}
	expBuckets = []string{
		getDatasourceBucket(),
		envBuckets[0],
		envBuckets[1],
		envBuckets[2],
	}
	for c, f := range expFiles {
		for bC, bucket := range expBuckets {
			fSend := f
			// In case of env buckets we prepend Datasets/
			if bC > 0 {
				fSend = "Datasets/" + f
			}
			mockS3.ExpPutObjectInput = append(mockS3.ExpPutObjectInput, s3.PutObjectInput{
				Bucket: aws.String(bucket), Key: aws.String(fSend), Body: loadFileBytes(expExpFilePaths[c], t),
			},
			)

			mockS3.QueuedPutObjectOutput = append(mockS3.QueuedPutObjectOutput, &s3.PutObjectOutput{})
		}
	}
	e = DatasourceEvent{
		Inpath:         "060883460-04-08-2021-09-04-35.zip",
		Rangespath:     "configs/StandardPseudoIntensities.csv",
		Outpath:        dir,
		DatasetID:      "test_datasource_missingfiles_name",
		DetectorConfig: "PIXL-EM-E2E",
	}

	str, err = executePipeline(e, s3access, ns, testFileCreationUnixTimeSec, artifactPreProcessedBucket, "", logger.NullLogger{})
	if err != nil {
		t.Errorf("Error executing pipeline: %v", err)
	}

	fmt.Printf(str)

	root := dir
	localFS := fileaccess.FSAccess{}

	actualfiles := []string{
		"test-datasource-1/PCR_D077T0637741562_000RCM_N00100360009835610066000J01.png",
		"test-datasource-1/dataset.bin",
		"test-datasource-1/diffraction-db.bin",
		"test-datasource-1/summary.json",
	}

	files, err := localFS.ListObjects(root, "")
	if err != nil {
		t.Errorf("Error finding files.")
	}

	// Read users file in
	var users datasetModel.SummaryFileData
	err = localFS.ReadJSON(dir, "test-datasource-1/summary.json", &users, false)
	if err != nil {
		t.Errorf("Failed to read in users file: %v", err)
	}

	if users.Title != "My Test Datasource" {
		t.Errorf("Data Source Incorrectly Named")
	}
	if !utils.StringSlicesEqual(actualfiles, files) {
		t.Errorf("File list was incorrect, got: %v, want: %v.", files, actualfiles)
	}

	// This is not an Example test, so we call it directly here and check its return value
	err = mockS3.FinishTest()
	if err != nil {
		t.Errorf("mockS3 reported errors: %v", err)
	}
}
