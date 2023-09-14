package main

import (
	"github.com/pixlise/core/v3/core/wstestlib"
)

func testTags(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create a new tag (not allowed)",
		`{"tagCreateReq":{"name": "someTag", "type": "expression", "scanId": "scan1"}}`,
		`{
			"msgId": 1,
			"status": "WS_NO_PERMISSION",
			"errorText": "TagCreateReq not allowed",
			"tagCreateResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Create a new tag (allowed)",
		`{"tagCreateReq":{"name": "someTag", "type": "expression", "scanId": "scan1"}}`,
		`{"msgId":1,"status":"WS_OK","tagCreateResp":{
			"tag": {
				"id": "${IDSAVE=tagId}",
				"name": "someTag",
				"type": "expression",
				"scanId": "scan1",
				"owner": {
					"id": "${IDSAVE=ownerId}",
					"name": "test2@pixlise.org - WS Integration Test",
					"email": "test2@pixlise.org"
				}
			}
		}}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

	u2.AddSendReqAction("List tags",
		`{"tagListReq":{}}`,
		`{"msgId":2,"status":"WS_OK","tagListResp":{
			"tags": [
				{
					"id": "${IDCHK=tagId}",
					"name": "someTag",
					"type": "expression",
					"scanId": "scan1",
					"owner": {
						"id": "${IDCHK=ownerId}",
						"name": "test2@pixlise.org - WS Integration Test",
						"email": "test2@pixlise.org"
					}
				}
			]
		}}`,
	)

	// Delete the tag
	u2.AddSendReqAction("Delete the tag",
		`{"tagDeleteReq":{"tagId": "${IDLOAD=tagId}"}}`,
		`{"msgId":3,"status":"WS_OK","tagDeleteResp":{}}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

}
