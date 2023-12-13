package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/auth0login"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

const imagePath = "images/download/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"

var imageGetJWT string

// Must be called before connecting to web socket
func testImageGet_PreWS(apiHost string) {
	var err error
	imageGetJWT, err = auth0login.GetJWT(test1Username, test1Password,
		auth0Params.ClientId, auth0Params.Secret, auth0Params.Domain, "http://localhost:4200/authenticate", auth0Params.Audience, "openid profile email")
	if err != nil {
		log.Fatalln(err)
	}

	testImageGet_NoJWT(apiHost)
	testImageGet_BadPath(apiHost, imageGetJWT)

	scanId := seedDBScanData(scan_Naltsos)
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)
	seedImages()
	seedImageLocations()
	seedImageFile(path.Base(imagePath), scanId, apiDatasetBucket)

	testImageGet_NoMembership(apiHost, imageGetJWT)
}

func seedImageFile(fileName string, scanId string, bucket string) {
	data, err := os.ReadFile("./test-files/" + fileName)
	if err != nil {
		log.Fatalln(err)
	}

	// Upload it where we need it for the test
	s3Path := filepaths.GetImageFilePath(path.Join(scanId, fileName))
	err = apiStorageFileAccess.WriteObject(bucket, s3Path, data)
	if err != nil {
		log.Fatalln(err)
	}
}

/*
// By now we should have cached member details in API
func testImageGet_PostWS(apiHost string) {
	seedDBScanData(scan_Naltsos)
	seedImages()
	seedImageLocations()

	// Ensure a web socket connection has been established
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Read non existant config",
		`{"detectorConfigReq":{"id": "non-existant"}}`,
		`{"msgId":1,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant not found",
			"detectorConfigResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// TODO: should seed user groups and configure one explicitly here for the test user to have access

	testImageGet_OK(apiHost, imageGetJWT)
}*/

func failIf(cond bool, err error) {
	if cond {
		caller := wstestlib.GetCaller(2)
		log.Fatalf("FAILED AT %v: %v", caller, err)
	}
}

func testImageGet_NoJWT(apiHost string) {
	resp, err := http.Get("http://" + path.Join(apiHost, imagePath))
	failIf(err != nil, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	failIf(string(body) != "Token not found\n" || resp.StatusCode != 500, fmt.Errorf("Unexpected response! Status %v, body: %v", resp.StatusCode, string(body)))
}

func doGet(scheme string, apiHost string, path string, query string, jwt string) (int, []byte, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	wsConnectUrl := url.URL{Scheme: scheme, Host: apiHost, Path: path, RawQuery: query}

	client := &http.Client{}
	req, err := http.NewRequest("GET", wsConnectUrl.String(), nil)
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

func testImageGet_BadPath(apiHost string, jwt string) {
	status, body, err := doGet("http", apiHost, "images/download/non-existant", "", jwt)

	failIf(err != nil, err)
	failIf(string(body) != "404 page not found\n" || status != 404, fmt.Errorf("Unexpected response! Status %v, body: %v", status, string(body)))
}

func testImageGet_NoMembership(apiHost string, jwt string) {
	status, body, err := doGet("http", apiHost, imagePath, "", jwt)

	failIf(err != nil, err)
	failIf(string(body) != "User group membership not found, can't determine permissions\n" || status != 400, fmt.Errorf("Unexpected response! Status %v, body: %v", status, string(body)))
}

func testImageGet_OK(apiHost string, jwt string) {
	status, body, err := doGet("http", apiHost, imagePath, "", jwt)
	failIf(err != nil, err)
	img, format, err := image.Decode(bytes.NewReader(body))
	var imgW, imgH int
	if img != nil {
		imgW = img.Bounds().Max.X
		imgH = img.Bounds().Max.Y
	}
	failIf(err != nil || format != "png" || status != 200 || imgW != 752 || imgH != 580,
		fmt.Errorf("Bad image response! Status %v, format %v image: %vx%v. Error: %v", status, format, imgW, imgH, err),
	)
}

func testImageGetScaled_OK(apiHost string, jwt string, minWidthPx int, minWidthPxExpected int, minHeightPxExpected int) {
	status, body, err := doGet("http", apiHost, imagePath, fmt.Sprintf("minwidth=%v", minWidthPx), jwt)
	failIf(err != nil, err)
	failIf(status != 200, fmt.Errorf("Unexpected status: %v. Body: %v", status, string(body)))
	/*
		fs := fileaccess.FSAccess{}
		err = fs.WriteObject("thumb.png", "", body)
	*/
	img, format, err := image.Decode(bytes.NewReader(body))
	failIf(err != nil, err)

	failIf(format != "png" || img.Bounds().Max.X != minWidthPxExpected || img.Bounds().Max.Y != minHeightPxExpected,
		fmt.Errorf("Bad image response! Expected %vx%v, got: status %v, body %v image: %vx%v. Error: %v", minWidthPxExpected, minHeightPxExpected, status, format, img.Bounds().Max.X, img.Bounds().Max.Y, err),
	)
}
