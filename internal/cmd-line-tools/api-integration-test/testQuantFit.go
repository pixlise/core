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

	exp64 := base64.StdEncoding.EncodeToString([]byte("   PIQUANT 3.2.16-master  Normal_Combined_AllPoints\nEnergy (keV), meas, calc, bkg, sigma, residual, DetCE, Ti_K, Ca_K, Rh_K_coh, Rh_L_coh, Rh_K_inc, Pileup, Rh_L_coh_Lb1\n-0.0154067, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n-0.0075045, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0\n0.000397676, 0, 0, 0, 1.41421, 0, 0, 0, 0, 0, 0, 0, 0, 0"))

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
