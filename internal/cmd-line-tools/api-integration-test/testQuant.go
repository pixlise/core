package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuants(apiHost string) {
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

	// Ensure files aren't there in S3 at this point
	// NOTE: This test was unfortunately written for a slightly weird scan that was copied between user accounts
	// so the username in the path is not the expected u1.GetUserId() one!
	//     filepaths.GetQuantPath(u1.GetUserId(), scanId, quantId+".bin")
	// Which evaluates to:
	//     Quantifications/089063943/u1.GetUserId()/3vjoovnrhkhv8ecd.bin
	// but it's the one referenced in the quant summary:
	//     UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications/3vjoovnrhkhv8ecd.bin
	// Which means we need to override the user id here:
	thisQuantRootPath := "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications/"
	err := apiStorageFileAccess.DeleteObject(apiUsersBucket, thisQuantRootPath+quantId+".bin")
	if err != nil {
		log.Fatalln(err)
	}

	// Now add u1 as a viewer
	seedDBOwnership(quantId, protos.ObjectType_OT_QUANTIFICATION, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}}, nil)

	u1.AddSendReqAction("List quants",
		`{"quantListReq":{}}`,
		fmt.Sprintf(`{"msgId": 5, "status": "WS_OK", "quantListResp": {
			"quants": [{
				"id": "%v",
				"scanId": "the-scan-id",
				"params": {
					"userParams": {
						"name": "Trial quant with Rh",
						"scanId": "the-scan-id",
						"elements": ["CO3","Rh","Na","Mg","Al","Si","P","S","Cl","K","Ca","Ti","Cr","Mn","Fe"],
						"detectorConfig": "PIXL/PiquantConfigs/v7",
						"runTimeSec": 60,
						"quantMode": "AB"
					},
					"dataBucket": "databucket",
					"scanFilePath": "Datasets/the-scan-id/dataset.bin",
					"piquantJobsBucket": "piquantbucket",
					"pmcCount": 100,
					"coresPerNode": 4,
					"startUnixTimeSec": 1652813392,
					"requestorUserId": "auth0|5df311ed8a0b5d0ebf5fb476",
					"PIQUANTVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8"
				},
				"elements": ["Rh2O3","Na2O","MgCO3","Al2O3","SiO2","P2O5","SO3","Cl","K2O","CaCO3","TiO2","Cr2O3","MnCO3","FeCO3-T"],
				"status": {
					"jobId": "%v",
					"status": "COMPLETE",
					"message": "Nodes ran: 7",
					"endUnixTimeSec": 1652813627,
					"outputFilePath": "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
					"otherLogFiles": ["node00001_piquant.log","node00001_stdout.log","node00002_piquant.log","node00002_stdout.log","node00003_piquant.log","node00003_stdout.log","node00004_piquant.log","node00004_stdout.log","node00005_piquant.log","node00005_stdout.log","node00006_piquant.log","node00006_stdout.log","node00007_piquant.log","node00007_stdout.log"]
				},
                "owner": {
                    "creatorUser": {},
                    "createdUnixSec": 1646262426,
                    "viewerUserCount": 1,
                    "sharedWithOthers": true
                }
			}]
		}}`, quantId, quantId),
	)

	u1.AddSendReqAction("Get quant summary only (should work)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v", "summaryOnly": true }}`, quantId),
		fmt.Sprintf(`{"msgId":6,"status":"WS_OK", "quantGetResp":{
			"summary": {
				"id": "%v",
				"scanId": "the-scan-id",
				"params": {
					"userParams": {
						"name": "Trial quant with Rh",
						"scanId": "the-scan-id",
						"elements": ["CO3","Rh","Na","Mg","Al","Si","P","S","Cl","K","Ca","Ti","Cr","Mn","Fe"],
						"detectorConfig": "PIXL/PiquantConfigs/v7",
						"runTimeSec": 60,
						"quantMode": "AB"
					},
					"dataBucket": "databucket",
					"scanFilePath": "Datasets/the-scan-id/dataset.bin",
					"piquantJobsBucket": "piquantbucket",
					"pmcCount": 100,
					"coresPerNode": 4,
					"startUnixTimeSec": 1652813392,
					"requestorUserId": "auth0|5df311ed8a0b5d0ebf5fb476",
					"PIQUANTVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8"
				},
				"elements": ["Rh2O3","Na2O","MgCO3","Al2O3","SiO2","P2O5","SO3","Cl","K2O","CaCO3","TiO2","Cr2O3","MnCO3","FeCO3-T"],
				"status": {
					"jobId": "%v",
					"status": "COMPLETE",
					"message": "Nodes ran: 7",
					"endUnixTimeSec": 1652813627,
					"outputFilePath": "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
					"otherLogFiles": ["node00001_piquant.log","node00001_stdout.log","node00002_piquant.log","node00002_stdout.log","node00003_piquant.log","node00003_stdout.log","node00004_piquant.log","node00004_stdout.log","node00005_piquant.log","node00005_stdout.log","node00006_piquant.log","node00006_stdout.log","node00007_piquant.log","node00007_stdout.log"]
				},
                "owner": {
                    "creatorUser": {},
                    "createdUnixSec": 1646262426,
                    "viewerUserCount": 1,
                    "sharedWithOthers": true
                }
			}
		}}`, quantId, quantId),
	)

	u1.AddSendReqAction("Get quant summary+data (should fail, no file in S3)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v" }}`, quantId),
		fmt.Sprintf(`{"msgId":7,"status":"WS_NOT_FOUND", "errorText": "%v not found", "quantGetResp":{}}`, quantId),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// As above, this quant has slightly "weird" paths...
	seedQuantFile(quantId+".bin", thisQuantRootPath+quantId+".bin" /*u1.GetUserId(), scanId*/, apiUsersBucket)
	seedQuantFile(quantId+".csv", thisQuantRootPath+quantId+".csv" /*u1.GetUserId(), scanId*/, apiUsersBucket)
	for _, logFile := range quantLogs {
		seedQuantFile("./"+quantId+"-logs/"+logFile, thisQuantRootPath+quantId+"-logs/"+logFile /*u1.GetUserId(), scanId*/, apiUsersBucket)
	}

	u1.AddSendReqAction("Get quant summary+data (should work)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v" }}`, quantId),
		fmt.Sprintf(`{"msgId":8,"status":"WS_OK", "quantGetResp":{
			"summary": {
				"id": "%v",
				"scanId": "the-scan-id",
				"params": {
					"userParams": {
						"name": "Trial quant with Rh",
						"scanId": "the-scan-id",
						"elements": ["CO3","Rh","Na","Mg","Al","Si","P","S","Cl","K","Ca","Ti","Cr","Mn","Fe"],
						"runTimeSec": 60,
						"quantMode": "AB",
						"detectorConfig": "PIXL/PiquantConfigs/v7"
					},
					"dataBucket": "databucket",
					"scanFilePath": "Datasets/the-scan-id/dataset.bin",
					"piquantJobsBucket": "piquantbucket",
					"pmcCount": 100,
					"coresPerNode": 4,
					"startUnixTimeSec": 1652813392,
					"requestorUserId": "auth0|5df311ed8a0b5d0ebf5fb476",
					"PIQUANTVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8"
				},
				"elements": ["Rh2O3","Na2O","MgCO3","Al2O3","SiO2","P2O5","SO3","Cl","K2O","CaCO3","TiO2","Cr2O3","MnCO3","FeCO3-T"],
				"status": {
					"jobId": "%v",
					"status": "COMPLETE",
					"message": "Nodes ran: 7",
					"endUnixTimeSec": 1652813627,
					"outputFilePath": "UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications",
					"otherLogFiles": ["node00001_piquant.log","node00001_stdout.log","node00002_piquant.log","node00002_stdout.log","node00003_piquant.log","node00003_stdout.log","node00004_piquant.log","node00004_stdout.log","node00005_piquant.log","node00005_stdout.log","node00006_piquant.log","node00006_stdout.log","node00007_piquant.log","node00007_stdout.log"]
				},
                "owner": {
                    "creatorUser": {},
                    "createdUnixSec": 1646262426,
                    "viewerUserCount": 1,
                    "sharedWithOthers": true
                }
			},
			"data": {
				"labels": [
					"Rh2O3_%%",
					"Na2O_%%",
					"MgCO3_%%",
					"Al2O3_%%",
					"SiO2_%%",
					"P2O5_%%",
					"SO3_%%",
					"Cl_%%",
					"K2O_%%",
					"CaCO3_%%",
					"TiO2_%%",
					"Cr2O3_%%",
					"MnCO3_%%",
					"FeCO3-T_%%",
					"Rh2O3_int",
					"Na2O_int",
					"MgCO3_int",
					"Al2O3_int",
					"SiO2_int",
					"P2O5_int",
					"SO3_int",
					"Cl_int",
					"K2O_int",
					"CaCO3_int",
					"TiO2_int",
					"Cr2O3_int",
					"MnCO3_int",
					"FeCO3-T_int",
					"Rh2O3_err",
					"Na2O_err",
					"MgCO3_err",
					"Al2O3_err",
					"SiO2_err",
					"P2O5_err",
					"SO3_err",
					"Cl_err",
					"K2O_err",
					"CaCO3_err",
					"TiO2_err",
					"Cr2O3_err",
					"MnCO3_err",
					"FeCO3-T_err",
					"total_counts",
					"livetime",
					"chisq",
					"eVstart",
					"eV/ch",
					"res",
					"iter",
					"Events",
					"Triggers"
				],
				"types": [
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_INT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_FLOAT",
					"QT_INT",
					"QT_INT",
					"QT_INT",
					"QT_INT"
				],
				"locationSet": [
					{
						"detector": "A",
						"location${LIST,MODE=CONTAINS,LENGTH=143}": [
							{
								"pmc": 86,
								"values": [
									{
										"fvalue": -1
									},
									{
										"fvalue": 1.0083
									},
									{
										"fvalue": 37.8801
									},
									{
										"fvalue": 0.5924
									},
									{
										"fvalue": 14.7764
									},
									{
										"fvalue": 0.0129
									},
									{
										"fvalue": 0.6127
									},
									{
										"fvalue": 0.9561
									},
									{},
									{
										"fvalue": 3.9732
									},
									{},
									{},
									{
										"fvalue": 1.1172
									},
									{
										"fvalue": 41.825
									},
									{},
									{
										"fvalue": 6.3
									},
									{
										"fvalue": 745.5
									},
									{
										"fvalue": 66.5
									},
									{
										"fvalue": 4243.8
									},
									{
										"fvalue": 6.4
									},
									{
										"fvalue": 602.8
									},
									{
										"fvalue": 1911.1
									},
									{},
									{
										"fvalue": 3643.5
									},
									{},
									{},
									{
										"fvalue": 2318.3
									},
									{
										"fvalue": 79901.1
									},
									{},
									{
										"fvalue": 1.6
									},
									{
										"fvalue": 2.4
									},
									{
										"fvalue": 0.2
									},
									{
										"fvalue": 0.8
									},
									{},
									{
										"fvalue": 0.2
									},
									{
										"fvalue": 0.3
									},
									{},
									{
										"fvalue": 0.5
									},
									{},
									{},
									{
										"fvalue": 0.4
									},
									{
										"fvalue": 2.1
									},
									{
										"ivalue": 109490
									},
									{
										"fvalue": 9.12
									},
									{
										"fvalue": 0.64
									},
									{
										"fvalue": -24.4
									},
									{
										"fvalue": 7.8811
									},
									{
										"ivalue": 178
									},
									{
										"ivalue": 23
									},
									{},
									{}
								]
							}
						]
					},
					{
						"detector": "B",
						"location${LIST,MODE=LENGTH,LENGTH=143}": []
					}
				]
			}
		}}`, quantId, quantId),
	)

	u1.AddSendReqAction("Delete non-existant quant (should fail)",
		`{"quantDeleteReq":{"quantId": "non-existant-quant" }}`,
		`{"msgId":9,"status":"WS_NOT_FOUND", "errorText": "non-existant-quant not found", "quantDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Delete quant (should fail, we're viewers!)",
		fmt.Sprintf(`{"quantDeleteReq":{"quantId": "%v" }}`, quantId),
		fmt.Sprintf(`{"msgId":10,"status":"WS_NO_PERMISSION", "errorText": "Edit access denied for: %v", "quantDeleteResp":{}}`, quantId),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u2 := wstestlib.MakeScriptedTestUser(auth0Params)

	u2.AddConnectAction("Connect user 2", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test2Username,
		Pass: test2Password,
	})

	u2.AddSendReqAction("User2: List quants",
		`{"quantListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","quantListResp":{}}`,
	)

	u2.AddSendReqAction("User2: Get quant from db (should fail, permissions dont allow)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v"}}`, quantId),
		fmt.Sprintf(`{
			"msgId":2,"status":"WS_NO_PERMISSION",
			"errorText": "View access denied for: %v", "quantGetResp":{}}`, quantId),
	)

	u2.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u2)

	// Set as editor
	seedDBOwnership(quantId, protos.ObjectType_OT_QUANTIFICATION, nil, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}})

	u1.AddSendReqAction("Delete quant (should work)",
		fmt.Sprintf(`{"quantDeleteReq":{"quantId": "%v" }}`, quantId),
		`{"msgId":11,"status":"WS_OK", "quantDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Get quant (should fail, not in db)",
		fmt.Sprintf(`{"quantGetReq":{"quantId": "%v" }}`, quantId),
		fmt.Sprintf(`{"msgId":12,"status":"WS_NOT_FOUND", "errorText": "%v not found", "quantGetResp":{}}`, quantId),
	)

	// Upload a quant, check that it worked, delete it
	u1.AddSendReqAction("Upload quant CSV (should work)",
		fmt.Sprintf(`{"quantUploadReq":{
			"scanId": "%v",
			"name": "uploaded Quant",
			"comments": "This was just uploaded from CSV",
			"csvData": "Header line\nPMC,Ca_%%,livetime,RTT,SCLK,filename\n1,5.3,9.9,98765,1234567890,Normal_A"
		}}`, scanId),
		`{"msgId":13,"status":"WS_OK", "quantUploadResp":{"createdQuantId": "${IDSAVE=uploadedQuantId}"}}`,
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
			"msgId": 14,
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
		`{"msgId":15,"status":"WS_OK", "quantDeleteResp":{}}`,
	)

	u1.AddSendReqAction("Get quant (should fail, not in db)",
		`{"quantGetReq":{"quantId": "${IDLOAD=uploadedQuantId}"}}`,
		fmt.Sprintf(`{"msgId":16,"status":"WS_NOT_FOUND", "errorText": "%v not found", "quantGetResp":{}}`, wstestlib.GetIdCreated("uploadedQuantId")),
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

func seedQuantFile(fileName string, s3Path string /*userId string, scanId string*/, bucket string) {
	data, err := os.ReadFile("./test-files/" + fileName)
	if err != nil {
		log.Fatalln(err)
	}

	// Upload it where we need it for the test
	//s3Path := filepaths.GetQuantPath(userId, scanId, fileName)
	err = apiStorageFileAccess.WriteObject(bucket, s3Path, data)
	if err != nil {
		log.Fatalln(err)
	}
}
