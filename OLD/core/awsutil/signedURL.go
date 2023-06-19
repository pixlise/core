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

package awsutil

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// RealURLSigner - The runtime URL signer (as opposed to the mock)
type RealURLSigner struct {
}

// GetSignedURL - Generates a signed URL for a given S3 object
func (r *RealURLSigner) GetSignedURL(svc s3iface.S3API, bucket string, path string, expirySec time.Duration) (string, error) {
	req, _ /*output*/ := svc.GetObjectRequest(
		&s3.GetObjectInput{
			Bucket:               aws.String(bucket),
			Key:                  aws.String(path),
			ResponseCacheControl: aws.String("no-cache"), // Added so context image loading doesn't break
		})
	/*
		if output == nil {
			return "", errors.New("Failed to get object: " + bucket + "/" + path)
		}
	*/
	urlStr, err := req.Presign(expirySec * time.Second)

	if err != nil {
		return "", err
	}

	return urlStr, nil
}
