package main

import (
	"github.com/pixlise/core/v3/core/wstestlib"
)

func testROIUserConfiguration(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create ROI",
		`{"regionOfInterestWriteReq":{
			"regionOfInterest": {
				"name": "User2 ROI",
				"description": "FOR ROI TEST checking",
				"scanId": "scan1",
				"scanEntryIndexesEncoded": [0,-1,5]
			}
		}}`,
		`{"msgId":1,"status":"WS_OK",
			"regionOfInterestWriteResp":{
				"regionOfInterest": {
					"id":"${IDSAVE=ROI_USER_CONFIG_SAVED_ID}",
					"scanId": "scan1",
					"name": "User2 ROI",
					"description": "FOR ROI TEST checking",
					"scanEntryIndexesEncoded": [
						0,
						-1,
						5
					],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
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

	u1.AddSendReqAction("Update shape and colour of ROI",
		`{"regionOfInterestDisplaySettingsWriteReq":{
			"id": "${IDLOAD=ROI_USER_CONFIG_SAVED_ID}",
			"displaySettings": {
				"shape": "triangle",
				"colour": "rgba(0, 0, 255, 0.5)"
			}
		}}`,
		`{"msgId":2,"status":"WS_OK",
			"regionOfInterestDisplaySettingsWriteResp":{
				"displaySettings": {
					"shape": "triangle",
					"colour": "rgba(0, 0, 255, 0.5)"
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Ignore response ID because this is User ID + ROI ID and if shape/colour matches, we already know it's correct
	u1.AddSendReqAction("Check shape and colour of ROI",
		`{"regionOfInterestDisplaySettingsGetReq":{
			"id": "${IDLOAD=ROI_USER_CONFIG_SAVED_ID}"
		}}`,
		`{"msgId":3,"status":"WS_OK",
			"regionOfInterestDisplaySettingsGetResp":{
				"displaySettings": {
					"id": "${IGNORE}",
					"shape": "triangle",
					"colour": "rgba(0, 0, 255, 0.5)"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Check shape and colour of ROI in listing",
		`{"regionOfInterestListReq":{
			"searchParams": {
				"scanId": "scan1"
			}
		}}`,
		`{"msgId":4,"status":"WS_OK",
			"regionOfInterestListResp":{
				"regionsOfInterest": {
					"${IDCHK=ROI_USER_CONFIG_SAVED_ID}": {
						"id": "${IDCHK=ROI_USER_CONFIG_SAVED_ID}",
						"scanId": "scan1",
						"name": "User2 ROI",
						"description": "FOR ROI TEST checking",
						"modifiedUnixSec": "${SECAGO=3}",
						"displaySettings": {
							"id": "${IGNORE}",
							"shape": "triangle",
							"colour": "rgba(0, 0, 255, 0.5)"
						},
						"owner": {
							"creatorUser": {
								"id": "${USERID}",
								"name": "${REGEXMATCH=Test}",
								"email": "${REGEXMATCH=.+@pixlise\\.org}"
							},
							"createdUnixSec": "${SECAGO=3}",
							"canEdit": true
						}
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Delete ROI",
		`{"regionOfInterestDeleteReq":{"id": "${IDLOAD=ROI_USER_CONFIG_SAVED_ID}"}}`,
		`{"msgId":5,"status":"WS_OK", "regionOfInterestDeleteResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
