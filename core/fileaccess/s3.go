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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pixlise/core/v4/core/utils"
)

// Implementation of file access using AWS S3
type S3Access struct {
	s3Api s3iface.S3API
}

func MakeS3Access(s3Api s3iface.S3API) S3Access {
	return S3Access{s3Api: s3Api}
}

// ListObjects - calls AWS ListObjectsV2 and if a continuation token is returned this keeps looping
// and storing more items until no more continuation tokens are left.
func (s3Access S3Access) ListObjects(bucket string, prefix string) ([]string, error) {
	continuationToken := ""
	result := []string{}

	params := s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	for true {
		// If we have a continuation token, add it to the parameters we send...
		if len(continuationToken) > 0 {
			params.ContinuationToken = aws.String(continuationToken)
		}

		listing, err := s3Access.s3Api.ListObjectsV2(&params)

		if err != nil {
			return []string{}, err
		}

		// Save the returned items...
		result = append(result, getPathsFromBucketContents(listing)...)

		if listing.IsTruncated != nil && *listing.IsTruncated && listing.NextContinuationToken != nil {
			continuationToken = *listing.NextContinuationToken
		} else {
			break
		}
	}

	return result, nil
}

func (s3Access S3Access) ObjectExists(bucket string, path string) (bool, error) {
	_, err := s3Access.s3Api.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	if err == nil {
		return true, nil
	}

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == "NotFound" {
			return false, nil
		}
	}

	return false, err
}

func (s3Access S3Access) ReadObject(bucket string, path string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	result, err := s3Access.s3Api.GetObject(input)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(result.Body)
}

func (s3Access S3Access) WriteObject(bucket string, path string, data []byte) error {
	input := &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	_, err := s3Access.s3Api.PutObject(input)
	return err
}

func (s3Access S3Access) ReadJSON(bucket string, s3Path string, itemsPtr interface{}, emptyIfNotFound bool) error {
	fileData, err := s3Access.ReadObject(bucket, s3Path)

	// If we got an error, and it's an S3 key not found, and we're told to ignore these and return empty data, then do so
	if err != nil {
		if emptyIfNotFound && s3Access.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(fileData, itemsPtr)
}

func (s3Access S3Access) WriteJSON(bucket string, s3Path string, itemsPtr interface{}) error {
	fileData, err := json.MarshalIndent(itemsPtr, "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return err
	}

	return s3Access.WriteObject(bucket, s3Path, fileData)
}

func (s3Access S3Access) WriteJSONNoIndent(bucket string, s3Path string, itemsPtr interface{}) error {
	fileData, err := json.Marshal(itemsPtr)
	if err != nil {
		return err
	}

	return s3Access.WriteObject(bucket, s3Path, fileData)
}

func (s3Access S3Access) DeleteObject(bucket string, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	_, err := s3Access.s3Api.DeleteObject(input)
	return err
}

func (s3Access S3Access) CopyObject(srcBucket string, srcPath string, dstBucket string, dstPath string) error {
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstPath),
		CopySource: aws.String(srcBucket + "/" + srcPath),
	}
	_, err := s3Access.s3Api.CopyObject(input)
	return err
}

func (s3Access S3Access) EmptyObjects(targetBucket string) error {
	iter := s3manager.NewDeleteListIterator(s3Access.s3Api, &s3.ListObjectsInput{
		Bucket: aws.String(targetBucket),
	})

	// Traverse iterator deleting each object
	if err := s3manager.NewBatchDeleteWithClient(s3Access.s3Api).Delete(aws.BackgroundContext(), iter); err != nil {
		return err
	}

	return nil
}

func (s3Access S3Access) IsNotFoundError(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == s3.ErrCodeNoSuchKey {
			return true
		}
	}
	return false
}

// Helper functions

// getPathsFromBucketContents - Returns only the paths that came back as part of listing a buckets contents
func getPathsFromBucketContents(contents *s3.ListObjectsV2Output) []string {
	result := make([]string, 0, len(contents.Contents))

	for _, item := range contents.Contents {
		//fmt.Println("Name:         ", *item.Key)
		//fmt.Println("Last modified:", *item.LastModified)
		//fmt.Println("Size:         ", *item.Size)
		//fmt.Println("Storage class:", *item.StorageClass)

		// We filter out paths that end in / from S3, these are pointless but can happen if
		// something was made via the web console with create directory, it creates these empty objects...
		if !strings.HasSuffix(*item.Key, "/") {
			result = append(result, *item.Key)
		}
	}

	return result
}

func GetBucketFromS3Url(url string) (string, error) {
	trimmedUrl := strings.TrimPrefix(url, "s3://")
	if trimmedUrl == url {
		return "", fmt.Errorf("GetBucketFromS3Url parameter was not a valid S3 url: %v", url)
	}

	// Get the bit before the first slash, that's the bucket
	slashPos := strings.Index(trimmedUrl, "/")
	if slashPos <= 0 {
		return "", fmt.Errorf("GetBucketFromS3Url failed to get bucket from S3 url: %v", url)
	}

	return trimmedUrl[0:slashPos], nil
}

func GetPathFromS3Url(url string) (string, error) {
	trimmedUrl := strings.TrimPrefix(url, "s3://")
	if trimmedUrl == url {
		return "", fmt.Errorf("GetPathFromS3Url parameter was not a valid S3 url: %v", url)
	}

	// Get the bit before the first slash, that's the bucket
	slashPos := strings.Index(trimmedUrl, "/")
	if slashPos <= 0 {
		return "", fmt.Errorf("GetPathFromS3Url failed to get path from S3 url: %v", url)
	}

	return trimmedUrl[slashPos+1:], nil
}
