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

	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testImageUpload(apiHost string) {
	scanId := seedDBScanData()

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Upload bad format image for scan",
		`{"imageUploadReq":{
			"name": "file_Name.bmp",
			"imageData": "aW1hZ2Ugb2YgYSBjYXQ="
		}}`,
		`{
			"msgId": 1,
			"status": "WS_SERVER_ERROR",
			"errorText": "Unexpected format: file_Name.bmp. Must be either PNG, JPG or 32bit float 4-channel TIF file",
			"imageUploadResp": {}
		}`,
	)

	u1.AddSendReqAction("Upload with missing name",
		`{"imageUploadReq":{
			"imageData": "aW1hZ2Ugb2YgYSBjYXQ="
		}}`,
		`{
			"msgId": 2,
			"status": "WS_BAD_REQUEST",
			"errorText": "Name is too short",
			"imageUploadResp": {}
		}`,
	)

	u1.AddSendReqAction("Upload missing origin id for scan",
		`{"imageUploadReq":{
			"name": "file_Name.png",
			"imageData": "aW1hZ2Ugb2YgYSBjYXQ="
		}}`,
		`{
			"msgId": 3,
			"status": "WS_BAD_REQUEST",
			"errorText": "OriginScanId is too short",
			"imageUploadResp": {}
		}`,
	)

	u1.AddSendReqAction("Upload corrupt image",
		`{"imageUploadReq":{
			"name": "file_Name.png",
			"imageData": "aW1hZ2Ugb2YgYSBjYXQ=",
			"originScanId": "scan123"
		}}`,
		`{
			"msgId": 4,
			"status": "WS_SERVER_ERROR",
			"errorText": "image: unknown format",
			"imageUploadResp": {}
		}`,
	)
	// Delete a non-existant image
	u1.AddSendReqAction("Delete non-existant imag",
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

	// Upload success
	u1.AddSendReqAction("Upload OK",
		fmt.Sprintf(`{"imageUploadReq":{
			"name": "file_Name.png",
			"imageData": "iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAIAAAACDbGyAAABhGlDQ1BJQ0MgcHJvZmlsZQAAKJF9kT1Iw1AUhU/TiqIVEQuKOGSoTnZREcdSxSJYKG2FVh1MXvoHTRqSFBdHwbXg4M9i1cHFWVcHV0EQ/AFxdXFSdJES70sKLWJ8cHkf571zuO8+QGhUmGoGooCqWUYqHhOzuVWx+xV9GEaAalBipp5IL2bgub7u4eP7XYRned/7c/UreZMBPpE4ynTDIt4gnt20dM77xCFWkhTic+JJgxokfuS67PIb56LDAs8MGZnUPHGIWCx2sNzBrGSoxDPEYUXVKF/Iuqxw3uKsVmqs1Sd/YTCvraS5TjWGOJaQQBIiZNRQRgUWIrRrpJhI0XnMwz/q+JPkkslVBiPHAqpQITl+8D/4PVuzMD3lJgVjQNeLbX+MA927QLNu29/Htt08AfzPwJXW9lcbwNwn6fW2Fj4CBraBi+u2Ju8BlzvAyJMuGZIj+amEQgF4P6NvygFDt0Dvmju31jlOH4AMzWr5Bjg4BCaKlL3u8e6ezrn9e6c1vx9+a3KrJcLc2QAAAAlwSFlzAAAuIwAALiMBeKU/dgAAAAd0SU1FB+cLFwQYLiRP4ucAAAAZdEVYdENvbW1lbnQAQ3JlYXRlZCB3aXRoIEdJTVBXgQ4XAAAAPklEQVQI1zXLsQ0AMQzDQGYZ1YGnNZDCy2QXewt9ETy7K7hsA8DM3Huxbbu7M1MSD+ccSQAPe+93UVURwd8HumEiarSzGIIAAAAASUVORK5CYII=",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 6,
			"status": "WS_OK",
			"imageUploadResp": {}
		}`,
	)

	// Duplicate upload should fail
	u1.AddSendReqAction("Duplicate upload should fail",
		fmt.Sprintf(`{"imageUploadReq":{
			"name": "file_Name.png",
			"imageData": "iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAIAAAACDbGyAAABhGlDQ1BJQ0MgcHJvZmlsZQAAKJF9kT1Iw1AUhU/TiqIVEQuKOGSoTnZREcdSxSJYKG2FVh1MXvoHTRqSFBdHwbXg4M9i1cHFWVcHV0EQ/AFxdXFSdJES70sKLWJ8cHkf571zuO8+QGhUmGoGooCqWUYqHhOzuVWx+xV9GEaAalBipp5IL2bgub7u4eP7XYRned/7c/UreZMBPpE4ynTDIt4gnt20dM77xCFWkhTic+JJgxokfuS67PIb56LDAs8MGZnUPHGIWCx2sNzBrGSoxDPEYUXVKF/Iuqxw3uKsVmqs1Sd/YTCvraS5TjWGOJaQQBIiZNRQRgUWIrRrpJhI0XnMwz/q+JPkkslVBiPHAqpQITl+8D/4PVuzMD3lJgVjQNeLbX+MA927QLNu29/Htt08AfzPwJXW9lcbwNwn6fW2Fj4CBraBi+u2Ju8BlzvAyJMuGZIj+amEQgF4P6NvygFDt0Dvmju31jlOH4AMzWr5Bjg4BCaKlL3u8e6ezrn9e6c1vx9+a3KrJcLc2QAAAAlwSFlzAAAuIwAALiMBeKU/dgAAAAd0SU1FB+cLFwQYLiRP4ucAAAAZdEVYdENvbW1lbnQAQ3JlYXRlZCB3aXRoIEdJTVBXgQ4XAAAAPklEQVQI1zXLsQ0AMQzDQGYZ1YGnNZDCy2QXewt9ETy7K7hsA8DM3Huxbbu7M1MSD+ccSQAPe+93UVURwd8HumEiarSzGIIAAAAASUVORK5CYII=",
			"originScanId": "%v"
		}}`, scanId),
		`{
			"msgId": 7,
			"status": "WS_BAD_REQUEST",
			"errorText": "file_Name.png already exists",
			"imageUploadResp": {}
		}`,
	)
	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, &protos.UserGroupList{UserIds: []string{u1.GetUserId()}}, nil)

	// Get the image
	u1.AddSendReqAction("Get uploaded image",
		`{"imageGetReq":{"imageName": "file_Name.png"}}`,
		fmt.Sprintf(`{"msgId": 8,
		"status": "WS_OK",
		"imageGetResp":{
			"image": {
				"name": "file_Name.png",
				"source": "SI_UPLOAD",
				"width": 5,
				"height": 5,
				"fileSize": 596,
				"purpose": "SIP_VIEWING",
				"associatedScanIds": [
					"%v"
				],
				"originScanId": "%v",
				"path": "%v/file_Name.png"
			}
		}
	}`, scanId, scanId, scanId),
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	// Download image data
	status, body, err := doGet("http", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "", imageGetJWT)
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
	status, body, err = doGet("http", apiHost, fmt.Sprintf("images/download/%v/file_Name.png", scanId), "minwidth=2", imageGetJWT)
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
		`{"msgId":9,
			"status": "WS_OK",
			"imageSetDefaultResp": {}
		}`,
	)

	// Try to delete
	u1.AddSendReqAction("Delete uploaded image, should fail because it's default",
		`{"imageDeleteReq":{
		"name": "file_Name.png"
	}}`,
		`{
		"msgId": 10,
		"status": "WS_BAD_REQUEST",
		"errorText": "Cannot delete image: \"file_Name.png\" because it is the default image for scans: [048300551]",
		"imageDeleteResp": {}
	}`,
	)

	// Unset default image
	u1.AddSendReqAction("set default image",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/another.png"}}`,
		`{"msgId":11,
		"status": "WS_OK",
		"imageSetDefaultResp": {}
	}`,
	)

	// Delete uploaded image
	u1.AddSendReqAction("Delete uploaded image",
		`{"imageDeleteReq":{
			"name": "file_Name.png"
		}}`,
		`{
			"msgId": 12,
			"status": "WS_OK",
			"imageDeleteResp": {}
		}`,
	)
	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}