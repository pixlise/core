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
	"net/http/httptest"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/api/config"
	"gitlab.com/pixlise/pixlise-go-api/api/esutil"
	"gitlab.com/pixlise/pixlise-go-api/core/api"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
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
				{Key: aws.String(viewStateS3Path + "histogram-1.json")},
				{Key: aws.String(viewStateS3Path + "chord-0.json")},
				{Key: aws.String(viewStateS3Path + "chord-1.json")},
				{Key: aws.String(viewStateS3Path + "table-xyz.json")},
				{Key: aws.String(viewStateS3Path + "binary-1.json")},
				{Key: aws.String(viewStateS3Path + "ternary-bottom-row-1.json")},
				{Key: aws.String(viewStateS3Path + "variogram-abc123.json")},
				{Key: aws.String(viewStateS3Path + "rgbuImages-33.json")},
				{Key: aws.String(viewStateS3Path + "rgbuPlot-44.json")},
				{Key: aws.String(viewStateS3Path + "parallelogram-55.json")},
				{Key: aws.String(viewStateS3Path + "roiQuantTable-ttt.json")},
				{Key: aws.String(viewStateS3Path + "spectrum-0.json")}, // the "new style" version that comes with a position id
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "histogram-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "chord-0.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "chord-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "table-xyz.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "binary-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "ternary-bottom-row-1.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "variogram-abc123.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuImages-33.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "rgbuPlot-44.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "parallelogram-55.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "roiQuantTable-ttt.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(viewStateS3Path + "spectrum-0.json"),
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
	}
}`))),
		},
		nil, // quant histogram
		nil, // chord-0
		nil, // chord-1
		{ // table
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
	//         "topWidgetSelectors": [],
	//         "bottomWidgetSelectors": []
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
	//         "engineering": {
	//             "panX": 0,
	//             "panY": 12,
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
	//     "histograms": {
	//         "1": {
	//             "showStdDeviation": false,
	//             "logScale": false,
	//             "expressionIDs": [],
	//             "visibleROIs": []
	//         }
	//     },
	//     "chordDiagrams": {
	//         "0": {
	//             "showForSelection": false,
	//             "expressionIDs": [],
	//             "displayROI": "",
	//             "threshold": 0,
	//             "drawMode": "BOTH"
	//         },
	//         "1": {
	//             "showForSelection": false,
	//             "expressionIDs": [],
	//             "displayROI": "",
	//             "threshold": 0,
	//             "drawMode": "BOTH"
	//         }
	//     },
	//     "ternaryPlots": {
	//         "bottom-row-1": {
	//             "showMmol": false,
	//             "expressionIDs": [],
	//             "visibleROIs": []
	//         }
	//     },
	//     "binaryPlots": {
	//         "1": {
	//             "showMmol": false,
	//             "expressionIDs": [],
	//             "visibleROIs": []
	//         }
	//     },
	//     "tables": {
	//         "xyz": {
	//             "showPureElements": true,
	//             "order": "atomic-number",
	//             "visibleROIs": []
	//         }
	//     },
	//     "roiQuantTables": {
	//         "ttt": {
	//             "roi": "the-roi",
	//             "quantIDs": [
	//                 "quant1",
	//                 "quant2"
	//             ]
	//         }
	//     },
	//     "variograms": {
	//         "abc123": {
	//             "expressionIDs": [],
	//             "visibleROIs": [],
	//             "varioModel": "exponential",
	//             "maxDistance": 0,
	//             "binCount": 0,
	//             "drawModeVector": false
	//         }
	//     },
	//     "spectrums": {
	//         "0": {
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
	//         "44": {
	//             "minerals": [],
	//             "yChannelA": "B",
	//             "yChannelB": "",
	//             "xChannelA": "",
	//             "xChannelB": "",
	//             "drawMonochrome": false
	//         }
	//     },
	//     "singleAxisRGBU": {},
	//     "rgbuImages": {
	//         "33": {
	//             "logColour": false,
	//             "brightness": 1.2
	//         }
	//     },
	//     "parallelograms": {
	//         "55": {
	//             "colourChannels": [
	//                 "R",
	//                 "G"
	//             ]
	//         }
	//     },
	//     "rois": {
	//         "roiColours": {
	//             "roi22": "rgba(128,0,255,0.5)",
	//             "roi99": "rgba(255,255,0,1)"
	//         }
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

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
	//     "rois": {
	//         "roiColours": {}
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

func Example_viewStateHandler_Put_spectrum_topright() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/myuserid.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"userid":"myuserid","notifications":{"topics":[],"hints":["point-select-alt","point-select-z-for-zoom","point-select-shift-for-pan","lasso-z-for-zoom","lasso-shift-for-pan","dwell-exists-test-fm-5x5-full","dwell-exists-069927431"],"uinotifications":[]},"userconfig":{"name":"peternemere","email":"peternemere@gmail.com","cell":"","data_collection":"1.0"}}`)))},
	}

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()
	//"Component":"http://example.com/foo","Message":"{\"alive\": true}","Version":"","Params":{"method":"GET"},"Environment":"unit-test","User":"myuserid"}
	var ExpIndexObject = []string{
		`{"Instance":"","Time":"0000-00-00T00:00:00-00:00","Component":"/view-state/TheDataSetID/spectrum-top1","Message":"{\n    \"panX\": 12,\n    \"zoomX\": 1,\n    \"energyCalibration\": [\n        {\n            \"detector\": \"B\",\n            \"eVStart\": 12.5,\n            \"eVPerChannel\": 17.8\n        }\n    ],\n    \"logScale\": true,\n    \"spectrumLines\": [\n        {\n            \"roiID\": \"dataset\",\n            \"lineExpressions\": [\n                \"bulk(A)\",\n                \"bulk(B)\"\n            ]\n        },\n        {\n            \"roiID\": \"selection\",\n            \"lineExpressions\": [\n                \"sum(bulk(A), bulk(B))\"\n            ]\n        },\n        {\n            \"roiID\": \"roi-123\",\n            \"lineExpressions\": [\n                \"sum(bulk(A), bulk(B))\"\n            ]\n        }\n    ],\n    \"xrflines\": [\n        {\n            \"visible\": true,\n            \"line_info\": {\n                \"Z\": 12,\n                \"K\": true,\n                \"L\": true,\n                \"M\": true,\n                \"Esc\": true\n            }\n        }\n    ],\n    \"showXAsEnergy\": true\n}","Response":"","Version":"","Params":{"method":"PUT"},"Environment":"unit-test","User":"myuserid"}`,
	}
	var ExpRespObject = []string{
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
	}

	var adjtime = "0000-00-00T00:00:00-00:00"
	d := esutil.DummyElasticClient{}
	foo, err := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, &adjtime)
	//defer d.FinishTest()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	apiConfig := config.APIConfig{EnvironmentName: "Test"}
	connection, err := esutil.Connect(foo, apiConfig)
	// NOTE: PUT expected JSON needs to have spaces not tabs
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
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, &connection, nil)
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
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// &map[]200
}

func Example_viewStateHandler_Put_spectrum_oldway_FAIL() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"roiColours": {
		"roi33": "rgba(255,255,0,1)",
		"roi22": "rgba(128,0,255,0.5)"
	}
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// DELETE not implemented! Should return 405
	req, _ := http.NewRequest("DELETE", "/view-state/TheDataSetID/widget", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}
