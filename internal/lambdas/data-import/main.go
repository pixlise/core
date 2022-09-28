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
	"github.com/pixlise/core/v2/data-import/importer"
)

func HandleRequest(ctx context.Context, event awsutil.Event) error {
	configBucket := os.Getenv("CONFIG_BUCKET")
	datasetBucket := os.Getenv("DATASETS_BUCKET")
	manualBucket := os.Getenv("MANUAL_BUCKET")
	envName := os.Getenv("ENVIRONMENT_NAME")

	// Normally we'd only expect event.Records to be of length 1...
	for _, record := range event.Records {
		err := importer.ImportForTrigger([]byte(record.SNS.Message), envName, configBucket, datasetBucket, manualBucket, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	os.Mkdir("/tmp/profile", 0750)
	lambda.Start(HandleRequest)
}
