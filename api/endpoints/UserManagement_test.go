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
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

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

func Example_user_config() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	userJSON := `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"test-dataset-available","config":{"method":{"ui":true,"sms":true,"email":true}}}],"hints":["hint a","hint b"],"uinotifications":null},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"+123456789","data_collection":"true"}}`
	userUpdatedJSON := `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"test-dataset-available","config":{"method":{"ui":true,"sms":true,"email":true}}}],"hints":["hint a","hint b"],"uinotifications":null},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"+123456789","data_collection":"false"}}`

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userJSON))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(userUpdatedJSON)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)

	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/user/config", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(fmt.Sprintf("ensure-valid: %v", resp.Code))
	fmt.Println(resp.Body)

	j := `{"name": "Niko Bellic", "email": "niko@spicule.co.uk","cell": "+123456789","data_collection": "false"}`

	req, _ = http.NewRequest("POST", "/user/config", bytes.NewReader([]byte(j)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(fmt.Sprintf("ensure-valid: %v", resp.Code))
	fmt.Println(resp.Body)

	// Output:
	// ensure-valid: 200
	// {
	//     "name": "Niko Bellic",
	//     "email": "niko@spicule.co.uk",
	//     "cell": "+123456789",
	//     "data_collection": "true"
	// }
	//
	// ensure-valid: 200
	// {
	//     "name": "Niko Bellic",
	//     "email": "niko@spicule.co.uk",
	//     "cell": "+123456789",
	//     "data_collection": "false"
	// }
}
