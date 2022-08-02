// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"flag"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/fileaccess"
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
