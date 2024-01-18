package main

import "github.com/pixlise/core/v4/core/wstestlib"

func testLogMsgs(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Request log level setting",
		`{"logGetLevelReq":{}}`,
		`{"msgId":1,"status":"WS_OK","logGetLevelResp":{"logLevelId": "INFO"}}`,
	)

	u1.AddSendReqAction("Change log level setting (no-perm)",
		`{"logSetLevelReq":{"logLevelId": "ERROR"}}`,
		`{"msgId":2,
			"status": "WS_NO_PERMISSION",
			"errorText": "LogSetLevelReq not allowed",
			"logSetLevelResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Request log level setting",
		`{"logGetLevelReq":{}}`,
		`{"msgId":1,"status":"WS_OK","logGetLevelResp":{"logLevelId": "INFO"}}`,
	)

	u2.AddSendReqAction("Change log level setting (invalid)",
		`{"logSetLevelReq":{"logLevelId": "VERBOSEMAX"}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST",
			"errorText": "Invalid log level name: VERBOSEMAX",
			"logSetLevelResp":{}}`,
	)

	u2.AddSendReqAction("Change log level setting",
		`{"logSetLevelReq":{"logLevelId": "ERROR"}}`,
		`{"msgId":3,"status":"WS_OK","logSetLevelResp":{"logLevelId": "ERROR"}}`,
	)

	u2.AddSendReqAction("Request log level setting again",
		`{"logGetLevelReq":{}}`,
		`{"msgId":4,"status":"WS_OK","logGetLevelResp":{"logLevelId": "ERROR"}}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)
}
