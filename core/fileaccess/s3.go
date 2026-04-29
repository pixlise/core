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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pixlise/core/v4/core/logger"
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

func (s3Access S3Access) ReadObjectStream(bucket string, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	result, err := s3Access.s3Api.GetObject(input)
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

func (s3Access S3Access) WriteObjectStream(bucket string, path string, stream io.Reader) error {
	uploader := s3manager.NewUploaderWithClient(s3Access.s3Api)

	upParams := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Body:   stream,
	}

	// Perform an upload
	_, err := uploader.Upload(upParams)
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
		CopySource: aws.String(path.Join(srcBucket, srcPath)),
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

func ClearBucketDir(bucket string, s3Path string, remoteFS FileAccess, logger logger.ILogger) error {
	files, err := remoteFS.ListObjects(bucket, s3Path)
	if err != nil {
		return err
	}

	logger.Infof("Clearing %v files from bucket path: s3://%v/%v...", len(files), bucket, s3Path)

	for c, file := range files {
		if c%100 == 0 {
			logger.Infof("  Clearing file %v of %v...", c, len(files))
		}

		err = remoteFS.DeleteObject(bucket, file)
		if err != nil {
			return err
		}
	}

	logger.Infof("Bucket dir cleared: s3://%v/%v...", bucket, s3Path)
	return nil
}

// Copies files to bucket
// If preserveStructure is true, preserves directory structure from sourcePath.
// If preserveStructure is false, copies all files flat (just filename, no subdirectories).
func CopyToBucket(remoteFS FileAccess, sourcePath string, destBucket string, destPath string, preserveStructure bool, log logger.ILogger) error {
	var uploadError error

	localFS := FSAccess{}
	fileList, _ := localFS.ListObjects(sourcePath, "")
	count := 0
	totalCount := len(fileList)

	jobs := make(chan uploadJob, totalCount)
	results := make(chan uploadResult, totalCount)

	// Start workers
	numUploaders := 1
	if totalCount > 10 {
		numUploaders = 5
	}
	for w := 1; w <= numUploaders; w++ {
		go uploadWorker(w, jobs, results)
	}

	err := filepath.Walk(sourcePath, func(currentPath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			var uploadPath string

			if preserveStructure {
				// Calculate relative path from sourcePath to preserve directory structure
				relPath, err := filepath.Rel(sourcePath, currentPath)
				if err != nil {
					log.Errorf("Failed to calculate relative path: %v", err)
					uploadError = err
					return nil
				}
				uploadPath = path.Join(destPath, filepath.ToSlash(relPath))
			} else {
				// Flat structure: just use the base filename
				sourceFile := filepath.Base(currentPath)
				uploadPath = path.Join(destPath, sourceFile)
			}

			jobs <- uploadJob{
				currentPath:  currentPath,
				uploadPath:   uploadPath,
				destBucket:   destBucket,
				log:          log,
				remoteFS:     remoteFS,
				currentCount: count + 1,
				totalCount:   totalCount,
			}

			count++
		}
		return nil
	})

	if err != nil {
		return err
	}

	if numUploaders > 0 && uploadError == nil {
		close(jobs)

		// Check each upload for an error
		fails := 0
		failedPaths := []string{}
		var firstError error
		for c := 0; c < totalCount; c++ {
			result := <-results
			if result.err != nil {
				if firstError == nil {
					firstError = result.err
				}
				fails++

				if len(failedPaths) < 10 {
					failedPaths = append(failedPaths, result.uploadPath)
				}
			}
		}

		if fails > 0 {
			uploadError = fmt.Errorf("Failed to upload %v files. Paths (up to 10): %v. First error: %v", fails, strings.Join(failedPaths, ","), firstError)
		}
	}

	return uploadError
}

type uploadJob struct {
	currentPath string
	uploadPath  string

	destBucket string

	totalCount   int
	currentCount int

	log logger.ILogger

	remoteFS FileAccess
}

type uploadResult struct {
	uploadPath string
	err        error
}

func uploadWorker(id int, jobs <-chan uploadJob, results chan<- uploadResult) {
	for j := range jobs {
		data, err := os.ReadFile(j.currentPath)
		if err != nil {
			j.log.Errorf("Failed to read file for upload: %v", j.currentPath)
		} else {
			j.log.Infof("-Uploading [%v/%v]: %v", j.currentCount, j.totalCount, j.currentPath)
			j.log.Infof("---->to s3://%v/%v", j.destBucket, j.uploadPath)
			err = j.remoteFS.WriteObject(j.destBucket, j.uploadPath, data)

			if err != nil {
				j.log.Errorf("Failed to upload to s3://%v/%v: %v", j.destBucket, j.uploadPath, err)
			}
		}

		results <- uploadResult{err: err, uploadPath: j.uploadPath}
	}
}
