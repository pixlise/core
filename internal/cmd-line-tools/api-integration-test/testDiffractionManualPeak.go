package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testDiffractionManualPeaks(apiHost string) {
	// Seed some DB data
	seedDBDiffractionManualPeak(
		[]*protos.ManualDiffractionPeak{
			{
				Id:             "069927431_2236_gb8csmu8iirzl18c",
				ScanId:         "069927431",
				Pmc:            2236,
				EnergykeV:      2.690999984741211,
				CreatedUnixSec: 1234567890,
				CreatorUserId:  "some-user-id",
			},
		},
	)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List manual diffraction peaks (non-existant should return empty result)",
		`{"diffractionPeakManualListReq":{
			"scanId": "non-existant-id"
		}}`,
		`{"msgId":1,
			"status": "WS_OK",
			"diffractionPeakManualListResp": {}
		}`,
	)

	u1.AddSendReqAction("List manual diffraction peaks (should work)",
		`{"diffractionPeakManualListReq":{
			"scanId": "069927431"
		}}`,
		`{"msgId":2,
			"status":"WS_OK",
			"diffractionPeakManualListResp": {
				"peaks": {
					"069927431_2236_gb8csmu8iirzl18c": {
						"pmc": 2236,
						"energykeV": 2.691,
						"createdUnixSec": 1234567890,
						"creatorUserId": "some-user-id"
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Add manual diffraction peak (should work)",
		`{"diffractionPeakManualInsertReq":{
			"scanId": "069927431",
			"pmc": 2132,
			"energykeV": 12.76
		}}`,
		`{"msgId":3,
			"status":"WS_OK",
			"diffractionPeakManualInsertResp": { "createdId": "${IDSAVE=createDiffractionPeakId2132}" }
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Add second manual diffraction peak (should work)",
		`{"diffractionPeakManualInsertReq":{
			"scanId": "069927431",
			"pmc": 1279,
			"energykeV": 7.886
		}}`,
		`{"msgId":4,
			"status":"WS_OK",
			"diffractionPeakManualInsertResp": { "createdId": "${IDSAVE=createDiffractionPeakId1279}" }
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Add invalid manual diffraction peak (should fail)",
		`{"diffractionPeakManualInsertReq":{
			"scanId": "069927431",
			"pmc": -3,
			"energykeV": 7.886
		}}`,
		`{"msgId":5,
			"status":"WS_BAD_REQUEST",
			"errorText": "Invalid PMC: -3",
			"diffractionPeakManualInsertResp": {}
		}`,
	)

	u1.AddSendReqAction("List manual diffraction peaks again (should work)",
		`{"diffractionPeakManualListReq":{
			"scanId": "069927431"
		}}`,
		`{"msgId":6,
			"status":"WS_OK",
			"diffractionPeakManualListResp": {
				"peaks": {
					"${IDCHK=createDiffractionPeakId1279}": {
						"pmc": 1279,
						"energykeV": 7.886,
						"createdUnixSec": "${SECAGO=3}",
						"creatorUserId": "${USERID}"
					},
					"${IDCHK=createDiffractionPeakId2132}": {
						"pmc": 2132,
						"energykeV": 12.76,
						"createdUnixSec": "${SECAGO=3}",
						"creatorUserId": "${USERID}"
					},
					"069927431_2236_gb8csmu8iirzl18c": {
						"pmc": 2236,
						"energykeV": 2.691,
						"createdUnixSec": 1234567890,
						"creatorUserId": "some-user-id"
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Delete non-existant diffraction peak (should fail)",
		`{"diffractionPeakManualDeleteReq":{
			"id": "non-existant-id"
		}}`,
		`{"msgId":7,
			"status":"WS_NOT_FOUND",
			"errorText":"non-existant-id not found",
			"diffractionPeakManualDeleteResp": {}
		}`,
	)

	u1.AddSendReqAction("Delete first manual diffraction peak (should work)",
		`{"diffractionPeakManualDeleteReq":{
			"id": "${IDLOAD=createDiffractionPeakId2132}"
		}}`,
		`{"msgId":8,
			"status":"WS_OK",
			"diffractionPeakManualDeleteResp": {}
		}`,
	)

	u1.AddSendReqAction("List manual diffraction peaks showing delete (should work)",
		`{"diffractionPeakManualListReq":{
			"scanId": "069927431"
		}}`,
		`{"msgId":9,
			"status":"WS_OK",
			"diffractionPeakManualListResp": {
				"peaks": {
					"069927431_2236_gb8csmu8iirzl18c": {
						"pmc": 2236,
						"energykeV": 2.691,
						"createdUnixSec": 1234567890,
						"creatorUserId": "some-user-id"
					},
					"${IDCHK=createDiffractionPeakId1279}": {
						"pmc": 1279,
						"energykeV": 7.886,
						"createdUnixSec": "${SECAGO=3}",
						"creatorUserId": "${USERID}"
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Add peak via user 2 should fail due to permissions
	u2 := wstestlib.MakeScriptedTestUser(auth0Params)

	u2.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("Add second manual diffraction peak (should work)",
		`{"diffractionPeakManualInsertReq":{
			"scanId": "069927431",
			"pmc": 1279,
			"energykeV": 7.886
		}}`,
		`{"msgId":1,
			"status": "WS_NO_PERMISSION",
			"errorText": "DiffractionPeakManualInsertReq not allowed",
			"diffractionPeakManualInsertResp": {}
		}`,
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

}

func seedDBDiffractionManualPeak(peaks []*protos.ManualDiffractionPeak) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.DiffractionManualPeaksName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	if len(peaks) > 0 {
		items := []interface{}{}
		for _, q := range peaks {
			items = append(items, q)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
