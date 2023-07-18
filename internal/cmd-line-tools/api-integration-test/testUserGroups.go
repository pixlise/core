package main

import (
	"fmt"

	"github.com/pixlise/core/v3/core/wstestlib"
)

func testUserGroups(apiHost string) {
	u1 := testUserGroupsPermission(apiHost)
	testUserGroupsFunctionality(apiHost, u1)
}

func testUserGroupsFunctionality(apiHost string, u1NonAdmin wstestlib.ScriptedTestUser) {
	nonAdminUserId := u1NonAdmin.GetUserId()

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
				"id": "${IDSAVE=createdGroupId}",
				"name": "M2020",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":4,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {}
				}
			]
		}}`,
	)

	u2.AddSendReqAction("Rename user group",
		`{"userGroupSetNameReq":{"name": "M2020 Scientists", "groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":5,"status":"WS_OK","userGroupSetNameResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":6,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {}
				}
			]
		}}`,
	)

	u2.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u2)

	// Try non-admin user editing the new group
	u1NonAdmin.AddSendReqAction("Add admin to user group",
		`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "abc123"}}`,
		`{"msgId":5,
			"status": "WS_NO_PERMISSION",
			"errorText": "Not allowed to edit user group",
			"userGroupAddAdminResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete admin from user group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "abc123"}}`,
		`{"msgId":6,
			"status": "WS_NO_PERMISSION",
			"errorText": "Not allowed to edit user group",
			"userGroupDeleteAdminResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Add member to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "abc123"}}`,
		`{"msgId":7,
			"status": "WS_NO_PERMISSION",
			"errorText": "Not allowed to edit user group",
			"userGroupAddMemberResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete member from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "abc123"}}`,
		`{"msgId":8,
			"status": "WS_NO_PERMISSION",
			"errorText": "Not allowed to edit user group",
			"userGroupDeleteMemberResp":{}}`,
	)

	u1NonAdmin.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u1NonAdmin)

	// Edits by admin of group
	u2.AddSendReqAction("Add admin user to bad group id",
		`{"userGroupAddAdminReq":{"groupId": "way-too-long-group-id", "adminUserId": "u123"}}`,
		`{"msgId":7,"status":"WS_BAD_REQUEST","errorText": "GroupId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add bad admin user id to group id",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "admin-user-id-that-is-way-too-long even-for-auth0"}}`,
		`{"msgId":8,"status":"WS_BAD_REQUEST","errorText": "AdminUserId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":9, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":10, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to created group",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":11, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {},
				"adminUserIds": ["%v"]
			}
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u2)

	// Check using the other user that they now can list and see this group
	u1NonAdmin.AddSendReqAction("List user groups for non-admin user",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":9,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {},
					"adminUserIds": ["%v"]
				}
			]
		}}`, nonAdminUserId),
	)

	u1NonAdmin.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1NonAdmin)

	u2.AddSendReqAction("Add another admin user to created group",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "123"}}`),
		fmt.Sprintf(`{"msgId":12, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {},
				"adminUserIds": ["%v", "123"]
			}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete test admin user from created group",
		fmt.Sprintf(`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "123"}}`),
		fmt.Sprintf(`{"msgId":13, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {},
				"adminUserIds": ["%v"]
			}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete non-existant admin user from created group",
		fmt.Sprintf(`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "non-existant"}}`),
		`{"msgId":14, "status": "WS_BAD_REQUEST",
			"errorText": "non-existant is not an admin","userGroupDeleteAdminResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":15,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {},
					"adminUserIds": ["%v"]
				}
			]
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u2)

	// Testing that the newly added user has admin rights now to edit the admins list
	u1NonAdmin.AddSendReqAction("Add another admin user from the user that was just added as an admin",
		`{"userGroupAddAdminReq":{
			"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"
		}}`,
		fmt.Sprintf(`{"msgId":10, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {},
				"adminUserIds": ["%v", "user1-added-admin"]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":11,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {},
					"adminUserIds": ["%v", "user1-added-admin"]
				}
			]
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete test admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"}}`,
		fmt.Sprintf(`{"msgId":12, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}",
				"viewers": {},
				"members": {},
				"adminUserIds": ["%v"]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":13,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {},
					"adminUserIds": ["%v"]
				}
			]
		}}`, nonAdminUserId),
	)

	// Test adding members (user ids and group ids)
	u1NonAdmin.AddSendReqAction("Add member group to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		fmt.Sprintf(`{"msgId":14,
			"status": "WS_OK",
				"userGroupAddMemberResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": {},
						"members": { "groupIds": ["group-abc123"] },
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add member group to user group again",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":15,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-abc123 is already a members.GroupId",
			"userGroupAddMemberResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Add member user to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		fmt.Sprintf(`{"msgId":16,
			"status": "WS_OK",
			"userGroupAddMemberResp":{
				"group": 
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": { "groupIds": ["group-abc123"], "userIds": ["user-abc123"] },
					"adminUserIds": ["%v"]
				}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add member user to user group again",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":17,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-abc123 is already a members.UserId",
			"userGroupAddMemberResp":{
		}}`,
	)

	// Test adding viewers (user ids and group ids)
	u1NonAdmin.AddSendReqAction("Add viewer group to user group",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		fmt.Sprintf(`{"msgId":18,
			"status": "WS_OK",
				"userGroupAddViewerResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": { "groupIds": ["group-viewabc123"] },
						"members": { "groupIds": ["group-abc123"], "userIds": ["user-abc123"] },
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add viewer group to user group again",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":19,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-viewabc123 is already a viewers.GroupId",
			"userGroupAddViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Add viewer user to user group",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		fmt.Sprintf(`{"msgId":20,
			"status": "WS_OK",
			"userGroupAddViewerResp":{
				"group": 
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": { "groupIds": ["group-viewabc123"], "userIds": ["user-viewerabc123"] },
					"members": { "groupIds": ["group-abc123"], "userIds": ["user-abc123"] },
					"adminUserIds": ["%v"]
				}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add viewer user to user group again",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":21,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-viewerabc123 is already a viewers.UserId",
			"userGroupAddViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":22,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": { "groupIds": ["group-viewabc123"], "userIds": ["user-viewerabc123"] },
					"members": { "groupIds": ["group-abc123"], "userIds": ["user-abc123"] },
					"adminUserIds": ["%v"]
				}
			]
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		fmt.Sprintf(`{"msgId":23,
			"status": "WS_OK",
				"userGroupDeleteMemberResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": { "groupIds": ["group-viewabc123"], "userIds": ["user-viewerabc123"] },
						"members": { "userIds": ["user-abc123"] },
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		fmt.Sprintf(`{"msgId":24,
			"status": "WS_OK",
				"userGroupDeleteMemberResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": { "groupIds": ["group-viewabc123"], "userIds": ["user-viewerabc123"] },
						"members": {},
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		fmt.Sprintf(`{"msgId":25,
			"status": "WS_OK",
				"userGroupDeleteViewerResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": { "userIds": ["user-viewerabc123"] },
						"members": {},
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		fmt.Sprintf(`{"msgId":26,
			"status": "WS_OK",
				"userGroupDeleteViewerResp":{
					"group": 
					{
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}",
						"viewers": {},
						"members": {},
						"adminUserIds": ["%v"]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":27,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-viewerabc123 is not a viewers.UserId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":28,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-viewabc123 is not a viewers.GroupId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":29,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-abc123 is not a members.UserId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":30,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-abc123 is not a members.GroupId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	// Ensure this group admin still cant delete the group
	u1NonAdmin.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":31,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteReq not allowed",
			"userGroupDeleteResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user again",
		`{"userGroupListReq":{}}`,
		fmt.Sprintf(`{"msgId":32,"status":"WS_OK","userGroupListResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": {},
					"adminUserIds": ["%v"]
				}
			]
		}}`, nonAdminUserId),
	)

	u1NonAdmin.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u1NonAdmin)

	// Add a member with u2 too to test that admins can do it
	u2.AddSendReqAction("Add member user to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc999"}}`,
		fmt.Sprintf(`{"msgId":16,
			"status": "WS_OK",
			"userGroupAddMemberResp":{
				"group": 
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"members": { "userIds": ["user-abc999"] },
					"adminUserIds": ["%v"]
				}
		}}`, nonAdminUserId),
	)

	// Finally, delete the group
	u2.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":17,
			"status": "WS_OK",
			"userGroupDeleteResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":18,"status":"WS_OK","userGroupListResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, 50000)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupsPermission(apiHost string) wstestlib.ScriptedTestUser {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List user groups",
		`{"userGroupListReq":{}}`,
		`{"msgId":1,
			"status": "WS_OK",
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

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	return u1
}
