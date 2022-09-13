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

func Example_viewStateHandler_ListSaved() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	viewStateSavedS3Path := viewStateS3Path + "Workspaces"
	sharedViewStateSavedS3Path := sharedViewStateS3Path + "Workspaces"

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(sharedViewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(sharedViewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(sharedViewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateSavedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(sharedViewStateSavedS3Path),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		nil,
		nil,
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateSavedS3Path + "/viewstate111.json")},
				{Key: aws.String(viewStateSavedS3Path + "/viewstate222.json")},
				{Key: aws.String(viewStateSavedS3Path + "/viewstate333.json")},
			},
		},
		nil,
		nil,
		{
			Contents: []*s3.Object{
				{Key: aws.String(sharedViewStateSavedS3Path + "/forall.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateSavedS3Path + "/viewstate111.json")},
				{Key: aws.String(viewStateSavedS3Path + "/viewstate222.json")},
				{Key: aws.String(viewStateSavedS3Path + "/viewstate333.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(sharedViewStateSavedS3Path + "/forall.json")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate111.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate222.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate333.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateSavedS3Path + "/forall.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate111.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate222.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateSavedS3Path + "/viewstate333.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateSavedS3Path + "/forall.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate111",
    "description": "viewstate111",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate222",
    "description": "viewstate222",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate333",
    "description": "viewstate333",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "forall",
    "description": "forall",
    "shared": true,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate111",
    "description": "viewstate111",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate222",
    "description": "viewstate222",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "viewstate333",
    "description": "viewstate333",
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "name": "forall",
    "description": "forall",
    "shared": true,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// None
	req, _ := http.NewRequest("GET", "/view-state/saved/TheDataSetID", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Only user
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Only shared
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Both
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID", bytes.NewReader([]byte("")))
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
	//         "id": "viewstate111",
	//         "name": "viewstate111",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     {
	//         "id": "viewstate222",
	//         "name": "viewstate222",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     {
	//         "id": "viewstate333",
	//         "name": "viewstate333",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// ]
	//
	// 200
	// [
	//     {
	//         "id": "shared-forall",
	//         "name": "forall",
	//         "shared": true,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// ]
	//
	// 200
	// [
	//     {
	//         "id": "viewstate111",
	//         "name": "viewstate111",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     {
	//         "id": "viewstate222",
	//         "name": "viewstate222",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     {
	//         "id": "viewstate333",
	//         "name": "viewstate333",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     {
	//         "id": "shared-forall",
	//         "name": "forall",
	//         "shared": true,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// ]
}

func Example_viewStateHandler_GetSaved() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111"
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        },
    "name": "",
    "description": ""
}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("GET", "/view-state/saved/TheDataSetID/viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, success
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// viewstate123 not found
	//
	// 200
	// {
	//     "viewState": {
	//         "analysisLayout": {
	//             "topWidgetSelectors": [],
	//             "bottomWidgetSelectors": []
	//         },
	//         "spectrum": {
	//             "panX": 0,
	//             "panY": 0,
	//             "zoomX": 1,
	//             "zoomY": 1,
	//             "spectrumLines": [],
	//             "logScale": true,
	//             "xrflines": [],
	//             "showXAsEnergy": false,
	//             "energyCalibration": []
	//         },
	//         "contextImages": {},
	//         "histograms": {},
	//         "chordDiagrams": {},
	//         "ternaryPlots": {},
	//         "binaryPlots": {},
	//         "tables": {},
	//         "roiQuantTables": {},
	//         "variograms": {},
	//         "spectrums": {},
	//         "rgbuPlots": {},
	//         "singleAxisRGBU": {},
	//         "rgbuImages": {},
	//         "parallelograms": {},
	//         "rois": {
	//             "roiColours": {
	//                 "roi22": "rgba(128,0,255,0.5)",
	//                 "roi99": "rgba(255,255,0,1)"
	//             }
	//         },
	//         "quantification": {
	//             "appliedQuantID": "quant111"
	//         },
	//         "selection": {
	//             "roiID": "roi12345",
	//             "roiName": "The best region",
	//             "locIdxs": [
	//                 3,
	//                 5,
	//                 7
	//             ]
	//         }
	//     },
	//     "name": "",
	//     "description": ""
	// }
}

func Example_viewStateHandler_GetSaved_ROIQuantFallbackCheck() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "",
            "quantificationByROI": {
                "roi22": "quant222",
                "roi88": "quant333"
            }
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        },
    "name": "",
    "description": ""
}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("GET", "/view-state/saved/TheDataSetID/viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, success
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// TODO: fix this, sometimes this can result in last quant being quant333, likely due to some map reading ordering issue

	// Output:
	// 404
	// viewstate123 not found
	//
	// 200
	// {
	//     "viewState": {
	//         "analysisLayout": {
	//             "topWidgetSelectors": [],
	//             "bottomWidgetSelectors": []
	//         },
	//         "spectrum": {
	//             "panX": 0,
	//             "panY": 0,
	//             "zoomX": 1,
	//             "zoomY": 1,
	//             "spectrumLines": [],
	//             "logScale": true,
	//             "xrflines": [],
	//             "showXAsEnergy": false,
	//             "energyCalibration": []
	//         },
	//         "contextImages": {},
	//         "histograms": {},
	//         "chordDiagrams": {},
	//         "ternaryPlots": {},
	//         "binaryPlots": {},
	//         "tables": {},
	//         "roiQuantTables": {},
	//         "variograms": {},
	//         "spectrums": {},
	//         "rgbuPlots": {},
	//         "singleAxisRGBU": {},
	//         "rgbuImages": {},
	//         "parallelograms": {},
	//         "rois": {
	//             "roiColours": {
	//                 "roi22": "rgba(128,0,255,0.5)",
	//                 "roi99": "rgba(255,255,0,1)"
	//             }
	//         },
	//         "quantification": {
	//             "appliedQuantID": "quant222"
	//         },
	//         "selection": {
	//             "roiID": "roi12345",
	//             "roiName": "The best region",
	//             "locIdxs": [
	//                 3,
	//                 5,
	//                 7
	//             ]
	//         }
	//     },
	//     "name": "",
	//     "description": ""
	// }
}

func Example_viewStateHandler_GetSavedShared() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/viewstate123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111"
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        },
    "name": "",
    "description": ""
}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("GET", "/view-state/saved/TheDataSetID/shared-viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, success
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID/shared-viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// viewstate123 not found
	//
	// 200
	// {
	//     "viewState": {
	//         "analysisLayout": {
	//             "topWidgetSelectors": [],
	//             "bottomWidgetSelectors": []
	//         },
	//         "spectrum": {
	//             "panX": 0,
	//             "panY": 0,
	//             "zoomX": 1,
	//             "zoomY": 1,
	//             "spectrumLines": [],
	//             "logScale": true,
	//             "xrflines": [],
	//             "showXAsEnergy": false,
	//             "energyCalibration": []
	//         },
	//         "contextImages": {},
	//         "histograms": {},
	//         "chordDiagrams": {},
	//         "ternaryPlots": {},
	//         "binaryPlots": {},
	//         "tables": {},
	//         "roiQuantTables": {},
	//         "variograms": {},
	//         "spectrums": {},
	//         "rgbuPlots": {},
	//         "singleAxisRGBU": {},
	//         "rgbuImages": {},
	//         "parallelograms": {},
	//         "rois": {
	//             "roiColours": {
	//                 "roi22": "rgba(128,0,255,0.5)",
	//                 "roi99": "rgba(255,255,0,1)"
	//             }
	//         },
	//         "quantification": {
	//             "appliedQuantID": "quant111"
	//         },
	//         "selection": {
	//             "roiID": "roi12345",
	//             "roiName": "The best region",
	//             "locIdxs": [
	//                 3,
	//                 5,
	//                 7
	//             ]
	//         }
	//     },
	//     "name": "",
	//     "description": ""
	// }
}

func Example_viewStateHandler_PutSaved_Force() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate123.json"), Body: bytes.NewReader([]byte(`{
    "viewState": {
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
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111"
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        }
    },
    "name": "viewstate123",
    "description": "",
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

	req, _ := http.NewRequest("PUT", "/view-state/saved/TheDataSetID/viewstate123?force=true", bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111",
            "quantificationByROI": {
                "roi22": "quant222",
                "roi88": "quant333"
            }
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        }
    },
    "name": "viewstate123 INCORRECT VIEW STATE SHOULD BE REPLACED!"
}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
}

func Example_viewStateHandler_PutSaved_OverwriteFail() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Checking if it exists
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate123.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111"
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        },
    "name": "",
    "description": ""
}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("PUT", "/view-state/saved/TheDataSetID/viewstate123", bytes.NewReader([]byte(`{
    "viewState": {
        "analysisLayout": {
            "bottomWidgetSelectors": []
        },
        "rois": {
            "roiColours": {
                "roi22": "rgba(128,0,255,0.5)",
                "roi99": "rgba(255,255,0,1)"
            }
        },
        "quantification": {
            "appliedQuantID": "quant111",
            "quantificationByROI": {
                "roi22": "quant222",
                "roi88": "quant333"
            }
        },
        "selection": {
            "roiID": "roi12345",
            "roiName": "The best region",
            "locIdxs": [
                3,
                5,
                7
            ]
        }
    },
    "name": "viewstate123 INCORRECT VIEW STATE SHOULD BE REPLACED!"
}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 409
	// viewstate123 already exists
}

func Example_viewStateHandler_DeleteSaved() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	collectionRoot := "UserContent/600f2a0806b6c70071d3d174/TheDataSetID/ViewState/WorkspaceCollections"

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate123.json"),
		},

		// Test 2: no collections
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},

		// Test 3: not in collections
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(collectionRoot + "/a collection.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(collectionRoot + "/Another-Collection.json"),
		},

		// Test 4: found in collection
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(collectionRoot + "/culprit.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		// Test 1: no view state file
		nil,

		// Test 2: exists
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "viewstate555", "viewState": {"selection": {}}}`))),
		},

		// Test 3: exists + collections returned
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "viewstate555", "viewState": {"selection": {}}}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "a collection", "viewStateIDs": ["some view state", "another"]}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "Another-Collection", "viewStateIDs": ["also not the one"]}`))),
		},

		// Test 4: exists + collections returned, one contains this view state!
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "viewstate555", "viewState": {"selection": {}}}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"name": "culprit", "viewStateIDs": ["some view state", "viewstate555", "another"]}`))),
		},
	}

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(collectionRoot),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(collectionRoot),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(collectionRoot),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		// Test 2: no collections
		{
			Contents: []*s3.Object{},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(collectionRoot + "/a collection.json"), LastModified: aws.Time(time.Unix(1634731920, 0))},
				{Key: aws.String(collectionRoot + "/Another-Collection.json"), LastModified: aws.Time(time.Unix(1634731921, 0))},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(collectionRoot + "/culprit.json"), LastModified: aws.Time(time.Unix(1634731922, 0))},
				{Key: aws.String(collectionRoot + "/Another-Collection.json"), LastModified: aws.Time(time.Unix(1634731923, 0))},
			},
		},
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Doesn't exist, should fail
	req, _ := http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, no collections, success
	req, _ = http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, collections checked (not in there), success
	req, _ = http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists but is in a collection, fail
	req, _ = http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/viewstate555", bytes.NewReader([]byte("")))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// viewstate123 not found
	//
	// 200
	//
	// 200
	//
	// 409
	// Workspace "viewstate555" is in collection "culprit". Please delete the workspace from all collections before before trying to delete it.
}

func Example_viewStateHandler_DeleteSavedShared() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1: not owned by user
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/viewstate123.json"),
		},

		// Test 2: owned by user
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"name": "viewstate123",
				"viewState": {"selection": {}},
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
				"name": "viewstate555",
				"viewState": {"selection": {}},
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/viewstate555.json"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Not owned by user, should fail
	req, _ := http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/shared-viewstate123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Exists, owned by user, success
	req, _ = http.NewRequest("DELETE", "/view-state/saved/TheDataSetID/shared-viewstate555", bytes.NewReader([]byte("")))
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

func Example_viewStateHandler_GetReferencedIDs() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/331.json"),
		},
		// Test 2
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/332.json"),
		},
		// Test 3
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/333.json"),
		},
		// Getting ROIs
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/TheDataSetID/ROI.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/TheDataSetID/ROI.json"),
		},
		// Getting expressions
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/DataExpressions.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/DataExpressions.json"),
		},
		// Getting rgb mixes
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/RGBMixes.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/RGBMixes.json"),
		},
		// Getting quant config
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/TheDataSetID/Quantifications/summary-quant123.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`}`))),
		},
		{
			// View state that references non-shared IDs. We want to make sure it returns the right ones and
			// count, so we return multiple IDs here:
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				 "viewState": {
					 "contextImages": { "0": { "mapLayers": [ { "expressionID": "rgbmix-123", "opacity": 1, "visible": true } ] } },
					 "quantification": {"appliedQuantID": "quant123"},
					 "binaryPlots": { "44": { "expressionIDs": ["shared-expr", "expr1"], "visibleROIs": ["shared-roi"] } },
					 "ternaryPlots": { "66": { "expressionIDs": ["shared-expr2"], "visibleROIs": ["roi2"] } }
				 },
				 "name": "333",
				"description": "the description of 333"
			}`))),
		},
		// ROIs
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roi2": {
		"name": "Dark patch 2",
		"description": "The second dark patch",
		"locationIndexes": [4, 55, 394],
		"creator": { "name": "Peter", "user_id": "u123" }
	}
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roi": {
		"name": "Shared patch 2",
		"description": "The shared patch",
		"locationIndexes": [4, 55, 394],
		"creator": { "name": "PeterN", "user_id": "u123" }
	}
}`))),
		},
		// Expressions
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"abc123": {
		"name": "Temp data",
		"expression": "housekeeping(\"something\")",
		"type": "All",
		"comments": "comments for abc123 expression",
		"creator": {
			"user_id": "444",
			"name": "Niko",
			"email": "niko@spicule.co.uk"
		}
	},
	"expr1": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "All",
		"comments": "comments for expr1",
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "peter@spicule.co.uk"
		}
	}
}`))),
		},
		nil, // simulate missing shared expressions file...
		// User RGB mixes
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"123": {
		"name": "Ca-Ti-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
		// Shared RGB mixes
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"380": {
		"name": "Fe-Ca-Al ratios",
		"red": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 2.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 3.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
		// Quant summary
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 93,
		"name": "my test quant",
		"dataBucket": "dev-pixlise-data",
		"datasetPath": "Datasets/rtt-456/5x5dataset.bin",
		"datasetID": "rtt-456",
		"jobBucket": "dev-pixlise-piquant-jobs",
		"detectorConfig": "PIXL",
		"elements": [
			"Sc",
			"Cr"
		],
		"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
		"runTimeSec": 120,
		"coresPerNode": 6,
		"startUnixTime": 1589948988,
		"creator": {
			"name": "peternemere",
			"user_id": "600f2a0806b6c70071d3d174",
            "email": ""
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": "",
		"quantMode": "AB"
	},
	"jobId": "quant123",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	]
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// User file not there, should say not found
	req, _ := http.NewRequest("GET", "/view-state/saved/TheDataSetID/331/references", bytes.NewReader([]byte{}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID/332/references", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Gets mix of shared and not shared IDs
	req, _ = http.NewRequest("GET", "/view-state/saved/TheDataSetID/333/references", bytes.NewReader([]byte{}))
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
	// 200
	// {
	//     "quant": {
	//         "id": "quant123",
	//         "name": "my test quant",
	//         "creator": {
	//             "name": "peternemere",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": ""
	//         }
	//     },
	//     "ROIs": [
	//         {
	//             "id": "roi2",
	//             "name": "Dark patch 2",
	//             "creator": {
	//                 "name": "Peter",
	//                 "user_id": "u123",
	//                 "email": ""
	//             }
	//         },
	//         {
	//             "id": "shared-roi",
	//             "name": "Shared patch 2",
	//             "creator": {
	//                 "name": "PeterN",
	//                 "user_id": "u123",
	//                 "email": ""
	//             }
	//         }
	//     ],
	//     "expressions": [
	//         {
	//             "id": "expr1",
	//             "name": "Calcium weight%",
	//             "creator": {
	//                 "name": "Peter N",
	//                 "user_id": "999",
	//                 "email": "peter@spicule.co.uk"
	//             }
	//         },
	//         {
	//             "id": "shared-expr",
	//             "name": "",
	//             "creator": {
	//                 "name": "",
	//                 "user_id": "",
	//                 "email": ""
	//             }
	//         },
	//         {
	//             "id": "shared-expr2",
	//             "name": "",
	//             "creator": {
	//                 "name": "",
	//                 "user_id": "",
	//                 "email": ""
	//             }
	//         }
	//     ],
	//     "rgbMixes": [
	//         {
	//             "id": "rgbmix-123",
	//             "name": "",
	//             "creator": {
	//                 "name": "",
	//                 "user_id": "",
	//                 "email": ""
	//             }
	//         }
	//     ],
	//     "nonSharedCount": 3
	// }
}

func Example_viewStateHandler_ShareViewState() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/331.json"),
		},
		// Test 2
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/332.json"),
		},
		// Test 3
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/333.json"),
		},
		// Test 4
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/334.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`}`))),
		},
		{
			// View state that references non-shared IDs. We want to make sure it returns the right ones and
			// count, so we return multiple IDs here:
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				 "viewState": {
					"quantification": {"appliedQuantID": "quant123"},
					"binaryPlots": { "44": { "expressionIDs": ["shared-expr", "expr1"], "visibleROIs": ["shared-roi"] } },
					"ternaryPlots": { "66": { "expressionIDs": ["shared-expr2"], "visibleROIs": ["roi2"] } }
				 },
				 "name": "333",
				"description": "the description of 333"
			}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"viewState": {
					"quantification": {"appliedQuantID": "shared-quant123"},
					"binaryPlots": { "77": { "expressionIDs": ["shared-expr", "shared-expr1"], "visibleROIs": ["shared-roi"] } },
					"ternaryPlots": { "99": { "expressionIDs": ["shared-expr2"], "visibleROIs": ["shared-roi2"] } }
				},
				 "name": "334",
				"description": "the description of 334"
			}`))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(sharedViewStateS3Path + "Workspaces/334.json"), Body: bytes.NewReader([]byte(`{
    "viewState": {
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
        "ternaryPlots": {
            "99": {
                "showMmol": false,
                "expressionIDs": [
                    "shared-expr2"
                ],
                "visibleROIs": [
                    "shared-roi2"
                ]
            }
        },
        "binaryPlots": {
            "77": {
                "showMmol": false,
                "expressionIDs": [
                    "shared-expr",
                    "shared-expr1"
                ],
                "visibleROIs": [
                    "shared-roi"
                ]
            }
        },
        "tables": {},
        "roiQuantTables": {},
        "variograms": {},
        "spectrums": {},
        "rgbuPlots": {},
        "singleAxisRGBU": {},
        "rgbuImages": {},
        "parallelograms": {},
        "rois": {
            "roiColours": {}
        },
        "quantification": {
            "appliedQuantID": "shared-quant123"
        },
        "selection": {
            "roiID": "",
            "roiName": "",
            "locIdxs": []
        }
    },
    "name": "334",
    "description": "the description of 334",
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
	req, _ := http.NewRequest("POST", "/share/view-state/TheDataSetID/331", bytes.NewReader([]byte{}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/view-state/TheDataSetID/332", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Share failed because of non-shared ids referenced by workspace
	req, _ = http.NewRequest("POST", "/share/view-state/TheDataSetID/333", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Share OK
	req, _ = http.NewRequest("POST", "/share/view-state/TheDataSetID/334", bytes.NewReader([]byte{}))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Sharing a shared one - should fail
	req, _ = http.NewRequest("POST", "/share/view-state/TheDataSetID/shared-335", bytes.NewReader([]byte{}))
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
	// 400
	// Cannot share workspaces if they reference non-shared objects
	//
	// 200
	// "334 shared"
	//
	// 400
	// Cannot share a shared ID
}

// Shares a view state, with automatic sharing of referenced items turned on
func Example_viewStateHandler_ShareViewState_AutoShare() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// Test 1
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "Workspaces/222.json"),
		},

		// Getting ROIs to be able to share...
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/TheDataSetID/ROI.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/TheDataSetID/ROI.json"),
		},
		// Getting expressions to be able to share...
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/DataExpressions.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/DataExpressions.json"),
		},
		// Getting rgb mixes
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/RGBMixes.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/RGBMixes.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			// View state that references non-shared IDs. We want to make sure it returns the right ones and
			// count, so we return multiple IDs here:
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				 "viewState": {
					"contextImages": { "0": { "mapLayers": [ { "expressionID": "rgbmix-123", "opacity": 1, "visible": true } ] } },
					"quantification": {"appliedQuantID": "shared-quant123"},
					"binaryPlots": { "44": { "expressionIDs": ["shared-expr", "expr1"], "visibleROIs": ["shared-roi"] } },
					"ternaryPlots": { "66": { "expressionIDs": ["shared-expr2"], "visibleROIs": ["roi2"] } }
				 },
				 "name": "222",
				"description": "the description of 222"
			}`))),
		},

		// ROIs
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roi2": {
		"name": "Dark patch 2",
		"description": "The second dark patch",
		"locationIndexes": [4, 55, 394],
		"creator": { "name": "Peter", "user_id": "u123" }
	}
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roi": {
		"name": "Shared patch 2",
		"shared": true,
		"description": "The shared patch",
		"locationIndexes": [4, 55, 394],
		"creator": { "name": "PeterN", "user_id": "u123" }
	}
}`))),
		},
		// Expressions
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"abc123": {
		"name": "Temp data",
		"expression": "housekeeping(\"something\")",
		"type": "All",
		"comments": "comments for abc123 expression",
		"creator": {
			"user_id": "444",
			"name": "Niko",
			"email": "niko@spicule.co.uk"
		}
	},
	"expr1": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "All",
		"comments": "comments for expr1",
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "peter@spicule.co.uk"
		}
	}
}`))),
		},
		nil, // simulate missing shared expressions file...
		// User RGB mixes
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"rgbmix-123": {
		"name": "Ca-Ti-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
		// Shared RGB mixes
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"380": {
		"name": "Fe-Ca-Al ratios",
		"red": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 2.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 3.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
		"shared": true,
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/TheDataSetID/ROI.json"), Body: bytes.NewReader([]byte(`{
    "roi": {
        "name": "Shared patch 2",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "description": "The shared patch",
        "shared": true,
        "creator": {
            "name": "PeterN",
            "user_id": "u123",
            "email": ""
        }
    },
    "roi2(sh)": {
        "name": "Dark patch 2",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "description": "The second dark patch",
        "shared": true,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": ""
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/DataExpressions.json"), Body: bytes.NewReader([]byte(`{
    "expr1(sh)": {
        "name": "Calcium weight%",
        "expression": "element(\"Ca\", \"%\")",
        "type": "All",
        "comments": "comments for expr1",
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/RGBMixes.json"), Body: bytes.NewReader([]byte(`{
    "380": {
        "name": "Fe-Ca-Al ratios",
        "red": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2.5,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 3.5,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3.5,
            "rangeMax": 6.3
        },
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "rgbmix-123roi": {
        "name": "Ca-Ti-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.5,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.5,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3.5,
            "rangeMax": 6.3
        },
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/TheDataSetID/ViewState/Workspaces/222.json"), Body: bytes.NewReader([]byte(`{
    "viewState": {
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
        "contextImages": {
            "0": {
                "panX": 0,
                "panY": 0,
                "zoomX": 0,
                "zoomY": 0,
                "showPoints": false,
                "showPointBBox": false,
                "pointColourScheme": "",
                "pointBBoxColourScheme": "",
                "contextImage": "",
                "contextImageSmoothing": "",
                "mapLayers": [
                    {
                        "expressionID": "rgbmix-123",
                        "opacity": 1,
                        "visible": true,
                        "displayValueRangeMin": 0,
                        "displayValueRangeMax": 0,
                        "displayValueShading": ""
                    }
                ],
                "roiLayers": null,
                "elementRelativeShading": false,
                "brightness": 0,
                "rgbuChannels": "",
                "unselectedOpacity": 0,
                "unselectedGrayscale": false,
                "colourRatioMin": 0,
                "colourRatioMax": 0,
                "removeTopSpecularArtifacts": false,
                "removeBottomSpecularArtifacts": false
            }
        },
        "histograms": {},
        "chordDiagrams": {},
        "ternaryPlots": {
            "66": {
                "showMmol": false,
                "expressionIDs": [
                    "shared-expr2"
                ],
                "visibleROIs": [
                    "shared-roi2(sh)"
                ]
            }
        },
        "binaryPlots": {
            "44": {
                "showMmol": false,
                "expressionIDs": [
                    "shared-expr",
                    "shared-expr1(sh)"
                ],
                "visibleROIs": [
                    "shared-roi"
                ]
            }
        },
        "tables": {},
        "roiQuantTables": {},
        "variograms": {},
        "spectrums": {},
        "rgbuPlots": {},
        "singleAxisRGBU": {},
        "rgbuImages": {},
        "parallelograms": {},
        "rois": {
            "roiColours": {}
        },
        "quantification": {
            "appliedQuantID": "shared-quant123"
        },
        "selection": {
            "roiID": "",
            "roiName": "",
            "locIdxs": []
        }
    },
    "name": "222",
    "description": "the description of 222",
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
		{},
		{},
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"roi2(sh)", "expr1(sh)", "123roi"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/view-state/TheDataSetID/222?auto-share=true", bytes.NewReader([]byte{}))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// "222 shared"
}
