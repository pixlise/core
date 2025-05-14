package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func testDiffractionStatus(apiHost string) {
	// Seed some DB data
	seedDBDiffractionStatuses(
		[]*protos.DetectedDiffractionPeakStatuses{
			{
				Id:     "176882177",
				ScanId: "176882177",
				Statuses: map[string]*protos.DetectedDiffractionPeakStatuses_PeakStatus{
					"1180-364": {Status: "not-anomaly", CreatedUnixSec: 1234567890, CreatorUserId: "some-user-id"},
					"1473-878": {Status: "not-anomaly"},
				},
			},
		},
	)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List statuses (non-existant should return empty result)",
		`{"diffractionPeakStatusListReq":{
			"scanId": "non-existant-id"
		}}`,
		`{"msgId":1,
			"status": "WS_OK",
			"diffractionPeakStatusListResp": {
				"peakStatuses": {
					"id": "non-existant-id",
					"scanId": "non-existant-id"
				}
			}
		}`,
	)

	u1.AddSendReqAction("List statuses",
		`{"diffractionPeakStatusListReq":{
			"scanId": "176882177"
		}}`,
		`{"msgId":2,
			"status":"WS_OK",
			"diffractionPeakStatusListResp": {
				"peakStatuses": {
					"id": "176882177",
					"scanId": "176882177",
					"statuses": {
						"1180-364": {
							"status": "not-anomaly",
							"createdUnixSec": 1234567890,
							"creatorUserId": "some-user-id"
						},
						"1473-878": {
							"status": "not-anomaly"
						}
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Set status",
		`{"diffractionPeakStatusWriteReq":{
			"scanId": "444",
			"diffractionPeakId": "555-560",
			"status": "something"
		}}`,
		`{"msgId":3,
			"status": "WS_OK",
			"diffractionPeakStatusWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("List statuses",
		`{"diffractionPeakStatusListReq":{
			"scanId": "444"
		}}`,
		`{"msgId":4,
			"status":"WS_OK",
			"diffractionPeakStatusListResp": {
				"peakStatuses": {
					"id": "444",
					"scanId": "444",
					"statuses": {
						"555-560": {
							"status": "something",
							"createdUnixSec": "${SECAGO=3}",
							"creatorUserId": "${USERID}"
						}
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Set status 2",
		`{"diffractionPeakStatusWriteReq":{
			"scanId": "444",
			"diffractionPeakId": "555-130",
			"status": "another"
		}}`,
		`{"msgId":5,
			"status": "WS_OK",
			"diffractionPeakStatusWriteResp": {}
		}`,
	)

	u1.AddSendReqAction("List statuses",
		`{"diffractionPeakStatusListReq":{
			"scanId": "444"
		}}`,
		`{"msgId":6,
			"status":"WS_OK",
			"diffractionPeakStatusListResp": {
				"peakStatuses": {
					"id": "444",
					"scanId": "444",
					"statuses": {
						"555-130": {
							"status": "another",
							"createdUnixSec": "${SECAGO=3}",
							"creatorUserId": "${USERID}"
						},
						"555-560": {
							"status": "something",
							"createdUnixSec": "${SECAGO=3}",
							"creatorUserId": "${USERID}"
						}
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Delete non-existant status",
		`{"diffractionPeakStatusDeleteReq":{
			"scanId": "non-existant-id",
			"diffractionPeakId": "333-444"
		}}`,
		`{"msgId":7,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant-id.333-444 not found",
			"diffractionPeakStatusDeleteResp": {}
		}`,
	)

	u1.AddSendReqAction("Delete created status",
		`{"diffractionPeakStatusDeleteReq":{
			"scanId": "444",
			"diffractionPeakId": "333-444"
		}}`,
		`{"msgId":8,
			"status": "WS_NOT_FOUND",
			"errorText": "444.333-444 not found",
			"diffractionPeakStatusDeleteResp": {}
		}`,
	)

	u1.AddSendReqAction("Delete created status",
		`{"diffractionPeakStatusDeleteReq":{
			"scanId": "444",
			"diffractionPeakId": "555-560"
		}}`,
		`{"msgId":9,
			"status": "WS_OK",
			"diffractionPeakStatusDeleteResp": {}
		}`,
	)

	u1.AddSendReqAction("List statuses",
		`{"diffractionPeakStatusListReq":{
			"scanId": "444"
		}}`,
		`{"msgId":10,
			"status":"WS_OK",
			"diffractionPeakStatusListResp": {
				"peakStatuses": {
					"id": "444",
					"scanId": "444",
					"statuses": {
						"555-130": {
							"status": "another",
							"createdUnixSec": "${SECAGO=3}",
							"creatorUserId": "${USERID}"
						}
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Edit via user 2 should fail due to permissions
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)
	u2.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Set status (no perms)",
		`{"diffractionPeakStatusWriteReq":{
		"scanId": "444",
		"diffractionPeakId": "555-560"
	}}`,
		`{"msgId":1,
		"status": "WS_NO_PERMISSION",
		"errorText": "DiffractionPeakStatusWriteReq not allowed",
		"diffractionPeakStatusWriteResp": {}
	}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)
}

func seedDBDiffractionStatuses(statusItems []*protos.DetectedDiffractionPeakStatuses) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.DiffractionDetectedPeakStatusesName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	if len(statusItems) > 0 {
		items := []interface{}{}
		for _, q := range statusItems {
			items = append(items, q)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
