package main

import (
	"github.com/pixlise/core/v3/core/wstestlib"
)

func testWidgetData(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

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
			"msgId": 1,
			"status": "WS_OK",
			"screenConfigurationWriteResp": {
				"screenConfiguration": {
					"id": "${IDSAVE=WIDGET_SCREEN_CONFIG}",
					"name": "Novarupta 2",
					"layouts": [
						{
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
									"id": "${IDSAVE=TERNARY_PLOT_ID}",
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

	u1.AddSendReqAction("Modify a widget (success)",
		`{
			"widgetDataWriteReq":{
				"widgetData": {
					"id": "${IDLOAD=TERNARY_PLOT_ID}",
					"ternary": {
						"showMmol": false,
						"expressionIDs": [
							"test-expression-1",
							"test-expression-2",
							"test-expression-3"
						],
						"visibleROIs": [
							{
								"id": "AllPoints-198509061",
								"scanId": "198509061"
							}
						]
					}
				}
			}
		}`,
		`{
			"msgId": 2,
			"status": "WS_OK",
			"widgetDataWriteResp": {
				"widgetData": {
					"id": "${IDCHK=TERNARY_PLOT_ID}",
					"ternary": {
						"expressionIDs": [
							"test-expression-1",
							"test-expression-2",
							"test-expression-3"
						],
						"visibleROIs": [
							{
								"id": "AllPoints-198509061",
								"scanId": "198509061"
							}
						]
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Get modified widget",
		`{
			"widgetDataGetReq":{
				"id": "${IDLOAD=TERNARY_PLOT_ID}"
			}
		}`,
		`{
			"msgId": 3,
			"status": "WS_OK",
			"widgetDataGetResp": {
				"widgetData": {
					"id": "${IDCHK=TERNARY_PLOT_ID}",
					"ternary": {
						"expressionIDs": [
							"test-expression-1",
							"test-expression-2",
							"test-expression-3"
						],
						"visibleROIs": [
							{
								"id": "AllPoints-198509061",
								"scanId": "198509061"
							}
						]
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
