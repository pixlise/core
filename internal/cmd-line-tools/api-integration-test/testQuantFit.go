package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/core/wstestlib"
)

func testQuantFit(apiHost string) {
	maxRunTimeSec := 60

	resetDBPiquantAndJobs()

	usr := wstestlib.MakeScriptedTestUser(auth0Params)
	usr.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	usr.AddSendReqAction("Check non-existant quant output",
		`{"quantLastOutputGetReq":{
			"piquantCommand": "quant",
			"scanId": "non-existant",
			"outputType": 1
		}}`,
		// NOTE: output is "" so proto would not return that field
		`{"msgId":1,"status":"WS_OK","quantLastOutputGetResp":{}}`,
	)

	usr.AddSendReqAction("Check non-existant log output",
		`{"quantLastOutputGetReq":{
			"piquantCommand": "quant",
			"scanId": "non-existant",
			"outputType": 2
		}}`,
		// NOTE: output is "" so proto would not return that field
		`{"msgId":2,"status":"WS_OK","quantLastOutputGetResp":{}}`,
	)

	expData := "   PIQUANT 3.2.16-master  Normal_Combined_AllPoints\nEnergy (keV), meas, calc, bkg, sigma, residual, DetCE, Ti_K, Ca_K, Rh_K_coh, Rh_L_coh, Rh_K_inc, Pileup, Rh_L_coh_Lb1\n-0.0154067, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n-0.0075045, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n0.000397676, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0"
	expData64 := base64.StdEncoding.EncodeToString([]byte(expData))

	// Snip off any garbage at the end
	for c := 0; c < 2; c++ {
		if strings.HasSuffix(expData64, "=") {
			expData64 = expData64[0 : len(expData64)-1]
		}
	}

	usr.AddSendReqAction("Create quant (fit)",
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
		`{"msgId":3,"status":"WS_OK","quantCreateResp":{
		"status": {
				"jobId": "${IDSAVE=quantFitId}",
				"jobType": "JT_RUN_FIT",
				"requestorUserId": "${USERID}",
				"message": "${IGNORE}",
				"status": "STATING",
				"startUnixTimeSec": "${SECAGO=60}",
				"lastUpdateUnixTimeSec": "${SECAGO=60}"
			}
				
		}}`,
	)

	expectedUpdates := []string{
		fmt.Sprintf(`{"quantCreateUpd":{
			"resultData": "${REGEXMATCH=%v.+}"
		}}`, expData64),
	}

	usr.CloseActionGroup(expectedUpdates, maxRunTimeSec*1000)

	wstestlib.ExecQueuedActions(&usr)

	usr.AddSendReqAction("Check quant log",
		`{"quantLastOutputGetReq":{
			"piquantCommand": "quant",
			"scanId": "983561",
			"outputType": 2
		}}`,
		// NOTE: had to convert the , to . so we dont break the parser thats reading REGEXMATCH=...
		`{"msgId":4,"status":"WS_OK","quantLastOutputGetResp":{
			"output": "${REGEXMATCH=-+\nPIQUANT   Quantitative X-ray Fluorescence Analysis\nWritten for PIXL. the Planetary Instrument for X-ray Lithochemistry\n3.2.16-master   W. T. Elam.+}"
		}}`,
	)

	usr.AddSendReqAction("Check quant output",
		`{"quantLastOutputGetReq":{
			"piquantCommand": "quant",
			"scanId": "983561",
			"outputType": 1
		}}`,
		// NOTE: had to convert the , to . so we dont break the parser thats reading REGEXMATCH=...
		// Also replace the () in the string to prevent regex match from failing. Putting \( and \) in didn't help...
		`{"msgId":5,"status":"WS_OK","quantLastOutputGetResp":{
			"output": "${REGEXMATCH=   PIQUANT 3.2.16-master  Normal_Combined_AllPoints\nEnergy .keV.. meas. calc. bkg. sigma. residual. DetCE. Ti_K. Ca_K. Rh_K_coh. Rh_L_coh. Rh_K_inc. Pileup. Rh_L_coh_Lb1\n-0.0154067.+}"
		}}`,
	)

	usr.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&usr)
}
