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
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"gitlab.com/pixlise/pixlise-go-api/api/filepaths"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
)

func main() {
	sess, err := awsutil.GetSession()
	if err != nil {
		fmt.Print(err)
		return
	}
	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		fmt.Print(err)
		return
	}
	fs := fileaccess.MakeS3Access(s3svc)

	// We have to write to stdout so it gets to cloudwatch logs via lambda magic
	//stdLog := logger.StdOutLogger{}

	var bucket string

	flag.StringVar(&bucket, "bucket", "", "Bucket")
	flag.Parse()

	// list all files where we need...
	summaryPaths := []string{}
	pages := 0
	totalPaths := 0

	params := s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(path.Join(filepaths.RootUserContent) + "/"),
	}

	summarySuffix := "-summary.json"
	s3svc.ListObjectsPages(&params, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		pages++
		for _, value := range page.Contents {
			totalPaths++
			if strings.HasSuffix(*value.Key, summarySuffix) {
				summaryPaths = append(summaryPaths, *value.Key)
			}
		}
		return true
	})

	//fmt.Printf("Pages: %v, paths: %v of %v\n%v\n", pages, len(summaryPaths), totalPaths, strings.Join(summaryPaths, "\n"))

	for _, thisPath := range summaryPaths {
		// Build a destination path
		fileName := filepath.Base(thisPath)
		id := fileName[:len(fileName)-len(summarySuffix)]
		dstPath := path.Join(filepath.Dir(thisPath), "summary-"+id+".json")
		//fmt.Printf("%v -> %v\n", thisPath, dstPath)

		err := fs.CopyObject(bucket, thisPath, bucket, dstPath)
		if err != nil {
			fmt.Printf("ERROR while copying %v: %v\n", thisPath, err)
		} else {
			fmt.Printf("Copied %v -> %v\n", thisPath, dstPath)

			// Now delete the old one
			err = fs.DeleteObject(bucket, thisPath)
			if err != nil {
				fmt.Printf("ERROR: Failed to delete %v: %v\n", thisPath, err)
			}
		}
	}
}
