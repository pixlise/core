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
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
)

const datasetBucket = "dev-pixlise-data"
const configBucket = "dev-pixlise-config"

func Example_updateDatasetsBucketFail() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := &logger.NullLogger{}

	// Listing returns an error
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(datasetBucket), Prefix: aws.String("Datasets/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{nil}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(updateDatasets(fs, datasetBucket, configBucket, l))

	// Output:
	// Returning error from ListObjectsV2

}

func Example_updateDatasetsErrorGettingFiles() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := &logger.NullLogger{}

	// Listing returns 1 item, get status returns error, check that it still requests 2nd item, 2nd item will fail to parse
	// but the func should still upload a blank datasets.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(datasetBucket), Prefix: aws.String("Datasets/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			IsTruncated: aws.Bool(false),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-123/summary.json")},
				{Key: aws.String("Datasets/abc-123/node1.json")},
				{Key: aws.String("Datasets/abc-123/params.json")},
				{Key: aws.String("Datasets/abc-456/summary.json")},
				{Key: aws.String("Datasets/abc-456/node1.json")},
				{Key: aws.String("Datasets/abc-456/params.json")},
				{Key: aws.String("Datasets/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/bad-dataset-ids.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-123/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-456/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // Pretend no bad dataset ids file, this shouldn't affect outcome
		// This is what we're testing
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/datasets.json"), Body: bytes.NewReader([]byte(`{
 "datasets": []
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(updateDatasets(fs, datasetBucket, configBucket, l))

	// Output:
	// <nil>
}

func Example_updateDatasetsTwoSummaryCombineNilJson() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := &logger.NullLogger{}

	// Listing returns 1 item, get status returns error, requests 2nd and 3rd item and properly combines the
	//two jsons into datasets.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(datasetBucket), Prefix: aws.String("Datasets/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			IsTruncated: aws.Bool(false),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-123/summary.json")},
				{Key: aws.String("Datasets/abc-123/node1.json")},
				{Key: aws.String("Datasets/abc-123/params.json")},
				{Key: aws.String("Datasets/abc-456/summary.json")},
				{Key: aws.String("Datasets/abc-789/summary.json")},
				{Key: aws.String("Datasets/abc-456/params.json")},
				{Key: aws.String("Datasets/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/bad-dataset-ids.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-123/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-456/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-789/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // Pretend no bad dataset ids file, this shouldn't affect outcome
		nil,
		// NOTE: Missing creation time, eg existing old datasets
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x11",
				"group": "the-group",
				"title": "5x5 title",
				"site": "5x5 site",
				"target": "5x5 target",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-6.jpg",
				"location_count": 4035,
				"data_file_size": 23212328,
				"context_images": 2,
				"tiff_context_images": 0,
				"normal_spectra": 8064,
				"dwell_spectra": 0,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL"
			   }`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x5-full",
				"group": "the-group",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-4042.jpg",
				"location_count": 1769,
				"data_file_size": 11202865,
				"context_images": 10,
				"tiff_context_images": 0,
				"normal_spectra": 3528,
				"dwell_spectra": 2,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL",
				"create_unixtime_sec": 1234567890
			   }`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/datasets.json"), Body: bytes.NewReader([]byte(`{
 "datasets": [
  {
   "dataset_id": "test-fm-5x11",
   "group": "the-group",
   "drive_id": 0,
   "site_id": 0,
   "target_id": "?",
   "site": "5x5 site",
   "target": "5x5 target",
   "title": "5x5 title",
   "sol": "",
   "rtt": 0,
   "sclk": 0,
   "context_image": "MCC-6.jpg",
   "location_count": 4035,
   "data_file_size": 23212328,
   "context_images": 2,
   "tiff_context_images": 0,
   "normal_spectra": 8064,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 0,
   "detector_config": "PIXL",
   "create_unixtime_sec": 0
  },
  {
   "dataset_id": "test-fm-5x5-full",
   "group": "the-group",
   "drive_id": 0,
   "site_id": 0,
   "target_id": "?",
   "site": "",
   "target": "",
   "title": "",
   "sol": "",
   "rtt": 0,
   "sclk": 0,
   "context_image": "MCC-4042.jpg",
   "location_count": 1769,
   "data_file_size": 11202865,
   "context_images": 10,
   "tiff_context_images": 0,
   "normal_spectra": 3528,
   "dwell_spectra": 2,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 0,
   "detector_config": "PIXL",
   "create_unixtime_sec": 1234567890
  }
 ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(updateDatasets(fs, datasetBucket, configBucket, l))

	// Output:
	// <nil>
}

func Example_updateDatasetsTwoSummaryCombineBadJson() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := &logger.NullLogger{}

	// Listing returns 1 item that is invalid json, does not parse it, returnrs error, and moves on
	// requests 2nd and 3rd item and properly combines the two jsons into datasets.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(datasetBucket), Prefix: aws.String("Datasets/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			IsTruncated: aws.Bool(false),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-123/summary.json")},
				{Key: aws.String("Datasets/abc-123/node1.json")},
				{Key: aws.String("Datasets/abc-123/params.json")},
				{Key: aws.String("Datasets/abc-456/summary.json")},
				{Key: aws.String("Datasets/abc-789/summary.json")},
				{Key: aws.String("Datasets/abc-456/params.json")},
				{Key: aws.String("Datasets/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/bad-dataset-ids.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-123/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-456/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-789/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // Pretend no bad dataset ids file, this shouldn't affect outcome
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
		// NOTE: Missing creation time, eg existing old datasets
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x11",
				"title": "5x5 title",
				"site": "5x5 site",
				"target": "5x5 target",
				"group": "groupie",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "230",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-6.jpg",
				"location_count": 4035,
				"data_file_size": 23212328,
				"context_images": 2,
				"tiff_context_images": 0,
				"normal_spectra": 8064,
				"dwell_spectra": 0,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL"
			   }`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x5-full",
				"group": "groupie",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "231",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-4042.jpg",
				"location_count": 1769,
				"data_file_size": 11202865,
				"context_images": 10,
				"tiff_context_images": 0,
				"normal_spectra": 3528,
				"dwell_spectra": 2,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL",
				"create_unixtime_sec": 1234567890
			   }`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/datasets.json"), Body: bytes.NewReader([]byte(`{
 "datasets": [
  {
   "dataset_id": "test-fm-5x11",
   "group": "groupie",
   "drive_id": 0,
   "site_id": 0,
   "target_id": "?",
   "site": "5x5 site",
   "target": "5x5 target",
   "title": "5x5 title",
   "sol": "230",
   "rtt": 0,
   "sclk": 0,
   "context_image": "MCC-6.jpg",
   "location_count": 4035,
   "data_file_size": 23212328,
   "context_images": 2,
   "tiff_context_images": 0,
   "normal_spectra": 8064,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 0,
   "detector_config": "PIXL",
   "create_unixtime_sec": 0
  },
  {
   "dataset_id": "test-fm-5x5-full",
   "group": "groupie",
   "drive_id": 0,
   "site_id": 0,
   "target_id": "?",
   "site": "",
   "target": "",
   "title": "",
   "sol": "231",
   "rtt": 0,
   "sclk": 0,
   "context_image": "MCC-4042.jpg",
   "location_count": 1769,
   "data_file_size": 11202865,
   "context_images": 10,
   "tiff_context_images": 0,
   "normal_spectra": 3528,
   "dwell_spectra": 2,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 0,
   "detector_config": "PIXL",
   "create_unixtime_sec": 1234567890
  }
 ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(updateDatasets(fs, datasetBucket, configBucket, l))

	// Output:
	// <nil>
}

func Example_updateDatasetsTwoSummaryCombineBadJsonWithBadIDMarked() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := &logger.NullLogger{}

	// Listing returns 1 item that is invalid json, does not parse it, returnrs error, and moves on
	// requests 2nd and 3rd item and properly combines the two jsons into datasets.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(datasetBucket), Prefix: aws.String("Datasets/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			IsTruncated: aws.Bool(false),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-123/summary.json")},
				{Key: aws.String("Datasets/abc-123/node1.json")},
				{Key: aws.String("Datasets/abc-123/params.json")},
				{Key: aws.String("Datasets/abc-456/summary.json")},
				{Key: aws.String("Datasets/abc-789/summary.json")},
				{Key: aws.String("Datasets/abc-456/params.json")},
				{Key: aws.String("Datasets/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/bad-dataset-ids.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-123/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-456/summary.json"),
		},
		{
			Bucket: aws.String(datasetBucket), Key: aws.String("Datasets/abc-789/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("[\"test-fm-5x11\",\"another-id\"]"))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
		// NOTE: Missing creation time, eg existing old datasets
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x11",
				"title": "5x5 title",
				"site": "5x5 site",
				"target": "5x5 target",
				"group": "groupie",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "230",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-6.jpg",
				"location_count": 4035,
				"data_file_size": 23212328,
				"context_images": 2,
				"tiff_context_images": 0,
				"normal_spectra": 8064,
				"dwell_spectra": 0,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL"
			   }`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "test-fm-5x5-full",
				"group": "groupie",
				"drive_id": 0,
				"site_id": 0,
				"target_id": "?",
				"sol": "231",
				"rtt": 0,
				"sclk": 0,
				"context_image": "MCC-4042.jpg",
				"location_count": 1769,
				"data_file_size": 11202865,
				"context_images": 10,
				"tiff_context_images": 0,
				"normal_spectra": 3528,
				"dwell_spectra": 2,
				"bulk_spectra": 2,
				"max_spectra": 2,
				"pseudo_intensities": 0,
				"detector_config": "PIXL",
				"create_unixtime_sec": 1234567890
			   }`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(configBucket), Key: aws.String("PixliseConfig/datasets.json"), Body: bytes.NewReader([]byte(`{
 "datasets": [
  {
   "dataset_id": "test-fm-5x5-full",
   "group": "groupie",
   "drive_id": 0,
   "site_id": 0,
   "target_id": "?",
   "site": "",
   "target": "",
   "title": "",
   "sol": "231",
   "rtt": 0,
   "sclk": 0,
   "context_image": "MCC-4042.jpg",
   "location_count": 1769,
   "data_file_size": 11202865,
   "context_images": 10,
   "tiff_context_images": 0,
   "normal_spectra": 3528,
   "dwell_spectra": 2,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 0,
   "detector_config": "PIXL",
   "create_unixtime_sec": 1234567890
  }
 ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(updateDatasets(fs, datasetBucket, configBucket, l))

	// Output:
	// <nil>
}
