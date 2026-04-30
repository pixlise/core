package jobexecutor

import (
	"fmt"
	"os"

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
		Args:    []string{"test-files/test.py", "input.csv"},
		RequiredFiles: []job.JobFilePath{
			{LocalPath: "test-files/test.py", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/test.py"},
			{LocalPath: "test-files/requirements.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/requirements.txt"},
			{LocalPath: "test-files/input.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Input/py-input.csv"},
		},
		OutputFiles: []job.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Output/stdout"},
			{LocalPath: "py-output.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Output/py-output.txt"},
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

	for _, f := range nodeCfg.RequiredFiles {
		d, e := os.ReadFile(f.LocalPath)
		fmt.Printf("Read %v: %v", f.LocalPath, e)
		if e == nil {
			fmt.Printf("Write S3 s3://%v/%v: %v\n", f.RemoteBucket, f.RemotePath, remoteFS.WriteObject(f.RemoteBucket, f.RemotePath, d))
		}
	}

	err = r.StartJob(jobGroup, apiCfg, specialUserIds.PIXLISESystemUserId, l)
	fmt.Printf("StartJob: %v\n", err)

	// When it's done, we download the output file and print it
	fmt.Println("==========")
	d, err := remoteFS.ReadObject("test-piquant", "RunnerTest/Py/Output/stdout")
	fmt.Printf("stdout (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	d, err = remoteFS.ReadObject("test-piquant", "RunnerTest/Py/Output/py-output.txt")
	fmt.Printf("py-output.txt (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	// Output:
	// GetJobStarter: <nil>
	// GetSession: <nil>
	// GetS3: <nil>
	// Read test-files/test.py: <nil>Write S3 s3://test-piquant/RunnerTest/Py/test.py: <nil>
	// Read test-files/requirements.txt: <nil>Write S3 s3://test-piquant/RunnerTest/Py/requirements.txt: <nil>
	// Read test-files/input.csv: <nil>Write S3 s3://test-piquant/RunnerTest/Py/Input/py-input.csv: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Starting test.py
	// Contents of /root/test-files
	// requirements.txt
	// test.py
	// input.csv
	// Writing output...
	// Finishing test.py
	//
	// ==========
	// py-output.txt (<nil>):
	// ----------
	// Example output from python
	// 1, 1.6
	// The end.
	// ===========
}

/*
func Example_jobexecutor_Run_docker_Lua() {
	nodeCfg := job.JobConfig{
		JobId:   "Job004",
		Command: "lua5.3",
		Args:    []string{"test-files/test.lua", "input.csv"},
		RequiredFiles: []job.JobFilePath{
			{LocalPath: "test-files/test.lua", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/test.lua"},
			{LocalPath: "test-files/input.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Input/lua-input.csv"},
		},
		OutputFiles: []job.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Output/stdout"},
			{LocalPath: "lua-output.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Output/lua-output.txt"},
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

	for _, f := range nodeCfg.RequiredFiles {
		d, e := os.ReadFile(f.LocalPath)
		fmt.Printf("Read %v: %v", f.LocalPath, e)
		if e == nil {
			fmt.Printf("Write S3 s3://%v/%v: %v\n", f.RemoteBucket, f.RemotePath, remoteFS.WriteObject(f.RemoteBucket, f.RemotePath, d))
		}
	}

	err = r.StartJob(jobGroup, apiCfg, specialUserIds.PIXLISESystemUserId, l)
	fmt.Printf("StartJob: %v\n", err)

	// When it's done, we download the output file and print it
	fmt.Println("==========")
	d, err := remoteFS.ReadObject("test-piquant", "RunnerTest/Lua/Output/stdout")
	fmt.Printf("stdout (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	d, err = remoteFS.ReadObject("test-piquant", "RunnerTest/Lua/Output/lua-output.txt")
	fmt.Printf("lua-output.txt (%v):\n----------\n", err)
	if err == nil {
		fmt.Println(string(d))
	}
	fmt.Println("==========")

	// Output:
	// GetJobStarter: <nil>
	// GetSession: <nil>
	// GetS3: <nil>
	// Read test-files/test.lua: <nil>Write S3 s3://test-piquant/RunnerTest/Lua/test.lua: <nil>
	// Read test-files/input.csv: <nil>Write S3 s3://test-piquant/RunnerTest/Lua/Input/lua-input.csv: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Starting test.lua
	// Contents of /root/test-files
	// input.csv
	// test.lua
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
*/
