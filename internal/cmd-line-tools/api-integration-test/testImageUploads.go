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

	"github.com/pixlise/core/v4/api/notificationSender"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
)

var uploadImgPNGData = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x05, 0x08, 0x02, 0x00, 0x00, 0x00, 0x02, 0x0d, 0xb1, 0xb2, 0x00, 0x00, 0x01, 0x84, 0x69, 0x43, 0x43, 0x50, 0x49, 0x43, 0x43, 0x20, 0x70, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65, 0x00, 0x00, 0x28, 0x91, 0x7d, 0x91, 0x3d, 0x48, 0xc3, 0x50, 0x14, 0x85, 0x4f, 0xd3, 0x8a, 0xa2, 0x15, 0x11, 0x0b, 0x8a, 0x38, 0x64, 0xa8, 0x4e, 0x76, 0x51, 0x11, 0xc7, 0x52, 0xc5, 0x22, 0x58, 0x28, 0x6d, 0x85, 0x56, 0x1d, 0x4c, 0x5e, 0xfa, 0x07, 0x4d, 0x1a, 0x92, 0x14, 0x17, 0x47, 0xc1, 0xb5, 0xe0, 0xe0, 0xcf, 0x62, 0xd5, 0xc1, 0xc5, 0x59, 0x57, 0x07, 0x57, 0x41, 0x10, 0xfc, 0x01, 0x71, 0x75, 0x71, 0x52, 0x74, 0x91, 0x12, 0xef, 0x4b, 0x0a, 0x2d, 0x62, 0x7c, 0x70, 0x79, 0x1f, 0xe7, 0xbd, 0x73, 0xb8, 0xef, 0x3e, 0x40, 0x68, 0x54, 0x98, 0x6a, 0x06, 0xa2, 0x80, 0xaa, 0x59, 0x46, 0x2a, 0x1e, 0x13, 0xb3, 0xb9, 0x55, 0xb1, 0xfb, 0x15, 0x7d, 0x18, 0x46, 0x80, 0x6a, 0x50, 0x62, 0xa6, 0x9e, 0x48, 0x2f, 0x66, 0xe0, 0xb9, 0xbe, 0xee, 0xe1, 0xe3, 0xfb, 0x5d, 0x84, 0x67, 0x79, 0xdf, 0xfb, 0x73, 0xf5, 0x2b, 0x79, 0x93, 0x01, 0x3e, 0x91, 0x38, 0xca, 0x74, 0xc3, 0x22, 0xde, 0x20, 0x9e, 0xdd, 0xb4, 0x74, 0xce, 0xfb, 0xc4, 0x21, 0x56, 0x92, 0x14, 0xe2, 0x73, 0xe2, 0x49, 0x83, 0x1a, 0x24, 0x7e, 0xe4, 0xba, 0xec, 0xf2, 0x1b, 0xe7, 0xa2, 0xc3, 0x02, 0xcf, 0x0c, 0x19, 0x99, 0xd4, 0x3c, 0x71, 0x88, 0x58, 0x2c, 0x76, 0xb0, 0xdc, 0xc1, 0xac, 0x64, 0xa8, 0xc4, 0x33, 0xc4, 0x61, 0x45, 0xd5, 0x28, 0x5f, 0xc8, 0xba, 0xac, 0x70, 0xde, 0xe2, 0xac, 0x56, 0x6a, 0xac, 0xd5, 0x27, 0x7f, 0x61, 0x30, 0xaf, 0xad, 0xa4, 0xb9, 0x4e, 0x35, 0x86, 0x38, 0x96, 0x90, 0x40, 0x12, 0x22, 0x64, 0xd4, 0x50, 0x46, 0x05, 0x16, 0x22, 0xb4, 0x6b, 0xa4, 0x98, 0x48, 0xd1, 0x79, 0xcc, 0xc3, 0x3f, 0xea, 0xf8, 0x93, 0xe4, 0x92, 0xc9, 0x55, 0x06, 0x23, 0xc7, 0x02, 0xaa, 0x50, 0x21, 0x39, 0x7e, 0xf0, 0x3f, 0xf8, 0x3d, 0x5b, 0xb3, 0x30, 0x3d, 0xe5, 0x26, 0x05, 0x63, 0x40, 0xd7, 0x8b, 0x6d, 0x7f, 0x8c, 0x03, 0xdd, 0xbb, 0x40, 0xb3, 0x6e, 0xdb, 0xdf, 0xc7, 0xb6, 0xdd, 0x3c, 0x01, 0xfc, 0xcf, 0xc0, 0x95, 0xd6, 0xf6, 0x57, 0x1b, 0xc0, 0xdc, 0x27, 0xe9, 0xf5, 0xb6, 0x16, 0x3e, 0x02, 0x06, 0xb6, 0x81, 0x8b, 0xeb, 0xb6, 0x26, 0xef, 0x01, 0x97, 0x3b, 0xc0, 0xc8, 0x93, 0x2e, 0x19, 0x92, 0x23, 0xf9, 0xa9, 0x84, 0x42, 0x01, 0x78, 0x3f, 0xa3, 0x6f, 0xca, 0x01, 0x43, 0xb7, 0x40, 0xef, 0x9a, 0x3b, 0xb7, 0xd6, 0x39, 0x4e, 0x1f, 0x80, 0x0c, 0xcd, 0x6a, 0xf9, 0x06, 0x38, 0x38, 0x04, 0x26, 0x8a, 0x94, 0xbd, 0xee, 0xf1, 0xee, 0x9e, 0xce, 0xb9, 0xfd, 0x7b, 0xa7, 0x35, 0xbf, 0x1f, 0x7e, 0x6b, 0x72, 0xab, 0x25, 0xc2, 0xdc, 0xd9, 0x00, 0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x2e, 0x23, 0x00, 0x00, 0x2e, 0x23, 0x01, 0x78, 0xa5, 0x3f, 0x76, 0x00, 0x00, 0x00, 0x07, 0x74, 0x49, 0x4d, 0x45, 0x07, 0xe7, 0x0b, 0x17, 0x04, 0x18, 0x2e, 0x24, 0x4f, 0xe2, 0xe7, 0x00, 0x00, 0x00, 0x19, 0x74, 0x45, 0x58, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x00, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x20, 0x77, 0x69, 0x74, 0x68, 0x20, 0x47, 0x49, 0x4d, 0x50, 0x57, 0x81, 0x0e, 0x17, 0x00, 0x00, 0x00, 0x3e, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x35, 0xcb, 0xb1, 0x0d, 0x00, 0x31, 0x0c, 0xc3, 0x40, 0x66, 0x19, 0xd5, 0x81, 0xa7, 0x35, 0x90, 0xc2, 0xcb, 0x64, 0x17, 0x7b, 0x0b, 0x7d, 0x11, 0x3c, 0xbb, 0x2b, 0xb8, 0x6c, 0x03, 0xc0, 0xcc, 0xdc, 0x7b, 0xb1, 0x6d, 0xbb, 0xbb, 0x33, 0x53, 0x12, 0x0f, 0xe7, 0x1c, 0x49, 0x00, 0x0f, 0x7b, 0xef, 0x77, 0x51, 0x55, 0x11, 0xc1, 0xdf, 0x07, 0xba, 0x61, 0x22, 0x6a, 0xb4, 0xb3, 0x18, 0x82, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}

func testImageUpload(apiHost string, userId1 string, userId2 string) {
	imageUploadJWT := wstestlib.GetJWTFromCache(apiHost, test1Username, test1Password)

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

	checkImageUploadError(
		"Upload bad format image for scan",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:      "file_Name.bmp",
			ImageData: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Unexpected format: file_Name.bmp. Must be either PNG, JPG or 32bit float 4-channel TIF file",
	)

	checkImageUploadError(
		"Upload with missing name",
		apiHost,
		&protos.ImageUploadHttpRequest{
			ImageData: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"Name is too short",
	)

	checkImageUploadError(
		"Upload missing origin id for scan",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:      "file_Name.png",
			ImageData: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"OriginScanId is too short",
	)

	checkImageUploadError(
		"Upload missing origin id for scan",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: "scan123",
			ImageData:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusNotFound,
		"scan123 not found",
	)

	// TODO: test uploading corrupt image
	/*
		checkImageUploadError(
			"Upload corrupt image",
			apiHost,
			&protos.ImageUploadHttpRequest{
				Name:      "file_Name.png",
				OriginScanId: "scan123",
				ImageData: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			imageUploadJWT,
			http.StatusNotFound,
			"scan123 not found"
		)*/

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Delete a non-existant image
	u1.AddSendReqAction("Delete non-existant image",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 1,
			"status": "WS_NOT_FOUND",
			"errorText": "doesnt-exist.png not found",
			"imageDeleteResp": {}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	checkImageUploadError(
		"Upload should fail because no access to originScanId",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: scanId,
			ImageData:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		imageUploadJWT,
		http.StatusUnauthorized,
		"View access denied for: OT_SCAN (048300551)",
	)

	// Now allow access to originScanId
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}}, nil)

	// Upload success
	checkImageUploadError(
		"Upload OK",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: scanId,
			ImageData:    uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		"",
	)

	// Upload another so we can switch default images
	checkImageUploadError(
		"Upload another OK",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "another.png",
			OriginScanId: scanId,
			ImageData:    uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		"",
	)

	// Duplicate upload should fail
	checkImageUploadError(
		"Duplicate upload should fail",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: scanId,
			ImageData:    uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusBadRequest,
		"048300551/file_Name.png already exists",
	)

	// Get the image
	u1.AddSendReqAction("Get uploaded image",
		`{"imageGetReq":{"imageName": "048300551/file_Name.png"}}`,
		fmt.Sprintf(`{"msgId": 2,
		"status": "WS_OK",
		"imageGetResp":{
			"image": {
				"source": "SI_UPLOAD",
				"width": 5,
				"height": 5,
				"fileSize": 596,
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
				"scanIds": [
					"048300551"
				]
			}
		}
	}`,
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Download image data
	status, body, err := doHTTPRequest("http", "GET", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "", nil, imageGetJWT)
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
	status, body, err = doHTTPRequest("http", "GET", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "minwidth=2", nil, imageGetJWT)
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
		`{"msgId":3,
			"status": "WS_OK",
			"imageSetDefaultResp": {}
		}`,
	)

	u1.AddSendReqAction("set non-existant default image (should fail)",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/doesnt-exist.png"}}`,
		`{"msgId":4,
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

	// Try to delete
	u1.AddSendReqAction("Delete uploaded image, should fail because it's default",
		`{"imageDeleteReq":{
		"name": "048300551/file_Name.png"
	}}`,
		`{
		"msgId": 5,
		"status": "WS_BAD_REQUEST",
		"errorText": "Cannot delete image: \"048300551/file_Name.png\" because it is the default image for scans: [048300551]",
		"imageDeleteResp": {}
	}`,
	)

	// Unset default image
	u1.AddSendReqAction("set default image",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/another.png"}}`,
		`{"msgId":6,
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
			"msgId": 7,
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
				"scanIds": [
					"048300551"
				]
			}
		}
	}`}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func checkImageUploadError(action string, apiHost string, req *protos.ImageUploadHttpRequest, imageUploadJWT string, expStatus int, expBody string) {
	fmt.Printf("%v sending request:", action)

	uploadBody, err := proto.Marshal(req)
	if err != nil {
		log.Fatalln(err)
	}

	status, respBody, err := doHTTPRequest("http", "PUT", apiHost, "images", "", bytes.NewBuffer(uploadBody), imageUploadJWT)

	if err != nil {
		log.Fatalln(err)
	}

	expBodyCompare := expBody
	if len(expBody) > 0 {
		expBodyCompare += "\n"
	}

	if status != expStatus || string(respBody) != expBodyCompare {
		log.Fatalf("[%v] Expected status=%v, body=%v.\nGot status=%v, body=%v", action, expStatus, expBody, status, string(respBody))
	}
}

func testImageMatchTransform(apiHost string) {
	imageUploadJWT := wstestlib.GetJWTFromCache(apiHost, test1Username, test1Password)

	scanId := seedDBScanData(scan_Naltsos)
	//scanImage := "PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"
	seedImages()
	seedImageLocations()

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
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

	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}}, nil)

	checkImageUploadError(
		"Upload matched OK",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: scanId,
			ImageData:    uploadImgPNGData,
		},
		imageUploadJWT,
		http.StatusOK,
		"",
	)

	u1.AddSendReqAction("Delete non-existant image (just really here to allow capturing notifications from the above)",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 4,
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
				]
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
			"msgId": 5,
			"status": "WS_SERVER_ERROR",
			"errorText": "Failed edit transform for image 048300551/file_Name.png - it is not a matched image",
			"imageSetMatchTransformResp": {}
		}`,
	)

	// Delete image
	u1.AddSendReqAction("Delete uploaded image",
		`{"imageDeleteReq":{
			"name": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 6,
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
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Delete non-existant image (just really here to allow capturing notifications from the above)",
		`{"imageDeleteReq":{
			"name": "doesnt-exist.png"
		}}`,
		`{
			"msgId": 7,
			"status": "WS_NOT_FOUND",
			"errorText": "doesnt-exist.png not found",
			"imageDeleteResp": {}
		}`,
	)

	// Re-upload as matched image
	checkImageUploadError(
		"Upload Matched OK",
		apiHost,
		&protos.ImageUploadHttpRequest{
			Name:         "file_Name.png",
			OriginScanId: scanId,
			ImageData:    uploadImgPNGData,
			Assocation: &protos.ImageUploadHttpRequest_BeamImageRef{
				BeamImageRef: &protos.ImageMatchTransform{
					BeamImageFileName: "match_image.png",
					XOffset:           0,
					YOffset:           0,
					XScale:            1,
					YScale:            1,
				},
			},
		},
		imageUploadJWT,
		http.StatusOK,
		"",
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
				]
			}
		}
	}`}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	// Set transform again
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
			"msgId": 8,
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
			"msgId": 9,
			"status": "WS_OK",
			"imageSetMatchTransformResp": {}
		}`,
	)

	u1.AddSendReqAction("ImageGetReq - should succeed",
		`{"imageGetReq":{
			"imageName": "048300551/file_Name.png"
		}}`,
		`{
			"msgId": 10,
			"status": "WS_OK",
			"imageGetResp": {
				"image": {
					"imagePath": "048300551/file_Name.png",
					"source": "SI_UPLOAD",
					"width": 5,
					"height": 5,
					"fileSize": 596,
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
