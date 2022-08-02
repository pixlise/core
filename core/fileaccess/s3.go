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
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pixlise/core/core/utils"
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

func (s3Access S3Access) ReadObject(bucket string, path string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	result, err := s3Access.s3Api.GetObject(input)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(result.Body)
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
	result := make([]string, len(contents.Contents))

	for i, item := range contents.Contents {
		//fmt.Println("Name:         ", *item.Key)
		//fmt.Println("Last modified:", *item.LastModified)
		//fmt.Println("Size:         ", *item.Size)
		//fmt.Println("Storage class:", *item.StorageClass)
		result[i] = *item.Key
	}

	return result
}
