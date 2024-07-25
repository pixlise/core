package main

import (
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testScreenConfiguration(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create a screen configuration (empty request, fails)",
		`{"screenConfigurationWriteReq":{}}`,
		`{
			"msgId": 1,
			"status": "WS_SERVER_ERROR",
			"errorText": "screen configuration must be specified",
			"screenConfigurationWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Create a screen configuration (no layouts, fails)",
		`{"screenConfigurationWriteReq":{"screenConfiguration":{"layouts": []}}}`,
		`{
			"msgId": 2,
			"status": "WS_SERVER_ERROR",
			"errorText": "screen configuration must have at least one layout",
			"screenConfigurationWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("Create a screen configuration (success)",
		`{
			"screenConfigurationWriteReq":{
				"screenConfiguration":{
					"id": "",
					"name": "Novarupta 2",
					"description": "",
					"tags": [],
					"modifiedUnixSec": 0,
					"scanConfigurations": {},
					"layouts": [
						{
							"rows": [ { "height": 3 }, { "height": 2 } ],
							"columns": [ { "width": 3 }, { "width": 2 }, { "width": 2 }, { "width": 2 } ],
							"widgets": [
								{
									"id": "",
									"type": "binary-plot",
									"startRow": 1,
									"startColumn": 1,
									"endRow": 2,
									"endColumn": 2
								},
								{
									"id": "",
									"type": "spectrum-chart",
									"startRow": 1,
									"startColumn": 2,
									"endRow": 2,
									"endColumn": 5
								},
								{
									"id": "",
									"type": "histogram",
									"startRow": 2,
									"startColumn": 1,
									"endRow": 3,
									"endColumn": 2
								},
								{
									"id": "",
									"type": "chord-diagram",
									"startRow": 2,
									"startColumn": 2,
									"endRow": 3,
									"endColumn": 3
								},
								{
									"id": "",
									"type": "ternary-plot",
									"startRow": 2,
									"startColumn": 3,
									"endRow": 3,
									"endColumn": 4
								},
								{
									"id": "",
									"type": "binary-plot",
									"startRow": 2,
									"startColumn": 4,
									"endRow": 3,
									"endColumn": 5
								}
							]
						}
					]
				}
			}
		}`,
		`{
			"msgId": 3,
			"status": "WS_OK",
			"screenConfigurationWriteResp": {
				"screenConfiguration": {
					"id": "${IDSAVE=SCREEN_CONFIG_SAVED_ID}",
					"name": "Novarupta 2",
					"layouts": [
						{
							"tabId": "${IGNORE}",
							"tabName": "Tab 1",
							"rows": [
								{
									"height": 3
								},
								{
									"height": 2
								}
							],
							"columns": [
								{
									"width": 3
								},
								{
									"width": 2
								},
								{
									"width": 2
								},
								{
									"width": 2
								}
							],
							"widgets": [
								{
									"id": "${IGNORE}",
									"type": "binary-plot",
									"startRow": 1,
									"startColumn": 1,
									"endRow": 2,
									"endColumn": 2,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "spectrum-chart",
									"startRow": 1,
									"startColumn": 2,
									"endRow": 2,
									"endColumn": 5,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "histogram",
									"startRow": 2,
									"startColumn": 1,
									"endRow": 3,
									"endColumn": 2,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "chord-diagram",
									"startRow": 2,
									"startColumn": 2,
									"endRow": 3,
									"endColumn": 3,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "ternary-plot",
									"startRow": 2,
									"startColumn": 3,
									"endRow": 3,
									"endColumn": 4,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "binary-plot",
									"startRow": 2,
									"startColumn": 4,
									"endRow": 3,
									"endColumn": 5,
									"data": {
										"id": "${IGNORE}"
									}
								}
							]
						}
					],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${IGNORE}",
							"email": "test1@pixlise.org"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Get the created screen configuration",
		`{"screenConfigurationGetReq":{"id":"${IDLOAD=SCREEN_CONFIG_SAVED_ID}"}}`,
		`{
			"msgId": 4,
			"status": "WS_OK",
			"screenConfigurationGetResp": {
				"screenConfiguration": {
					"id": "${IDCHK=SCREEN_CONFIG_SAVED_ID}",
					"name": "Novarupta 2",
					"layouts": [
						{
							"tabId": "${IGNORE}",
							"tabName": "Tab 1",
							"rows": [
								{
									"height": 3
								},
								{
									"height": 2
								}
							],
							"columns": [
								{
									"width": 3
								},
								{
									"width": 2
								},
								{
									"width": 2
								},
								{
									"width": 2
								}
							],
							"widgets": [
								{
									"id": "${IGNORE}",
									"type": "binary-plot",
									"startRow": 1,
									"startColumn": 1,
									"endRow": 2,
									"endColumn": 2,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "spectrum-chart",
									"startRow": 1,
									"startColumn": 2,
									"endRow": 2,
									"endColumn": 5,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "histogram",
									"startRow": 2,
									"startColumn": 1,
									"endRow": 3,
									"endColumn": 2,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "chord-diagram",
									"startRow": 2,
									"startColumn": 2,
									"endRow": 3,
									"endColumn": 3,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "ternary-plot",
									"startRow": 2,
									"startColumn": 3,
									"endRow": 3,
									"endColumn": 4,
									"data": {
										"id": "${IGNORE}"
									}
								},
								{
									"id": "${IGNORE}",
									"type": "binary-plot",
									"startRow": 2,
									"startColumn": 4,
									"endRow": 3,
									"endColumn": 5,
									"data": {
										"id": "${IGNORE}"
									}
								}
							]
						}
					],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${IGNORE}",
							"email": "test1@pixlise.org"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Delete the created screen configuration",
		`{"screenConfigurationDeleteReq":{"id":"${IDLOAD=SCREEN_CONFIG_SAVED_ID}"}}`,
		`{
			"msgId": 5,
			"status": "WS_OK",
			"screenConfigurationDeleteResp": {"id": "${IDCHK=SCREEN_CONFIG_SAVED_ID}"}
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

	u2.AddSendReqAction("Create a screen configuration (not allowed)",
		`{
			"screenConfigurationWriteReq":{
				"screenConfiguration":{
					"id": "",
					"name": "Novarupta 2",
					"description": "",
					"tags": [],
					"modifiedUnixSec": 0,
					"scanConfigurations": {},
					"layouts": [
						{
							"rows": [ { "height": 3 }, { "height": 2 } ],
							"columns": [ { "width": 3 }, { "width": 2 }, { "width": 2 }, { "width": 2 } ],
							"widgets": [
								{
									"id": "",
									"type": "binary-plot",
									"startRow": 1,
									"startColumn": 1,
									"endRow": 2,
									"endColumn": 2
								},
								{
									"id": "",
									"type": "spectrum-chart",
									"startRow": 1,
									"startColumn": 2,
									"endRow": 2,
									"endColumn": 5
								},
								{
									"id": "",
									"type": "histogram",
									"startRow": 2,
									"startColumn": 1,
									"endRow": 3,
									"endColumn": 2
								},
								{
									"id": "",
									"type": "chord-diagram",
									"startRow": 2,
									"startColumn": 2,
									"endRow": 3,
									"endColumn": 3
								},
								{
									"id": "",
									"type": "ternary-plot",
									"startRow": 2,
									"startColumn": 3,
									"endRow": 3,
									"endColumn": 4
								},
								{
									"id": "",
									"type": "binary-plot",
									"startRow": 2,
									"startColumn": 4,
									"endRow": 3,
									"endColumn": 5
								}
							]
						}
					]
				}
			}
		}`,
		`{
			"msgId": 1,
    		"status": "WS_NO_PERMISSION",
    		"errorText": "ScreenConfigurationWriteReq not allowed",
			"screenConfigurationWriteResp": {}
		}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

}
