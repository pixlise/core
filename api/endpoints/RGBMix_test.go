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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
)

const rgbMixUserS3Path = "UserContent/600f2a0806b6c70071d3d174/RGBMixes.json"
const rgbMixSharedS3Path = "UserContent/shared/RGBMixes.json"
const rgbMixFileData = `{
	"abc123": {
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
	},
	"def456": {
		"name": "Ca-Fe-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.4,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.4,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 3.4,
			"rangeMax": 6.3
		},
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		}
	}
}`

func Example_RGBMixHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		// Shared items, NOTE this returns an old-style "element" for checking backwards compatibility!
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"ghi789": {
		"name": "Na-Fe-Al ratios",
		"red": {
			"expressionID": "expr-for-Na",
			"rangeMin": 1,
			"rangeMax": 2
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2,
			"rangeMax": 5
		},
		"blue": {
			"element": "Fe",
			"rangeMin": 3,
			"rangeMax": 6
		},
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/rgb-mix", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/rgb-mix", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/rgb-mix", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {}
	//
	// 200
	// {}
	//
	// 200
	// {
	//     "abc123": {
	//         "name": "Ca-Ti-Al ratios",
	//         "red": {
	//             "expressionID": "expr-for-Ca",
	//             "rangeMin": 1.5,
	//             "rangeMax": 4.3
	//         },
	//         "green": {
	//             "expressionID": "expr-for-Al",
	//             "rangeMin": 2.5,
	//             "rangeMax": 5.3
	//         },
	//         "blue": {
	//             "expressionID": "expr-for-Ti",
	//             "rangeMin": 3.5,
	//             "rangeMax": 6.3
	//         },
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "def456": {
	//         "name": "Ca-Fe-Al ratios",
	//         "red": {
	//             "expressionID": "expr-for-Ca",
	//             "rangeMin": 1.4,
	//             "rangeMax": 4.3
	//         },
	//         "green": {
	//             "expressionID": "expr-for-Al",
	//             "rangeMin": 2.4,
	//             "rangeMax": 5.3
	//         },
	//         "blue": {
	//             "expressionID": "expr-for-Fe",
	//             "rangeMin": 3.4,
	//             "rangeMax": 6.3
	//         },
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "shared-ghi789": {
	//         "name": "Na-Fe-Al ratios",
	//         "red": {
	//             "expressionID": "expr-for-Na",
	//             "rangeMin": 1,
	//             "rangeMax": 2
	//         },
	//         "green": {
	//             "expressionID": "expr-for-Al",
	//             "rangeMin": 2,
	//             "rangeMax": 5
	//         },
	//         "blue": {
	//             "expressionID": "expr-elem-Fe-%",
	//             "rangeMin": 3,
	//             "rangeMax": 6
	//         },
	//         "shared": true,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// }
}

func Example_RGBMixHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// POST not implemented! Should return 405
	req, _ := http.NewRequest("GET", "/rgb-mix/abc123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

func Example_RGBMixHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "rgbmix-id16": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "rgbmix-id17": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
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
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "rgbmix-id18": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
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
	idGen.ids = []string{"id16", "id17", "id18"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"name": "Sodium and stuff",
	"red": {
		"expressionID": "expr-for-Na",
		"rangeMin": 1,
		"rangeMax": 2
	},
	"green": {
		"expressionID": "expr-for-Fe",
		"rangeMin": 2,
		"rangeMax": 5
	},
	"blue": {
		"expressionID": "expr-for-Ti",
		"rangeMin": 3,
		"rangeMax": 6
	}
}`
	const putItemWithElement = `{
	"name": "Sodium and stuff",
	"red": {
		"expressionID": "expr-for-Na",
		"rangeMin": 1,
		"rangeMax": 2
	},
	"green": {
		"element": "Fe",
		"rangeMin": 2,
		"rangeMax": 5
	},
	"blue": {
		"expressionID": "expr-for-Ti",
		"rangeMin": 3,
		"rangeMax": 6
	}
}`

	// File not in S3, should work
	req, _ := http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already contains stuff, this is added
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Adding old-style with element defined, should fail
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItemWithElement)))
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
	// RGB Mix definition with elements is deprecated
}

func Example_RGBMixHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
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
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "def456": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
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
		"name": "Sodium and stuff",
		"red": {
			"expressionID": "expr-for-Na",
			"rangeMin": 1,
			"rangeMax": 2
		},
		"green": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 2,
			"rangeMax": 5
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3,
			"rangeMax": 6
		}
	}`

	// File not in S3, not found
	req, _ := http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, not found
	req, _ = http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already this id, should overwrite
	req, _ = http.NewRequest("PUT", "/rgb-mix/def456", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain this id, not found
	req, _ = http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Can't edit shared ids
	req, _ = http.NewRequest("PUT", "/rgb-mix/shared-111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// aaa111 not found
	//
	// 404
	// aaa111 not found
	//
	// 200
	//
	// 404
	// aaa111 not found
	//
	// 400
	// Cannot edit shared RGB mixes
}

func Example_RGBMixHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": false,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc999", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/rgb-mix/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// abc123 not found
	//
	// 404
	// abc123 not found
	//
	// 404
	// abc999 not found
	//
	// 200
	//
	// 401
	// def456 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
}

func Example_RGBMixHandler_Share() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": true,
        "creator": {
            "name": "The sharer",
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path), Body: bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": true,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    },
    "rgbmix-ddd222": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
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
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"ddd222"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/rgb-mix/zzz222", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/rgb-mix/def456", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// abc123 not found
	//
	// 404
	// abc123 not found
	//
	// 404
	// zzz222 not found
	//
	// 200
	// "rgbmix-ddd222"
}

func Example_RGBMixHandler_Share_Fail() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "abc123": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "shared-abcd123",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "xyz123",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": false,
        "creator": {
            "name": "Niko",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User trying to share RGB mix with non-shared expressions, should fail
	req, _ := http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// When sharing RGB mix, it must only reference shared expressions
}
