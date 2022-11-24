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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"gopkg.in/auth0.v4/management"

	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
)

// TODO: this is not really a unit test, as it calls out to Auth0 and interacts with a test user, adding/removing a role!
// We should probably stub out the Auth0 interaction, but then what are we testing, the code does very little besides provide
// an interface to call through to Auth0 from.

// For now, for this test to pass, you need to define the following env variables:
// PIXLISE_API_TEST_AUTH0_DOMAIN
// PIXLISE_API_TEST_AUTH0_CLIENT_ID
// PIXLISE_API_TEST_AUTH0_SECRET

// NOTE: in VS Code you need to add to settings.json: go.testEnvVars to JSON {"NAME": "VAR", ...} to have the above
// so you can click "run test" in this file

// Setting test Auth keys
func setTestAuth0Config(svcs *services.APIServices) {
	svcs.Config.Auth0Domain = os.Getenv("PIXLISE_API_TEST_AUTH0_DOMAIN")
	svcs.Config.Auth0ManagementClientID = os.Getenv("PIXLISE_API_TEST_AUTH0_CLIENT_ID")
	svcs.Config.Auth0ManagementSecret = os.Getenv("PIXLISE_API_TEST_AUTH0_SECRET")

	if len(svcs.Config.Auth0ManagementClientID) <= 0 {
		panic("Missing one or more env vars for testing: PIXLISE_API_TEST_AUTH0_DOMAIN, PIXLISE_API_TEST_AUTH0_CLIENT_ID, PIXLISE_API_TEST_AUTH0_SECRET")
	}
}

// Using a known user & role:
// User is: test@pixlise.org
const knownTestUserID = "auth0|5f45d7b8b5abff006d4fdb91"

// Role is: "No Permissions"
const knownTestRoleID = "rol_KdjHrTCteclbY7om"

func seemsValid(user auth0UserInfo) bool {
	return len(user.UserID) > 0 && len(user.Name) > 0 && len(user.Email) > 0 //&& user.CreatedUnixSec > 0
}

func Example_userManagementUserQuery_And_UserGet() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/user/query", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("query: %v\n", resp.Code)

	var users []auth0UserInfo
	err := json.Unmarshal(resp.Body.Bytes(), &users)

	fmt.Printf("%v|%v\n", err, len(users) > 0 && seemsValid(users[0]))

	req, _ = http.NewRequest("GET", "/user/by-id/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("by-id: %v\n", resp.Code)

	var user auth0UserInfo
	err = json.Unmarshal(resp.Body.Bytes(), &user)

	fmt.Printf("%v|%v\n", err, seemsValid(user))

	// Output:
	// query: 200
	// <nil>|true
	// by-id: 200
	// <nil>|true
}

func Example_userManagement_AddDeleteRole() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	// To test add/delete of roles:

	// In case test ran before, we call delete first, then list roles, ensuring it's not in there.
	req, _ := http.NewRequest("DELETE", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-del: %v\n", resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("check-del: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// We then add the role, list roles, ensure it's there
	req, _ = http.NewRequest("POST", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("add: %v\n", resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-added: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Finally, delete role, list roles, ensure it's gone
	req, _ = http.NewRequest("DELETE", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("delete: %v\n", resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-del-2: %v\n", resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("add-back: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// ensure-del: 200
	//
	// check-del: 200
	// null
	//
	// add: 200
	//
	// ensure-added: 200
	// [
	//     {
	//         "id": "rol_KdjHrTCteclbY7om",
	//         "name": "No Permissions",
	//         "description": "When a user has signed up and we don't know who they are, we assign this."
	//     }
	// ]
	//
	// delete: 200
	//
	// ensure-del-2: 200
	// null
	//
	// add-back: 200
}

func Example_userManagement_Roles_And_UserByRole() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/user/all-roles", nil)
	resp := executeRequest(req, apiRouter.Router)

	// Expect to parse this thing, and get SOME roles back
	var roles []roleInfo
	err := json.Unmarshal(resp.Body.Bytes(), &roles)

	fmt.Println(resp.Code)
	fmt.Printf("%v|%v\n", err, len(roles) > 0 && len(roles[0].ID) > 0 && len(roles[0].Name) > 0 && len(roles[0].Description) > 0)

	// request users for first role, hopefully we have some assigned, else test will fail!
	if len(roles) > 0 {
		req, _ = http.NewRequest("GET", "/user/by-role/"+roles[0].ID, nil)
		resp = executeRequest(req, apiRouter.Router)

		var users []auth0UserInfo
		err = json.Unmarshal(resp.Body.Bytes(), &users)

		fmt.Println(resp.Code)
		fmt.Printf("%v|%v\n", err, len(users) > 0 && seemsValid(users[0]))
	}

	// Output:
	// 200
	// <nil>|true
	// 200
	// <nil>|true
}

func Test_user_config_get(t *testing.T) {
	expectedResponse := `{
    "name": "Niko Bellic",
    "email": "niko@spicule.co.uk",
    "cell": "+123456789",
    "data_collection": "true"
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
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
	}

	runOneURLCallTest(t, "GET", "/user/config", nil, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_user_config_post(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`{"name": "Niko Bellic", "email": "niko@spicule.co.uk","cell": "+1234567890","data_collection": "false"}`))

	expectedResponse := `{
    "name": "Niko Bellic",
    "email": "niko@spicule.co.uk",
    "cell": "+1234567890",
    "data_collection": "false"
}
`
	mockMongoResponses := []primitive.D{
		// User read
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
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
		// User saved
		mtest.CreateSuccessResponse(),
	}

	runOneURLCallTest(t, "POST", "/user/config", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}

func Test_user_edit_field_name(t *testing.T) {
	// Get auth0 api config
	svcs := MakeMockSvcs(nil, nil, nil, nil)
	setTestAuth0Config(&svcs)
	api, err := InitAuth0ManagementAPI(svcs.Config)
	if err != nil {
		t.Errorf("Failed to init auth0 API: %v", err)
	}

	auth0User := management.User{}
	preTestName := "TEST USER - test commenced"
	auth0User.Name = &preTestName

	err = api.User.Update("auth0|600f2a0806b6c70071d3d174", &auth0User)
	if err != nil {
		t.Errorf("Failed to set initial user name: %v", err)
	}

	requestPayload := bytes.NewReader([]byte(`"TEST USER"`))

	expectedResponse := ""

	mockMongoResponses := []primitive.D{
		// User read
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
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
		// User saved
		mtest.CreateSuccessResponse(),
	}

	// TODO: This isn't a good test, we have no way of verifying what was written into mongo! But we can verify what was written to auth0
	runOneURLCallTest(t, "PUT", "/user/field/name", requestPayload, 200, expectedResponse, mockMongoResponses, func() {
		user, err := api.User.Read("auth0|600f2a0806b6c70071d3d174")
		if err != nil {
			t.Errorf("Failed to query user after test: %v", err)
		}

		if *user.Name != "TEST USER" {
			t.Errorf("Expected auth0 user to be named TEST USER not: %v", *user.Name)
		}
	})
}

func Test_user_edit_field_email(t *testing.T) {
	// Get auth0 api config
	svcs := MakeMockSvcs(nil, nil, nil, nil)
	setTestAuth0Config(&svcs)
	api, err := InitAuth0ManagementAPI(svcs.Config)
	if err != nil {
		t.Errorf("Failed to init auth0 API: %v", err)
	}

	auth0User := management.User{}
	preTestEmail := "test_user_commenced@pixlise.org"
	auth0User.Email = &preTestEmail

	err = api.User.Update("auth0|600f2a0806b6c70071d3d174", &auth0User)
	if err != nil {
		t.Errorf("Failed to set initial user email: %v", err)
	}

	requestPayload := bytes.NewReader([]byte(`"test_user@pixlise.org"`))

	expectedResponse := ""

	mockMongoResponses := []primitive.D{
		// User read
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
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
		// User saved
		mtest.CreateSuccessResponse(),
	}

	// TODO: This isn't a good test, we have no way of verifying what was written into mongo! But we can verify what was written to auth0
	runOneURLCallTest(t, "PUT", "/user/field/email", requestPayload, 200, expectedResponse, mockMongoResponses, func() {
		user, err := api.User.Read("auth0|600f2a0806b6c70071d3d174")
		if err != nil {
			t.Errorf("Failed to query user after test: %v", err)
		}

		if *user.Email != "test_user@pixlise.org" {
			t.Errorf("Expected auth0 user to have email test_user@pixlise.org not: %v", *user.Email)
		}
	})
}

func Example_user_edit_field_error() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("PUT", "/user/field/flux", bytes.NewReader([]byte("Something")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("status: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// status: 500
	// Unrecognised field: flux
}

func Test_user_bulk_edit(t *testing.T) {
	requestPayload := bytes.NewReader([]byte(`[
	{
		"UserID": "123",
		"Name": "Michael Collins"
	},
	{
		"UserID": "456",
		"Name": "Neil Armstrong"
	}
]`))

	expectedResponse := ""

	mockMongoResponses := []primitive.D{
		// User read
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "123"},
				{"Config", bson.D{
					{"Name", "collins"},
					{"Email", "collins@space.com"},
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
		// User saved
		mtest.CreateSuccessResponse(),
		// User read
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.users",
			mtest.FirstBatch,
			bson.D{
				{"Userid", "456"},
				{"Config", bson.D{
					{"Name", "neil"},
					{"Email", "neil@space.com"},
					{"Cell", "+123456789"},
					{"DataCollection", "true"},
				}},
			},
		),
		// User saved
		mtest.CreateSuccessResponse(),
	}

	// TODO: This isn't a good test, we have no way of verifying what was written into mongo! But we can verify what was written to auth0
	runOneURLCallTest(t, "POST", "/user/bulk-user-details", requestPayload, 200, expectedResponse, mockMongoResponses, nil)
}
