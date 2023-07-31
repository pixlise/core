package main

import "github.com/pixlise/core/v3/core/wstestlib"

func testUserDetails(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Request details",
		`{"userDetailsReq":{}}`,
		`{"msgId":1,"status":"WS_OK","userDetailsResp":{
				"details":{"info":{"id":"${USERID}","name":"test1@pixlise.org - WS Integration Test","email":"test1@pixlise.org"},
				"permissions": [
					"EDIT_ELEMENT_SET",
					"EDIT_EXPRESSION",
					"EDIT_EXPRESSION_GROUP",
					"EDIT_OWN_USER",
					"EDIT_ROI",
					"SHARE"
				]}}}`,
	)

	u1.AddSendReqAction("Edit details",
		`{"userDetailsWriteReq":{ "name": "Test 1 User", "email": "test1-2@pixlise.org", "dataCollectionVersion": "1.2.3" }}`,
		`{"msgId":2,"status":"WS_OK","userDetailsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request details again",
		`{"userDetailsReq":{}}`,
		`{"msgId":3,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"${USERID}","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.3",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_EXPRESSION",
                "EDIT_EXPRESSION_GROUP",
                "EDIT_OWN_USER",
                "EDIT_ROI",
                "SHARE"
            ]}}}`,
	)

	u1.AddSendReqAction("Edit data collection version only",
		`{"userDetailsWriteReq":{ "dataCollectionVersion": "1.2.4" }}`,
		`{"msgId":4,"status":"WS_OK","userDetailsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request details again",
		`{"userDetailsReq":{}}`,
		`{"msgId":5,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"${USERID}","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_EXPRESSION",
                "EDIT_EXPRESSION_GROUP",
                "EDIT_OWN_USER",
                "EDIT_ROI",
                "SHARE"
            ]}}}`,
	)

	u1.AddSendReqAction("Send blank data collection version",
		`{"userDetailsWriteReq":{ "dataCollectionVersion": "" }}`,
		`{"msgId":6,"status":"WS_BAD_REQUEST", "errorText": "DataCollectionVersion is too short", "userDetailsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request details again",
		`{"userDetailsReq":{}}`,
		`{"msgId":7,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"${USERID}","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_EXPRESSION",
                "EDIT_EXPRESSION_GROUP",
                "EDIT_OWN_USER",
                "EDIT_ROI",
                "SHARE"
            ]}}}`,
	)

	u1.AddSendReqAction("Edit but with invalid fields",
		`{"userDetailsWriteReq":{ "name": "one ridiculously long name that can't be possibly ever be valid" }}`,
		`{"msgId":8,"status":"WS_BAD_REQUEST", "errorText": "Name is too long", "userDetailsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request details again",
		`{"userDetailsReq":{}}`,
		`{"msgId":9,"status":"WS_OK","userDetailsResp":{
			"details":{"info":{"id":"${USERID}","name":"Test 1 User","email":"test1-2@pixlise.org"},
			"dataCollectionVersion": "1.2.4",
            "permissions": [
                "EDIT_ELEMENT_SET",
                "EDIT_EXPRESSION",
                "EDIT_EXPRESSION_GROUP",
                "EDIT_OWN_USER",
                "EDIT_ROI",
                "SHARE"
            ]}}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)

	// Editing notification settings
	u1.ClearActions()

	u1.AddSendReqAction("Request notification settings",
		`{"userNotificationSettingsReq":{}}`,
		`{"msgId":10,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{}}}`,
	)

	u1.AddSendReqAction("Add a few notification subscriptions",
		`{"userNotificationSettingsWriteReq":{
			"notifications":{
				"topicSettings":{
					"new-dataset": 1,
					"shared-item": 2
				}
			}
		}}`,
		`{"msgId":11,"status":"WS_OK","userNotificationSettingsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request notification settings",
		`{"userNotificationSettingsReq":{}}`,
		`{"msgId":12,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{
				"topicSettings": {
					"new-dataset": "NOTIF_EMAIL",
					"shared-item": "NOTIF_UI"
				}}}}`,
	)

	u1.AddSendReqAction("Remove a notification, while changing another",
		`{"userNotificationSettingsWriteReq":{
			"notifications":{
				"topicSettings":{
					"new-dataset": 3
				}
			}
		}}`,
		`{"msgId":13,"status":"WS_OK","userNotificationSettingsWriteResp":{}}`,
	)

	u1.AddSendReqAction("Request notification settings",
		`{"userNotificationSettingsReq":{}}`,
		`{"msgId":14,"status":"WS_OK","userNotificationSettingsResp":{
			"notifications":{
				"topicSettings": {
					"new-dataset": "NOTIF_BOTH"
				}}}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)

	// Run the test
	wstestlib.ExecQueuedActions(&u1)
}
