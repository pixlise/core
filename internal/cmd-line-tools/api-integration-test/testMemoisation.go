package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/utils"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testMemoisation(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	key := utils.RandStringBytesMaskImpr(10)

	u1.AddSendReqAction("Request memoisation (should return not found)",
		fmt.Sprintf(`{"memoiseGetReq":{"key": "%v"}}`, key),
		fmt.Sprintf(`{"msgId":1,"status":"WS_NOT_FOUND","errorText": "%v not found", "memoiseGetResp":{}}`, key),
	)

	u1.AddSendReqAction("Write memoisation (should fail, no key))",
		`{"memoiseWriteReq":{}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST","errorText": "Key is too short", "memoiseWriteResp":{}}`,
	)

	u1.AddSendReqAction("Write memoisation (should fail, no data))",
		fmt.Sprintf(`{"memoiseWriteReq":{"key": "%v"}}`, key),
		`{"msgId":3,"status":"WS_BAD_REQUEST","errorText": "Missing data field", "memoiseWriteResp":{}}`,
	)

	u1.AddSendReqAction("Write memoisation (should succeed))",
		fmt.Sprintf(`{"memoiseWriteReq":{"key": "%v", "data": "SGVsbG8="}}`, key),
		`{"msgId":4,"status":"WS_OK","memoiseWriteResp":{ "memoTimeUnixSec": "${SECAGO=5}" }}`,
	)

	u1.AddSendReqAction("Request memoisation (should succeed)",
		fmt.Sprintf(`{"memoiseGetReq":{"key": "%v"}}`, key),
		fmt.Sprintf(`{"msgId":5,"status":"WS_OK","memoiseGetResp":{
			"item": {
				"key": "%v",
				"memoTimeUnixSec": "${SECAGO=5}",
				"data": "SGVsbG8="
			}
		}}`, key),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
