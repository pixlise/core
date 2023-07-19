package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

const userGroupWaitTime = 60000

func testUserGroups(apiHost string) {
	// Seed the user DB with some user info for user ids we'll be adding as part of this test
	addDBUsers(&protos.UserDBItem{
		Id: "user-abc123",
		Info: &protos.UserInfo{
			Id:    "user-abc123",
			Name:  "User ABC 123",
			Email: "user@abc123.com",
		},
	})

	// Check that if a user doesn't have permissions, they get errors
	u1 := testUserGroupsNoPermission(apiHost)

	// User group creation, rename, list, get
	u2 := testUserGroupCreation(apiHost)

	// Test that non-admin users can't add/delete members, viewers or admins to the
	// user group
	testUserGroupAddDeleteMembersAdminsViewersNoPerm(u1)

	// Request from non-admin user to be added to group
	testUserGroupJoin(u1)

	// Check join requests, add user as member of group, verify join request removed
	testAddRemoveUserAsGroupMember(u2, u1.GetUserId())

	// Test adding admins to the group by admin authorised user
	testUserGroupAdminAdd(u2, u1.GetUserId())

	// Test that the user who was added to the group as admin can now see it
	testUserCanSeeGroup(u1)

	// Test adding and deleting another admin
	testUserGroupsAddDeleteAdmin(u2, u1.GetUserId())

	// Testing that the newly added user has admin rights now to edit the admins list
	testUserCanEditGroup(u1)

	// Add a member with u2 too to test that admins can do it
	testUserGroupAdminAddAdmin(u2, u1.GetUserId())

	// Finally, delete the group
	testUserGroupAdminDeleteGroup(u2)
}

func testUserGroupCreation(apiHost string) wstestlib.ScriptedTestUser {
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
				"info": {
					"id": "${IDSAVE=createdGroupId}",
					"name": "M2020",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":4,"status":"WS_OK","userGroupListResp":{
			"groupInfos": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020",
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`,
	)

	u2.AddSendReqAction("Rename user group",
		`{"userGroupSetNameReq":{"name": "M2020 Scientists", "groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":5,"status":"WS_OK","userGroupSetNameResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":6,"status":"WS_OK","userGroupListResp":{
			"groupInfos": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
	return u2
}

func testUserGroupAddDeleteMembersAdminsViewersNoPerm(u1NonAdmin wstestlib.ScriptedTestUser) {
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

	u1NonAdmin.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1NonAdmin)
}

func testUserGroupJoin(u1NonAdmin wstestlib.ScriptedTestUser) {
	u1NonAdmin.AddSendReqAction("Check non-admin user cant view join requests for group",
		`{"userGroupJoinListReq":{"groupId": "non-existant"}}`,
		`{"msgId":9,
			"status":"WS_NOT_FOUND",
			"errorText": "non-existant not found",
			"userGroupJoinListResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Check non-admin user cant view join requests for group",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":10,
			"status":"WS_NO_PERMISSION",
			"errorText": "Not allowed to edit user group",
			"userGroupJoinListResp":{}}`,
	)

	// User should not have permissions to view join requests...
	u1NonAdmin.AddSendReqAction("Request to join non-existant group id",
		`{"userGroupJoinReq":{"groupId": "non-existant"}}`,
		`{"msgId":11,
			"status":"WS_NOT_FOUND",
			"errorText": "non-existant not found",
			"userGroupJoinResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Request to join created group id",
		`{"userGroupJoinReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":12,"status":"WS_OK","userGroupJoinResp":{}}`,
	)

	u1NonAdmin.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1NonAdmin)
}

func testAddRemoveUserAsGroupMember(u2 wstestlib.ScriptedTestUser, nonAdminUserId string) {
	u2.AddSendReqAction("Check for join requests",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":7,
			"status":"WS_OK",
			"userGroupJoinListResp":{
				"requests": [
				{
					"id": "${IDSAVE=createdJoinReqId}",
					"userId": "%v",
					"joinGroupId": "${IDCHK=createdGroupId}",
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Add user1 as member of group",
		fmt.Sprintf(`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":8,
			"status": "WS_OK",
				"userGroupAddMemberResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"	
						},
						"viewers": {},
						"members": { "users": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}] }
					}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Check for join requests again to see if cleared",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":9,
			"status":"WS_OK",
			"userGroupJoinListResp":{}
		}`,
	)

	u2.AddSendReqAction("Delete user as member",
		fmt.Sprintf(`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "%v"}}`, nonAdminUserId),
		`{"msgId":10,
			"status": "WS_OK",
			"userGroupDeleteMemberResp":{
				"group": {
					"info": {
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}"
					},
					"viewers": {},
					"members": {}
				}
			}
		}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupAdminAdd(u2 wstestlib.ScriptedTestUser, nonAdminUserId string) {
	// Edits by admin of group
	u2.AddSendReqAction("Add admin user to bad group id",
		`{"userGroupAddAdminReq":{"groupId": "way-too-long-group-id", "adminUserId": "u123"}}`,
		`{"msgId":11,"status":"WS_BAD_REQUEST","errorText": "GroupId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add bad admin user id to group id",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "admin-user-id-that-is-way-too-long even-for-auth0"}}`,
		`{"msgId":12,"status":"WS_BAD_REQUEST","errorText": "AdminUserId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":13, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":14, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to created group",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":15, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserCanSeeGroup(u1NonAdmin wstestlib.ScriptedTestUser) {
	// Check using the other user that they now can list and see this group
	u1NonAdmin.AddSendReqAction("List user groups for non-admin user",
		`{"userGroupListReq":{}}`,
		`{"msgId":13,"status":"WS_OK","userGroupListResp":{
			"groupInfos": [{
				"id": "${IDCHK=createdGroupId}",
				"name": "M2020 Scientists",
				"createdUnixSec": "${SECAGO=5}"
			}]
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Get user group for non-admin user",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":14,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	u1NonAdmin.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1NonAdmin)
}

func testUserGroupsAddDeleteAdmin(u2 wstestlib.ScriptedTestUser, nonAdminUserId string) {
	u2.AddSendReqAction("Add another admin user to created group",
		`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "123"}}`,
		fmt.Sprintf(`{"msgId":16, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"}
				]
			}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete test admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "123"}}`,
		fmt.Sprintf(`{"msgId":17, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete non-existant admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "non-existant"}}`,
		`{"msgId":18, "status": "WS_BAD_REQUEST",
			"errorText": "non-existant is not an admin","userGroupDeleteAdminResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":19,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserCanEditGroup(u1NonAdmin wstestlib.ScriptedTestUser) {
	nonAdminUserId := u1NonAdmin.GetUserId()

	u1NonAdmin.AddSendReqAction("Add another admin user from the user that was just added as an admin",
		`{"userGroupAddAdminReq":{
			"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"
		}}`,
		fmt.Sprintf(`{"msgId":15, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "user1-added-admin"}
				]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":16,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "user1-added-admin"}
				]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete test admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"}}`,
		fmt.Sprintf(`{"msgId":17, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Get user group for non-admin user again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":18,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	// Test adding members (user ids and group ids)
	u1NonAdmin.AddSendReqAction("Add member group to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		fmt.Sprintf(`{"msgId":19,
			"status": "WS_OK",
				"userGroupAddMemberResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"	
						},
						"viewers": {},
						"members": { "groups": [{"id": "group-abc123"}] },
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add member group to user group again",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":20,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-abc123 is already a members.GroupId",
			"userGroupAddMemberResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Add member user to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		fmt.Sprintf(`{"msgId":21,
			"status": "WS_OK",
			"userGroupAddMemberResp":{
				"group": 
				{
					"info": {
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}"
					},
					"viewers": {},
					"members": { "groups": [{"id": "group-abc123"}], "users": [{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }] },
					"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add member user to user group again",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":22,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-abc123 is already a members.UserId",
			"userGroupAddMemberResp":{
		}}`,
	)

	// Test adding viewers (user ids and group ids)
	u1NonAdmin.AddSendReqAction("Add viewer group to user group",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		fmt.Sprintf(`{"msgId":23,
			"status": "WS_OK",
				"userGroupAddViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "groups": [{"id": "group-viewabc123"}] },
						"members": { "groups": [{"id": "group-abc123"}], "users": [{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }] },
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add viewer group to user group again",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":24,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-viewabc123 is already a viewers.GroupId",
			"userGroupAddViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Add viewer user to user group",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		fmt.Sprintf(`{"msgId":25,
			"status": "WS_OK",
			"userGroupAddViewerResp":{
				"group": 
				{
					"info": {
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}"
					},
					"viewers": { "groups": [{"id": "group-viewabc123"}], "users": [{"id": "user-viewerabc123"}] },
					"members": { "groups": [{"id": "group-abc123"}], "users": [{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }] },
					"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Add viewer user to user group again",
		`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":26,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-viewerabc123 is already a viewers.UserId",
			"userGroupAddViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Get user group for non-admin user again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":27,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": { "groups": [{"id": "group-viewabc123"}], "users": [{"id": "user-viewerabc123"}] },
				"members": { "groups": [{"id": "group-abc123"}], "users": [{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }] },
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		fmt.Sprintf(`{"msgId":28,
			"status": "WS_OK",
				"userGroupDeleteMemberResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "groups": [{"id": "group-viewabc123"}], "users": [{"id": "user-viewerabc123"}] },
						"members": { "users": [{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }] },
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		fmt.Sprintf(`{"msgId":29,
			"status": "WS_OK",
				"userGroupDeleteMemberResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "groups": [{"id": "group-viewabc123"}], "users": [{"id": "user-viewerabc123"}] },
						"members": {},
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		fmt.Sprintf(`{"msgId":30,
			"status": "WS_OK",
				"userGroupDeleteViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "users": [{"id": "user-viewerabc123"}] },
						"members": {},
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		fmt.Sprintf(`{"msgId":31,
			"status": "WS_OK",
				"userGroupDeleteViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": {},
						"members": {},
						"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":32,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-viewerabc123 is not a viewers.UserId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":33,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-viewabc123 is not a viewers.GroupId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":34,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-abc123 is not a members.UserId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":35,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-abc123 is not a members.GroupId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	// Ensure this group admin still cant delete the group
	u1NonAdmin.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":36,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteReq not allowed",
			"userGroupDeleteResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Get user groups for non-admin user again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":37,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId),
	)

	u1NonAdmin.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1NonAdmin)
}

func testUserGroupAdminAddAdmin(u2 wstestlib.ScriptedTestUser, nonAdminUserId string) {
	u2.AddSendReqAction("Add member user to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc999"}}`,
		fmt.Sprintf(`{"msgId":20,
			"status": "WS_OK",
			"userGroupAddMemberResp":{
				"group": 
				{
					"info": {
						"id": "${IDCHK=createdGroupId}",
						"name": "M2020 Scientists",
						"createdUnixSec": "${SECAGO=5}"
					},
					"viewers": {},
					"members": { "users": [{"id": "user-abc999"}] },
					"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupAdminDeleteGroup(u2 wstestlib.ScriptedTestUser) {
	// Finally, delete the group
	u2.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":21,
			"status": "WS_OK",
			"userGroupDeleteResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":22,"status":"WS_OK","userGroupListResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupsNoPermission(apiHost string) wstestlib.ScriptedTestUser {
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

	u1.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1)

	return u1
}

func addDBUsers(user *protos.UserDBItem) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.UsersName)
	ctx := context.TODO()
	/* We DON'T drop the table!!
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}*/

	_ /*result*/, err := coll.InsertOne(ctx, user)
	if err != nil {
		log.Fatalln(err)
	}
}
