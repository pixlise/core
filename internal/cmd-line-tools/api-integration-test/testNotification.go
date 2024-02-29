package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testNotification(apiHost string) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.NotificationsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u1.AddSendReqAction("Get notifications (should be empty)",
		`{"notificationReq":{}}`,
		`{"msgId":1,"status":"WS_OK","notificationResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Send notification to self",
		fmt.Sprintf(`{"sendUserNotificationReq":{
			"userIds": ["%v"],
			"notification": { "subject": "test subject", "contents": "The body"}
		}}`, u1.GetUserId()),
		`{"msgId":2,"status":"WS_OK","sendUserNotificationResp":{}}`,
	)

	// Expecting to see an update message
	u1.CloseActionGroup([]string{`{
		"notificationUpd": {
			"notification": {
				"id": "${IDSAVE=notificationId}",
				"subject": "test subject",
				"contents": "The body\nThis message was sent by test2@pixlise.org - WS Integration Test",
				"from": "test2@pixlise.org - WS Integration Test",
				"timeStampUnixSec": "${SECAGO=5}",
				"notificationType": "NT_USER_MESSAGE"
			}
		}
	}`}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Get notifications (should see some)",
		`{"notificationReq":{}}`,
		`{"msgId": 3, "status": "WS_OK", "notificationResp": {
				"notification": [
					{
						"id": "${IDSAVE=notificationId}",
						"destUserId": "${USERID}",
						"subject": "test subject",
						"contents": "The body\nThis message was sent by test2@pixlise.org - WS Integration Test",
						"from": "test2@pixlise.org - WS Integration Test",
						"timeStampUnixSec": "${SECAGO=5}",
						"notificationType": "NT_USER_MESSAGE"
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("Dismiss notification",
		`{"notificationDismissReq":{}}`,
		`{"msgId":4,"status": "WS_BAD_REQUEST","errorText": "Id is too short","notificationDismissResp":{}}`,
	)

	u1.AddSendReqAction("Dismiss notification",
		`{"notificationDismissReq":{"id": "${IDLOAD=notificationId}"}}`,
		`{"msgId":5,"status":"WS_OK", "notificationDismissResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Get notifications (should be empty)",
		`{"notificationReq":{}}`,
		`{"msgId":6,"status":"WS_OK","notificationResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
