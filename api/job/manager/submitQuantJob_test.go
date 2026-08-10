package jobmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/job/jobnode"
	"github.com/pixlise/core/v4/api/notificationSender"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Returns the original working dir, which any caller must defer os.Chdir to otherwise subsequent tests will fail!
func initJobManagerTest(logLevel *logger.LogLevel, timestamps []int64) (string, string, services.APIServices) {
	idGen := idgen.MockIDGenerator{
		IDs: []string{"id123"},
	}

	// Move to a working directory!
	origWD, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	wd := filepath.Join(origWD, "test-workdir")
	err = os.RemoveAll(wd)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(wd, 0777)
	if err != nil {
		log.Fatal(err)
	}

	// Copy the bucket seed data to a working dir that represents our bucket(s)
	err = fileaccess.CopyDirectoryLocally("./test-files", filepath.Join(wd, "test-bucket-root"), true)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}

	bucketSimRoot := "./test-bucket-root"

	svcs := servicesMock.MakeMockSvcsWithFS(bucketSimRoot, &idGen, logLevel)
	svcs.MongoDB = wstestlib.GetDBWithEnvironment("jobtest")
	//svcs.Config.JobRunnerDockerImage = "ghcr.io/pixlise/job-runner:latest"
	svcs.Config.QuantExecutor = "local:" + bucketSimRoot //jobexecutor.MakeLocalExecutor(bucketSimRoot)
	svcs.Config.Jobs.CoresPerNode = 4
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{QueuedTimeStamps: timestamps}

	// Make sure the PIQUANT executable is one dir up
	fileaccess.CopyFileLocally("./test-bucket-root/Piquant", "./Piquant")
	err = os.Chmod("./Piquant", 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Clear stuff pre-test otherwise we get duplicate key errors
	ctx := context.TODO()
	svcs.MongoDB.Drop(ctx)

	svcs.Notifier = &notificationSender.MockNotificationSender{}

	return origWD, bucketSimRoot, svcs
}

func printResults(svcs services.APIServices) {
	// At this point, check that the expected stuff has indeed happened
	ctx := context.TODO()
	cursor, err := svcs.MongoDB.Collection(dbCollections.JobQueueName).Find(ctx, bson.M{}, options.Find())
	fmt.Printf("QueryQ: %v\n", err)

	if err == nil {
		// There queue should be empty
		qItems := []*protos.JobQueueItem{}
		err = cursor.All(context.TODO(), &qItems)
		if err != nil {
			fmt.Printf("QueryQ read: %v\n", err)
		}
		fmt.Printf("Queue items at end: %v\n", len(qItems))
	}

	cursor, err = svcs.MongoDB.Collection(dbCollections.JobsName).Find(ctx, bson.M{}, options.Find())
	fmt.Printf("Query jobs: %v\n", err)

	if err == nil {
		// There queue should be empty
		jobItems := []*jobconfig.JobGroupConfig{}
		err = cursor.All(context.TODO(), &jobItems)
		if err != nil {
			fmt.Printf("Query jobs read: %v\n", err)
		}
		fmt.Printf("Jobs at end: %v\n", len(jobItems))
		if len(jobItems) > 0 {
			fmt.Printf("Job[0] id: %v\n", jobItems[0].JobGroupId)
		}
	}

	cursor, err = svcs.MongoDB.Collection(dbCollections.JobStatusName).Find(ctx, bson.M{}, options.Find())
	fmt.Printf("Query status: %v\n", err)

	if err == nil {
		// There queue should be empty
		statusItems := []*protos.JobStatus{}
		err = cursor.All(context.TODO(), &statusItems)
		if err != nil {
			fmt.Printf("Query status read: %v\n", err)
		}
		fmt.Printf("Job status at end: %v\n", len(statusItems))
		if len(statusItems) > 0 {
			fmt.Printf("JobStatus[0] id: %v, status: %v, msg: \"%v\"\n", statusItems[0].JobId, statusItems[0].Status, statusItems[0].Message)
		}
	}

	cursor, err = svcs.MongoDB.Collection(dbCollections.QuantificationsName).Find(ctx, bson.M{}, options.Find())
	fmt.Printf("Quant: %v\n", err)

	if err == nil {
		// There queue should be empty
		quantItems := []*protos.QuantificationSummary{}
		err = cursor.All(context.TODO(), &quantItems)
		if err != nil {
			fmt.Printf("Quant read: %v\n", err)
		}
		fmt.Printf("Quants at end: %v\n", len(quantItems))
		if len(quantItems) > 0 {
			fmt.Printf("Quant[0] id: %v, status: %v, msg: \"%v\"\n", quantItems[0].Id, quantItems[0].Status.Status, quantItems[0].Status.Message)
		}
	}
}

func Example_jobmanager_SubmitQuantJob_Naltsos() {
	logLev := logger.LogInfo
	origWD, _, svcs := initJobManagerTest(&logLev, []int64{
		1668142579, // dataset local file cache time stamp
		1668142580, // start time stamp
		1668142581, // queue time stamp
		1668142582, // queue read time stamp
		1668142583, // queue read time stamp
		1668142584, // queue read time stamp
		1668142585, // queue read time stamp
		1668142586, // queue read time stamp
		1668142587, // queue read time stamp
	})
	defer os.Chdir(origWD)

	svcs.Log = &logger.StdOutLogger{}
	svcs.Log.SetLogLevel(logger.LogDebug)

	jm, err := CreateJobManager(&svcs, 0, false, false, true)
	fmt.Printf("jm Create: %v\n", err)

	createParams := &protos.QuantCreateParams{
		Command:        "map",
		Name:           "AutoQuant-PIXL(AB)",
		ScanId:         "048300551", // Naltsos
		Pmcs:           []int32{100, 101},
		Elements:       []string{"Fe", "Ca"},
		DetectorConfig: "PIXL/v7",
		Parameters:     "-Fe,1",
		RunTimeSec:     60,
		QuantMode:      "Combined",
		//RoiIDs []string: ,
		//IncludeDwells: ,
	}

	status, err := jm.SubmitQuantJob(createParams, nil, nil)
	fmt.Printf("SubmitQuantJob: %v, %v\n", status.Status, err)

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "", servicesMock.JobBucketForUnitTest, svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{"quant-id123-node-0"})

	jm.RunCheckJobQueueForTest()

	printResults(svcs)

	// Output:
	// jm Create: <nil>
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/048300551/dataset.bin
	// DEBUG: spectraPerNode: 2, PMCs per node: 2 for 2 spectra, nodes: 1
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitQuantJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "quant-id123-node-0"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/048300551/quant-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/048300551/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v7/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v7/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", LocalPath:"Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/node00001.pmcs", LocalPath:"node00001.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", "Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", "node00001.pmcs", "Fe,Ca", "map00001.csv", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/piquant-logs/stdout00001.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/piquant-logs/piquant00001.log", LocalPath:"map00001.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/output/result00001.csv", LocalPath:"map00001.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/048300551/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 843477 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/048300551/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 256 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v7/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa" -> "Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa
	// DEBUG:  Downloaded 4375 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v7/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv" -> "Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv
	// DEBUG:  Downloaded 7588 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/048300551/quant-id123/node00001.pmcs" -> "node00001.pmcs":
	// DEBUG:  Local path is <CWD>/node00001.pmcs
	// DEBUG:  Downloaded 64 bytes
	// DEBUG:  Wrote file: <CWD>/node00001.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa,Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv,node00001.pmcs,Fe,Ca,map00001.csv,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/quant-id123/piquant-logs/stdout00001.log
	// DEBUG: Upload map00001.csv_log.txt -> s3://job-bucket/JobData/048300551/quant-id123/piquant-logs/piquant00001.log
	// DEBUG: Upload map00001.csv -> s3://job-bucket/JobData/048300551/quant-id123/output/result00001.csv
	// INFO: Job quant-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group quant-id123 has 1 ran, 1 completed nodes of 1
	// DEBUG:   CheckJobQueue running job group quant-id123 completion task...
	// INFO: updateJobStatus: quant-id123 with status GATHERING_RESULTS, message: Combining CSVs from 1 nodes...
	// INFO: Data Types Saved:
	// INFO:   FeO-T_% as F
	// INFO:   CaO_% as F
	// INFO:   FeO-T_int as F
	// INFO:   CaO_int as F
	// INFO:   FeO-T_err as F
	// INFO:   CaO_err as F
	// INFO:   total_counts as I
	// INFO:   livetime as F
	// INFO:   chisq as F
	// INFO:   eVstart as F
	// INFO:   eV/ch as F
	// INFO:   res as I
	// INFO:   iter as I
	// INFO:   Events as I
	// INFO:   Triggers as I
	// INFO: Elements found: [FeO-T CaO]
	// ERROR: Failed to read auto-share info for quantification triggered by PIXLISEImport. Quant won't be shared
	// ERROR: Failed to read scan 048300551 for sending new quant notification
	// ==>SysNotifyQuantChanged(quant-id123)
	// INFO: updateJobStatus: quant-id123 with status COMPLETE, message: Nodes ran: 1
	// DEBUG:   CheckJobQueue completed job group quant-id123
	// DEBUG:   CheckJobQueue clearing job queue items for quant-id123
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// QueryQ: <nil>
	// Queue items at end: 0
	// Query jobs: <nil>
	// Jobs at end: 1
	// Job[0] id: quant-id123
	// Query status: <nil>
	// Job status at end: 1
	// JobStatus[0] id: quant-id123, status: COMPLETE, msg: "Nodes ran: 1"
	// Quant: <nil>
	// Quants at end: 1
	// Quant[0] id: quant-id123, status: COMPLETE, msg: "Nodes ran: 1"
}

func Example_jobmanager_SubmitQuantJob_983561() {
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

	createParams := &protos.QuantCreateParams{
		Command:        "map",
		Name:           "AutoQuant-PIXL(AB)",
		ScanId:         "983561",
		Pmcs:           []int32{68, 69, 70, 71, 72, 73, 74},
		Elements:       []string{"Ca", "Ti"},
		DetectorConfig: "PIXL/v5",
		Parameters:     "-q,pPIETXCFsr -b,0,12,60,910,2800,16 -Fe,1",
		RunTimeSec:     60,
		QuantMode:      "Combined",
		//RoiIDs []string: ,
		//IncludeDwells: ,
	}

	status, err := jm.SubmitQuantJob(createParams, nil, nil)
	fmt.Printf("SubmitQuantJob: %v, %v\n", status.Status, err)

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "", servicesMock.JobBucketForUnitTest, svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{"quant-id123-node-0", "quant-id123-node-1", "quant-id123-node-2", "quant-id123-node-3"})

	jm.RunCheckJobQueueForTest()

	printResults(svcs)

	// Output:
	// jm Create: <nil>
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/983561/dataset.bin
	// INFO: Using node count override: 4
	// DEBUG: spectraPerNode: 2, PMCs per node: 2 for 7 spectra, nodes: 4
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitQuantJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "quant-id123-node-0"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/983561/quant-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node00001.pmcs", LocalPath:"node00001.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node00001.pmcs", "Ca,Ti", "map00001.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout00001.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant00001.log", LocalPath:"map00001.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result00001.csv", LocalPath:"map00001.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 328 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node00001.pmcs" -> "node00001.pmcs":
	// DEBUG:  Local path is <CWD>/node00001.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node00001.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node00001.pmcs,Ca,Ti,map00001.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout00001.log
	// DEBUG: Upload map00001.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant00001.log
	// DEBUG: Upload map00001.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result00001.csv
	// INFO: Job quant-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// INFO: Instance the-test-instance starting job "quant-id123-node-1"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/983561/quant-id123 for node 1
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-1", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node00002.pmcs", LocalPath:"node00002.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node00002.pmcs", "Ca,Ti", "map00002.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout00002.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant00002.log", LocalPath:"map00002.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result00002.csv", LocalPath:"map00002.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 328 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node00002.pmcs" -> "node00002.pmcs":
	// DEBUG:  Local path is <CWD>/node00002.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node00002.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node00002.pmcs,Ca,Ti,map00002.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-1 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout00002.log
	// DEBUG: Upload map00002.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant00002.log
	// DEBUG: Upload map00002.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result00002.csv
	// INFO: Job quant-id123-node-1 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// INFO: Instance the-test-instance starting job "quant-id123-node-2"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/983561/quant-id123 for node 2
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-2", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node00003.pmcs", LocalPath:"node00003.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node00003.pmcs", "Ca,Ti", "map00003.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout00003.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant00003.log", LocalPath:"map00003.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result00003.csv", LocalPath:"map00003.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 328 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node00003.pmcs" -> "node00003.pmcs":
	// DEBUG:  Local path is <CWD>/node00003.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node00003.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node00003.pmcs,Ca,Ti,map00003.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-2 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout00003.log
	// DEBUG: Upload map00003.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant00003.log
	// DEBUG: Upload map00003.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result00003.csv
	// INFO: Job quant-id123-node-2 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// INFO: Instance the-test-instance starting job "quant-id123-node-3"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/983561/quant-id123 for node 3
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-3", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node00004.pmcs", LocalPath:"node00004.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node00004.pmcs", "Ca,Ti", "map00004.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout00004.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant00004.log", LocalPath:"map00004.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result00004.csv", LocalPath:"map00004.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 328 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node00004.pmcs" -> "node00004.pmcs":
	// DEBUG:  Local path is <CWD>/node00004.pmcs
	// DEBUG:  Downloaded 36 bytes
	// DEBUG:  Wrote file: <CWD>/node00004.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node00004.pmcs,Ca,Ti,map00004.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-3 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout00004.log
	// DEBUG: Upload map00004.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant00004.log
	// DEBUG: Upload map00004.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result00004.csv
	// INFO: Job quant-id123-node-3 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// DEBUG: CheckJobQueue found 1 job groups
	// DEBUG:   CheckJobQueue job group quant-id123 has 4 ran, 4 completed nodes of 4
	// DEBUG:   CheckJobQueue running job group quant-id123 completion task...
	// INFO: updateJobStatus: quant-id123 with status GATHERING_RESULTS, message: Combining CSVs from 4 nodes...
	// INFO: Data Types Saved:
	// INFO:   CaO_% as F
	// INFO:   TiO2_% as F
	// INFO:   CaO_int as F
	// INFO:   TiO2_int as F
	// INFO:   CaO_err as F
	// INFO:   TiO2_err as F
	// INFO:   total_counts as I
	// INFO:   chisq as F
	// INFO:   eVstart as F
	// INFO:   eV/ch as F
	// INFO: Elements found: [CaO TiO2]
	// ERROR: Failed to read auto-share info for quantification triggered by PIXLISEImport. Quant won't be shared
	// ERROR: Failed to read scan 983561 for sending new quant notification
	// ==>SysNotifyQuantChanged(quant-id123)
	// INFO: updateJobStatus: quant-id123 with status COMPLETE, message: Nodes ran: 4
	// DEBUG:   CheckJobQueue completed job group quant-id123
	// DEBUG:   CheckJobQueue clearing job queue items for quant-id123
	// DEBUG:   CheckJobQueue found 0 not-started jobs
	// QueryQ: <nil>
	// Queue items at end: 0
	// Query jobs: <nil>
	// Jobs at end: 1
	// Job[0] id: quant-id123
	// Query status: <nil>
	// Job status at end: 1
	// JobStatus[0] id: quant-id123, status: COMPLETE, msg: "Nodes ran: 4"
	// Quant: <nil>
	// Quants at end: 1
	// Quant[0] id: quant-id123, status: COMPLETE, msg: "Nodes ran: 4"
}

func Example_jobmanager_SubmitQuantJob_983561_FailJobNotFound() {
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
	})
	defer os.Chdir(origWD)

	svcs.Config.Jobs.NodeCountOverride = 4
	svcs.Log = &logger.StdOutLogger{}
	svcs.Log.SetLogLevel(logger.LogDebug)

	jm, err := CreateJobManager(&svcs, 0, false, false, true)
	fmt.Printf("jm Create: %v\n", err)

	createParams := &protos.QuantCreateParams{
		Command:        "map",
		Name:           "AutoQuant-PIXL(AB)",
		ScanId:         "983561",
		Pmcs:           []int32{68, 69, 70, 71, 72, 73, 74},
		Elements:       []string{"Ca", "Ti"},
		DetectorConfig: "PIXL/v5",
		Parameters:     "-q,pPIETXCFsr -b,0,12,60,910,2800,16 -Fe,1",
		RunTimeSec:     60,
		QuantMode:      "Combined",
		//RoiIDs []string: ,
		//IncludeDwells: ,
	}

	status, err := jm.SubmitQuantJob(createParams, nil, nil)
	fmt.Printf("SubmitQuantJob: %v, %v\n", status.Status, err)

	// Run the job node queue processing code
	jn := jobnode.CreateJobNode("pixlise-job", "", servicesMock.JobBucketForUnitTest, svcs.InstanceId, svcs.FS, svcs.MongoDB, svcs.Log, svcs.TimeStamper)
	jn.StartJobs([]string{"quant-id123-node-0", "id2"})

	printResults(svcs)

	// time.Sleep(3 * time.Second)
	// jm.RunCheckJobQueueForTest()

	// Output:
	// jm Create: <nil>
	// DEBUG: Downloading file: s3://datasets-bucket/Scans/983561/dataset.bin
	// INFO: Using node count override: 4
	// DEBUG: spectraPerNode: 2, PMCs per node: 2 for 7 spectra, nodes: 4
	// INFO: WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing
	// SubmitQuantJob: STARTING, <nil>
	// INFO: Instance the-test-instance starting job "quant-id123-node-0"...
	// WARNING: Running job locally, recommended for use for tests only!
	// INFO: Running job from s3://job-bucket/JobData/983561/quant-id123 for node 0
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"quant-id123-node-0", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/request.json", LocalPath:"request.json", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node00001.pmcs", LocalPath:"node00001.pmcs", ApplyNodeIndex:3}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node00001.pmcs", "Ca,Ti", "map00001.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout00001.log", LocalPath:"stdout", ApplyNodeIndex:2}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant00001.log", LocalPath:"map00001.csv_log.txt", ApplyNodeIndex:3}, jobconfig.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result00001.csv", LocalPath:"map00001.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/request.json" -> "request.json":
	// DEBUG:  Local path is <CWD>/request.json
	// DEBUG:  Downloaded 328 bytes
	// DEBUG:  Wrote file: <CWD>/request.json
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node00001.pmcs" -> "node00001.pmcs":
	// DEBUG:  Local path is <CWD>/node00001.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node00001.pmcs
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node00001.pmcs,Ca,Ti,map00001.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout00001.log
	// DEBUG: Upload map00001.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant00001.log
	// DEBUG: Upload map00001.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result00001.csv
	// INFO: Job quant-id123-node-0 run complete: ""
	// Output:
	// -----------------
	// No output saved from local job run
	// -----------------
	// ERROR: Instance the-test-instance failed to find job id2 in queue, skipped
	// QueryQ: <nil>
	// Queue items at end: 4
	// Query jobs: <nil>
	// Jobs at end: 1
	// Job[0] id: quant-id123
	// Query status: <nil>
	// Job status at end: 1
	// JobStatus[0] id: quant-id123, status: PREPARING_NODES, msg: "Preparing 4 nodes..."
	// Quant: <nil>
	// Quants at end: 0
}
