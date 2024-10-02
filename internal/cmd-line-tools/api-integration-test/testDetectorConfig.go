package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func testDetectorConfig(apiHost string) {
	// Seed the DB with a config
	configItem := protos.DetectorConfig{
		Id:              "PIXL",
		MinElement:      11,
		MaxElement:      92,
		XrfeVLowerBound: 800,
		XrfeVUpperBound: 20000,
		XrfeVResolution: 230,
		WindowElement:   14,
		TubeElement:     45,
		DefaultParams:   "",
		MmBeamRadius:    0.05999999865889549,
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.DetectorConfigsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, &configItem)
	if err != nil {
		log.Fatalln(err)
	}

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Read non existant config",
		`{"detectorConfigReq":{"id": "non-existant"}}`,
		`{"msgId":1,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant not found",
			"detectorConfigResp":{}}`,
	)

	u1.AddSendReqAction("Read existing config",
		`{"detectorConfigReq":{"id": "PIXL"}}`,
		`{"msgId":2,
			"status":"WS_OK",
			"detectorConfigResp":{
				"config":
				{
					"id": "PIXL",
					"minElement": 11,
					"maxElement": 92,
					"xrfeVLowerBound": 800,
					"xrfeVUpperBound": 20000,
					"xrfeVResolution": 230,
					"windowElement": 14,
					"tubeElement": 45,
					"mmBeamRadius": 0.06,
					"elevAngle": 70
				},
				"piquantConfigVersions": [
					"v5",
					"v6",
					"v7"
				]
			}
		}`,
	)

	u1.AddSendReqAction("Read config list",
		`{"detectorConfigListReq":{}}`,
		`{"msgId":3,
			"status":"WS_OK",
			"detectorConfigListResp":{
				"configs": [
					"PIXL"
				]
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
