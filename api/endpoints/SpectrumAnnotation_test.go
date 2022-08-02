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

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
)

const annotationS3Path = "UserContent/600f2a0806b6c70071d3d174/rtt-123/SpectrumAnnotation.json"
const annotationSharedS3Path = "UserContent/shared/rtt-123/SpectrumAnnotation.json"
const annotations2x = `{
	"5": {
		"eV": 12345,
		"roiID": "roi123",
		"name": "Weird part of spectrum",
		"shared": false,
		"creator": { "name": "Tom", "user_id": "u124", "email":"niko@spicule.co.uk" }
	},
	"8": {
		"eV": 555,
		"roiID": "roi123",
		"name": "Left of spectrum",
		"shared": false,
		"creator": { "name": "Peter", "user_id": "u123", "email":"niko@spicule.co.uk" }
	}
}`

func Example_spectrumAnnotationHandler_List() {
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
		"creator": { "name": "Tom", "user_id": "u124", "email": "" }
	}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/annotation/rtt-123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/annotation/rtt-123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/annotation/rtt-123", nil)
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
	//     "5": {
	//         "name": "Weird part of spectrum",
	//         "roiID": "roi123",
	//         "eV": 12345,
	//         "shared": false,
	//         "creator": {
	//             "name": "Tom",
	//             "user_id": "u124",
	//             "email": "niko@spicule.co.uk"
	//         }
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
	//         }
	//     },
	//     "shared-93": {
	//         "name": "right of spectrum",
	//         "roiID": "roi111",
	//         "eV": 20000,
	//         "shared": true,
	//         "creator": {
	//             "name": "Tom",
	//             "user_id": "u124",
	//             "email": ""
	//         }
	//     }
	// }
}

func Example_spectrumAnnotationHandler_Get() {
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File not in S3, should return 404
	req, _ := http.NewRequest("GET", "/annotation/rtt-123/8", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File in S3 empty, should return 404
	req, _ = http.NewRequest("GET", "/annotation/rtt-123/8", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains stuff, using ID thats not in there, should return 404
	req, _ = http.NewRequest("GET", "/annotation/rtt-123/6", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains stuff, using ID that exists
	req, _ = http.NewRequest("GET", "/annotation/rtt-123/8", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Check that shared file was loaded if shared ID sent in
	req, _ = http.NewRequest("GET", "/annotation/rtt-123/shared-8", nil)
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
	// 6 not found
	//
	// 200
	// {
	//     "name": "Left of spectrum",
	//     "roiID": "roi123",
	//     "eV": 555,
	//     "shared": false,
	//     "creator": {
	//         "name": "Peter",
	//         "user_id": "u123",
	//         "email": "niko@spicule.co.uk"
	//     }
	// }
	//
	// 200
	// {
	//     "name": "Left of spectrum",
	//     "roiID": "roi123",
	//     "eV": 555,
	//     "shared": true,
	//     "creator": {
	//         "name": "Peter",
	//         "user_id": "u123",
	//         "email": "niko@spicule.co.uk"
	//     }
	// }
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
		s3.PutObjectInput{
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
        }
    }
}`)),
		},
		s3.PutObjectInput{
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
        }
    }
}`)),
		},
		s3.PutObjectInput{
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
        }
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
        }
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
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"id1", "id2", "id3"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
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
	//         }
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
	//         }
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
	//         }
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
	//         }
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
	//         }
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
		s3.PutObjectInput{
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
        }
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
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
	//         }
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
	//         }
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
		s3.PutObjectInput{
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
        }
    }
}`)),
		},
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(annotationSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
		s3.PutObjectInput{
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
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"83"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
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
