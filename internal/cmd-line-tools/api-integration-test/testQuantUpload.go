package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuantUpload(apiHost string) {
	scanId := "the-scan-id"
	quantId := "3vjoovnrhkhv8ecd"

	quantLogs := []string{
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
	}

	seedDBQuants([]*protos.QuantificationSummary{
		{
			Id:     quantId,
			ScanId: scanId,
			Params: &protos.QuantStartingParameters{
				UserParams: &protos.QuantCreateParams{
					Command: "",
					Name:    "Trial quant with Rh",
					ScanId:  scanId,
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
					DetectorConfig: "PIXL/PiquantConfigs/v7",
					Parameters:     "",
					RunTimeSec:     60,
					QuantMode:      "AB",
					RoiIDs:         []string{},
					IncludeDwells:  false,
				},
				PmcCount:          100,
				ScanFilePath:      "Datasets/" + scanId + "/dataset.bin",
				DataBucket:        "databucket",
				PiquantJobsBucket: "piquantbucket",
				CoresPerNode:      4,
				StartUnixTimeSec:  1652813392,
				RequestorUserId:   "auth0|5df311ed8a0b5d0ebf5fb476",
				PIQUANTVersion:    "registry.gitlab.com/pixlise/piquant/runner:3.2.8",
				Comments:          "",
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
				JobId:          quantId,
				Status:         5,
				Message:        "Nodes ran: 7",
				EndUnixTimeSec: 1652813627,
				OutputFilePath: "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
				OtherLogFiles:  quantLogs,
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

	// Upload a quant, check that it worked, delete it
	u1.AddSendReqAction("Upload quant CSV (should work)",
		fmt.Sprintf(`{"quantUploadReq":{
			"scanId": "%v",
			"name": "uploaded Quant",
			"comments": "This was just uploaded from CSV",
			"csvData": "Header line\nPMC,Ca_%%,livetime,RTT,SCLK,filename\n1,5.3,9.9,98765,1234567890,Normal_A"
		}}`, scanId),
		`{"msgId":2,"status":"WS_OK", "quantUploadResp":{"createdQuantId": "${IDSAVE=uploadedQuantId}"}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Check that the files have been deleted
	items, err := apiStorageFileAccess.ListObjects(apiUsersBucket, filepaths.RootQuantificationPath+"/"+scanId+"/")
	if err != nil {
		log.Fatalln(err)
	}

	if len(items) != 2 {
		log.Fatalf("Quant upload must've failed")
	}

	// Now create a quant by uploading a CSV
	u1.AddSendReqAction("Get quant summary+data (should work)",
		`{"quantGetReq":{"quantId": "${IDLOAD=uploadedQuantId}"}}`,
		fmt.Sprintf(`{
			"msgId": 3,
			"status": "WS_OK",
			"quantGetResp": {
				"summary": {
					"id": "${IDCHK=uploadedQuantId}",
					"scanId": "%v",
					"params": {
						"userParams": {
							"command": "map",
							"name": "uploaded Quant",
							"scanId": "%v",
							"elements": ["Ca"],
							"quantMode": "ABManual"
						},
						"dataBucket": "devpixlise-datasets0030ee04-ox1crk4uej2x",
						"scanFilePath": "Scans/the-scan-id/dataset.bin",
						"piquantJobsBucket": "devpixlise-piquantjobs2a7b0239-wcx2ijxt49jc",
						"startUnixTimeSec": "${SECAGO=3}",
						"requestorUserId": "${USERID}",
						"PIQUANTVersion": "N/A",
						"comments": "This was just uploaded from CSV"
					},
					"elements": [
						"Ca"
					],
					"status": {
						"jobId": "${IDCHK=uploadedQuantId}",
						"status": "COMPLETE",
						"message": "user-supplied quantification processed",
						"endUnixTimeSec": "${SECAGO=3}",
						"outputFilePath": "Quantifications/the-scan-id/auth0|649e54491154cac52ec21718"
					},
					"owner": {
						"creatorUser": {
							"id": "auth0|649e54491154cac52ec21718",
							"name": "test1@pixlise.org - WS Integration Test",
							"email": "test1@pixlise.org"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				},
				"data": {
					"labels": [
						"Ca_%%",
						"livetime"
					],
					"types": [
						"QT_FLOAT",
						"QT_FLOAT"
					],
					"locationSet": [
						{
							"detector": "A",
							"location": [
								{
									"pmc": 1,
									"rtt": 98765,
									"sclk": 1234567890,
									"values": [
										{
											"fvalue": 5.3
										},
										{
											"fvalue": 9.9
										}
									]
								}
							]
						}
					]
				}
			}
		}`, scanId, scanId),
	)

	u1.AddSendReqAction("Delete uploaded quant (should work)",
		`{"quantDeleteReq":{"quantId": "${IDLOAD=uploadedQuantId}"}}`,
		`{"msgId":4,"status":"WS_OK", "quantDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Get quant (should fail, not in db)",
		`{"quantGetReq":{"quantId": "${IDLOAD=uploadedQuantId}"}}`,
		fmt.Sprintf(`{"msgId":5,"status":"WS_NOT_FOUND", "errorText": "%v not found", "quantGetResp":{}}`, wstestlib.GetIdCreated("uploadedQuantId")),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	items, err = apiStorageFileAccess.ListObjects(apiUsersBucket, filepaths.RootQuantificationPath+"/"+scanId+"/")
	if err != nil {
		log.Fatalln(err)
	}

	if len(items) > 0 {
		log.Fatalf("Failed to delete all uploaded quant files. Remaining: %v\n", strings.Join(items, ", "))
	}
}
