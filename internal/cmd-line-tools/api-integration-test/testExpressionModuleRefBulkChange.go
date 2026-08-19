package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testExpressionModuleRefBulkChange(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create module (should work)",
		`{"dataModuleWriteReq":{"name": "GeoToolkit", "comments": "Geology toolkit", "initialSourceCode": "source 1.0"}}`,
		`{"msgId":1,"status":"WS_OK","dataModuleWriteResp":{
			"module": {
				"id": "${IDSAVE=ExprRefModuleId}",
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

	u1.AddSendReqAction("Add version to module (should work)",
		`{"dataModuleAddVersionReq":{"moduleId": "${IDLOAD=ExprRefModuleId}", "versionUpdate": "MV_PATCH", "comments": "v0.0.2 comment","sourceCode": "source 0.0.2"}}`,
		`{"msgId":2,"status":"WS_OK","dataModuleAddVersionResp":{
			"module": {
				"id": "${IDCHK=ExprRefModuleId}",
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
						"timeStampUnixSec": "${SECAGO=3}"
					},
					{
						"version": {
							"patch": 2
						},
						"comments": "v0.0.2 comment",
						"timeStampUnixSec": "${SECAGO=3}",
						"sourceCode": "source 0.0.2"
					}
				]
			}
		}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	moduleId := wstestlib.GetIdCreated("ExprRefModuleId")

	// Create 3 expressions, 2 reference the module (with differing versions), one does not
	u1.AddSendReqAction("Create expression 1 (mod ref old)",
		`{"expressionWriteReq":{
			"expression": {
				"name": "User1 Expression1",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"sourceLanguage": "LUA",
				"sourceCode": "element(\"Ca\")",
				"moduleReferences": [
					{
						"moduleId": "a-module",
						"version": {"major": 2, "patch": 1}
					},
					{
						"moduleId": "${IDLOAD=ExprRefModuleId}",
						"version": {"patch": 4}
					}
				]
			}
		}}`,
		`{"msgId":3,"status":"WS_OK",
			"expressionWriteResp":{
				"expression": {
					"id":"${IDSAVE=ExprModRef1}",
					"name": "User1 Expression1",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR RUNTIME STAT SAVE checking",
					"moduleReferences": [
						{
							"moduleId": "a-module",
							"version": {
								"major": 2,
								"patch": 1
							}
						},
						{
							"moduleId": "${IDCHK=ExprRefModuleId}",
							"version": {
								"patch": 4
							}
						}
					],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Create expression 2 (mod ref last ver)",
		`{"expressionWriteReq":{
			"expression": {
				"name": "User1 Expression2",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"sourceLanguage": "LUA",
				"sourceCode": "element(\"Ca\")",
				"moduleReferences": [{
					"moduleId": "${IDLOAD=ExprRefModuleId}",
					"version": {"patch": 1}
				}]
			}
		}}`,
		`{"msgId":4,"status":"WS_OK",
			"expressionWriteResp":{
				"expression": {
					"id":"${IDSAVE=ExprModRef2}",
					"name": "User1 Expression2",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR RUNTIME STAT SAVE checking",
					"moduleReferences": [
						{
							"moduleId": "${IDCHK=ExprRefModuleId}",
							"version": {
								"patch": 1
							}
						}
					],
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				}
			}
		}`,
	)

	u1.AddSendReqAction("Create expression 3 (no mod ref)",
		`{"expressionWriteReq":{
			"expression": {
				"name": "User1 Expression3",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"sourceLanguage": "LUA",
				"sourceCode": "element(\"Ca\")"
			}
		}}`,
		`{"msgId":5,"status":"WS_OK",
			"expressionWriteResp":{
				"expression": {
					"id":"${IDSAVE=ExprModRef3}",
					"name": "User1 Expression3",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR RUNTIME STAT SAVE checking",
					"modifiedUnixSec": "${SECAGO=3}",
					"owner": {
						"creatorUser": {
							"id": "${USERID}",
							"name": "${REGEXMATCH=test}",
							"email": "${REGEXMATCH=.+@pixlise\\.org}"
						},
						"createdUnixSec": "${SECAGO=3}",
						"canEdit": true
					}
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	expr3Id := wstestlib.GetIdCreated("ExprModRef3")

	u1.AddSendReqAction("Send bulk edit for no ids",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"moduleId": "${IDLOAD=ExprRefModuleId}",
			"version": {"major": 1, "patch": 1}
		}}`,
		`{
			"msgId": 6,
			"status": "WS_BAD_REQUEST",
			"errorText": "ExpressionIds must contain at least 1 items",
			"bulkReplaceExpressionModuleReferenceResp": {}
		}`,
	)

	u1.AddSendReqAction("Send bulk edit for empty module",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"version": {"major": 1, "patch": 1},
			"expressionIds": [
				"${IDLOAD=ExprModRef1}",
				"${IDLOAD=ExprModRef2}",
				"${IDLOAD=ExprModRef3}",
				"non-existant-expr-id"
			]
		}}`,
		`{
			"msgId": 7,
			"status": "WS_BAD_REQUEST",
			"errorText": "ModuleId is too short",
			"bulkReplaceExpressionModuleReferenceResp": {}
		}`,
	)

	u1.AddSendReqAction("Send bulk edit for invalid module",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"moduleId": "non-existant-module",
			"version": {"major": 1, "patch": 1},
			"expressionIds": [
				"${IDLOAD=ExprModRef1}",
				"${IDLOAD=ExprModRef2}",
				"${IDLOAD=ExprModRef3}",
				"non-existant-expr-id"
			]
		}}`,
		`{
			"msgId": 8,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant-module not found",
			"bulkReplaceExpressionModuleReferenceResp": {}
		}`,
	)

	u1.AddSendReqAction("Send bulk edit for invalid module version",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"moduleId": "${IDLOAD=ExprRefModuleId}",
			"version": {"major": 4, "minor": 3, "patch": 1},
			"expressionIds": [
				"${IDLOAD=ExprModRef1}",
				"${IDLOAD=ExprModRef2}",
				"${IDLOAD=ExprModRef3}",
				"non-existant-expr-id"
			]
		}}`,
		fmt.Sprintf(`{
			"msgId": 9,
			"status": "WS_NOT_FOUND",
			"errorText": "%v, version: 4.3.1 not found",
			"bulkReplaceExpressionModuleReferenceResp": {}
		}`, moduleId),
	)

	u1.AddSendReqAction("Send bulk edit, should succeed but return errors for missing expr",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"moduleId": "${IDLOAD=ExprRefModuleId}",
			"version": {"patch": 2},
			"expressionIds": [
				"${IDLOAD=ExprModRef1}",
				"${IDLOAD=ExprModRef2}",
				"non-existant-expr-id"
			]
		}}`,
		`{"msgId":10,"status":"WS_OK",
			"bulkReplaceExpressionModuleReferenceResp": {
			"errors": {
				"non-existant-expr-id": "Failed to read expression non-existant-expr-id: non-existant-expr-id not found"
			}
		}}`,
	)

	u1.AddSendReqAction("Send bulk edit, should succeed but return error for expr 3 that doesn't have ref",
		`{"bulkReplaceExpressionModuleReferenceReq":{
			"moduleId": "${IDLOAD=ExprRefModuleId}",
			"version": {"patch": 2},
			"expressionIds": [
				"${IDLOAD=ExprModRef3}"
			]
		}}`,
		fmt.Sprintf(`{"msgId":11,"status":"WS_OK",
			"bulkReplaceExpressionModuleReferenceResp": {
			"errors": {
				"%v": "Skipped updating reference for expression %v - it does not reference module %v"
			}
		}}`, expr3Id, expr3Id, moduleId),
	)

	// Check each expression now
	u1.AddSendReqAction("Get expr 1",
		`{"expressionGetReq":{"id": "${IDLOAD=ExprModRef1}"}}`,
		`{"msgId":12,"status":"WS_OK","expressionGetResp": {
			"expression": {
				"id":"${IDCHK=ExprModRef1}",
				"name": "User1 Expression1",
				"sourceCode": "element(\"Ca\")",
				"sourceLanguage": "LUA",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"moduleReferences": [
					{
						"moduleId": "a-module",
						"version": {
							"major": 2,
							"patch": 1
						}
					},
					{
						"moduleId": "${IDCHK=ExprRefModuleId}",
						"version": {
							"patch": 2
						}
					}
				],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
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

	u1.AddSendReqAction("Get expr 2",
		`{"expressionGetReq":{"id": "${IDLOAD=ExprModRef2}"}}`,
		`{"msgId":13,"status":"WS_OK","expressionGetResp": {
			"expression": {
				"id":"${IDCHK=ExprModRef2}",
				"name": "User1 Expression2",
				"sourceCode": "element(\"Ca\")",
				"sourceLanguage": "LUA",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"moduleReferences": [
					{
						"moduleId": "${IDCHK=ExprRefModuleId}",
						"version": {
							"patch": 2
						}
					}
				],
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
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

	u1.AddSendReqAction("Get expr 3",
		`{"expressionGetReq":{"id": "${IDLOAD=ExprModRef3}"}}`,
		`{"msgId":14,"status":"WS_OK","expressionGetResp": {
			"expression": {
				"id":"${IDCHK=ExprModRef3}",
				"name": "User1 Expression3",
				"sourceCode": "element(\"Ca\")",
				"sourceLanguage": "LUA",
				"comments": "FOR RUNTIME STAT SAVE checking",
				"modifiedUnixSec": "${SECAGO=3}",
				"owner": {
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
	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
