package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pixlise/core/v4/core/utils"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testQuantCreate(apiHost string) {
	resetDBPiquantAndJobs()

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
	/*var wg sync.WaitGroup

	for i, datasetID := range datasetIDs {
		wg.Add(1)
		go func(i int, datasetID string) {
			defer wg.Done()

			now := time.Now().Format(timeFormat)
			fmt.Printf(" %v   Quantify [dataset: %v, quant name: %v] with config: %v, PMC count: %v\n", now, datasetID, quantNames[i], detectorConfig[i], len(pmcList[i]))
			runQuantificationTest(i, apiHost, test1Username, test1Password, datasetID, pmcList[i], elementList, detectorConfig[i], quantNames[i], expectedFinalState[i])
		}(i, datasetID)
	}*/

	// Wait for all
	fmt.Println("\n---------------------------------------------------------")
	now := time.Now().Format(timeFormat)
	fmt.Printf(" %v  STARTING quantifications, will wait for them to complete...\n", now)
	fmt.Printf("---------------------------------------------------------\n\n")

	//wg.Wait()

	// Unfortunately, we now have to run tests in serial, because parallel implies the order they complete is not known so we can't
	// write a list of expected messages (due to the addition of notifications for quant complete). If we run them serially, we know
	// quant 2 won't get a quant 1 "completed" notification!
	for i, datasetID := range datasetIDs {
		now := time.Now().Format(timeFormat)
		fmt.Printf(" %v   Quantify [dataset: %v, quant name: %v] with config: %v, PMC count: %v\n", now, datasetID, quantNames[i], detectorConfig[i], len(pmcList[i]))
		runQuantificationTest(i, apiHost, test1Username, test1Password, datasetID, pmcList[i], elementList, detectorConfig[i], quantNames[i], expectedFinalState[i])
	}

	fmt.Println("---------------------------------------------------------")
	now = time.Now().Format(timeFormat)
	fmt.Printf(" %v  QUANTIFICATIONS completed...\n", now)
	fmt.Printf("---------------------------------------------------------\n\n")
}

func runQuantificationTest(idx int, apiHost string, user string, pass string,
	scanId string, pmcList []int32, elementList []string, detectorConfig string, quantName string, expectedFinalState string) {
	var maxRunTimeSec = 240
	var maxAgeSec = maxRunTimeSec
	if expectedFinalState == "ERROR" {
		maxRunTimeSec = 20
	} else {
		maxAgeSec += 30 // TODO: Not sure why, but we seem to get timestamps that are too old?
	}

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
				"startUnixTimeSec": "${SECAGO=%v}",
				"jobItemId": "${IGNORE}",
				"jobType": "JT_RUN_QUANT"
			}
		}}`, idx+1, maxAgeSec),
	)

	finalMsg := fmt.Sprintf(`{"quantCreateUpd":{
		"status": {
			"jobId": "${IDCHK=quantCreate%v}",
			"logId": "${IDCHK=quantCreate%v}",
			"message": "${IGNORE}",
			"status": "%v",
			"startUnixTimeSec": "${SECAGO=%v}",
			"lastUpdateUnixTimeSec": "${SECAGO=%v}",
			"endUnixTimeSec": "${SECAGO=%v}"`, idx+1, idx+1, expectedFinalState, maxAgeSec, maxAgeSec, maxAgeSec)
	if expectedFinalState != "ERROR" {
		finalMsg += `,
			"outputFilePath": "${IGNORE}",
			"otherLogFiles": "${IGNORE}"`
	}
	finalMsg += `
		}
	}}`

	expectedUpdates := []string{
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "Cores/Node: 4",
				"status": "PREPARING_NODES",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxAgeSec, maxAgeSec),
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "${IGNORE}",
				"status": "RUNNING",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxAgeSec, maxAgeSec),
		fmt.Sprintf(`{"quantCreateUpd":{
			"status": {
				"jobId": "${IDCHK=quantCreate%v}",
				"logId": "${IDCHK=quantCreate%v}",
				"message": "${IGNORE}",
				"status": "GATHERING_RESULTS",
				"startUnixTimeSec": "${SECAGO=%v}",
				"lastUpdateUnixTimeSec": "${SECAGO=%v}"
			}
		}}`, idx+1, idx+1, maxAgeSec, maxAgeSec),
	}

	/*if expectedFinalState != "ERROR" {
		expectedUpdates = append(expectedUpdates, fmt.Sprintf(`{"notificationUpd": {"notification": { "notificationType": "NT_SYS_DATA_CHANGED", "quantId":"${IDCHK=quantCreate%v}"}}}`, idx+1))
	}*/

	expectedUpdates = append(expectedUpdates, finalMsg)

	usr.CloseActionGroup(expectedUpdates, maxRunTimeSec*1000)

	// Ignoring messages above - they differ per quant, example of simple small quant is:
	// RUNNING: Node count: 1, Spectra/Node: 9
	// GATHERING_RESULTS: Combining CSVs from 1 nodes...
	// COMPLETE: Nodes ran: 1
	// But we may have multiple nodes, etc so lets just ignore the messages and rely on state

	wstestlib.ExecQueuedActions(&usr)

	if expectedFinalState == "COMPLETE" {
		quantId := wstestlib.GetIdCreated(fmt.Sprintf("quantCreate%v", idx+1))

		// Export the CSV
		usr.AddSendReqAction(fmt.Sprintf("Export quant %v (should work)", quantName),
			fmt.Sprintf(`{"exportFilesReq":{ "exportTypes": ["EDT_QUANT_CSV"], "quantId": "%v" }}`, quantId),
			fmt.Sprintf(`{"msgId":2,"status":"WS_OK", "exportFilesResp":{
				"files": [{
					"name": "%v.csv",
					"extension": "csv",
					"content": "${IGNORE}"
				}]
			}}`, quantId),
			// "content": "${ZIPCMP,SKIPCSVLINES=0,PATH=./test-files/quant-exp-output/%v}"
			// Parameter for Sprintf: idx+1
		)

		usr.AddSendReqAction(fmt.Sprintf("Delete quant %v (should work)", quantName),
			fmt.Sprintf(`{"quantDeleteReq":{"quantId": "%v" }}`, quantId),
			`{"msgId":3,"status":"WS_OK", "quantDeleteResp":{}}`,
		)

		usr.CloseActionGroup([]string{
			fmt.Sprintf(`{"notificationUpd": {
				"notification": { "notificationType": "NT_SYS_DATA_CHANGED", "quantId":"%v"}}}`, quantId),
		}, 10000)
		wstestlib.ExecQueuedActions(&usr)
	}
}
