package main

import (
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testSelectionMsgs(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Get selection (not exist, should return empty, no error)",
		`{"selectedScanEntriesReq":{"scanIds": ["abc123"]}}`,
		`{"msgId":1,
			"status": "WS_OK",
			"selectedScanEntriesResp": {"scanIdEntryIndexes": {"abc123": {}}}
		}`,
	)

	u1.AddSendReqAction("Save selection (should work)",
		`{"selectedScanEntriesWriteReq":{
			"scanIdEntryIndexes": {
				"abc123": {"indexes": [5,12,17,224312]},
				"def456": {"indexes": [144,256]}
		}}}`,
		`{"msgId":2,
			"status": "WS_OK",
			"selectedScanEntriesWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Get selection for abc123 (should return what was saved for abc123)",
		`{"selectedScanEntriesReq":{"scanIds": ["abc123"]}}`,
		`{"msgId":3,
			"status": "WS_OK",
			"selectedScanEntriesResp": {"scanIdEntryIndexes": { "abc123": { "indexes": [5,12,17,224312] }} }
		}`,
	)

	u1.AddSendReqAction("Get selection for both (should return what was saved)",
		`{"selectedScanEntriesReq":{"scanIds": ["abc123", "def456"]}}`,
		`{"msgId":4,
			"status": "WS_OK",
			"selectedScanEntriesResp": {
				"scanIdEntryIndexes": {
					"abc123": { "indexes": [5,12,17,224312] },
					"def456": { "indexes": [144, 256] }
				}
			}
		}`,
	)

	u1.AddSendReqAction("Get selection with non-existant one (should return what was saved)",
		`{"selectedScanEntriesReq":{"scanIds": ["abc123", "def456", "eee999"]}}`,
		`{"msgId":5,
			"status": "WS_OK",
			"selectedScanEntriesResp": {
				"scanIdEntryIndexes": {
					"abc123": { "indexes": [5,12,17,224312] },
					"def456": { "indexes": [144, 256] },
					"eee999": {}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Get pixel selection (not exist, should return empty, no error)",
		`{"selectedImagePixelsReq":{"image": "abc123"}}`,
		`{"msgId":6,
			"status": "WS_OK",
			"selectedImagePixelsResp": {"pixelIndexes": {}}
		}`,
	)

	u1.AddSendReqAction("Save selection (should work)",
		`{"selectedImagePixelsWriteReq":{"image": "abc123", "pixelIndexes": {"indexes": [58283,12343,17,886432113]}}}`,
		`{"msgId":7,
			"status": "WS_OK",
			"selectedImagePixelsWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Get selection (should return what was saved)",
		`{"selectedImagePixelsReq":{"image": "abc123"}}`,
		`{"msgId":8,
			"status": "WS_OK",
			"selectedImagePixelsResp": {"pixelIndexes": { "indexes": [58283,12343,17,886432113] }}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
