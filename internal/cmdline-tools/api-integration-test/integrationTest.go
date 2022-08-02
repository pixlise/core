// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	apiNotifications "github.com/pixlise/core/core/notifications"

	"github.com/pixlise/core/api/endpoints"
	"github.com/pixlise/core/core/auth0login"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/utils"
	protos "github.com/pixlise/core/generated-protos"

	"google.golang.org/protobuf/proto"
)

func generateURL(environment string) string {
	return "https://" + environment + "-api.review.pixlise.org"
}

// Checks API version is valid (just as a string, checks with regex, does not check against an expected deployed version!)
func checkAPIVersion(environment string, expectedVersion string) error {
	resp, err := http.Get(generateURL(environment) + "/version")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result endpoints.ComponentVersionsGetResponse
	err = json.Unmarshal(body, &result)
	//fmt.Printf("%v", string(body))

	if err != nil {
		return err
	}
	theApiVersion := result.Components[0].Version

	// regex for valid API version
	if len(expectedVersion) <= 0 {
		// We don't have an expected one set, so just do a regex check
		var versionRegex = regexp.MustCompile(`(v?\d{1,2}.\d{1,2}.\d{1,2})|(N/A - Local build)`)
		if versionRegex.MatchString(theApiVersion) == false {
			return fmt.Errorf("Error fetching API version, got: %v", theApiVersion)
		}

		fmt.Printf(" Accepting version as valid: %v\n", theApiVersion)
	} else if theApiVersion != expectedVersion {
		return fmt.Errorf("Expected API version '%v', got: '%v'", expectedVersion, theApiVersion)
	}

	fmt.Printf("Version matched: %v", theApiVersion)

	return nil
}

// Requests all dataset summaries, and verifies they're well formatted
func requestAndValidateDatasets(JWT string, environment string) ([]datasetModel.APIDatasetSummary, error) {
	var result = []datasetModel.APIDatasetSummary{}
	req, err := http.NewRequest("GET", generateURL(environment)+"/dataset", nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+JWT)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}

	//storing errors in a dynamic slice to allow for multiple errors across multiple datasets
	//to be outputted to better diagnosis issues
	errCount := 0
	downloadLimit := 1
	for c, item := range result {
		dserror := isValidDatasetItem(item, JWT)

		if dserror != nil {
			fmt.Printf("%v\n", dserror)
			errCount++
		}

		if c >= downloadLimit {
			break
		}
	}

	if errCount > 0 {
		return result, errors.New("Dataset query failed")
	}

	fmt.Printf(" Received %v dataset summaries\n", len(result))
	return result, nil
}

func getQuantStatus(JWT string, environment string, datasetID string, jobID string) (string, error) {
	getReq, err := http.NewRequest("GET", generateURL(environment)+"/quantification/"+datasetID, nil)
	if err != nil {
		return "", err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return "", err
	}
	defer getResp.Body.Close()
	getBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return "", err
	}

	if getResp.Status != "200 OK" {
		return "", fmt.Errorf("Failed to get quant status for dataset %v: %v, response: %v", datasetID, getResp.Status, string(getBody))
	}

	var result endpoints.QuantListingResponse
	err = json.Unmarshal(getBody, &result)
	if err != nil {
		return "", err
	}

	// Finding where the current job is 7lq6mbw4sf2e8ehf
	jobIndex := -1
	for i := range result.Summaries {
		if result.Summaries[i].JobStatus.JobID == jobID {
			jobIndex = i
			break
		}
	}

	if jobIndex < 0 {
		return "", fmt.Errorf("Failed to find quant job: %v in quant list", jobID)
		//return "unknown", nil
	}

	return string(result.Summaries[jobIndex].JobStatus.Status), nil
}

// Runs a quantification and checks that it steps through the expected states
func quantVerification(JWT string, environment string, datasetID string, pmcList []int, elementList string, detectorConfig string, quantName string) (string, error) {
	pmcListOfStrings := "["
	for c, pmc := range pmcList {
		if c > 0 {
			pmcListOfStrings += ","
		}
		pmcListOfStrings += strconv.Itoa(pmc)
	}
	pmcListOfStrings += "]"

	var jsonStr = `{"name":"` + quantName + `","pmcs":` + pmcListOfStrings + `,"elements":` + elementList + `,"parameters":"-q,pPIETXCFsr -b,0,12,60,910,2800,16","detectorConfig":` + detectorConfig + `,"runTimeSec":60,"roiID":null,"elementSetID":"","quantMode":"Combined"}`
	req, err := http.NewRequest("POST", generateURL(environment)+"/quantification/"+datasetID, bytes.NewBuffer([]byte(jsonStr)))
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := string(body)
	if resp.Status != "200 OK" {
		return "", fmt.Errorf("Error starting quantification: %v, response: %v", resp.Status, bodyStr)
	}

	// Get jobID from response body
	jobID := strings.TrimSpace(bodyStr)
	jobID = strings.ReplaceAll(jobID, `"`, "")

	// Now we wait, job should step through various states
	expectedJobStates := []string{"starting", "preparing_nodes", "nodes_running", "gathering_results", "complete"}

	// We allow a max quant run time of 5 minutes, should finish wayyy before then
	const maxRunTimeSec = 600
	const checkInterval = 10
	nextCheckInterval := 90 // we wait a bit longer for the first go, wasn't that reliable after 10sec.
	lastStatus := ""
	for c := 0; c < maxRunTimeSec/checkInterval; c++ {
		time.Sleep(time.Duration(nextCheckInterval) * time.Second)

		nextCheckInterval = checkInterval // subsequent checks are more frequent

		status, err := getQuantStatus(JWT, environment, datasetID, jobID)

		// If we ever fail to get a status back, stop here
		if err != nil {
			return "", fmt.Errorf("getQuantStatus failed for dataset %v: %v, response: %v", datasetID, resp.Status, err)
		}

		// Make sure the state returned is one of the ones we expect
		validStatus := false
		for _, expStatus := range expectedJobStates {
			if status == expStatus {
				validStatus = true
				break
			}
		}

		if !validStatus {
			return "", fmt.Errorf("Found unexpected job status '%v' for dataset %v, job id: %v", status, datasetID, jobID)
		}

		if status != lastStatus {
			now := time.Now().Format(timeFormat)
			fmt.Printf(" %v   Quant job: %v for dataset: %v - status changed to: %v\n", now, jobID, datasetID, status)
			lastStatus = status
		}

		if status == "complete" {
			break
		}
	}

	// If the status never completed in our wait time, that's an error
	if lastStatus != "complete" {
		return "", fmt.Errorf("Quant job: %v for dataset %v: timed out!", jobID, datasetID)
	}

	return jobID, err
}

//deletes quant after running it
func deleteQuant(JWT string, jobID string, environment string, datasetID string) error {
	req, err := http.NewRequest("DELETE", generateURL(environment)+"/quantification/"+datasetID+"/"+jobID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.Status != "200 OK" {
		return fmt.Errorf("Failed to delete quantification: %v, response: %v", resp.Status, string(body))
	}
	return nil
}

//checks the exporter works on the hardcoded datasets added below
func verifyExport(JWT string, jobID string, environment string, datasetID string, fileName string, fileIds []string) error {
	jsonStr := `{"fileName": "` + fileName + `", "quantificationId":"` + jobID + `", "fileIds":[`
	for c, file := range fileIds {
		if c > 0 {
			jsonStr += ","
		}
		jsonStr += "\"" + file + "\""
	}

	jsonStr += `]}`
	var jsonBytes = []byte(jsonStr)
	req, err := http.NewRequest("POST", generateURL(environment)+"/export/files/"+datasetID, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{
		Timeout: 0, //time.Second * 300,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read zip body: %v", err)
	}

	if resp.Status != "200 OK" {
		return fmt.Errorf("Export status fail: %v, response: %v", resp.Status, string(body))
	}

	// Check response headers
	expContentDisposition := `attachment; filename="` + fileName + `"`
	if resp.Header["Content-Disposition"][0] != expContentDisposition {
		return fmt.Errorf("Missing Content-Disposition from response header")
	}

	if resp.Header["Content-Length"][0] == "0" {
		return fmt.Errorf("Unexpected content length")
	}

	// Check body zip contents
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("Failed to read zip content: %v", err)
	}

	// Check that the files seem remotely correct...
	fileNamePrefix := fileName[0 : len(fileName)-4]
	expectedFileNames := map[string]bool{
		fileNamePrefix + "-map-by-PIQUANT.csv":                true,
		fileNamePrefix + "-beam-locations.csv":                true,
		fileNamePrefix + "-unquantified-weight-pct.csv":       true,
		fileNamePrefix + "-roi-pmcs.csv":                      true,
		fileNamePrefix + "-Normal-BulkSum ROI All Points.csv": true,
		fileNamePrefix + "-Normal ROI All Points.csv":         true,
		//fileNamePrefix + "-Dwell-BulkSum ROI All Points.csv":  true,
		//fileNamePrefix + "-Dwell ROI All Points.csv":          true,
	}
	/*hasPNG hasTIF, hasTXT,*/ hasCSV := false //, false, false, false

	for _, zipFile := range zipReader.File {
		/*if !hasPNG && strings.HasSuffix(zipFile.Name, ".png") {
			hasPNG = true
		}
		if !hasTIF && strings.HasSuffix(zipFile.Name, ".tif") {
			hasTIF = true
		}
		if !hasTXT && strings.HasSuffix(zipFile.Name, ".txt") {
			hasTXT = true
		}*/
		if !hasCSV && strings.HasSuffix(zipFile.Name, ".csv") {
			hasCSV = true
		}

		_, ok := expectedFileNames[zipFile.Name]
		if ok {
			expectedFileNames[zipFile.Name] = false
		}
	}

	// If didn't see any files of these types...
	if /*hasPNG == false || hasTIF == false || hasTXT == false ||*/ hasCSV == false {
		return fmt.Errorf("One or more files missing from export zip")
	}

	// If didn't see any of the expected file names...
	for k, v := range expectedFileNames {
		if v {
			return fmt.Errorf("Export zip did not contain: %v", k)
		}
	}

	return err
}

// Ensures dataset summary has valid fields and has no errors. Returns an error, or nil if no error
func isValidDatasetItem(dataset datasetModel.APIDatasetSummary, JWT string) error {
	datasetIDPat := regexp.MustCompile(`.+`)
	if !datasetIDPat.MatchString(dataset.DatasetID) {
		return errors.New("Missing DatasetID")
	}

	if len(dataset.ContextImage) == 0 {
		// Context image count should be 0
		// For now, that 1 dataset we have without a context image set generates with count of 1 so allow this
		if dataset.ContextImages != 0 && dataset.ContextImages != 1 {
			return fmt.Errorf("Expected 0 Context Images in Dataset: %v", dataset.DatasetID)
		}
	} else {
		// Validate the image is a file name, and context image count > 0
		contextImagePat := regexp.MustCompile(`.+(.jpg|.png)`)
		if len(dataset.ContextImage) > 0 && contextImagePat.MatchString(dataset.ContextImage) == false {
			return fmt.Errorf("Missing Context Image in Dataset: %v", dataset.DatasetID)
		}

		if dataset.ContextImages == 0 {
			return fmt.Errorf("Expected > 0 Context Images in Dataset: %v", dataset.DatasetID)
		}
	}

	if dataset.DataFileSize == 0 {
		return fmt.Errorf("Invalid Data File Size in Dataset: %v", dataset.DatasetID)
	}

	if len(dataset.DetectorConfig) <= 0 {
		return fmt.Errorf("Missing Detector Config in Dataset: %v", dataset.DatasetID)
	}

	datasetLinkPat := regexp.MustCompile(`https://.+/dataset`)
	datasetLink := datasetLinkPat.FindString(dataset.DataSetLink)
	if datasetLink == "" {
		return fmt.Errorf("Missing Dataset Image Link in Dataset: %v", dataset.DatasetID)
	}

	// If no context image, expect no link... otherwise expect both
	if len(dataset.ContextImage) > 0 {
		// Expect link to be set correctly too
		contextImageLinkPat := regexp.MustCompile(`https://.*(.png|.jpg)`)
		contextImageLink := contextImageLinkPat.FindString(dataset.ContextImageLink)
		if len(contextImageLink) <= 0 {
			return fmt.Errorf("Missing Context Image Link in Dataset: %v", dataset.DatasetID)
		}
	} else {
		// Context image is empty, ensure link is too
		if len(dataset.ContextImageLink) > 0 {
			return fmt.Errorf("Context image is empty, expected link to be empty also, got: %v", dataset.ContextImageLink)
		}
	}

	return nil
}

func checkFileDownload(JWT string, url string) ([]byte, error) {
	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()
	body, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	if getResp.Status != "200 OK" {
		return nil, fmt.Errorf("Download status fail: %v, response: %v", getResp.Status, string(body))
	}

	headerBytes, err := strconv.Atoi(getResp.Header["Content-Length"][0])
	if err != nil {
		return nil, fmt.Errorf("Failed to read content-length: %v", err)
	}

	//comparing bytes of body and content length header
	if headerBytes != len(body) {
		return nil, fmt.Errorf("Content-length: %v does not match downloaded size: %v", headerBytes, len(body))
	}

	return body, nil
}

func getAlerts(JWT string, environment string) ([]apiNotifications.UINotificationObj, error) {
	getReq, err := http.NewRequest("GET", generateURL(environment)+"/notification/alerts", nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()
	body, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	if getResp.Status != "200 OK" {
		return nil, fmt.Errorf("Alerts status fail: %v, response: %v", getResp.Status, string(body))
	}

	var alerts []apiNotifications.UINotificationObj
	err = json.Unmarshal(body, &alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func runAthenaTests(env string) error {
	testusers := []string{"600f2a0806b6c70071d3d999"}
	notifications := apiNotifications.NotificationStack{
		Notifications: nil,
		FS:            nil,
		Bucket:        "",
		Track:         nil,
		AdminEmails:   nil,
		Environment:   env,
		Logger:        nil,
		Backend:       "",
	}
	res, err := notifications.GetSubscribersByTopicID(testusers, "deliberate-non-topic")
	if err != nil {
		return err
	}
	if len(res) > 0 {
		return errors.New("Got notifications back for invalid topic")
	}

	res, err = notifications.GetSubscribersByTopicID(testusers, "test-data-source")
	if err != nil {
		return err
	}
	if len(res) != 1 {
		return errors.New("Expected 1 item back for test-data-source")
	}

	if res[0].Userid != testusers[0] {
		return fmt.Errorf("got unexpected user id back for test-data-source topic: %v", res[0].Userid)
	}

	res, err = notifications.GetAllUsers()
	if err != nil {
		return err
	}

	if len(res) <= 0 {
		return fmt.Errorf("Got no users back")
	}

	res, err = notifications.GetSubscribersByTopic("new-dataset-available")
	if err != nil {
		return err
	}

	if len(res) <= 0 {
		return fmt.Errorf("expected more items for new-dataset-available")
	}

	return nil
}

const timeFormat = "15:04:05" // "2006-01-02 15:04:05"
var lastStartedTestName = ""

func printTestStart(name string) string {
	timeNow := time.Now().Format(timeFormat)

	fmt.Println("---------------------------------------------------------")
	fmt.Printf(" %v TEST: %v\n", timeNow, name)
	//fmt.Println("---------------------------------------------------------")

	lastStartedTestName = name

	// Not even sure why this is returned anymore, seems it's not always passed as
	// name param to printTestResult, but we use lastStartedTestName now anyway
	return name
}

var failedTestNames = []string{}

func printTestResult(err error, name string) {
	suffix := ""
	if len(name) > 0 {
		suffix = " [" + name + "]"
	}

	timeNow := time.Now().Format(timeFormat)

	if err == nil {
		fmt.Printf(" %v  PASS%v", timeNow, suffix)
	} else {
		fmt.Printf(" %v  FAILED%v: %v\n", timeNow, suffix, err)
		failedTestNames = append(failedTestNames, lastStartedTestName)
	}
	fmt.Println("")
}

func runQuantificationTestsForDataset(
	JWT string, environment string, datasetID string, detectorConfig string, pmcList []int, elementList string, quantName string, exportColumns []string) string {
	resultPrint := printTestStart(fmt.Sprintf("Quantification of dataset: %v with config: %v, PMC count: %v", datasetID, detectorConfig, len(pmcList)))
	jobID, err := quantVerification(JWT, environment, datasetID, pmcList, elementList, detectorConfig, quantName)
	printTestResult(err, resultPrint)
	if err != nil {
		// If quant failed, don't try the rest of these...
		return ""
	}

	resultPrint = printTestStart(fmt.Sprintf("Export of quantification: %v", jobID))
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

	err = verifyExport(JWT, jobID, environment, datasetID, "export-test.zip", fileIds)
	printTestResult(err, resultPrint)

	// Download the quant file
	resultPrint = printTestStart(fmt.Sprintf("Download and verify quantification: %v", jobID))
	quantBytes, err := checkFileDownload(JWT, "https://api"+environment+".pixlise.org/quantification/download/"+datasetID+"/"+jobID)

	if err == nil {
		// Downloaded, so check that we have the right # of PMCs and elements...
		err = checkQuantificationContents(quantBytes, pmcList, exportColumns)
	}
	printTestResult(err, resultPrint)

	resultPrint = printTestStart(fmt.Sprintf("Delete generated quantification: %v for dataset: %v", jobID, datasetID))
	err = deleteQuant(JWT, jobID, environment, datasetID)
	printTestResult(err, resultPrint)

	return jobID
}

func checkQuantificationContents(quantBytes []byte, expPMCList []int, expOutputElements []string) error {
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
	expPMCs := map[int]bool{} // TODO: REFACTOR: Need generic utils.SetStringsInMap for this...
	for _, pmc := range expPMCList {
		expPMCs[pmc] = true
	}

	expElements := map[string]bool{}
	utils.SetStringsInMap(expOutputElements, expElements)

	keys := make([]int, 0, len(q.LocationSet[0].Location))

	for _, loc := range q.LocationSet[0].Location {
		pmc := loc.Pmc
		keys = append(keys, int(pmc))

		val, pmcExpected := expPMCs[int(pmc)]
		if !pmcExpected {
			return fmt.Errorf("Quant contained unexpected PMC: %v", pmc)
		}
		if !val {
			return fmt.Errorf("Quant contained duplicated PMC: %v", pmc)
		}
		expPMCs[int(pmc)] = false
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
}

func main() {
	rand.Seed(time.Now().UnixNano())
	startupTime := time.Now()

	if len(os.Args) != 6 {
		fmt.Println("Arguments: environment, user, password, auth0_user_id, expected_version")
		fmt.Println("  Where:")
		fmt.Println("  - environment is one of [dev, staging, prod]")
		fmt.Println("  - user/password are the test account Auth0 login details")
		fmt.Println("  - auth0_user_id is the users Auth0 user id (without Auth0| prefix)")
		fmt.Println("  - expected_version is what we expect the API to return, eg 0.0.35-ALPHA. Or nil if don't care")
		os.Exit(1)
	}

	// Check arguments
	var rawEnv = os.Args[1]
	var environment string
	if rawEnv == "dev" {
		environment = "dev"
	} else if rawEnv == "staging" {
		environment = "-staging"
	} else if rawEnv == "prod" {
		environment = ""
	} else {
		fmt.Println("Environment should be one of: dev, staging, or prod.")
		os.Exit(1)
	}

	fmt.Println("Running integration test for env: " + rawEnv)

	var username = os.Args[2]
	var password = os.Args[3]
	var auth0UserID = os.Args[4]
	var expectedVersion = os.Args[5]

	// If expectedVersion is nil, clear it
	if expectedVersion == "nil" {
		expectedVersion = ""
	}

	printTestStart("API Version")
	err := checkAPIVersion(environment, expectedVersion)
	printTestResult(err, "")
	if err != nil {
		// If API version call is broken, probably everything is...
		os.Exit(1)
	}

	printTestStart("Athena queries")
	err = runAthenaTests(rawEnv)
	printTestResult(err, "")

	// TODO: Maybe we need to change this if we go open source?
	printTestStart("Getting JWT (Auth0 login)")
	JWT, err := auth0login.GetJWT(username, password, "***REMOVED***", "***REMOVED***", "pixlise.au.auth0.com", "http://localhost:4200/authenticate", "pixlise-backend", "openid profile email")
	if err == nil && len(JWT) <= 0 {
		err = errors.New("JWT returned is empty")
	}
	printTestResult(err, "")
	if err != nil {
		// No point continuing, we couldn't log in!
		os.Exit(1)
	}

	// Check to see if there are alerts. If some come back we warn and check again, as maybe prev unit test run has left some over?
	printTestStart("Alerts (Before quantification tests)")
	preQuantAlerts, err := getAlerts(JWT, environment)
	if len(preQuantAlerts) > 0 {
		fmt.Printf(" WARNING: alerts came back with %v items. Will call again and verify it's cleared...\n", len(preQuantAlerts))
	}
	printTestResult(err, "")

	if len(preQuantAlerts) > 0 {
		// Re-check alerts, they should be empty now because the last call would've cleared them
		time.Sleep(3 * time.Second) // just in case...

		printTestStart("Alerts (Re-check)")
		alerts2, err2 := getAlerts(JWT, environment)
		if len(alerts2) > 0 {
			err2 = errors.New("Alerts expected to be empty after clearing")
		}
		printTestResult(err2, "")
	}

	printTestStart("Dataset listing")
	datasets, err := requestAndValidateDatasets(JWT, environment)
	printTestResult(err, "")
	if err != nil {
		os.Exit(1)
	}

	// Randomly pick a dataset and download its bin file and context image
	downloadTestIdx := rand.Int() % len(datasets)
	printTestStart(fmt.Sprintf("Downloading dataset binary file for: %v, id=%v", datasets[downloadTestIdx].Title, datasets[downloadTestIdx].DatasetID))
	_, err = checkFileDownload(JWT, datasets[downloadTestIdx].DataSetLink)
	printTestResult(err, "")

	if err == nil {
		printTestStart(fmt.Sprintf("Downloading dataset context image file for: %v, id=%v", datasets[downloadTestIdx].Title, datasets[downloadTestIdx].DatasetID))
		_, err = checkFileDownload(JWT, datasets[downloadTestIdx].ContextImageLink)
		printTestResult(err, "")
	}

	// Test quantifications on a few pre-determined datasets
	elementList := `["Ca","Ti"]`
	quantColumns := []string{"CaO_%", "TiO2_%"}
	detectorConfig := []string{`"PIXL/v5"`, `"PIXL/v5"`, `"Breadboard/v1"`}
	pmcsFor5x5 := []int{}
	for c := 4043; c < 5806; c++ {
		if c != 4827 {
			pmcsFor5x5 = append(pmcsFor5x5, c)
		}
	}
	pmcList := [][]int{{68, 69, 70, 71}, pmcsFor5x5, {68, 69, 70, 71}}
	datasetIDs := []string{"983561", "test-fm-5x5-full", "test-kingscourt"} // test-laguna was timing out because saving the high rest TIFFs took longer than 1 minute, which seems to be the test limit

	// NOTE: By using 2 of the same names, we also test that the delete
	// didn't leave something behind and another can't be named that way
	quantNameSuffix := utils.RandStringBytesMaskImpr(8)
	quantNames := []string{"integration-test-same-name-" + quantNameSuffix, "integration-test-5x5-" + quantNameSuffix, "integration-test-same-name-" + quantNameSuffix}

	quantJobIDs := []string{}
	for i, datasetID := range datasetIDs {
		jobID := runQuantificationTestsForDataset(JWT, environment, datasetID, detectorConfig[i], pmcList[i], elementList, quantNames[i], quantColumns)
		if jobID == "" {
			printTestResult(fmt.Errorf("No JOB ID Returned for quant execution %v", quantNames[i]), "")
		}
		quantJobIDs = append(quantJobIDs, jobID)
	}

	// Test quant failing by supplying an invalid detector config (missing the /version)
	printTestStart("Check quantification failure return values")
	jobID, err := quantVerification(JWT, environment, "983561", pmcList[0], elementList, `"PIXL/v5"`, quantNames[0])
	if jobID != "" && (err != nil && err.Error() != "Error starting quantification: 400 Bad Request, response: DetectorConfig not in expected format") {
		printTestResult(fmt.Errorf("Unexpected result when running invalid quant: %v", err), "")
	} else {
		printTestResult(nil, "")
	}

	// Check that the expected alerts were generated during quantifications
	// This a start & finish alert for each job ID...
	expAlerts := map[string]bool{}

	for c, jobId := range quantJobIDs {
		qj := fmt.Sprintf("Started Quantification: %v (id: %v). Click on Quant Tracker tab to follow progress.", quantNames[c], jobId)
		qjf := fmt.Sprintf("Quantification %v Processing Complete", quantNames[c])
		fmt.Printf("%v\n", qj)
		fmt.Printf("%v\n", qjf)
		expAlerts[qj] = true
		expAlerts[qjf] = true
	}

	printTestStart("Alerts (Post quantification tests)")
	postQuantAlerts, err := getAlerts(JWT, environment)

	if err == nil {
		// NOTE: this covers the case where there are duplicate alerts coming in and we don't consider that an error!
		if len(postQuantAlerts) < len(expAlerts) {
			err = fmt.Errorf("Alerts came back with '%v' items, expected '%v'", len(postQuantAlerts), len(expAlerts))
		} else {
			if len(postQuantAlerts) > len(expAlerts)+1 {
				fmt.Printf(" WARNING: Got '%v' alerts, expected '%v'\n", len(postQuantAlerts), len(expAlerts))
			}

			// Check that they all match what we're expecting:
			// - Time range is anywhere from our test startup to now
			// - Text we've got in expAlerts
			// - User ID is known
			// - Topic we can deduce...
			currTime := time.Now()

			for _, alert := range postQuantAlerts {
				if alert.Timestamp.Before(startupTime) || alert.Timestamp.After(currTime) {
					err = fmt.Errorf("Alert timestamp was unexpected: %v", alert.Timestamp)
					break
				}

				if alert.UserID != auth0UserID {
					err = fmt.Errorf("Alert user ID was unexpected: %v", alert.UserID)
					break
				}

				if _, ok := expAlerts[alert.Message]; !ok {
					err = fmt.Errorf("Alert message was unexpected: %v. Available Messages:\n", alert.Message)
					for k, _ := range expAlerts {
						fmt.Printf("Message: %v\n", k)
					}
					break
				}

				// We should be able to work out the topic based on message
				expTopic := "Quantification Processing Start"
				if strings.HasSuffix(alert.Message, "Processing Complete") {
					expTopic = "Quantification Processing Complete"
					break
				}

				if alert.Topic != expTopic {
					err = fmt.Errorf("Alert topic was unexpected: %v", alert.Topic)
					break
				}
			}
		}
	}

	if err != nil {
		// Print out what was received, to aid debugging
		fmt.Printf("Alerts received: +%v\n", postQuantAlerts)
	}

	printTestResult(err, "")

	fmt.Println("\n==============================")

	if rawEnv == "staging" || rawEnv == "prod" {

		printTestStart("OCS Integration Test")
		//err = runOCSTests()
		printTestResult(err, "")

		fmt.Println("\n==============================")

		printTestStart("Publish Integration Test")
		//err = runPublishTests()
		printTestResult(err, "")

		fmt.Println("\n==============================")
	}
	if len(failedTestNames) == 0 {
		fmt.Println("PASSED All Tests!")
		os.Exit(0)
	}

	fmt.Println("FAILED One or more tests:")
	for _, name := range failedTestNames {
		fmt.Printf("- %v\n", name)
	}
	os.Exit(1)
}
