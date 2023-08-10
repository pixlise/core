package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuants(apiHost string) {
	scanId := "the-scan-id"
	quantId := "3vjoovnrhkhv8ecd"

	seedDBQuants([]*protos.QuantificationSummary{
		{
			Id: quantId,
			Params: &protos.QuantStartingParametersWithPMCCount{
				Params: &protos.QuantStartingParameters{
					Name:              "Trial quant with Rh",
					DataBucket:        "databucket",
					DatasetPath:       "Datasets/" + scanId + "/dataset.bin",
					DatasetID:         scanId,
					PiquantJobsBucket: "piquantbucket",
					DetectorConfig:    "PIXL/PiquantConfigs/v7",
					Elements: []string{
						"CO3",
						"Rh",
						"Na",
						"Mg",
						"Al",
						"Si",
						"P",
						"S",
						"Cl",
						"K",
						"Ca",
						"Ti",
						"Cr",
						"Mn",
						"Fe",
					},
					Parameters:       "",
					RunTimeSec:       60,
					CoresPerNode:     4,
					StartUnixTimeSec: 1652813392,
					RequestorUserId:  "auth0|5df311ed8a0b5d0ebf5fb476",
					RoiID:            "wob0wm8cogiot1rp",
					ElementSetID:     "",
					PIQUANTVersion:   "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
					QuantMode:        "AB",
					Comments:         "",
					RoiIDs:           []string{},
					IncludeDwells:    false,
					Command:          "",
				},
			},
			Elements: []string{
				"Rh2O3",
				"Na2O",
				"MgCO3",
				"Al2O3",
				"SiO2",
				"P2O5",
				"SO3",
				"Cl",
				"K2O",
				"CaCO3",
				"TiO2",
				"Cr2O3",
				"MnCO3",
				"FeCO3-T",
			},
			Status: &protos.JobStatus{
				JobID:          quantId,
				Status:         5,
				Message:        "Nodes ran: 7",
				EndUnixTimeSec: 1652813627,
				OutputFilePath: "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
				PiquantLogs: []string{
					"node00001_piquant.log",
					"node00001_stdout.log",
					"node00002_piquant.log",
					"node00002_stdout.log",
					"node00003_piquant.log",
					"node00003_stdout.log",
					"node00004_piquant.log",
					"node00004_stdout.log",
					"node00005_piquant.log",
					"node00005_stdout.log",
					"node00006_piquant.log",
					"node00006_stdout.log",
					"node00007_piquant.log",
					"node00007_stdout.log",
				},
			},
		},
	})

	seedDBOwnership(quantId, protos.ObjectType_OT_QUANTIFICATION, nil, nil)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List quants",
		`{"quantListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","quantListResp":{}}`,
	)

	u1.AddSendReqAction("Get with missing ID",
		`{"quantGetReq":{}}`,
		`{"msgId":2,"status":"WS_NOT_FOUND","errorText": " not found", "quantGetResp":{}}`,
	)

	u1.AddSendReqAction("Get non-existant quant",
		`{"quantGetReq":{"quantId": "non-existant-id"}}`,
		`{"msgId":3,"status":"WS_NOT_FOUND","errorText": "non-existant-id not found", "quantGetResp":{}}`,
	)

	u1.AddSendReqAction("Get quant from db (should fail, permissions dont allow)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v"}}`, quantId),
		fmt.Sprintf(`{
			"msgId":4,"status":"WS_NO_PERMISSION",
			"errorText": "View access denied for: %v", "quantGetResp":{}}`, quantId),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Now add u1 as a viewer
	seedDBOwnership(quantId, protos.ObjectType_OT_QUANTIFICATION, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}}, nil)

	u1.AddSendReqAction("List quants",
		`{"quantListReq":{}}`,
		fmt.Sprintf(`{"msgId": 5, "status": "WS_OK", "quantListResp": {
			"quants": [{
				"id": "%v",
				"params": {
					"params": {
						"name": "Trial quant with Rh",
						"dataBucket": "databucket",
						"datasetPath": "Datasets/the-scan-id/dataset.bin",
						"datasetID": "the-scan-id",
						"piquantJobsBucket": "piquantbucket",
						"detectorConfig": "PIXL/PiquantConfigs/v7",
						"elements": ["CO3","Rh","Na","Mg","Al","Si","P","S","Cl","K","Ca","Ti","Cr","Mn","Fe"],
						"runTimeSec": 60,
						"coresPerNode": 4,
						"startUnixTimeSec": 1652813392,
						"requestorUserId": "auth0|5df311ed8a0b5d0ebf5fb476",
						"roiID": "wob0wm8cogiot1rp",
						"PIQUANTVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
						"quantMode": "AB"
					}
				},
				"elements": ["Rh2O3","Na2O","MgCO3","Al2O3","SiO2","P2O5","SO3","Cl","K2O","CaCO3","TiO2","Cr2O3","MnCO3","FeCO3-T"],
				"status": {
					"jobID": "%v",
					"status": "COMPLETE",
					"message": "Nodes ran: 7",
					"endUnixTimeSec": 1652813627,
					"outputFilePath": "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
					"piquantLogs": ["node00001_piquant.log","node00001_stdout.log","node00002_piquant.log","node00002_stdout.log","node00003_piquant.log","node00003_stdout.log","node00004_piquant.log","node00004_stdout.log","node00005_piquant.log","node00005_stdout.log","node00006_piquant.log","node00006_stdout.log","node00007_piquant.log","node00007_stdout.log"]
				}
			}]
		}}`, quantId, quantId),
	)

	u1.AddSendReqAction("Get quant (should work)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v"}}`, quantId),
		fmt.Sprintf(`{"msgId":6,"status":"WS_OK",
			"quantGetResp":{
				"summary": {
					"id": "%v",
				"params": {
					"params": {
						"name": "Trial quant with Rh",
						"dataBucket": "databucket",
						"datasetPath": "Datasets/the-scan-id/dataset.bin",
						"datasetID": "the-scan-id",
						"piquantJobsBucket": "piquantbucket",
						"detectorConfig": "PIXL/PiquantConfigs/v7",
						"elements": ["CO3","Rh","Na","Mg","Al","Si","P","S","Cl","K","Ca","Ti","Cr","Mn","Fe"],
						"runTimeSec": 60,
						"coresPerNode": 4,
						"startUnixTimeSec": 1652813392,
						"requestorUserId": "auth0|5df311ed8a0b5d0ebf5fb476",
						"roiID": "wob0wm8cogiot1rp",
						"PIQUANTVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
						"quantMode": "AB"
					}
				},
				"elements": ["Rh2O3","Na2O","MgCO3","Al2O3","SiO2","P2O5","SO3","Cl","K2O","CaCO3","TiO2","Cr2O3","MnCO3","FeCO3-T"],
				"status": {
					"jobID": "%v",
					"status": "COMPLETE",
					"message": "Nodes ran: 7",
					"endUnixTimeSec": 1652813627,
					"outputFilePath": "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
					"piquantLogs": ["node00001_piquant.log","node00001_stdout.log","node00002_piquant.log","node00002_stdout.log","node00003_piquant.log","node00003_stdout.log","node00004_piquant.log","node00004_stdout.log","node00005_piquant.log","node00005_stdout.log","node00006_piquant.log","node00006_stdout.log","node00007_piquant.log","node00007_stdout.log"]
				}
			}
		}}`, quantId, quantId),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func seedDBQuants(quants []*protos.QuantificationSummary) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.QuantificationsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	if len(quants) > 0 {
		items := []interface{}{}
		for _, q := range quants {
			items = append(items, q)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
