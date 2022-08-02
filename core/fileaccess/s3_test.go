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

package fileaccess

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
)

func Example_S3ListingWithContinuation() {
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
