package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/wstestlib"
)

// Using a known user & role:
// User is: test@pixlise.org
//const knownTestUserID = "auth0|5f45d7b8b5abff006d4fdb91"

// Role is: "No Permissions"
//const knownTestRoleID = "rol_KdjHrTCteclbY7om"

func testUserManagement(apiHost string) {
	nonPermissionedUserId := testUserManagementPermission(apiHost)
	testUserManagementFunctionality(apiHost, nonPermissionedUserId)
}

func testUserManagementFunctionality(apiHost string, userIdToEdit string) {
	// User 2 has access
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	const knownRoleNoPermissions = "v3-No Permissions"
	const knownRoleUnassignedUser = "Unassigned New User"

	u2.AddSendReqAction("List all roles",
		`{"userRoleListReq":{}}`,
		fmt.Sprintf(`{"msgId":1, "status": "WS_OK",
			"userRoleListResp":{
				"roles${LIST,MODE=CONTAINS,MINLENGTH=2}": [
					{
						"id": "${IDSAVE=noPermissionRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					},
					{
						"id": "${IDSAVE=unassignedRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					}
				]
			}}`, knownRoleNoPermissions, knownRoleUnassignedUser),
	)

	// Allow long ago login, we might be caching JWTs to run test
	u2.AddSendReqAction("List all users",
		`{"userListReq":{}}`,
		`{"msgId":2, "status": "WS_OK",
			"userListResp": {
				"details${LIST,MODE=CONTAINS,MINLENGTH=1}": [
					{
						"auth0User": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}",
							"iconURL": "${REGEXMATCH=^https://.*}"
						},
						"pixliseUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAFTER=1688083200}",
						"lastLoginUnixSec": "${SECAGO=180}"
					}
				]
			}
		}`,
	)

	u2.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Invalid list roles for a user",
		`{"userRolesListReq":{}}`,
		`{"msgId":3, 
			"status": "WS_BAD_REQUEST",
			"errorText": "UserId is too short",
			"userRolesListResp": {} }`,
	)

	u2.AddSendReqAction("List roles for a non-existant user",
		`{"userRolesListReq":{"userId": "auth0|non-existant-user-id-999"}}`,
		`{"msgId":4, 
			"status": "WS_NOT_FOUND",
			"errorText": "404 Not Found: The user does not exist.",
			"userRolesListResp": {} }`,
	)

	u2.AddSendReqAction("List roles for a user",
		fmt.Sprintf(`{"userRolesListReq":{"userId": "%v"}}`, userIdToEdit), //u2.GetUserId()),
		fmt.Sprintf(`{"msgId":5, "status": "WS_OK",
			"userRolesListResp": {
				"roles": [
					{
						"id": "${IDCHK=unassignedRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					}
				]
			}
		}`, knownRoleUnassignedUser),
	)

	u2.AddSendReqAction("Add role to user",
		fmt.Sprintf(`{"userAddRoleReq":{"userId": "%v", "roleId": "${IDLOAD=noPermissionRoleId}"}}`, userIdToEdit), //u2.GetUserId()),
		`{"msgId":6, "status": "WS_OK",
			"userAddRoleResp": {}
		}`,
	)

	u2.AddSendReqAction("List roles for edited user",
		fmt.Sprintf(`{"userRolesListReq":{"userId": "%v"}}`, userIdToEdit), //u2.GetUserId()),
		fmt.Sprintf(`{"msgId":7, "status": "WS_OK",
			"userRolesListResp": {
				"roles${LIST,MODE=CONTAINS}": [
					{
						"id": "${IDCHK=unassignedRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					},
					{
						"id": "${IDCHK=noPermissionRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					}
				]
			}
		}`, knownRoleUnassignedUser, knownRoleNoPermissions),
	)

	u2.AddSendReqAction("Delete role from user",
		fmt.Sprintf(`{"userDeleteRoleReq":{"userId": "%v", "roleId": "${IDLOAD=noPermissionRoleId}"}}`, userIdToEdit), //u2.GetUserId()),
		`{"msgId":8, "status": "WS_OK",
			"userDeleteRoleResp": {}
		}`,
	)

	u2.AddSendReqAction("List roles for a user",
		fmt.Sprintf(`{"userRolesListReq":{"userId": "%v"}}`, userIdToEdit), //u2.GetUserId()),
		fmt.Sprintf(`{"msgId":9, "status": "WS_OK",
			"userRolesListResp": {
				"roles": [
					{
						"id": "${IDCHK=unassignedRoleId}",
						"name": "%v",
						"description": "${IGNORE}"
					}
				]
			}
		}`, knownRoleUnassignedUser),
	)

	u2.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)
}

// Return the users id used for this test...
func testUserManagementPermission(apiHost string) string {
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

	return u1.GetUserId()
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
