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

package fileaccess

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/utils"
)

type testData struct {
	Name        string `json:"name"`
	Value       int    `json:"value"`
	Description string `json:"description"`
}

func runTest(fs FileAccess, bucket string) {
	// Write pretty printed JSON
	fmt.Printf("JSON: %v\n", fs.WriteJSON(bucket, "the-files/pretty.json", testData{Name: "Hello", Value: 778, Description: "World"}))

	// Write non-indented JSON
	fmt.Printf("JSON no-indent: %v\n", fs.WriteJSONNoIndent(bucket, "the-files/subdir/ugly.json", testData{Name: "Hello", Value: 778, Description: "World"}))

	// Check file exists, should fail
	exists, err := fs.ObjectExists(bucket, "the-files/data.bin")
	fmt.Printf("Exists1: %v|%v\n", exists, err)

	// Write binary data
	fmt.Printf("Binary: %v\n", fs.WriteObject(bucket, "the-files/data.bin", []byte{250, 130, 10, 0, 33}))

	// Check file exists, should exist now...
	exists, err = fs.ObjectExists(bucket, "the-files/data.bin")
	fmt.Printf("Exists2: %v|%v\n", exists, err)

	// Copy a file
	fmt.Printf("Copy: %v\n", fs.CopyObject(bucket, "the-files/pretty.json", bucket, "the-files/subdir/copied.json"))

	// Copy a file, bad path
	err = fs.CopyObject(bucket, "the-files/prettyzzz.json", bucket, "the-files/subdir/copied2.json")
	fmt.Printf("Copy bad path, got not found error: %v\n", fs.IsNotFoundError(err)) // Don't print aws error because it changes between tests (contains req id)

	// Read each back/verify their contents
	var contents testData
	err = fs.ReadJSON(bucket, "the-files/pretty.json", &contents, false)
	fmt.Printf("Read JSON: %v, %v\n", err, contents)

	err = fs.ReadJSON(bucket, "the-files/pretty.json", &contents, false)
	fmt.Printf("Read JSON no-indent: %v, %v\n", err, contents)

	data, err := fs.ReadObject(bucket, "the-files/data.bin")
	fmt.Printf("Read Binary: %v, %v\n", err, data)

	// Read bad path, then check that this is a not found error
	err = fs.ReadJSON(bucket, "the-files/prettyzzz.json", &contents, false)
	fmt.Printf("Read bad path, got not found error: %v\n", fs.IsNotFoundError(err)) // Don't print aws error because it changes between tests (contains req id)

	// Read the binary file as JSON, should fail to deserialise and get a different error code
	err = fs.ReadJSON(bucket, "the-files/data.bin", &contents, false)
	fmt.Printf("Read bad JSON: %v\n", err)

	// Check this is not seen as a "not found" error
	fmt.Printf("Not a \"not found\" error: %v\n", !fs.IsNotFoundError(err))

	// List files
	listing, err := fs.ListObjects(bucket, "the-files/")
	fmt.Printf("Listing: %v, %v\n", err, listing)

	listing, err = fs.ListObjects(bucket, "the-files/subdir")
	fmt.Printf("Listing subdir: %v, %v\n", err, listing)

	// Listing with a prefix
	listing, err = fs.ListObjects(bucket, "the-files/subdir/ug")
	fmt.Printf("Listing with prefix: %v, %v\n", err, listing)

	// Listing with bad path
	listing, err = fs.ListObjects(bucket, "the-files/non-existant-path/ug")
	fmt.Printf("Listing bad path: %v, %v\n", err, listing)

	// Delete the copy
	fmt.Printf("Delete copy: %v\n", fs.DeleteObject(bucket, "the-files/subdir/copied.json"))

	// Delete bin file
	fmt.Printf("Delete bin: %v\n", fs.DeleteObject(bucket, "the-files/data.bin"))

	// Check listing changed
	listing, err = fs.ListObjects(bucket, "the-files/")
	fmt.Printf("Listing2: %v, %v\n", err, listing)

	listing, err = fs.ListObjects(bucket, "the-files/subdir")
	fmt.Printf("Listing subdir2: %v, %v\n", err, listing)

	// Empty dir
	fmt.Printf("Empty dir: %v\n", fs.EmptyObjects(bucket))

	// List emptied dir
	listing, err = fs.ListObjects(bucket, "")
	fmt.Printf("Listing subdir3: %v, %v\n", err, listing)
}

func Example_localFileSystem() {
	// First, clear any files we may have there already
	fmt.Printf("Setup: %v\n", os.RemoveAll("./test-output/"))

	// Now run the tests
	runTest(&FSAccess{}, "./test-output")

	// NOTE: test output must match the output from S3 (except cleanup steps)

	// Output:
	// Setup: <nil>
	// JSON: <nil>
	// JSON no-indent: <nil>
	// Exists1: false|<nil>
	// Binary: <nil>
	// Exists2: true|<nil>
	// Copy: <nil>
	// Copy bad path, got not found error: true
	// Read JSON: <nil>, {Hello 778 World}
	// Read JSON no-indent: <nil>, {Hello 778 World}
	// Read Binary: <nil>, [250 130 10 0 33]
	// Read bad path, got not found error: true
	// Read bad JSON: invalid character 'ú' looking for beginning of value
	// Not a "not found" error: true
	// Listing: <nil>, [the-files/data.bin the-files/pretty.json the-files/subdir/copied.json the-files/subdir/ugly.json]
	// Listing subdir: <nil>, [the-files/subdir/copied.json the-files/subdir/ugly.json]
	// Listing with prefix: <nil>, [the-files/subdir/ugly.json]
	// Listing bad path: <nil>, []
	// Delete copy: <nil>
	// Delete bin: <nil>
	// Listing2: <nil>, [the-files/pretty.json the-files/subdir/ugly.json]
	// Listing subdir2: <nil>, [the-files/subdir/ugly.json]
	// Empty dir: <nil>
	// Listing subdir3: <nil>, []
}

func Example_s3() {
	rand.Seed(time.Now().UnixNano())
	sess, err := awsutil.GetSessionWithRegion("us-east-1")
	if err != nil {
		fmt.Println("Failed to get AWS session")
		return
	}
	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		fmt.Println("Failed to get S3")
		return
	}

	fmt.Printf("Setup: %v\n", err)

	fs := MakeS3Access(s3svc)

	// Create test S3 bucket for this purpose
	testBucket := "api-fileaccess-s3-test-" + utils.RandStringBytesMaskImpr(10)
	_, err = s3svc.CreateBucket(
		&s3.CreateBucketInput{
			Bucket: aws.String(testBucket),
			//CreateBucketConfiguration:
		},
	)
	if err != nil {
		fmt.Printf("Failed to create test S3 bucket: %v\n", err)
		return
	}

	defer func() {
		_, err := s3svc.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(testBucket)})
		fmt.Printf("Delete bucket errors: %v\n", err)
	}()

	// Now run the tests
	runTest(fs, testBucket)

	// NOTE: test output must match the output from local file system (except cleanup steps)

	// Output:
	// Setup: <nil>
	// JSON: <nil>
	// JSON no-indent: <nil>
	// Exists1: false|<nil>
	// Binary: <nil>
	// Exists2: true|<nil>
	// Copy: <nil>
	// Copy bad path, got not found error: true
	// Read JSON: <nil>, {Hello 778 World}
	// Read JSON no-indent: <nil>, {Hello 778 World}
	// Read Binary: <nil>, [250 130 10 0 33]
	// Read bad path, got not found error: true
	// Read bad JSON: invalid character 'ú' looking for beginning of value
	// Not a "not found" error: true
	// Listing: <nil>, [the-files/data.bin the-files/pretty.json the-files/subdir/copied.json the-files/subdir/ugly.json]
	// Listing subdir: <nil>, [the-files/subdir/copied.json the-files/subdir/ugly.json]
	// Listing with prefix: <nil>, [the-files/subdir/ugly.json]
	// Listing bad path: <nil>, []
	// Delete copy: <nil>
	// Delete bin: <nil>
	// Listing2: <nil>, [the-files/pretty.json the-files/subdir/ugly.json]
	// Listing subdir2: <nil>, [the-files/subdir/ugly.json]
	// Empty dir: <nil>
	// Listing subdir3: <nil>, []
	// Delete bucket errors: <nil>
}

func Example_MakeValidObjectName() {
	fmt.Println(MakeValidObjectName("my file!", true))
	fmt.Println(MakeValidObjectName("this/path/to.bin", true))
	fmt.Println(MakeValidObjectName("Hope \"this\" isn't too $expensive", true))
	fmt.Println(MakeValidObjectName("This-file is it", true))
	fmt.Println(MakeValidObjectName("A!B#C$D/E\\F", true))
	fmt.Println(MakeValidObjectName("This-file; is it", true))
	fmt.Println(MakeValidObjectName("This-file is it", true))
	fmt.Println(MakeValidObjectName("This-file is it", false))

	// Output:
	// my file
	// this_path_to.bin
	// Hope this isnt too expensive
	// This-file is it
	// ABCD_E_F
	// This-file is it
	// This-file is it
	// This-file_is_it
}

func Example_IsValidObjectName() {
	fmt.Println(IsValidObjectName("name"))
	fmt.Println(IsValidObjectName("Name With Spaces"))
	fmt.Println(IsValidObjectName("Name With Spaces"))
	fmt.Println(IsValidObjectName(""))
	fmt.Println(IsValidObjectName("Name \"Quote"))

	// Output:
	// true
	// true
	// true
	// false
	// false
}
