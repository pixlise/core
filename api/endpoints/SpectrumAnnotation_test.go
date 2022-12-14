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

const annotationS3Path = "UserContent/600f2a0806b6c70071d3d174/rtt-123/SpectrumAnnotation.json"
const annotationSharedS3Path = "UserContent/shared/rtt-123/SpectrumAnnotation.json"
const annotations2x = `{
	"5": {
		"eV": 12345,
		"roiID": "roi123",
		"name": "Weird part of spectrum",
		"shared": false,
		"creator": { "name": "Tom", "user_id": "u124", "email":"niko@spicule.co.uk" },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
	},
	"8": {
		"eV": 555,
		"roiID": "roi123",
		"name": "Left of spectrum",
		"shared": false,
		"creator": { "name": "Peter", "user_id": "u123", "email":"niko@spicule.co.uk" },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
	}
}`

func Test_spectrumAnnotationHandler_List(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// User u124 (first one)
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "u124"},
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
			// u123 - not found
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
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()
		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
			},
		}
		mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
			nil,
			nil,
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
		"93": {
			"eV": 20000,
			"name": "right of spectrum",
			"roiID": "roi111",
			"shared": true,
			"creator": { "name": "Tom", "user_id": "u124", "email": "" },
			"create_unix_time_sec": 1668100002,
			"mod_unix_time_sec": 1668100003
		}
	}`))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/annotation/rtt-123", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/annotation/rtt-123", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/annotation/rtt-123", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "5": {
        "name": "Weird part of spectrum",
        "roiID": "roi123",
        "eV": 12345,
        "shared": false,
        "creator": {
            "name": "Tom Barber",
            "user_id": "u124",
            "email": "tom@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "8": {
        "name": "Left of spectrum",
        "roiID": "roi123",
        "eV": 555,
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "shared-93": {
        "name": "right of spectrum",
        "roiID": "roi111",
        "eV": 20000,
        "shared": true,
        "creator": {
            "name": "Tom Barber",
            "user_id": "u124",
            "email": "tom@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100002,
        "mod_unix_time_sec": 1668100003
    }
}
`)
	})
}

func Test_spectrumAnnotationHandler_Get(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "u123"},
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
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
			},
		}
		mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
			nil,
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		// File not in S3, should return 404
		req, _ := http.NewRequest("GET", "/annotation/rtt-123/8", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `8 not found
`)

		// File in S3 empty, should return 404
		req, _ = http.NewRequest("GET", "/annotation/rtt-123/8", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `8 not found
`)

		// File contains stuff, using ID thats not in there, should return 404
		req, _ = http.NewRequest("GET", "/annotation/rtt-123/6", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `6 not found
`)

		// File contains stuff, using ID that exists
		req, _ = http.NewRequest("GET", "/annotation/rtt-123/8", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Left of spectrum",
    "roiID": "roi123",
    "eV": 555,
    "shared": false,
    "creator": {
        "name": "Tom Barber",
        "user_id": "u123",
        "email": "tom@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100001,
    "mod_unix_time_sec": 1668100001
}
`)

		// Check that shared file was loaded if shared ID sent in
		req, _ = http.NewRequest("GET", "/annotation/rtt-123/shared-8", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Left of spectrum",
    "roiID": "roi123",
    "eV": 555,
    "shared": true,
    "creator": {
        "name": "Tom Barber",
        "user_id": "u123",
        "email": "tom@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100001,
    "mod_unix_time_sec": 1668100001
}
`)
	})
}

func Example_spectrumAnnotationHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
	}
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path), Body: bytes.NewReader([]byte(`{
    "id1": {
        "name": "The modified flag",
        "roiID": "roi222",
        "eV": 9999,
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path), Body: bytes.NewReader([]byte(`{
    "id2": {
        "name": "The modified flag",
        "roiID": "roi222",
        "eV": 9999,
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path), Body: bytes.NewReader([]byte(`{
    "5": {
        "name": "Weird part of spectrum",
        "roiID": "roi123",
        "eV": 12345,
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "8": {
        "name": "Left of spectrum",
        "roiID": "roi123",
        "eV": 555,
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "id3": {
        "name": "The modified flag",
        "roiID": "roi222",
        "eV": 9999,
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
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"id1", "id2", "id3"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579, 1668142580, 1668142581},
	}
	apiRouter := MakeRouter(svcs)

	body := `{
	"name": "The modified flag",
	"roiID": "roi222",
	"eV": 9999
}`

	req, _ := http.NewRequest("POST", "/annotation/rtt-123", bytes.NewReader([]byte(body)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/annotation/rtt-123", bytes.NewReader([]byte(body)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/annotation/rtt-123", bytes.NewReader([]byte(body)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "id1": {
	//         "name": "The modified flag",
	//         "roiID": "roi222",
	//         "eV": 9999,
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142579,
	//         "mod_unix_time_sec": 1668142579
	//     }
	// }
	//
	// 200
	// {
	//     "id2": {
	//         "name": "The modified flag",
	//         "roiID": "roi222",
	//         "eV": 9999,
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142580,
	//         "mod_unix_time_sec": 1668142580
	//     }
	// }
	//
	// 200
	// {
	//     "5": {
	//         "name": "Weird part of spectrum",
	//         "roiID": "roi123",
	//         "eV": 12345,
	//         "shared": false,
	//         "creator": {
	//             "name": "Tom",
	//             "user_id": "u124",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668100000,
	//         "mod_unix_time_sec": 1668100000
	//     },
	//     "8": {
	//         "name": "Left of spectrum",
	//         "roiID": "roi123",
	//         "eV": 555,
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter",
	//             "user_id": "u123",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668100001,
	//         "mod_unix_time_sec": 1668100001
	//     },
	//     "id3": {
	//         "name": "The modified flag",
	//         "roiID": "roi222",
	//         "eV": 9999,
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142581,
	//         "mod_unix_time_sec": 1668142581
	//     }
	// }
}

func Example_spectrumAnnotationHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path), Body: bytes.NewReader([]byte(`{
    "5": {
        "name": "Updated Item",
        "roiID": "roi444",
        "eV": 8888,
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668142579
    },
    "8": {
        "name": "Left of spectrum",
        "roiID": "roi123",
        "eV": 555,
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": "niko@spicule.co.uk"
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
    "name": "Updated Item",
    "roiID": "roi444",
    "eV": 8888
}`

	const routePath = "/annotation/rtt-123/3"

	// File not in S3, should work
	req, _ := http.NewRequest("PUT", routePath, bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("PUT", routePath, bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// ROI annotations for this exist, but we're adding a new annotation
	req, _ = http.NewRequest("PUT", routePath, bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// ROI annotations for this exist, but we're editing an existing annotation
	req, _ = http.NewRequest("PUT", "/annotation/rtt-123/5", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 3 not found
	//
	// 404
	// 3 not found
	//
	// 404
	// 3 not found
	//
	// 200
	// {
	//     "5": {
	//         "name": "Updated Item",
	//         "roiID": "roi444",
	//         "eV": 8888,
	//         "shared": false,
	//         "creator": {
	//             "name": "Tom",
	//             "user_id": "u124",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668100000,
	//         "mod_unix_time_sec": 1668142579
	//     },
	//     "8": {
	//         "name": "Left of spectrum",
	//         "roiID": "roi123",
	//         "eV": 555,
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter",
	//             "user_id": "u123",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668100001,
	//         "mod_unix_time_sec": 1668100001
	//     }
	// }
}

func Example_spectrumAnnotationHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "25": {
        "eV": 12345,
        "name": "Weird part of spectrum",
        "roiID": "roi123",
        "shared": false,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path), Body: bytes.NewReader([]byte(`{
    "8": {
        "name": "Left of spectrum",
        "roiID": "roi123",
        "eV": 555,
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const routePath = "/annotation/rtt-123/3"

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", routePath, nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", routePath, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", routePath, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/annotation/rtt-123/5", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/annotation/rtt-123/shared-5", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/annotation/rtt-123/shared-25", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 3 not found
	//
	// 404
	// 3 not found
	//
	// 404
	// 3 not found
	//
	// 200
	//
	// 401
	// 5 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
}

func Example_spectrumAnnotationHandler_Share() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(annotations2x))),
		},
		// Shared file
		// NOTE that no create_unix_time_sec or mod_unix_time_sec supplied
		// this is because we didn't have this field in the past, pretend this is an
		// old shared file, and see if the put at the end omits the empty 2 fields
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "25": {
        "eV": 12345,
        "roiID": "roi123",
        "name": "Weird part of spectrum",
        "shared": true,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path), Body: bytes.NewReader([]byte(`{
    "25": {
        "name": "Weird part of spectrum",
        "roiID": "roi123",
        "eV": 12345,
        "shared": true,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    },
    "83": {
        "name": "Left of spectrum",
        "roiID": "roi123",
        "eV": 555,
        "shared": true,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": "niko@spicule.co.uk"
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

	var idGen MockIDGenerator
	idGen.ids = []string{"83"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/annotation/rtt-123/8", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/annotation/rtt-123/8", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/annotation/rtt-123/7", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/annotation/rtt-123/8", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 8 not found
	//
	// 404
	// 8 not found
	//
	// 404
	// 7 not found
	//
	// 200
	// "83"
}
