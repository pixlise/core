package main

import (
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testImage3DPoint(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Get non-existant points",
		`{"image3DModelPointsReq":{"imageName": "non-existant.png"}}`,
		`{
			"msgId":1,
			"status":"WS_BAD_REQUEST",
			"errorText":"3D points not found for image: \"non-existant.png\"",
			"image3DModelPointsResp":{}
			}`,
	)

	u1.AddSendReqAction("Upload empty points (should fail)",
		`{"image3DModelPointUploadReq":{"points": {"imageName": "non-existant.png"}}}`,
		`{
			"msgId":2,
			"status":"WS_BAD_REQUEST",
			"errorText":"Point list is empty",
			"image3DModelPointUploadResp":{}
		}`,
	)

	u1.AddSendReqAction("Upload points for non existant image",
		`{"image3DModelPointUploadReq":{"points": {"imageName": "non-existant.png", "points": [{"x": 1, "y": 2, "z": 3}, {"x": 4, "y": 5, "z": 6}]}}}`,
		`{
			"msgId":3,
			"status":"WS_BAD_REQUEST",
  			"errorText": "Image \"non-existant.png\" not found",
			"image3DModelPointUploadResp":{}
		}`,
	)

	u1.AddSendReqAction("Upload points for real image with RTT missing at start of path",
		`{"image3DModelPointUploadReq":{"points": {"imageName": "PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png", "points": [{"x": 1, "y": 2, "z": 3}, {"x": 4, "y": 5, "z": 6}]}}}`,
		`{
			"msgId":4,
			"status": "WS_OK",
			"image3DModelPointUploadResp":{}
		}`,
	)

	u1.AddSendReqAction("Get points for real image v2",
		`{"image3DModelPointsReq":{"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
		`{
			"msgId":5,
			"status": "WS_OK",
			"image3DModelPointsResp": {
				"points": {
					"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J__.png",
					"points": [
						{
							"x": 1,
							"y": 2,
							"z": 3
						},
						{
							"x": 4,
							"y": 5,
							"z": 6
						}
					]
				}
			}
		}`,
	)

	u1.AddSendReqAction("Get points for real image v3",
		`{"image3DModelPointsReq":{"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J03.png"}}`,
		`{
			"msgId":6,
			"status": "WS_OK",
			"image3DModelPointsResp": {
				"points": {
					"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J__.png",
					"points": [
						{
							"x": 1,
							"y": 2,
							"z": 3
						},
						{
							"x": 4,
							"y": 5,
							"z": 6
						}
					]
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
