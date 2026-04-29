package jobstarter

import (
	"fmt"
	"os"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func Example_jobstarter_Run_docker_Python() {
	nodeCfg := jobrunner.JobConfig{
		JobId:   "Job003",
		Command: "python",
		Args:    []string{"test-files/test.py", "input.csv"},
		RequiredFiles: []jobrunner.JobFilePath{
			{LocalPath: "test-files/test.py", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/test.py"},
			{LocalPath: "test-files/input.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Input/py-input.csv"},
		},
		OutputFiles: []jobrunner.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Output/stdout"},
			{LocalPath: "py-output.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Py/Output/py-output.txt"},
		},
	}

	jobGroup := JobGroupConfig{
		JobGroupId:  "JG001",
		DockerImage: "pixlise-job-runner",
		NodeCount:   1,
		NodeConfig:  nodeCfg,
	}

	// cfgJSON, err := json.Marshal(cfg)
	// fmt.Printf("cfgErr: %v\n", err)

	r, err := GetJobStarter("docker")
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
	// Read test-files/input.csv: <nil>Write S3 s3://test-piquant/RunnerTest/Py/Input/py-input.csv: <nil>
	// StartJob: <nil>
	// ==========
	// stdout (<nil>):
	// ----------
	// Starting test.py
	// Contents of /root/test-files
	// input.csv
	// test.py
	// Writing output...
	// Finishing test.py
	//
	// ==========
	// py-output.txt (<nil>):
	// ----------
	// Example output from python
	// 1, 1.6
	// The end.
	// ==========
}

func Example_jobstarter_Run_docker_Lua() {
	nodeCfg := jobrunner.JobConfig{
		JobId:   "Job004",
		Command: "lua5.3",
		Args:    []string{"test-files/test.lua", "input.csv"},
		RequiredFiles: []jobrunner.JobFilePath{
			{LocalPath: "test-files/test.lua", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/test.lua"},
			{LocalPath: "test-files/input.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Input/lua-input.csv"},
		},
		OutputFiles: []jobrunner.JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Output/stdout"},
			{LocalPath: "lua-output.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Lua/Output/lua-output.txt"},
		},
	}

	jobGroup := JobGroupConfig{
		JobGroupId:  "JG001",
		DockerImage: "pixlise-job-runner",
		NodeCount:   1,
		NodeConfig:  nodeCfg,
	}

	// cfgJSON, err := json.Marshal(cfg)
	// fmt.Printf("cfgErr: %v\n", err)

	r, err := GetJobStarter("docker")
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
