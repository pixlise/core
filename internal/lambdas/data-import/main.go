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
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pixlise/core/v2/core/awsutil"
	dataImportRunner "github.com/pixlise/core/v2/data-import/runner"
)

func HandleRequest(ctx context.Context, event awsutil.Event) error {
	configBucket := os.Getenv("CONFIG_BUCKET")
	datasetBucket := os.Getenv("DATASETS_BUCKET")
	manualBucket := os.Getenv("MANUAL_BUCKET")
	envName := os.Getenv("ENVIRONMENT_NAME")

	for _, record := range event.Records {
		return dataImportRunner.RunDatasetImport([]byte(record.SNS.Message), configBucket, datasetBucket, manualBucket, envName)
	}
	return nil
}

func main() {
	/*
		// Garde failing: 76481028 Matched image PMC is 0
		// Dourbes 245: 89063943
		err := processImportTrigger([]byte(`{
			"datasetaddons": {
				"dir": "dataset-addons/089063943/custom-meta.json",
				"log": "dataimport-12345678"
			}
		}`))
		fmt.Printf("%v", err)
	*/
	os.Mkdir("/tmp/profile", 0750)
	lambda.Start(HandleRequest)
}
