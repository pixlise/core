package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func testMultiQuant(apiHost string) {
	resetDBPiquantAndJobs()

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.OwnershipName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.CreateCollection(ctx, dbCollections.OwnershipName)
	if err != nil {
		log.Fatal(err)
	}
	coll = db.Collection(dbCollections.RegionsOfInterestName)
	err = coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.CreateCollection(ctx, dbCollections.RegionsOfInterestName)
	if err != nil {
		log.Fatal(err)
	}

	scanId := seedDBScanData(scan_Beaujeu)
	seedS3File("sol138.bin", filepaths.GetScanFilePath(scanId, filepaths.DatasetFileName), apiDatasetBucket)
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Ensure empty
	u1.AddSendReqAction("List quants",
		`{"quantListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","quantListResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, empty name",
		`{"quantCombineReq":{
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				}
			]
		}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST","errorText":"Name cannot be empty","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, should ref more than 1 (supplied 0)",
		`{"quantCombineReq":{
			"name": "here's a name",
			"description": "combined quants",
			"roiZStack": []
		}}`,
		`{"msgId":3,"status":"WS_BAD_REQUEST","errorText":"Must reference more than 1 ROI","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, should ref more than 1 (supplied 1)",
		`{"quantCombineReq":{
			"name": "here's a name",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				}
			]
		}}`,
		`{"msgId":4,"status":"WS_BAD_REQUEST","errorText":"Must reference more than 1 ROI","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, missing scan id",
		`{"quantCombineReq":{
			"name": "here's a name",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-456"
				}
			]
		}}`,
		`{"msgId":5,"status":"WS_BAD_REQUEST","errorText":"ScanId is too short","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, cant load scan",
		`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "non-existant-scan",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-456"
				}
			]
		}}`,
		`{"msgId":6,"status":"WS_NOT_FOUND","errorText":"non-existant-scan not found","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant - should fail, scan access denied",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%s",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-456"
				}
			]
		}}`, scanId),
		fmt.Sprintf(`{"msgId":7,"status":"WS_NO_PERMISSION","errorText":"View access denied for: %s","quantCombineResp":{}}`, scanId),
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	u1Viewer := &protos.UserGroupList{
		UserIds: []string{u1.GetUserId()},
	}
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, u1Viewer, nil)

	u1.AddSendReqAction("MultiQuant - should fail, cant load quant",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%v",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-123"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-456"
				}
			]
		}}`, scanId),
		`{"msgId":8,"status":"WS_NOT_FOUND","errorText":"quant-123 not found","quantCombineResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Seed quant
	quantPath := "Quantifications/" + scanId + "/" + u1.GetUserId() + "/"
	seedDBQuants([]*protos.QuantificationSummary{
		{
			Id:     "quant-combined",
			ScanId: scanId,
			Params: &protos.QuantStartingParameters{
				UserParams: &protos.QuantCreateParams{
					Command:        "",
					Name:           "Quant Combined",
					ScanId:         scanId,
					Elements:       []string{"Ca", "Ti", "Fe", "Si"},
					DetectorConfig: "PIXL/PiquantConfigs/v7",
					Parameters:     "",
					RunTimeSec:     60,
					QuantMode:      "Combined",
					RoiIDs:         []string{},
					IncludeDwells:  false,
				},
				PmcCount:          313,
				ScanFilePath:      "Scans/" + scanId + "/dataset.bin",
				DataBucket:        "databucket",
				PiquantJobsBucket: "piquantbucket",
				CoresPerNode:      4,
				StartUnixTimeSec:  1652813392,
				RequestorUserId:   "auth0|5df311ed8a0b5d0ebf5fb476",
				PIQUANTVersion:    "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
				Comments:          "",
			},
			Elements: []string{"CaO", "TiO2", "FeO-T", "SiO2"},
			Status: &protos.JobStatus{
				JobId:          "quant-combined",
				Status:         5,
				Message:        "Nodes ran: 7",
				EndUnixTimeSec: 1652813627,
				OutputFilePath: quantPath,
				OtherLogFiles:  []string{},
			},
		},
		{
			Id:     "quant-AB",
			ScanId: scanId,
			Params: &protos.QuantStartingParameters{
				UserParams: &protos.QuantCreateParams{
					Command:        "",
					Name:           "Quant AB",
					ScanId:         scanId,
					Elements:       []string{"Ca", "Ti", "Fe", "Si"},
					DetectorConfig: "PIXL/PiquantConfigs/v7",
					Parameters:     "",
					RunTimeSec:     60,
					QuantMode:      "AB",
					RoiIDs:         []string{},
					IncludeDwells:  false,
				},
				PmcCount:          313,
				ScanFilePath:      "Scans/" + scanId + "/dataset.bin",
				DataBucket:        "databucket",
				PiquantJobsBucket: "piquantbucket",
				CoresPerNode:      4,
				StartUnixTimeSec:  1652813392,
				RequestorUserId:   "auth0|5df311ed8a0b5d0ebf5fb476",
				PIQUANTVersion:    "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
				Comments:          "",
			},
			Elements: []string{"CaO", "TiO2", "FeO-T", "SiO2"},
			Status: &protos.JobStatus{
				JobId:          "quant-AB",
				Status:         5,
				Message:        "Nodes ran: 7",
				EndUnixTimeSec: 1652813627,
				OutputFilePath: quantPath,
				OtherLogFiles:  []string{},
			},
		},
		{
			Id:     "quant-3elem",
			ScanId: scanId,
			Params: &protos.QuantStartingParameters{
				UserParams: &protos.QuantCreateParams{
					Command:        "",
					Name:           "Quant 3 element",
					ScanId:         scanId,
					Elements:       []string{"Fe", "Ti", "Ca"},
					DetectorConfig: "PIXL/PiquantConfigs/v7",
					Parameters:     "",
					RunTimeSec:     60,
					QuantMode:      "Combined",
					RoiIDs:         []string{},
					IncludeDwells:  false,
				},
				PmcCount:          313,
				ScanFilePath:      "Scans/" + scanId + "/dataset.bin",
				DataBucket:        "databucket",
				PiquantJobsBucket: "piquantbucket",
				CoresPerNode:      4,
				StartUnixTimeSec:  1652813392,
				RequestorUserId:   "auth0|5df311ed8a0b5d0ebf5fb476",
				PIQUANTVersion:    "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
				Comments:          "",
			},
			Elements: []string{"FeO-T", "TiO2", "CaO_%"},
			Status: &protos.JobStatus{
				JobId:          "quant-3elem",
				Status:         5,
				Message:        "Nodes ran: 7",
				EndUnixTimeSec: 1652813627,
				OutputFilePath: quantPath,
				OtherLogFiles:  []string{},
			},
		},
	})

	seedS3File("combined.bin", filepaths.GetQuantPath(u1.GetUserId(), scanId, "quant-combined.bin"), apiUsersBucket)
	seedS3File("AB.bin", filepaths.GetQuantPath(u1.GetUserId(), scanId, "quant-AB.bin"), apiUsersBucket)
	seedS3File("combined-3elem.bin", filepaths.GetQuantPath(u1.GetUserId(), scanId, "quant-3elem.bin"), apiUsersBucket)

	ownerships := []*protos.OwnershipItem{
		{
			Id:             scanId,
			ObjectType:     protos.ObjectType_OT_SCAN,
			CreatorUserId:  "",
			CreatedUnixSec: 1646262426,
		},
		{
			Id:             "quant-combined",
			ObjectType:     protos.ObjectType_OT_QUANTIFICATION,
			CreatorUserId:  "",
			CreatedUnixSec: 1646262426,
		},
		{
			Id:             "quant-AB",
			ObjectType:     protos.ObjectType_OT_QUANTIFICATION,
			CreatorUserId:  "",
			CreatedUnixSec: 1646262426,
		},
		{
			Id:             "quant-3elem",
			ObjectType:     protos.ObjectType_OT_QUANTIFICATION,
			CreatorUserId:  "",
			CreatedUnixSec: 1646262426,
		},
	}
	seedDBOwnershipMulti(ownerships, u1Viewer, nil)

	u1.AddSendReqAction("MultiQuant - should fail, quants incompatible",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%v",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-combined"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-AB"
				}
			]
		}}`, scanId),
		`{"msgId":9,"status":"WS_BAD_REQUEST","errorText":"Detectors don't match other quantifications: quant-AB","quantCombineResp":{}}`,
	)

	// NOTE: this traverses the z-stack in reverse order, so the first one encountered is roi-third!
	u1.AddSendReqAction("MultiQuant - should fail, cant load roi",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%v",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-combined"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-3elem"
				},
				{
					"roiId": "roi-third",
					"quantificationId": "quant-combined"
				}
			]
		}}`, scanId),
		`{"msgId":10,"status":"WS_NOT_FOUND","errorText":"roi-third not found","quantCombineResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Seed ROIs into DB
	seedDBROI([]*protos.ROIItem{
		{
			Id:                      "roi-first",
			ScanId:                  scanId,
			Name:                    "1st ROI",
			Description:             "1st",
			ScanEntryIndexesEncoded: []int32{23, 29},
		},
		{
			Id:                      "roi-second",
			ScanId:                  scanId,
			Name:                    "Second ROI",
			ScanEntryIndexesEncoded: []int32{22, 25, 29},
		},
		{
			Id:                      "roi-third",
			ScanId:                  scanId,
			Name:                    "Third ROI (shared)",
			Description:             "The third one",
			ScanEntryIndexesEncoded: []int32{23, 26},
		},
	})

	u1.AddSendReqAction("MultiQuant - should work",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%v",
			"description": "combined quants",
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-combined"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-3elem"
				},
				{
					"roiId": "roi-third",
					"quantificationId": "quant-combined"
				}
			]
		}}`, scanId),
		`{"msgId":11,"status":"WS_OK","quantCombineResp":{"jobId": "${IDSAVE=multiQuantId}"}}`,
	)

	u1.AddSendReqAction("MultiQuant summary only - should work",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "here's a name",
			"scanId": "%v",
			"description": "combined quants",
			"summaryOnly": true,
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-combined"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-3elem"
				},
				{
					"roiId": "roi-third",
					"quantificationId": "quant-combined"
				}
			]
		}}`, scanId),
		`{"msgId":12,"status":"WS_BAD_REQUEST","errorText": "Name already used: here's a name","quantCombineResp":{}}`,
	)

	u1.AddSendReqAction("MultiQuant summary only - should work",
		fmt.Sprintf(`{"quantCombineReq":{
			"name": "summary-only test",
			"scanId": "%v",
			"description": "combined quants",
			"summaryOnly": true,
			"roiZStack": [
				{
					"roiId": "roi-first",
					"quantificationId": "quant-combined"
				},
				{
					"roiId": "roi-second",
					"quantificationId": "quant-3elem"
				},
				{
					"roiId": "roi-third",
					"quantificationId": "quant-combined"
				}
			]
		}}`, scanId),
		`{"msgId":13,"status":"WS_OK","quantCombineResp":{
			"summary": {
				"detectors": [
					"Combined"
				],
				"weightPercents": {
					"CaO": {
						"values": [
							0.10906311
						],
						"roiIds": [
							"roi-first",
							"roi-second",
							"roi-third"
						],
						"roiNames": [
							"1st ROI",
							"Second ROI",
							"Third ROI (shared)"
						]
					},
					"FeO-T": {
						"values": [
							0.43899643
						],
						"roiIds": [
							"roi-first",
							"roi-second",
							"roi-third"
						],
						"roiNames": [
							"1st ROI",
							"Second ROI",
							"Third ROI (shared)"
						]
					},
					"SiO2": {
						"values": [
							0.44375953
						],
						"roiIds": [
							"roi-first",
							"roi-third"
						],
						"roiNames": [
							"1st ROI",
							"Third ROI (shared)"
						]
					},
					"TiO2": {
						"values": [
							0.009725777
						],
						"roiIds": [
							"roi-first",
							"roi-second",
							"roi-third"
						],
						"roiNames": [
							"1st ROI",
							"Second ROI",
							"Third ROI (shared)"
						]
					}
				}
			}
		}}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)
}
