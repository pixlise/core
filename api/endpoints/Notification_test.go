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
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_subscription_get(t *testing.T) {
	expectedResponse := `{
    "topics": [
        {
            "name": "topic z",
            "config": {
                "method": {
                    "ui": true,
                    "sms": false,
                    "email": true
                }
            }
        }
    ]
}
`
	mockMongoResponses := []primitive.D{
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "600f2a0806b6c70071d3d174"},
				{"Notifications", bson.D{
					{"Topics", bson.A{
						bson.D{
							{"Name", "topic z"},
							{"Config", bson.D{
								{"Method", bson.D{
									{"ui", true},
									{"sms", false},
									{"email", true},
								}},
							}},
						}}},
				}},
				{"Config", bson.D{
					{"Name", "Niko Bellic"},
					{"Email", "niko@spicule.co.uk"},
					{"Cell", ""},
					{"DataCollection", "unknown"},
				}},
			},
		),
	}

	runOneURLCallTest(t, "GET", "/notification/subscriptions", nil, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_subscriptions_get_empty_topics(t *testing.T) {
	expectedResponse := `{
    "topics": []
}
`
	mockMongoResponses := []primitive.D{
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "600f2a0806b6c70071d3d174"},
				{"Notifications", bson.D{
					{"Topics", bson.A{}},
				}},
				{"Config", bson.D{
					{"Name", "Niko Bellic"},
					{"Email", "niko@spicule.co.uk"},
					{"Cell", ""},
					{"DataCollection", "unknown"},
				}},
			},
		),
	}

	runOneURLCallTest(t, "GET", "/notification/subscriptions", nil, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_subscriptions_get_no_user(t *testing.T) {
	expectedResponse := `600f2a0806b6c70071d3d174 not found
`
	mockMongoResponses := []primitive.D{
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

	runOneURLCallTest(t, "GET", "/notification/subscriptions", nil, 404, expectedResponse, mockMongoResponses, nil)
}

func Test_subscription_post(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`{"topics": [{
	"name": "topic c",
	"config": {
		"method": {
			"ui": true,
			"sms": false,
			"email": false
		}
	}
}, {
	"name": "topic d",
	"config": {
		"method": {
			"ui": true,
			"sms": false,
			"email": false
		}
	}
}]}`))

	expectedResponse := `{
    "topics": [
        {
            "name": "topic c",
            "config": {
                "method": {
                    "ui": true,
                    "sms": false,
                    "email": false
                }
            }
        },
        {
            "name": "topic d",
            "config": {
                "method": {
                    "ui": true,
                    "sms": false,
                    "email": false
                }
            }
        }
    ]
}
`
	mockMongoResponses := []primitive.D{
		mtest.CreateCursorResponse(
			1,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "600f2a0806b6c70071d3d174"},
				{"Notifications", bson.D{
					{"Topics", bson.A{}},
				}},
				{"Config", bson.D{
					{"Name", "Niko Bellic"},
					{"Email", "niko@spicule.co.uk"},
					{"Cell", ""},
					{"DataCollection", "unknown"},
				}},
			},
		),
		mtest.CreateSuccessResponse(), // NOTE: not sure where this gets gobbled up...
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "POST", "/notification/subscriptions", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_subscription_post_no_user(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`{"topics": [{
	"name": "topic c",
	"config": {
		"method": {
			"ui": true,
			"sms": false,
			"email": false
		}
	}
}, {
	"name": "topic d",
	"config": {
		"method": {
			"ui": true,
			"sms": false,
			"email": false
		}
	}
}]}`))

	expectedResponse := `{
    "topics": [
        {
            "name": "topic c",
            "config": {
                "method": {
                    "ui": true,
                    "sms": false,
                    "email": false
                }
            }
        },
        {
            "name": "topic d",
            "config": {
                "method": {
                    "ui": true,
                    "sms": false,
                    "email": false
                }
            }
        }
    ]
}
`
	mockMongoResponses := []primitive.D{
		// Signify no user exists...
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
		// User saved
		mtest.CreateSuccessResponse(),
		// User overwritten (with topic set)
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "POST", "/notification/subscriptions", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_alerts_get(t *testing.T) {
	expectedResponse := `[
    {
        "topic": "test-data-source",
        "message": "New Data Source Available",
        "timestamp": "2021-02-01T01:01:01Z",
        "userid": "600f2a0806b6c70071d3d174"
    },
    {
        "topic": "test-data-source",
        "message": "Another Source Available",
        "timestamp": "2021-02-04T01:01:01Z",
        "userid": "600f2a0806b6c70071d3d174"
    }
]
`
	mockMongoResponses := []primitive.D{
		// Get user request
		mtest.CreateCursorResponse(
			1,
			"userdatabase-unit_test.notifications",
			mtest.FirstBatch,
			bson.D{
				{"Topic", "test-data-source"},
				{"Message", "New Data Source Available"},
				{"Timestamp", "2021-02-01T01:01:01.000Z"},
				{"Userid", "600f2a0806b6c70071d3d174"},
			},
		),
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.notifications",
			mtest.NextBatch,
			bson.D{
				{"Topic", "test-data-source"},
				{"Message", "Another Source Available"},
				{"Timestamp", "2021-02-04T01:01:01.000Z"},
				{"Userid", "600f2a0806b6c70071d3d174"},
			},
		),
		// Deleted alerts
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "GET", "/notification/alerts", nil, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_alerts_no_user(t *testing.T) {
	runOneURLCallTest(t, "GET", "/notification/alerts", nil, 200, `[]
`, makeNotFoundMongoResponse(), nil)
}

func Test_hints_no_user(t *testing.T) {
	runOneURLCallTest(t, "GET", "/notification/hints", nil, 200, `{
    "hints": []
}
`, makeNotFoundMongoResponse(), nil)
}

func Test_hints_post(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`{
    "hints": [
        "hint c",
        "hint d"
    ]
}
`))
	expectedResponse := `{
    "hints": [
        "hint c",
        "hint d"
    ]
}
`
	mockMongoResponses := []primitive.D{
		// Get user
		mtest.CreateCursorResponse(
			1,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "600f2a0806b6c70071d3d174"},
				{"Notifications", bson.D{
					{"Topics", bson.A{}},
				}},
				{"Config", bson.D{
					{"Name", "Niko Bellic"},
					{"Email", "niko@spicule.co.uk"},
					{"Cell", ""},
					{"DataCollection", "unknown"},
				}},
			},
		),
		mtest.CreateSuccessResponse(), // Not sure what gobbles this up
		// Saved hints in mongo
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "POST", "/notification/hints", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_hints_post_no_user(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`{
    "hints": [
        "hint c",
        "hint d"
    ]
}
`))
	expectedResponse := `{
    "hints": [
        "hint c",
        "hint d"
    ]
}
`
	mockMongoResponses := []primitive.D{
		// Get user (none)
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
		// Write user
		mtest.CreateSuccessResponse(),
		// Saved hints in mongo
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "POST", "/notification/hints", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}
