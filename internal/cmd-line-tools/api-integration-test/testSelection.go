package main

import "github.com/pixlise/core/v3/core/wstestlib"

func testSelectionMsgs(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Get selection (not exist, should return empty, no error)",
		`{"selectedScanEntriesReq":{"scanId": "abc123"}}`,
		`{"msgId":1,
			"status": "WS_OK",
			"selectedScanEntriesResp": {"entryIndexes": {}}
		}`,
	)

	u1.AddSendReqAction("Save selection (should work)",
		`{"selectedScanEntriesWriteReq":{"scanId": "abc123", "entryIndexes": {"indexes": [5,12,17,224312]}}}`,
		`{"msgId":2,
			"status": "WS_OK",
			"selectedScanEntriesWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Get selection (should return what was saved)",
		`{"selectedScanEntriesReq":{"scanId": "abc123"}}`,
		`{"msgId":3,
			"status": "WS_OK",
			"selectedScanEntriesResp": {"entryIndexes": { "indexes": [5,12,17,224312] }}
		}`,
	)

	u1.AddSendReqAction("Get pixel selection (not exist, should return empty, no error)",
		`{"selectedImagePixelsReq":{"image": "abc123"}}`,
		`{"msgId":4,
			"status": "WS_OK",
			"selectedImagePixelsResp": {"pixelIndexes": {}}
		}`,
	)

	u1.AddSendReqAction("Save selection (should work)",
		`{"selectedImagePixelsWriteReq":{"image": "abc123", "pixelIndexes": {"indexes": [58283,12343,17,886432113]}}}`,
		`{"msgId":5,
			"status": "WS_OK",
			"selectedImagePixelsWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Get selection (should return what was saved)",
		`{"selectedImagePixelsReq":{"image": "abc123"}}`,
		`{"msgId":6,
			"status": "WS_OK",
			"selectedImagePixelsResp": {"pixelIndexes": { "indexes": [58283,12343,17,886432113] }}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
