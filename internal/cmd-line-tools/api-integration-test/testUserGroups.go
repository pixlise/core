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

	// Testing that the summary list of joinable groups is correct
	testJoinableUserGroupSummaryList(apiHost)

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

	// Testing admin user ignoring join request
	testAdminUserIgnoreGroupJoin(apiHost)

	// Testing user group viewer leave group
	testUserGroupViewerLeavingGroup(apiHost)

	// Testing user group add member checks viewer entry already, and vice-versa
	testUserGroupAccessDemotion(apiHost)
	testUserGroupAccessPromotionAndDuplicateGroupName(apiHost)

	// Testing that notifications are sent out to group admins for join requests
	// Both live (via Upd message) and after admin user connects and requests notifications
	testJoinRequestNotificationLive(apiHost)
	testJoinRequestNotificationAfterConnect(apiHost)
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
					"createdUnixSec": "${SECAGO=5}",
					"lastUserJoinedUnixSec": "${SECAGO=5}"
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
					"createdUnixSec": "${SECAGO=5}",
					"lastUserJoinedUnixSec": "${SECAGO=5}"
				}
			]
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Create valid user group",
		`{"userGroupCreateReq":{"name": "GroupWithSubGroups"}}`,
		`{"msgId":7,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupWithSubGroupsId}",
					"name": "GroupWithSubGroups",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Add createdGroupId group as sub-group member of group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupWithSubGroupsId}", "groupMemberId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":8,
			"status": "WS_OK",
				"userGroupAddMemberResp":{
					"group": {
						"info": {
							"id": "${IDCHK=createdGroupWithSubGroupsId}",
							"name": "GroupWithSubGroups",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": {},
						"members": {
							"groups": [
								{
									"id": "${IDCHK=createdGroupId}",
									"name": "M2020 Scientists",
									"createdUnixSec": "${SECAGO=5}"
								}
							]
						}
					}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Fetch user group to verify subgroups",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupWithSubGroupsId}"}}`,
		`{"msgId":9,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupWithSubGroupsId}",
					"name": "GroupWithSubGroups",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {
					"groups": [
						{
							"id": "${IDCHK=createdGroupId}",
							"name": "M2020 Scientists",
							"createdUnixSec": "${SECAGO=5}"
						}
					]
				}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Delete user group (expect success)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupWithSubGroupsId}"}}`,
		`{"msgId":10,
			"status": "WS_OK",
			"userGroupDeleteResp":{}}`,
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
		fmt.Sprintf(`{"msgId":11,
			"status":"WS_OK",
			"userGroupJoinListResp":{
				"requests": [
				{
					"id": "${IDSAVE=createdJoinReqId}",
					"userId": "%v",
					"joinGroupId": "${IDCHK=createdGroupId}",
					"details": {
						"id": "%v",
						"name": "Test 1 User",
						"email": "test1@pixlise.org"
					},
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`, nonAdminUserId, nonAdminUserId),
	)

	u2.AddSendReqAction("Add user1 as member of group",
		fmt.Sprintf(`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":12,
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
		`{"msgId":13,
			"status":"WS_OK",
			"userGroupJoinListResp":{}
		}`,
	)

	u2.AddSendReqAction("Delete user as member",
		fmt.Sprintf(`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "%v"}}`, nonAdminUserId),
		`{"msgId":14,
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
		`{"msgId":15,"status":"WS_BAD_REQUEST","errorText": "GroupId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add bad admin user id to group id",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "admin-user-id-that-is-way-too-long even-for-auth0"}}`,
		`{"msgId":16,"status":"WS_BAD_REQUEST","errorText": "AdminUserId is too long","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":17, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to non-existant group",
		`{"userGroupAddAdminReq":{"groupId": "non-existant", "adminUserId": "u123"}}`,
		`{"msgId":18, "status": "WS_NOT_FOUND",
			"errorText": "non-existant not found","userGroupAddAdminResp":{}}`,
	)

	u2.AddSendReqAction("Add admin user to created group",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":19, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": { "users": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}] },
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId, nonAdminUserId),
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
				"createdUnixSec": "${SECAGO=5}",
				"lastUserJoinedUnixSec": "${SECAGO=5}",
				"relationshipToUser": "UGR_ADMIN"
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
				"members": { "users": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}] },
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
		fmt.Sprintf(`{"msgId":20, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"}
				]},
				"adminUsers": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"}
				]
			}
		}}`, nonAdminUserId, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete test admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "123"}}`,
		fmt.Sprintf(`{"msgId":21, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"}
				]},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId, nonAdminUserId),
	)

	u2.AddSendReqAction("Delete non-existant admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "non-existant"}}`,
		`{"msgId":22, "status": "WS_BAD_REQUEST",
			"errorText": "non-existant is not an admin","userGroupDeleteAdminResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		fmt.Sprintf(`{"msgId":23,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"}
				]},
				"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`, nonAdminUserId, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserCanEditGroup(u1NonAdmin wstestlib.ScriptedTestUser) {
	u1NonAdmin.AddSendReqAction("Add another admin user from the user that was just added as an admin",
		`{"userGroupAddAdminReq":{
			"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"
		}}`,
		`{"msgId":15, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"},
					{"id": "user1-added-admin"}
				]},
				"adminUsers": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "user1-added-admin"}
				]
			}
		}}`,
	)

	u1NonAdmin.AddSendReqAction("List user groups for non-admin user",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":16,"status":"WS_OK","userGroupResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"},
					{"id": "user1-added-admin"}
				]},
				"adminUsers": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "user1-added-admin"}
				]
			}
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete test admin user from created group",
		`{"userGroupDeleteAdminReq":{"groupId": "${IDLOAD=createdGroupId}", "adminUserId": "user1-added-admin"}}`,
		`{"msgId":17, "status": "WS_OK","userGroupDeleteAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"},
					{"id": "user1-added-admin"}
				]},
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Get user group for non-admin user again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":18,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {"users": [
					{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
					{"id": "123"},
					{"id": "user1-added-admin"}
				]},
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	// Test adding members (user ids and group ids)
	u1NonAdmin.AddSendReqAction("Add member group to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":19,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "123"},
							{"id": "user1-added-admin"}
						],
						"groups": [{"id": "group-abc123"}]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
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
		`{"msgId":21,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "123"},
							{"id": "user1-added-admin"},
							{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }
						],
						"groups": [{"id": "group-abc123"}]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`,
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
		`{"msgId":23,
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
						"members": {
							"users": [
								{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
								{"id": "123"},
								{"id": "user1-added-admin"},
								{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }
							],
							"groups": [{"id": "group-abc123"}]
						},
						"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
					}
		}}`,
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
		`{"msgId":25,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "123"},
							{"id": "user1-added-admin"},
							{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }
						],
						"groups": [{"id": "group-abc123"}]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`,
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
		`{"msgId":27,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": { "groups": [{"id": "group-viewabc123"}], "users": [{"id": "user-viewerabc123"}] },
				"members": {
					"users": [
						{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
						{"id": "123"},
						{"id": "user1-added-admin"},
						{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }
					],
					"groups": [{"id": "group-abc123"}]
				},
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":28,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "123"},
							{"id": "user1-added-admin"},
							{"id": "user-abc123", "name": "User ABC 123", "email": "user@abc123.com" }
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":29,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "123"},
							{"id": "user1-added-admin"}
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete auto-added member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "123"}}`,
		`{"msgId":30,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "user1-added-admin"}
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete auto-added member 2 user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user1-added-admin"}}`,
		`{"msgId":31,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":32,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":33,
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
					"members": {
						"users": [
							{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}
						]
					},
					"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
			}
		}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer user from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "userViewerId": "user-viewerabc123"}}`,
		`{"msgId":34,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-viewerabc123 is not a viewers.UserId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant viewer group from user group",
		`{"userGroupDeleteViewerReq":{"groupId": "${IDLOAD=createdGroupId}", "groupViewerId": "group-viewabc123"}}`,
		`{"msgId":35,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-viewabc123 is not a viewers.GroupId",
			"userGroupDeleteViewerResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member user from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc123"}}`,
		`{"msgId":36,
			"status": "WS_BAD_REQUEST",
			"errorText": "user-abc123 is not a members.UserId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	u1NonAdmin.AddSendReqAction("Delete non-existant member group from user group",
		`{"userGroupDeleteMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "groupMemberId": "group-abc123"}}`,
		`{"msgId":37,
			"status": "WS_BAD_REQUEST",
			"errorText": "group-abc123 is not a members.GroupId",
			"userGroupDeleteMemberResp":{
		}}`,
	)

	// Ensure this group admin still cant delete the group
	u1NonAdmin.AddSendReqAction("Delete user group (no perm)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":38,
			"status": "WS_NO_PERMISSION",
			"errorText": "UserGroupDeleteReq not allowed",
			"userGroupDeleteResp":{}}`,
	)

	u1NonAdmin.AddSendReqAction("Get user groups for non-admin user again",
		`{"userGroupReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":39,"status":"WS_OK","userGroupResp":{
			"group":{
				"info": {
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {
					"users": [
						{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}
					]
				},
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	u1NonAdmin.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1NonAdmin)
}

func testUserGroupAdminAddAdmin(u2 wstestlib.ScriptedTestUser, nonAdminUserId string) {
	u2.AddSendReqAction("Add member user to user group",
		`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId}", "userMemberId": "user-abc999"}}`,
		fmt.Sprintf(`{"msgId":24,
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
					"members": {
						"users": [
							{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"},
							{"id": "user-abc999"}
						]
					},
					"adminUsers": [{"id": "%v", "name": "${IGNORE}", "email": "${IGNORE}"}]
				}
		}}`, nonAdminUserId, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupAdminDeleteGroup(u2 wstestlib.ScriptedTestUser) {
	// Add the group as a viewer of something...
	u2.AddSendReqAction("Create element set so we can reference group",
		`{"elementSetWriteReq": {
			"elementSet": {
				"name": "User2 ElementSet",
				"lines": [
					{
						"Z":   14,
						"M":   true
					}
				]
			}
		}}`,
		`{"msgId":25, "status":"WS_OK", "elementSetWriteResp":{
			"elementSet":{
				"id":"${IDSAVE=u2CreatedElementSetId}",
				"name":"User2 ElementSet",
				"lines":[{"Z":14, "M":true}],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Add created group as viewer of an object",
		`{"objectEditAccessReq": { "objectId": "${IDLOAD=u2CreatedElementSetId}", "objectType": 2, "addViewers": { "groupIds": [ "${IDLOAD=createdGroupId}" ] }}}`,
		`{"msgId":26,"status":"WS_OK",
			"objectEditAccessResp": {
				"ownership": {
					"id": "${IDCHK=u2CreatedElementSetId}",
					"objectType": "OT_ELEMENT_SET",
					"creatorUserId": "${USERID}",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {
						"groupIds": [
							"${IDCHK=createdGroupId}"
						]
					},
					"editors": {
						"userIds": [
							"${USERID}"
						]
					}
				}
			}
		}`,
	)

	u2.AddSendReqAction("Delete user group (should fail due to reference)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":27,
			"status": "WS_SERVER_ERROR",
			"errorText": "Cannot delete user group because it is a member/viewer of 1 items",
			"userGroupDeleteResp":{}}`,
	)

	//
	u2.AddSendReqAction("Remove the group as a viewer of object",
		`{"objectEditAccessReq": { "objectId": "${IDLOAD=u2CreatedElementSetId}", "objectType": 2, "deleteViewers": { "groupIds": [ "${IDLOAD=createdGroupId}" ] }}}`,
		`{"msgId":28,"status":"WS_OK",
			"objectEditAccessResp": {
				"ownership": {
					"id": "${IDCHK=u2CreatedElementSetId}",
					"objectType": "OT_ELEMENT_SET",
					"creatorUserId": "${USERID}",
					"createdUnixSec": "${SECAGO=5}",
					"viewers": {},
					"editors": {
						"userIds": [
							"${USERID}"
						]
					}
				}
			}
		}`,
	)

	u2.AddSendReqAction("Delete user group (expect success)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=createdGroupId}"}}`,
		`{"msgId":29,
			"status": "WS_OK",
			"userGroupDeleteResp":{}}`,
	)

	u2.AddSendReqAction("List user groups again",
		`{"userGroupListReq":{}}`,
		`{"msgId":30,"status":"WS_OK","userGroupListResp":{}}`,
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

func testAdminUserIgnoreGroupJoin(apiHost string) {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 2",
		`{"userGroupCreateReq":{"name": "M2020 Users"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId2}",
					"name": "M2020 Users",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Request to join created group 2 id",
		`{"userGroupJoinReq":{"groupId": "${IDLOAD=createdGroupId2}"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupJoinResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1)

	u2.AddSendReqAction("Ensure admin got join request",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId2}"}}`,
		fmt.Sprintf(`{"msgId":2,
			"status":"WS_OK",
			"userGroupJoinListResp":{
				"requests": [
				{
					"id": "${IDSAVE=createdJoinReqId2}",
					"userId": "%v",
					"joinGroupId": "${IDCHK=createdGroupId2}",
					"details": {
						"id": "%v",
						"name": "Test 1 User",
						"email": "test1@pixlise.org"
					},
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`, u1.GetUserId(), u1.GetUserId()),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Request to join created group 2 id",
		`{"userGroupIgnoreJoinReq":{"groupId": "${IDLOAD=createdGroupId2}", "requestId": "${IDLOAD=createdJoinReqId2}"}}`,
		`{"msgId":3,"status":"WS_OK","userGroupIgnoreJoinResp":{}}`,
	)

	u2.AddSendReqAction("Ensure admin has no join requests",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId2}"}}`,
		`{"msgId":4,
			"status":"WS_OK",
			"userGroupJoinListResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testJoinableUserGroupSummaryList(apiHost string) {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 3",
		`{"userGroupCreateReq":{"name": "M2020", "description": "test"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=joinableGroupId1}",
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

	u2.AddSendReqAction("Request list of joinable user groups",
		`{"userGroupListJoinableReq":{}}`,
		`{"msgId":2,"status":"WS_OK","userGroupListJoinableResp":{
			"groups": [
				{
					"id": "${IDCHK=createdGroupId}",
					"name": "M2020 Scientists",
					"lastUserJoinedUnixSec": "${SECAGO=5}"
				},
				{
					"id": "${IDCHK=joinableGroupId1}",
					"name": "M2020",
					"description": "test",
					"lastUserJoinedUnixSec": "${SECAGO=5}"
				}
			]
		}}`,
	)

	u2.AddSendReqAction("Delete user group (expect success)",
		`{"userGroupDeleteReq":{"groupId": "${IDLOAD=joinableGroupId1}"}}`,
		`{"msgId":3,
			"status": "WS_OK",
			"userGroupDeleteResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupViewerLeavingGroup(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.CloseActionGroup([]string{}, 10)
	wstestlib.ExecQueuedActions(&u1)

	nonAdminUserId := u1.GetUserId()

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 3",
		`{"userGroupCreateReq":{"name": "PIXL Scientists"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId3}",
					"name": "PIXL Scientists",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Add non-admin user to this group
	u2.AddSendReqAction("Add non-admin user as viewer of group",
		fmt.Sprintf(`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId3}", "userViewerId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":2,
			"status": "WS_OK",
				"userGroupAddViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId3}",
							"name": "PIXL Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "users": [{"id": "%v", "name":"${REGEXMATCH=Test}","email":"${REGEXMATCH=.+@pixlise\\.org}"}] },
						"members": {}
					}
		}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u1.AddSendReqAction("Group viewer leaves group",
		fmt.Sprintf(`{"userGroupDeleteViewerReq":{
			"groupId": "${IDLOAD=createdGroupId3}",
			"userViewerId": "%v"
		}}`, nonAdminUserId /*Can't use: u1.GetUserId() - not connected yet*/),
		`{"msgId":1,
			"status": "WS_OK",
				"userGroupDeleteViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId3}",
							"name": "PIXL Scientists",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": {},
						"members": {}
					}
		}}`,
	)

	u1.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1)
}

func testUserGroupAccessDemotion(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.CloseActionGroup([]string{}, 10)
	wstestlib.ExecQueuedActions(&u1)

	nonAdminUserId := u1.GetUserId()

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 4",
		`{"userGroupCreateReq":{"name": "PIXL Sci"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId4}",
					"name": "PIXL Sci",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Add non-admin user to this group
	u2.AddSendReqAction("Add non-admin user as viewer of group",
		fmt.Sprintf(`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId4}", "userViewerId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":2,
			"status": "WS_OK",
				"userGroupAddViewerResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId4}",
							"name": "PIXL Sci",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": { "users": [{"id": "%v", "name":"${REGEXMATCH=Test}","email":"${REGEXMATCH=.+@pixlise\\.org}"}] },
						"members": {}
					}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Add non-admin user as member of group  (fail cos already viewer)",
		fmt.Sprintf(`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId4}", "userMemberId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":3,
			"status": "WS_BAD_REQUEST",
			"errorText": "%v is already a viewers.UserId",
			"userGroupAddMemberResp": {}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testUserGroupAccessPromotionAndDuplicateGroupName(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.CloseActionGroup([]string{}, 10)
	wstestlib.ExecQueuedActions(&u1)

	nonAdminUserId := u1.GetUserId()

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 7",
		`{"userGroupCreateReq":{"name": "PIXL Sci2"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId7}",
					"name": "PIXL Sci2",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	u2.AddSendReqAction("Create valid user group 7 (Fail to create duplicate name!)",
		`{"userGroupCreateReq":{"name": "PIXL Sci2"}}`,
		`{"msgId":2,
			"status": "WS_BAD_REQUEST",
			"errorText": "Name: \"PIXL Sci2\" already exists",
			"userGroupCreateResp": {}
		}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Add non-admin user to this group
	u2.AddSendReqAction("Add non-admin user as member of group",
		fmt.Sprintf(`{"userGroupAddMemberReq":{"groupId": "${IDLOAD=createdGroupId7}", "userMemberId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":3,
			"status": "WS_OK",
				"userGroupAddMemberResp":{
					"group": 
					{
						"info": {
							"id": "${IDCHK=createdGroupId7}",
							"name": "PIXL Sci2",
							"createdUnixSec": "${SECAGO=5}"
						},
						"viewers": {},
						"members": { "users": [{"id": "%v", "name":"${REGEXMATCH=Test}","email":"${REGEXMATCH=.+@pixlise\\.org}"}] }
					}
		}}`, nonAdminUserId),
	)

	u2.AddSendReqAction("Add non-admin user as viewer of group  (fail cos already member)",
		fmt.Sprintf(`{"userGroupAddViewerReq":{"groupId": "${IDLOAD=createdGroupId7}", "userViewerId": "%v"}}`, nonAdminUserId),
		fmt.Sprintf(`{"msgId":4,
			"status": "WS_BAD_REQUEST",
			"errorText": "%v is already a members.UserId",
			"userGroupAddViewerResp": {}}`, nonAdminUserId),
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testJoinRequestNotificationLive(apiHost string) {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 6",
		`{"userGroupCreateReq":{"name": "M2020 Science"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId6}",
					"name": "M2020 Science",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	// Subscribe for notifications too
	u2.AddSendReqAction("Subscribe to notifications for u2",
		`{"userNotificationReq":{}}`,
		`{"msgId":2,"status":"WS_OK","userNotificationResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Add u2 as admin of the group to ensure they get the notification
	// TODO: they should also get it if they have the PIXLISE_ADMIN role
	// but for now that doesn't work

	u2.AddSendReqAction("Add user2 as group admin",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId6}", "adminUserId": "%v"}}`, u2.GetUserId()),
		`{"msgId":3, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId6}",
					"name": "M2020 Science",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": { "users": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}] },
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	u2.AddSleepAction("Wait after subscribe", 1000)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Request to join created group 6 id",
		`{"userGroupJoinReq":{"groupId": "${IDLOAD=createdGroupId6}"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupJoinResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1)

	u2.AddSendReqAction("Ensure admin got join request",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId6}"}}`,
		fmt.Sprintf(`{"msgId": 4,
			"status":"WS_OK",
			"userGroupJoinListResp":{
				"requests": [
				{
					"id": "${IDSAVE=createdJoinReqId6}",
					"userId": "%v",
					"joinGroupId": "${IDCHK=createdGroupId6}",
					"details": {
						"id": "%v",
						"name": "Test 1 User",
						"email": "test1@pixlise.org"
					},
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`, u1.GetUserId(), u1.GetUserId()),
	)

	// Expecting to see an update here...
	u2.CloseActionGroup([]string{
		fmt.Sprintf(`{
		"userNotificationUpd": {
			"notification": {
				"subject": "${REGEXMATCH=.+has requested to join group M2020 Science}",
				"contents": "${REGEXMATCH=You are being sent this because you are an administrator of PIXLISE user group M2020 Science.+}",
				"from": "PIXLISE API",
				"timeStampUnixSec": "${SECAGO=5}",
				"actionLink": "/user-group/join-requests",
				"meta": {
					"requestorId": "%v",
					"type": "join-group-request"
				}
			}
		}
	}`, u1.GetUserId())}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("Request to join created group 4 id",
		`{"userGroupIgnoreJoinReq":{"groupId": "${IDLOAD=createdGroupId6}", "requestId": "${IDLOAD=createdJoinReqId6}"}}`,
		`{"msgId":5,"status":"WS_OK","userGroupIgnoreJoinResp":{}}`,
	)

	u2.AddSendReqAction("Ensure admin has no join requests",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId6}"}}`,
		`{"msgId":6,
			"status":"WS_OK",
			"userGroupJoinListResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func testJoinRequestNotificationAfterConnect(apiHost string) {
	setupGroup5(apiHost)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Request to join created group 5 id",
		`{"userGroupJoinReq":{"groupId": "${IDLOAD=createdGroupId5}"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupJoinResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u1)

	// Connect user 2 again
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Ensure admin got join request",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId5}"}}`,
		fmt.Sprintf(`{"msgId": 1,
			"status":"WS_OK",
			"userGroupJoinListResp":{
				"requests": [
				{
					"id": "${IDSAVE=createdJoinReqId5}",
					"userId": "%v",
					"joinGroupId": "${IDCHK=createdGroupId5}",
					"details": {
						"id": "%v",
						"name": "Test 1 User",
						"email":"test1@pixlise.org"
					},
					"createdUnixSec": "${SECAGO=5}"
				}
			]
		}}`, u1.GetUserId(), u1.GetUserId()),
	)

	// Not expecting to see an update here...
	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Request notifications, which should deliver the notification from DB
	u2.AddSendReqAction("Subscribe to notifications for u2",
		`{"userNotificationReq":{}}`,
		`{"msgId":2,"status":"WS_OK","userNotificationResp":{}}`,
	)

	u2.AddSendReqAction("Request to join created group 5 id",
		`{"userGroupIgnoreJoinReq":{"groupId": "${IDLOAD=createdGroupId5}", "requestId": "${IDLOAD=createdJoinReqId5}"}}`,
		`{"msgId":3,"status":"WS_OK","userGroupIgnoreJoinResp":{}}`,
	)

	u2.AddSendReqAction("Ensure admin has no join requests",
		`{"userGroupJoinListReq":{"groupId": "${IDLOAD=createdGroupId5}"}}`,
		`{"msgId":4,
			"status":"WS_OK",
			"userGroupJoinListResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
}

func setupGroup5(apiHost string) {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create valid user group 5",
		`{"userGroupCreateReq":{"name": "M2020 Engineers"}}`,
		`{"msgId":1,"status":"WS_OK","userGroupCreateResp":{
			"group": {
				"info": {
					"id": "${IDSAVE=createdGroupId5}",
					"name": "M2020 Engineers",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": {}
			}
		}}`,
	)

	// In this scenario we DON'T subscribe for notifications, so it should go to DB...

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)

	// Add u2 as admin of the group to ensure they get the notification
	// TODO: they should also get it if they have the PIXLISE_ADMIN role
	// but for now that doesn't work

	u2.AddSendReqAction("Add user2 as group admin",
		fmt.Sprintf(`{"userGroupAddAdminReq":{"groupId": "${IDLOAD=createdGroupId5}", "adminUserId": "%v"}}`, u2.GetUserId()),
		`{"msgId":2, "status": "WS_OK","userGroupAddAdminResp":{
			"group": {
				"info": {
					"id": "${IDCHK=createdGroupId5}",
					"name": "M2020 Engineers",
					"createdUnixSec": "${SECAGO=5}"
				},
				"viewers": {},
				"members": { "users": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}] },
				"adminUsers": [{"id": "${USERID}", "name": "${IGNORE}", "email": "${IGNORE}"}]
			}
		}}`,
	)

	//u2.AddDisonnectAction("Disconnect after u2 becomes admin")

	u2.CloseActionGroup([]string{}, userGroupWaitTime)
	wstestlib.ExecQueuedActions(&u2)
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
