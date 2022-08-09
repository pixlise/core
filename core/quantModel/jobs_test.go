// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package quantModel

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/roiModel"
)

/*
5x11, 4035 PMCs, 9 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:23 (with params)
	=> 4035*2=8070 spectra on 160 cores in 203 sec => 50.44 spectra/core in 203 sec => 4.02sec/spectra
5x11, 4035 PMCs, 6 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:47 (with params)
	=> 4035*2=8070 spectra on 160 cores in 167 sec => 50.44 spectra/core in 167 sec => 3.31sec/spectra
5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:52 (with params)
	=> 4035*2=8070 spectra on 160 cores in 112 sec => 50.44 spectra/core in 112 sec => 2.22sec/spectra

5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 10 nodes. Runtime: 3:32 (with params)
	=> 4035*2=8070 spectra on 80 cores in 212 sec => 100.88 spectra/core in 212 sec => 2.10 sec/spectra

3 elements => 2.10sec/spectra
3 elements => 2.22sec/spectra
  3 elems jumped 1.09sec
6 elements => 3.31sec/spectra
  3 elems jumped 0.71sec
9 elements => 4.02sec/spectra

Assumptions:
- Lets make this calcatable: 9elem=4sec/spectra, 3elem = 2sec/spectra, linearly interpolate in this range
- Works out to elements = 3*sec - 3
- To calculate node count, we are given Core count, Runtime desired, Spectra count, Element count
- Using the above:
  Runtime = Spectra*SpectraRuntime / (Core*Nodes)
  Nodes = Spectra*SpectraRuntime / (Runtime * Core)

  SpectraRuntime is calculated using the above formula:
  Elements = 3 * Sec - 3
  SpectraRuntime = (Elements+3) / 3

  Nodes = Spectra*((Elements + 3) / 3) / (RuntimeDesired * Cores)
  Nodes = Spectra*(Elements+3) / 3*(RuntimeDesired * Cores)

  Example using the values from above:
  Nodes = 8070*(3+3)/(3*120*8)
  Nodes = 8070*6/5088 = 9.5, close to 10

  Nodes = 8070*(9+3)/(3*203*8)
  Nodes = 96840 / 4872 = 19.9, close to 20

  Nodes = 8070*(6+3)/(3*167*8)
  Nodes = 72630 / 4008 = 18.12, close to 20

  If we're happy to run 6 elems, 8070 spectra, 8 cores in 5 minutes:
  Nodes = 8070*(6+3) / (3*300*8)
  Nodes = 72630 / 7200 = 10 nodes... seems reasonable
*/
func Example_estimateNodeCount() {
	// Based on experimental runs in: https://github.com/pixlise/core/-/issues/113

	// Can only use the ones where we had allcoation of 7.5 set in kubernetes, because the others weren't maxing out cores

	// 5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 10 nodes. Runtime: 3:22
	fmt.Println(estimateNodeCount(4035*2, 3, 3*60+22, 8, 50))
	// 5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:52 (with params)
	fmt.Println(estimateNodeCount(4035*2, 3, 60+52, 8, 50))
	// 5x11, 4035 PMCs, 4 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:11 (with params)
	fmt.Println(estimateNodeCount(4035*2, 4, 2*60+11, 8, 50))
	// 5x11, 4035 PMCs, 5 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:26 (with params)
	fmt.Println(estimateNodeCount(4035*2, 5, 2*60+26, 8, 50))
	// 5x11, 4035 PMCs, 6 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:47 (with params)
	fmt.Println(estimateNodeCount(4035*2, 6, 2*60+47, 8, 50))
	// 5x11, 4035 PMCs, 7 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:55 (no params though)
	fmt.Println(estimateNodeCount(4035*2, 7, 2*60+55, 8, 50))
	// 5x11, 4035 PMCs, 8 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:12 (with params)
	fmt.Println(estimateNodeCount(4035*2, 8, 3*60+12, 8, 50))
	// 5x11, 4035 PMCs, 9 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:23 (with params)
	fmt.Println(estimateNodeCount(4035*2, 9, 3*60+23, 8, 50))
	// 5x11, 4035 PMCs, 10 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:35 (with params)
	fmt.Println(estimateNodeCount(4035*2, 10, 3*60+35, 8, 50))
	// 5x11, 4035 PMCs, 11 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:46 (with params)
	fmt.Println(estimateNodeCount(4035*2, 11, 3*60+46, 8, 50))

	// 5x5, 1769 PMCs, 11 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:47 (with params)
	fmt.Println(estimateNodeCount(1769*2, 11, 60+47, 8, 50))

	// 5x5, 1769 PMCs, 4 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 0:59 (with params)
	fmt.Println(estimateNodeCount(1769*2, 4, 59, 8, 50))

	// Ensure the max cores have an effect
	fmt.Println(estimateNodeCount(1769*2, 4, 59, 8, 6))

	// It's a bit unfortunate we ran all but 1 tests on the same number of cores, but
	// the above data varies the spectra count, element count and expected runtime

	// Below we'd expect 20 for all answers except the first one, but there's a slight (and
	// allowable) drift because we're not exactly spot on with our estimate, and there's
	// fixed overhead time we aren't even calculating properly

	// Output:
	// 10
	// 18
	// 18
	// 18
	// 18
	// 19
	// 19
	// 20
	// 20
	// 21
	// 19
	// 17
	// 6
}

func Example_filesPerNode() {
	fmt.Println(filesPerNode(8088, 5))
	fmt.Println(filesPerNode(8068, 3))

	// Output:
	// 1619
	// 2690
}

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

var testROICreator = pixlUser.UserInfo{
	Name:   "Niko Bellic",
	UserID: "600f2a0806b6c70071d3d174",
	Email:  "niko@rockstar.com",
	Permissions: map[string]bool{
		"access:the-group":     true,
		"access:groupie":       true,
		"access:another-group": true,
	},
}

var testROIs = []ROIWithPMCs{
	{
		PMCs: []int{7, 15, 388},
		ID:   "roi1-id",
		ROISavedItem: &roiModel.ROISavedItem{
			ROIItem: &roiModel.ROIItem{
				Name:            "roi1",
				LocationIndexes: []int32{1, 0, 2},
			},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  false,
				Creator: testROICreator,
			},
		},
	},
	{
		PMCs: []int{7, 450},
		ID:   "roi2-id",
		ROISavedItem: &roiModel.ROISavedItem{
			ROIItem: &roiModel.ROIItem{
				Name:            "roi2",
				LocationIndexes: []int32{0, 3},
			},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  false,
				Creator: testROICreator,
			},
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
18, 7.1, 415, 7840
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row3
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row2
PMC, CaO_%, CaO_int, RTT
18, 7.1, 415, 7840
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row3
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

func Example_combineQuantOutputs_DownloadFails() {
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row2
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
PMC, CaO_%, CaO_int, RTT
30, 5.1, 400, 7890
12, 6.1, 405, 7800
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row2
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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

func Example_processQuantROIsToPMCs_Combined_DownloadFails() {
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`Header row1
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

func Example_cleanLogName() {
	// Don't fix it...
	fmt.Println(cleanLogName("node00001_data.log"))
	// Do fix it...
	fmt.Println(cleanLogName("node00001.pmcs_stdout.log"))
	// Do fix it...
	fmt.Println(cleanLogName("NODE00001.PMCS_stdout.log"))

	// Output:
	// node00001_data.log
	// node00001_stdout.log
	// NODE00001_stdout.log
}
