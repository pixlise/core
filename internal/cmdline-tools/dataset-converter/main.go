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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"

	dataConverter "github.com/pixlise/core/v2/data-import/data-converter"
	"github.com/pixlise/core/v2/data-import/importer"
)

func main() {
	fmt.Println("==============================")
	fmt.Println("=  PIXLISE dataset importer  =")
	fmt.Println("==============================")

	ilog := &logger.StdOutLogger{}

	// This can be run in various modes...

	var argImportFrom = flag.String("source", "local", "Source to import from: local, cloud, trigger")
	var argDatasetBucket = flag.String("dataset-bucket", "", "Dataset bucket name")
	var argPseudoPath = flag.String("pseudo-path", "", "Path to pseudo-intensity file")
	var argImportPath = flag.String("import-path", "", "Path to import directory")
	var argConfigBucket = flag.String("config-bucket", "", "Config bucket name")
	var argManualUploadBucket = flag.String("manual-bucket", "", "Manual uploads bucket name")
	var argDatasetID = flag.String("dataset-id", "", "Dataset ID to import")
	var argTrigger = flag.String("trigger", "", "SNS trigger message, serialised as string")

	flag.Parse()

	var datasetIDImported string
	var err error

	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("AWS GetSession failed: %v", err)
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("AWS GetS3 failed: %v", err)
	}

	localFS := &fileaccess.FSAccess{}
	remoteFS := fileaccess.MakeS3Access(svc)

	// Ensure this exists
	if len(*argDatasetBucket) <= 0 {
		log.Fatalf("dataset-bucket not set")
	}

	switch *argImportFrom {
	case "local":
		// Ensure these exist
		if len(*argImportPath) <= 0 {
			log.Fatalf("import-path not set")
		}
		if len(*argPseudoPath) <= 0 {
			log.Fatalf("pseudo-path not set")
		}
		if len(*argDatasetID) <= 0 {
			log.Fatalf("dataset-id not set")
		}

		workingDir := ""
		workingDir, err = ioutil.TempDir("", "data-converter")
		if err != nil {
			log.Fatalf("Failed to create working dir: %v", err)
		}
		datasetIDImported, err = dataConverter.ImportFromLocalFileSystem(localFS, remoteFS, workingDir, *argImportPath, *argPseudoPath, *argDatasetBucket, *argDatasetID, ilog)
	case "cloud":
		// Ensure these exist
		if len(*argConfigBucket) <= 0 {
			log.Fatalf("config-bucket not set")
		}
		if len(*argManualUploadBucket) <= 0 {
			log.Fatalf("manual-bucket not set")
		}
		if len(*argDatasetID) <= 0 {
			log.Fatalf("dataset-id not set")
		}
		_, datasetIDImported, err = dataConverter.ImportDataset(localFS, remoteFS, *argConfigBucket, *argManualUploadBucket, *argDatasetBucket, *argDatasetID, ilog, true)
	case "trigger":
		/* An example case, where trigger message is set to:
		{
			"datasetaddons": {
				"dir": "dataset-addons/089063943/custom-meta.json",
				"log": "dataimport-12345678"
			}
		}*/
		// Ensure these exist
		if len(*argConfigBucket) <= 0 {
			log.Fatalf("config-bucket not set")
		}
		if len(*argManualUploadBucket) <= 0 {
			log.Fatalf("manual-bucket not set")
		}
		if len(*argTrigger) <= 0 {
			log.Fatalf("trigger not set")
		}
		_, err = importer.ImportForTrigger([]byte(*argTrigger), "cmd-line", *argConfigBucket, *argDatasetBucket, *argManualUploadBucket, ilog, remoteFS)
	default:
		log.Fatalf("Unknown source: %v", *argImportFrom)
		return
	}

	if err != nil {
		ilog.Errorf("Import error: %v", err)
		os.Exit(1)
	}

	ilog.Infof("Import complete: %v", datasetIDImported)
}
