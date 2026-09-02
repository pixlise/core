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
		`{"jobListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","jobListResp":{}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}

func testJobSchedule(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &client.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction("List scheduled jobs, should be empty",
		`{"scheduledJobListReq":{}}`,
		`{"msgId":1,"status":"WS_OK","scheduledJobListResp":{}}`,
	)

	u1.AddSendReqAction("Modify non-existant scheduled job",
		`{"setScheduledJobReq":{"job": {
			"id": "sched123",
			"scheduleType": 1,
			"jobOrder": -3,
			"jobType": 6,
			"jobParameters": {"scanId": "imported", "expressionId": "expr123", "quant": "id:quant123"}
		}}}`,
		`{
		"msgId": 2,
		"status": "WS_SERVER_ERROR",
		"errorText": "Failed to update scheduled job \"sched123\": Not found",
		"setScheduledJobResp": {}
		}`,
	)

	u1.AddSendReqAction("Create invalid scheduled job",
		`{"setScheduledJobReq":{"job": {
			"jobType": 6,
			"scheduleType": 1,
			"jobParameters": {"scanId": "imported", "expressionId": "expr123", "quant": "id:quant123"},
			"intervalSec": 12
		}}}`,
		`{"msgId": 3,
			"status": "WS_SERVER_ERROR",
  			"errorText": "AFTER_IMPORT jobs must not have time fields set",
			"setScheduledJobResp": {}
		}`,
	)

	u1.AddSendReqAction("Create valid scheduled job",
		`{"setScheduledJobReq":{"job": {
			"scheduleType": 1,
			"jobOrder": -3,
			"jobType": 6,
			"jobParameters": {"scanId": "imported", "expressionId": "expr123", "quant": "id:quant123"}
		}}}`,
		`{"msgId":4,"status":"WS_OK","setScheduledJobResp":{"job": {
			"id": "${IDSAVE=schedjob1}",
			"scheduleType": "AFTER_IMPORT",
			"jobOrder": "-3",
			"jobType": "JT_RUN_EXPRESSION",
			"jobParameters": {
				"expressionId": "expr123",
				"quant": "id:quant123",
				"scanId": "imported"
			}
		}}}`,
	)

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)

	u1.AddSendReqAction("List scheduled jobs, should contain new one",
		`{"scheduledJobListReq":{}}`,
		`{"msgId":5,"status":"WS_OK","scheduledJobListResp":{	
			"jobs": [{
				"id": "${IDCHK=schedjob1}",
				"scheduleType": "AFTER_IMPORT",
				"jobOrder": "-3",
				"jobType": "JT_RUN_EXPRESSION",
				"jobParameters": {
					"expressionId": "expr123",
					"quant": "id:quant123",
					"scanId": "imported"
				}
			}]
		}}`,
	)

	u1.AddSendReqAction("Edit scheduled job",
		`{"setScheduledJobReq":{"job": {
			"id": "${IDLOAD=schedjob1}",
			"scheduleType": 1,
			"jobOrder": 2,
			"jobType": 6,
			"jobParameters": {"scanId": "imported", "expressionId": "expr1234", "quant": "id:quant123"}
		}}}`,
		`{"msgId":6,"status":"WS_OK","setScheduledJobResp":{"job": {
			"id": "${IDCHK=schedjob1}",
			"scheduleType": "AFTER_IMPORT",
			"jobOrder": "2",
			"jobType": "JT_RUN_EXPRESSION",
			"jobParameters": {
				"expressionId": "expr1234",
				"quant": "id:quant123",
				"scanId": "imported"
			}
		}}}`,
	)

	u1.AddSendReqAction("List scheduled jobs, should contain edited one",
		`{"scheduledJobListReq":{}}`,
		`{"msgId":7,"status":"WS_OK","scheduledJobListResp":{
			"jobs": [{
				"id": "${IDCHK=schedjob1}",
				"scheduleType": "AFTER_IMPORT",
				"jobOrder": "2",
				"jobType": "JT_RUN_EXPRESSION",
				"jobParameters": {
					"expressionId": "expr1234",
					"quant": "id:quant123",
					"scanId": "imported"
				}
			}]
		}}`,
	)

	u1.AddSendReqAction("Delete non-existant job",
		`{"deleteScheduledJobReq":{"id": "non-existant-sched-job123"}}`,
		`{"msgId":8,
		"status": "WS_SERVER_ERROR",
		"errorText": "Failed to delete scheduled job \"non-existant-sched-job123\": Not found",
		"deleteScheduledJobResp": {}}`,
	)

	u1.AddSendReqAction("Delete created job",
		`{"deleteScheduledJobReq":{"id": "${IDLOAD=schedjob1}"}}`,
		`{"msgId":9,"status":"WS_OK","deleteScheduledJobResp":{}}`,
	)

	u1.AddSendReqAction("List scheduled jobs, should contain edited one",
		`{"scheduledJobListReq":{}}`,
		`{"msgId":10,"status":"WS_OK","scheduledJobListResp":{}}`,
	)

	// string id = 1; // @gotags: bson:"_id,omitempty"
	// ScheduleType scheduleType = 2;
	// int64 scheduledFirstTimeUnixSec = 3;
	// int64 intervalSec = 7;
	// int64 jobOrder = 4;
	// JobType jobType = 5;
	// map<string, string> jobParameters = 6;

	u1.CloseActionGroup([]string{}, 5000)
	wstestlib.ExecQueuedActions(&u1)
}
