package job

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
)

func Example_RunJob_NoConfig() {
	os.Setenv("JOB_CONFIG", "")
	fmt.Printf("%v\n", RunJob(false))

	// Output:
	// JOB_CONFIG env var not set
}

func Example_RunJob_BadConfig() {
	os.Setenv("JOB_CONFIG", "{ \"Some\": \"unfinished JSON }")
	fmt.Printf("%v\n", RunJob(false))

	// Output:
	// Failed to parse env var JOB_CONFIG: unexpected end of JSON input
}

func Example_RunJob_NoCommand() {
	cfg := JobConfig{
		JobId: "Job001",
	}

	cfgJSON, err := json.Marshal(cfg)
	fmt.Printf("cfgErr: %v\n", err)

	os.Setenv("JOB_CONFIG", string(cfgJSON))
	fmt.Printf("Job: %v\n", RunJob(false))

	// Output:
	// cfgErr: <nil>
	// INFO: Preparing job: Job001
	// Config: {"JobId":"Job001","RequiredFiles":null,"Command":"","Args":null,"OutputFiles":null}
	// INFO: Job config struct: job.JobConfig{JobId:"Job001", RequiredFiles:[]job.JobFilePath(nil), Command:"", Args:[]string(nil), OutputFiles:[]job.JobFilePath(nil)}
	// Job: No command specified
}

func Example_RunJob_BadInputLocalPath() {
	cfg := JobConfig{
		JobId:         "Job001",
		RequiredFiles: []JobFilePath{{LocalPath: "", RemoteBucket: "test-piquant", RemotePath: "jobs/Job001/input.csv"}},
		Command:       "ls",
	}

	cfgJSON, err := json.Marshal(cfg)
	fmt.Printf("cfgErr: %v\n", err)

	os.Setenv("JOB_CONFIG", string(cfgJSON))
	fmt.Printf("Job: %v\n", RunJob(false))

	// Output:
	// cfgErr: <nil>
	// INFO: Preparing job: Job001
	// Config: {"JobId":"Job001","RequiredFiles":[{"RemoteBucket":"test-piquant","RemotePath":"jobs/Job001/input.csv","LocalPath":""}],"Command":"ls","Args":null,"OutputFiles":null}
	// INFO: Job config struct: job.JobConfig{JobId:"Job001", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"jobs/Job001/input.csv", LocalPath:""}}, Command:"ls", Args:[]string(nil), OutputFiles:[]job.JobFilePath(nil)}
	// INFO: AWS S3 setup...
	// INFO: Downloading files...
	// INFO: Download "s3://test-piquant/jobs/Job001/input.csv" -> "":
	// Job: No localPath specified
}

func Example_RunJob_BadInputRemotePath() {
	cfg := JobConfig{
		JobId:         "Job001",
		RequiredFiles: []JobFilePath{{LocalPath: "input.csv", RemoteBucket: "test-piquant", RemotePath: "jobs/Job001/input.csv"}},
		Command:       "ls",
	}

	cfgJSON, err := json.Marshal(cfg)
	fmt.Printf("cfgErr: %v\n", err)

	os.Setenv("JOB_CONFIG", string(cfgJSON))
	fmt.Printf("Job: %v\n", RunJob(false))

	// Output:
	// cfgErr: <nil>
	// INFO: Preparing job: Job001
	// Config: {"JobId":"Job001","RequiredFiles":[{"RemoteBucket":"test-piquant","RemotePath":"jobs/Job001/input.csv","LocalPath":"input.csv"}],"Command":"ls","Args":null,"OutputFiles":null}
	// INFO: Job config struct: job.JobConfig{JobId:"Job001", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"jobs/Job001/input.csv", LocalPath:"input.csv"}}, Command:"ls", Args:[]string(nil), OutputFiles:[]job.JobFilePath(nil)}
	// INFO: AWS S3 setup...
	// INFO: Downloading files...
	// INFO: Download "s3://test-piquant/jobs/Job001/input.csv" -> "input.csv":
	// INFO:  Local file will be written to working dir
	// Job: Failed to download s3://test-piquant/jobs/Job001/input.csv: Not found
}

func Example_RunJob_UploadNotThere() {
	cfg := JobConfig{
		JobId:   "Job001",
		Command: NoOpCommand,
		OutputFiles: []JobFilePath{
			{LocalPath: "nofile.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Output/file.csv"},
		},
	}

	cfgJSON, err := json.Marshal(cfg)
	fmt.Printf("cfgErr: %v\n", err)

	os.Setenv("JOB_CONFIG", string(cfgJSON))
	fmt.Printf("Job: %v\n", RunJob(false))

	// Output:
	// cfgErr: <nil>
	// INFO: Preparing job: Job001
	// Config: {"JobId":"Job001","RequiredFiles":null,"Command":"noop","Args":null,"OutputFiles":[{"RemoteBucket":"test-piquant","RemotePath":"RunnerTest/Output/file.csv","LocalPath":"nofile.txt"}]}
	// INFO: Job config struct: job.JobConfig{JobId:"Job001", RequiredFiles:[]job.JobFilePath(nil), Command:"noop", Args:[]string(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"RunnerTest/Output/file.csv", LocalPath:"nofile.txt"}}}
	// INFO: AWS S3 setup...
	// INFO: Downloading files...
	// DEBUG: exec.Command starting "noop", args: []
	// INFO: Job Job001 runtime was 0 sec
	// ERROR: Job Job001 did not generate expected output file: nofile.txt
	// Job: Job Job001 failed to generate/upload output files: nofile.txt
}

func Example_RunJob_DownloadUploadOK() {
	// Before we do anything, ensure the file is in S3
	sess, err := awsutil.GetSession()
	fmt.Printf("GetSession: %v\n", err)
	s3, err := awsutil.GetS3(sess)
	fmt.Printf("GetS3: %v\n", err)
	remoteFS := fileaccess.MakeS3Access(s3)

	fmt.Printf("Write S3 input.csv: %v\n", remoteFS.WriteObject("test-piquant", "RunnerTest/input.csv", []byte("hello")))
	fmt.Printf("Write S3 input2.csv: %v\n", remoteFS.WriteObject("test-piquant", "RunnerTest/input2.csv", []byte("hello2")))
	fmt.Printf("Write local data.txt: %v\n", os.WriteFile("data.txt", []byte("hello"), dirperm))

	cfg := JobConfig{
		JobId: "Job001",
		RequiredFiles: []JobFilePath{
			{LocalPath: "inputfile.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/input.csv"},
			{LocalPath: "second.csv", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/input2.csv"},
		},
		Command: NoOpCommand,
		OutputFiles: []JobFilePath{
			{LocalPath: "stdout", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Output/stdout"},
			{LocalPath: "data.txt", RemoteBucket: "test-piquant", RemotePath: "RunnerTest/Output/file.csv"},
		},
	}

	cfgJSON, err := json.Marshal(cfg)
	fmt.Printf("cfgErr: %v\n", err)

	os.Setenv("JOB_CONFIG", string(cfgJSON))
	fmt.Printf("Job: %v\n", RunJob(false))

	// Output:
	// GetSession: <nil>
	// GetS3: <nil>
	// Write S3 input.csv: <nil>
	// Write S3 input2.csv: <nil>
	// Write local data.txt: <nil>
	// cfgErr: <nil>
	// INFO: Preparing job: Job001
	// Config: {"JobId":"Job001","RequiredFiles":[{"RemoteBucket":"test-piquant","RemotePath":"RunnerTest/input.csv","LocalPath":"inputfile.csv"},{"RemoteBucket":"test-piquant","RemotePath":"RunnerTest/input2.csv","LocalPath":"second.csv"}],"Command":"noop","Args":null,"OutputFiles":[{"RemoteBucket":"test-piquant","RemotePath":"RunnerTest/Output/stdout","LocalPath":"stdout"},{"RemoteBucket":"test-piquant","RemotePath":"RunnerTest/Output/file.csv","LocalPath":"data.txt"}]}
	// INFO: Job config struct: job.JobConfig{JobId:"Job001", RequiredFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"RunnerTest/input.csv", LocalPath:"inputfile.csv"}, job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"RunnerTest/input2.csv", LocalPath:"second.csv"}}, Command:"noop", Args:[]string(nil), OutputFiles:[]job.JobFilePath{job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"RunnerTest/Output/stdout", LocalPath:"stdout"}, job.JobFilePath{RemoteBucket:"test-piquant", RemotePath:"RunnerTest/Output/file.csv", LocalPath:"data.txt"}}}
	// INFO: AWS S3 setup...
	// INFO: Downloading files...
	// INFO: Download "s3://test-piquant/RunnerTest/input.csv" -> "inputfile.csv":
	// INFO:  Local file will be written to working dir
	// INFO:  Downloaded 5 bytes
	// INFO:  Wrote file: inputfile.csv
	// INFO: Download "s3://test-piquant/RunnerTest/input2.csv" -> "second.csv":
	// INFO:  Local file will be written to working dir
	// INFO:  Downloaded 6 bytes
	// INFO:  Wrote file: second.csv
	// DEBUG: exec.Command starting "noop", args: []
	// INFO: Job Job001 runtime was 0 sec
	// INFO: Uploaded stdout log to: s3://test-piquant/RunnerTest/Output/stdout
	// INFO: Upload data.txt -> s3://test-piquant/RunnerTest/Output/file.csv
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
