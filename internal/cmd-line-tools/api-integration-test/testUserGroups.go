package main

import (
	"fmt"

	"github.com/pixlise/core/v3/core/wstestlib"
)

func testUserGroups(apiHost string) {
	testUserGroupsPermission(apiHost)
	testUserGroupsFunctionality(apiHost)
}

func testUserGroupsFunctionality(apiHost string) {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("List user groups",
		`{"userGroupListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","userGroupListResp":{}}`,
	)

	u2.AddSendReqAction("Create invalid user group",
		`{"userGroupCreateReq":{"name": "a very inconveniently long name that shouldn't be allowed"}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST","errorText": "Name is too long","userGroupCreateResp":{}}`,
	)

	u2.AddSendReqAction("Create valid user group",
		`{"userGroupCreateReq":{"name": "M2020"}}`,
		`{"msgId":3,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"id": "$ID=createdGroupId$",
				"name": "M2020",
				"createdUnixSec": "$SECAGO=5$",
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	createdGroupId := u2.GetIdCreated("createdGroupId")

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":4,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "%v",
					"name": "M2020",
					"createdUnixSec": "$SECAGO=5$",
					"members": {}
				}
			]
		}}`, createdGroupId),
	)

	u2.AddSendReqAction("Rename user group",
		fmt.Sprintf(`{"userGroupSetNameReq":{"name": "M2020 Scientists", "groupId": "%v"}}`, createdGroupId),
		fmt.Sprintf(`{"msgId":5,"status":"WS_OK","userGroupSetNameResp":{
			"group": {
				"id": "%v",
				"name": "M2020 Scientists",
				"createdUnixSec": "$SECAGO=5$",
				"members": {}
			}
		}}`, createdGroupId),
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":6,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "%v",
					"name": "M2020 Scientists",
					"createdUnixSec": "$SECAGO=5$",
					"members": {}
				}
			]
		}}`, createdGroupId),
	)
	/*
		u1.AddSendReqAction("Add admin user",
			fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "%v", "userId": "%v"}}`, ,),
			`{"msgId":5,"status":"WS_OK","userGroupSetNameResp":{}}`,
		)

	*/

	u2.CloseActionGroup([]string{}, 50000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupsPermission(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List user groups (no perm)",
		`{"userGroupListReq":{}}`,
		`{"msgId":1,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupListReq not allowed",
			"userGroupListResp":{}}`,
	)

	u1.AddSendReqAction("Create user group (no perm)",
		`{"userGroupCreateReq":{"name": "The group"}}`,
		`{"msgId":2,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupCreateReq not allowed",
			"userGroupCreateResp":{}}`,
	)

	u1.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "123"}}`,
		`{"msgId":3,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteReq not allowed",
			"userGroupDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Rename user group (no perm)",
		`{"userGroupSetNameReq":{"groupId": "123", "name": "The new name"}}`,
		`{"msgId":4,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupSetNameReq not allowed",
			"userGroupSetNameResp":{}}`,
	)
	/* Cant do these, they request the item from DB first
	u1.AddSendReqAction("Add admin to user group (no perm)",
		`{"userGroupAddAdminReq":{"groupId": "123", "adminUserId": "abc123"}}`,
		`{"msgId":5,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupAddAdminReq not allowed",
			"userGroupAddAdminResp":{}}`,
	)

	u1.AddSendReqAction("Delete admin from user group (no perm)",
		`{"userGroupDeleteAdminReq":{"groupId": "123", "adminUserId": "abc123"}}`,
		`{"msgId":6,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteAdminReq not allowed",
			"userGroupDeleteAdminResp":{}}`,
	)

	u1.AddSendReqAction("Add member to user group (no perm)",
		`{"userGroupAddMemberReq":{"groupId": "123", "groupMemberId": "abc123"}}`,
		`{"msgId":7,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupAddMemberReq not allowed",
			"userGroupAddMemberResp":{}}`,
	)

	u1.AddSendReqAction("Delete member from user group (no perm)",
		`{"userGroupDeleteMemberReq":{"groupId": "123", "groupMemberId": "abc123"}}`,
		`{"msgId":8,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteMemberReq not allowed",
			"userGroupDeleteMemberResp":{}}`,
	)
	*/
	u1.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)
}
