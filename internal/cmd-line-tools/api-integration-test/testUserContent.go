package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/wstestlib"
)

type contentMessaging struct {
	itemName string
	listName string

	// Invalid items - JUST the fields within the structure, so test can add ids as needed
	// Also specifying the expected error message for this item
	/* Example:
	"name": "name that is way too long to be a valid name, or missing fields, etc",
	"description": "User1 ROI1"
	*/
	invalidItemsToCreate [][]string

	// Valid items - JUST the fields within the structure, so test can add ids as needed
	// Each list contains 3 strings:
	// - the request sent for creation
	// - the item as seen in response and GET,
	// - the item as seen in LIST
	// as these will likely differ if we have list items just as a "summary"
	/* Example request item:
	"name": "User1 ROI1",
	"description": "User1 ROI1",
	"scanId": "048300551"
	*/
	validItemsToCreate [][]string

	// Invalid edits - JUST the fields within the structure and the expected error
	invalidItemsToEdit [][]string

	// Valid edits - JUST the fields within the structure and the expected returned items
	// specified as 3 strings per list, containing REQ, GET and LIST
	validItemsToEdit [][]string

	objectType string
}

func testUserContent(apiHost string, contentMessaging map[string]contentMessaging) {
	// One "smart" test sequence that can be configured to test things automatically, applying the same preconditions
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect user 1", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	var u1ExpectedRespSeqNo = 1
	createdItemIds := map[string][]string{}
	u1ItemsForGet := map[string][]string{}
	u1ItemsForList := map[string][]string{}

	for msgName, msgContents := range contentMessaging {
		// We will end up with items for this user to see
		u1ItemsForGet[msgName] = []string{}
		u1ItemsForList[msgName] = []string{}

		u1.AddSendReqAction(fmt.Sprintf("%v List", msgName),
			fmt.Sprintf(`{"%vListReq":{}}`, msgName),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_OK","%vListResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++

		u1.AddSendReqAction(fmt.Sprintf("%v Get non-existant id", msgName),
			fmt.Sprintf(`{"%vGetReq": { "id": "non-existant-id"}}`, msgName),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "%vGetResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++

		u1.AddSendReqAction(fmt.Sprintf("%v Delete non-existant item", msgName),
			fmt.Sprintf(`{"%vDeleteReq": { "id": "non-existant-id" }}`, msgName),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "%vDeleteResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++

		u1.AddSendReqAction(fmt.Sprintf("%v Edit non-existant item", msgName),
			fmt.Sprintf(`{"%vWriteReq": {
				"%v": {
					"id": "non-existant-id",
					%v
				}
			}}`, msgName, msgContents.itemName, msgContents.validItemsToCreate[0][0]),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "%vWriteResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++

		for _, invalidItem := range msgContents.invalidItemsToCreate {
			u1.AddSendReqAction(fmt.Sprintf("%v Create invalid item (no indexes defined)", msgName),
				fmt.Sprintf(`{"%vWriteReq": {
					"%v": {
						%v
					}
				}}`, msgName, msgContents.itemName, invalidItem[0]),
				fmt.Sprintf(`{"msgId":%v, "status":"WS_BAD_REQUEST", "errorText": "%v", "%vWriteResp":{}}`, u1ExpectedRespSeqNo, invalidItem[1], msgName),
			)
			u1ExpectedRespSeqNo++
		}

		for _, validItem := range msgContents.validItemsToCreate {
			u1.AddSendReqAction(fmt.Sprintf("%v Create valid item", msgName),
				fmt.Sprintf(`{"%vWriteReq": {
						"%v": {
							%v
						}
					}}`, msgName, msgContents.itemName, validItem[0]),
				fmt.Sprintf(`{"msgId":%v, "status":"WS_OK",
					"%vWriteResp": {
						"%v": {
							"id":"${IDSAVE=%vCreated1}",
							%v,
							"modifiedUnixSec": "${SECAGO=3}",
							"owner": {
								"creatorUser": {
									"id": "${USERID}",
									"name": "${REGEXMATCH=test}",
									"email": "${REGEXMATCH=.+@pixlise\\.org}"
								},
								"createdUnixSec": "${SECAGO=3}",
								"canEdit": "${IGNORE}"
							}
						}
				}}`, u1ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, validItem[1]),
			)
			u1ExpectedRespSeqNo++

			// From this point we expect these items to exist for user
			u1ItemsForGet[msgName] = append(u1ItemsForGet[msgName], validItem[1])
			u1ItemsForList[msgName] = append(u1ItemsForGet[msgName], validItem[2])
		}
	}

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)

	for msgName := range contentMessaging {
		id := wstestlib.GetIdCreated(msgName + "Created1") // Remember the ID that was created
		createdItemIds[msgName] = []string{id}
	}

	// Login as another user and list items to verify none are coming back here too
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)

	u2.AddConnectAction("Connect user 2", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	var u2ExpectedRespSeqNo = 1
	u2ItemsForGet := map[string][]string{}
	u2ItemsForList := map[string][]string{}

	for msgName := range contentMessaging {
		// We will end up with items for this user to see
		u2ItemsForGet[msgName] = []string{}
		u2ItemsForList[msgName] = []string{}

		u2.AddSendReqAction(fmt.Sprintf("%v List for user 2", msgName),
			fmt.Sprintf(`{"%vListReq":{}}`, msgName),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_OK","%vListResp":{}}`, u2ExpectedRespSeqNo, msgName),
		)
		u2ExpectedRespSeqNo++
	}

	// Stop here, we need the user id going forward...
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	u2.ClearActions()

	for msgName, msgContents := range contentMessaging {
		createdId := createdItemIds[msgName][0]

		u2.AddSendReqAction(fmt.Sprintf("%v Get created item for user 2", msgName),
			fmt.Sprintf(`{"%vGetReq": { "id": "${IDLOAD=%vCreated1}"}}`, msgName, msgName),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_NO_PERMISSION", "errorText": "View access denied for: %v (%v)", "%vGetResp":{}}`, u2ExpectedRespSeqNo, msgContents.objectType, createdId, msgName),
		)
		u2ExpectedRespSeqNo++

		u2.AddSendReqAction(fmt.Sprintf("%v Get permissions for user 1's created item", msgName),
			fmt.Sprintf(`{"getOwnershipReq": { "objectId": "${IDLOAD=%vCreated1}", "objectType": "%v"}}`, msgName, msgContents.objectType),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_NO_PERMISSION","errorText": "View access denied for: %v (%v)","getOwnershipResp":{}}`, u2ExpectedRespSeqNo, msgContents.objectType, createdId),
		)
		u2ExpectedRespSeqNo++

		u2.AddSendReqAction(fmt.Sprintf("%v Share user 1s created item", msgName),
			fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "${IDLOAD=%vCreated1}", "objectType": "%v", "addViewers": { "userIds": [ "%v" ] }}}`, msgName, msgContents.objectType, u2.GetUserId()),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_NO_PERMISSION","errorText": "Edit access denied for: %v (%v)","objectEditAccessResp":{}}`, u2ExpectedRespSeqNo, msgContents.objectType, createdId),
		)
		u2ExpectedRespSeqNo++
	}

	// Verify the above
	u2.CloseActionGroup([]string{}, 60000)

	// Run the test
	wstestlib.ExecQueuedActions(&u2)

	// Back to user 1
	u1.ClearActions()

	for msgName, msgContents := range contentMessaging {
		u1.AddSendReqAction(fmt.Sprintf("%v Get created item for user 1", msgName),
			fmt.Sprintf(`{"%vGetReq": { "id": "${IDLOAD=%vCreated1}"}}`, msgName, msgName),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_OK", "%vGetResp":{
				"%v":{
					"id":"${IDCHK=%vCreated1}",
					%v,
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": "${IGNORE}"
					}
				}
			}}`, u1ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, u1ItemsForGet[msgName][0]),
		)
		u1ExpectedRespSeqNo++

		for _, item := range msgContents.invalidItemsToEdit {
			u1.AddSendReqAction(fmt.Sprintf("%v Edit created item with invalid request", msgName),
				fmt.Sprintf(`{"%vWriteReq": {
					"%v": {
						"id": "${IDLOAD=%vCreated1}",
						%v
					}
				}}`, msgName, msgContents.itemName, msgName, item[0]),
				fmt.Sprintf(`{"msgId":%v,
					"status": "WS_BAD_REQUEST",
					"errorText": "%v",
					"%vWriteResp": {}
				}`, u1ExpectedRespSeqNo, item[1], msgName),
			)
			u1ExpectedRespSeqNo++
		}

		for _, editItem := range msgContents.validItemsToEdit {
			u1.AddSendReqAction(fmt.Sprintf("%v Edit created item", msgName),
				fmt.Sprintf(`{"%vWriteReq": {
					"%v": {
						"id": "${IDLOAD=%vCreated1}",
						%v
					}
				}}`, msgName, msgContents.itemName, msgName, editItem[0]),
				fmt.Sprintf(`{"msgId":%v, "status":"WS_OK", "%vWriteResp":{
					"%v":{
						"id":"${IDCHK=%vCreated1}",
						%v,
						"modifiedUnixSec": "${SECAGO=3}",
						"owner": {
							"creatorUser": {
								"id": "${USERID}",
								"name": "${REGEXMATCH=test}",
								"email": "${REGEXMATCH=.+@pixlise\\.org}"
							},
							"createdUnixSec": "${SECAGO=3}",
							"canEdit": "${IGNORE}"
						}
					}
				}}`, u1ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, editItem[1]),
			)
			u1ExpectedRespSeqNo++

			// Item has been edited
			u1ItemsForGet[msgName] = []string{editItem[1]}
			u1ItemsForList[msgName] = []string{editItem[2]}

			u1.AddSendReqAction(fmt.Sprintf("%v Get edited item", msgName),
				fmt.Sprintf(`{"%vGetReq": { "id": "${IDLOAD=%vCreated1}"}}`, msgName, msgName),
				fmt.Sprintf(`{"msgId":%v, "status":"WS_OK", "%vGetResp":{
					"%v":{
						"id":"${IDCHK=%vCreated1}",
						%v,
						"modifiedUnixSec": "${SECAGO=3}",
						"owner": {
							"creatorUser": {
								"id": "${USERID}",
								"name": "${REGEXMATCH=test}",
								"email": "${REGEXMATCH=.+@pixlise\\.org}"
							},
							"createdUnixSec": "${SECAGO=3}",
							"canEdit": "${IGNORE}"
						}
					}
				}}`, u1ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, u1ItemsForGet[msgName][0]),
			)
			u1ExpectedRespSeqNo++

			u1.AddSendReqAction(fmt.Sprintf("%v List items", msgName),
				fmt.Sprintf(`{"%vListReq":{}}`, msgName),
				fmt.Sprintf(`{"msgId":%v, "status": "WS_OK", "%vListResp": {
							"%v":{
								"${IDCHK=%vCreated1}": {
								"id":"${IDCHK=%vCreated1}",
								%v,
								"modifiedUnixSec": "${SECAGO=3}",
								"owner": {
									"creatorUser": {
										"id": "${USERID}",
										"name": "${REGEXMATCH=test}",
										"email": "${REGEXMATCH=.+@pixlise\\.org}"
									},
									"createdUnixSec": "${SECAGO=3}",
									"canEdit": "${IGNORE}"
								}
							}
						}
					}
				}`, u1ExpectedRespSeqNo, msgName, msgContents.listName, msgName, msgName, u1ItemsForList[msgName][0]),
			)
			u1ExpectedRespSeqNo++
		}
	}

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Test sharing by user 1
	u1.ClearActions()

	for msgName, msgContents := range contentMessaging {
		u1.AddSendReqAction(fmt.Sprintf("%v Get permissions for created item as user 1", msgName),
			fmt.Sprintf(`{"getOwnershipReq": { "objectId": "${IDLOAD=%vCreated1}", "objectType": "%v" }}`, msgName, msgContents.objectType),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_OK",
				"getOwnershipResp": {
					"ownership": {
						"id": "${IDCHK=%vCreated1}",
						"objectType": "%v",
						"creatorUserId": "${USERID}",
						"createdUnixSec": "${SECAGO=6}",
						"editors": {
							"userIds": ["%v"]
						}
					}
				}
			}`, u1ExpectedRespSeqNo, msgName, msgContents.objectType, u1.GetUserId()),
		)
		u1ExpectedRespSeqNo++

		u1.AddSendReqAction(fmt.Sprintf("%v Share created item with user 2", msgName),
			fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "${IDLOAD=%vCreated1}", "objectType": "%v", "addViewers": { "userIds": [ "%v" ] }}}`, msgName, msgContents.objectType, u2.GetUserId()),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_OK",
				"objectEditAccessResp": {
					"ownership": {
						"id": "${IDCHK=%vCreated1}",
						"objectType": "%v",
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
			}`, u1ExpectedRespSeqNo, msgName, msgContents.objectType, u2.GetUserId(), u1.GetUserId()),
		)
		u1ExpectedRespSeqNo++

		// From this point, user 2 can see user1's object
		u2ItemsForGet[msgName] = u1ItemsForGet[msgName]
		u2ItemsForList[msgName] = u1ItemsForList[msgName]

		u1.AddSendReqAction(fmt.Sprintf("%v Get shared item", msgName),
			fmt.Sprintf(`{"%vGetReq": { "id": "${IDLOAD=%vCreated1}"}}`, msgName, msgName),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_OK", "%vGetResp":{
				"%v": {
					"id":"${IDCHK=%vCreated1}",
					%v,
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"viewerUserCount": 1,
						"sharedWithOthers": true,
						"canEdit": "${IGNORE}"
					}
				}
			}}`, u1ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, u1ItemsForGet[msgName][0]),
		)
		u1ExpectedRespSeqNo++

		u1.AddSendReqAction(fmt.Sprintf("%v List items", msgName),
			fmt.Sprintf(`{"%vListReq":{}}`, msgName),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_OK", "%vListResp": {
					"%v":{
						"${IDCHK=%vCreated1}": {
							"id":"${IDCHK=%vCreated1}",
							%v,
							"modifiedUnixSec": "${SECAGO=3}",
							"owner": {
								"creatorUser": {
									"id": "${USERID}",
									"name": "${REGEXMATCH=test}",
									"email": "${REGEXMATCH=.+@pixlise\\.org}"
								},
								"createdUnixSec": "${SECAGO=3}",
								"viewerUserCount": 1,
								"sharedWithOthers": true,
								"canEdit": "${IGNORE}"
							}
						}
					}
				}
			}`, u1ExpectedRespSeqNo, msgName, msgContents.listName, msgName, msgName, u1ItemsForList[msgName][0]),
		)
		u1ExpectedRespSeqNo++
	}

	u1.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u1)

	// Back to user 2 - we should be able to view the shared item but still not edit
	u2.ClearActions()

	for msgName, msgContents := range contentMessaging {
		createdId := createdItemIds[msgName][0]
		u2.AddSendReqAction(fmt.Sprintf("%v List items", msgName),
			fmt.Sprintf(`{"%vListReq":{}}`, msgName),
			fmt.Sprintf(`{"msgId":%v, "status": "WS_OK", "%vListResp": {
					"%v":{
						"${IDCHK=%vCreated1}": {
							"id":"${IDCHK=%vCreated1}",
							%v,
							"modifiedUnixSec": "${SECAGO=3}",
							"owner": {
								"creatorUser": {
									"id": "%v",
									"name": "${REGEXMATCH=test}",
									"email": "${REGEXMATCH=.+@pixlise\\.org}"
								},
								"createdUnixSec": "${SECAGO=3}",
								"viewerUserCount": 1,
								"sharedWithOthers": true,
								"canEdit": "${IGNORE}"
							}
						}
					}
				}
			}`, u2ExpectedRespSeqNo, msgName, msgContents.listName, msgName, msgName, u2ItemsForList[msgName][0], u1.GetUserId()),
		)
		u2ExpectedRespSeqNo++

		u2.AddSendReqAction(fmt.Sprintf("%v Get shared item", msgName),
			fmt.Sprintf(`{"%vGetReq": { "id": "${IDLOAD=%vCreated1}"}}`, msgName, msgName),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_OK", "%vGetResp":{
				"%v": {
					"id":"${IDCHK=%vCreated1}",
					%v,
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "%v",
							"name": "${IGNORE}",
							"email": "${IGNORE}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"viewerUserCount": 1,
						"sharedWithOthers": true,
						"canEdit": "${IGNORE}"
					}
				}
			}}`, u2ExpectedRespSeqNo, msgName, msgContents.itemName, msgName, u2ItemsForGet[msgName][0], u1.GetUserId()),
		)
		u2ExpectedRespSeqNo++

		u2.AddSendReqAction(fmt.Sprintf("%v Edit created item, should fail, user2 is a viewer", msgName),
			fmt.Sprintf(`{"%vWriteReq": {
				"%v": {
					"id": "${IDLOAD=%vCreated1}",
					"name": "User1 Item Edited by User2"
				}
			}}`, msgName, msgContents.itemName, msgName),
			fmt.Sprintf(`{"msgId":%v, "status":"WS_NO_PERMISSION", "errorText": "Edit access denied for: %v (%v)", "%vWriteResp":{}}`,
				u2ExpectedRespSeqNo, msgContents.objectType, createdId, msgName),
		)
		u2ExpectedRespSeqNo++
	}

	u2.CloseActionGroup([]string{}, 60000)

	wstestlib.ExecQueuedActions(&u2)

	// TODO: Share with a group, check: "viewerUserCount" "sharedWithOthers" changes. Unshare group, unshare viewer, check again

	// Back to user 1 - delete the item
	u1.ClearActions()

	for msgName := range contentMessaging {
		u1.AddSendReqAction(fmt.Sprintf("%v Delete created item", msgName),
			fmt.Sprintf(`{"%vDeleteReq": { "id": "${IDLOAD=%vCreated1}" }}`, msgName, msgName),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_OK","%vDeleteResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++

		// Should not exist any more for anyone
		u1ItemsForList[msgName] = []string{}
		u1ItemsForGet[msgName] = []string{}
		u2ItemsForList[msgName] = []string{}
		u2ItemsForGet[msgName] = []string{}

		u1.AddSendReqAction(fmt.Sprintf("%v List to confirm delete", msgName),
			fmt.Sprintf(`{"%vListReq":{}}`, msgName),
			fmt.Sprintf(`{"msgId":%v,"status":"WS_OK","%vListResp":{}}`, u1ExpectedRespSeqNo, msgName),
		)
		u1ExpectedRespSeqNo++
	}

	// Verify the above
	u1.CloseActionGroup([]string{}, 60000)
	wstestlib.ExecQueuedActions(&u1)
}
