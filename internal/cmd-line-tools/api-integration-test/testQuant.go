package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/utils"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuants(apiHost string) {
	testQuantCreate(apiHost)
	//testQuantFit(apiHost)
	//testMultiQuant(apiHost)
	//testQuantGetListDeleteUpload(apiHost)
}

func testQuantCreate(apiHost string) {
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

	// Fire off a few fail ones
	u1.AddSendReqAction("Create quant",
		`{"quantCreateReq":{
			"params": {
				"command": "non-existant-mode",
				"name": "failed-quant1",
				"scanId": "some-scan",
				"pmcs": [7, 9, 10],
				"elements": ["Ca", "Fe"],
				"detectorConfig": "PIXL/v5",
				"parameters": "-b,0,12,60,910,2800,16",
				"runTimeSec": 60,
				"quantMode": "Combined",
				"roiIDs": [],
				"includeDwells": false
			}
		}}`,
		`{"msgId":2,"status":"WS_BAD_REQUEST","errorText": "Unexpected command requested: non-existant-mode", "quantCreateResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// We will run quants with these...
	// Test quantifications on a few pre-determined datasets
	elementList := []string{"Ca", "Ti"}
	//quantColumns := []string{"CaO_%", "TiO2_%"}
	detectorConfig := []string{
		"PIXL/v5",
		"PIXL/v5",
		"Breadboard/v1",
		"NonExistant/v666", // fails! Added because it's an easy way to ensure a quant fails
	}
	pmcsFor5x5 := []int32{}
	//for c := 4043; c < 5806; c++ {
	for c := 4043; c < 4300; c++ {
		//if c != 4827 {
		pmcsFor5x5 = append(pmcsFor5x5, int32(c))
		//}
	}
	pmcList := [][]int32{{68, 69, 70, 71, 72, 73, 74, 75}, pmcsFor5x5, {68, 69, 70, 71, 72, 73, 74, 75}, {68, 69, 70, 71, 72, 73, 74, 75}}
	datasetIDs := []string{
		"983561",
		"test-fm-5x5-full",
		"test-kingscourt",
		"983561", // again, but fail case!
	}
	// Once used test-laguna but stopped due to something about timing out because saving the high res TIFs took longer than 1 minute, which seems to be the test limit?!

	// NOTE: By using 2 of the same names, we also test that the delete
	// didn't leave something behind and another can't be named that way
	quantNameSuffix := utils.RandStringBytesMaskImpr(8)
	quantNames := []string{
		"integration-test 983561 " + quantNameSuffix,
		"integration-test 5x5 " + quantNameSuffix,
		"integration-test kingscourt " + quantNameSuffix,
		"integration-test 983561(fail) " + quantNameSuffix,
	}
	expectedFinalState := []string{"COMPLETE", "COMPLETE", "COMPLETE", "ERROR"}

	// Start each quant
	var wg sync.WaitGroup

	for i, datasetID := range datasetIDs {
		wg.Add(1)
		go func(i int, datasetID string) {
			defer wg.Done()

			now := time.Now().Format(timeFormat)
			fmt.Printf(" %v   Quantify [dataset: %v, quant name: %v] with config: %v, PMC count: %v\n", now, datasetID, quantNames[i], detectorConfig[i], len(pmcList[i]))
			runQuantification(i, apiHost, test1Username, test1Password, datasetID, pmcList[i], elementList, detectorConfig[i], quantNames[i], expectedFinalState[i])
		}(i, datasetID)
	}

	// Wait for all
	fmt.Println("\n---------------------------------------------------------")
	now := time.Now().Format(timeFormat)
	fmt.Printf(" %v  STARTING quantifications, will wait for them to complete...\n", now)
	fmt.Printf("---------------------------------------------------------\n\n")

	wg.Wait()

	fmt.Println("---------------------------------------------------------")
	now = time.Now().Format(timeFormat)
	fmt.Printf(" %v  QUANTIFICATIONS completed...\n", now)
	fmt.Printf("---------------------------------------------------------\n\n")
}

func runQuantification(idx int, apiHost string, user string, pass string,
	scanId string, pmcList []int32, elementList []string, detectorConfig string, quantName string, expectedFinalState string) {
	const maxRunTimeSec = 180

	// Each quant run creates a new session so we separate out the resp/update streams and can "expect" messages
	usr := wstestlib.MakeScriptedTestUser(auth0Params)
	usr.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Start the quant, should get a job ID back
	pmcListStr := ""
	for c, pmc := range pmcList {
		if c > 0 {
			pmcListStr += ","
		}
		pmcListStr += strconv.Itoa(int(pmc))
	}

	elemListStr := ""
	for c, elem := range elementList {
		if c > 0 {
			elemListStr += ","
		}
		elemListStr += fmt.Sprintf("\"%v\"", elem)
	}

	usr.AddSendReqAction("Create quant "+quantName,
		fmt.Sprintf(`{"quantCreateReq":{
			"params": {
				"command": "map",
				"name": "%v",
				"scanId": "%v",
				"pmcs": [%v],
				"elements": [%v],
				"detectorConfig": "%v",
				"parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
				"runTimeSec": 60,
				"quantMode": "Combined",
				"roiIDs": [],
				"includeDwells": false
			}
		}}`, quantName, scanId, pmcListStr, elemListStr, detectorConfig),
		fmt.Sprintf(`{"msgId":1,"status":"WS_OK","quantCreateResp":{
			"status": {
				"jobId": "${IDSAVE=quantCreate%v}",
				"status": "STARTING",
				"startUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, maxRunTimeSec),
	)

	finalMsg := fmt.Sprintf(`{"quantCreateUpd":{
		"status": {
			"jobId": "${IDCHK=quantCreate%v}",
			"logId": "${IDCHK=quantCreate%v}",
			"message": "${IGNORE}",
			"status": "%v",
			"startUnixTimeSec": "${SECAGO=%v}",
			"lastUpdateUnixTimeSec": "${SECAGO=%v}",
			"endUnixTimeSec": "${SECAGO=%v}"`, idx+1, idx+1, expectedFinalState, maxRunTimeSec, maxRunTimeSec, maxRunTimeSec)
	if expectedFinalState != "ERROR" {
		finalMsg += `,
			"outputFilePath": "${IGNORE}",
			"otherLogFiles": "${IGNORE}"`
	}
	finalMsg += `
		}
	}}`

	usr.CloseActionGroup([]string{
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "Cores/Node: 4",
				"status": "PREPARING_NODES",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxRunTimeSec, maxRunTimeSec),
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "${IGNORE}",
				"status": "RUNNING",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxRunTimeSec, maxRunTimeSec),
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "${IGNORE}",
				"status": "GATHERING_RESULTS",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxRunTimeSec, maxRunTimeSec),
		finalMsg,
	}, maxRunTimeSec*1000)

	// Ignoring messages above - they differ per quant, example of simple small quant is:
	// RUNNING: Node count: 1, Spectra/Node: 9
	// GATHERING_RESULTS: Combining CSVs from 1 nodes...
	// COMPLETE: Nodes ran: 1
	// But we may have multiple nodes, etc so lets just ignore the messages and rely on state

	wstestlib.ExecQueuedActions(&usr)

	if expectedFinalState == "COMPLETE" {
		quantId := wstestlib.GetIdCreated(fmt.Sprintf("quantCreate%v", idx+1))

		usr.AddSendReqAction(fmt.Sprintf("Delete quant %v (should work)", quantName),
			fmt.Sprintf(`{"quantDeleteReq":{"quantId": "%v" }}`, quantId),
			`{"msgId":2,"status":"WS_OK", "quantDeleteResp":{}}`,
		)

		usr.CloseActionGroup([]string{}, 5000)
		wstestlib.ExecQueuedActions(&usr)
	}
}

/*
func verifyQuantificationOKThenDelete(jobId string, scanId string, detectorConfig string, pmcList []int32, elementList []string, quantName string, exportColumns []string) error {
	/*resultPrint := printTestStart(fmt.Sprintf("Export of quantification: %v", jobID))
	exportColumnsStr := "["
	for c, col := range exportColumns {
		if c > 0 {
			exportColumnsStr += ","
		}
		exportColumnsStr += fmt.Sprintf("\"%v\"", col)
	}
	exportColumnsStr += "]"
	fileIds := []string{
		"raw-spectra",
		"quant-map-csv",
		"quant-map-tif",
		"beam-locations",
		"rois",
		"context-image",
		"unquantified-weight",
	}

	err := verifyExport(JWT, jobID, environment, datasetID, "export-test.zip", fileIds)
	printTestResult(err, resultPrint)
	if err != nil {
		return err
	}
* /
	// Download the quant file
	resultPrint := printTestStart(fmt.Sprintf("Download and verify quantification: %v", jobId))
	// TODO ADD GENERATE URL
	quantBytes, err := checkFileDownload(JWT, generateURL(environment)+"/quantification/download/"+scanId+"/"+jobId)

	if err == nil {
		// Downloaded, so check that we have the right # of PMCs and elements...
		err = checkQuantificationContents(quantBytes, pmcList, exportColumns)
	}
	printTestResult(err, resultPrint)
	if err != nil {
		return err
	}

	resultPrint = printTestStart(fmt.Sprintf("Delete generated quantification: %v for dataset: %v", jobID, datasetID))
	err = deleteQuant(JWT, jobID, environment, datasetID)
	printTestResult(err, resultPrint)

	return err
}

func checkQuantificationContents(quantBytes []byte, expPMCList []int32, expOutputElements []string) error {
	q := &protos.Quantification{}
	err := proto.Unmarshal(quantBytes, q)
	if err != nil {
		return err
	}

	// Verify the quant created as expected...
	if len(q.LocationSet) != 1 || q.LocationSet[0].Detector != "Combined" {
		return errors.New("Expected single detector named Combined")
	}

	// Make a lookup map for expected PMCs and output columns
	expPMCs := map[int32]bool{} // TODO: REFACTOR: Need generic utils.SetStringsInMap for this...
	for _, pmc := range expPMCList {
		expPMCs[pmc] = true
	}

	expElements := map[string]bool{}
	utils.SetStringsInMap(expOutputElements, expElements)

	keys := make([]int, 0, len(q.LocationSet[0].Location))

	for _, loc := range q.LocationSet[0].Location {
		pmc := loc.Pmc
		keys = append(keys, int(pmc))

		val, pmcExpected := expPMCs[pmc]
		if !pmcExpected {
			return fmt.Errorf("Quant contained unexpected PMC: %v", pmc)
		}
		if !val {
			return fmt.Errorf("Quant contained duplicated PMC: %v", pmc)
		}
		expPMCs[pmc] = false
	}

	sort.Ints(keys)

	// At the end, all our expected PMCs should've been found...
	for pmc, notFound := range expPMCs {
		if notFound {
			return fmt.Errorf("Quant missing expected PMC: %v", pmc)
		}
	}

	for _, label := range q.Labels {
		val, ok := expElements[label]
		if ok {
			// This is an expected label, ensure it's only found once and, mark it as found
			if !val {
				return fmt.Errorf("Quant contained duplicate column: %v", label)
			}
			expElements[label] = false
		}
	}

	for outputElem, notFound := range expElements {
		if notFound {
			return fmt.Errorf("Quant missing expected output element: %v", outputElem)
		}
	}

	return nil
}*/

func testQuantGetListDeleteUpload(apiHost string) {
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
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.QuantificationsName)
	if err != nil {
		log.Fatal(err)
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
