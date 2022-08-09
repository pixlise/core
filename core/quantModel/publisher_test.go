package quantModel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
)

// This Example demonstrates how to check for existing published quants.
func Example_checkExistingQuants() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	datasetsbucket := "datasets-bucket"
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(datasetsbucket), Key: aws.String("Publish/publications.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`
{
	"Datasets": [
		{
			"dataset-id": "mydataset",
			"job-id": "blah",
			"publications": [
				{
					"publisher": "abc",
					"version": 1,
					"timestamp": "2018-09-21T12:42:31Z"
				},
				{
					"publisher": "abc",
					"version": 2,
					"timestamp": "2018-09-22T12:42:31Z"
				}
			]
		}
	]
}
`))),
		},
	}

	fs := fileaccess.MakeS3Access(&mockS3)

	// Check for the latest version of the published quant. It should be version 2.
	i, err := checkCurrentlyPublishedQuantVersion(fs, datasetsbucket, "mydataset")
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	fmt.Printf("Version found: %v\n", i)
	// Output:
	// Version found: 2
}

// This Example we demonstrate sort capabilities for unordered publication times.
func Example_checkExistingQuantsUnsorted() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	datasetsbucket := "datasets-bucket"
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(datasetsbucket), Key: aws.String("Publish/publications.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`
{
	"Datasets": [
		{
			"dataset-id": "mydataset",
			"job-id": "blah",
			"publications": [
				{
					"publisher": "abc",
					"version": 2,
					"timestamp": "2018-09-22T12:42:31Z"
				},
				{
					"publisher": "abc",
					"version": 1,
					"timestamp": "2018-09-21T12:42:31Z"
				}
			]
		}
	]
}
`))),
		},
	}
	fs := fileaccess.MakeS3Access(&mockS3)

	// Check for the latest version of the published quant. It should be version 2. But we have deliberately switched
	// the order of the publications to ensure it doesn't match the last in the list.
	i, err := checkCurrentlyPublishedQuantVersion(fs, datasetsbucket, "mydataset")
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	fmt.Printf("Version found: %v\n", i)
	// Output:
	// Version found: 2
}

// Stage the quantification ready for uploading to OCS
func Example_stageQuant() {

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	datasetsbucket := "datasets-bucket"

	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			Bucket:     aws.String(datasetsbucket),
			Key:        aws.String("Publish/Staging/mydatasetid/myjobid.csv"),
			CopySource: aws.String(datasetsbucket + "/UserContent/shared/mydatasetid/Quantifications/myjobid.csv"),
		},
		{
			Bucket:     aws.String(datasetsbucket),
			Key:        aws.String("Publish/Staging/mydatasetid/summary-myjobid.json"),
			CopySource: aws.String(datasetsbucket + "/UserContent/shared/mydatasetid/Quantifications/summary-myjobid.json"),
		},
	}
	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
		{},
	}
	datasetid := "mydatasetid"
	jobid := "myjobid"
	products := ProductSet{
		OcsPath:         "",
		SourceBucket:    "",
		SourcePrefix:    "",
		DatasetID:       "",
		JobID:           "",
		PqrFileName:     "myjobid.csv",
		PqrMetaFileName: "myjobid.csv.met",
		PqpFileName:     "summary-myjobid.json",
		PqpMetaFileName: "summary-myjobid.json.met",
	}
	fs := fileaccess.MakeS3Access(&mockS3)
	err := stageQuant(fs, datasetsbucket, datasetid, jobid, products)
	if err != nil {
		fmt.Printf("%v", err)
	}

	// Output:
	//
}

func Example_generateMetFiles() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	datasetsbucket := "datasets-bucket"

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(datasetsbucket), Key: aws.String("Publish/Staging/mydatasetid/myjobid.csv.met"), Body: bytes.NewReader([]byte(`{
    "description": "PiQuant Results File"
}`)),
		},
		{
			Bucket: aws.String(datasetsbucket), Key: aws.String("Publish/Staging/mydatasetid/summary-myjobid.json.met"), Body: bytes.NewReader([]byte(`{
    "description": "PiQuant Runtime Parameters File"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	datasetid := "mydatasetid"
	jobid := "myjobid"
	products := ProductSet{
		OcsPath:         "",
		SourceBucket:    "",
		SourcePrefix:    "",
		DatasetID:       datasetid,
		JobID:           jobid,
		PqrFileName:     "myjobid.csv",
		PqrMetaFileName: "myjobid.csv.met",
		PqpFileName:     "summary-myjobid.json",
		PqpMetaFileName: "summary-myjobid.json.met",
	}
	fs := fileaccess.MakeS3Access(&mockS3)
	err := stageMetFiles(fs, datasetsbucket, products)
	if err != nil {
		fmt.Printf("%v", err)
	}

	// Output:
	//
}

// Test code to ensure we make quant products correctly.
func Example_makeQuantProducts() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	datasetsbucket := "datasets-bucket"
	usersbucket := "users-bucket"

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(usersbucket), Key: aws.String("UserContent/shared/mydatasetid/Quantifications/summary-myjobid.json"),
		},
		{
			Bucket: aws.String(datasetsbucket), Key: aws.String("Datasets/mydatasetid/summary.json"),
		},
	}
	// Real quant summary and dataset data from S3
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`
{
    "shared": false,
    "params": {
        "pmcsCount": 2809,
        "name": "sum then quantify",
        "dataBucket": "prodstack-persistencepixlisedata4f446ecf-m36oehuca7uc",
        "datasetPath": "Datasets/069927431/dataset.bin",
        "datasetID": "069927431",
        "jobBucket": "prodstack-persistencepiquantjobs65c7175e-12qccz2o7aimo",
        "detectorConfig": "PIXL/PiquantConfigs/v6",
        "elements": [
            "Ca",
            "Ti",
            "Fe",
            "Al"
        ],
        "parameters": "",
        "runTimeSec": 60,
        "coresPerNode": 4,
        "startUnixTime": 1630629236,
        "creator": {
            "name": "peternemere",
            "user_id": "5de45d85ca40070f421a3a34",
            "email": "peternemere@gmail.com"
        },
        "roiID": "",
        "elementSetID": "",
        "piquantVersion": "registry.gitlab.com/pixlise/piquant/runner:3.2.8-ALPHA",
        "quantMode": "CombinedBulk",
        "comments": "",
        "roiIDs": [
            "scxb9y7xk027mcqq",
            "ee4e94n3fsu3lcqq",
            "4vvypcnk1xtd4sn2",
            "w28i1d4imca1o24e"
        ]
    },
    "elements": [
        "CaO",
        "TiO2",
        "FeO-T",
        "Al2O3"
    ],
    "jobId": "x80xad4l7k6vzalf",
    "status": "complete",
    "message": "Nodes ran: 1",
    "endUnixTime": 1630629323,
    "outputFilePath": "UserContent/5de45d85ca40070f421a3a34/069927431/Quantifications",
    "piquantLogList": [
        "node00001_piquant.log",
        "node00001_stdout.log"
    ]
}
`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`
{
 "dataset_id": "069927431",
 "group": "PIXL-FM",
 "drive_id": 0,
 "site_id": 7,
 "target_id": "?",
 "site": "",
 "target": "",
 "title": "Bellegarde",
 "sol": "0187",
 "rtt": 69927431,
 "sclk": 683483744,
 "context_image": "PCW_0187_0683484439_000RCM_N00700000699274310005075J04.png",
 "location_count": 2816,
 "data_file_size": 17623427,
 "context_images": 21,
 "normal_spectra": 5618,
 "dwell_spectra": 20,
 "bulk_spectra": 2,
 "max_spectra": 2,
 "pseudo_intensities": 2809,
 "detector_config": "PIXL",
 "create_unixtime_sec": 1642519029
}
`))),
		},
	}
	datasetid := "mydatasetid"
	jobid := "myjobid"

	fs := fileaccess.MakeS3Access(&mockS3)
	ocsproducts, err := makeQuantProducts(fs, usersbucket, datasetsbucket, datasetid, jobid, 1)
	if err != nil {
		fmt.Printf("%v", err)
	}
	ocsjson, err := json.MarshalIndent(ocsproducts, "", "  ")
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Printf("%v", string(ocsjson))

	// Output:
	// {
	//   "OcsPath": "/ods/surface/sol/00187/soas/rdr/pixl/PQA",
	//   "SourceBucket": "users-bucket",
	//   "SourcePrefix": "Publish/Staging",
	//   "DatasetID": "mydatasetid",
	//   "JobID": "myjobid",
	//   "PqrFileName": "PES_0187_0683484439_000PQR_N00700000699274310005075J01.CSV",
	//   "PqrMetaFileName": "PES_0187_0683484439_000PQR_N00700000699274310005075J01.CSV.MET",
	//   "PqpFileName": "PES_0187_0683484439_000PQP_N00700000699274310005075J01.JSON",
	//   "PqpMetaFileName": "PES_0187_0683484439_000PQP_N00700000699274310005075J01.JSON.MET"
	// }

}
