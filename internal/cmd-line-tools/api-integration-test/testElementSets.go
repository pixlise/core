package main

import (
	"fmt"

	"github.com/pixlise/core/v3/core/wstestlib"
)

func testElementSets(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect user 1", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List",
		`{"elementSetListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","elementSetListResp":{}}`,
	)

	u1.AddSendReqAction("Get non-existant id",
		`{"elementSetGetReq": { "id": "non-existant-id"}}`,
		`{"msgId":2, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetGetResp":{}}`,
	)

	u1.AddSendReqAction("Create invalid item",
		`{"elementSetWriteReq": {
			"elementSet": {
				"name": "User1 ElementSet1",
				"lines": []
			}
		}}`,
		`{"msgId":3, "status":"WS_BAD_REQUEST", "errorText": "Lines length is invalid", "elementSetWriteResp":{}}`,
	)

	u1.AddSendReqAction("Delete non-existant item",
		`{"elementSetDeleteReq": { "id": "non-existant-id" }}`,
		`{"msgId":4, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Edit non-existant item",
		`{"elementSetWriteReq": {
			"elementSet": {
				"id": "non-existant-id",
				"name": "User1 ElementSet1",
				"lines": [
					{
						"Z":   14,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					}
				]
			}
		}}`,
		`{"msgId":5, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetWriteResp":{}}`,
	)

	u1.AddSendReqAction("Create valid item",
		`{"elementSetWriteReq": {
			"elementSet": {
				"name": "User1 ElementSet1",
				"lines": [
					{
						"Z":   14,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					},
					{
						"Z":   16,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					}
				]
			}
		}}`,
		`{"msgId":6, "status":"WS_OK", "elementSetWriteResp":{
			"elementSet":{
				"id":"$ID=elem1$",
				"name":"User1 ElementSet1",
				"lines":[{"Z":14, "M":true}, {"Z":16, "M":true}],
				"owner": {
					"creatorUser": {
						"id": "$USERID$",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=3$"
				}
			}
		}}`,
	)

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)

	u1CreatedElementSetId1 := u1.GetIdCreated("elem1") // Remember the ID that was created

	// Login as another user and list items to verify none are coming back here too
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)

	u2.AddConnectAction("Connect user 2", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("List for user 2",
		`{"elementSetListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","elementSetListResp":{}}`,
	)

	// Stop here, we need the user id going forward...
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	u2.ClearActions()

	u2.AddSendReqAction("Get created item for user 2",
		fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":2, "status": "WS_NO_PERMISSION", "errorText": "View access denied for: %v", "elementSetGetResp":{}}`, u1CreatedElementSetId1),
	)

	u2.AddSendReqAction("Get permissions for user 1's created item",
		fmt.Sprintf(`{"getOwnershipReq": { "objectId": "%v", "objectType": 2 }}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":3,"status":"WS_NO_PERMISSION","errorText": "View access denied for: %v","getOwnershipResp":{}}`, u1CreatedElementSetId1),
	)

	u2.AddSendReqAction("Share user 1s created item",
		fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "%v", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u1CreatedElementSetId1, u2.GetUserId()),
		fmt.Sprintf(`{"msgId":4,"status":"WS_NO_PERMISSION","errorText": "Edit access denied for: %v","objectEditAccessResp":{}}`, u1CreatedElementSetId1),
	)

	// Verify the above
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	// Back to user 1
	u1.ClearActions()

	u1.AddSendReqAction("Get created item for user 1",
		fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":7, "status":"WS_OK", "elementSetGetResp":{
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1",
				"lines": [
					{
						"Z":   14,
						"M":   true
					},
					{
						"Z":   16,
						"M":   true
					}
				],
				"owner": {
					"creatorUser": {
						"id": "$USERID$",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=3$"
				}
			}
		}}`, u1CreatedElementSetId1),
	)
	u1.AddSendReqAction("Edit created item with invalid request",
		fmt.Sprintf(`{"elementSetWriteReq": {
			"elementSet": {
				"id": "%v",
				"name": "This name is way way too long for any element set to seriously be named this way",
				"lines": [
					{
						"Z":   17,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					}
				]
			}
		}}`, u1CreatedElementSetId1),
		`{
			"msgId": 8,
			"status": "WS_BAD_REQUEST",
			"errorText": "Name length is invalid",
			"elementSetWriteResp": {}
		}`,
	)
	u1.AddSendReqAction("Edit created item",
		fmt.Sprintf(`{"elementSetWriteReq": {
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited",
				"lines": [
					{
						"Z":   17,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					}
				]
			}
		}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":9, "status":"WS_OK", "elementSetWriteResp":{
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited",
				"lines": [
					{
						"Z":   17,
						"M":   true
					}
				],
				"owner": {
					"creatorUser": {
						"id": "$USERID$",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=3$"
				}
			}
		}}`, u1CreatedElementSetId1),
	)

	u1.AddSendReqAction("Get edited item",
		fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":10, "status":"WS_OK", "elementSetGetResp":{
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited",
				"lines": [
					{
						"Z":   17,
						"M":   true
					}
				],
				"modifedUnixSec": "$SECAGO=3$",
				"owner": {
					"creatorUser": {
						"id": "$USERID$",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=3$"
				}
			}
		}}`, u1CreatedElementSetId1),
	)

	u1.AddSendReqAction("List items",
		`{"elementSetListReq":{}}`,
		fmt.Sprintf(`{
			"msgId": 11,
			"status": "WS_OK",
			"elementSetListResp": {
				"elementSets": {
					"%v": {
						"id": "%v",
						"name": "User1 ElementSet1-Edited",
						"atomicNumbers": [
							17
						],
						"modifedUnixSec": "$SECAGO=3$",
						"owner": {
							"creatorUser": {
								"id": "$USERID$",
								"name": "$IGNORE$",
								"email": "$IGNORE$"
							},
							"createdUnixSec": "$SECAGO=3$"
						}
					}
				}
			}
		}`, u1CreatedElementSetId1, u1CreatedElementSetId1),
	)

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Test sharing by user 1
	u1.ClearActions()

	u1.AddSendReqAction("Get permissions for created item as user 1",
		fmt.Sprintf(`{"getOwnershipReq": { "objectId": "%v", "objectType": 2 }}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{
			"msgId": 12,
			"status": "WS_OK",
			"getOwnershipResp": {
				"ownership": {
					"id": "%v",
					"objectType": "OT_ELEMENT_SET",
					"creatorUserId": "$USERID$",
					"createdUnixSec": "$SECAGO=6$",
					"editors": {
						"userIds": ["%v"]
					}
				}
			}
		}`, u1CreatedElementSetId1, u1.GetUserId()),
	)

	u1.AddSendReqAction("Share created item with user 2",
		fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "%v", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u1CreatedElementSetId1, u2.GetUserId()),
		fmt.Sprintf(`{
			"msgId": 13,
			"status": "WS_OK",
			"objectEditAccessResp": {
				"ownership": {
					"id": "%v",
					"objectType": "OT_ELEMENT_SET",
					"creatorUserId": "$USERID$",
					"createdUnixSec": "$SECAGO=6$",
					"viewers": {
						"userIds": ["%v"]
					},
					"editors": {
						"userIds": ["%v"]
					}
				}
			}
		}`, u1CreatedElementSetId1, u2.GetUserId(), u1.GetUserId()),
	)

	u1.AddSendReqAction("Get shared item",
		fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":14, "status":"WS_OK", "elementSetGetResp":{
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited",
				"lines": [
					{
						"Z":   17,
						"M":   true
					}
				],
				"modifedUnixSec": "$SECAGO=6$",
				"owner": {
					"creatorUser": {
						"id": "$USERID$",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=6$"
				}
			}
		}}`, u1CreatedElementSetId1),
	)

	u1.AddSendReqAction("List items",
		`{"elementSetListReq":{}}`,
		fmt.Sprintf(`{
			"msgId": 15,
			"status": "WS_OK",
			"elementSetListResp": {
				"elementSets": {
					"%v": {
						"id": "%v",
						"name": "User1 ElementSet1-Edited",
						"atomicNumbers": [
							17
						],
						"modifedUnixSec": "$SECAGO=6$",
						"owner": {
							"creatorUser": {
								"id": "$USERID$",
								"name": "$IGNORE$",
								"email": "$IGNORE$"
							},
							"createdUnixSec": "$SECAGO=6$"
						}
					}
				}
			}
		}`, u1CreatedElementSetId1, u1CreatedElementSetId1),
	)

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Back to user 2 - we should be able to view the shared item but still not edit
	u2.ClearActions()

	u2.AddSendReqAction("List for user 2",
		`{"elementSetListReq":{}}`,
		fmt.Sprintf(`{
			"msgId": 5,
			"status": "WS_OK",
			"elementSetListResp": {
				"elementSets": {
					"%v": {
						"id": "%v",
						"name": "User1 ElementSet1-Edited",
						"atomicNumbers": [
							17
						],
						"modifedUnixSec": "$SECAGO=6$",
						"owner": {
							"creatorUser": {
								"id": "%v",
								"name": "$IGNORE$",
								"email": "$IGNORE$"
							},
							"createdUnixSec": "$SECAGO=6$"
						}
					}
				}
			}
		}`, u1CreatedElementSetId1, u1CreatedElementSetId1, u1.GetUserId()),
	)

	u2.AddSendReqAction("Get shared item",
		fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":6, "status":"WS_OK", "elementSetGetResp":{
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited",
				"lines": [
					{
						"Z":   17,
						"M":   true
					}
				],
				"modifedUnixSec": "$SECAGO=6$",
				"owner": {
					"creatorUser": {
						"id": "%v",
						"name": "$IGNORE$",
						"email": "$IGNORE$"
					},
					"createdUnixSec": "$SECAGO=6$"
				}
			}
		}}`, u1CreatedElementSetId1, u1.GetUserId()),
	)

	u2.AddSendReqAction("Edit created item, should fail, user2 is a viewer",
		fmt.Sprintf(`{"elementSetWriteReq": {
			"elementSet": {
				"id": "%v",
				"name": "User1 ElementSet1-Edited by User2",
				"lines": [
					{
						"Z":   19,
						"K":   false,
						"L":   false,
						"M":   true,
						"Esc": false
					}
				]
			}
		}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":7, "status":"WS_NO_PERMISSION", "errorText": "Edit access denied for: %v", "elementSetWriteResp":{}}`, u1CreatedElementSetId1),
	)

	u2.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u2)

	// Back to user 1 - delete the element set
	u1.ClearActions()

	u1.AddSendReqAction("Delete created item",
		fmt.Sprintf(`{"elementSetDeleteReq": { "id": "%v" }}`, u1CreatedElementSetId1),
		`{"msgId":16,"status":"WS_OK","elementSetDeleteResp":{}}`,
	)

	u1.AddSendReqAction("List",
		`{"elementSetListReq":{}}`,
		`{"msgId":17,"status":"WS_OK","elementSetListResp":{}}`,
	)

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)
}
