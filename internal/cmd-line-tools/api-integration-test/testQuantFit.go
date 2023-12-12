package main

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuantFit(apiHost string) {
	maxRunTimeSec := 60

	db := wstestlib.GetDB()
	ctx := context.TODO()
	// Seed jobs
	coll := db.Collection(dbCollections.JobStatusName)
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.JobStatusName)
	if err != nil {
		log.Fatal(err)
	}

	// Seed piquant versions
	coll = db.Collection(dbCollections.PiquantVersionName)
	err = coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.PiquantVersionName)
	if err != nil {
		log.Fatal(err)
	}
	insertResult, err := coll.InsertOne(context.TODO(), &protos.PiquantVersion{
		Id:              "current",
		Version:         "registry.gitlab.com/pixlise/piquant/runner:3.2.16",
		ModifiedUnixSec: 1234567890,
		ModifierUserId:  "user-123",
	})
	if err != nil || insertResult.InsertedID != "current" {
		panic(err)
	}

	usr := wstestlib.MakeScriptedTestUser(auth0Params)
	usr.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	exp64 := b64.StdEncoding.EncodeToString([]byte("   PIQUANT 3.2.16-master  Normal_Combined_AllPoints\nEnergy (keV), meas, calc, bkg, sigma, residual, DetCE, Ti_K, Ca_K, Rh_K_coh, Rh_L_coh, Rh_K_inc, Pileup, Rh_L_coh_Lb1\n-0.0154067, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n-0.0075045, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n0.000397676, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0"))

	// Snip off any garbage at the end
	for c := 0; c < 2; c++ {
		if strings.HasSuffix(exp64, "=") {
			exp64 = exp64[0 : len(exp64)-1]
		}
	}

	usr.AddSendReqAction("Create quant",
		`{"quantCreateReq":{
			"params": {
				"command": "quant",
				"scanId": "983561",
				"pmcs": [68, 69, 70, 71, 72, 73, 74, 75],
				"elements": ["Ca", "Ti"],
				"detectorConfig": "PIXL/v5",
				"parameters": "-Fe,1",
				"runTimeSec": 60,
				"quantMode": "Combined",
				"roiIDs": []
			}
		}}`,
		fmt.Sprintf(`{"msgId":1,"status":"WS_OK","quantCreateResp":{
			"resultData": "${REGEXMATCH=%v.+}"
		}}`, exp64),
	)

	// NOTE: we don't expect to get job update messages for these, they're "one-shot", where we get the data back in the response!
	usr.CloseActionGroup([]string{}, maxRunTimeSec*1000)

	wstestlib.ExecQueuedActions(&usr)
}
