package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pixlise/core/v4/core/wstestlib"
)

var uploadedBreadboardScanId = "TEST_breadboard_upload"

func testImports(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Check if our upload exists (maybe last test failed?)
	/*u1.AddSendReqAction("List scans",
		`{"scanListReq":{}}`,
		`{"msgId":1,
			"status":"WS_OK",
			"scanListResp":{"scans${LIST,MODE=LENGTH,MINLENGTH=1}": []}}`,
	)*/
	u1.AddSendReqAction("List scans",
		`{"scanListReq":{}}`,
		`{"msgId":1,
			"status":"WS_OK",
			"scanListResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 2000)
	wstestlib.ExecQueuedActions(&u1)

	// Verify autoshare works
	u1.AddSendReqAction("Set upload autoshare empty, should find it deleted",
		fmt.Sprintf(`{"scanAutoShareWriteReq":{"entry": {"id":"%v"}}}`, u1.GetUserId()),
		`{"msgId":2,"status":"WS_OK","scanAutoShareWriteResp":{}}`,
	)

	u1.AddSendReqAction("Get upload autoshare, should be not found",
		fmt.Sprintf(`{"scanAutoShareReq":{"id":"%v"}}`, u1.GetUserId()),
		fmt.Sprintf(`{"msgId":3,"status":"WS_NOT_FOUND","errorText":"%v not found","scanAutoShareResp":{}}`, u1.GetUserId()),
	)

	u1.AddSendReqAction("Set upload autoshare",
		fmt.Sprintf(`{"scanAutoShareWriteReq":{"entry": {"id":"%v", "viewers": {"groupIds": ["group1", "group2"]}}}}`, u1.GetUserId()),
		`{"msgId":4,"status":"WS_OK","scanAutoShareWriteResp":{}}`,
	)

	u1.AddSendReqAction("Get upload autoshare",
		fmt.Sprintf(`{"scanAutoShareReq":{"id":"%v"}}`, u1.GetUserId()),
		`{"msgId":5,"status":"WS_OK","scanAutoShareResp": {"entry": {"id":"${USERID}", "viewers": {"groupIds": ["group1", "group2"]}}}}`,
	)

	u1.AddSendReqAction("Set upload autoshare to self",
		fmt.Sprintf(`{"scanAutoShareWriteReq":{"entry": {"id":"%v", "editors": {"userIds": ["%v"]}}}}`, u1.GetUserId(), u1.GetUserId()),
		`{"msgId":6,"status":"WS_OK","scanAutoShareWriteResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	testScanImport(apiHost, u1)
	testImageImport(u1)
	testScanDelete(u1)

	// TODO: Test/simulate a FM downlink
}

func doPut(scheme string, apiHost string, path string, query string, body io.Reader, jwt string) (int, []byte, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	wsConnectUrl := url.URL{Scheme: scheme, Host: apiHost, Path: path, RawQuery: query}

	client := &http.Client{}
	req, err := http.NewRequest("PUT", wsConnectUrl.String(), body)
	req.Header.Set("Authorization", "Bearer "+jwt)
	if err != nil {
		return 0, []byte{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, []byte{}, err
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, []byte{}, err
	}

	return resp.StatusCode, b, err
}

func testScanImport(apiHost string, u1 wstestlib.ScriptedTestUser) {
	// Test bad upload scenarios
	u1.AddSendReqAction("Upload scan for format (should fail, bad format)",
		`{"scanUploadReq":{"id":"bad1", "format": "dtu-breadboard", "zipFileName": "abcd.zip"}}`,
		`{"msgId":7,
			"status":"WS_BAD_REQUEST",
			"errorText":"Unexpected format: \"dtu-breadboard\"",
			"scanUploadResp":{}}`,
	)

	u1.AddSendReqAction("Upload scan for format (should fail, bad zip)",
		`{"scanUploadReq":{"id":"bad1", "format": "dtu-breadboard", "zipFileName": "abcd.zip"}}`,
		`{"msgId":8,
			"status":"WS_BAD_REQUEST",
			"errorText":"Unexpected format: \"dtu-breadboard\"",
			"scanUploadResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Now upload the zip file!
	f, err := os.Open("./test-files/scan-uploads/upload_kingscourt.zip")
	if err != nil {
		log.Fatal(err)
	}
	statusCode, _, err := doPut("http", apiHost, "/scan", "scan=upload1&filename=abcd.zip", f, imageGetJWT)
	if err != nil {
		log.Fatal(err)
	}
	if statusCode != 200 {
		log.Fatalf("Status code for image put was: %v", statusCode)
	}

	// Test a simple upload of a breadboard zip
	u1.AddSendReqAction("Upload scan (should succeed)",
		`{"scanUploadReq":{"id":"upload1", "format": "sbu-breadboard", "zipFileName": "abcd.zip"}}`,
		`{"msgId":9,"status":"WS_OK","scanUploadResp":{
			"jobId": "${IDSAVE=breadboardImportJobId}"
		}}`,
	)

	u1.CloseActionGroup([]string{
		`{"scanUploadUpd":{
			"status": {
				"jobId": "${IDCHK=breadboardImportJobId}",
				"logId": "${IGNORE}",
				"message": "Starting importer",
				"status": "STARTING",
				"startUnixTimeSec": "${SECAGO=8}",
				"lastUpdateUnixTimeSec": "${SECAGO=8}"
			}
		}}`,
		/*`{"scanUploadUpd":{
			"status": {
				"jobId": "${IDCHK=breadboardImportJobId}",
				"logId": "${IGNORE}",
				"message": "Cores/Node: 4",
				"status": "PREPARING_NODES",
				"startUnixTimeSec": "${SECAGO=8}",
				"lastUpdateUnixTimeSec": "${SECAGO=8}"
			}
		}}`,*/
		`{"scanUploadUpd":{
			"status": {
				"jobId": "${IDCHK=breadboardImportJobId}",
				"logId": "${IGNORE}",
				"message": "Importing Files",
				"status": "RUNNING",
				"startUnixTimeSec": "${SECAGO=8}",
				"lastUpdateUnixTimeSec": "${SECAGO=8}"
			}
		}}`,
		`{"scanUploadUpd":{
			"status": {
				"jobId": "${IDCHK=breadboardImportJobId}",
				"logId": "${IGNORE}",
				"message": "Imported successfully",
				"status": "COMPLETE",
				"startUnixTimeSec": "${SECAGO=8}",
				"lastUpdateUnixTimeSec": "${SECAGO=8}",
				"endUnixTimeSec": "${SECAGO=8}"
			}
		}}`,
		`{"scanListUpd": {}}`,
	}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Make sure it appeared in the list of scans
	u1.AddSendReqAction("List scans expecting new upload",
		`{"scanListReq":{}}`,
		`{"msgId":10,
			"status":"WS_OK",
			"scanListResp":{"scans": [
				{
					"id": "${IDSAVE=breadboardImportScanId}",
					"title": "${IDSAVE=breadboardImportScanTitle}",
					"dataTypes": [
						{
							"dataType": "SD_XRF",
							"count": 8
						}
					],
					"instrument": "SBU_BREADBOARD",
					"instrumentConfig": "StonyBrookBreadboard",
					"timestampUnixSec": "${SECAGO=10}",
					"meta": {
						"DriveID": "0",
						"RTT": "",
						"SCLK": "0",
						"SOL": "",
						"Site": "",
						"SiteID": "0",
						"Target": "",
						"TargetID": "0"
					},
					"contentCounts": {
						"BulkSpectra": 2,
						"DwellSpectra": 0,
						"MaxSpectra": 2,
						"NormalSpectra": 8,
						"PseudoIntensities": 0
					},
					"creatorUserId": "${USERID}"
				}
			]
		}}`,
	)

	u1.CloseActionGroup([]string{}, 2000)
	wstestlib.ExecQueuedActions(&u1)
}

func testImageImport(u1 wstestlib.ScriptedTestUser) {
	// Test a simple upload of an image to breadboard, along with transform adjustment
	u1.AddSendReqAction("Upload image to uploaded scan",
		`{"imageUploadReq":{
			"name": "kingscourtctx.jpg",
			"imageData": "${FILEBYTES=./test-files/scan-uploads/kingscourtctx.jpg}",
			"associatedScanIds": ["${IDLOAD=breadboardImportScanId}"],
			"originScanId": "${IDLOAD=breadboardImportScanId}"
		}}`,
		`{"msgId":11,
			"status":"WS_OK",
			"imageUploadResp":{}}`,
	)

	u1.AddSendReqAction("List scans expecting new upload+1 image",
		`{"scanListReq":{}}`,
		`{"msgId":12,
			"status":"WS_OK",
			"scanListResp":{"scans": [
				{
					"id": "${IDSAVE=breadboardImportScanId}",
					"title": "${IDSAVE=breadboardImportScanTitle}",
					"dataTypes": [
						{
							"dataType": "SD_XRF",
							"count": 8
						}
					],
					"instrument": "SBU_BREADBOARD",
					"instrumentConfig": "StonyBrookBreadboard",
					"timestampUnixSec": "${SECAGO=10}",
					"meta": {
						"DriveID": "0",
						"RTT": "",
						"SCLK": "0",
						"SOL": "",
						"Site": "",
						"SiteID": "0",
						"Target": "",
						"TargetID": "0"
					},
					"contentCounts": {
						"BulkSpectra": 2,
						"DwellSpectra": 0,
						"MaxSpectra": 2,
						"NormalSpectra": 8,
						"PseudoIntensities": 0
					},
					"creatorUserId": "${USERID}"
				}
			]
		}}`,
	)

	u1.CloseActionGroup([]string{}, 2000)
	wstestlib.ExecQueuedActions(&u1)
}

func testScanDelete(u1 wstestlib.ScriptedTestUser) {
	// Now delete it
	u1.AddSendReqAction("Delete new upload (should fail, bad verification)",
		`{"scanDeleteReq":{"scanId": "${IDLOAD=breadboardImportScanId}", "scanNameForVerification": "My new upload"}}`,
		fmt.Sprintf(`{"msgId":13,
			"status": "WS_BAD_REQUEST",
			"errorText": "Specified title did not match scan title of: \"%v\"",
			"scanDeleteResp":{}}`, wstestlib.GetIdCreated("breadboardImportScanId")),
	)

	u1.AddSendReqAction("Delete new upload",
		`{"scanDeleteReq":{"scanId": "${IDLOAD=breadboardImportScanId}", "scanNameForVerification": "${IDLOAD=breadboardImportScanId}"}}`,
		`{"msgId":14,
			"status":"WS_OK",
			"scanDeleteResp":{}}`,
	)

	// Check
	u1.AddSendReqAction("List scans expecting new upload",
		`{"scanListReq":{}}`,
		`{"msgId":15,
			"status":"WS_OK",
			"scanListResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 2000)
	wstestlib.ExecQueuedActions(&u1)
}
