// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"net/http"
	"strings"

	"github.com/pixlise/core/v4/api/notificationSender"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
)

var uploadImgPNGData = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x05, 0x08, 0x02, 0x00, 0x00, 0x00, 0x02, 0x0d, 0xb1, 0xb2, 0x00, 0x00, 0x01, 0x84, 0x69, 0x43, 0x43, 0x50, 0x49, 0x43, 0x43, 0x20, 0x70, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65, 0x00, 0x00, 0x28, 0x91, 0x7d, 0x91, 0x3d, 0x48, 0xc3, 0x50, 0x14, 0x85, 0x4f, 0xd3, 0x8a, 0xa2, 0x15, 0x11, 0x0b, 0x8a, 0x38, 0x64, 0xa8, 0x4e, 0x76, 0x51, 0x11, 0xc7, 0x52, 0xc5, 0x22, 0x58, 0x28, 0x6d, 0x85, 0x56, 0x1d, 0x4c, 0x5e, 0xfa, 0x07, 0x4d, 0x1a, 0x92, 0x14, 0x17, 0x47, 0xc1, 0xb5, 0xe0, 0xe0, 0xcf, 0x62, 0xd5, 0xc1, 0xc5, 0x59, 0x57, 0x07, 0x57, 0x41, 0x10, 0xfc, 0x01, 0x71, 0x75, 0x71, 0x52, 0x74, 0x91, 0x12, 0xef, 0x4b, 0x0a, 0x2d, 0x62, 0x7c, 0x70, 0x79, 0x1f, 0xe7, 0xbd, 0x73, 0xb8, 0xef, 0x3e, 0x40, 0x68, 0x54, 0x98, 0x6a, 0x06, 0xa2, 0x80, 0xaa, 0x59, 0x46, 0x2a, 0x1e, 0x13, 0xb3, 0xb9, 0x55, 0xb1, 0xfb, 0x15, 0x7d, 0x18, 0x46, 0x80, 0x6a, 0x50, 0x62, 0xa6, 0x9e, 0x48, 0x2f, 0x66, 0xe0, 0xb9, 0xbe, 0xee, 0xe1, 0xe3, 0xfb, 0x5d, 0x84, 0x67, 0x79, 0xdf, 0xfb, 0x73, 0xf5, 0x2b, 0x79, 0x93, 0x01, 0x3e, 0x91, 0x38, 0xca, 0x74, 0xc3, 0x22, 0xde, 0x20, 0x9e, 0xdd, 0xb4, 0x74, 0xce, 0xfb, 0xc4, 0x21, 0x56, 0x92, 0x14, 0xe2, 0x73, 0xe2, 0x49, 0x83, 0x1a, 0x24, 0x7e, 0xe4, 0xba, 0xec, 0xf2, 0x1b, 0xe7, 0xa2, 0xc3, 0x02, 0xcf, 0x0c, 0x19, 0x99, 0xd4, 0x3c, 0x71, 0x88, 0x58, 0x2c, 0x76, 0xb0, 0xdc, 0xc1, 0xac, 0x64, 0xa8, 0xc4, 0x33, 0xc4, 0x61, 0x45, 0xd5, 0x28, 0x5f, 0xc8, 0xba, 0xac, 0x70, 0xde, 0xe2, 0xac, 0x56, 0x6a, 0xac, 0xd5, 0x27, 0x7f, 0x61, 0x30, 0xaf, 0xad, 0xa4, 0xb9, 0x4e, 0x35, 0x86, 0x38, 0x96, 0x90, 0x40, 0x12, 0x22, 0x64, 0xd4, 0x50, 0x46, 0x05, 0x16, 0x22, 0xb4, 0x6b, 0xa4, 0x98, 0x48, 0xd1, 0x79, 0xcc, 0xc3, 0x3f, 0xea, 0xf8, 0x93, 0xe4, 0x92, 0xc9, 0x55, 0x06, 0x23, 0xc7, 0x02, 0xaa, 0x50, 0x21, 0x39, 0x7e, 0xf0, 0x3f, 0xf8, 0x3d, 0x5b, 0xb3, 0x30, 0x3d, 0xe5, 0x26, 0x05, 0x63, 0x40, 0xd7, 0x8b, 0x6d, 0x7f, 0x8c, 0x03, 0xdd, 0xbb, 0x40, 0xb3, 0x6e, 0xdb, 0xdf, 0xc7, 0xb6, 0xdd, 0x3c, 0x01, 0xfc, 0xcf, 0xc0, 0x95, 0xd6, 0xf6, 0x57, 0x1b, 0xc0, 0xdc, 0x27, 0xe9, 0xf5, 0xb6, 0x16, 0x3e, 0x02, 0x06, 0xb6, 0x81, 0x8b, 0xeb, 0xb6, 0x26, 0xef, 0x01, 0x97, 0x3b, 0xc0, 0xc8, 0x93, 0x2e, 0x19, 0x92, 0x23, 0xf9, 0xa9, 0x84, 0x42, 0x01, 0x78, 0x3f, 0xa3, 0x6f, 0xca, 0x01, 0x43, 0xb7, 0x40, 0xef, 0x9a, 0x3b, 0xb7, 0xd6, 0x39, 0x4e, 0x1f, 0x80, 0x0c, 0xcd, 0x6a, 0xf9, 0x06, 0x38, 0x38, 0x04, 0x26, 0x8a, 0x94, 0xbd, 0xee, 0xf1, 0xee, 0x9e, 0xce, 0xb9, 0xfd, 0x7b, 0xa7, 0x35, 0xbf, 0x1f, 0x7e, 0x6b, 0x72, 0xab, 0x25, 0xc2, 0xdc, 0xd9, 0x00, 0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x2e, 0x23, 0x00, 0x00, 0x2e, 0x23, 0x01, 0x78, 0xa5, 0x3f, 0x76, 0x00, 0x00, 0x00, 0x07, 0x74, 0x49, 0x4d, 0x45, 0x07, 0xe7, 0x0b, 0x17, 0x04, 0x18, 0x2e, 0x24, 0x4f, 0xe2, 0xe7, 0x00, 0x00, 0x00, 0x19, 0x74, 0x45, 0x58, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x00, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x20, 0x77, 0x69, 0x74, 0x68, 0x20, 0x47, 0x49, 0x4d, 0x50, 0x57, 0x81, 0x0e, 0x17, 0x00, 0x00, 0x00, 0x3e, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x35, 0xcb, 0xb1, 0x0d, 0x00, 0x31, 0x0c, 0xc3, 0x40, 0x66, 0x19, 0xd5, 0x81, 0xa7, 0x35, 0x90, 0xc2, 0xcb, 0x64, 0x17, 0x7b, 0x0b, 0x7d, 0x11, 0x3c, 0xbb, 0x2b, 0xb8, 0x6c, 0x03, 0xc0, 0xcc, 0xdc, 0x7b, 0xb1, 0x6d, 0xbb, 0xbb, 0x33, 0x53, 0x12, 0x0f, 0xe7, 0x1c, 0x49, 0x00, 0x0f, 0x7b, 0xef, 0x77, 0x51, 0x55, 0x11, 0xc1, 0xdf, 0x07, 0xba, 0x61, 0x22, 0x6a, 0xb4, 0xb3, 0x18, 0x82, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}

func testImageUpload(apiHost string, userId1 string, userId2 string) {
	imageUploadJWT := client.GetJWTFromCache(wstestlib.MakeAPIHostWithProtocol(apiHost), test1Username, test1Password)

	scanId := seedDBScanData(scan_Naltsos)
	seedDBUserNotifications(map[string]*protos.UserNotificationSettings{
		userId1: {
			TopicSettings: map[string]protos.NotificationMethod{
				notificationSender.NOTIF_TOPIC_SCAN_NEW:  protos.NotificationMethod_NOTIF_BOTH,
				notificationSender.NOTIF_TOPIC_IMAGE_NEW: protos.NotificationMethod_NOTIF_BOTH,
			},
		},
		userId2: {
			TopicSettings: map[string]protos.NotificationMethod{
				notificationSender.NOTIF_TOPIC_SCAN_NEW:  protos.NotificationMethod_NOTIF_BOTH,
				notificationSender.NOTIF_TOPIC_IMAGE_NEW: protos.NotificationMethod_NOTIF_BOTH,
			},
		},
	})

	// Gotta clear upload area in bucket too so we don't interfere with tests rerunning
	if err := apiStorageFileAccess.EmptyObjects(apiUploadBucket, "MultiPartUploads/"+scanId+"/"); err != nil {
		log.Fatalf("Failed to clear upload file part bucket: %v", err)
	}

	// This got controvertial over time - originally we were uploading image data in a web socket message
	// but that was too small
	// We moved to uploading image in a HTTP request which covered most uses, but then we wanted to import
	// gigabyte+ sized images, so we needed to upload in multiple HTTP requests in chunks with resumability
	// Then we wanted to do this with scans and quants too so made a generic file uploader
	// Now the test is a bit confusing, we are first testing file upload errors THEN testing sending bad
	// import requests (that use the uploaded file)

	// These are attempts to upload just the file (with and without intent set to image, which does more checks)
	// Actual testing of importing happens further down
	checkFileUploadResult(
		"Upload file with missing name",
		apiHost,
		&uploadFileReq{
			data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Invalid filename: \"\"",
	)

	checkFileUploadResult(
		"Upload file with only write pos set",
		apiHost,
		&uploadFileReq{
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			writePos: 1,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Invalid filename: \"\"",
	)

	checkFileUploadResult(
		"Upload file with only total bytes set",
		apiHost,
		&uploadFileReq{
			data:       []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			totalBytes: 13,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Invalid filename: \"\"",
	)

	checkFileUploadResult(
		"Upload with intent=unknown",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.jpg",
			intent:   "unknown",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Unknown intent: \"unknown\"",
	)
	// Checks for the use of intent=image, which should check a few extra things
	checkFileUploadResult(
		"Upload bad format image for scan",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.bmp",
			intent:   "image",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Unexpected format: file_Name.bmp. Must be PNG, JPG or TIF, or the special RGBU 32bit float 4-channel TIF format",
	)

	checkFileUploadResult(
		"Upload image to non-existant scan",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.bmp",
			intent:   "image",
			scan:     "doesnt-exist",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusNotFound,
		"doesnt-exist not found",
	)

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	checkFileUploadResult(
		"Import image with missing name",
		apiHost,
		&uploadFileReq{
			data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Invalid filename: \"\"",
	)

	u1.AddSendReqAction("Import non-existant image",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 1,
			"status": "WS_SERVER_ERROR",
  			"errorText": "Invalid image name",
			"imageImportResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	checkFileUploadResult(
		"Import image missing scan id",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			intent:   "image",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Missing scan id",
	)

	checkFileUploadResult(
		"Upload for non-existant scan",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     "scan123",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusNotFound,
		"scan123 not found",
	)

	// TODO: test uploading corrupt image
	/*
		checkFileUploadResult(
			"Upload corrupt image",
			apiHost,
			&uploadFileReq{
				filename:      "file_Name.png",
				scan: "scan123",
				data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			imageUploadJWT,
			http.StatusNotFound,
			"scan123 not found"
		)*/

	// Delete a non-existant image
	u1.AddSendReqAction("Delete non-existant image",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 2,
			"status": "WS_NOT_FOUND",
			"errorText": "doesnt-exist.png not found",
			"imageDeleteResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	checkFileUploadResult(
		fmt.Sprintf("Upload should fail because no access to %v", scanId),
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusUnauthorized,
		"Edit access denied for: SCAN named \"Naltsos\" (id: 048300551)",
	)

	// Now allow access to originScanId
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}})

	// Upload success
	checkFileUploadResult(
		"Upload OK",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		`{"bytesReceived":"596","completed":true}`,
	)

	// Import the image
	u1.AddSendReqAction("Import newly uploaded image",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "file_Name.png",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 3,
			"status": "WS_OK",
			"imageImportResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{`{"notificationUpd": {
        "notification": {
			"id": "${IGNORE}",
			"notificationType": "NT_USER_MESSAGE",
            "subject": "New image added to scan: 048300551",
            "contents": "A new image named file_Name.png was added to scan: 048300551 (id: 048300551)",
            "from": "Data Importer",
			"timeStampUnixSec": "${SECAGO=10}",
            "actionLink": "analysis?scan_id=048300551&image=048300551/file_Name.png"
        }
    }}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"imageName": "048300551/file_Name.png",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Get the image
	u1.AddSendReqAction("Get uploaded image",
		`{"imageGetReq":{"imageName": "048300551/file_Name.png"}}`,
		fmt.Sprintf(`{"msgId": 4,
		"status": "WS_OK",
		"imageGetResp":{
			"image": {
				"source": "SI_UPLOAD",
				"width": 5,
				"height": 5,
				"fileSize64": "596",
				"purpose": "SIP_VIEWING",
				"associatedScanIds": [
					"%v"
				],
				"originScanId": "%v",
				"imagePath": "%v/file_Name.png"
			}
		}
	}`, scanId, scanId, scanId),
	)

	// Upload another so we can switch default images
	checkFileUploadResult(
		"Upload another OK",
		apiHost,
		&uploadFileReq{
			filename: "another.png",
			scan:     scanId,
			data:     uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		`{"bytesReceived":"596","completed":true}`,
	)

	// Duplicate upload WITHOUT INTENT should work, replacing the file really
	checkFileUploadResult(
		"Duplicate upload WITHOUT INTENT should work",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		`{"bytesReceived":"596","completed":true}`,
	)

	// Duplicate upload should fail as this checks if the image exists already due to intent specified
	checkFileUploadResult(
		"Duplicate upload should fail",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     uploadImgPNGData,
			intent:   "image",
		},
		imageUploadJWT,
		http.StatusBadRequest,
		`Image "048300551/file_Name.png" already exists`,
	)

	// Try import the newly uploaded image
	u1.AddSendReqAction("Import to for non-existant origin id",
		`{"imageImportReq":{
			"name": "file_Name.png",
			"originScanId": "scan123"
		}}`,
		`{
			"msgId": 5,
			"status": "WS_NOT_FOUND",
  			"errorText": "scan123 not found",
			"imageImportResp": {}
		}`,
	)

	// u1.CloseActionGroup([]string{}, 5000)
	// wstestlib.ExecQueuedActions(&u1)

	// Import the image correctly
	u1.AddSendReqAction("Import should be OK",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "another.png",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 6,
			"status": "WS_OK",
			"imageImportResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{`{"notificationUpd": {
        "notification": {
			"id": "${IGNORE}",
			"notificationType": "NT_USER_MESSAGE",
            "subject": "New image added to scan: 048300551",
            "contents": "A new image named another.png was added to scan: 048300551 (id: 048300551)",
            "from": "Data Importer",
			"timeStampUnixSec": "${SECAGO=10}",
            "actionLink": "analysis?scan_id=048300551&image=048300551/another.png"
        }
    }}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"imageName": "048300551/another.png",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Download image data
	status, body, err := doHTTPRequest("GET", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "", nil, imageGetJWT)
	failIf(err != nil, err)
	img, format, err := image.Decode(bytes.NewReader(body))
	var imgW, imgH int
	if img != nil {
		imgW = img.Bounds().Max.X
		imgH = img.Bounds().Max.Y
	}
	failIf(err != nil || format != "png" || status != 200 || imgW != 5 || imgH != 5,
		fmt.Errorf("Failed to download uploaded image! Status %v, format %v image: %vx%v. Error: %v", status, format, imgW, imgH, err),
	)

	// Download a scaled version (forcing generation of cached scaled image)
	status, body, err = doHTTPRequest("GET", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "minwidth=2", nil, imageGetJWT)
	failIf(err != nil, err)
	img, format, err = image.Decode(bytes.NewReader(body))
	if img != nil {
		imgW = img.Bounds().Max.X
		imgH = img.Bounds().Max.Y
	}
	failIf(err != nil || format != "png" || status != 200 || imgW != 5 || imgH != 5,
		fmt.Errorf("Failed to download scaled uploaded image! Status %v, format %v image: %vx%v. Error: %v", status, format, imgW, imgH, err),
	)

	// Set it as default image
	u1.AddSendReqAction("set default image",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/file_Name.png"}}`,
		`{"msgId":7,
			"status": "WS_OK",
			"imageSetDefaultResp": {}
		}`,
	)

	u1.AddSendReqAction("set non-existant default image (should fail)",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/doesnt-exist.png"}}`,
		`{"msgId":8,
			"status": "WS_NOT_FOUND",
			"errorText": "048300551/doesnt-exist.png not found",
			"imageSetDefaultResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Upload beam locations for this image
	u1.AddSendReqAction("Upload image beam locations with missing fields",
		`{"imageBeamLocationUploadReq":{"imageName": "048300551/file_Name.png"}}`,
		`{"msgId": 9,
			"status": "WS_SERVER_ERROR",
			"errorText": "Request missing location",
			"imageBeamLocationUploadResp":{}}`,
	)

	u1.AddSendReqAction("Upload image beam locations with version 1",
		`{"imageBeamLocationUploadReq":{"imageName": "048300551/file_Name.png", "location": {
			"scanId": "123",
			"beamVersion": 1,
			"instrument": 2,
			"locations": [{"i": 3.2, "j": 8.3}, {"i": 7.2, "j": 28.3}]
		}}}`,
		`{"msgId": 10,
			"status": "WS_SERVER_ERROR",
			"errorText": "Currently only version 0 supported for image beam location upload",
			"imageBeamLocationUploadResp":{}}`,
	)

	u1.AddSendReqAction("Upload image beam locations non-existant scan",
		`{"imageBeamLocationUploadReq":{"imageName": "048300551/file_Name.png", "location": {
			"scanId": "non-existant-scan123",
			"beamVersion": 0,
			"instrument": 2,
			"locations": [{"i": 3.2, "j": 8.3}, {"i": 7.2, "j": 28.3}]
		}}}`,
		`{"msgId": 11,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan123 not found",
			"imageBeamLocationUploadResp":{}}`,
	)

	u1.AddSendReqAction("Upload image beam locations should succeed",
		fmt.Sprintf(`{"imageBeamLocationUploadReq":{"imageName": "048300551/file_Name.png", "location": {
			"scanId": "%v",
			"beamVersion": 0,
			"instrument": 2,
			"locations": [{"i": 3.2, "j": 8.3}, {"i": 7.2, "j": 28.3}]
		}}}`, scanId),
		`{"msgId": 12,
			"status": "WS_OK",
			"imageBeamLocationUploadResp":{}}`,
	)

	// Try to delete
	u1.AddSendReqAction("Delete uploaded image, should fail because it's default",
		`{"imageDeleteReq":{
		"name": "048300551/file_Name.png"
	}}`,
		`{
		"msgId": 13,
		"status": "WS_BAD_REQUEST",
		"errorText": "Cannot delete image: \"048300551/file_Name.png\" because it is the default image for scans: [048300551]",
		"imageDeleteResp": {}
	}`,
	)

	// Unset default image
	u1.AddSendReqAction("set default image",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/another.png"}}`,
		`{"msgId": 14,
		"status": "WS_OK",
		"imageSetDefaultResp": {}
	}`,
	)

	// Delete uploaded image
	u1.AddSendReqAction("Delete uploaded image",
		`{"imageDeleteReq":{
			"name": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 15,
			"status": "WS_OK",
			"imageDeleteResp": {}
		}`,
	)
	u1.CloseActionGroup([]string{
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"imageName": "048300551/file_Name.png",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func testImageMultipartUpload(apiHost string) {
	imageUploadJWT := client.GetJWTFromCache(wstestlib.MakeAPIHostWithProtocol(apiHost), test1Username, test1Password)

	// TODO: test uploading corrupt image
	/*
		checkFileUploadResult(
			"Upload corrupt image",
			apiHost,
			&uploadFileReq{
				filename:      "file_Name.png",
				scan: "scan123",
				data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			imageUploadJWT,
			http.StatusNotFound,
			"scan123 not found"
		)*/

	scanId := scan_Naltsos.Id

	// Send bad resume message with missing query params
	respBody := getFileUploadResult(
		"Checking resume status",
		apiHost,
		&uploadFileReq{
			filename: "toResume.png",
			scan:     scanId,
			intent:   "image",
		},
		imageUploadJWT,
		http.StatusBadRequest,
	)

	exp := "Uploading without [totalBytes, writePos] set can't contain empty file data (body)\n"
	if string(respBody) != exp {
		log.Fatalf("Expected \"%v\", got \"%v\"", exp, string(respBody))
	}

	respBody = getFileUploadResult(
		"Checking resume status",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			intent:     "image",
			totalBytes: uint64(len(uploadImgPNGData)),
		},
		imageUploadJWT,
		http.StatusOK,
	)

	resp := &protos.UploadHttpResponseBody{}
	err := protojson.Unmarshal(respBody, resp)

	if err != nil {
		log.Fatalln(err)
	}

	if resp.BytesReceived != 0 {
		log.Fatalf("Expected resp body BytesReceived to be 0, got: %v", resp.BytesReceived)
	}

	checkFileUploadResult(
		"Send 3rd chunk of image, should fail, not the first one expected",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[20:60],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   30,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"UploadPart for 048300551/toResume.png (30 bytes so far) error: expected total bytes so far of 0, got 30\n"
	)

	checkFileUploadResult(
		"Send first chunk of image",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[0:20],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   0,
		},
		imageUploadJWT,
		http.StatusOK,
		"{\"bytesReceived\":\"20\"}",
	)

	checkFileUploadResult(
		"Send second chunk of image",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[20:50],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   20,
		},
		imageUploadJWT,
		http.StatusOK,
		"{\"bytesReceived\":\"50\"}",
	)

	respBody = getFileUploadResult(
		"Checking resume status after 2 chunks uploaded",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			totalBytes: uint64(len(uploadImgPNGData)),
		},
		imageUploadJWT,
		http.StatusOK,
	)

	err = protojson.Unmarshal(respBody, resp)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.BytesReceived != 50 {
		log.Fatalf("Expected resp body BytesReceived to be 50, got: %v", resp.BytesReceived)
	}

	respBody = getFileUploadResult(
		"Send duplicate 2nd chunk of image, should error",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[20:50],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   20,
		},
		imageUploadJWT,
		http.StatusBadRequest,
	)

	exp = "UploadPart for 048300551/toResume.png (20 bytes so far) error: expected total bytes so far of 50, got 20\n"
	if string(respBody) != exp {
		log.Fatalf("Expected \"%v\", got \"%v\"", exp, string(respBody))
	}

	respBody = getFileUploadResult(
		"Send 4th chunk (skipping 3rd), should error",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[180:],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   60,
		},
		imageUploadJWT,
		http.StatusBadRequest,
	)

	exp = "UploadPart for 048300551/toResume.png (60 bytes so far) error: expected total bytes so far of 50, got 60\n"
	if string(respBody) != exp {
		log.Fatalf("Expected \"%v\", got \"%v\"", exp, string(respBody))
	}

	respBody = getFileUploadResult(
		"Checking resume status after 2 chunks and some errors uploaded",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			totalBytes: uint64(len(uploadImgPNGData)),
		},
		imageUploadJWT,
		http.StatusOK,
	)

	err = protojson.Unmarshal(respBody, resp)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.BytesReceived != 50 {
		log.Fatalf("Expected resp body BytesReceived to be 50, got: %v", resp.BytesReceived)
	}

	// Check that we can't request partially uploaded image yet
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})
	u1.AddSendReqAction("Get image - should fail, not imported yet",
		`{"imageGetReq":{"imageName": "048300551/toResume.png"}}`,
		`{
			"msgId": 1,
  			"status": "WS_SERVER_ERROR",
			"errorText": "Failed to find image 048300551/toResume.png, version count was 0",
			"imageGetResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	checkFileUploadResult(
		"Send third chunk of image",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[50:200],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   50,
		},
		imageUploadJWT,
		http.StatusOK,
		"{\"bytesReceived\":\"200\"}",
	)

	checkFileUploadResult(
		"Send last chunk of image, should be finished",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[200:],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   200,
		},
		imageUploadJWT,
		http.StatusOK,
		"{\"bytesReceived\":\"596\",\"completed\":true}",
	)

	u1.AddSendReqAction("Import multi-part multi-resumed image",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "toResume.png",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 2,
			"status": "WS_OK",
			"imageImportResp": {}
		}`,
	)

	// Check that we CAN request fully uploaded image
	u1.AddSendReqAction("Get image",
		`{"imageGetReq":{"imageName": "048300551/toResume.png"}}`,
		fmt.Sprintf(`{"msgId": 3,
			"status": "WS_OK",
			"imageGetResp":{
				"image": {
					"source": "SI_UPLOAD",
					"width": 5,
					"height": 5,
					"fileSize64": "596",
					"purpose": "SIP_VIEWING",
					"associatedScanIds": [
						"%v"
					],
					"originScanId": "%v",
					"imagePath": "%v/toResume.png"
				}
			}
		}`, scanId, scanId, scanId),
	)

	u1.CloseActionGroup([]string{
		`{"notificationUpd": {
			"notification": {
				"id": "${IGNORE}",
				"notificationType": "NT_USER_MESSAGE",
				"subject": "New image added to scan: 048300551",
				"contents": "A new image named toResume.png was added to scan: 048300551 (id: 048300551)",
				"from": "Data Importer",
				"timeStampUnixSec": "${SECAGO=10}",
				"actionLink": "analysis?scan_id=048300551&image=048300551/toResume.png"
			}
		}}`,
		`{"notificationUpd": {
				"notification": {
					"notificationType": "NT_SYS_DATA_CHANGED",
					"imageName": "048300551/toResume.png",
					"scanIds": [
						"048300551"
					]
				}
			}
		}`,
	}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	checkFileUploadResult(
		"Checking resume status after image upload completed, should signal it's completed uploading",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			totalBytes: uint64(len(uploadImgPNGData)),
		},
		imageUploadJWT,
		http.StatusOK,
		"{\"bytesReceived\":\"596\",\"completed\":true}"
	)

	checkFileUploadResult(
		"Send extra 5th chunk of image, should fail",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[200:230],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   250,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"UploadPart for 048300551/toResume.png (250 bytes so far) error: unexpected status already complete\n"
	)

	checkFileUploadResult(
		"Send last chunk of image again, should fail",
		apiHost,
		&uploadFileReq{
			filename:   "toResume.png",
			scan:       scanId,
			data:       uploadImgPNGData[200:],
			totalBytes: uint64(len(uploadImgPNGData)),
			writePos:   200,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"UploadPart for 048300551/toResume.png (200 bytes so far) error: unexpected status already complete\n"
	)
}

type uploadFileReq struct {
	intent     string
	scan       string
	filename   string
	totalBytes uint64
	writePos   uint64
	data       []byte
}

func getFileUploadResult(action string, apiHost string,
	req *uploadFileReq,
	imageUploadJWT string,
	expStatus int) []byte {
	fmt.Printf("%v - sending PUT request:\n", action)

	query := fmt.Sprintf("scan=%v&filename=%v", req.scan, req.filename)
	if req.totalBytes > 0 || req.writePos > 0 {
		query += fmt.Sprintf("&totalBytes=%v&writePos=%v", req.totalBytes, req.writePos)
	}
	if len(req.intent) > 0 {
		query += fmt.Sprintf("&intent=%v", req.intent)
	}

	status, respBody, err := doHTTPRequest("PUT", apiHost, "file", query, bytes.NewBuffer(req.data), imageUploadJWT)

	if err != nil {
		log.Fatalln(err)
	}

	if status != expStatus {
		log.Fatalf("[%v] Expected status=%v, got status=%v\nBody: %v", action, expStatus, status, string(respBody))
	}

	return respBody
}

func checkFileUploadResult(action string, apiHost string,
	uploadReq *uploadFileReq,
	JWT string, expStatus int, expBody string) {
	respBody := getFileUploadResult(action, apiHost, uploadReq, JWT, expStatus)

	// JSON processing in Go sometimes puts in spaces, sometimes not, both of these are valid:
	// {"bytesReceived":"596", "completed":true}
	// {"bytesReceived":"596","completed":true}
	// So here we remove the space
	respStr := strings.ReplaceAll(string(respBody), "\", \"", "\",\"")

	expBodyCompare := expBody
	if len(expBody) > 0 && strings.HasSuffix(respStr, "\n") {
		expBodyCompare += "\n"
	}

	if respStr != expBodyCompare {
		log.Fatalf("[%v]\nExpected body=%v\nReceived body=%v", action, expBody, string(respBody))
	}
}

func testImageMatchTransform(apiHost string) {
	imageUploadJWT := client.GetJWTFromCache(wstestlib.MakeAPIHostWithProtocol(apiHost), test1Username, test1Password)

	scanId := seedDBScanData(scan_Naltsos)
	//scanImage := "PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"
	seedImages()
	seedImageLocations()

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("SetImageMatchTransform - should fail, missing transform",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 1,
			"status": "WS_BAD_REQUEST",
			"errorText": "Transform must be set",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.AddSendReqAction("SetImageMatchTransform - should fail, bad transform",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png",
			"transform": {
				"beamImageFileName": "beamImage.png",
				"xOffset": -1,
				"yOffset": -1,
				"xScale": 0,
				"yScale": 1
			}
		}}`,
		`{
			"msgId": 2,
			"status": "WS_BAD_REQUEST",
			"errorText": "Transform must have positive scale values",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.AddSendReqAction("SetImageMatchTransform - should fail, bad image",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png",
			"transform": {
				"beamImageFileName": "beamImage.png",
				"xOffset": -1,
				"xScale": 10,
				"yScale": 1
			}
		}}`,
		`{
			"msgId": 3,
			"status": "WS_NOT_FOUND",
			"errorText": "048300551/file_Name.png not found",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}})

	checkFileUploadResult(
		"Upload matched OK",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     uploadImgPNGData,
			intent:   "image",
		},
		imageUploadJWT,
		http.StatusOK,
		`{"bytesReceived":"596","completed":true}`,
	)

	u1.AddSendReqAction("Import matched image",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "file_Name.png",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 4,
			"status": "WS_OK",
			"imageImportResp": {}
		}`,
	)

	u1.AddSendReqAction("Delete non-existant image (just really here to allow capturing notifications from the above)",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 5,
			"status": "WS_NOT_FOUND",
			"errorText": "doesnt-exist.png not found",
			"imageDeleteResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{
		`{
			"notificationUpd": {
				"notification": {
					"id": "${IGNORE}",
					"notificationType": "NT_USER_MESSAGE",
					"subject": "New image added to scan: 048300551",
					"contents": "A new image named file_Name.png was added to scan: 048300551 (id: 048300551)",
					"from": "Data Importer",
					"timeStampUnixSec": "${SECAGO=10}",
					"actionLink": "analysis?scan_id=048300551&image=048300551/file_Name.png"
				}
			}
		}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				],
				"imageName": "048300551/file_Name.png"
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("SetImageMatchTransform - should succeed",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png",
			"transform": {
				"beamImageFileName": "beamImage.png",
				"xOffset": -1,
				"xScale": 10,
				"yScale": 1
			}
		}}`,
		`{
			"msgId": 6,
			"status": "WS_SERVER_ERROR",
			"errorText": "Failed edit transform for image 048300551/file_Name.png - it is not a matched image",
			"imageSetMatchTransformResp": {}
		}`,
	)

	// Delete image
	u1.AddSendReqAction("Delete uploaded image 2",
		`{"imageDeleteReq":{
			"name": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 7,
			"status": "WS_OK",
			"imageDeleteResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"imageName": "048300551/file_Name.png",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Delete non-existant image (just really here to allow capturing notifications from the above) 2",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 8,
			"status": "WS_NOT_FOUND",
			"errorText": "doesnt-exist.png not found",
			"imageDeleteResp": {}
		}`,
	)

	// Re-upload as matched image
	checkFileUploadResult(
		"Upload Matched OK2",
		apiHost,
		&uploadFileReq{
			filename: "file_Name.png",
			scan:     scanId,
			data:     uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		`{"bytesReceived":"596","completed":true}`,
	)

	u1.AddSendReqAction("Import as matched image",
		fmt.Sprintf(`{"imageImportReq":{
			"name": "file_Name.png",
			"originScanId": "%v",
			"beamImageRef": {
				"beamImageFileName": "match_image.png",
				"xOffset":           0,
				"yOffset":           0,
				"xScale":            1,
				"yScale":            1
			}
		}}`, scanId),
		`{
			"msgId": 9,
			"status": "WS_OK",
			"imageImportResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{
		`{
			"notificationUpd": {
				"notification": {
					"id": "${IGNORE}",
					"notificationType": "NT_USER_MESSAGE",
					"subject": "New image added to scan: 048300551",
					"contents": "A new image named file_Name.png was added to scan: 048300551 (id: 048300551)",
					"from": "Data Importer",
					"timeStampUnixSec": "${SECAGO=10}",
					"actionLink": "analysis?scan_id=048300551&image=048300551/file_Name.png"
				}
			}
		}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"imageName": "048300551/file_Name.png",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Set transform again
	u1.AddSendReqAction("SetImageMatchTransform - should succeed 2",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png",
			"transform": {
				"beamImageFileName": "beamImage.png",
				"xOffset": -1,
				"xScale": 10,
				"yScale": 1
			}
		}}`,
		`{
			"msgId": 10,
			"status": "WS_OK",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.AddSendReqAction("SetImageMatchTransform - overwrite should succeed",
		`{"imageSetMatchTransformReq":{
			"imageName": "048300551/file_Name.png",
			"transform": {
				"beamImageFileName": "beamImage.png",
				"xOffset": -1,
				"yOffset": 20,
				"xScale": 10,
				"yScale": 4
			}
		}}`,
		`{
			"msgId": 11,
			"status": "WS_OK",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.AddSendReqAction("ImageGetReq - should succeed",
		`{"imageGetReq":{
			"imageName": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 12,
			"status": "WS_OK",
			"imageGetResp": {
				"image": {
					"imagePath": "048300551/file_Name.png",
					"source": "SI_UPLOAD",
					"width": 5,
					"height": 5,
					"fileSize64": "596",
					"purpose": "SIP_VIEWING",
					"associatedScanIds": [
						"048300551"
					],
					"originScanId": "048300551",
					"matchInfo": {
						"beamImageFileName": "match_image.png",
						"xOffset": -1,
						"yOffset": 20,
						"xScale": 10,
						"yScale": 4
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)
}
