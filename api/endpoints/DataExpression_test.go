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
	"github.com/pixlise/core/v2/core/awsutil"
)

const exprS3Path = "UserContent/600f2a0806b6c70071d3d174/DataExpressions.json"
const exprSharedS3Path = "UserContent/shared/DataExpressions.json"
const exprFile = `{
	"abc123": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "ContextImage",
		"comments": "comments for abc123 expression",
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		}
	},
	"def456": {
		"name": "Iron Error",
		"expression": "element(\"Fe\", \"err\")",
		"type": "BinaryPlot",
		"comments": "comments for def456 expression",
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		}
	}
}`

func Example_dataExpressionHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			// Note: No comments!
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"ghi789": {
		"name": "Iron %",
		"expression": "element(\"Fe\", \"%\")",
		"type": "TernaryPlot",
		"creator": {
			"user_id": "999",
			"name": "Peter N",
			"email": "niko@spicule.co.uk"
		}
	}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/data-expression", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/data-expression", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/data-expression", nil)
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
	//         "name": "Calcium weight%",
	//         "expression": "element(\"Ca\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "comments for abc123 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "def456": {
	//         "name": "Iron Error",
	//         "expression": "element(\"Fe\", \"err\")",
	//         "type": "BinaryPlot",
	//         "comments": "comments for def456 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "shared-ghi789": {
	//         "name": "Iron %",
	//         "expression": "element(\"Fe\", \"%\")",
	//         "type": "TernaryPlot",
	//         "comments": "",
	//         "shared": true,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// }
}

func Example_dataExpressionHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// POST not implemented! Should return 405
	req, _ := http.NewRequest("GET", "/data-expression/abc123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

func Example_dataExpressionHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "id16": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "id17": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
        "name": "Calcium weight%",
        "expression": "element(\"Ca\", \"%\")",
        "type": "ContextImage",
        "comments": "comments for abc123 expression",
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "def456": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "id18": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
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
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"name": "Sodium weight%",
	"expression": "element(\"Na\", \"%\")",
	"type": "ContextImage",
	"comments": "sodium comment here"
}`

	// File not in S3, should work
	req, _ := http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already contains stuff, this is added
	req, _ = http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "id16": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
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
	//     "id17": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
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
	//     "abc123": {
	//         "name": "Calcium weight%",
	//         "expression": "element(\"Ca\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "comments for abc123 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "def456": {
	//         "name": "Iron Error",
	//         "expression": "element(\"Fe\", \"err\")",
	//         "type": "BinaryPlot",
	//         "comments": "comments for def456 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "id18": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// }
}

func Example_dataExpressionHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
        "name": "Calcium weight%",
        "expression": "element(\"Ca\", \"%\")",
        "type": "ContextImage",
        "comments": "comments for abc123 expression",
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        }
    },
    "def456": {
        "name": "Iron Int",
        "expression": "element(\"Fe\", \"int\")",
        "type": "TernaryPlot",
        "comments": "Iron comment",
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
		"name": "Iron Int",
        "expression": "element(\"Fe\", \"int\")",
        "type": "TernaryPlot",
        "comments": "Iron comment"
	}`

	// File not in S3, not found
	req, _ := http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, not found
	req, _ = http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already this id, should overwrite
	req, _ = http.NewRequest("PUT", "/data-expression/def456", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain this id, not found
	req, _ = http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Can't edit shared ids
	req, _ = http.NewRequest("PUT", "/data-expression/shared-111", bytes.NewReader([]byte(putItem)))
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
	// {
	//     "abc123": {
	//         "name": "Calcium weight%",
	//         "expression": "element(\"Ca\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "comments for abc123 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     },
	//     "def456": {
	//         "name": "Iron Int",
	//         "expression": "element(\"Fe\", \"int\")",
	//         "type": "TernaryPlot",
	//         "comments": "Iron comment",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// }
	//
	// 404
	// aaa111 not found
	//
	// 400
	// Cannot edit shared expressions
}

func Example_dataExpressionHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "def456": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "def456": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/abc999", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/data-expression/shared-def456", nil)
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
	// {
	//     "def456": {
	//         "name": "Iron Error",
	//         "expression": "element(\"Fe\", \"err\")",
	//         "type": "BinaryPlot",
	//         "comments": "comments for def456 expression",
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         }
	//     }
	// }
	//
	// 401
	// def456 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
	// {}
}

func Example_dataExpressionHandler_Share() {
	sharedExpressionsContents := `{
		"aaa333": {
			"name": "Calcium Error",
			"expression": "element(\"Ca\", \"err\")",
			"type": "TernaryPlot",
			"comments": "calcium comments",
			"shared": false,
			"creator": {
				"name": "The sharer",
				"user_id": "600f2a0806b6c70071d3d174",
				"email": "niko@spicule.co.uk"
			}
		}
	}`
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path), Body: bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "Calcium Error",
        "expression": "element(\"Ca\", \"err\")",
        "type": "TernaryPlot",
        "comments": "calcium comments",
        "shared": false,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    },
    "ddd222": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
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
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/data-expression/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/data-expression/abc123", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/data-expression/zzz222", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/data-expression/def456", bytes.NewReader([]byte(putItem)))
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
	// "ddd222"
}
