package main

import "github.com/pixlise/core/v3/core/wstestlib"

func testPiquantMsgs(apiHost string) {
	testPiquantNotAllowedMsgs(apiHost)
	u2 := testPiquantVersionAllowedMsgs(apiHost)
	testPiquantConfigAllowedMsgs(u2)
}

func testPiquantNotAllowedMsgs(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Get current piquant version (no perm)",
		`{"piquantCurrentVersionReq":{}}`,
		`{"msgId":1,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantCurrentVersionReq not allowed",
			"piquantCurrentVersionResp": {}
		}`,
	)

	u1.AddSendReqAction("Set current piquant version (no perm)",
		`{"piquantWriteCurrentVersionReq":{"piquantVersion": "registry.gitlab.com/pixlise/piquant/more/runner:2.3.4"}}`,
		`{"msgId":2,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantWriteCurrentVersionReq not allowed",
			"piquantWriteCurrentVersionResp":{}
		}`,
	)

	u1.AddSendReqAction("piquantConfigListReq (no perm)",
		`{"piquantConfigListReq":{}}`,
		`{"msgId":3,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantConfigListReq not allowed",
			"piquantConfigListResp":{}
		}`,
	)

	u1.AddSendReqAction("piquantConfigVersionsListReq (no perm)",
		`{"piquantConfigVersionsListReq":{}}`,
		`{"msgId":4,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantConfigVersionsListReq not allowed",
			"piquantConfigVersionsListResp":{}
		}`,
	)

	u1.AddSendReqAction("piquantConfigVersionReq (no perm)",
		`{"piquantConfigVersionReq":{}}`,
		`{"msgId":5,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantConfigVersionReq not allowed",
			"piquantConfigVersionResp":{}
		}`,
	)

	u1.AddSendReqAction("piquantVersionListReq (no perm)",
		`{"piquantVersionListReq":{}}`,
		`{"msgId":6,
			"status": "WS_NO_PERMISSION",
			"errorText": "PiquantVersionListReq not allowed",
			"piquantVersionListResp":{}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func testPiquantVersionAllowedMsgs(apiHost string) wstestlib.ScriptedTestUser {
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Get current piquant version (should fail)",
		`{"piquantCurrentVersionReq":{}}`,
		`{"msgId":1,
			"status": "WS_NOT_FOUND",
			"errorText": "PIQUANT version not found",
			"piquantCurrentVersionResp":{}
		}`,
	)

	u2.AddSendReqAction("Set current piquant version (should fail)",
		`{"piquantWriteCurrentVersionReq":{"piquantVersion": "registry.gitlab.com/pixlise/piquant/more/indirection/to-create-some-namethats_way_toolong.tobe_valid:2.3.4"}}`,
		`{"msgId":2,
			"status": "WS_BAD_REQUEST",
			"errorText": "PiquantVersion is too long",
			"piquantWriteCurrentVersionResp":{
			}
		}`,
	)

	u2.AddSendReqAction("Set current piquant version (should work)",
		`{"piquantWriteCurrentVersionReq":{"piquantVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.1221"}}`,
		`{"msgId":3,"status":"WS_OK",
			"piquantWriteCurrentVersionResp":{
			}
		}`,
	)

	u2.AddSendReqAction("Get current piquant version (should work)",
		`{"piquantCurrentVersionReq":{}}`,
		`{"msgId":4,"status":"WS_OK",
			"piquantCurrentVersionResp":{
				"piquantVersion": {
					"id": "current",
					"version": "registry.gitlab.com/pixlise/piquant/runner:3.2.1221",
					"modifiedUnixSec": "${SECAGO=3}",
					"modifierUserId": "${USERID}"
				}
			}
		}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

	return u2
}

func testPiquantConfigAllowedMsgs(u2 wstestlib.ScriptedTestUser) {
	u2.AddSendReqAction("piquantConfigListReq (should return whats in S3 file structure)",
		`{"piquantConfigListReq":{}}`,
		`{"msgId":5,
			"status": "WS_OK",
			"piquantConfigListResp": {
				"configNames${LIST,MODE=CONTAINS,MINLENGTH=2}": [
					"Breadboard",
					"PIXL"
				]
			}
		}`,
	)

	u2.AddSendReqAction("piquantConfigVersionsListReq (should fail, no config id)",
		`{"piquantConfigVersionsListReq":{}}`,
		`{"msgId":6,
			"status": "WS_BAD_REQUEST",
			"errorText": "ConfigId is too short",
			"piquantConfigVersionsListResp":{}
		}`,
	)

	u2.AddSendReqAction("piquantConfigVersionsListReq (should work)",
		`{"piquantConfigVersionsListReq":{"configId": "Breadboard"}}`,
		`{"msgId":7,
			"status": "WS_OK",
			"piquantConfigVersionsListResp":{
				"versions${LIST,MODE=CONTAINS,MINLENGTH=3}": ["v1", "v2"]
			}
		}`,
	)

	u2.AddSendReqAction("piquantConfigVersionReq (should fail, no version)",
		`{"piquantConfigVersionReq":{}}`,
		`{"msgId":8,
			"status": "WS_BAD_REQUEST",
			"errorText": "Version is too short",
			"piquantConfigVersionResp":{}
		}`,
	)

	u2.AddSendReqAction("piquantConfigVersionReq (should fail, no config)",
		`{"piquantConfigVersionReq":{"version": "2"}}`,
		`{"msgId":9,
			"status": "WS_BAD_REQUEST",
			"errorText": "ConfigId is too short",
			"piquantConfigVersionResp":{}
		}`,
	)

	u2.AddSendReqAction("piquantConfigVersionReq (should work)",
		`{"piquantConfigVersionReq":{"configId": "Breadboard", "version": "v2"}}`,
		`{"msgId":10,
			"status": "WS_OK",
			"piquantConfigVersionResp":{
				"piquantConfig": {
					"description": "Breadboard config Apr2022",
					"configFile": "Configuration_BB_2019_Teflon_02_04_2022.msa",
					"opticEfficiencyFile": "BB_2019_Efficiency_Teflon_low_E_revised_02_04_2022.txt",
					"calibrationFile": "BB_2019_ECF_file_02_18_2022.txt"
				}
			}
		}`,
	)
	/*
		u2.AddSendReqAction("piquantVersionListReq (no perm)",
			`{"piquantVersionListReq":{}}`,
			`{"msgId":11,
				"status": "WS_OK",
				"piquantVersionListResp":{}
			}`,
		)
	*/
	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)
}
