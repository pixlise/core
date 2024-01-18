package quantification

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
)

var testROIs = []roiItemWithPMCs{
	{
		PMCs: []int{7, 15, 388},
		ROIItem: &protos.ROIItem{
			Id:     "roi1-id",
			ScanId: "roi1-scan",
			Name:   "roi1",
			// Description
			ScanEntryIndexesEncoded: []int32{1, 0, 2},
			// ImageName
			// PixelIndexesEncoded
			// MistROIItem
			// IsMIST
			// Tags
			// ModifiedUnixSec
			// DisplaySettings
			// Owner
		},
	},
	{
		PMCs: []int{7, 450},
		ROIItem: &protos.ROIItem{
			Id:     "roi2-id",
			ScanId: "roi2-scan",
			Name:   "roi2",
			// Description
			ScanEntryIndexesEncoded: []int32{0, 3},
			// ImageName
			// PixelIndexesEncoded
			// MistROIItem
			// IsMIST
			// Tags
			// ModifiedUnixSec
			// DisplaySettings
			// Owner
		},
	},
}

func Example_makeROIPMCListFileContents_Combined() {
	fmt.Println(makeROIPMCListFileContents(testROIs, "5x11dataset.bin", true, false, map[int32]bool{}))

	// Output:
	// 5x11dataset.bin
	// roi1-id:7|Normal|A,7|Normal|B,15|Normal|A,15|Normal|B,388|Normal|A,388|Normal|B
	// roi2-id:7|Normal|A,7|Normal|B,450|Normal|A,450|Normal|B
	//  <nil>
}

func Example_makeROIPMCListFileContents_Combined_Dwells() {
	fmt.Println(makeROIPMCListFileContents(testROIs, "5x11dataset.bin", true, true, map[int32]bool{15: true}))

	// Output:
	// 5x11dataset.bin
	// roi1-id:7|Normal|A,7|Normal|B,15|Normal|A,15|Normal|B,15|Dwell|A,15|Dwell|B,388|Normal|A,388|Normal|B
	// roi2-id:7|Normal|A,7|Normal|B,450|Normal|A,450|Normal|B
	//  <nil>
}

func Example_makeROIPMCListFileContents_AB() {
	fmt.Println(makeROIPMCListFileContents(testROIs, "5x11dataset.bin", false, false, map[int32]bool{}))

	// Output:
	// 5x11dataset.bin
	// roi1-id:7|Normal|A,15|Normal|A,388|Normal|A
	// roi1-id:7|Normal|B,15|Normal|B,388|Normal|B
	// roi2-id:7|Normal|A,450|Normal|A
	// roi2-id:7|Normal|B,450|Normal|B
	//  <nil>
}

func Example_makeROIPMCListFileContents_AB_Dwells() {
	fmt.Println(makeROIPMCListFileContents(testROIs, "5x11dataset.bin", false, true, map[int32]bool{15: true}))

	// Output:
	// 5x11dataset.bin
	// roi1-id:7|Normal|A,15|Normal|A,15|Dwell|A,388|Normal|A
	// roi1-id:7|Normal|B,15|Normal|B,15|Dwell|B,388|Normal|B
	// roi2-id:7|Normal|A,450|Normal|A
	// roi2-id:7|Normal|B,450|Normal|B
	//  <nil>
}

func Example_processQuantROIsToPMCs_Combined_OK() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, filename, CaO_int, RTT
15, 5.1, Normal_A_roi1-id, 400, 7890
7, 6.1, Normal_B_roi2-id, 405, 7800
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", true, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// <nil>
	// The custom header
	// PMC, CaO_%, filename, CaO_int, RTT
	// 7, 5.1, Normal_A_roi1-id, 400, 7890
	// 15, 5.1, Normal_A_roi1-id, 400, 7890
	// 388, 5.1, Normal_A_roi1-id, 400, 7890
	// 7, 6.1, Normal_B_roi2-id, 405, 7800
	// 450, 6.1, Normal_B_roi2-id, 405, 7800
}

func Example_processQuantROIsToPMCs_SeparateAB_OK() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, filename, RTT
15, 5.1, 400, Normal_A_roi1-id, 7890
15, 5.2, 401, Normal_B_roi1-id, 7890
7, 6.1, 405, Normal_A_roi2-id, 7800
7, 6.2, 406, Normal_B_roi2-id, 7800
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", false, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// <nil>
	// The custom header
	// PMC, CaO_%, CaO_int, filename, RTT
	// 7, 5.1, 400, Normal_A_roi1-id, 7890
	// 7, 5.2, 401, Normal_B_roi1-id, 7890
	// 15, 5.1, 400, Normal_A_roi1-id, 7890
	// 15, 5.2, 401, Normal_B_roi1-id, 7890
	// 388, 5.1, 400, Normal_A_roi1-id, 7890
	// 388, 5.2, 401, Normal_B_roi1-id, 7890
	// 7, 6.1, 405, Normal_A_roi2-id, 7800
	// 7, 6.2, 406, Normal_B_roi2-id, 7800
	// 450, 6.1, 405, Normal_A_roi2-id, 7800
	// 450, 6.2, 406, Normal_B_roi2-id, 7800
}

func Example_processQuantROIsToPMCs_SeparateAB_InvalidFileName() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, filename, RTT
15, 5.1, 400, Normal_A_roi1-id, 7890
15, 5.2, 401, Normal_B, 7890
7, 6.1, 405, Normal_A, 7800
7, 6.2, 406, Normal_B, 7800
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", false, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// Invalid file name read: Normal_B from map CSV: JobData/abc123/output/node001.pmcs_result.csv, line 4
}

func Example_processQuantROIsToPMCs_Combined_NoFileNameCol() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
15, 5.1, 400, 7890
7, 6.1, 405, 7800
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", true, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// Map csv: JobData/abc123/output/node001.pmcs_result.csv, does not contain a filename column (used to match up ROIs)
}

func Example_processQuantROIsToPMCs_Combined_DownloadError() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", true, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// Failed to read map CSV: JobData/abc123/output/node001.pmcs_result.csv
}

func Example_processQuantROIsToPMCs_Combined_CSVRowCountROICountMismatch() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT, filename
15, 5.1, 400, 7890, Normal_A_roi1-id
7, 6.1, 405, 7800, Normal_A_roi1-id
12, 6.7, 407, 7700, Normal_A_roi1-id
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", true, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// PMC 12 in CSV: JobData/abc123/output/node001.pmcs_result.csv doesn't exist in ROI: roi1
}

func Example_processQuantROIsToPMCs_Combined_InvalidPMC() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const jobsBucket = "jobs-bucket"

	// Some of our files are empty, not there, have content
	// and they're meant to end up combined into one response...
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobsBucket), Key: aws.String("JobData/abc123/output/node001.pmcs_result.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: io.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, filename, RTT
15, 5.1, 400, Normal_A_roi1-id, 7890
Qwerty, 6.1, 405, Normal_A_roi1-id, 7800
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	outputCSV, err := processQuantROIsToPMCs(fs, jobsBucket, "JobData/abc123", "The custom header", "node001.pmcs", true, testROIs)

	fmt.Printf("%v\n", err)
	fmt.Println(outputCSV)

	// Output:
	// Failed to process map CSV: JobData/abc123/output/node001.pmcs_result.csv, invalid PMC Qwerty at line 4
}
