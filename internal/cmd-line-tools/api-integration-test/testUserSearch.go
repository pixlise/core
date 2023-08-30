package main

import (
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testUserSearch(apiHost string) {
	// Seed the user DB with some user info for user ids we'll be adding as part of this test
	addDBUsers(&protos.UserDBItem{
		Id: "user-search1",
		Info: &protos.UserInfo{
			Id:    "user-search1",
			Name:  "Ricky Gervais",
			Email: "search@one.com",
		},
	})
	addDBUsers(&protos.UserDBItem{
		Id: "user-search2",
		Info: &protos.UserInfo{
			Id:    "user-search2",
			Name:  "Richard Dawkins",
			Email: "search@two.com",
		},
	})
	addDBUsers(&protos.UserDBItem{
		Id: "user-search3",
		Info: &protos.UserInfo{
			Id:    "user-search3",
			Name:  "George Carlin",
			Email: "search3@onetwo.com",
		},
	})

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("Search for users 1",
		`{"userSearchReq":{"searchString": "two"}}`,
		`{"msgId":1,"status":"WS_OK","userSearchResp":{
			"users": [
				{
					"id": "user-search2",
					"name": "Richard Dawkins",
					"email": "search@two.com"
				},
				{
					"id": "user-search3",
					"name": "George Carlin",
					"email": "search3@onetwo.com"
				}
			]
		}}`,
	)

	u1.AddSendReqAction("Search for users 2",
		`{"userSearchReq":{"searchString": "one"}}`,
		`{"msgId":2,"status":"WS_OK","userSearchResp":{
			"users": [
				{
					"id": "user-search1",
					"name": "Ricky Gervais",
					"email": "search@one.com"
				},
				{
					"id": "user-search3",
					"name": "George Carlin",
					"email": "search3@onetwo.com"
				}
			]
		}}`,
	)

	u1.AddSendReqAction("Search for users 3",
		`{"userSearchReq":{"searchString": "Ric"}}`,
		`{"msgId":3,"status":"WS_OK","userSearchResp":{
			"users": [
				{
					"id": "user-search1",
					"name": "Ricky Gervais",
					"email": "search@one.com"
				},
				{
					"id": "user-search2",
					"name": "Richard Dawkins",
					"email": "search@two.com"
				}
			]
		}}`,
	)

	u1.AddSendReqAction("Search for users 4",
		`{"userSearchReq":{"searchString": "in"}}`,
		`{"msgId":4,"status":"WS_OK","userSearchResp":{
			"users": [
				{
					"id": "user-search2",
					"name": "Richard Dawkins",
					"email": "search@two.com"
				},
				{
					"id": "user-search3",
					"name": "George Carlin",
					"email": "search3@onetwo.com"
				}
			]
		}}`,
	)

	u1.AddSendReqAction("Search for users 5",
		`{"userSearchReq":{"searchString": ".com"}}`,
		`{"msgId":5,"status":"WS_OK","userSearchResp":{
			"users": [
				{
					"id": "user-search1",
					"name": "Ricky Gervais",
					"email": "search@one.com"
				},
				{
					"id": "user-search2",
					"name": "Richard Dawkins",
					"email": "search@two.com"
				},
				{
					"id": "user-search3",
					"name": "George Carlin",
					"email": "search3@onetwo.com"
				}
			]
		}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
