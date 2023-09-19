package main

import "github.com/pixlise/core/v3/core/wstestlib"

func testExpressionRuntimeMsgs(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create expression",
		`{"expressionWriteReq":{
			"expression": {
				"name": "User1 Expression",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"sourceLanguage": "LUA",
				"sourceCode": "element(\"Ca\")"
			}
		}}`,
		`{"msgId":1,"status":"WS_OK",
			"expressionWriteResp":{
				"expression": {
					"id":"${IDSAVE=CreatedForStat1}",
					"name": "User1 Expression",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR RUNTIME STAT SAVE checking",
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}"
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Save expression runtime stats (should fail)",
		`{"expressionWriteExecStatReq":{
			"id": "${IDLOAD=CreatedForStat1}",
			"stats": {
				"dataRequired": [],
				"runtimeMsPer1000Pts": 2
			}
		}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST",
			"errorText": "Invalid stats in request",
			"expressionWriteExecStatResp":{}
		}`,
	)

	u1.AddSendReqAction("Save expression runtime stats",
		`{"expressionWriteExecStatReq":{
			"id": "${IDLOAD=CreatedForStat1}",
			"stats": {
				"dataRequired": ["Fe", "Ca"],
				"runtimeMsPer1000Pts": 2.1
			}
		}}`,
		`{"msgId":3,"status":"WS_OK",
			"expressionWriteExecStatResp":{}
		}`,
	)

	u1.AddSendReqAction("Read expression back expecting runtime stats",
		`{"expressionGetReq":{
			"id": "${IDLOAD=CreatedForStat1}"
		}}`,
		`{"msgId":4,"status":"WS_OK",
			"expressionGetResp":{
				"expression": {
					"id":"${IDCHK=CreatedForStat1}",
					"name": "User1 Expression",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR RUNTIME STAT SAVE checking",
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}"
					},
					"recentExecStats": {
						"dataRequired": [
							"Fe",
							"Ca"
						],
						"runtimeMsPer1000Pts": 2.1,
						"timeStampUnixSec": "${SECAGO=3}"
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Delete this expression so we don't mess up listings for the user content one",
		`{"expressionDeleteReq":{"id": "${IDLOAD=CreatedForStat1}"}}`,
		`{"msgId":5,"status":"WS_OK", "expressionDeleteResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
