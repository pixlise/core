package jobmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	jobexecutor "github.com/pixlise/core/v4/api/job/executor"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func Example_jobmanager_SubmitQuantJob_Naltsos() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

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

	defer os.Chdir(origWD)
	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}

	bucketSimRoot := "./test-bucket-root"

	logLev := logger.LogInfo
	svcs := servicesMock.MakeMockSvcsWithFS(bucketSimRoot, &idGen, &logLev)
	svcs.MongoDB = wstestlib.GetDBWithEnvironment("jobtest")
	svcs.Config.JobRunnerDockerImage = "ghcr.io/pixlise/job-runner:latest"
	svcs.Config.QuantExecutor = jobexecutor.MakeLocalExecutor(bucketSimRoot)
	svcs.Config.CoresPerNode = 4
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{
			1668142579, // dataset local file cache time stamp
			1668142580, // start time stamp
		},
	}

	// Make sure the PIQUANT executable is one dir up
	fileaccess.CopyFileLocally("./test-bucket-root/Piquant", "./Piquant")
	err = os.Chmod("./Piquant", 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Clear stuff pre-test otherwise we get duplicate key errors
	ctx := context.TODO()
	svcs.MongoDB.Drop(ctx)

	jm, err := Create(&svcs)

	fmt.Printf("%v\n", err)

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
	var requestorUserSess *sessionuser.SessionUser

	err = jm.SubmitQuantJob(createParams, requestorUserSess)
	fmt.Printf("%v\n", err)

	// Output:
	// <nil>
	// INFO: Preparing job: quant-id123-node-0
	// Config: {"JobId":"quant-id123-node-0","RequiredFiles":[{"RemoteBucket":"datasets-bucket","RemotePath":"Scans/048300551/dataset.bin","LocalPath":"dataset.bin","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v7/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa","LocalPath":"Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v7/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv","LocalPath":"Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv","ApplyNodeIndex":0},{"RemoteBucket":"job-bucket","RemotePath":"JobData/048300551/quant-id123/node000000.pmcs","LocalPath":"node000000.pmcs","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/048300551/quant-id123/params.json","LocalPath":"params.json","ApplyNodeIndex":0}],"Command":"./Piquant","Args":["map","Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa","Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv","node000000.pmcs","Fe,Ca","map000000.csv","-Fe,1","-t,4"],"ArgIndexToApplyNodeIndexes":null,"OutputFiles":[{"RemoteBucket":"job-bucket","RemotePath":"JobData/048300551/quant-id123/piquant-logs/stdout000000.log","LocalPath":"stdout","ApplyNodeIndex":2},{"RemoteBucket":"job-bucket","RemotePath":"JobData/048300551/quant-id123/piquant-logs/piquant000000.log","LocalPath":"map000000.csv_log.txt","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/048300551/quant-id123/output/result000000.csv","LocalPath":"map000000.csv","ApplyNodeIndex":3}]}
	// INFO: Job config struct: job.JobConfig{JobId:"quant-id123-node-0", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/048300551/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v7/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v7/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", LocalPath:"Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/node000000.pmcs", LocalPath:"node000000.pmcs", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/params.json", LocalPath:"params.json", ApplyNodeIndex:0}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa", "Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv", "node000000.pmcs", "Fe,Ca", "map000000.csv", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/piquant-logs/stdout000000.log", LocalPath:"stdout", ApplyNodeIndex:2}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/piquant-logs/piquant000000.log", LocalPath:"map000000.csv_log.txt", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/048300551/quant-id123/output/result000000.csv", LocalPath:"map000000.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/048300551/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 843477 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v7/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa" -> "Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa
	// DEBUG:  Downloaded 4375 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v7/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv" -> "Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv
	// DEBUG:  Downloaded 7588 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/048300551/quant-id123/node000000.pmcs" -> "node000000.pmcs":
	// DEBUG:  Local path is <CWD>/node000000.pmcs
	// DEBUG:  Downloaded 64 bytes
	// DEBUG:  Wrote file: <CWD>/node000000.pmcs
	// DEBUG: Download "s3://job-bucket/JobData/048300551/quant-id123/params.json" -> "params.json":
	// DEBUG:  Local path is <CWD>/params.json
	// DEBUG:  Downloaded 2216 bytes
	// DEBUG:  Wrote file: <CWD>/params.json
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Rev2_Sept2021.msa,Calibration_PIXL_FM_SurfaceOps_5minECFs_Rev1_Jul2021.csv,node000000.pmcs,Fe,Ca,map000000.csv,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/048300551/quant-id123/piquant-logs/stdout000000.log
	// DEBUG: Upload map000000.csv_log.txt -> s3://job-bucket/JobData/048300551/quant-id123/piquant-logs/piquant000000.log
	// DEBUG: Upload map000000.csv -> s3://job-bucket/JobData/048300551/quant-id123/output/result000000.csv
	// <nil>
}

func Example_jobmanager_SubmitQuantJob_983561() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

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

	defer os.Chdir(origWD)
	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}

	bucketSimRoot := "./test-bucket-root"

	svcs := servicesMock.MakeMockSvcsWithFS(bucketSimRoot, &idGen, nil)
	svcs.MongoDB = wstestlib.GetDBWithEnvironment("jobtest")
	svcs.Config.JobRunnerDockerImage = "ghcr.io/pixlise/job-runner:latest"
	svcs.Config.QuantExecutor = jobexecutor.MakeLocalExecutor(bucketSimRoot)
	svcs.Config.CoresPerNode = 4
	svcs.Config.NodeCountOverride = 4
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{
			1668142579, // dataset local file cache time stamp
			1668142580, // start time stamp
		},
	}

	// Make sure the PIQUANT executable is one dir up
	fileaccess.CopyFileLocally("./test-bucket-root/Piquant", "./Piquant")
	err = os.Chmod("./Piquant", 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Clear stuff pre-test otherwise we get duplicate key errors
	ctx := context.TODO()
	svcs.MongoDB.Drop(ctx)

	jm, err := Create(&svcs)

	fmt.Printf("%v\n", err)

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
	var requestorUserSess *sessionuser.SessionUser

	err = jm.SubmitQuantJob(createParams, requestorUserSess)
	fmt.Printf("%v\n", err)

	// Output:
	// <nil>
	// INFO: Preparing job: quant-id123-node-0
	// Config: {"JobId":"quant-id123-node-0","RequiredFiles":[{"RemoteBucket":"datasets-bucket","RemotePath":"Scans/983561/dataset.bin","LocalPath":"dataset.bin","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","LocalPath":"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","LocalPath":"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","ApplyNodeIndex":0},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/node000000.pmcs","LocalPath":"node000000.pmcs","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/params.json","LocalPath":"params.json","ApplyNodeIndex":0}],"Command":"./Piquant","Args":["map","Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","node000000.pmcs","Ca,Ti","map000000.csv","-q,pPIETXCFsr","-b,0,12,60,910,2800,16","-Fe,1","-t,4"],"ArgIndexToApplyNodeIndexes":null,"OutputFiles":[{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/stdout000000.log","LocalPath":"stdout","ApplyNodeIndex":2},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/piquant000000.log","LocalPath":"map000000.csv_log.txt","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/output/result000000.csv","LocalPath":"map000000.csv","ApplyNodeIndex":3}]}
	// INFO: Job config struct: job.JobConfig{JobId:"quant-id123-node-0", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node000000.pmcs", LocalPath:"node000000.pmcs", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/params.json", LocalPath:"params.json", ApplyNodeIndex:0}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node000000.pmcs", "Ca,Ti", "map000000.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout000000.log", LocalPath:"stdout", ApplyNodeIndex:2}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant000000.log", LocalPath:"map000000.csv_log.txt", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result000000.csv", LocalPath:"map000000.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node000000.pmcs" -> "node000000.pmcs":
	// DEBUG:  Local path is <CWD>/node000000.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node000000.pmcs
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/params.json" -> "params.json":
	// DEBUG:  Local path is <CWD>/params.json
	// DEBUG:  Downloaded 2250 bytes
	// DEBUG:  Wrote file: <CWD>/params.json
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node000000.pmcs,Ca,Ti,map000000.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-0 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout000000.log
	// DEBUG: Upload map000000.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant000000.log
	// DEBUG: Upload map000000.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result000000.csv
	// INFO: Preparing job: quant-id123-node-1
	// Config: {"JobId":"quant-id123-node-1","RequiredFiles":[{"RemoteBucket":"datasets-bucket","RemotePath":"Scans/983561/dataset.bin","LocalPath":"dataset.bin","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","LocalPath":"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","LocalPath":"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","ApplyNodeIndex":0},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/node000001.pmcs","LocalPath":"node000001.pmcs","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/params.json","LocalPath":"params.json","ApplyNodeIndex":0}],"Command":"./Piquant","Args":["map","Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","node000001.pmcs","Ca,Ti","map000001.csv","-q,pPIETXCFsr","-b,0,12,60,910,2800,16","-Fe,1","-t,4"],"ArgIndexToApplyNodeIndexes":null,"OutputFiles":[{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/stdout000001.log","LocalPath":"stdout","ApplyNodeIndex":2},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/piquant000001.log","LocalPath":"map000001.csv_log.txt","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/output/result000001.csv","LocalPath":"map000001.csv","ApplyNodeIndex":3}]}
	// INFO: Job config struct: job.JobConfig{JobId:"quant-id123-node-1", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node000001.pmcs", LocalPath:"node000001.pmcs", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/params.json", LocalPath:"params.json", ApplyNodeIndex:0}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node000001.pmcs", "Ca,Ti", "map000001.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout000001.log", LocalPath:"stdout", ApplyNodeIndex:2}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant000001.log", LocalPath:"map000001.csv_log.txt", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result000001.csv", LocalPath:"map000001.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node000001.pmcs" -> "node000001.pmcs":
	// DEBUG:  Local path is <CWD>/node000001.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node000001.pmcs
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/params.json" -> "params.json":
	// DEBUG:  Local path is <CWD>/params.json
	// DEBUG:  Downloaded 2250 bytes
	// DEBUG:  Wrote file: <CWD>/params.json
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node000001.pmcs,Ca,Ti,map000001.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-1 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout000001.log
	// DEBUG: Upload map000001.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant000001.log
	// DEBUG: Upload map000001.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result000001.csv
	// INFO: Preparing job: quant-id123-node-2
	// Config: {"JobId":"quant-id123-node-2","RequiredFiles":[{"RemoteBucket":"datasets-bucket","RemotePath":"Scans/983561/dataset.bin","LocalPath":"dataset.bin","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","LocalPath":"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","LocalPath":"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","ApplyNodeIndex":0},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/node000002.pmcs","LocalPath":"node000002.pmcs","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/params.json","LocalPath":"params.json","ApplyNodeIndex":0}],"Command":"./Piquant","Args":["map","Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","node000002.pmcs","Ca,Ti","map000002.csv","-q,pPIETXCFsr","-b,0,12,60,910,2800,16","-Fe,1","-t,4"],"ArgIndexToApplyNodeIndexes":null,"OutputFiles":[{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/stdout000002.log","LocalPath":"stdout","ApplyNodeIndex":2},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/piquant000002.log","LocalPath":"map000002.csv_log.txt","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/output/result000002.csv","LocalPath":"map000002.csv","ApplyNodeIndex":3}]}
	// INFO: Job config struct: job.JobConfig{JobId:"quant-id123-node-2", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node000002.pmcs", LocalPath:"node000002.pmcs", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/params.json", LocalPath:"params.json", ApplyNodeIndex:0}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node000002.pmcs", "Ca,Ti", "map000002.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout000002.log", LocalPath:"stdout", ApplyNodeIndex:2}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant000002.log", LocalPath:"map000002.csv_log.txt", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result000002.csv", LocalPath:"map000002.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node000002.pmcs" -> "node000002.pmcs":
	// DEBUG:  Local path is <CWD>/node000002.pmcs
	// DEBUG:  Downloaded 60 bytes
	// DEBUG:  Wrote file: <CWD>/node000002.pmcs
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/params.json" -> "params.json":
	// DEBUG:  Local path is <CWD>/params.json
	// DEBUG:  Downloaded 2250 bytes
	// DEBUG:  Wrote file: <CWD>/params.json
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node000002.pmcs,Ca,Ti,map000002.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-2 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout000002.log
	// DEBUG: Upload map000002.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant000002.log
	// DEBUG: Upload map000002.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result000002.csv
	// INFO: Preparing job: quant-id123-node-3
	// Config: {"JobId":"quant-id123-node-3","RequiredFiles":[{"RemoteBucket":"datasets-bucket","RemotePath":"Scans/983561/dataset.bin","LocalPath":"dataset.bin","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","LocalPath":"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","ApplyNodeIndex":0},{"RemoteBucket":"config-bucket","RemotePath":"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","LocalPath":"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","ApplyNodeIndex":0},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/node000003.pmcs","LocalPath":"node000003.pmcs","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/params.json","LocalPath":"params.json","ApplyNodeIndex":0}],"Command":"./Piquant","Args":["map","Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa","Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv","node000003.pmcs","Ca,Ti","map000003.csv","-q,pPIETXCFsr","-b,0,12,60,910,2800,16","-Fe,1","-t,4"],"ArgIndexToApplyNodeIndexes":null,"OutputFiles":[{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/stdout000003.log","LocalPath":"stdout","ApplyNodeIndex":2},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/piquant-logs/piquant000003.log","LocalPath":"map000003.csv_log.txt","ApplyNodeIndex":3},{"RemoteBucket":"job-bucket","RemotePath":"JobData/983561/quant-id123/output/result000003.csv","LocalPath":"map000003.csv","ApplyNodeIndex":3}]}
	// INFO: Job config struct: job.JobConfig{JobId:"quant-id123-node-3", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"datasets-bucket", RemotePath:"Scans/983561/dataset.bin", LocalPath:"dataset.bin", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", LocalPath:"Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"config-bucket", RemotePath:"DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", LocalPath:"Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", ApplyNodeIndex:0}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/node000003.pmcs", LocalPath:"node000003.pmcs", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/params.json", LocalPath:"params.json", ApplyNodeIndex:0}}, Command:"./Piquant", Args:[]string{"map", "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa", "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv", "node000003.pmcs", "Ca,Ti", "map000003.csv", "-q,pPIETXCFsr", "-b,0,12,60,910,2800,16", "-Fe,1", "-t,4"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/stdout000003.log", LocalPath:"stdout", ApplyNodeIndex:2}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/piquant-logs/piquant000003.log", LocalPath:"map000003.csv_log.txt", ApplyNodeIndex:3}, job.JobFilePath{RemoteBucket:"job-bucket", RemotePath:"JobData/983561/quant-id123/output/result000003.csv", LocalPath:"map000003.csv", ApplyNodeIndex:3}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://datasets-bucket/Scans/983561/dataset.bin" -> "dataset.bin":
	// DEBUG:  Local path is <CWD>/dataset.bin
	// DEBUG:  Downloaded 1851910 bytes
	// DEBUG:  Wrote file: <CWD>/dataset.bin
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa" -> "Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa":
	// DEBUG:  Local path is <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG:  Downloaded 4643 bytes
	// DEBUG:  Wrote file: <CWD>/Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa
	// DEBUG: Download "s3://config-bucket/DetectorConfig/PIXL/PiquantConfigs/v5/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv" -> "Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv":
	// DEBUG:  Local path is <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG:  Downloaded 7585 bytes
	// DEBUG:  Wrote file: <CWD>/Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/node000003.pmcs" -> "node000003.pmcs":
	// DEBUG:  Local path is <CWD>/node000003.pmcs
	// DEBUG:  Downloaded 36 bytes
	// DEBUG:  Wrote file: <CWD>/node000003.pmcs
	// DEBUG: Download "s3://job-bucket/JobData/983561/quant-id123/params.json" -> "params.json":
	// DEBUG:  Local path is <CWD>/params.json
	// DEBUG:  Downloaded 2250 bytes
	// DEBUG:  Wrote file: <CWD>/params.json
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "./Piquant", args: [map,Config_PIXL_FM_SurfaceOps_Optic8_Jun2021.msa,Calibration_PIXL_FM_ShelfBugFixed_5minECFs_Jun2021.csv,node000003.pmcs,Ca,Ti,map000003.csv,-q,pPIETXCFsr,-b,0,12,60,910,2800,16,-Fe,1,-t,4]
	// INFO: Job quant-id123-node-3 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://job-bucket/JobData/983561/quant-id123/piquant-logs/stdout000003.log
	// DEBUG: Upload map000003.csv_log.txt -> s3://job-bucket/JobData/983561/quant-id123/piquant-logs/piquant000003.log
	// DEBUG: Upload map000003.csv -> s3://job-bucket/JobData/983561/quant-id123/output/result000003.csv
	// <nil>
}
