package main

import (
	"fmt"

	"github.com/pixlise/core/v3/core/wstestlib"
)

func testROI(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect user 1", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List",
		`{"regionOfInterestListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","regionOfInterestListResp":{}}`,
	)

	u1.AddSendReqAction("Get non-existant id",
		`{"regionOfInterestGetReq": { "id": "non-existant-id"}}`,
		`{"msgId":2, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "regionOfInterestGetResp":{}}`,
	)

	u1.AddSendReqAction("Create invalid item (no indexes defined)",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"name": "Invalid ROI",
				"description": "Ye Invalid ROIe",
				"scanId": "048300551"
			}
		}}`,
		`{"msgId":3, "status":"WS_BAD_REQUEST", "errorText": "ROI must have location or pixel indexes defined", "regionOfInterestWriteResp":{}}`,
	)

	u1.AddSendReqAction("Delete non-existant item",
		`{"regionOfInterestDeleteReq": { "id": "non-existant-id" }}`,
		`{"msgId":4, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "regionOfInterestDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Edit non-existant item",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"id": "non-existant-id",
				"name": "Non existant ROI",
				"description": "Ye Non existant ROIe",
				"scanId": "048300551"
			}
		}}`,
		`{"msgId":5, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "regionOfInterestWriteResp":{}}`,
	)

	u1.AddSendReqAction("Create valid item",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"name": "User1 ROI1",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98]
			}
		}}`,
		`{"msgId":6, "status":"WS_OK", "regionOfInterestWriteResp":{
			"regionOfInterest":{
				"id":"${IDSAVE=u1CreatedROIId1}",
				"name": "User1 ROI1",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`,
	)

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)

	u1CreatedROIId1 := wstestlib.GetIdCreated("u1CreatedROIId1") // Remember the ID that was created

	// Login as another user and list items to verify none are coming back here too
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)

	u2.AddConnectAction("Connect user 2", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("List for user 2",
		`{"regionOfInterestListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","regionOfInterestListResp":{}}`,
	)

	// Stop here, we need the user id going forward...
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	u2.ClearActions()

	u2.AddSendReqAction("Get created item for user 2",
		`{"regionOfInterestGetReq": { "id": "${IDLOAD=u1CreatedROIId1}"}}`,
		fmt.Sprintf(`{"msgId":2, "status": "WS_NO_PERMISSION", "errorText": "View access denied for: %v", "regionOfInterestGetResp":{}}`, u1CreatedROIId1),
	)

	u2.AddSendReqAction("Get permissions for user 1's created item",
		`{"getOwnershipReq": { "objectId": "${IDLOAD=u1CreatedROIId1}", "objectType": 2 }}`,
		fmt.Sprintf(`{"msgId":3,"status":"WS_NO_PERMISSION","errorText": "View access denied for: %v","getOwnershipResp":{}}`, u1CreatedROIId1),
	)

	u2.AddSendReqAction("Share user 1s created item",
		fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "${IDLOAD=u1CreatedROIId1}", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u2.GetUserId()),
		fmt.Sprintf(`{"msgId":4,"status":"WS_NO_PERMISSION","errorText": "Edit access denied for: %v","objectEditAccessResp":{}}`, u1CreatedROIId1),
	)

	// Verify the above
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	// Back to user 1
	u1.ClearActions()

	u1.AddSendReqAction("Get created item for user 1",
		`{"regionOfInterestGetReq": { "id": "${IDLOAD=u1CreatedROIId1}"}}`,
		`{"msgId":7, "status":"WS_OK", "regionOfInterestGetResp":{
			"regionOfInterest":{
				"id":"${IDCHK=u1CreatedROIId1}",
				"name": "User1 ROI1",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`,
	)
	u1.AddSendReqAction("Edit created item with invalid request",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"id": "${IDLOAD=u1CreatedROIId1}",
				"scanId": "048300551",
				"name": "The ROI",
				"imageName": "WhatsAnImageDoingHere.png"
			}
		}}`,
		`{
			"msgId": 8,
			"status": "WS_BAD_REQUEST",
			"errorText": "ROI must have location or pixel indexes defined",
			"regionOfInterestWriteResp": {}
		}`,
	)
	u1.AddSendReqAction("Edit created item",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"id": "${IDLOAD=u1CreatedROIId1}",
				"scanId": "048300551",
				"name": "The ROI",
				"locationIndexesEncoded": [14, 123, -1, 126, 98, 88]
			}
		}}`,
		`{"msgId":9, "status":"WS_OK", "regionOfInterestWriteResp":{
			"regionOfInterest":{
				"id":"${IDCHK=u1CreatedROIId1}",
				"name": "The ROI",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98, 88],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`,
	)

	u1.AddSendReqAction("Get edited item",
		`{"regionOfInterestGetReq": { "id": "${IDLOAD=u1CreatedROIId1}"}}`,
		`{"msgId":10, "status":"WS_OK", "regionOfInterestGetResp":{
			"regionOfInterest":{
				"id":"${IDCHK=u1CreatedROIId1}",
				"name": "The ROI",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98, 88],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`,
	)

	u1.AddSendReqAction("List items",
		`{"regionOfInterestListReq":{}}`,
		`{
			"msgId": 11,
			"status": "WS_OK",
			"regionOfInterestListResp": {
				"regionsOfInterest":{
					"${IDCHK=u1CreatedROIId1}": {
						"id":"${IDCHK=u1CreatedROIId1}",
						"name": "The ROI",
						"description": "User1 ROI1",
						"scanId": "048300551",
						"modifiedUnixSec": "${SECAGO=3}",
						"owner": {
							"creatorUser": {
								"id": "${USERID}",
								"name": "${IGNORE}",
								"email": "${IGNORE}"
							},
							"createdUnixSec": "${SECAGO=3}"
						}
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Test sharing by user 1
	u1.ClearActions()

	u1.AddSendReqAction("Get permissions for created item as user 1",
		`{"getOwnershipReq": { "objectId": "${IDLOAD=u1CreatedROIId1}", "objectType": 2 }}`,
		fmt.Sprintf(`{
			"msgId": 12,
			"status": "WS_OK",
			"getOwnershipResp": {
				"ownership": {
					"id": "${IDCHK=u1CreatedROIId1}",
					"objectType": "OT_ROI",
					"creatorUserId": "${USERID}",
					"createdUnixSec": "${SECAGO=6}",
					"editors": {
						"userIds": ["%v"]
					}
				}
			}
		}`, u1.GetUserId()),
	)

	u1.AddSendReqAction("Share created item with user 2",
		fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "${IDLOAD=u1CreatedROIId1}", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u2.GetUserId()),
		fmt.Sprintf(`{
			"msgId": 13,
			"status": "WS_OK",
			"objectEditAccessResp": {
				"ownership": {
					"id": "${IDCHK=u1CreatedROIId1}",
					"objectType": "OT_ROI",
					"creatorUserId": "${USERID}",
					"createdUnixSec": "${SECAGO=6}",
					"viewers": {
						"userIds": ["%v"]
					},
					"editors": {
						"userIds": ["%v"]
					}
				}
			}
		}`, u2.GetUserId(), u1.GetUserId()),
	)

	u1.AddSendReqAction("Get shared item",
		`{"regionOfInterestGetReq": { "id": "${IDLOAD=u1CreatedROIId1}"}}`,
		`{"msgId":14, "status":"WS_OK", "regionOfInterestGetResp":{
			"regionOfInterest": {
				"id":"${IDCHK=u1CreatedROIId1}",
				"name": "The ROI",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98, 88],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`,
	)

	u1.AddSendReqAction("List items",
		`{"regionOfInterestListReq":{}}`,
		`{
			"msgId": 15,
			"status": "WS_OK",
			"regionOfInterestListResp": {
				"regionsOfInterest":{
					"${IDCHK=u1CreatedROIId1}": {
						"id":"${IDCHK=u1CreatedROIId1}",
						"name": "The ROI",
						"description": "User1 ROI1",
						"scanId": "048300551",
						"modifiedUnixSec": "${SECAGO=3}",
						"owner": {
							"creatorUser": {
								"id": "${USERID}",
								"name": "${IGNORE}",
								"email": "${IGNORE}"
							},
							"createdUnixSec": "${SECAGO=3}"
						}
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Back to user 2 - we should be able to view the shared item but still not edit
	u2.ClearActions()

	u2.AddSendReqAction("List for user 2",
		`{"regionOfInterestListReq":{}}`,
		fmt.Sprintf(`{
			"msgId": 5,
			"status": "WS_OK",
			"regionOfInterestListResp": {
				"regionsOfInterest":{
					"${IDCHK=u1CreatedROIId1}": {
						"id":"${IDCHK=u1CreatedROIId1}",
						"name": "The ROI",
						"description": "User1 ROI1",
						"scanId": "048300551",
						"modifiedUnixSec": "${SECAGO=3}",
						"owner": {
							"creatorUser": {
								"id": "%v",
								"name": "${IGNORE}",
								"email": "${IGNORE}"
							},
							"createdUnixSec": "${SECAGO=3}"
						}
					}
				}
			}
		}`, u1.GetUserId()),
	)

	u2.AddSendReqAction("Get shared item",
		`{"regionOfInterestGetReq": { "id": "${IDLOAD=u1CreatedROIId1}"}}`,
		fmt.Sprintf(`{"msgId":6, "status":"WS_OK", "regionOfInterestGetResp":{
			"regionOfInterest": {
				"id":"${IDCHK=u1CreatedROIId1}",
				"name": "The ROI",
				"description": "User1 ROI1",
				"scanId": "048300551",
				"locationIndexesEncoded": [14, 123, -1, 126, 98, 88],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
					"creatorUser": {
						"id": "%v",
						"name": "${IGNORE}",
						"email": "${IGNORE}"
					},
					"createdUnixSec": "${SECAGO=3}"
				}
			}
		}}`, u1.GetUserId()),
	)

	u2.AddSendReqAction("Edit created item, should fail, user2 is a viewer",
		`{"regionOfInterestWriteReq": {
			"regionOfInterest": {
				"id": "${IDLOAD=u1CreatedROIId1}",
				"name": "User1 ROI1-Edited by User2"
			}
		}}`,
		fmt.Sprintf(`{"msgId":7, "status":"WS_NO_PERMISSION", "errorText": "Edit access denied for: %v", "regionOfInterestWriteResp":{}}`, u1CreatedROIId1),
	)

	u2.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u2)

	// Back to user 1 - delete the element set
	u1.ClearActions()

	u1.AddSendReqAction("Delete created item",
		`{"regionOfInterestDeleteReq": { "id": "${IDLOAD=u1CreatedROIId1}" }}`,
		`{"msgId":16,"status":"WS_OK","regionOfInterestDeleteResp":{}}`,
	)

	u1.AddSendReqAction("List",
		`{"regionOfInterestListReq":{}}`,
		`{"msgId":17,"status":"WS_OK","regionOfInterestListResp":{}}`,
	)

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)
}
