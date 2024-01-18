package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/wstestlib"
)

func testDataModules(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List modules",
		`{"dataModuleListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","dataModuleListResp":{}}`,
	)

	u1.AddSendReqAction("Get non-existant module (latest)",
		`{"dataModuleGetReq":{"id": "non-existant-id"}}`,
		`{"msgId":2,"status":"WS_NOT_FOUND","errorText":"non-existant-id not found","dataModuleGetResp":{}}`,
	)

	u1.AddSendReqAction("Get non-existant module (version specified)",
		`{"dataModuleGetReq":{"id": "non-existant-id", "version": {"major": 1, "minor": 2, "patch": 3}}}`,
		`{"msgId":3,"status":"WS_NOT_FOUND","errorText":"non-existant-id not found","dataModuleGetResp":{}}`,
	)

	u1.AddSendReqAction("Add version to non-existant module (should fail)",
		`{"dataModuleAddVersionReq":{"moduleId": "non-existant-id", "versionUpdate": "MV_PATCH", "sourceCode": "source 1 2 3", "comments": "Nothing exciting"}}`,
		`{"msgId":4,"status":"WS_NOT_FOUND","errorText":"non-existant-id not found","dataModuleAddVersionResp":{}}`,
	)

	u1.AddSendReqAction("Create invalid module (should fail)",
		`{"dataModuleWriteReq":{"name": "My failed module which has a very long name", "comments": "This should fail", "initialSourceCode": "source 1.0"}}`,
		`{"msgId":5,"status":"WS_BAD_REQUEST","errorText":"Invalid module name: My failed module which has a very long name","dataModuleWriteResp":{}}`,
	)

	u1.AddSendReqAction("Create valid module (should work)",
		`{"dataModuleWriteReq":{"name": "GeoToolkit", "comments": "Geology toolkit", "initialSourceCode": "source 1.0"}}`,
		`{"msgId":6,"status":"WS_OK","dataModuleWriteResp":{
			"module": {
				"id": "${IDSAVE=moduleId1}",
				"name": "GeoToolkit",
				"comments": "Geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
				"versions": [
					{
						"version": {
							"patch": 1
						},
						"comments": "Geology toolkit",
						"timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 1.0"
					}
				]
			}
		}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	moduleId1 := wstestlib.GetIdCreated("moduleId1")

	u1.AddSendReqAction("Edit the module (should fail, source code field)",
		`{"dataModuleWriteReq":{"id": "${IDLOAD=moduleId1}", "name": "GeoKit", "comments": "The geology toolkit", "initialSourceCode": "source 1.1"}}`,
		`{"msgId":7,"status":"WS_BAD_REQUEST",
			"errorText": "InitialSourceCode must not be set for module updates, only name and comments allowed to change",
			"dataModuleWriteResp":{}
		}`,
	)

	u1.AddSendReqAction("Edit the module (should work)",
		`{"dataModuleWriteReq":{"id": "${IDLOAD=moduleId1}", "name": "TheGeokit", "comments": "Our geology toolkit"}}`,
		`{"msgId":8,"status":"WS_OK","dataModuleWriteResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				}
			}
		}}`,
	)

	u1.AddSendReqAction("List modules again",
		`{"dataModuleListReq":{}}`,
		`{"msgId":9,"status":"WS_OK","dataModuleListResp":{
			"modules": {
				"${IDCHK=moduleId1}": {
					"id": "${IDCHK=moduleId1}",
					"name": "TheGeokit",
					"comments": "Our geology toolkit",
					"modifiedUnixSec": "${SECAGO=3}",
					"creator": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"versions": [
						{
							"version": {
								"patch": 1
							},
							"comments": "Geology toolkit",
							"timeStampUnixSec": "${SECAGO=3}"
						}
					]
				}
			}
		}}`,
	)

	u1.AddSendReqAction("Get created module (no version)",
		`{"dataModuleGetReq":{"id": "${IDLOAD=moduleId1}"}}`,
		`{"msgId":10,"status":"WS_OK","dataModuleGetResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
                "versions": [
                    {
                        "version": {
                            "patch": 1
                        },
                        "comments": "Geology toolkit",
                        "timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 1.0"
                    }
                ]
			}
		}}`,
	)

	u1.AddSendReqAction("Get created module (version specified)",
		`{"dataModuleGetReq":{"id": "${IDLOAD=moduleId1}", "version": {"major": 0, "minor": 0, "patch": 1}}}`,
		`{"msgId":11,"status":"WS_OK","dataModuleGetResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
				"versions": [
					{
						"version": {
							"patch": 1
						},
						"comments": "Geology toolkit",
						"timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 1.0"
					}
				]
			}
		}}`,
	)

	u1.AddSendReqAction("Get created module (bad version specified)",
		`{"dataModuleGetReq":{"id": "${IDLOAD=moduleId1}", "version": {"major": 0, "minor": 0, "patch": 3}}}`,
		fmt.Sprintf(`{"msgId":12,"status":"WS_NOT_FOUND","errorText": "%v, version: 0.0.3 not found", "dataModuleGetResp":{}}`, moduleId1),
	)

	u1.AddSendReqAction("Add invalid version to module (should fail)",
		`{"dataModuleAddVersionReq":{"moduleId": "${IDLOAD=moduleId1}", "versionUpdate": "MV_PATCH", "comments": "v0.0.2 comment fail", "tags": ["tag-id-123"]}}`,
		`{"msgId":13,"status":"WS_BAD_REQUEST","errorText":"SourceCode is too short", "dataModuleAddVersionResp":{}}`,
	)

	u1.AddSendReqAction("Add valid version to module (should work)",
		`{"dataModuleAddVersionReq":{"moduleId": "${IDLOAD=moduleId1}", "versionUpdate": "MV_PATCH", "comments": "v0.0.2 comment", "tags": ["tag-id-123"], "sourceCode": "source 0.0.2"}}`,
		`{"msgId":14,"status":"WS_OK","dataModuleAddVersionResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
				"versions": [
					{
						"version": {
							"patch": 1
						},
						"comments": "Geology toolkit",
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"patch": 2
						},
						"tags": [
							"tag-id-123"
						],
						"comments": "v0.0.2 comment",
						"timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 0.0.2"
					}
				]
			}
		}}`,
	)

	u1.AddSendReqAction("Add minor version to module (should work)",
		`{"dataModuleAddVersionReq":{"moduleId": "${IDLOAD=moduleId1}", "versionUpdate": "MV_MINOR", "tags": ["tag-id-123", "tag-id-234"], "sourceCode": "source 0.1.0"}}`,
		`{"msgId":15,"status":"WS_OK","dataModuleAddVersionResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
				"versions": [
					{
						"version": {
							"patch": 1
						},
						"comments": "Geology toolkit",
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"patch": 2
						},
						"tags": [
							"tag-id-123"
						],
						"comments": "v0.0.2 comment",
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"minor": 1
						},
						"tags": [
							"tag-id-123",
							"tag-id-234"
						],
						"timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 0.1.0"
					}
				]
			}
		}}`,
	)

	u1.AddSendReqAction("Add major version to module (should work)",
		`{"dataModuleAddVersionReq":{"moduleId": "${IDLOAD=moduleId1}", "versionUpdate": "MV_MAJOR", "tags": ["tag-id-234"], "comments": "1.0.0 reached", "sourceCode": "source 1.0.0"}}`,
		`{"msgId":16,"status":"WS_OK","dataModuleAddVersionResp":{
			"module": {
				"id": "${IDCHK=moduleId1}",
				"name": "TheGeokit",
				"comments": "Our geology toolkit",
				"modifiedUnixSec": "${SECAGO=3}",
				"creator": {
					"creatorUser": {
						"id": "${USERID}",
						"name": "${REGEXMATCH=test}",
						"email": "${REGEXMATCH=.+@pixlise\\.org}"
					},
					"createdUnixSec": "${SECAGO=3}",
					"canEdit": true
				},
				"versions": [
					{
						"version": {
							"patch": 1
						},
						"comments": "Geology toolkit",
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"patch": 2
						},
						"tags": [
							"tag-id-123"
						],
						"comments": "v0.0.2 comment",
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"minor": 1
						},
						"tags": [
							"tag-id-123",
							"tag-id-234"
						],
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"major": 1
						},
						"tags": [
							"tag-id-234"
						],
						"timeStampUnixSec": "${SECAGO=3}",
						"comments": "1.0.0 reached",
						"sourceCode": "source 1.0.0"
					}
				]
			}
		}}`,
	)

	u1.AddSendReqAction("List modules to see more versions",
		`{"dataModuleListReq":{}}`,
		`{"msgId":17,"status":"WS_OK","dataModuleListResp":{
			"modules": {
				"${IDCHK=moduleId1}": {
					"id": "${IDCHK=moduleId1}",
					"name": "TheGeokit",
					"comments": "Our geology toolkit",
					"modifiedUnixSec": "${SECAGO=3}",
					"creator": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					},
					"versions": [
						{
							"version": {
								"patch": 1
							},
							"comments": "Geology toolkit",
							"timeStampUnixSec": "${SECAGO=3}"
						},
						{
							"version": {
								"patch": 2
							},
							"tags": [
								"tag-id-123"
							],
							"comments": "v0.0.2 comment",
							"timeStampUnixSec": "${SECAGO=3}"
						},
						{
							"version": {
								"minor": 1
							},
							"tags": [
								"tag-id-123",
								"tag-id-234"
							],
							"timeStampUnixSec": "${SECAGO=3}"
						},
						{
							"version": {
								"major": 1
							},
							"tags": [
								"tag-id-234"
							],
							"timeStampUnixSec": "${SECAGO=3}",
							"comments": "1.0.0 reached"
						}
					]
				}
			}
		}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
