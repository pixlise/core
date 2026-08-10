package jobrunner

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func Example_jobrunner_RunJob_BadConfigs() {
	fs := fileaccess.MakeFSAccessS3Simulator("./test-bucket-root")

	fmt.Printf("%v\n", RunJob("", "", 10, fs))
	fmt.Printf("%v\n", RunJob("bucket", "", 10, fs))
	fmt.Printf("%v\n", RunJob("bucket", "path/to/job", 1000000, fs))
	fmt.Printf("%v\n", RunJob("bucket", "path/to/job", 10, fs))

	// Output:
	// RunJob: bucket not set
	// RunJob: path not set
	// RunJob: nodeIndex too high
	// INFO: Running job from s3://bucket/path/to/job for node 10
	// Failed to read job config s3://bucket/path/to/job/params.json: open ./test-bucket-root/bucket/path/to/job/params.json: no such file or directory
}

func initTest() (string, fileaccess.FileAccess) {
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
	// err = fileaccess.CopyDirectoryLocally("./test-files", filepath.Join(wd, "test-bucket-root"), true)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}

	return origWD, fileaccess.MakeFSAccessS3Simulator("./test-bucket-root")
}

func writeConfig(cfg jobconfig.JobGroupConfig, to string) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(path.Dir(to), 0777)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(to, cfgJSON, 0777)
	if err != nil {
		log.Fatal(err)
	}
}

func Example_jobrunner_RunJob_NoCommand() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"-4", RequiredFiles:[]jobconfig.JobFilePath{}, Command:"", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{}}
	// Job: No command specified
}

func Example_jobrunner_RunJob_BadInputLocalPath() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			RequiredFiles: []jobconfig.JobFilePath{{LocalPath: "", RemoteBucket: "test-piquant", RemotePath: "jobs/Job001/input.csv"}},
			Command:       "ls",
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"-4", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"jobs/Job001/input.csv", LocalPath:"", ApplyNodeIndex:0}}, Command:"ls", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://test-piquant/jobs/Job001/input.csv" -> "":
	// Job: No localPath specified
}

func Example_jobrunner_RunJob_BadInputRemotePath() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			JobId:         "Job001-0",
			RequiredFiles: []jobconfig.JobFilePath{{LocalPath: "input.csv", RemoteBucket: "test-piquant", RemotePath: "jobs/Job001/input.csv"}},
			Command:       "ls",
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"Job001-0-4", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"jobs/Job001/input.csv", LocalPath:"input.csv", ApplyNodeIndex:0}}, Command:"ls", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://test-piquant/jobs/Job001/input.csv" -> "input.csv":
	// DEBUG:  Local path is <CWD>/input.csv
	// Job: Failed to download s3://test-piquant/jobs/Job001/input.csv: Not found
}

func Example_jobrunner_RunJob_BadCommand() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			JobId:   "Job001-0",
			Command: "dostuff",
			OutputFiles: []jobconfig.JobFilePath{
				{LocalPath: "nofile.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_UploadNotThere/Output/file.csv"},
			},
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"Job001-0-4", RequiredFiles:[]jobconfig.JobFilePath{}, Command:"dostuff", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_UploadNotThere/Output/file.csv", LocalPath:"nofile.txt", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "dostuff", args: []
	// ERROR: Job Job001-0-4 failed: exec: "dostuff": executable file not found in $PATH
	// Job: Job Job001-0-4 failed: exec: "dostuff": executable file not found in $PATH
}

func Example_jobrunner_RunJob_UploadNotThere() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			JobId:   "Job001-0",
			Command: "ls",
			OutputFiles: []jobconfig.JobFilePath{
				{LocalPath: "nofile.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_UploadNotThere/Output/file.csv"},
			},
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"Job001-0-4", RequiredFiles:[]jobconfig.JobFilePath{}, Command:"ls", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_UploadNotThere/Output/file.csv", LocalPath:"nofile.txt", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "ls", args: []
	// INFO: Job Job001-0-4 runtime was < 10 sec
	// ERROR: Job Job001-0-4 did not generate expected output file: nofile.txt
	// Job: Job Job001-0-4 failed to generate/upload output files: nofile.txt
}

func Example_jobrunner_RunJob_DownloadUploadOK() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			JobId: "Job001-0",
			RequiredFiles: []jobconfig.JobFilePath{
				{LocalPath: "inputfile.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_DownloadUploadOK/input.csv"},
				{LocalPath: "second.csv", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_DownloadUploadOK/input2.csv"},
			},
			Command: "ls",
			OutputFiles: []jobconfig.JobFilePath{
				{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_DownloadUploadOK/Output/stdout"},
				{LocalPath: "data.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_DownloadUploadOK/Output/file.csv"},
			},
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	// Before we do anything, ensure files are in our "bucket"
	fmt.Printf("Write S3 input.csv: %v\n", fs.WriteObject("test-piquant", "Example_jobrunner_RunJob_DownloadUploadOK/input.csv", []byte("hello")))
	fmt.Printf("Write S3 input2.csv: %v\n", fs.WriteObject("test-piquant", "Example_jobrunner_RunJob_DownloadUploadOK/input2.csv", []byte("hello2")))
	fmt.Printf("Write local data.txt: %v\n", os.WriteFile("data.txt", []byte("hello"), dirperm))

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// Write S3 input.csv: <nil>
	// Write S3 input2.csv: <nil>
	// Write local data.txt: <nil>
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"Job001-0-4", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_DownloadUploadOK/input.csv", LocalPath:"inputfile.csv", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_DownloadUploadOK/input2.csv", LocalPath:"second.csv", ApplyNodeIndex:0}}, Command:"ls", Args:[]string{}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_DownloadUploadOK/Output/stdout", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_DownloadUploadOK/Output/file.csv", LocalPath:"data.txt", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_DownloadUploadOK/input.csv" -> "inputfile.csv":
	// DEBUG:  Local path is <CWD>/inputfile.csv
	// DEBUG:  Downloaded 5 bytes
	// DEBUG:  Wrote file: <CWD>/inputfile.csv
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_DownloadUploadOK/input2.csv" -> "second.csv":
	// DEBUG:  Local path is <CWD>/second.csv
	// DEBUG:  Downloaded 6 bytes
	// DEBUG:  Wrote file: <CWD>/second.csv
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "ls", args: []
	// INFO: Job Job001-0-4 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://test-piquant/Example_jobrunner_RunJob_DownloadUploadOK/Output/stdout
	// DEBUG: Upload data.txt -> s3://test-piquant/Example_jobrunner_RunJob_DownloadUploadOK/Output/file.csv
	// Job: <nil>
}

func Example_jobrunner_RunJob_SeedAndDownloadOK() {
	origWD, fs := initTest()
	defer os.Chdir(origWD)

	writeConfig(jobconfig.JobGroupConfig{
		JobGroupId: "Job001",
		NodeCount:  2,
		NodeConfig: jobconfig.JobConfig{
			JobId: "Job001-0",
			RequiredFiles: []jobconfig.JobFilePath{
				{LocalPath: "test.lua", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/test.lua"},
				{LocalPath: "test.py", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/test.py"},
				{LocalPath: "requirements.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/requirements.txt"},
				{LocalPath: "lua-requirements.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/lua-requirements.txt"},
			},
			Command: "cp",
			Args:    []string{"requirements.txt", "data.txt"},
			OutputFiles: []jobconfig.JobFilePath{
				{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/Output/stdout"},
				{LocalPath: "data.txt", RemoteBucket: "test-piquant", RemotePath: "Example_jobrunner_RunJob_SeedAndDownloadOK/Output/file.csv"},
			},
		},
	}, "./test-bucket-root/job-bucket/path/to/job001/params.json")

	// Before we do anything, ensure the file is in S3
	l := &logger.StdOutLoggerForTest{}
	err := fileaccess.CopyToBucket(fs, filepath.Join(origWD, "test-files"), "test-piquant", "Example_jobrunner_RunJob_SeedAndDownloadOK", false, l)
	fmt.Printf("CopyToBucket: %v\n", err)

	fmt.Printf("Job: %v\n", RunJob("job-bucket", "path/to/job001", 4, fs))

	// Output:
	// CopyToBucket: <nil>
	// INFO: Running job from s3://job-bucket/path/to/job001 for node 4
	// DEBUG: Job config struct: jobconfig.JobConfig{JobId:"Job001-0-4", RequiredFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/test.lua", LocalPath:"test.lua", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/test.py", LocalPath:"test.py", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/requirements.txt", LocalPath:"requirements.txt", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/lua-requirements.txt", LocalPath:"lua-requirements.txt", ApplyNodeIndex:0}}, Command:"cp", Args:[]string{"requirements.txt", "data.txt"}, ArgIndexToApplyNodeIndexes:[]int(nil), OutputFiles:[]jobconfig.JobFilePath{jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/Output/stdout", LocalPath:"stdout", ApplyNodeIndex:0}, jobconfig.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"Example_jobrunner_RunJob_SeedAndDownloadOK/Output/file.csv", LocalPath:"data.txt", ApplyNodeIndex:0}}}
	// INFO: Downloading files...
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/test.lua" -> "test.lua":
	// DEBUG:  Local path is <CWD>/test.lua
	// DEBUG:  Downloaded 920 bytes
	// DEBUG:  Wrote file: <CWD>/test.lua
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/test.py" -> "test.py":
	// DEBUG:  Local path is <CWD>/test.py
	// DEBUG:  Downloaded 553 bytes
	// DEBUG:  Wrote file: <CWD>/test.py
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/requirements.txt" -> "requirements.txt":
	// DEBUG:  Local path is <CWD>/requirements.txt
	// DEBUG:  Downloaded 5 bytes
	// DEBUG:  Wrote file: <CWD>/requirements.txt
	// DEBUG: Download "s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/lua-requirements.txt" -> "lua-requirements.txt":
	// DEBUG:  Local path is <CWD>/lua-requirements.txt
	// DEBUG:  Downloaded 3 bytes
	// DEBUG:  Wrote file: <CWD>/lua-requirements.txt
	// INFO: Checking for required libraries...
	// INFO: Running job...
	// DEBUG: exec.Command starting "cp", args: [requirements.txt,data.txt]
	// INFO: Job Job001-0-4 runtime was < 10 sec
	// DEBUG: Uploaded stdout log to: s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/Output/stdout
	// DEBUG: Upload data.txt -> s3://test-piquant/Example_jobrunner_RunJob_SeedAndDownloadOK/Output/file.csv
	// Job: <nil>
}

/*
func Example_getDatasetFileFromPMCList() {
	fmt.Println(getDatasetFileFromPMCList("../test/data/pixlise-datasets/list.pmcs"))

	// Output:
	// 5x11dataset.bin <nil>
}

func Example_prepConfigsForPiquant() {
	params := quantModel.PiquantParams{
		RunTimeEnv:     "test",
		JobID:          "job-123",
		JobsPath:       "Jobs",
		DatasetPath:    "Downloads/SOL-00001/Experiment-00002",
		DetectorConfig: "PiquantConfig/PIXL",
		Elements: []string{
			"Al",
			"Ti",
			"Ca",
			"Fe",
		},
		Parameters:        "-q,pPIETXCFsr -b,0,12,60,910,280,16 -t,6",
		DatasetsBucket:    "dataset-bucket",
		PiquantJobsBucket: "job-bucket",
		ConfigBucket:      "config-bucket",
		QuantName:         "config test",
		PMCListName:       "file1.pmcs",
	}

	var mockS3 awsutil.MockS3Client
	l := &logger.StdOutLogger{}

	// Listing returns 1 item, get status returns error, check that it still requests 2nd item, 2nd item will fail to parse
	// but the func should still upload a blank jobs.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(params.ConfigBucket), Prefix: aws.String(params.DetectorConfig),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("PiquantConfig/PIXL/AP04_LVCMFM01_Teflon_1800s_011919_0602_28kV_20uA_10C_Efficiency.txt")},
				{Key: aws.String("PiquantConfig/PIXL/Config_PIXL_FM_ElemCal_CMH_May2019.msa")},
				{Key: aws.String("PiquantConfig/PIXL/FM_3_glasses_ECF_06_19_2019.txt")},
				{Key: aws.String("PiquantConfig/PIXL/FM_Efficiency_profile_Teflon_05_26_2019.txt")},
				{Key: aws.String("PiquantConfig/PIXL/config.json")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(params.ConfigBucket), Key: aws.String("PiquantConfig/PIXL/AP04_LVCMFM01_Teflon_1800s_011919_0602_28kV_20uA_10C_Efficiency.txt"),
		},
		{
			Bucket: aws.String(params.ConfigBucket), Key: aws.String("PiquantConfig/PIXL/Config_PIXL_FM_ElemCal_CMH_May2019.msa"),
		},
		{
			Bucket: aws.String(params.ConfigBucket), Key: aws.String("PiquantConfig/PIXL/FM_3_glasses_ECF_06_19_2019.txt"),
		},
		{
			Bucket: aws.String(params.ConfigBucket), Key: aws.String("PiquantConfig/PIXL/FM_Efficiency_profile_Teflon_05_26_2019.txt"),
		},
		{
			Bucket: aws.String(params.ConfigBucket), Key: aws.String("PiquantConfig/PIXL/config.json"),
		},
		{
			Bucket: aws.String(params.PiquantJobsBucket), Key: aws.String("Jobs/job-123/file1.pmcs"),
		},
		{
			Bucket: aws.String(params.DatasetsBucket), Key: aws.String("Downloads/SOL-00001/Experiment-00002/TheDataset.bin"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Contents of AP04_LVCMFM01_Teflon_1800s_011919_0602_28kV_20uA_10C_Efficiency.txt`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Contents of Config_PIXL_FM_ElemCal_CMH_May2019.msa`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Contents of FM_3_glasses_ECF_06_19_2019.txt`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Contents of FM_Efficiency_profile_Teflon_05_26_2019.txt`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"description": "PIXL configuration",
	"config-file": "Config_PIXL_FM_ElemCal_CMH_May2019.msa",
	"optic-efficiency": "AP04_LVCMFM01_Teflon_1800s_011919_0602_28kV_20uA_10C_Efficiency.txt",
	"calibration-file": "FM_3_glasses_ECF_06_19_2019.txt",
	"standards-file": ""
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`TheDataset.bin
1
2
3
4`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Contents of TheDataset.bin`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	paths, err := prepConfigsForPiquant(params, l, &fs)
	fmt.Println(err)

	fmt.Println(filepath.Base(paths.config))
	fmt.Println(filepath.Base(paths.calibration))
	fmt.Println(filepath.Base(paths.dataset))
	fmt.Println(filepath.Base(paths.pmcList))

	// Check the optic file setup worked
	data, err := ioutil.ReadFile(paths.config)
	fmt.Println(err)
	datastr := string(data)
	fmt.Printf("%v\n", strings.Contains(datastr, fmt.Sprintf("\n##OPTICFILE : %v", filepath.Dir(paths.config))))

	fmt.Println(err)
	fmt.Println(mockS3.FinishTest())

	// Output:
	// <nil>
	// Config_PIXL_FM_ElemCal_CMH_May2019.msa
	// FM_3_glasses_ECF_06_19_2019.txt
	// TheDataset.bin
	// file1.pmcs
	// <nil>
	// true
	// <nil>
	// <nil>
}

func Example_saveOutputs() {
	const jobBucket = "job-bucket"
	const jobPath = "jobs/some-job-123"

	var mockS3 awsutil.MockS3Client
	l := &logger.StdOutLogger{}

	// Set up expected S3 calls
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String(path.Join(jobPath, "output", "files_001.pmcs_result.csv")), Body: bytes.NewReader([]byte(`csv file contents`)),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String(path.Join(jobPath, "piquant-logs", "files_001.pmcs_piquant.log")), Body: bytes.NewReader([]byte(`log file contents`)),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String(path.Join(jobPath, "piquant-logs", "files_001.pmcs_stdout.log")), Body: bytes.NewReader([]byte(`piquant stdout`)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
	}

	// Create some "output" files
	tmpPath := os.TempDir()
	csvFile, err := ioutil.TempFile(tmpPath, "output.csv")
	fmt.Printf("%v\n", err)
	_, err = csvFile.WriteString("csv file contents")
	fmt.Printf("%v\n", err)
	csvFile.Close()

	logFile, err := ioutil.TempFile(tmpPath, "threads.log")
	fmt.Printf("%v\n", err)
	_, err = logFile.WriteString("log file contents")
	fmt.Printf("%v\n", err)
	logFile.Close()

	fs := fileaccess.MakeS3Access(&mockS3)
	saveOutputs(jobBucket, jobPath, "files_001.pmcs", csvFile.Name(), logFile.Name(), "piquant stdout", l, fs)

	fmt.Println(mockS3.FinishTest())

	// Cleanup
	os.RemoveAll(tmpPath)

	// Output:
	// <nil>
	// <nil>
	// <nil>
	// <nil>
	// <nil>
}

func Test_loadParams(t *testing.T) {
	// Set default QuantParams
	want := "new-pmc"
	var defaultParams quantModel.PiquantParams
	defaultParams.PMCListName = want
	paramsJson, _ := json.Marshal(defaultParams)
	os.Setenv("QUANT_PARAMS", string(paramsJson))

	// Test that default params are loaded
	t.Run("Default", func(t *testing.T) {
		if got := loadParams(); got.PMCListName != want {
			t.Errorf("loadParams.PMCListName = %v, want %v", got, want)
		}
	})

	// Override PMCListName via JOB_COMPLETION_INDEX env var
	os.Setenv("JOB_COMPLETION_INDEX", "41")
	want = "node00042.pmcs"
	t.Run("NodeIndexOverride", func(t *testing.T) {
		if got := loadParams(); got.PMCListName != want {
			t.Errorf("loadParams.PMCListName = %v, want %v", got, want)
		}
	})

	// Cleanup
	os.Unsetenv("JOB_COMPLETION_INDEX")
	os.Unsetenv("QUANT_PARAMS")
}
*/
