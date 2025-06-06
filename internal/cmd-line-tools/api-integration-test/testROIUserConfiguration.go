package main

import (
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testROIUserConfiguration(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
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
		`{"msgId":5,"status":"WS_OK", "regionOfInterestDeleteResp":{"deletedIds": ["${IDCHK=ROI_USER_CONFIG_SAVED_ID}"]}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func testNormalROIBulkWrite(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create Bulk ROIs",
		`{"regionOfInterestBulkWriteReq":{
			"regionsOfInterest": [
				{
					"name": "ROI 1",
					"scanId": "scan1",
					"scanEntryIndexesEncoded": [0,-1,5]
				},
				{
					"name": "ROI 2",
					"scanId": "scan1",
					"scanEntryIndexesEncoded": [7,-1,10]
				},
				{
					"name": "ROI 3",
					"scanId": "scan1",
					"scanEntryIndexesEncoded": [13,-1,15]
				}
			]
		}}`,
		`{"msgId":1,"status":"WS_OK",
			"regionOfInterestBulkWriteResp":{
				"regionsOfInterest": [
				{
					"id":"${IDSAVE=BULK_ROI_1}",
					"scanId": "scan1",
					"name": "ROI 1",
					"scanEntryIndexesEncoded": [0,-1,5],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
        			"associatedROIId": "${IGNORE}"
				},
				{
					"id":"${IDSAVE=BULK_ROI_2}",
					"scanId": "scan1",
					"name": "ROI 2",
					"scanEntryIndexesEncoded": [7,-1,10],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
        			"associatedROIId": "${IGNORE}"
				},
				{
					"id":"${IDSAVE=BULK_ROI_3}",
					"scanId": "scan1",
					"name": "ROI 3",
					"scanEntryIndexesEncoded": [13,-1,15],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
        			"associatedROIId": "${IGNORE}"
				}]
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Can't use regionOfInterestListReq because it returns a map and the order doesn't stay constant
	// so here we check the individual 3 ROIs created

	u1.AddSendReqAction("Get Created ROI 1",
		`{"regionOfInterestGetReq":{"id": "${IDLOAD=BULK_ROI_1}"}}`,
		`{"msgId":2,"status":"WS_OK",
			"regionOfInterestGetResp":{
				"regionOfInterest": {
					"id":"${IDCHK=BULK_ROI_1}",
					"scanId": "scan1",
					"name": "ROI 1",
					"scanEntryIndexesEncoded": [0,-1,5],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"associatedROIId": "${IDCHK=BULK_ROI_1}"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Get Created ROI 2",
		`{"regionOfInterestGetReq":{"id": "${IDLOAD=BULK_ROI_2}"}}`,
		`{"msgId":3,"status":"WS_OK",
			"regionOfInterestGetResp":{
				"regionOfInterest": {
					"id":"${IDCHK=BULK_ROI_2}",
					"scanId": "scan1",
					"name": "ROI 2",
					"scanEntryIndexesEncoded": [7,-1,10],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"associatedROIId": "${IDCHK=BULK_ROI_1}"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Get Created ROI 3",
		`{"regionOfInterestGetReq":{"id": "${IDLOAD=BULK_ROI_3}"}}`,
		`{"msgId":4,"status":"WS_OK",
			"regionOfInterestGetResp":{
				"regionOfInterest": {
					"id":"${IDCHK=BULK_ROI_3}",
					"scanId": "scan1",
					"name": "ROI 3",
					"scanEntryIndexesEncoded": [13,-1,15],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"associatedROIId": "${IDCHK=BULK_ROI_1}"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Delete Second ROI",
		`{"regionOfInterestDeleteReq":{"id": "${IDLOAD=BULK_ROI_2}", "isMIST": false, "isAssociatedROIId": false}}`,
		`{"msgId":5,"status":"WS_OK",
			"regionOfInterestDeleteResp":{
				"deletedIds": ["${IDCHK=BULK_ROI_2}"]
		}}`,
	)

	u1.AddSendReqAction("Get Created ROI 2 (should fail)",
		`{"regionOfInterestGetReq":{"id": "${IDLOAD=BULK_ROI_2}"}}`,
		`{"msgId":6,"status":"WS_NOT_FOUND",
  			"errorText": "${IGNORE}",
  			"regionOfInterestGetResp": {}}`,
	)

	u1.AddSendReqAction("Get Created ROI 3",
		`{"regionOfInterestGetReq":{"id": "${IDLOAD=BULK_ROI_3}"}}`,
		`{"msgId":7,"status":"WS_OK",
			"regionOfInterestGetResp":{
				"regionOfInterest": {
					"id":"${IDCHK=BULK_ROI_3}",
					"scanId": "scan1",
					"name": "ROI 3",
					"scanEntryIndexesEncoded": [13,-1,15],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=Test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"associatedROIId": "${IDCHK=BULK_ROI_1}"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Delete Associated ROIs",
		`{"regionOfInterestDeleteReq":{"id": "${IDLOAD=BULK_ROI_1}", "isMIST": false, "isAssociatedROIId": true}}`,
		`{"msgId":8,"status":"WS_OK",
			"regionOfInterestDeleteResp":{
				"deletedIds": ["${IDCHK=BULK_ROI_1}", "${IDCHK=BULK_ROI_3}"]
		}}`,
	)

	u1.AddSendReqAction("List Created ROIs, should be empty",
		`{"regionOfInterestListReq":{}}`,
		`{"msgId":9,"status":"WS_OK",
			"regionOfInterestListResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
