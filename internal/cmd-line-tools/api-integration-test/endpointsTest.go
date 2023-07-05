package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

func runEndpointTests(apiHost string) {
	testUserDetails(apiHost)
	//testElementSets(apiHost)
}

func testUserDetails(apiHost string) {
	u1 := makeScriptedTestUser()
	u1.addAction("Connect", actionItem{
		connect: &connectInfo{
			host: apiHost,
			user: test1Username,
			pass: test1Password,
		},
	})

	u1.addAction("Request details", actionItem{
		sendReq: `{"userDetailsReq":{}}`,
	})

	u1.addAction("Edit details", actionItem{
		sendReq: `{"userDetailsWriteReq":{ "name": "Test 1 User", "email": "test1-2@pixlise.org", "dataCollectionVersion": "1.2.3" }}`,
	})

	u1.addAction("Request details again", actionItem{
		sendReq: `{"userDetailsReq":{}}`,
	})

	u1.addAction("Edit data collection version only", actionItem{
		sendReq: `{"userDetailsWriteReq":{ "dataCollectionVersion": "1.2.4" }}`,
	})

	u1.addAction("Request details again", actionItem{
		sendReq: `{"userDetailsReq":{}}`,
	})

	u1.addAction("Send blank data collection version", actionItem{
		sendReq: `{"userDetailsWriteReq":{ "dataCollectionVersion": "" }}`,
	})

	u1.addAction("Request details again", actionItem{
		sendReq: `{"userDetailsReq":{}}`,
	})

	u1.addAction("Edit but with invalid fields", actionItem{
		sendReq: `{"userDetailsWriteReq":{ "name": "one ridiculously long name that can't be possibly ever be valid" }}`,
	})

	u1.addAction("Request details again", actionItem{
		sendReq: `{"userDetailsReq":{}}`,
	})

	u1.addExpectedMessages([]string{
		`{"msgId":1,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"$USERID$","name":"test1@pixlise.org - WS Integration Test","email":"test1@pixlise.org"},
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_OWN_USER",
                "SHARE"
            ]}}}`,
		`{"msgId":2,"status":"WS_OK","userDetailsWriteResp":{}}`,
		`{"msgId":3,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"$USERID$","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.3",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_OWN_USER",
                "SHARE"
            ]}}}`,
		`{"msgId":4,"status":"WS_OK","userDetailsWriteResp":{}}`,
		`{"msgId":5,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"$USERID$","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_OWN_USER",
                "SHARE"
            ]}}}`,
		`{"msgId":6,"status":"WS_BAD_REQUEST", "errorText": "DataCollectionVersion is too short", "userDetailsWriteResp":{}}`,
		`{"msgId":7,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"$USERID$","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_OWN_USER",
                "SHARE"
            ]}}}`,
		`{"msgId":8,"status":"WS_BAD_REQUEST", "errorText": "Name is too long", "userDetailsWriteResp":{}}`,
		`{"msgId":9,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"$USERID$","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_OWN_USER",
                "SHARE"
            ]}}}`,
	}, 5000)

	// Run the test
	execQueuedActions(&u1)

	// Editing notification settings
	u1.clearActions()

	u1.addAction("Request notification settings", actionItem{
		sendReq: `{"userNotificationSettingsReq":{}}`,
	})

	u1.addAction("Add a few notification subscriptions", actionItem{
		sendReq: `{"userNotificationSettingsWriteReq":{
			"notifications":{
				"topicSettings":{
					"new-dataset": 1,
					"shared-item": 2
				}
			}
		}}`,
	})

	u1.addAction("Request notification settings", actionItem{
		sendReq: `{"userNotificationSettingsReq":{}}`,
	})

	u1.addAction("Remove a notification, while changing another", actionItem{
		sendReq: `{"userNotificationSettingsWriteReq":{
			"notifications":{
				"topicSettings":{
					"new-dataset": 3
				}
			}
		}}`,
	})

	u1.addAction("Request notification settings", actionItem{
		sendReq: `{"userNotificationSettingsReq":{}}`,
	})

	u1.addExpectedMessages([]string{
		`{"msgId":10,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{}}}`,
		`{"msgId":11,"status":"WS_OK","userNotificationSettingsWriteResp":{}}`,
		`{"msgId":12,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{
				"topicSettings": {
					"new-dataset": "NOTIF_EMAIL",
					"shared-item": "NOTIF_UI"
				}}}}`,
		`{"msgId":13,"status":"WS_OK","userNotificationSettingsWriteResp":{}}`,
		`{"msgId":14,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{
				"topicSettings": {
					"new-dataset": "NOTIF_BOTH"
				}}}}`,
	}, 5000)

	// Run the test
	execQueuedActions(&u1)

	// Editing hints
	u1.clearActions()

	u1.addAction("Request hints", actionItem{
		sendReq: `{"userHintsReq":{}}`,
	})

	u1.addAction("Add hint", actionItem{
		sendReq: `{"userDismissHintReq":{"hint":"context-zoom"}}`,
	})

	u1.addAction("Request hints again", actionItem{
		sendReq: `{"userHintsReq":{}}`,
	})

	u1.addAction("Add another hint", actionItem{
		sendReq: `{"userDismissHintReq":{"hint":"spectrum-pan"}}`,
	})

	u1.addAction("Request hints again", actionItem{
		sendReq: `{"userHintsReq":{}}`,
	})

	u1.addAction("Disable all hints", actionItem{
		sendReq: `{"userHintsToggleReq":{"enabled":false}}`,
	})

	u1.addAction("Request hints again", actionItem{
		sendReq: `{"userHintsReq":{}}`,
	})

	u1.addExpectedMessages([]string{
		`{"msgId":15,"status":"WS_OK","userHintsResp":{"hints":{"enabled": true}}}`, // note, dismissedHints is empty, so 0-value sent is nothing...
		`{"msgId":16,"status":"WS_OK","userDismissHintResp":{}}`,
		`{"msgId":17,"status":"WS_OK","userHintsResp":{"hints":{"dismissedHints": ["context-zoom"], "enabled": true}}}`,
		`{"msgId":18,"status":"WS_OK","userDismissHintResp":{}}`,
		`{"msgId":19,"status":"WS_OK","userHintsResp":{"hints":{"dismissedHints": ["context-zoom", "spectrum-pan"], "enabled": true}}}`,
		`{"msgId":20,"status":"WS_OK","userHintsToggleResp":{}}`,
		`{"msgId":21,"status":"WS_OK","userHintsResp":{"hints":{"dismissedHints": ["context-zoom", "spectrum-pan"]}}}`, // note, enabled is false, so 0-value sent is nothing...
	}, 5000)

	// Run the test
	execQueuedActions(&u1)
}

func testElementSets(apiHost string) {
	u1 := makeScriptedTestUser()

	u1.addAction("Connect user 1", actionItem{
		connect: &connectInfo{
			host: apiHost,
			user: test1Username,
			pass: test1Password,
		},
	})

	u1.addAction("List", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	u1.addAction("Get non-existant id", actionItem{
		sendReq: `{"elementSetGetReq": { "id": "non-existant-id"}}`,
	})

	u1.addAction("Create invalid item", actionItem{
		sendReq: `{"elementSetWriteReq": {
			"elementSet": {
				"name": "User1 ElementSet1",
				"lines": []
			}
		}}`,
	})

	u1.addAction("Delete non-existant item", actionItem{
		sendReq: `{"elementSetDeleteReq": { "id": "non-existant-id" }}`,
	})

	u1.addAction("Edit non-existant item", actionItem{
		sendReq: `{"elementSetWriteReq": {
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
	})

	u1.addAction("Create valid item", actionItem{
		sendReq: `{"elementSetWriteReq": {
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
	})

	// Verify the above
	u1.addExpectedMessages([]string{
		`{"msgId":1,"status":"WS_OK","elementSetListResp":{}}`,
		`{"msgId":2, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetGetResp":{}}`,
		`{"msgId":3, "status":"WS_BAD_REQUEST", "errorText": "Lines length is invalid", "elementSetWriteResp":{}}`,
		`{"msgId":4, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetDeleteResp":{}}`,
		`{"msgId":5, "status":"WS_NOT_FOUND", "errorText": "non-existant-id not found", "elementSetWriteResp":{}}`,
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
	}, 60000)

	// Run the test
	execQueuedActions(&u1)

	u1CreatedElementSetId1 := u1.getIdCreated("elem1") // Remember the ID that was created

	// Login as another user and list items to verify none are coming back here too
	u2 := makeScriptedTestUser()

	u2.addAction("Connect user 2", actionItem{
		connect: &connectInfo{
			host: apiHost,
			user: test2Username,
			pass: test2Password,
		},
	})

	u2.addAction("List for user 2", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	// Stop here, we need the user id going forward...
	u2.addExpectedMessages([]string{
		`{"msgId":1,"status":"WS_OK","elementSetListResp":{}}`,
	}, 60000)

	// Run the test
	execQueuedActions(&u2)

	u2.clearActions()

	u2.addAction("Get created item for user 2", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
	})

	u2.addAction("Get permissions for user 1's created item", actionItem{
		sendReq: fmt.Sprintf(`{"getOwnershipReq": { "objectId": "%v", "objectType": 2 }}`, u1CreatedElementSetId1),
	})

	u2.addAction("Share user 1s created item", actionItem{
		sendReq: fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "%v", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u1CreatedElementSetId1, u2.user.userId),
	})

	// Verify the above
	u2.addExpectedMessages([]string{
		fmt.Sprintf(`{"msgId":2, "status": "WS_NO_PERMISSION", "errorText": "View access denied for: %v", "elementSetGetResp":{}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":3,"status":"WS_NO_PERMISSION","errorText": "View access denied for: %v","getOwnershipResp":{}}`, u1CreatedElementSetId1),
		fmt.Sprintf(`{"msgId":4,"status":"WS_NO_PERMISSION","errorText": "Edit access denied for: %v","objectEditAccessResp":{}}`, u1CreatedElementSetId1),
	}, 60000)

	// Run the test
	execQueuedActions(&u2)

	// Back to user 1
	u1.clearActions()

	u1.addAction("Get created item for user 1", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
	})
	u1.addAction("Edit created item with invalid request", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetWriteReq": {
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
	})
	u1.addAction("Edit created item", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetWriteReq": {
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
	})

	u1.addAction("Get edited item", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
	})

	u1.addAction("List items", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	u1.addExpectedMessages([]string{
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

		`{
			"msgId": 8,
			"status": "WS_BAD_REQUEST",
			"errorText": "Name length is invalid",
			"elementSetWriteResp": {}
		}`,

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
	}, 60000)

	execQueuedActions(&u1)

	// Test sharing by user 1
	u1.clearActions()

	u1.addAction("Get permissions for created item as user 1", actionItem{
		sendReq: fmt.Sprintf(`{"getOwnershipReq": { "objectId": "%v", "objectType": 2 }}`, u1CreatedElementSetId1),
	})

	u1.addAction("Share created item with user 2", actionItem{
		sendReq: fmt.Sprintf(`{"objectEditAccessReq": { "objectId": "%v", "objectType": 2, "addViewers": { "userIds": [ "%v" ] }}}`, u1CreatedElementSetId1, u2.user.userId),
	})

	u1.addAction("Get shared item", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
	})

	u1.addAction("List items", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	u1.addExpectedMessages([]string{
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
		}`, u1CreatedElementSetId1, u1.user.userId),

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
		}`, u1CreatedElementSetId1, u2.user.userId, u1.user.userId),

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
	}, 60000)

	execQueuedActions(&u1)

	// Back to user 2 - we should be able to view the shared item but still not edit
	u2.clearActions()

	u2.addAction("List for user 2", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	u2.addAction("Get shared item", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetGetReq": { "id": "%v"}}`, u1CreatedElementSetId1),
	})

	u2.addAction("Edit created item, should fail, user2 is a viewer", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetWriteReq": {
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
	})

	u2.addExpectedMessages([]string{
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
		}`, u1CreatedElementSetId1, u1CreatedElementSetId1, u1.user.userId),

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
		}}`, u1CreatedElementSetId1, u1.user.userId),

		fmt.Sprintf(`{"msgId":7, "status":"WS_NO_PERMISSION", "errorText": "Edit access denied for: %v", "elementSetWriteResp":{}}`, u1CreatedElementSetId1),
	}, 60000)

	execQueuedActions(&u2)

	// Back to user 1 - delete the element set
	u1.clearActions()

	u1.addAction("Delete created item", actionItem{
		sendReq: fmt.Sprintf(`{"elementSetDeleteReq": { "id": "%v" }}`, u1CreatedElementSetId1),
	})

	u1.addAction("List", actionItem{
		sendReq: `{"elementSetListReq":{}}`,
	})

	// Verify the above
	u1.addExpectedMessages([]string{
		`{"msgId":16,"status":"WS_OK","elementSetDeleteResp":{}}`,
		`{"msgId":17,"status":"WS_OK","elementSetListResp":{}}`,
	}, 60000)

	execQueuedActions(&u1)
}

func execQueuedActions(u *scriptedTestUser) {
	// Program counter doesn't seem useful right now
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "UNKNOWN file"
		line = -1
	} else {
		// dont need the whole path
		file = filepath.Base(file)
	}

	// Run the actions
	fmt.Printf("Running actions [%v (%v)]\n", file, line)

	for {
		running, err := u.runNextAction()
		if err != nil {

			log.Fatalf("%v (%v): %v\n", file, line, err)
		}
		if !running {
			fmt.Println("Queued actions complete")
			fmt.Printf("-----------------------\n\n")
			break
		}
	}
}
