package main

import "github.com/pixlise/core/v4/core/wstestlib"

func testDOI(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Create expression",
		`{"expressionWriteReq":{
			"expression": {
				"name": "User1 Expression",
				"comments": "FOR DOI TEST checking",
				"sourceLanguage": "LUA",
				"sourceCode": "element(\"Ca\")"
			}
		}}`,
		`{"msgId":1,"status":"WS_OK",
			"expressionWriteResp":{
				"expression": {
					"id": "${IDSAVE=DOI_SAVED_ID}",
					"name": "User1 Expression",
					"sourceCode": "element(\"Ca\")",
					"sourceLanguage": "LUA",
					"comments": "FOR DOI TEST checking",
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

	u1.AddSendReqAction("Publish DOI for expression",
		`{"publishExpressionToZenodoReq":{
			"id": "${IDLOAD=DOI_SAVED_ID}",
			"output": "Zipped expression output",
			"metadata": {
				"title": "DOI Test",
				"creators": [
					{
						"name": "Test User",
						"affiliation": "Pixlise",
						"orcid": "0000-0002-1825-0097"
					}
				],
				"description": "This is a test DOI",
				"keywords": "DOI, Test",
				"notes": "This is a test DOI",
				"relatedIdentifiers": [
					{
						"identifier": "https://pixlise.org",
						"relation": "isAlternateIdentifier"
					}
				],
				"contributors": [],
				"references": "",
				"version": "1.0",
				"doi": "",
				"doiBadge": "",
				"doiLink": ""
			}
		}}`,
		`{"msgId":2,"status":"WS_OK",
			"publishExpressionToZenodoResp":{
				"doi": {
					"id": "${IDCHK=DOI_SAVED_ID}",
					"title": "DOI Test",
					"creators": [
						{
							"name": "Test User",
							"affiliation": "Pixlise",
							"orcid": "0000-0002-1825-0097"
						}
					],
					"description": "This is a test DOI",
					"keywords": "DOI, Test",
					"notes": "This is a test DOI",
					"relatedIdentifiers": [
						{
							"identifier": "https://pixlise.org",
							"relation": "isAlternateIdentifier"
						}
					],
					"version": "1.0",
					"doi": "${IGNORE}",
					"doiBadge": "${IGNORE}",
					"doiLink": "${IGNORE}"
				}
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, 10000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("Get DOI",
		`{"zenodoDOIGetReq":{
			"id": "${IDLOAD=DOI_SAVED_ID}"
		}}`,
		`{"msgId":3,"status":"WS_OK",
			"zenodoDOIGetResp":{
				"doi": {
					"id": "${IDCHK=DOI_SAVED_ID}",
					"title": "DOI Test",
					"creators": [
						{
							"name": "Test User",
							"affiliation": "Pixlise",
							"orcid": "0000-0002-1825-0097"
						}
					],
					"description": "This is a test DOI",
					"keywords": "DOI, Test",
					"notes": "This is a test DOI",
					"relatedIdentifiers": [
						{
							"identifier": "https://pixlise.org",
							"relation": "isAlternateIdentifier"
						}
					],
					"version": "1.0",
					"doi": "${IGNORE}",
					"doiBadge": "${IGNORE}",
					"doiLink": "${IGNORE}"
				}
			}
		}`,
	)

	u1.AddSendReqAction("Delete DOI expression so we don't mess up listings for the user content one",
		`{"expressionDeleteReq":{"id": "${IDLOAD=DOI_SAVED_ID}"}}`,
		`{"msgId":4,"status":"WS_OK", "expressionDeleteResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

}
