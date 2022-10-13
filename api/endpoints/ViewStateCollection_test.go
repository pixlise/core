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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
)

func Example_viewStateHandler_ListCollections() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	viewStateSavedS3Path := viewStateS3Path + "WorkspaceCollections"
	sharedViewStateSavedS3Path := sharedViewStateS3Path + "WorkspaceCollections"

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(sharedViewStateSavedS3Path),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateSavedS3Path + "/The view state.json"), LastModified: aws.Time(time.Unix(1634731913, 0))},
				{Key: aws.String(viewStateSavedS3Path + "/Another-state 123.json"), LastModified: aws.Time(time.Unix(1634731914, 0))},
				{Key: aws.String(viewStateSavedS3Path + "/My 10th collection save.json"), LastModified: aws.Time(time.Unix(1634731915, 0))},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateSavedS3Path + "/For all.json"), LastModified: aws.Time(time.Unix(1634731917, 0))},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Exists, success
	req, _ := http.NewRequest("GET", "/view-state/collections/TheDataSetID", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// [
	//     {
	//         "name": "The view state",
	//         "modifiedUnixSec": 1634731913
	//     },
	//     {
	//         "name": "Another-state 123",
	//         "modifiedUnixSec": 1634731914
	//     },
	//     {
	//         "name": "My 10th collection save",
	//         "modifiedUnixSec": 1634731915
	//     },
	//     {
	//         "name": "shared-For all",
	//         "modifiedUnixSec": 1634731917
	//     }
	// ]
}

func Example_viewStateHandler_GetCollection() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/The 1st one.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/Another_collection-01-01-2022.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/State one.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/The end.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/Collection with creator.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/State one.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/The end.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "Another_collection-01-01-2022",
    "viewStateIDs": [
        "State one",
        "The end"
    ],
	"description": "some description",
    "viewStates": null
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewState": {"quantification": {"appliedQuantID": "quant for state one"}}}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewState": {"quantification": {"appliedQuantID": "quant for the end"}}}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "Another_collection-01-01-2022",
    "viewStateIDs": [
        "State one",
        "The end"
    ],
    "description": "some description",
    "viewStates": null,
    "shared": false,
    "creator": {
        "name": "Roman Bellic",
        "user_id": "another-user-123",
        "email": "roman@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"viewState": {
		"quantification": {
			"appliedQuantID": "quant for state one"
		},
		"selection": {
			"locIdxs": [1, 2],
			"pixelSelectionImageName": "file.tif",
			"pixelIdxs": [3,4]
		}
	}
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewState": {"quantification": {"appliedQuantID": "quant for the end"}}}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("GET", "/view-state/collections/TheDataSetID/The 1st one", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists (no creator info saved), success
	req, _ = http.NewRequest("GET", "/view-state/collections/TheDataSetID/Another_collection-01-01-2022", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists (with creator info), success
	req, _ = http.NewRequest("GET", "/view-state/collections/TheDataSetID/Collection with creator", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// The 1st one not found
	//
	// 200
	// {
	//     "viewStateIDs": [
	//         "State one",
	//         "The end"
	//     ],
	//     "name": "Another_collection-01-01-2022",
	//     "description": "some description",
	//     "viewStates": {
	//         "State one": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for state one"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": []
	//             }
	//         },
	//         "The end": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for the end"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": []
	//             }
	//         }
	//     }
	// }
	//
	// 200
	// {
	//     "viewStateIDs": [
	//         "State one",
	//         "The end"
	//     ],
	//     "name": "Another_collection-01-01-2022",
	//     "description": "some description",
	//     "viewStates": {
	//         "State one": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for state one"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": [
	//                     1,
	//                     2
	//                 ],
	//                 "pixelSelectionImageName": "file.tif",
	//                 "pixelIdxs": [
	//                     3,
	//                     4
	//                 ]
	//             }
	//         },
	//         "The end": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for the end"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": []
	//             }
	//         }
	//     },
	//     "shared": false,
	//     "creator": {
	//         "name": "Roman Bellic",
	//         "user_id": "another-user-123",
	//         "email": "roman@spicule.co.uk"
	//     }
	// }
}

func Example_viewStateHandler_GetCollectionShared() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/The 1st one.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/The one that works.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "The one that works",
    "viewStateIDs": [
        "State one",
        "The end"
    ],
	"description": "some description",
    "shared": true,
    "creator": {
        "name": "Roman Bellic",
        "user_id": "another-user-123",
        "email": "roman@spicule.co.uk"
    },
    "viewStates": {
        "State one": {
            "analysisLayout": {
                "topWidgetSelectors": [],
                "bottomWidgetSelectors": []
            },
            "spectrum": {
                "panX": 0,
                "panY": 0,
                "zoomX": 1,
                "zoomY": 1,
                "spectrumLines": [],
                "logScale": true,
                "xrflines": [],
                "showXAsEnergy": false,
                "energyCalibration": []
            },
            "contextImages": {},
            "histograms": {},
            "chordDiagrams": {},
            "ternaryPlots": {},
            "binaryPlots": {},
            "tables": {},
            "roiQuantTables": {},
            "variograms": {},
            "spectrums": {},
            "rgbuPlots": {},
            "singleAxisRGBU": {},
            "rgbuImages": {},
            "parallelograms": {},
            "annotations": {
                "savedAnnotations": []
            },
            "rois": {
                "roiColours": {},
                "roiShapes": {}
            },
            "quantification": {
                "appliedQuantID": "quant for state one"
            },
            "selection": {
                "roiID": "",
                "roiName": "",
                "locIdxs": []
            }
        },
        "The end": {
            "analysisLayout": {
                "topWidgetSelectors": [],
                "bottomWidgetSelectors": []
            },
            "spectrum": {
                "panX": 0,
                "panY": 0,
                "zoomX": 1,
                "zoomY": 1,
                "spectrumLines": [],
                "logScale": true,
                "xrflines": [],
                "showXAsEnergy": false,
                "energyCalibration": []
            },
            "contextImages": {},
            "histograms": {},
            "chordDiagrams": {},
            "ternaryPlots": {},
            "binaryPlots": {},
            "tables": {},
            "roiQuantTables": {},
            "variograms": {},
            "spectrums": {},
            "rgbuPlots": {},
            "singleAxisRGBU": {},
            "rgbuImages": {},
            "parallelograms": {},
            "annotations": {
                "savedAnnotations": []
            },
            "rois": {
                "roiColours": {},
				"roiShapes": {}
            },
            "quantification": {
                "appliedQuantID": "quant for the end"
            },
            "selection": {
                "roiID": "",
                "roiName": "",
                "locIdxs": []
            }
        }
    }
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("GET", "/view-state/collections/TheDataSetID/shared-The 1st one", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, success
	req, _ = http.NewRequest("GET", "/view-state/collections/TheDataSetID/shared-The one that works", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// The 1st one not found
	//
	// 200
	// {
	//     "viewStateIDs": [
	//         "State one",
	//         "The end"
	//     ],
	//     "name": "The one that works",
	//     "description": "some description",
	//     "viewStates": {
	//         "State one": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for state one"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": []
	//             }
	//         },
	//         "The end": {
	//             "analysisLayout": {
	//                 "topWidgetSelectors": [],
	//                 "bottomWidgetSelectors": []
	//             },
	//             "spectrum": {
	//                 "panX": 0,
	//                 "panY": 0,
	//                 "zoomX": 1,
	//                 "zoomY": 1,
	//                 "spectrumLines": [],
	//                 "logScale": true,
	//                 "xrflines": [],
	//                 "showXAsEnergy": false,
	//                 "energyCalibration": []
	//             },
	//             "contextImages": {},
	//             "histograms": {},
	//             "chordDiagrams": {},
	//             "ternaryPlots": {},
	//             "binaryPlots": {},
	//             "tables": {},
	//             "roiQuantTables": {},
	//             "variograms": {},
	//             "spectrums": {},
	//             "rgbuPlots": {},
	//             "singleAxisRGBU": {},
	//             "rgbuImages": {},
	//             "parallelograms": {},
	//             "annotations": {
	//                 "savedAnnotations": []
	//             },
	//             "rois": {
	//                 "roiColours": {},
	//                 "roiShapes": {}
	//             },
	//             "quantification": {
	//                 "appliedQuantID": "quant for the end"
	//             },
	//             "selection": {
	//                 "roiID": "",
	//                 "roiName": "",
	//                 "locIdxs": []
	//             }
	//         }
	//     },
	//     "shared": true,
	//     "creator": {
	//         "name": "Roman Bellic",
	//         "user_id": "another-user-123",
	//         "email": "roman@spicule.co.uk"
	//     }
	// }
}

func Example_viewStateHandler_PutCollection() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/The best collection 23_09_2021.json"), Body: bytes.NewReader([]byte(`{
    "viewStateIDs": [
        "state one",
        "second View State",
        "Third-state"
    ],
    "name": "The best collection 23_09_2021",
    "description": "the desc",
    "viewStates": null,
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("PUT", "/view-state/collections/TheDataSetID/The best collection 23_09_2021", bytes.NewReader([]byte(`{
    "viewStateIDs": [
        "state one",
        "second View State",
        "Third-state"
    ],
    "name": "The wrong name",
    "description": "the desc"
}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
}

func Example_viewStateHandler_DeleteCollection() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/viewstate123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewStateIDs": ["view one", "view state 2", "num 3 view"], "name": "viewState555"}`))),
		},
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/viewstate555.json"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("DELETE", "/view-state/collections/TheDataSetID/viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, success
	req, _ = http.NewRequest("DELETE", "/view-state/collections/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// View state collection not found
	//
	// 200
	//
}

func Example_viewStateHandler_DeleteCollectionShared() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/viewstate123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"viewStateIDs": ["view one", "view state 2"],
				"name": "viewstate123",
				"shared": true,
				"creator": {
					"name": "Roman Bellic",
					"user_id": "another-user-123",
					"email": "roman@spicule.co.uk"
				}
			}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"viewStateIDs": ["view one", "view state 2", "num 3 view"],
				"name": "viewState555",
				"shared": true,
				"creator": {
					"name": "Niko Bellic",
					"user_id": "600f2a0806b6c70071d3d174",
					"email": "niko@spicule.co.uk"
				}
			}`))),
		},
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/viewstate555.json"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Not created by user, should fail
	req, _ := http.NewRequest("DELETE", "/view-state/collections/TheDataSetID/shared-viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Created by user, success
	req, _ = http.NewRequest("DELETE", "/view-state/collections/TheDataSetID/shared-viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 401
	// viewstate123 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
	//
}

func Example_viewStateHandler_ShareCollection() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1 - just collection
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/331.json"),
		},
		// Test 2 - just collection
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/332.json"),
		},
		// Test 3 - collection+view state, which fails
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/333.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/The first one.json"),
		},
		// Test 4 - collection+view state files
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "WorkspaceCollections/334.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/The first one.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/Another workspace.json"),
		},
	}

	const collectionResp = `{
    "name": "Another_collection-01-01-2022",
    "viewStateIDs": [
        "The first one",
        "Another workspace"
    ],
	"description": "some description"
}`

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(collectionResp))),
		},
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(collectionResp))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewState": {"quantification": {"appliedQuantID": "quant1"}}}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"viewState": {"quantification": {"appliedQuantID": "quant2"}}}`))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "WorkspaceCollections/334.json"), Body: bytes.NewReader([]byte(`{
    "viewStateIDs": [
        "The first one",
        "Another workspace"
    ],
    "name": "Another_collection-01-01-2022",
    "description": "some description",
    "viewStates": {
        "Another workspace": {
            "analysisLayout": {
                "topWidgetSelectors": [],
                "bottomWidgetSelectors": []
            },
            "spectrum": {
                "panX": 0,
                "panY": 0,
                "zoomX": 1,
                "zoomY": 1,
                "spectrumLines": [],
                "logScale": true,
                "xrflines": [],
                "showXAsEnergy": false,
                "energyCalibration": []
            },
            "contextImages": {},
            "histograms": {},
            "chordDiagrams": {},
            "ternaryPlots": {},
            "binaryPlots": {},
            "tables": {},
            "roiQuantTables": {},
            "variograms": {},
            "spectrums": {},
            "rgbuPlots": {},
            "singleAxisRGBU": {},
            "rgbuImages": {},
            "parallelograms": {},
            "annotations": {
                "savedAnnotations": []
            },
            "rois": {
                "roiColours": {},
                "roiShapes": {}
            },
            "quantification": {
                "appliedQuantID": "quant2"
            },
            "selection": {
                "roiID": "",
                "roiName": "",
                "locIdxs": []
            }
        },
        "The first one": {
            "analysisLayout": {
                "topWidgetSelectors": [],
                "bottomWidgetSelectors": []
            },
            "spectrum": {
                "panX": 0,
                "panY": 0,
                "zoomX": 1,
                "zoomY": 1,
                "spectrumLines": [],
                "logScale": true,
                "xrflines": [],
                "showXAsEnergy": false,
                "energyCalibration": []
            },
            "contextImages": {},
            "histograms": {},
            "chordDiagrams": {},
            "ternaryPlots": {},
            "binaryPlots": {},
            "tables": {},
            "roiQuantTables": {},
            "variograms": {},
            "spectrums": {},
            "rgbuPlots": {},
            "singleAxisRGBU": {},
            "rgbuImages": {},
            "parallelograms": {},
            "annotations": {
                "savedAnnotations": []
            },
            "rois": {
                "roiColours": {},
                "roiShapes": {}
            },
            "quantification": {
                "appliedQuantID": "quant1"
            },
            "selection": {
                "roiID": "",
                "roiName": "",
                "locIdxs": []
            }
        }
    },
    "shared": true,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/view-state-collection/TheDataSetID/331", bytes.NewReader([]byte{}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/view-state-collection/TheDataSetID/332", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Referenced view state file not found
	req, _ = http.NewRequest("POST", "/share/view-state-collection/TheDataSetID/333", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File and view states found, share OK
	req, _ = http.NewRequest("POST", "/share/view-state-collection/TheDataSetID/334", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 331 not found
	//
	// 404
	// 332 not found
	//
	// 404
	// The first one not found
	//
	// 200
	// "334 shared"
}
