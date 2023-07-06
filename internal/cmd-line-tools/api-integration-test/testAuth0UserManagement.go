package main

import "github.com/pixlise/core/v3/core/wstestlib"

// Using a known user & role:
// User is: test@pixlise.org
//const knownTestUserID = "auth0|5f45d7b8b5abff006d4fdb91"

// Role is: "No Permissions"
//const knownTestRoleID = "rol_KdjHrTCteclbY7om"

func testUserManagement(apiHost string) {
	testUserManagementPermission(apiHost)
	testUserManagementFunctionality(apiHost)
}

func testUserManagementFunctionality(apiHost string) {
	// User 2 has access
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("List all roles",
		`{"userRoleListReq":{}}`,
		`{"msgId":1, "status": "WS_OK",
			"userRoleListResp":{
				"roles#LIST,MODE=LENGTH,MINLENGTH=1#": []
			}}`,
	)

	// List roles
	// List users
	// List roles for a user
	// Add role to user
	// Delete role from  user

	u2.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)
}

func testUserManagementPermission(apiHost string) {
	// User 1 doesn't have the right to do anything user management-wise, check that we do prevent it
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Try list all roles",
		`{"userRoleListReq":{}}`,
		`{"msgId":1, "status": "WS_NO_PERMISSION",
			"errorText": "UserRoleListReq not allowed","userRoleListResp":{}}`,
	)

	u1.AddSendReqAction("Try list roles for a user",
		`{"userRolesListReq":{}}`,
		`{"msgId":2, "status": "WS_NO_PERMISSION",
			"errorText": "UserRolesListReq not allowed","userRolesListResp":{}}`,
	)

	u1.AddSendReqAction("Try delete role",
		`{"userDeleteRoleReq":{}}`,
		`{"msgId":3, "status": "WS_NO_PERMISSION",
			"errorText": "UserDeleteRoleReq not allowed","userDeleteRoleResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)
}

/*

	// In case test ran before, we call delete first, then list roles, ensuring it's not in there.
	req, _ := http.NewRequest("DELETE", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-del: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Stop for a sec so we don't hit auth0 API rate limit
	time.Sleep(1 * time.Second)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("check-del: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// We then add the role, list roles, ensure it's there
	req, _ = http.NewRequest("POST", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("add: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Stop for a sec so we don't hit auth0 API rate limit
	time.Sleep(1 * time.Second)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-added: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Finally, delete role, list roles, ensure it's gone
	req, _ = http.NewRequest("DELETE", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("delete: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Stop for a sec so we don't hit auth0 API rate limit
	time.Sleep(1 * time.Second)

	req, _ = http.NewRequest("GET", "/user/roles/"+knownTestUserID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-del-2: %v\n", resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/user/roles/"+knownTestUserID+"/"+knownTestRoleID, nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("add-back: %v\n", resp.Code)
	fmt.Println(resp.Body)

	// Stop for a sec so we don't hit auth0 API rate limit
	time.Sleep(1 * time.Second)
*/

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
