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
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/notifications"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

const viewStateS3Path = "UserContent/600f2a0806b6c70071d3d174/TheDataSetID/ViewState/"
const sharedViewStateS3Path = "UserContent/shared/TheDataSetID/ViewState/"

func Example_viewStateHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const contextImgJSON = `
	"zoomX": 2,
	"showPoints": true,
	"showPointBBox": false,
	"pointColourScheme": "BW",
	"pointBBoxColourScheme": "PURPLE_CYAN",
	"contextImageSmoothing": "linear",
    "mapLayers": [
        {
			"expressionID": "Fe",
			"opacity": 0.3,
			"visible": true,
			"displayValueRangeMin": 12,
			"displayValueRangeMax": 48.8,
			"displayValueShading": "SHADE_VIRIDIS"
        }
	],
	"roiLayers": [
		{
			"roiID": "roi123",
			"opacity": 0.7,
			"visible": true
		}
	],
	"contextImage": "file1.jpg",
	"elementRelativeShading": true
}`

	// Single request results in loading multiple files from S3. First it gets a directory listing...
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateS3Path),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateS3Path + "not-a-widget.json")}, // Not a recognised file name
				{Key: aws.String(viewStateS3Path + "spectrum.json")},
				{Key: aws.String(viewStateS3Path + "spectrum.txt")}, // Not right extension
				{Key: aws.String(viewStateS3Path + "contextImage-map.json")},
				{Key: aws.String(viewStateS3Path + "contextImage-analysis.json")},
				{Key: aws.String(viewStateS3Path + "contextImage-engineering.json")},
				{Key: aws.String(viewStateS3Path + "quantification.json")},
				{Key: aws.String(viewStateS3Path + "selection.json")},
				{Key: aws.String(viewStateS3Path + "roi.json")},
				{Key: aws.String(viewStateS3Path + "analysisLayout.json")},
				{Key: aws.String(viewStateS3Path + "histogram-1.json")},
				{Key: aws.String(viewStateS3Path + "chord-0.json")},
				{Key: aws.String(viewStateS3Path + "chord-1.json")},
				{Key: aws.String(viewStateS3Path + "table-undercontext.json")},
				{Key: aws.String(viewStateS3Path + "table-underspectrum0.json")},
				{Key: aws.String(viewStateS3Path + "binary-underspectrum0.json")},
				{Key: aws.String(viewStateS3Path + "ternary-underspectrum2.json")},
				{Key: aws.String(viewStateS3Path + "variogram-abc123.json")},
				{Key: aws.String(viewStateS3Path + "rgbuImages-33.json")},
				{Key: aws.String(viewStateS3Path + "rgbuPlot-underspectrum1.json")},
				{Key: aws.String(viewStateS3Path + "parallelogram-55.json")},
				{Key: aws.String(viewStateS3Path + "roiQuantTable-ttt.json")},
				{Key: aws.String(viewStateS3Path + "spectrum-top1.json")}, // the "new style" version that comes with a position id
			},
		},
	}

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-map.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-analysis.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-engineering.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "quantification.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "selection.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roi.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "analysisLayout.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "histogram-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "chord-0.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "chord-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "table-undercontext.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "table-underspectrum0.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "binary-underspectrum0.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "ternary-underspectrum2.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "variogram-abc123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuImages-33.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuPlot-underspectrum1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "parallelogram-55.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roiQuantTable-ttt.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum-top1.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "xrflines": [
        {
            "visible": true,
            "line_info": {
                "Z": 12,
                "K": true,
                "L": true,
                "M": true,
                "Esc": true
            }
        }
	],
	"panX": 32,
	"zoomY": 3,
	"logScale": false,
	"energyCalibration": [
		{
			"detector": "A",
			"eVStart": 0.1,
			"eVPerChannel": 10.3
		}
	]
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"panY": 10,` + contextImgJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"panY": 11,` + contextImgJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"panY": 12,` + contextImgJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"appliedQuantID": "quant111",
	"quantificationByROI": {
		"roi22": "quant222",
		"roi88": "quant333"
	}
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roiID": "roi12345",
	"roiName": "The best region",
	"locIdxs": [3,5,7],
	"pixelSelectionImageName": "image.tif",
	"pixelIdxs": [9],
    "cropPixelIdxs": [9]
}`))),
		},
		{ // roi
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roiColours": {
		"roi99": "rgba(255,255,0,1)",
		"roi22": "rgba(128,0,255,0.5)"
	},
	"roiShapes": {}
}`))),
		},
		{ // analysisLayout
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"topWidgetSelectors": [
		"context-image",
		"spectrum-widget"
	],
	"bottomWidgetSelectors": [
		"table-widget",
		"binary-plot-widget",
		"rgbu-plot-widget",
		"ternary-plot-widget"
	]
}`))),
		},
		nil, // quant histogram
		nil, // chord-0
		nil, // chord-1
		{ // table
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"showPureElements": true}`))),
		},
		{ // table 2
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"showPureElements": true}`))),
		},
		{ // binary
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{ // ternary
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{ // variogram
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{ // rgbuImages
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{ "brightness": 1.2 }`))),
		},
		{ // rgbuPlot
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{ "yChannelA": "B" }`))),
		},
		{ // parallelogram
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{ "colourChannels": ["R", "G"] }`))),
		},
		{ // roiQuantTable
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{ "quantIDs": ["quant1", "quant2"], "roi": "the-roi" }`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "xrflines": [
        {
            "visible": false,
            "line_info": {
                "Z": 12,
                "K": true,
                "L": true,
                "M": true,
                "Esc": true
            }
        }
	],
	"panX": 30,
	"zoomY": 1,
	"logScale": false,
	"energyCalibration": [
		{
			"detector": "A",
			"eVStart": 0.1,
			"eVPerChannel": 10.3
		}
	]
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Various bits should return in the response...
	req, _ := http.NewRequest("GET", "/view-state/TheDataSetID", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	//
	// 200
	// {
	//     "analysisLayout": {
	//         "topWidgetSelectors": [
	//             "context-image",
	//             "spectrum-widget"
	//         ],
	//         "bottomWidgetSelectors": [
	//             "table-widget",
	//             "binary-plot-widget",
	//             "rgbu-plot-widget",
	//             "ternary-plot-widget"
	//         ]
	//     },
	//     "spectrum": {
	//         "panX": 32,
	//         "panY": 0,
	//         "zoomX": 1,
	//         "zoomY": 3,
	//         "spectrumLines": [],
	//         "logScale": false,
	//         "xrflines": [
	//             {
	//                 "line_info": {
	//                     "Z": 12,
	//                     "K": true,
	//                     "L": true,
	//                     "M": true,
	//                     "Esc": true
	//                 },
	//                 "visible": true
	//             }
	//         ],
	//         "showXAsEnergy": false,
	//         "energyCalibration": [
	//             {
	//                 "detector": "A",
	//                 "eVStart": 0.1,
	//                 "eVPerChannel": 10.3
	//             }
	//         ]
	//     },
	//     "contextImages": {
	//         "analysis": {
	//             "panX": 0,
	//             "panY": 11,
	//             "zoomX": 2,
	//             "zoomY": 1,
	//             "showPoints": true,
	//             "showPointBBox": false,
	//             "pointColourScheme": "BW",
	//             "pointBBoxColourScheme": "PURPLE_CYAN",
	//             "contextImage": "file1.jpg",
	//             "contextImageSmoothing": "linear",
	//             "mapLayers": [
	//                 {
	//                     "expressionID": "Fe",
	//                     "opacity": 0.3,
	//                     "visible": true,
	//                     "displayValueRangeMin": 12,
	//                     "displayValueRangeMax": 48.8,
	//                     "displayValueShading": "SHADE_VIRIDIS"
	//                 }
	//             ],
	//             "roiLayers": [
	//                 {
	//                     "roiID": "roi123",
	//                     "opacity": 0.7,
	//                     "visible": true
	//                 }
	//             ],
	//             "elementRelativeShading": true,
	//             "brightness": 1,
	//             "rgbuChannels": "RGB",
	//             "unselectedOpacity": 0.4,
	//             "unselectedGrayscale": false,
	//             "colourRatioMin": 0,
	//             "colourRatioMax": 0,
	//             "removeTopSpecularArtifacts": false,
	//             "removeBottomSpecularArtifacts": false
	//         },
	//         "map": {
	//             "panX": 0,
	//             "panY": 10,
	//             "zoomX": 2,
	//             "zoomY": 1,
	//             "showPoints": true,
	//             "showPointBBox": false,
	//             "pointColourScheme": "BW",
	//             "pointBBoxColourScheme": "PURPLE_CYAN",
	//             "contextImage": "file1.jpg",
	//             "contextImageSmoothing": "linear",
	//             "mapLayers": [
	//                 {
	//                     "expressionID": "Fe",
	//                     "opacity": 0.3,
	//                     "visible": true,
	//                     "displayValueRangeMin": 12,
	//                     "displayValueRangeMax": 48.8,
	//                     "displayValueShading": "SHADE_VIRIDIS"
	//                 }
	//             ],
	//             "roiLayers": [
	//                 {
	//                     "roiID": "roi123",
	//                     "opacity": 0.7,
	//                     "visible": true
	//                 }
	//             ],
	//             "elementRelativeShading": true,
	//             "brightness": 1,
	//             "rgbuChannels": "RGB",
	//             "unselectedOpacity": 0.4,
	//             "unselectedGrayscale": false,
	//             "colourRatioMin": 0,
	//             "colourRatioMax": 0,
	//             "removeTopSpecularArtifacts": false,
	//             "removeBottomSpecularArtifacts": false
	//         }
	//     },
	//     "histograms": {},
	//     "chordDiagrams": {},
	//     "ternaryPlots": {
	//         "underspectrum2": {
	//             "showMmol": false,
	//             "expressionIDs": [],
	//             "visibleROIs": []
	//         }
	//     },
	//     "binaryPlots": {
	//         "underspectrum0": {
	//             "showMmol": false,
	//             "expressionIDs": [],
	//             "visibleROIs": []
	//         }
	//     },
	//     "tables": {
	//         "undercontext": {
	//             "showPureElements": true,
	//             "order": "atomic-number",
	//             "visibleROIs": []
	//         }
	//     },
	//     "roiQuantTables": {},
	//     "variograms": {},
	//     "spectrums": {
	//         "top1": {
	//             "panX": 30,
	//             "panY": 0,
	//             "zoomX": 1,
	//             "zoomY": 1,
	//             "spectrumLines": [],
	//             "logScale": false,
	//             "xrflines": [
	//                 {
	//                     "line_info": {
	//                         "Z": 12,
	//                         "K": true,
	//                         "L": true,
	//                         "M": true,
	//                         "Esc": true
	//                     },
	//                     "visible": false
	//                 }
	//             ],
	//             "showXAsEnergy": false,
	//             "energyCalibration": [
	//                 {
	//                     "detector": "A",
	//                     "eVStart": 0.1,
	//                     "eVPerChannel": 10.3
	//                 }
	//             ]
	//         }
	//     },
	//     "rgbuPlots": {
	//         "underspectrum1": {
	//             "minerals": [],
	//             "yChannelA": "B",
	//             "yChannelB": "",
	//             "xChannelA": "",
	//             "xChannelB": "",
	//             "drawMonochrome": false
	//         }
	//     },
	//     "singleAxisRGBU": {},
	//     "rgbuImages": {},
	//     "parallelograms": {},
	//     "annotations": {
	//         "savedAnnotations": []
	//     },
	//     "rois": {
	//         "roiColours": {
	//             "roi22": "rgba(128,0,255,0.5)",
	//             "roi99": "rgba(255,255,0,1)"
	//         },
	//         "roiShapes": {}
	//     },
	//     "quantification": {
	//         "appliedQuantID": "quant111"
	//     },
	//     "selection": {
	//         "roiID": "roi12345",
	//         "roiName": "The best region",
	//         "locIdxs": [
	//             3,
	//             5,
	//             7
	//         ],
	//         "pixelSelectionImageName": "image.tif",
	//         "pixelIdxs": [
	//             9
	//         ],
	//         "cropPixelIdxs": [
	//             9
	//         ]
	//     }
	// }
}

func Example_viewStateHandler_List_WithReset() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Single request results in loading multiple files from S3. First it gets a directory listing...
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateS3Path),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateS3Path + "not-a-widget.json")},                    // Not a recognised file name
				{Key: aws.String(viewStateS3Path + "Workspaces/workspace.json")},            // workspace file, should not be deleted
				{Key: aws.String(viewStateS3Path + "WorkspaceCollections/collection.json")}, // collection file, should not be deleted
				{Key: aws.String(viewStateS3Path + "spectrum.json")},
			},
		},
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateS3Path + "Workspaces/workspace.json")},            // workspace file, should not be deleted
				{Key: aws.String(viewStateS3Path + "WorkspaceCollections/collection.json")}, // collection file, should not be deleted
			},
		},
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		// Test 4
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "not-a-widget.json"),
		},
		// Test 5
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum.json"),
		},
	}
	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
		{},
	}

	// Querying blessed quant (because quant it has is empty)
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/TheDataSetID/Quantifications/blessed-quant.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // There isn't a blessed quant!
	}

	// Various bits should return in the response...
	req, _ := http.NewRequest("GET", "/view-state/TheDataSetID?reset=true", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	//
	// 200
	// {
	//     "analysisLayout": {
	//         "topWidgetSelectors": [],
	//         "bottomWidgetSelectors": []
	//     },
	//     "spectrum": {
	//         "panX": 0,
	//         "panY": 0,
	//         "zoomX": 1,
	//         "zoomY": 1,
	//         "spectrumLines": [],
	//         "logScale": true,
	//         "xrflines": [],
	//         "showXAsEnergy": false,
	//         "energyCalibration": []
	//     },
	//     "contextImages": {},
	//     "histograms": {},
	//     "chordDiagrams": {},
	//     "ternaryPlots": {},
	//     "binaryPlots": {},
	//     "tables": {},
	//     "roiQuantTables": {},
	//     "variograms": {},
	//     "spectrums": {},
	//     "rgbuPlots": {},
	//     "singleAxisRGBU": {},
	//     "rgbuImages": {},
	//     "parallelograms": {},
	//     "annotations": {
	//         "savedAnnotations": []
	//     },
	//     "rois": {
	//         "roiColours": {},
	//         "roiShapes": {}
	//     },
	//     "quantification": {
	//         "appliedQuantID": ""
	//     },
	//     "selection": {
	//         "roiID": "",
	//         "roiName": "",
	//         "locIdxs": []
	//     }
	// }
}

func Example_viewStateHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/view-state/TheDataSetID/widget", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

func Example_viewStateHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// POST not implemented! Should return 405
	req, _ := http.NewRequest("POST", "/view-state/TheDataSetID", bytes.NewReader([]byte(`{
	"quantification": "The Name",
	"roi": "12"
}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

// NOTE: This is a special test, because it also has tracking turned on, and has a logger middleware installed
// This is so we test that the middleware correctly identifies these PUT msgs as something that needs to be
// saved and does so.
func Test_viewStateHandler_Put_spectrum_topright_AND_middleware_activity_logging(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		//mt.AddMockResponses()

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		mockS3.ExpPutObjectInput = []s3.PutObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum-top1.json"), Body: bytes.NewReader([]byte(`{
    "panX": 12,
    "panY": 0,
    "zoomX": 1,
    "zoomY": 0,
    "spectrumLines": [
        {
            "roiID": "dataset",
            "lineExpressions": [
                "bulk(A)",
                "bulk(B)"
            ]
        },
        {
            "roiID": "selection",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        },
        {
            "roiID": "roi-123",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        }
    ],
    "logScale": true,
    "xrflines": [
        {
            "line_info": {
                "Z": 12,
                "K": true,
                "L": true,
                "M": true,
                "Esc": true
            },
            "visible": true
        }
    ],
    "showXAsEnergy": true,
    "energyCalibration": [
        {
            "detector": "B",
            "eVStart": 12.5,
            "eVPerChannel": 17.8
        }
    ]
}`)),
			},
			// The middleware should write out a copy of this message to the user activity tracking spot
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("Activity/2022-11-11/id-1234.json"), Body: bytes.NewReader([]byte(`{
    "Instance": "",
    "Time": "2022-11-11T04:56:19Z",
    "Component": "/view-state/TheDataSetID/spectrum-top1",
    "Message": "{\n    \"panX\": 12,\n    \"zoomX\": 1,\n    \"energyCalibration\": [\n        {\n            \"detector\": \"B\",\n            \"eVStart\": 12.5,\n            \"eVPerChannel\": 17.8\n        }\n    ],\n    \"logScale\": true,\n    \"spectrumLines\": [\n        {\n            \"roiID\": \"dataset\",\n            \"lineExpressions\": [\n                \"bulk(A)\",\n                \"bulk(B)\"\n            ]\n        },\n        {\n            \"roiID\": \"selection\",\n            \"lineExpressions\": [\n                \"sum(bulk(A), bulk(B))\"\n            ]\n        },\n        {\n            \"roiID\": \"roi-123\",\n            \"lineExpressions\": [\n                \"sum(bulk(A), bulk(B))\"\n            ]\n        }\n    ],\n    \"xrflines\": [\n        {\n            \"visible\": true,\n            \"line_info\": {\n                \"Z\": 12,\n                \"K\": true,\n                \"L\": true,\n                \"M\": true,\n                \"Esc\": true\n            }\n        }\n    ],\n    \"showXAsEnergy\": true\n}",
    "Response": "",
    "Version": "",
    "Params": {
        "method": "PUT"
    },
    "Environment": "unit-test",
    "User": "myuserid"
}`)),
			},
		}

		mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
			{},
			{},
		}

		// Set up extra bits for middleware testing
		var idGen MockIDGenerator
		idGen.ids = []string{"id-1234"}

		svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668142579},
		}

		setTestAuth0Config(&svcs)

		notifications, err := notifications.MakeNotificationStack(mt.Client, "unit_test", nil, &logger.StdOutLoggerForTest{}, []string{})
		if err != nil {
			t.Error(err)
		}

		svcs.Notifications = notifications

		// Add requestor as a tracked user, so we should see activity saved
		svcs.Notifications.SetTrack("myuserid", true)

		apiRouter := MakeRouter(svcs)

		mockvalidator := api.MockJWTValidator{}
		logware := LoggerMiddleware{&svcs, &mockvalidator}

		apiRouter.Router.Use(logware.Middleware)
		const putItem = `{
    "panX": 12,
    "zoomX": 1,
    "energyCalibration": [
        {
            "detector": "B",
            "eVStart": 12.5,
            "eVPerChannel": 17.8
        }
    ],
    "logScale": true,
    "spectrumLines": [
        {
            "roiID": "dataset",
            "lineExpressions": [
                "bulk(A)",
                "bulk(B)"
            ]
        },
        {
            "roiID": "selection",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        },
        {
            "roiID": "roi-123",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        }
    ],
    "xrflines": [
        {
            "visible": true,
            "line_info": {
                "Z": 12,
                "K": true,
                "L": true,
                "M": true,
                "Esc": true
            }
        }
    ],
    "showXAsEnergy": true
}`

		const routePath = "/view-state/TheDataSetID/"

		req, _ := http.NewRequest("PUT", routePath+"spectrum-top1", bytes.NewReader([]byte(putItem)))
		//req.Header.Set("content-type", "application/json")
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, "")

		// Wait a bit for any threads to finish (part of the middleware test)
		time.Sleep(2 * time.Second)
	})
}

func Example_viewStateHandler_Put_spectrum_oldway_FAIL() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "panX": 12,
    "zoomX": 1,
    "energyCalibration": [
        {
            "detector": "B",
            "eVStart": 12.5,
            "eVPerChannel": 17.8
        }
    ],
    "logScale": true,
    "spectrumLines": [
        {
            "roiID": "dataset",
            "lineExpressions": [
                "bulk(A)",
                "bulk(B)"
            ]
        },
        {
            "roiID": "selection",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        },
        {
            "roiID": "roi-123",
            "lineExpressions": [
                "sum(bulk(A), bulk(B))"
            ]
        }
    ],
    "xrflines": [
        {
            "visible": true,
            "line_info": {
                "Z": 12,
                "K": true,
                "L": true,
                "M": true,
                "Esc": true
            }
        }
    ],
    "showXAsEnergy": true
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"spectrum", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Unknown widget: spectrum
}

func Example_viewStateHandler_Put_contextImage() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-analysis.json"), Body: bytes.NewReader([]byte(`{
    "panX": 12,
    "panY": 0,
    "zoomX": 1,
    "zoomY": 0,
    "showPoints": true,
    "showPointBBox": false,
    "pointColourScheme": "BW",
    "pointBBoxColourScheme": "PURPLE_CYAN",
    "contextImage": "context123.png",
    "contextImageSmoothing": "nearest",
    "mapLayers": [
        {
            "expressionID": "Ca",
            "opacity": 0.1,
            "visible": false,
            "displayValueRangeMin": 12,
            "displayValueRangeMax": 48.8,
            "displayValueShading": "SHADE_VIRIDIS"
        },
        {
            "expressionID": "Ti",
            "opacity": 0.4,
            "visible": true,
            "displayValueRangeMin": 24,
            "displayValueRangeMax": 25.5,
            "displayValueShading": "SHADE_PURPLE"
        }
    ],
    "roiLayers": [
        {
            "roiID": "roi111",
            "opacity": 0.8,
            "visible": true
        }
    ],
    "elementRelativeShading": false,
    "brightness": 1.3,
    "rgbuChannels": "GRU",
    "unselectedOpacity": 0.2,
    "unselectedGrayscale": true,
    "colourRatioMin": 0,
    "colourRatioMax": 1.3,
    "removeTopSpecularArtifacts": false,
    "removeBottomSpecularArtifacts": false
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "panX": 12,
    "zoomX": 1,
    "showPoints": true,
    "showPointBBox": false,
    "pointColourScheme": "BW",
    "pointBBoxColourScheme": "PURPLE_CYAN",
    "contextImage": "context123.png",
    "contextImageSmoothing": "nearest",
    "brightness": 1.3,
    "rgbuChannels": "GRU",
    "unselectedOpacity": 0.2,
    "unselectedGrayscale": true,
    "colourRatioMax": 1.3,
    "mapLayers": [
        {
            "expressionID": "Ca",
            "opacity": 0.1,
            "visible": false,
            "displayValueRangeMin": 12,
            "displayValueRangeMax": 48.8,
            "displayValueShading": "SHADE_VIRIDIS"
        },
        {
            "expressionID": "Ti",
            "opacity": 0.4,
            "visible": true,
            "displayValueRangeMin": 24,
            "displayValueRangeMax": 25.5,
            "displayValueShading": "SHADE_PURPLE"
        }
    ],
    "roiLayers": [
        {
            "roiID": "roi111",
            "opacity": 0.8,
            "visible": true
        }
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"contextImage-analysis", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_quantification() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "quantification.json"), Body: bytes.NewReader([]byte(`{
    "appliedQuantID": "54321"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"appliedQuantID": "54321"
}`

	const routePath = "/view-state/TheDataSetID/"

	// Spectrum
	req, _ := http.NewRequest("PUT", routePath+"quantification", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_histogram() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "histogram-top-left.json"), Body: bytes.NewReader([]byte(`{
    "showStdDeviation": true,
    "logScale": true,
    "expressionIDs": [
        "Fe",
        "Ca"
    ],
    "visibleROIs": [
        "roi123",
        "roi456",
        "roi789"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "showStdDeviation": true,
    "logScale": true,
    "expressionIDs": [
        "Fe",
        "Ca"
    ],
    "visibleROIs": [
        "roi123",
        "roi456",
        "roi789"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"histogram-top-left", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_selection() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "selection.json"), Body: bytes.NewReader([]byte(`{
    "roiID": "3333",
    "roiName": "Dark patch",
    "locIdxs": [
        999,
        888,
        777
    ],
    "pixelSelectionImageName": "file.tif",
    "pixelIdxs": [
        333
    ],
    "cropPixelIdxs": [
        333,
        334
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "roiID": "3333",
    "roiName": "Dark patch",
    "locIdxs": [
        999,
        888,
        777
    ],
    "pixelSelectionImageName": "file.tif",
    "pixelIdxs": [
        333
    ],
    "cropPixelIdxs": [
        333,
		334
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"selection", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_chord() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "chord-111.json"), Body: bytes.NewReader([]byte(`{
    "showForSelection": false,
    "expressionIDs": [
        "abc123"
    ],
    "displayROI": "roi999",
    "threshold": 0.8,
    "drawMode": "POSITIVE"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "showForSelection": false,
    "expressionIDs": [
        "abc123"
	],
	"displayROI": "roi999",
    "threshold": 0.8,
    "drawMode": "POSITIVE"
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"chord-111", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_binary() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "binary-bottom12.json"), Body: bytes.NewReader([]byte(`{
    "showMmol": false,
    "expressionIDs": [
        "Fe",
        "Ca"
    ],
    "visibleROIs": [
        "roi123",
        "roi456"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "showMmol": false,
    "visibleROIs": [
        "roi123",
        "roi456"
    ],
    "expressionIDs": [
        "Fe",
        "Ca"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"binary-bottom12", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_ternary() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "ternary-5.json"), Body: bytes.NewReader([]byte(`{
    "showMmol": false,
    "expressionIDs": [
        "Fe",
        "Ca",
        "Sr"
    ],
    "visibleROIs": [
        "roi123",
        "roi456"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "showMmol": false,
    "visibleROIs": [
        "roi123",
        "roi456"
    ],
    "expressionIDs": [
        "Fe",
        "Ca",
        "Sr"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"ternary-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_table() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "table-5.json"), Body: bytes.NewReader([]byte(`{
    "showPureElements": false,
    "order": "something",
    "visibleROIs": [
        "roi123",
        "roi456"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "showPureElements": false,
    "order": "something",
    "visibleROIs": [
        "roi123",
        "roi456"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"table-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_roiQuantTable() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roiQuantTable-5.json"), Body: bytes.NewReader([]byte(`{
    "roi": "something",
    "quantIDs": [
        "quant1",
        "Q2"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "roi": "something",
    "quantIDs": [
        "quant1",
        "Q2"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"roiQuantTable-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_parallelogram() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "parallelogram-5.json"), Body: bytes.NewReader([]byte(`{
    "colourChannels": [
        "R",
        "G",
        "B"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "colourChannels": [
        "R",
        "G",
        "B"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"parallelogram-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_rgbuImages() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuImages-5.json"), Body: bytes.NewReader([]byte(`{
    "logColour": false,
    "brightness": 1.2
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "logColour": false,
    "brightness": 1.2
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"rgbuImages-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_rgbuPlots() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuPlot-5.json"), Body: bytes.NewReader([]byte(`{
    "minerals": [
        "Plagioclase",
        "Olivine"
    ],
    "yChannelA": "B",
    "yChannelB": "U",
    "xChannelA": "R",
    "xChannelB": "G",
    "drawMonochrome": true
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "xChannelA": "R",
	"xChannelB": "G",
	"yChannelA": "B",
	"yChannelB": "U",
	"drawMonochrome": true,
    "minerals": [
        "Plagioclase",
        "Olivine"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"rgbuPlot-5", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_roi() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roi.json"), Body: bytes.NewReader([]byte(`{
    "roiColours": {
        "roi22": "rgba(128,0,255,0.5)",
        "roi33": "rgba(255,255,0,1)"
    },
    "roiShapes": {}
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"roiColours": {
		"roi33": "rgba(255,255,0,1)",
		"roi22": "rgba(128,0,255,0.5)"
	},
    "roiShapes": {}
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"roi", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Put_analysisLayout() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "analysisLayout.json"), Body: bytes.NewReader([]byte(`{
    "topWidgetSelectors": [
        "spectrum"
    ],
    "bottomWidgetSelectors": [
        "chord",
        "binary"
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
    "topWidgetSelectors": [
        "spectrum"
    ],
    "bottomWidgetSelectors": [
        "chord",
        "binary"
    ]
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"analysisLayout", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_viewStateHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// DELETE not implemented! Should return 405
	req, _ := http.NewRequest("DELETE", "/view-state/TheDataSetID/widget", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

// Saving an entire view state. This clears view state files in the S3 directory we're writing to, and writes new ones
// based on the view state being passed in
func Example_viewStateHandler_Put_all() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Expecting a listing of view state dir
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String(viewStateS3Path),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String(viewStateS3Path + "not-a-widget.json")},                    // Not a recognised file name
				{Key: aws.String(viewStateS3Path + "Workspaces/workspace.json")},            // workspace file, should not be deleted
				{Key: aws.String(viewStateS3Path + "WorkspaceCollections/collection.json")}, // collection file, should not be deleted
				{Key: aws.String(viewStateS3Path + "spectrum-top1.json")},
			},
		},
	}

	// Expecting to delete only view state files (not workspace/collection)
	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		// Test 4
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "not-a-widget.json"),
		},
		// Test 5
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum-top1.json"),
		},
	}
	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
		{},
	}

	// Expecting a PUT for layout, selection, quant, ROI and each widget
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "annotations.json"), Body: bytes.NewReader([]byte(`{
    "savedAnnotations": []
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roi.json"), Body: bytes.NewReader([]byte(`{
    "roiColours": {
        "roi22": "rgba(128,0,255,0.5)",
        "roi33": "rgba(255,255,0,1)"
    },
    "roiShapes": {}
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "quantification.json"), Body: bytes.NewReader([]byte(`{
    "appliedQuantID": "9qntb8w2joq4elti"
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "selection.json"), Body: bytes.NewReader([]byte(`{
    "roiID": "",
    "roiName": "",
    "locIdxs": [
        345,
        347,
        348,
        1273
    ]
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "analysisLayout.json"), Body: bytes.NewReader([]byte(`{
    "topWidgetSelectors": [
        "context-image",
        "spectrum-widget"
    ],
    "bottomWidgetSelectors": [
        "table-widget",
        "binary-plot-widget",
        "rgbu-plot-widget",
        "ternary-plot-widget"
    ]
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-analysis.json"), Body: bytes.NewReader([]byte(`{
    "panX": -636.63446,
    "panY": -674.23505,
    "zoomX": 2.6251905,
    "zoomY": 2.6251905,
    "showPoints": true,
    "showPointBBox": true,
    "pointColourScheme": "PURPLE_CYAN",
    "pointBBoxColourScheme": "PURPLE_CYAN",
    "contextImage": "PCCR0257_0689789827_000MSA_N008000008906394300060LUD01.tif",
    "contextImageSmoothing": "linear",
    "mapLayers": [],
    "roiLayers": [
        {
            "roiID": "AllPoints",
            "opacity": 1,
            "visible": false
        },
        {
            "roiID": "SelectedPoints",
            "opacity": 1,
            "visible": false
        }
    ],
    "elementRelativeShading": true,
    "brightness": 1,
    "rgbuChannels": "R/G",
    "unselectedOpacity": 0.3,
    "unselectedGrayscale": false,
    "colourRatioMin": 0.5,
    "colourRatioMax": 2.25,
    "removeTopSpecularArtifacts": false,
    "removeBottomSpecularArtifacts": false
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "contextImage-map.json"), Body: bytes.NewReader([]byte(`{
    "panX": -116.896935,
    "panY": -145.20177,
    "zoomX": 1.0904286,
    "zoomY": 1.0904286,
    "showPoints": true,
    "showPointBBox": true,
    "pointColourScheme": "PURPLE_CYAN",
    "pointBBoxColourScheme": "PURPLE_CYAN",
    "contextImage": "",
    "contextImageSmoothing": "linear",
    "mapLayers": [],
    "roiLayers": [
        {
            "roiID": "AllPoints",
            "opacity": 1,
            "visible": false
        },
        {
            "roiID": "SelectedPoints",
            "opacity": 1,
            "visible": false
        }
    ],
    "elementRelativeShading": true,
    "brightness": 1,
    "rgbuChannels": "RGB",
    "unselectedOpacity": 0.4,
    "unselectedGrayscale": false,
    "colourRatioMin": 0,
    "colourRatioMax": 0,
    "removeTopSpecularArtifacts": false,
    "removeBottomSpecularArtifacts": false
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "ternary-underspectrum2.json"), Body: bytes.NewReader([]byte(`{
    "showMmol": false,
    "expressionIDs": [
        "vge9tz6fkbi2ha1p",
        "shared-j1g1sx285s6yqjih",
        "r4zd5s2tfgr8rahy"
    ],
    "visibleROIs": [
        "AllPoints",
        "SelectedPoints"
    ]
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuPlot-underspectrum1.json"), Body: bytes.NewReader([]byte(`{
    "minerals": [],
    "yChannelA": "B",
    "yChannelB": "U",
    "xChannelA": "R",
    "xChannelB": "B",
    "drawMonochrome": false
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum-top1.json"), Body: bytes.NewReader([]byte(`{
    "panX": -53.19157,
    "panY": -37.737877,
    "zoomX": 3.5776386,
    "zoomY": 1.3382256,
    "spectrumLines": [
        {
            "roiID": "AllPoints",
            "lineExpressions": [
                "bulk(A)",
                "bulk(B)"
            ]
        },
        {
            "roiID": "SelectedPoints",
            "lineExpressions": [
                "bulk(A)",
                "bulk(B)"
            ]
        }
    ],
    "logScale": true,
    "xrflines": [],
    "showXAsEnergy": true,
    "energyCalibration": [
        {
            "detector": "A",
            "eVStart": -20.759016,
            "eVPerChannel": 7.8629937
        },
        {
            "detector": "B",
            "eVStart": -20.759016,
            "eVPerChannel": 7.8629937
        }
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
		{},
		{},
		{},
		{},
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const wholeState = `{
	"analysisLayout": {
		"topWidgetSelectors": ["context-image", "spectrum-widget"],
		"bottomWidgetSelectors": ["table-widget", "binary-plot-widget", "rgbu-plot-widget", "ternary-plot-widget"]
	},
	"contextImages": {
		"analysis": {
			"panX": -636.63446,
			"panY": -674.23505,
			"zoomX": 2.6251905,
			"zoomY": 2.6251905,
			"showPoints": true,
			"showPointBBox": true,
			"pointColourScheme": "PURPLE_CYAN",
			"pointBBoxColourScheme": "PURPLE_CYAN",
			"contextImage": "PCCR0257_0689789827_000MSA_N008000008906394300060LUD01.tif",
			"contextImageSmoothing": "linear",
			"mapLayers": [],
			"roiLayers": [{
				"roiID": "AllPoints",
				"opacity": 1,
				"visible": false
			}, {
				"roiID": "SelectedPoints",
				"opacity": 1,
				"visible": false
			}],
			"elementRelativeShading": true,
			"brightness": 1,
			"rgbuChannels": "R/G",
			"unselectedOpacity": 0.3,
			"unselectedGrayscale": false,
			"colourRatioMin": 0.5,
			"colourRatioMax": 2.25,
			"removeTopSpecularArtifacts": false,
			"removeBottomSpecularArtifacts": false
		},
		"map": {
			"panX": -116.896935,
			"panY": -145.20177,
			"zoomX": 1.0904286,
			"zoomY": 1.0904286,
			"showPoints": true,
			"showPointBBox": true,
			"pointColourScheme": "PURPLE_CYAN",
			"pointBBoxColourScheme": "PURPLE_CYAN",
			"contextImage": "",
			"contextImageSmoothing": "linear",
			"mapLayers": [],
			"roiLayers": [{
				"roiID": "AllPoints",
				"opacity": 1,
				"visible": false
			}, {
				"roiID": "SelectedPoints",
				"opacity": 1,
				"visible": false
			}],
			"elementRelativeShading": true,
			"brightness": 1,
			"rgbuChannels": "RGB",
			"unselectedOpacity": 0.4,
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
		"undercontext": {
			"showMmol": false,
			"expressionIDs": ["expr-elem-K2O-%(Combined)", "expr-elem-Na2O-%(Combined)", "expr-elem-MgO-%(Combined)"],
			"visibleROIs": ["AllPoints", "9s5vkwjxl6539jbp", "tsiaam7uvs00yjom", "und4hnr30l61ha3u", "newa1c3apifnygtm", "y0o44g8n4z3ts40x", "SelectedPoints"]
		},
		"underspectrum1": {
			"showMmol": false,
			"expressionIDs": ["expr-elem-Na2O-%(Combined)", "shared-uds1s1t27qf97b03", "expr-elem-MgO-%(Combined)"],
			"visibleROIs": ["AllPoints", "SelectedPoints"]
		},
		"underspectrum2": {
			"showMmol": false,
			"expressionIDs": ["vge9tz6fkbi2ha1p", "shared-j1g1sx285s6yqjih", "r4zd5s2tfgr8rahy"],
			"visibleROIs": ["AllPoints", "SelectedPoints"]
		}
	},
	"binaryPlots": {
		"undercontext": {
			"showMmol": false,
			"expressionIDs": ["", ""],
			"visibleROIs": ["AllPoints", "SelectedPoints"]
		},
		"underspectrum1": {
			"showMmol": false,
			"expressionIDs": ["expr-elem-SiO2-%(Combined)", "expr-elem-Al2O3-%(Combined)"],
			"visibleROIs": ["AllPoints", "SelectedPoints"]
		}
	},
	"tables": {
		"underspectrum0": {
			"showPureElements": false,
			"order": "atomic-number",
			"visibleROIs": ["AllPoints", "SelectedPoints", "jvi1p1awm77fsywc", "6mbhyd8nbyj4um4p", "1mk9xra5qejh3tvk"]
		}
	},
	"roiQuantTables": {},
	"variograms": {
		"undercontext": {
			"expressionIDs": ["expr-elem-K2O-%"],
			"visibleROIs": ["AllPoints", "SelectedPoints"],
			"varioModel": "exponential",
			"maxDistance": 6.5188847,
			"binCount": 1668,
			"drawModeVector": false
		}
	},
	"spectrums": {
		"top0": {
			"panX": -137.13159,
			"panY": 0,
			"zoomX": 1.6592865,
			"zoomY": 1,
			"spectrumLines": [{
				"roiID": "AllPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}, {
				"roiID": "SelectedPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -18.5,
				"eVPerChannel": 7.862
			}, {
				"detector": "B",
				"eVStart": -22.4,
				"eVPerChannel": 7.881
			}]
		},
		"top1": {
			"panX": -53.19157,
			"panY": -37.737877,
			"zoomX": 3.5776386,
			"zoomY": 1.3382256,
			"spectrumLines": [{
				"roiID": "AllPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}, {
				"roiID": "SelectedPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -20.759016,
				"eVPerChannel": 7.8629937
			}, {
				"detector": "B",
				"eVStart": -20.759016,
				"eVPerChannel": 7.8629937
			}]
		},
		"undercontext": {
			"panX": 0,
			"panY": 0,
			"zoomX": 1,
			"zoomY": 1,
			"spectrumLines": [{
				"roiID": "AllPoints",
				"lineExpressions": ["bulk(A)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -18.5,
				"eVPerChannel": 7.862
			}, {
				"detector": "B",
				"eVStart": -22.4,
				"eVPerChannel": 7.881
			}]
		},
		"underspectrum0": {
			"panX": 0,
			"panY": 0,
			"zoomX": 1,
			"zoomY": 1,
			"spectrumLines": [{
				"roiID": "AllPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}, {
				"roiID": "SelectedPoints",
				"lineExpressions": ["bulk(A)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -18.5,
				"eVPerChannel": 7.862
			}, {
				"detector": "B",
				"eVStart": -22.4,
				"eVPerChannel": 7.881
			}]
		},
		"underspectrum1": {
			"panX": 0,
			"panY": 0,
			"zoomX": 1,
			"zoomY": 1,
			"spectrumLines": [{
				"roiID": "AllPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}, {
				"roiID": "SelectedPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -18.5,
				"eVPerChannel": 7.862
			}, {
				"detector": "B",
				"eVStart": -22.4,
				"eVPerChannel": 7.881
			}]
		},
		"underspectrum2": {
			"panX": 0,
			"panY": 0,
			"zoomX": 1,
			"zoomY": 1,
			"spectrumLines": [{
				"roiID": "SelectedPoints",
				"lineExpressions": ["bulk(A)", "bulk(B)"]
			}],
			"logScale": true,
			"xrflines": [],
			"showXAsEnergy": true,
			"energyCalibration": [{
				"detector": "A",
				"eVStart": -18.5,
				"eVPerChannel": 7.862
			}, {
				"detector": "B",
				"eVStart": -22.4,
				"eVPerChannel": 7.881
			}]
		}
	},
	"rgbuPlots": {
		"underspectrum0": {
			"minerals": ["plag", "sanidine", "microline", "aug", "opx", "Fo89", "Fo11", "Chalcedor", "calsite", "gypsum", "dolomite", "FeS2", "FeS", "Fe3O4"],
			"yChannelA": "G",
			"yChannelB": "R",
			"xChannelA": "B",
			"xChannelB": "R",
			"drawMonochrome": false
		},
		"underspectrum1": {
			"minerals": [],
			"yChannelA": "B",
			"yChannelB": "U",
			"xChannelA": "R",
			"xChannelB": "B",
			"drawMonochrome": false
		},
		"underspectrum2": {
			"minerals": [],
			"yChannelA": "U",
			"yChannelB": "R",
			"xChannelA": "U",
			"xChannelB": "B",
			"drawMonochrome": false
		}
	},
	"singleAxisRGBU": {},
	"rgbuImages": {
		"top1": {
			"logColour": false,
			"brightness": 1
		}
	},
	"parallelograms": {},
	"annotations": {
		"savedAnnotations": []
	},
	"rois": {
		"roiColours": {
			"roi22": "rgba(128,0,255,0.5)",
			"roi33": "rgba(255,255,0,1)"
		},
		"roiShapes": {}
	},
	"quantification": {
		"appliedQuantID": "9qntb8w2joq4elti"
	},
	"selection": {
		"roiID": "",
		"roiName": "",
		"locIdxs": [345, 347, 348, 1273]
	}
}`

	const routePath = "/view-state/TheDataSetID/"

	req, _ := http.NewRequest("PUT", routePath+"all", bytes.NewReader([]byte(wholeState)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}
