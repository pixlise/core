package jobexecutor

import (
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

var dockerImage = "ghcr.io/pixlise/job-runner:latest"

// var dockerImage = "pixlise-job-runner"

func Example_jobexecutor_Run_docker_Python() {
	nodeCfg := job.JobConfig{
		JobId:   "Job003",
		Command: "python",
		Args:    []string{"test.py", "Input/input.csv"},
		RequiredFiles: []job.JobFilePath{
			{LocalPath: "test.py", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Python/test.py"},
			{LocalPath: "requirements.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Python/requirements.txt"},
			{LocalPath: "Input/input.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Python/Input/input.csv"},
		},
		OutputFiles: []job.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Python/Output/stdout"},
			{LocalPath: "py-output.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Python/Output/py-output.txt"},
		},
	}

	jobGroup := JobGroupConfig{
		JobGroupId:  "JG001",
		DockerImage: dockerImage,
		NodeCount:   1,
		NodeConfig:  nodeCfg,
	}

	r, err := GetJobExecutor("docker")
	fmt.Printf("GetJobStarter: %v\n", err)

	l := &logger.StdOutLoggerForTest{}

	apiCfg := config.APIConfig{}

	sess, err := awsutil.GetSession()
	fmt.Printf("GetSession: %v\n", err)
	s3, err := awsutil.GetS3(sess)
	fmt.Printf("GetS3: %v\n", err)
	remoteFS := fileaccess.MakeS3Access(s3)

	err = fileaccess.CopyToBucket(remoteFS, "test-files/python", "test-piquant", "Example_jobexecutor_Run_docker_Python", true, l)
	fmt.Printf("CopyToBucket: %v\n", err)

	err = r.StartJob(jobGroup, apiCfg, specialUserIds.PIXLISESystemUserId, l)
	fmt.Printf("StartJob: %v\n", err)

	// When it's done, we download the output file and print it
	fmt.Println("==========")
	d, err := remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Python/Output/stdout")
	fmt.Printf("stdout (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	d, err = remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Python/Output/py-output.txt")
	fmt.Printf("py-output.txt (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	// Output:
	// GetJobStarter: <nil>
	// GetSession: <nil>
	// GetS3: <nil>
	// CopyToBucket: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Starting test.py
	// Contents of /run/Input
	// input.csv
	// Writing output...
	// Finishing test.py
	//
	// ==========
	// py-output.txt (<nil>):
	// ----------
	// Example output from python
	// 2,  7.3
	// The end.
	// ==========
}

func Example_jobexecutor_Run_docker_Lua() {
	nodeCfg := job.JobConfig{
		JobId:   "Job004",
		Command: "lua5.3",
		Args:    []string{"test.lua", "input.csv"},
		RequiredFiles: []job.JobFilePath{
			{LocalPath: "test.lua", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Lua/test.lua"},
			{LocalPath: "lua-requirements.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Lua/lua-requirements.txt"},
			{LocalPath: "Input/input.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Lua/Input/input.csv"},
		},
		OutputFiles: []job.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Lua/Output/stdout"},
			{LocalPath: "lua-output.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Lua/Output/lua-output.txt"},
		},
	}

	jobGroup := JobGroupConfig{
		JobGroupId:  "JG001",
		DockerImage: dockerImage,
		NodeCount:   1,
		NodeConfig:  nodeCfg,
	}

	// cfgJSON, err := json.Marshal(cfg)
	// fmt.Printf("cfgErr: %v\n", err)

	r, err := GetJobExecutor("docker")
	fmt.Printf("GetJobStarter: %v\n", err)

	l := &logger.StdOutLoggerForTest{}

	apiCfg := config.APIConfig{}

	sess, err := awsutil.GetSession()
	fmt.Printf("GetSession: %v\n", err)
	s3, err := awsutil.GetS3(sess)
	fmt.Printf("GetS3: %v\n", err)
	remoteFS := fileaccess.MakeS3Access(s3)

	err = fileaccess.CopyToBucket(remoteFS, "test-files/lua", "test-piquant", "Example_jobexecutor_Run_docker_Lua", true, l)
	fmt.Printf("CopyToBucket: %v\n", err)

	err = r.StartJob(jobGroup, apiCfg, specialUserIds.PIXLISESystemUserId, l)
	fmt.Printf("StartJob: %v\n", err)

	// When it's done, we download the output file and print it
	fmt.Println("==========")
	d, err := remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Lua/Output/stdout")
	fmt.Printf("stdout (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	d, err = remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Lua/Output/lua-output.txt")
	fmt.Printf("lua-output.txt (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	// Output:
	// GetJobStarter: <nil>
	// GetSession: <nil>
	// GetS3: <nil>
	// CopyToBucket: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Starting test.lua
	// Contents of /run/Input
	// input.csv
	// Writing output...
	// Finishing test.lua
	//
	// ==========
	// lua-output.txt (<nil>):
	// ----------
	// Example output from lua
	// 1, 1.6
	// The end.
	// ==========
}

func Example_jobexecutor_Run_docker_Piquant() {
	nodeCfg := job.JobConfig{
		JobId:   "Job005",
		Command: "./Piquant",
		Args:    []string{"quant", "Config_PIXL_FM_SurfaceOps_Rev1_Jul2021.msa", "Calibrate_Master_ECF_new_BB_01_08_2020.csv", "BulkA.msa", "Fe_K Ca_K Ti_K", "quant.csv"},
		RequiredFiles: []job.JobFilePath{
			{LocalPath: "BulkA.msa", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Piquant/Input/BulkA.msa"},
			{LocalPath: "Calibrate_Master_ECF_new_BB_01_08_2020.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Piquant/Input/Calibrate_Master_ECF_new_BB_01_08_2020.csv"},
			{LocalPath: "Config_PIXL_FM_SurfaceOps_Rev1_Jul2021.msa", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Piquant/Input/Config_PIXL_FM_SurfaceOps_Rev1_Jul2021.msa"},
		},
		OutputFiles: []job.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Piquant/Output/stdout"},
			{LocalPath: "quant.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobexecutor_Run_docker_Piquant/Output/quant.csv"},
		},
	}

	jobGroup := JobGroupConfig{
		JobGroupId:  "JG001",
		DockerImage: dockerImage,
		NodeCount:   1,
		NodeConfig:  nodeCfg,
	}

	r, err := GetJobExecutor("docker")
	fmt.Printf("GetJobStarter: %v\n", err)

	l := &logger.StdOutLoggerForTest{}

	apiCfg := config.APIConfig{}

	sess, err := awsutil.GetSession()
	fmt.Printf("GetSession: %v\n", err)
	s3, err := awsutil.GetS3(sess)
	fmt.Printf("GetS3: %v\n", err)
	remoteFS := fileaccess.MakeS3Access(s3)

	err = fileaccess.CopyToBucket(remoteFS, "test-files/piquant", "test-piquant", "Example_jobexecutor_Run_docker_Piquant", true, l)
	fmt.Printf("CopyToBucket: %v\n", err)

	err = r.StartJob(jobGroup, apiCfg, specialUserIds.PIXLISESystemUserId, l)
	fmt.Printf("StartJob: %v\n", err)

	// When it's done, we download the output file and print it
	fmt.Println("==========")
	d, err := remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Piquant/Output/stdout")
	fmt.Printf("stdout (%v):\n----------\n", err)
	if err == nil {
		fmt.Printf("Found element sum: %v\n", strings.Contains(string(d), "Element sum 28.73 %"))
	}
	fmt.Println("==========")

	d, err = remoteFS.ReadObject("test-piquant", "Example_jobexecutor_Run_docker_Piquant/Output/quant.csv")
	fmt.Printf("quant.csv (%v):\n----------\n", err)
	if err == nil {
		lines := strings.Split(string(d), "\n")
		fmt.Printf("%v\n%v\nline 45:\n%v\n", lines[0], lines[1], lines[45])
	}
	fmt.Println("==========")

	// Output:
	// GetJobStarter: <nil>
	// GetSession: <nil>
	// GetS3: <nil>
	// CopyToBucket: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Found element sum: true
	// ==========
	// quant.csv (<nil>):
	// ----------
	//    PIQUANT 3.2.17-master  BulkA.msa
	// Energy (keV), meas, calc, bkg, sigma, residual, DetCE, Fe_K, Ti_K, Ca_K, Rh_K_coh, Rh_L_coh, Rh_K_inc, Pileup, Rh_L_coh_Lb1
	// line 45:
	// 0.328962, 0, 1.50022e-08, 7.38965e-39, 1.41421, -1.50022e-08, 0, 7.38965e-39, 7.38965e-39, 7.38965e-39, 7.38965e-39, 1.50022e-08, 7.38965e-39, 7.38965e-39, 7.38965e-39
	// ==========
}
