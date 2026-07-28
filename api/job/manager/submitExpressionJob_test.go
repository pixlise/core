package jobmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job/jobnode"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/proto"
)

func Example_jobmanager_makeLuaExpressionId() {
	fmt.Println(makeLuaExpressionId("memo123"))
	fmt.Println(makeLuaExpressionId("exprcachev1_GeoAndDiff_3_5_3_TiO2_690422275_quant-h6q1rzkba0e0zt8g"))
	fmt.Println(makeLuaExpressionId("{\"scanId\":\"690422275\",\"exprId\":\"9b4h4zjuynpshf7c\",\"quantId\":\"quant-h6q1rzkba0e0zt8g\",\"roiId\":\"AllPoints-690422275\",\"units\":0},Resp:false,exprMod:1777502963,spectra:4674,90,0"))

	// Output:
	// expr-lua-bWVtbzEyMw
	// expr-lua-ZXhwcmNhY2hldjFfR2VvQW5kRGlmZl8zXzVfM19UaU8yXzY5MDQyMjI3NV9xdWFudC1oNnExcnprYmEwZTB6dDhn
	// expr-lua-eyJzY2FuSWQiOiI2OTA0MjIyNzUiLCJleHBySWQiOiI5YjRoNHpqdXlucHNoZjdjIiwicXVhbnRJZCI6InF1YW50LWg2cTFyemtiYTBlMHp0OGciLCJyb2lJZCI6IkFsbFBvaW50cy02OTA0MjIyNzUiLCJ1bml0cyI6MH0sUmVzcDpmYWxzZSxleHByTW9kOjE3Nzc1MDI5NjMsc3BlY3RyYTo0Njc0LDkwLDA
}

func Example_jobmanager_canRunExpressionJob() {
	ts := &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785129501}}
	can, clear := canRunExpressionJob(nil, ts, 600)
	fmt.Printf("nil item:\t\t\t%v|%v\n", can, clear)

	existingJobItem := &protos.JobStatus{
		JobId:                 "",
		Status:                protos.JobStatus_COMPLETE,
		Message:               "blah",
		Name:                  "blah",
		JobType:               protos.JobType_JT_RUN_EXPRESSION,
		LastUpdateUnixTimeSec: 1785129500,
	}

	ts = &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785129501}}
	can, clear = canRunExpressionJob(existingJobItem, ts, 600)
	fmt.Printf("invalid item:\t\t%v|%v\n", can, clear)

	existingJobItem.JobId = "expr-lua-1dvqt5afqk9jdfuw"
	existingJobItem.Status = protos.JobStatus_COMPLETE
	ts = &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785129501}}
	can, clear = canRunExpressionJob(existingJobItem, ts, 600)
	fmt.Printf("complete item:\t\t%v|%v\n", can, clear)

	ts = &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785130101}}
	can, clear = canRunExpressionJob(existingJobItem, ts, 600)
	fmt.Printf("old complete item:\t%v|%v\n", can, clear)

	existingJobItem.Status = protos.JobStatus_GATHERING_RESULTS
	ts = &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785129501}}
	can, clear = canRunExpressionJob(existingJobItem, ts, 600)
	fmt.Printf("still running:\t\t%v|%v\n", can, clear)

	existingJobItem.Status = protos.JobStatus_ERROR
	ts = &timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{1785129501}}
	can, clear = canRunExpressionJob(existingJobItem, ts, 600)
	fmt.Printf("error:\t\t\t\t%v|%v\n", can, clear)

	// Output:
	// nil item:			true|false
	// invalid item:		true|false
	// complete item:		false|false
	// old complete item:	true|true
	// still running:		false|false
	// error:				true|true
}

func Example_jobmanager_SubmitExpressionJob_048300551_NoExpr() {
	logLev := logger.LogInfo
	origWD, _, svcs := initJobManagerTest(&logLev, []int64{
		1668142579, // dataset local file cache time stamp
		1668142580, // start time stamp
		1668142581, // queue time stamp
		1668142582, // queue time stamp
		1668142583, // queue time stamp
		1668142584, // queue time stamp
		1668142585, // queue time stamp
		1668142586, // queue time stamp
		1668142587, // queue time stamp
		1668142588, // queue time stamp
		1668142589, // queue time stamp
		1668142590, // queue time stamp
		1668142591, // queue time stamp
		1668142592, // queue time stamp
		1668142593, // queue time stamp
	})
	defer os.Chdir(origWD)

	svcs.Config.Jobs.NodeCountOverride = 4
	svcs.Log = &logger.StdOutLogger{}
	svcs.Log.SetLogLevel(logger.LogDebug)

	jm, err := CreateJobManager(&svcs, 0, false, true)
	fmt.Printf("jm Create: %v\n", err)

	jm.SubmitExpressionJob("048300551", "quant-ggy6zxhn23p7rlv9", "non-existant-expr", "", "", nil, nil)

	// Output:
	// jm Create: <nil>
	// ERROR: SubmitExpressionJob error: Failed to read map[_id:non-existant-expr] from collection expressions: mongo: no documents in result
}

// More tests to write:
// Missing quant file
// Missing scan file
// Missing source file in S3
// Missing diffraction file

func Example_jobmanager_SubmitExpressionJob_048300551_NoQuant() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	runExprJobTest(exprId, scanId, quantId, modIds, modVers, "", "", "non-existant", false)

	// Output:
	// jm Create: <nil>
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9non-existant", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109720 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9non-existant,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// ERROR: Expression runner could not fetch quant: mongo: no documents in result
	// ERROR: Job expr-lua-bWVtbzEyMw-node-0 failed: <string>:2221: PIXLISE-Lua Runtime error: Expression runner could not fetch quant: mongo: no documents in result
	// stack traceback:
	// 	[G]: in function 'exists'
	// 	<string>:2221: in function 'getElmtList'
	// 	<string>:2229: in main chunk
	// 	[G]: ?
	// ERROR: Failed to start job expr-lua-bWVtbzEyMw (node 0): Job expr-lua-bWVtbzEyMw-node-0 failed: <string>:2221: PIXLISE-Lua Runtime error: Expression runner could not fetch quant: mongo: no documents in result
	// stack traceback:
	// 	[G]: in function 'exists'
	// 	<string>:2221: in function 'getElmtList'
	// 	<string>:2229: in main chunk
	// 	[G]: ?
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 0 completed nodes of 1
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status ERROR, message: 1 nodes failed
	// INFO:   Marking job expr-lua-bWVtbzEyMw as ERROR due to nodes not all completing
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: Failed to read map[_id:exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9] from collection memoisedItems: mongo: no documents in result
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: Failed to read map[_id:exprcachev1_GeoAndDiff_3_5_3_geometry_048300551] from collection memoisedItems: mongo: no documents in result
	// Read memoised memo123 errors: Failed to read map[_id:memo123] from collection memoisedItems: mongo: no documents in result
	// Decode memoised memo123 errors: <nil>
}

func Example_jobmanager_SubmitExpressionJob_048300551_NoScan() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	runExprJobTest(exprId, scanId, quantId, modIds, modVers, "", "non-existant", "", false)

	// Output:
	// jm Create: <nil>
	// ERROR: SubmitExpressionJob error: Failed to read map[_id:048300551non-existant] from collection scans: mongo: no documents in result
	// SubmitExpressionJob: UNKNOWN, Failed to read map[_id:048300551non-existant] from collection scans: mongo: no documents in result
}

func Example_jobmanager_SubmitExpressionJob_048300551_NoModVersion() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v99.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	runExprJobTest(exprId, scanId, quantId, modIds, modVers, "", "", "", false)

	// Output:
	// jm Create: <nil>
	// ERROR: SubmitExpressionJob error: Failed to read map[_id:ng46r8vwzr3z28ui-v0.8.0] from collection moduleVersions: mongo: no documents in result
	// SubmitExpressionJob: UNKNOWN, Failed to read map[_id:ng46r8vwzr3z28ui-v0.8.0] from collection moduleVersions: mongo: no documents in result
}

func Example_jobmanager_SubmitExpressionJob_048300551_ExprModSyntaxError() {
	exprId := "u59sahioy18frfl9-badver"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v99.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	runExprJobTest(exprId, scanId, quantId, modIds, modVers, "", "", "", false)

	// Output:
	// jm Create: <nil>
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9-badver", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 103831 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9-badver,memoKey=memo123]
	// ERROR: Job expr-lua-bWVtbzEyMw-node-0 failed: <string> line:977(column:12) near 'error':   parse error
	//
	// ERROR: Failed to start job expr-lua-bWVtbzEyMw (node 0): Job expr-lua-bWVtbzEyMw-node-0 failed: <string> line:977(column:12) near 'error':   parse error
	//
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 0 completed nodes of 1
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status ERROR, message: 1 nodes failed
	// INFO:   Marking job expr-lua-bWVtbzEyMw as ERROR due to nodes not all completing
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: Failed to read map[_id:exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9] from collection memoisedItems: mongo: no documents in result
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: Failed to read map[_id:exprcachev1_GeoAndDiff_3_5_3_geometry_048300551] from collection memoisedItems: mongo: no documents in result
	// Read memoised memo123 errors: Failed to read map[_id:memo123] from collection memoisedItems: mongo: no documents in result
	// Decode memoised memo123 errors: <nil>
}

func Example_jobmanager_SubmitExpressionJob_048300551_OK() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	runExprJobTest(exprId, scanId, quantId, modIds, modVers, "", "", "", true)

	// Output:
	// jm Create: <nil>
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109708 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// DEBUG: Downloading file: s3://users-bucket/Quantifications/048300551/PIXLISEImport/quant-ggy6zxhn23p7rlv9.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/diffraction-db.bin
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log
	// DEBUG: Upload output.csv -> s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group expr-lua-bWVtbzEyMw completion task...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// QueryQ: <nil>
	// Queue items at end: 0
	// Query jobs: <nil>
	// Jobs at end: 1
	// Job[0] id: expr-lua-bWVtbzEyMw
	// Query status: <nil>
	// Job status at end: 1
	// JobStatus[0] id: expr-lua-bWVtbzEyMw, status: COMPLETE, msg: "Nodes ran: 1"
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: <nil>
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: <nil>
	// Read memoised memo123 errors: <nil>
	// Decode memoised memo123 errors: <nil>
	// Reading expected-expr-output.txt error: <nil>
	// Expected csv format ok: true
}

func Example_jobmanager_SubmitExpressionJob_048300551_OK_NoDuplicateRuns() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"
	jm, origWD := setupForTest(exprId, scanId, quantId, modIds, modVers)
	defer os.Chdir(origWD)
	svcs := jm.svcs

	status, err := jm.SubmitExpressionJob(scanId, quantId, exprId, "", "memo123", nil, nil)
	var s protos.JobStatus_Status
	if status != nil {
		s = status.Status
	}
	fmt.Printf("SubmitExpressionJob: %v, %v\n", s, err)

	if err != nil {
		return
	}

	// Start the same thing again
	status, err = jm.SubmitExpressionJob(scanId, quantId, exprId, "", "memo123", nil, nil)
	if status != nil {
		s = status.Status
	}

	fmt.Printf("SubmitExpressionJob2: %v, %v\n", s, err)

	if err != nil {
		return
	}

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "",
		servicesMock.JobBucketForUnitTest, servicesMock.ConfigBucketForUnitTest, servicesMock.UsersBucketForUnitTest, servicesMock.DatasetsBucketForUnitTest,
		svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{status.JobItemId + "-node-0"})

	jm.RunCheckJobQueueForTest()
	// time.Sleep(10 * time.Second)
	// jm.RunCheckJobQueueForTest()

	verifyMemoItems(svcs, origWD)

	// Output:
	// jm Create: <nil>
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob: STARTING, <nil>
	// INFO: Found existing expression job for expr-lua-bWVtbzEyMw with state STARTING. Skipping starting a new/duplicate one.
	// SubmitExpressionJob2: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109708 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// DEBUG: Downloading file: s3://users-bucket/Quantifications/048300551/PIXLISEImport/quant-ggy6zxhn23p7rlv9.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/diffraction-db.bin
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log
	// DEBUG: Upload output.csv -> s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group expr-lua-bWVtbzEyMw completion task...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: <nil>
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: <nil>
	// Read memoised memo123 errors: <nil>
	// Decode memoised memo123 errors: <nil>
	// Reading expected-expr-output.txt error: <nil>
	// Expected csv format ok: true
}

func Example_jobmanager_SubmitExpressionJob_048300551_OK_AllowSecondRunToOverwrite() {
	exprId := "u59sahioy18frfl9"
	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"
	jm, origWD := setupForTest(exprId, scanId, quantId, modIds, modVers)
	defer os.Chdir(origWD)
	svcs := jm.svcs

	svcs.Config.ExpressionRerunIntervalSec = 5
	status, err := jm.SubmitExpressionJob(scanId, quantId, exprId, "", "memo123", nil, nil)
	var s protos.JobStatus_Status
	if status != nil {
		s = status.Status
	}
	fmt.Printf("SubmitExpressionJob: %v, %v\n", s, err)

	if err != nil {
		return
	}

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "",
		servicesMock.JobBucketForUnitTest, servicesMock.ConfigBucketForUnitTest, servicesMock.UsersBucketForUnitTest, servicesMock.DatasetsBucketForUnitTest,
		svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{status.JobItemId + "-node-0"})

	jm.RunCheckJobQueueForTest()
	// time.Sleep(10 * time.Second)
	// jm.RunCheckJobQueueForTest()

	time.Sleep(time.Second)

	verifyMemoItems(svcs, origWD)

	// Set new time stamps
	ts := []int64{}
	startTS := svcs.TimeStamper.GetTimeNowSec()
	for c := int64(0); c < 20; c++ {
		ts = append(ts, int64(startTS+20+c))
	}
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{QueuedTimeStamps: ts}

	// Start the same thing again
	status, err = jm.SubmitExpressionJob(scanId, quantId, exprId, "", "memo123", nil, nil)
	if status != nil {
		s = status.Status
	}

	fmt.Printf("SubmitExpressionJob2: %v, %v\n", s, err)

	if err != nil {
		return
	}

	// Run the job node queue processing code
	jn = jobnode.CreateJobNode("pixlise-job", "",
		servicesMock.JobBucketForUnitTest, servicesMock.ConfigBucketForUnitTest, servicesMock.UsersBucketForUnitTest, servicesMock.DatasetsBucketForUnitTest,
		svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{status.JobItemId + "-node-0"})

	jm.RunCheckJobQueueForTest()
	// time.Sleep(10 * time.Second)
	// jm.RunCheckJobQueueForTest()

	time.Sleep(time.Second)
	verifyMemoItems(svcs, origWD)

	// Output:
	// jm Create: <nil>
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109708 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// DEBUG: Downloading file: s3://users-bucket/Quantifications/048300551/PIXLISEImport/quant-ggy6zxhn23p7rlv9.bin
	// DEBUG: Total locally cached files: 1, 117113 bytes, removed 0
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: Total locally cached files: 2, 960590 bytes, removed 0
	// DEBUG: Reading local file: /tmp/scan-048300551-dataset.bin
	//
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/diffraction-db.bin
	// DEBUG: Total locally cached files: 3, 983221 bytes, removed 0
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log
	// DEBUG: Upload output.csv -> s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group expr-lua-bWVtbzEyMw completion task...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: <nil>
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: <nil>
	// Read memoised memo123 errors: <nil>
	// Decode memoised memo123 errors: <nil>
	// Reading expected-expr-output.txt error: <nil>
	// Expected csv format ok: true
	// INFO: Cleared expr-lua-bWVtbzEyMw entry in jobs: &{DeletedCount:1}
	// INFO: Cleared expr-lua-bWVtbzEyMw entry in jobStatuses: &{DeletedCount:1}
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitExpressionJob2: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "expr-lua-bWVtbzEyMw-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-bWVtbzEyMw-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109708 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// DEBUG: Reading local file: /tmp/quant-quant-ggy6zxhn23p7rlv9-quant.bin
	//
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/stdout.log
	// DEBUG: Upload output.csv -> s3://job-bucket/JobData/048300551/expr-lua-bWVtbzEyMw/output/output.csv
	// INFO: Job expr-lua-bWVtbzEyMw-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-bWVtbzEyMw has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group expr-lua-bWVtbzEyMw completion task...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: updateJobStatus: expr-lua-bWVtbzEyMw with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-bWVtbzEyMw
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: <nil>
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: <nil>
	// Read memoised memo123 errors: <nil>
	// Decode memoised memo123 errors: <nil>
	// Reading expected-expr-output.txt error: <nil>
	// Expected csv format ok: true
}

func setupForTest(exprId, scanId, quantId string, modIds, modVers []string) (*JobManager, string) {
	logLev := logger.LogInfo
	ts := []int64{}

	// Reasons for time stamps:
	// dataset local file cache time stamp
	// start time stamp
	// queue time stamp

	for c := 0; c < 20; c++ {
		ts = append(ts, int64(1668142579+c))
	}
	origWD, _, svcs := initJobManagerTest(&logLev, ts)

	svcs.Config.Jobs.NodeCountOverride = 4
	svcs.Log = &logger.StdOutLogger{}
	svcs.Log.SetLogLevel(logger.LogDebug) // LogInfo)

	jm, err := CreateJobManager(&svcs, 0, false, true)
	fmt.Printf("jm Create: %v\n", err)
	ctx := context.TODO()
	svcs.MongoDB.Drop(ctx)
	expressionrunner.SeedDBForExpressionTest(filepath.Join(origWD, "..", "test-files-db-seed"), scanId, quantId, exprId, modIds, modVers, svcs.MongoDB)

	return jm, origWD
}

func runExprJobTest(exprId, scanId, quantId string, modIds, modVers []string, exprSuffix, scanSuffix, quantSuffix string, printResultAtEnd bool) {
	jm, origWD := setupForTest(exprId, scanId, quantId, modIds, modVers)
	defer os.Chdir(origWD)
	svcs := jm.svcs

	status, err := jm.SubmitExpressionJob(scanId+scanSuffix, quantId+quantSuffix, exprId+exprSuffix, "", "memo123", nil, nil)
	var s protos.JobStatus_Status
	if status != nil {
		s = status.Status
	}
	fmt.Printf("SubmitExpressionJob: %v, %v\n", s, err)

	if err != nil {
		return
	}

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "",
		servicesMock.JobBucketForUnitTest, servicesMock.ConfigBucketForUnitTest, servicesMock.UsersBucketForUnitTest, servicesMock.DatasetsBucketForUnitTest,
		svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{status.JobItemId + "-node-0"})

	jm.RunCheckJobQueueForTest()
	// time.Sleep(10 * time.Second)
	// jm.RunCheckJobQueueForTest()

	if printResultAtEnd {
		printResults(false, svcs)
	}

	verifyMemoItems(svcs, origWD)
}

func verifyMemoItems(svcs *services.APIServices, origWD string) {
	// Verify that we memoised the stuff we expected
	memKeys := []string{
		"exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9",
		"exprcachev1_GeoAndDiff_3_5_3_geometry_048300551",
		"memo123",
	}

	for c, key := range memKeys {
		memItem := &protos.MemoisedItem{}
		err := expressionrunner.ReadOne(dbCollections.MemoisedItemsName, bson.M{"_id": key}, memItem, svcs.MongoDB)
		fmt.Printf("Read memoised %v errors: %v\n", key, err)

		// Verify the last one
		if c == len(memKeys)-1 {
			memResult := protos.MemDataQueryResult{}
			err := proto.Unmarshal(memItem.Data, &memResult)
			fmt.Printf("Decode memoised %v errors: %v\n", key, err)

			if memResult.ResultValues != nil {
				pmcLookup := map[uint32]float32{}
				for _, item := range memResult.ResultValues.Values {
					pmcLookup[item.Pmc] = item.Value
				}

				exprPath := filepath.Join(origWD, "test-files", "expected-expr-output.txt")
				fields, err := dataImportHelpers.ReadCSV(exprPath, 1, ',')
				fmt.Printf("Reading %v error: %v\n", filepath.Base(exprPath), err)
				if err == nil {
					for c, cols := range fields {
						if c == 0 {
							fmt.Printf("Expected csv format ok: %v\n", len(cols) != 2 || cols[0] != "\"PMC\"" || cols[1] != "\"value\"")
							continue
						}

						pmc, err := strconv.Atoi(cols[0])
						if err != nil {
							log.Fatalf("Expected expr result line %v invalid PMC: %v", c, err)
						}

						value, err := strconv.ParseFloat(cols[1], 32)
						if err != nil {
							log.Fatalf("Expected expr result line %v invalid value: %v", c, err)
						}

						expVal := pmcLookup[uint32(pmc)]
						if expVal != float32(value) {
							log.Fatalf("Expected expr line %v value %v doesn't match calculated value %v", c, expVal, value)
						}
					}
				}
			}
		}
	}
}
