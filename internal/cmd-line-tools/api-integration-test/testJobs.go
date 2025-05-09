package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/wstestlib"
)

func testJobs(apiHost string) {
	// Drop jobs
	db := wstestlib.GetDB()
	ctx := context.TODO()
	// Seed jobs
	coll := db.Collection(dbCollections.JobsName)
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}

	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List jobs, should be empty",
		`{"scanListJobsReq":{}}`,
		`{"msgId":1,"status":"WS_OK","scanListJobsResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("Create job1",
		`{"scanJobWriteReq":{job1}}`,
		`{"msgId":2,"status":"WS_OK","scanJobWriteResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("Create job2",
		`{"scanJobWriteReq":{job2}}`,
		`{"msgId":2,"status":"WS_OK","scanJobWriteResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("List jobs, should be 2",
		`{"scanListJobsReq":{}}`,
		`{"msgId":1,"status":"WS_OK","scanListJobsResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("Delete job1",
		`{"scanJobDeleteReq":{job1}}`,
		`{"msgId":2,"status":"WS_OK","scanJobDeleteResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("List jobs, should be 2",
		`{"scanListJobsReq":{}}`,
		`{"msgId":1,"status":"WS_OK","scanListJobsResp":{"jobs": []}}`,
	)

	u1.AddSendReqAction("Trigger non-existant job",
		`{"scanTriggerJobReq":{job1}}`,
		`{"msgId":1,"status":"WS_OK","scanTriggerJobResp":{}}`,
	)

	u1.AddSendReqAction("Trigger job 2",
		`{"scanTriggerJobReq":{job2}}`,
		`{"msgId":1,"status":"WS_OK","scanTriggerJobResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
