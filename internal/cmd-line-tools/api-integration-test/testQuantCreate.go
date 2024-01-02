package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/pixlise/core/v3/core/utils"
	"github.com/pixlise/core/v3/core/wstestlib"
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
	var wg sync.WaitGroup

	for i, datasetID := range datasetIDs {
		wg.Add(1)
		go func(i int, datasetID string) {
			defer wg.Done()

			now := time.Now().Format(timeFormat)
			fmt.Printf(" %v   Quantify [dataset: %v, quant name: %v] with config: %v, PMC count: %v\n", now, datasetID, quantNames[i], detectorConfig[i], len(pmcList[i]))
			runQuantificationTest(i, apiHost, test1Username, test1Password, datasetID, pmcList[i], elementList, detectorConfig[i], quantNames[i], expectedFinalState[i])
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

func runQuantificationTest(idx int, apiHost string, user string, pass string,
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

		// Export the CSV
		usr.AddSendReqAction(fmt.Sprintf("Export quant %v (should work)", quantName),
			fmt.Sprintf(`{"exportFilesReq":{ "exportTypes": ["EDT_QUANT_CSV"], "quantId": "%v" }}`, quantId),
			fmt.Sprintf(`{"msgId":2,"status":"WS_OK", "exportFilesResp":{
				"zipData": "${ZIPCMP,SKIPCSVLINES=0,PATH=./test-files/quant-exp-output/%v}"
			}}`, idx+1),
		)

		usr.AddSendReqAction(fmt.Sprintf("Delete quant %v (should work)", quantName),
			fmt.Sprintf(`{"quantDeleteReq":{"quantId": "%v" }}`, quantId),
			`{"msgId":3,"status":"WS_OK", "quantDeleteResp":{}}`,
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
