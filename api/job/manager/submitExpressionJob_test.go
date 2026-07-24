package jobmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job/jobnode"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/proto"
)

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

	jm, err := CreateJobManager(&svcs, 0, false, false, true)
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
	// INFO: Instance the-test-instance starting job "expr-lua-id123-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9non-existant", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-id123/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 109720 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9non-existant,expressionId=u59sahioy18frfl9,memoKey=memo123]
	// ERROR: Expression runner could not fetch quant: mongo: no documents in result
	// ERROR: Job expr-lua-id123-node-0 failed: <string>:2221: PIXLISE-Lua Runtime error: Expression runner could not fetch quant: mongo: no documents in result
	// stack traceback:
	// 	[G]: in function 'exists'
	// 	<string>:2221: in function 'getElmtList'
	// 	<string>:2229: in main chunk
	// 	[G]: ?
	// ERROR: Failed to start job expr-lua-id123 (node 0): Job expr-lua-id123-node-0 failed: <string>:2221: PIXLISE-Lua Runtime error: Expression runner could not fetch quant: mongo: no documents in result
	// stack traceback:
	// 	[G]: in function 'exists'
	// 	<string>:2221: in function 'getElmtList'
	// 	<string>:2229: in main chunk
	// 	[G]: ?
	// INFO: Job expr-lua-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-id123 has 1 ran, 0 completed nodes of 1
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-id123
	// INFO: updateJobStatus: expr-lua-id123 with status ERROR, message: 1 nodes failed
	// INFO:   Marking job expr-lua-id123 as ERROR due to nodes not all completing
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
	// INFO: Instance the-test-instance starting job "expr-lua-id123-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9-badver", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-id123/source.lua" -> "source.lua":
	// DEBUG:  Local path is <CWD>/source.lua
	// DEBUG:  Downloaded 103831 bytes
	// DEBUG:  Wrote file: <CWD>/source.lua
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "lua-expression", args: [scanId=048300551,quantId=quant-ggy6zxhn23p7rlv9,expressionId=u59sahioy18frfl9-badver,memoKey=memo123]
	// ERROR: Job expr-lua-id123-node-0 failed: <string> line:977(column:12) near 'error':   parse error
	//
	// ERROR: Failed to start job expr-lua-id123 (node 0): Job expr-lua-id123-node-0 failed: <string> line:977(column:12) near 'error':   parse error
	//
	// INFO: Job expr-lua-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-id123 has 1 ran, 0 completed nodes of 1
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-id123
	// INFO: updateJobStatus: expr-lua-id123 with status ERROR, message: 1 nodes failed
	// INFO:   Marking job expr-lua-id123 as ERROR due to nodes not all completing
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
	// INFO: Instance the-test-instance starting job "expr-lua-id123-node-0"...
	// Running lua expression job locally!
	// INFO: Running job from s3://job-bucket/JobData/048300551/expr-lua-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"expr-lua-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/source.lua", LocalPath:"source.lua", ApplyNodeIndex:0}}, Command:"lua-expression", Args:[]string{"scanId=048300551", "quantId=quant-ggy6zxhn23p7rlv9", "expressionId=u59sahioy18frfl9", "memoKey=memo123"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/stdout.log", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/expr-lua-id123/output/output.csv", LocalPath:"output.csv", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://job-bucket/JobData/048300551/expr-lua-id123/source.lua" -> "source.lua":
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
	// INFO: Job expr-lua-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/expr-lua-id123/output/stdout.log
	// DEBUG: Upload output.csv -> s3://job-bucket/JobData/048300551/expr-lua-id123/output/output.csv
	// INFO: Job expr-lua-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group expr-lua-id123 has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group expr-lua-id123 completion task...
	// INFO: updateJobStatus: expr-lua-id123 with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: updateJobStatus: expr-lua-id123 with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group expr-lua-id123
	// DEBUG:   CheckJobQueue clearing job queue items for expr-lua-id123
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// QueryQ: <nil>
	// Queue items at end: 0
	// Query jobs: <nil>
	// Jobs at end: 1
	// Job[0] id: expr-lua-id123
	// Query status: <nil>
	// Job status at end: 1
	// JobStatus[0] id: expr-lua-id123, status: COMPLETE, msg: "Nodes ran: 1"
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_Al2O3_048300551_quant-ggy6zxhn23p7rlv9 errors: <nil>
	// Read memoised exprcachev1_GeoAndDiff_3_5_3_geometry_048300551 errors: <nil>
	// Read memoised memo123 errors: <nil>
	// Decode memoised memo123 errors: <nil>
	// Reading expected-expr-output.txt error: <nil>
	// Expected csv format ok: true
}

func runExprJobTest(exprId, scanId, quantId string, modIds, modVers []string, exprSuffix, scanSuffix, quantSuffix string, printResultAtEnd bool) {
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
	svcs.Log.SetLogLevel(logger.LogDebug) // LogInfo)

	jm, err := CreateJobManager(&svcs, 0, false, false, true)
	fmt.Printf("jm Create: %v\n", err)
	ctx := context.TODO()
	svcs.MongoDB.Drop(ctx)
	expressionrunner.SeedDBForExpressionTest(filepath.Join(origWD, "..", "test-files-db-seed"), scanId, quantId, exprId, modIds, modVers, svcs.MongoDB)

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
							fmt.Printf("Expected csv format ok: %v", len(cols) != 2 || cols[0] != "\"PMC\"" || cols[1] != "\"value\"")
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
