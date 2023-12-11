package quantification

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
)

func Example_makeQuantJobPMCLists() {
	fmt.Println(makeQuantJobPMCLists([]int32{1, 2, 3, 4, 5, 6, 7, 8}, 3))
	fmt.Println(makeQuantJobPMCLists([]int32{1, 2, 3, 4, 5, 6, 7, 8, 9}, 3))
	fmt.Println(makeQuantJobPMCLists([]int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 3))

	// Output:
	// [[1 2 3] [4 5 6] [7 8]]
	// [[1 2 3] [4 5 6] [7 8 9]]
	// [[1 2 3] [4 5 6] [7 8 9] [10]]
}

func Example_makeIndividualPMCListFileContents_Combined() {
	fmt.Println(makeIndividualPMCListFileContents([]int32{15, 7, 388}, "5x11dataset.bin", true, false, map[int32]bool{}))

	// Output:
	// 5x11dataset.bin
	// 15|Normal|A,15|Normal|B
	// 7|Normal|A,7|Normal|B
	// 388|Normal|A,388|Normal|B
	//  <nil>
}

func Example_makeIndividualPMCListFileContents_Combined_Dwell() {
	fmt.Println(makeIndividualPMCListFileContents([]int32{15, 7, 388}, "5x11dataset.bin", true, true, map[int32]bool{15: true}))

	// Output:
	// 5x11dataset.bin
	// 15|Normal|A,15|Normal|B,15|Dwell|A,15|Dwell|B
	// 7|Normal|A,7|Normal|B
	// 388|Normal|A,388|Normal|B
	//  <nil>
}

func Example_makeIndividualPMCListFileContents_AB() {
	fmt.Println(makeIndividualPMCListFileContents([]int32{15, 7, 388}, "5x11dataset.bin", false, false, map[int32]bool{}))

	// Output:
	// 5x11dataset.bin
	// 15|Normal|A
	// 15|Normal|B
	// 7|Normal|A
	// 7|Normal|B
	// 388|Normal|A
	// 388|Normal|B
	//  <nil>
}

func Example_makeIndividualPMCListFileContents_AB_Dwell() {
	fmt.Println(makeIndividualPMCListFileContents([]int32{15, 7, 388}, "5x11dataset.bin", false, true, map[int32]bool{15: true}))

	// Output:
	// 5x11dataset.bin
	// 15|Normal|A,15|Dwell|A
	// 15|Normal|B,15|Dwell|B
	// 7|Normal|A
	// 7|Normal|B
	// 388|Normal|A
	// 388|Normal|B
	//  <nil>
}

func Example_combineQuantOutputs_OK() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node002.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node003.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
18, 7.1, 415, 7840
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row3
PMC, CaO_%, CaO_int, RTT
3, 1.1, 450, 7830
40, 8.1, 455, 7870
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	combinedCSV, err := combineQuantOutputs(fs, jobsBucket, "JobData/abc123", "The custom header", []string{"node001.pmcs", "node002.pmcs", "node003.pmcs"})

	fmt.Printf("%v\n", err)
	fmt.Println(combinedCSV)

	// Output:
	// <nil>
	// The custom header
	// PMC, CaO_%, CaO_int, RTT
	// 3, 1.1, 450, 7830
	// 12, 6.1, 405, 7800
	// 18, 7.1, 415, 7840
	// 30, 5.1, 400, 7890
	// 40, 8.1, 455, 7870
}

func Example_combineQuantOutputs_DuplicatePMC() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node002.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node003.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
18, 7.1, 415, 7840
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row3
PMC, CaO_%, CaO_int, RTT
3, 1.1, 450, 7830
30, 1.3, 451, 7833
40, 8.1, 455, 7870
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	combinedCSV, err := combineQuantOutputs(fs, jobsBucket, "JobData/abc123", "The custom header", []string{"node001.pmcs", "node002.pmcs", "node003.pmcs"})

	fmt.Printf("%v\n", err)
	fmt.Println(combinedCSV)

	// Output:
	// <nil>
	// The custom header
	// PMC, CaO_%, CaO_int, RTT
	// 3, 1.1, 450, 7830
	// 12, 6.1, 405, 7800
	// 18, 7.1, 415, 7840
	// 30, 5.1, 400, 7890
	// 30, 1.3, 451, 7833
	// 40, 8.1, 455, 7870
}

func Example_combineQuantOutputs_DownloadError() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node002.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		nil,
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	combinedCSV, err := combineQuantOutputs(fs, jobsBucket, "JobData/abc123", "The custom header", []string{"node001.pmcs", "node002.pmcs", "node003.pmcs"})

	fmt.Printf("%v\n", err)
	fmt.Println(combinedCSV)

	// Output:
	// Failed to combine map segment: JobData/abc123/output/node002.pmcs_result.csv
}

func Example_combineQuantOutputs_BadPMC() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node002.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
NaN, 7.1, 415, 7840
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	combinedCSV, err := combineQuantOutputs(fs, jobsBucket, "JobData/abc123", "The custom header", []string{"node001.pmcs", "node002.pmcs", "node003.pmcs"})

	fmt.Printf("%v\n", err)
	fmt.Println(combinedCSV)

	// Output:
	// Failed to combine map segment: JobData/abc123/output/node002.pmcs_result.csv, invalid PMC NaN at line 3
}

func Example_combineQuantOutputs_LastLineCutOff() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node002.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
31
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	combinedCSV, err := combineQuantOutputs(fs, jobsBucket, "JobData/abc123", "The custom header", []string{"node001.pmcs", "node002.pmcs", "node003.pmcs"})

	fmt.Printf("%v\n", err)
	fmt.Println(combinedCSV)

	// Output:
	// Failed to combine map segment: JobData/abc123/output/node002.pmcs_result.csv, no PMC at line 3
}
