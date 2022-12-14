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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

const roiS3Path = "UserContent/600f2a0806b6c70071d3d174/TheDataSetID/ROI.json"
const roiSharedS3Path = "UserContent/shared/TheDataSetID/ROI.json"
const roi2XItems = `{
    "331": {
        "name": "Dark patch 2",
        "description": "The second dark patch",
        "locationIndexes": [4, 55, 394],
        "creator": { "name": "Peter", "user_id": "u123" },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000,
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        }
    },
    "772": {
        "name": "White spot",
        "locationIndexes": [14, 5, 94],
        "creator": { "name": "Tom", "user_id": "u124" },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001,
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        }
    }
}`

func Test_roiHandler_List(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// u77 - not found
			mtest.CreateCursorResponse(
				1,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.NextBatch,
			),
			// User u85
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "u85"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Tom Barber"},
						{"Email", "tom@spicule.co.uk"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()
		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/NewDataSet/ROI.json"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/NewDataSet/ROI.json"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/AnotherDataSetID/ROI.json"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/AnotherDataSetID/ROI.json"),
			},
		}
		mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
			nil, // No file in S3
			nil, // No file in S3
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
					"331": {
						"name": "dark patch",
						"locationIndexes": [4, 55, 394],
						"shared": false,
						"creator": { "name": "Peter", "user_id": "u77", "email": "" },
						"imageName": "dtu_context_rgbu.tif",
						"mistROIItem": {
							"species": "",
							"mineralGroupID": "",
							"ID_Depth": 0,
							"ClassificationTrail": "",
							"formula": ""
						},
						"create_unix_time_sec": 1668100000,
						"mod_unix_time_sec": 1668100000
					}
				}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
					"007": {
						"description": "james bonds shared ROI",
						"name": "james bond",
						"locationIndexes": [99],
						"shared": false,
						"creator": { "name": "Tom", "user_id": "u85", "email": ""},
						"create_unix_time_sec": 1668100003,
						"mod_unix_time_sec": 1668100003,
						"mistROIItem": {
							"species": "",
							"mineralGroupID": "",
							"ID_Depth": 0,
							"ClassificationTrail": "",
							"formula": ""
						}
					}
				}`))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/roi/NewDataSet", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/roi/TheDataSetID", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/roi/AnotherDataSetID", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "331": {
        "name": "dark patch",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "description": "",
        "imageName": "dtu_context_rgbu.tif",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u77",
            "email": ""
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "shared-007": {
        "name": "james bond",
        "locationIndexes": [
            99
        ],
        "description": "james bonds shared ROI",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": true,
        "creator": {
            "name": "Tom Barber",
            "user_id": "u85",
            "email": "tom@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100003,
        "mod_unix_time_sec": 1668100003
    }
}
`)
	})
}

func Example_roiHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "id999": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9
        ],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100007,
        "mod_unix_time_sec": 1668100007
    }
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path), Body: bytes.NewReader([]byte(`{
    "id3": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9,
            199
        ],
        "description": "Posted item!",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142579,
        "mod_unix_time_sec": 1668142579
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path), Body: bytes.NewReader([]byte(`{
    "id4": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9,
            199
        ],
        "description": "Posted item!",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142580,
        "mod_unix_time_sec": 1668142580
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path), Body: bytes.NewReader([]byte(`{
    "331": {
        "name": "Dark patch 2",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "description": "The second dark patch",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": ""
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "772": {
        "name": "White spot",
        "locationIndexes": [
            14,
            5,
            94
        ],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "id5": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9,
            199
        ],
        "description": "Posted item!",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142581,
        "mod_unix_time_sec": 1668142581
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path), Body: bytes.NewReader([]byte(`{
    "id6": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9,
            199
        ],
        "description": "Posted item!",
        "imageName": "the_img.png",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142583,
        "mod_unix_time_sec": 1668142583
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
	idGen.ids = []string{"id3", "id4", "id5", "id6"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579, 1668142580, 1668142581, 1668142582, 1668142583},
	}
	apiRouter := MakeRouter(svcs)

	const postItem = `{
		"name": "White spot",
		"locationIndexes": [ 3, 9, 199 ],
		"description": "Posted item!"
	}`
	const postItemWithImageName = `{
		"name": "White spot",
		"imageName": "the_img.png",
		"locationIndexes": [ 3, 9, 199 ],
		"description": "Posted item!"
	}`

	const routePath = "/roi/TheDataSetID"

	// File not in S3, should work
	req, _ := http.NewRequest("POST", routePath, bytes.NewReader([]byte(postItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", routePath, bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already has an ROI by this name by another user, should work
	req, _ = http.NewRequest("POST", routePath, bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already has an ROI by this name by same user, should FAIL
	req, _ = http.NewRequest("POST", routePath, bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// With imageName field, should work
	req, _ = http.NewRequest("POST", routePath, bytes.NewReader([]byte(postItemWithImageName)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
	// 200
	//
	// 200
	//
	// 400
	// ROI name already used: White spot
	//
	// 200
}

func Example_roiHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path), Body: bytes.NewReader([]byte(`{
    "331": {
        "name": "White spot",
        "locationIndexes": [
            3,
            9,
            199
        ],
        "description": "Updated item!",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668142579
    },
    "772": {
        "name": "White spot",
        "locationIndexes": [
            14,
            5,
            94
        ],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}

	apiRouter := MakeRouter(svcs)

	const putItem = `{
		"name": "White spot",
		"locationIndexes": [ 3, 9, 199 ],
		"description": "Updated item!"
	}`

	// Put finds file missing, ERROR
	req, _ := http.NewRequest("PUT", "/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Put finds empty file, ERROR
	req, _ = http.NewRequest("PUT", "/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Put cant find item, ERROR
	req, _ = http.NewRequest("PUT", "/roi/TheDataSetID/22", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Put with bad name, ERROR
	req, _ = http.NewRequest("PUT", "/roi/TheDataSetID/22", bytes.NewReader([]byte(`{
		"name": "",
		"locationIndexes": [ 3, 9, 199 ],
		"description": "Updated item!"
	}`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Put shared item, ERROR
	req, _ = http.NewRequest("PUT", "/roi/TheDataSetID/shared-331", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Put finds item, OK
	req, _ = http.NewRequest("PUT", "/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// roi 331 not found
	//
	// 404
	// roi 331 not found
	//
	// 404
	// roi 22 not found
	//
	// 400
	// invalid roi name: ""
	//
	// 400
	// cannot edit shared rois
	//
	// 200
}

func Example_roiHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "331": {
        "name": "Dark patch 2",
        "description": "The second dark patch",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "creator": { "name": "Peter", "user_id": "600f2a0806b6c70071d3d174" },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000,
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        }
    },
    "772": {
        "name": "White spot",
        "locationIndexes": [
            14,
            5,
            94
        ],
        "creator": { "name": "Tom", "user_id": "u124" },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001,
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        }
    }
}`))),
		},
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path), Body: bytes.NewReader([]byte(`{
    "772": {
        "name": "White spot",
        "locationIndexes": [
            14,
            5,
            94
        ],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete shared item, OK
	req, _ := http.NewRequest("DELETE", "/roi/TheDataSetID/shared-331", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds file missing, ERROR
	req, _ = http.NewRequest("DELETE", "/roi/TheDataSetID/331", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/roi/TheDataSetID/331", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/roi/TheDataSetID/22", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/roi/TheDataSetID/331", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
	// 404
	// 331 not found
	//
	// 404
	// 331 not found
	//
	// 404
	// 22 not found
	//
	// 401
	// 331 not owned by 600f2a0806b6c70071d3d174
}

func Example_roiHandler_Share() {
	sharedROIContents := `{
    "99": {
        "name": "Shared already",
        "locationIndexes": [33],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": true,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174"
        },
        "create_unix_time_sec": 1668100008,
        "mod_unix_time_sec": 1668100008
    }
}`

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedROIContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedROIContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedROIContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(roi2XItems))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedROIContents))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(roiSharedS3Path), Body: bytes.NewReader([]byte(`{
    "16": {
        "name": "Dark patch 2",
        "locationIndexes": [
            4,
            55,
            394
        ],
        "description": "The second dark patch",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": true,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": ""
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668142579
    },
    "99": {
        "name": "Shared already",
        "locationIndexes": [
            33
        ],
        "description": "",
        "mistROIItem": {
            "species": "",
            "mineralGroupID": "",
            "ID_Depth": 0,
            "ClassificationTrail": "",
            "formula": ""
        },
        "shared": true,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": ""
        },
        "create_unix_time_sec": 1668100008,
        "mod_unix_time_sec": 1668100008
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"16"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/roi/TheDataSetID/333", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/roi/TheDataSetID/331", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 331 not found
	//
	// 404
	// 331 not found
	//
	// 404
	// 333 not found
	//
	// 200
	// "16"
}
