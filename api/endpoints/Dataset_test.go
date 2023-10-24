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

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/awsutil"

	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/pixlUser"
)

func Example_parseQueryParams() {
	params := []map[string]string{
		map[string]string{"unknown": "field"},
		map[string]string{"location_count": "30"},
		map[string]string{"location_count": "lt|550"},
		map[string]string{"location_count": "gt|1234"},
		map[string]string{"location_count": "bw|10|33"},
		map[string]string{"dataset_id": "30"},
		map[string]string{"dataset_id": "lt|30"},
		map[string]string{"dataset_id": "30", "whatever": "field", "location_count": "gt|1234"},
		map[string]string{"dataset_id": "30", "location_count": "gt|1234", "whatever": "value"},
	}

	for _, v := range params {
		q, err := parseQueryParams(v)
		fmt.Printf("%v|%v\n", err, q)
	}

	// Output:
	// Search not permitted on field: unknown|[]
	// <nil>|[{location_count = 30}]
	// <nil>|[{location_count < 550}]
	// <nil>|[{location_count > 1234}]
	// <nil>|[{location_count > 10} {location_count < 33}]
	// <nil>|[{dataset_id = 30}]
	// <nil>|[{dataset_id < 30}]
	// Search not permitted on field: whatever|[]
	// Search not permitted on field: whatever|[]
}

func Example_matchesSearch() {
	ds := datasetModel.SummaryFileData{
		DatasetID:         "590340",
		Group:             "the-group",
		DriveID:           292,
		SiteID:            1,
		TargetID:          "?",
		SOL:               "10",
		RTT:               590340,
		SCLK:              123456,
		ContextImage:      "MCC-234.png",
		LocationCount:     446,
		DataFileSize:      2699388,
		ContextImages:     1,
		NormalSpectra:     882,
		DwellSpectra:      0,
		BulkSpectra:       2,
		MaxSpectra:        2,
		PseudoIntensities: 441,
		DetectorConfig:    "PIXL",
	}

	queryItems := [][]queryItem{
		{queryItem{"location_count", "=", "446"}},
		{queryItem{"location_count", "=", "445"}},
		{queryItem{"location_count", ">", "445"}},
		{queryItem{"location_count", "<", "500"}},
		{queryItem{"dataset_id", ">", "590300"}},
		{queryItem{"sol", ">", "7"}, queryItem{"sol", "<", "141"}},
		{queryItem{"location_count", "=", "446"}, queryItem{"detector_config", "=", "PIXL"}},
		{
			queryItem{"location_count", "=", "446"},
			queryItem{"sol", "=", "10"},
			queryItem{"rtt", "=", "1590340"},
			queryItem{"sclk", ">", "123450"},
			queryItem{"data_file_size", ">", "2600000"},
			queryItem{"normal_spectra", ">", "800"},
			queryItem{"drive_id", ">", "290"},
			queryItem{"site_id", "=", "1"},
		},
		{
			queryItem{"location_count", "=", "446"},
			queryItem{"sol", "=", "10"},
			queryItem{"rtt", "=", "590340"},
			queryItem{"sclk", ">", "123450"},
			queryItem{"data_file_size", ">", "2600000"},
			queryItem{"normal_spectra", ">", "800"},
			queryItem{"drive_id", ">", "292"},
			queryItem{"site_id", "=", "1"},
		},
		{},
		{queryItem{"group_id", "=", "group1|the-group|anotherone"}},
		{queryItem{"group_id", "=", "group1|not-the-group|anotherone"}},
	}

	for _, q := range queryItems {
		match, err := matchesSearch(q, ds)
		fmt.Printf("%v|%v\n", err, match)
	}

	// Output:
	// <nil>|true
	// <nil>|false
	// <nil>|true
	// <nil>|true
	// Failed to compare dataset_id, can only use = for values "590300", "590340"|false
	// <nil>|true
	// <nil>|true
	// <nil>|false
	// <nil>|false
	// <nil>|true
	// <nil>|true
	// <nil>|false
}

func Example_datasetHandler_List() {
	const datasetsJSON = `{
"datasets": [
  {
   "dataset_id": "590340",
   "title": "the title",
   "site": "the site",
   "target": "the target",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "",
   "rtt": 590340,
   "sclk": 0,
   "context_image": "MCC-234.png",
   "location_count": 446,
   "data_file_size": 2699388,
   "context_images": 1,
   "tiff_context_images": 0,
   "normal_spectra": 882,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 441,
   "detector_config": "PIXL"
  },
  {
   "dataset_id": "983561",
   "group": "the-group",
   "drive_id": 36,
   "site_id": 1,
   "target_id": "?",
   "sol": "",
   "rtt": 983561,
   "sclk": 0,
   "context_image": "MCC-66.png",
   "location_count": 313,
   "data_file_size": 1840596,
   "context_images": 5,
   "tiff_context_images": 0,
   "normal_spectra": 612,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 306,
   "detector_config": "PIXL",
   "create_unixtime_sec": 1234567890
  },
  {
   "dataset_id": "222333",
   "group": "another-group",
   "drive_id": 36,
   "site_id": 1,
   "target_id": "?",
   "sol": "30",
   "rtt": 222333,
   "sclk": 0,
   "context_image": "MCC-66.png",
   "location_count": 313,
   "data_file_size": 1840596,
   "context_images": 5,
   "tiff_context_images": 0,
   "normal_spectra": 612,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 306,
   "detector_config": "PIXL",
   "create_unixtime_sec": 1234567891
  }
]
}`
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/datasets.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(datasetsJSON))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Email:  "niko@rockstar.com",
		Permissions: map[string]bool{
			"access:the-group":     true,
			"access:groupie":       true,
			"access:another-group": true,
			"access:super-admin":   true,
			"read:data-analysis":   true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset", nil) // Should return all items. NOTE: tests link creation (though no host name specified so won't have a valid link)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Request again with a different user, which excludes groups
	delete(mockUser.Permissions, "access:another-group")
	fmt.Printf("Permissions left: %v\n", len(mockUser.Permissions))
	req, _ = http.NewRequest("GET", "/dataset", nil) // Should return less based on group difference. NOTE: tests link creation (though no host name specified so won't have a valid link)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset?normal_spectra=882&detector_config=PIXL", nil) // Should filter with query string. NOTE: tests link creation (though no host name specified so won't have a valid link)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset?detector_config=Breadboard", nil) // Should return empty list, no items match query
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset?group_id=the-group|another", nil) // Should return item with the-group as its group id
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/dataset?title=he", nil) // Should return the one with title that contains "he" - we only have 1 title set
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// []
	//
	// 200
	// [
	//     {
	//         "dataset_id": "590340",
	//         "group": "groupie",
	//         "drive_id": 292,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "the site",
	//         "target": "the target",
	//         "title": "the title",
	//         "sol": "",
	//         "rtt": "000590340",
	//         "sclk": 0,
	//         "context_image": "MCC-234.png",
	//         "location_count": 446,
	//         "data_file_size": 2699388,
	//         "context_images": 1,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 882,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 441,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 0,
	//         "dataset_link": "https:///dataset/download/590340/dataset",
	//         "context_image_link": "https:///dataset/download/590340/MCC-234.png"
	//     },
	//     {
	//         "dataset_id": "983561",
	//         "group": "the-group",
	//         "drive_id": 36,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "",
	//         "target": "",
	//         "title": "",
	//         "sol": "",
	//         "rtt": "000983561",
	//         "sclk": 0,
	//         "context_image": "MCC-66.png",
	//         "location_count": 313,
	//         "data_file_size": 1840596,
	//         "context_images": 5,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 612,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 306,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 1234567890,
	//         "dataset_link": "https:///dataset/download/983561/dataset",
	//         "context_image_link": "https:///dataset/download/983561/MCC-66.png"
	//     },
	//     {
	//         "dataset_id": "222333",
	//         "group": "another-group",
	//         "drive_id": 36,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "",
	//         "target": "",
	//         "title": "",
	//         "sol": "30",
	//         "rtt": "000222333",
	//         "sclk": 0,
	//         "context_image": "MCC-66.png",
	//         "location_count": 313,
	//         "data_file_size": 1840596,
	//         "context_images": 5,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 612,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 306,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 1234567891,
	//         "dataset_link": "https:///dataset/download/222333/dataset",
	//         "context_image_link": "https:///dataset/download/222333/MCC-66.png"
	//     }
	// ]
	//
	// Permissions left: 4
	// 200
	// [
	//     {
	//         "dataset_id": "590340",
	//         "group": "groupie",
	//         "drive_id": 292,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "the site",
	//         "target": "the target",
	//         "title": "the title",
	//         "sol": "",
	//         "rtt": "000590340",
	//         "sclk": 0,
	//         "context_image": "MCC-234.png",
	//         "location_count": 446,
	//         "data_file_size": 2699388,
	//         "context_images": 1,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 882,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 441,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 0,
	//         "dataset_link": "https:///dataset/download/590340/dataset",
	//         "context_image_link": "https:///dataset/download/590340/MCC-234.png"
	//     },
	//     {
	//         "dataset_id": "983561",
	//         "group": "the-group",
	//         "drive_id": 36,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "",
	//         "target": "",
	//         "title": "",
	//         "sol": "",
	//         "rtt": "000983561",
	//         "sclk": 0,
	//         "context_image": "MCC-66.png",
	//         "location_count": 313,
	//         "data_file_size": 1840596,
	//         "context_images": 5,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 612,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 306,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 1234567890,
	//         "dataset_link": "https:///dataset/download/983561/dataset",
	//         "context_image_link": "https:///dataset/download/983561/MCC-66.png"
	//     }
	// ]
	//
	// 200
	// [
	//     {
	//         "dataset_id": "590340",
	//         "group": "groupie",
	//         "drive_id": 292,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "the site",
	//         "target": "the target",
	//         "title": "the title",
	//         "sol": "",
	//         "rtt": "000590340",
	//         "sclk": 0,
	//         "context_image": "MCC-234.png",
	//         "location_count": 446,
	//         "data_file_size": 2699388,
	//         "context_images": 1,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 882,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 441,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 0,
	//         "dataset_link": "https:///dataset/download/590340/dataset",
	//         "context_image_link": "https:///dataset/download/590340/MCC-234.png"
	//     }
	// ]
	//
	// 200
	// []
	//
	// 200
	// [
	//     {
	//         "dataset_id": "983561",
	//         "group": "the-group",
	//         "drive_id": 36,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "",
	//         "target": "",
	//         "title": "",
	//         "sol": "",
	//         "rtt": "000983561",
	//         "sclk": 0,
	//         "context_image": "MCC-66.png",
	//         "location_count": 313,
	//         "data_file_size": 1840596,
	//         "context_images": 5,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 612,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 306,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 1234567890,
	//         "dataset_link": "https:///dataset/download/983561/dataset",
	//         "context_image_link": "https:///dataset/download/983561/MCC-66.png"
	//     }
	// ]
	//
	// 200
	// [
	//     {
	//         "dataset_id": "590340",
	//         "group": "groupie",
	//         "drive_id": 292,
	//         "site_id": 1,
	//         "target_id": "?",
	//         "site": "the site",
	//         "target": "the target",
	//         "title": "the title",
	//         "sol": "",
	//         "rtt": "000590340",
	//         "sclk": 0,
	//         "context_image": "MCC-234.png",
	//         "location_count": 446,
	//         "data_file_size": 2699388,
	//         "context_images": 1,
	//         "tiff_context_images": 0,
	//         "normal_spectra": 882,
	//         "dwell_spectra": 0,
	//         "bulk_spectra": 2,
	//         "max_spectra": 2,
	//         "pseudo_intensities": 441,
	//         "detector_config": "PIXL",
	//         "create_unixtime_sec": 0,
	//         "dataset_link": "https:///dataset/download/590340/dataset",
	//         "context_image_link": "https:///dataset/download/590340/MCC-234.png"
	//     }
	// ]
}

func Example_datasetHandler_Stream_BadGroup_403() {
	const summaryJSON = `{
   "dataset_id": "590340",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "0",
   "rtt": 590340,
   "sclk": 0,
   "context_image": "MCC-234.png",
   "location_count": 446,
   "data_file_size": 2699388,
   "context_images": 1,
   "tiff_context_images": 0,
   "normal_spectra": 882,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 441,
   "detector_config": "PIXL"
}`

	const publicDatasetsJSON = `{
   "590340": {
      "dataset_id": "590340",
      "public": false,
      "public_release_utc_time_sec": 0,
      "sol": ""
   }
}`

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String("config-bucket"), Key: aws.String("PixliseConfig/datasets-auth.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(publicDatasetsJSON))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		"Niko Bellic",
		"600f2a0806b6c70071d3d174",
		"niko@rockstar.com",
		map[string]bool{
			"access:the-group":   true,
			"access:super-admin": true,
			"read:data-analysis": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/download/590340/dataset", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 403
	// dataset 590340 not permitted
}

func Example_datasetHandler_Stream_OK() {
	const summaryJSON = `{
   "dataset_id": "590340",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "0",
   "rtt": 590340,
   "sclk": 0,
   "context_image": "MCC-234.png",
   "location_count": 446,
   "data_file_size": 2699388,
   "context_images": 1,
   "tiff_context_images": 0,
   "normal_spectra": 882,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 441,
   "detector_config": "PIXL"
}`

	datasetBytes := []byte{50, 60, 61, 62, 70}

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/dataset.bin"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(datasetBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(datasetBytes)), // return some printable chars so easier to compare in Output comment
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		"Niko Bellic",
		"600f2a0806b6c70071d3d174",
		"niko@rockstar.com",
		map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/download/590340/dataset", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Output:
	// 200
	// [attachment; filename="dataset.bin"]
	// [5]
	// 2<=>F
}

func Example_datasetHandler_Stream_NoSuchDataset() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		"Niko Bellic",
		"600f2a0806b6c70071d3d174",
		"niko@rockstar.com",
		map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/download/590340/dataset", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 590340 not found
}

func Example_datasetHandler_Stream_BadSummary() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		"Niko Bellic",
		"600f2a0806b6c70071d3d174",
		"niko@rockstar.com",
		map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/dataset/download/590340/dataset", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 500
	// failed to verify dataset group permission
}

func Example_datasetHandler_MCC_Stream_OK() {
	const summaryJSON = `{
   "dataset_id": "590340",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "30",
   "rtt": 590340,
   "sclk": 0,
   "context_image": "MCC-234.png",
   "location_count": 446,
   "data_file_size": 2699388,
   "context_images": 1,
   "tiff_context_images": 0,
   "normal_spectra": 882,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 441,
   "detector_config": "PIXL"
}`

	mccBytes := []byte{60, 112, 110, 103, 62}

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/MCC-234.png"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/MCC-234.png"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/MCC-234.png"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/MCC-455.png"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/indiana-jones.txt"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(mccBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(mccBytes)), // return some printable chars so easier to compare in Output comment
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(mccBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(mccBytes)), // return some printable chars so easier to compare in Output comment
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(mccBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(mccBytes)), // return some printable chars so easier to compare in Output comment
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(mccBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(mccBytes)), // return some printable chars so easier to compare in Output comment
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		"Niko Bellic",
		"600f2a0806b6c70071d3d174",
		"niko@rockstar.com",
		map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	paths := []string{
		"/dataset/download/590340/context-image",
		"/dataset/download/590340/context-thumb",
		"/dataset/download/590340/context-mark-thumb",
		// a different context image
		"/dataset/download/590340/MCC-455.png",
		// non-existant file
		"/dataset/download/590340/indiana-jones.txt",
	}

	for _, path := range paths {
		req, _ := http.NewRequest("GET", path, nil) // Should return empty list, datasets.json fails to download
		resp := executeRequest(req, apiRouter.Router)

		fmt.Println(resp.Code)
		// Make sure the response is the right kind...
		fmt.Println(resp.HeaderMap["Content-Disposition"])
		fmt.Println(resp.HeaderMap["Content-Length"])
		fmt.Println(resp.Body)
	}

	// Output:
	// 200
	// [attachment; filename="MCC-234.png"]
	// [5]
	// <png>
	// 200
	// [attachment; filename="MCC-234.png"]
	// [5]
	// <png>
	// 200
	// [attachment; filename="MCC-234.png"]
	// [5]
	// <png>
	// 200
	// [attachment; filename="MCC-455.png"]
	// [5]
	// <png>
	// 404
	// []
	// []
	// indiana-jones.txt not found
}

func Example_datasetCreatePost_BadFormat() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/dataset/abc-123", bytes.NewReader([]byte("data goes here")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/dataset/abc-123?format=some-format", bytes.NewReader([]byte("data goes here")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 500
	// Unexpected format: ""
	//
	// 500
	// Unexpected format: "some-format"
}

func Example_datasetCreatePost_CantCheckExists() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("UploadedDatasets/the-test_dataset")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	body, err := ioutil.ReadFile("./test-data/just-image.zip")
	fmt.Println(err)

	req, _ := http.NewRequest("POST", "/dataset/the-test dataset?format=jpl-breadboard", bytes.NewReader(body))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// <nil>
	// 500
	// Failed to list existing files for dataset ID: the-test_dataset. Error: Returning error from ListObjectsV2
}

func Example_datasetCreatePost_BadZip() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("UploadedDatasets/the-test_dataset")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	body, err := ioutil.ReadFile("./test-data/just-image.zip")
	fmt.Println(err)

	req, _ := http.NewRequest("POST", "/dataset/the-test dataset?format=jpl-breadboard", bytes.NewReader(body))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// <nil>
	// 500
	// Zip file must only contain MSA files. Found: Non-abraded sample.png
}

func Example_datasetCreatePost_Exists() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("UploadedDatasets/the-test_dataset")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UploadedDatasets/the-test_dataset/file.txt")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	body, err := ioutil.ReadFile("./test-data/just-image.zip")
	fmt.Println(err)

	req, _ := http.NewRequest("POST", "/dataset/the-test dataset?format=jpl-breadboard", bytes.NewReader(body))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// <nil>
	// 500
	// Dataset ID already exists: the-test_dataset
}

func Example_datasetCreatePost_Success() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(artifactManualUploadBucket), Prefix: aws.String("UploadedDatasets/the-test_dataset")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{},
		},
	}

	expectedSpectraBytes, err := ioutil.ReadFile("./test-data/expected-spectra.zip")
	fmt.Printf("expected spectra read error: %v\n", err)

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("UploadedDatasets/the-test_dataset/spectra.zip"), Body: bytes.NewReader(expectedSpectraBytes),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("UploadedDatasets/the-test_dataset/import.json"), Body: bytes.NewReader([]byte(`{
    "datasetid": "the-test_dataset",
    "title": "the-test_dataset",
    "targetid": "0",
    "target": "",
    "siteid": 0,
    "site": "",
    "group": "JPL Breadboard",
    "beamfile": "",
    "beamparams": "10,0,10,0",
    "housekeeping": "",
    "contextimgdir": "",
    "msadir": "spectra",
    "pseudointensitycsv": "",
    "ignoremsa": "",
    "singledetectormsa": false,
    "genpmcs": true,
    "readtype": "Normal",
    "detaduplicate": false,
    "genbulkmax": true,
    "detectorconfig": "Breadboard",
    "bulkquantfile": "",
    "ev_xperchan_a": 0,
    "ev_offset_a": 0,
    "ev_xperchan_b": 0,
    "ev_offset_b": 0,
    "exclude_normal_dwell": false,
    "sol": ""
}`)),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("UploadedDatasets/the-test_dataset/detector.json"), Body: bytes.NewReader([]byte(`{
    "detector": "jpl-breadboard"
}`)),
		},
		{
			Bucket: aws.String(artifactManualUploadBucket), Key: aws.String("UploadedDatasets/the-test_dataset/creator.json"), Body: bytes.NewReader([]byte(`{
    "name": "Niko Bellic",
    "user_id": "600f2a0806b6c70071d3d174",
    "email": "niko@spicule.co.uk"
}`)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
		{},
	}
	idGen := services.MockIDGenerator{
		IDs: []string{"uuu333"},
	}

	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)

	mockSNS := awsutil.MockSNS{
		ExpInput: []sns.PublishInput{
			{
				Message:  aws.String("{\"datasetID\":\"the-test_dataset\",\"logID\":\"dataimport-uuu333\"}"),
				TopicArn: aws.String("arn:1:2:3:4:5"),
			},
		},
		QueuedOutput: []sns.PublishOutput{
			{},
		},
	}
	svcs.SNS = &mockSNS

	svcs.Config.ManualUploadBucket = artifactManualUploadBucket
	apiRouter := MakeRouter(svcs)

	body, err := os.ReadFile("./test-data/just-msas.zip")
	fmt.Printf("expected upload file read error: %v\n", err)

	req, _ := http.NewRequest("POST", "/dataset/the-test dataset?format=jpl-breadboard", bytes.NewReader(body))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// expected spectra read error: <nil>
	// expected upload file read error: <nil>
	// 200
	// "the-test_dataset-dataimport-uuu333"
}
