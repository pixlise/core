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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v4/core/awsutil"
)

func Example_s3ListingWithContinuation() {
	const bucket = "dev-pixlise-data"
	const listPath = "Datasets/"

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(bucket), Prefix: aws.String(listPath),
		},
		{
			Bucket: aws.String(bucket), Prefix: aws.String(listPath), ContinuationToken: aws.String("cont-1"),
		},
		{
			Bucket: aws.String(bucket), Prefix: aws.String(listPath), ContinuationToken: aws.String("cont-2"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			IsTruncated:           aws.Bool(true),
			NextContinuationToken: aws.String("cont-1"),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-123/summary.json")},
				{Key: aws.String("Datasets/abc-123/node1.json")},
				{Key: aws.String("Datasets/abc-123/params.json")},
			},
		},
		{
			IsTruncated:           aws.Bool(true),
			NextContinuationToken: aws.String("cont-2"),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-456/summary.json")},
				{Key: aws.String("Datasets/abc-789/summary.json")},
				{Key: aws.String("Datasets/")}, // Happens when we create a path in S3 web console, but has no use for us, so we filter it
				{Key: aws.String("Datasets/abc-456/params.json")},
			},
		},
		{
			IsTruncated: aws.Bool(false),
			Contents: []*s3.Object{
				{Key: aws.String("Datasets/abc-456/output/combined.csv")},
			},
		},
	}

	fs := MakeS3Access(&mockS3)
	list, err := fs.ListObjects(bucket, listPath)
	fmt.Printf("%v, list: %v\n", err, list)

	// Output:
	// <nil>, list: [Datasets/abc-123/summary.json Datasets/abc-123/node1.json Datasets/abc-123/params.json Datasets/abc-456/summary.json Datasets/abc-789/summary.json Datasets/abc-456/params.json Datasets/abc-456/output/combined.csv]
}

func Example_getBucketFromS3Url() {
	b, err := GetBucketFromS3Url("some/path/file.json")
	fmt.Printf("%v|%v\n", b, err)

	b, err = GetBucketFromS3Url("s3:///path/file.json")
	fmt.Printf("%v|%v\n", b, err)

	b, err = GetBucketFromS3Url("s3://bucket")
	fmt.Printf("%v|%v\n", b, err)

	b, err = GetBucketFromS3Url("s3://the_bucket/some/path.json")
	fmt.Printf("%v|%v\n", b, err)

	// Output:
	// |GetBucketFromS3Url parameter was not a valid S3 url: some/path/file.json
	// |GetBucketFromS3Url failed to get bucket from S3 url: s3:///path/file.json
	// |GetBucketFromS3Url failed to get bucket from S3 url: s3://bucket
	// the_bucket|<nil>
}

func Example_getPathFromS3Url() {
	p, err := GetPathFromS3Url("some/path/file.json")
	fmt.Printf("%v|%v\n", p, err)

	p, err = GetPathFromS3Url("s3:///path/file.json")
	fmt.Printf("%v|%v\n", p, err)

	p, err = GetPathFromS3Url("s3://bucket")
	fmt.Printf("%v|%v\n", p, err)

	p, err = GetPathFromS3Url("s3://the_bucket/some/path.json")
	fmt.Printf("%v|%v\n", p, err)

	// Output:
	// |GetPathFromS3Url parameter was not a valid S3 url: some/path/file.json
	// |GetPathFromS3Url failed to get path from S3 url: s3:///path/file.json
	// |GetPathFromS3Url failed to get path from S3 url: s3://bucket
	// some/path.json|<nil>
}
