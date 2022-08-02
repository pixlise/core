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

package importer

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/pixlise/core/core/logger"
)

////////////////////////////////////////////////////////////////////////
// Directory listing

func GetDirListing(path string, extFilterOrEmpty string, jobLog logger.ILogger) ([]string, error) {
	s3Bucket := getBucketFromS3Path(path)
	if len(s3Bucket) > 0 {
		return getFileListFromS3(s3Bucket, path, jobLog)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	extWithDot := strings.ToUpper(extFilterOrEmpty)
	if len(extWithDot) > 0 && extWithDot[0:1] != "." {
		extWithDot = "." + extWithDot
	}

	result := []string{}
	for _, file := range files {
		if len(extWithDot) <= 0 || strings.ToUpper(filepath.Ext(file.Name())) == extWithDot {
			result = append(result, file.Name())
		}
	}

	return result, nil
}

func getFileListFromS3(bucket string, path string, jobLog logger.ILogger) ([]string, error) {
	// TODO: remove hard-coded AWS regions
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	svc := s3.New(sess)

	jobLog.Infof("Fetching Objects from S3 folder: %s %s", bucket, path)
	list := s3.ListObjectsInput{Bucket: aws.String(bucket), Prefix: aws.String(path)}
	obj, err := svc.ListObjects(&list)

	s := []string{}
	if err != nil {
		return s, fmt.Errorf("Unable to list objects %s, %v", path, err)
	}

	files := obj.Contents

	for _, f := range files {
		s = append(s, *f.Key)
	}

	jobLog.Infof("Files found in S3: %x", len(s))
	return s, nil
}

////////////////////////////////////////////////////////////////////////
// File loading

// Returns path to delete if needs deletion (got from S3)
func getFile(path string, jobLog logger.ILogger) (*os.File, error, string) {
	var err error = nil
	fromS3 := false

	s3Bucket := getBucketFromS3Path(path)
	if len(s3Bucket) > 0 {
		path, err = fetchFileFromS3(s3Bucket, path, jobLog)
		if err != nil {
			return nil, err, ""
		}

		fromS3 = true
	}
	/*
		// make sure its absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err, ""
		}

		f, err := os.Open(absPath)*/

	f, err := os.Open(path)
	if !fromS3 {
		// Don't return the path as a path to delete, we only delete if we downloaded it from S3!
		path = ""
	}
	return f, err, path
}

func cleanup(f *os.File, delPath string) {
	f.Close()

	if len(delPath) > 0 {
		os.Remove(delPath)
	}
}

func ReadCSV(path string, headerIdx int, sep rune, jobLog logger.ILogger) ([][]string, error) {
	csvFile, err, delPath := getFile(path, jobLog)
	if err != nil {
		return nil, err
	}

	defer cleanup(csvFile, delPath)

	if headerIdx > 0 {
		n := 0
		for n < headerIdx {
			n = n + 1
			row1, err := bufio.NewReader(csvFile).ReadSlice('\n')
			if err != nil {
				return nil, err
			}
			_, err = csvFile.Seek(int64(len(row1)), io.SeekStart)
			if err != nil {
				return nil, err
			}
		}
	}

	r := csv.NewReader(csvFile)
	r.TrimLeadingSpace = true
	r.Comma = sep

	// Some of our CSV files contain multiple tables, that we detect during parsing, so instead of using
	// ReadAll() here, which blows up when the # cols differs, we read each line, and if we get the error
	// "wrong number of fields", we can ignore it and keep reading
	rows := [][]string{}
	var lineRecord []string
	for true {
		lineRecord, err = r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			if csverr, ok := err.(*csv.ParseError); !ok && csverr.Err != csv.ErrFieldCount {
				return nil, err
			}
		}

		rows = append(rows, lineRecord)
	}

	return rows, nil
}

func ReadFileLines(path string, jobLog logger.ILogger) ([]string, error) {
	file, err, delPath := getFile(path, jobLog)
	if err != nil {
		return nil, err
	}

	defer cleanup(file, delPath)

	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func ReadJSON(path string, ifcPtr interface{}, jobLog logger.ILogger) error {
	file, err, delPath := getFile(path, jobLog)
	if err != nil {
		return err
	}

	defer cleanup(file, delPath)

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, ifcPtr)
}

func fetchFileFromS3(bucket string, s3path string, jobLog logger.ILogger) (string, error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	downloader := s3manager.NewDownloader(sess)

	folder, _ := filepath.Split(s3path)
	jobLog.Infof("Creating directory: %s", "/tmp/"+folder)
	err := os.MkdirAll("/tmp/"+folder, os.ModePerm)

	if err != nil {
		return "", fmt.Errorf("Unable to create path %q, %v", folder, err)
	}
	jobLog.Infof("Downloading file to %s", "/tmp/"+s3path)

	// TODO: use a function to get temp file path???
	createPath := path.Join("/tmp", s3path)
	file, err := os.Create(createPath)
	if err != nil {
		return "", fmt.Errorf("Unable to create item %v, %v", createPath, err)
	}

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(s3path),
		})
	if err != nil {
		return "", fmt.Errorf("Unable to download item %q, %v", s3path, err)
	}

	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
	return createPath, nil
}

// If not an S3 path, returns empty string
func getBucketFromS3Path(path string) string {
	if !strings.HasPrefix(path, "s3://") {
		return ""
	}

	// So s3://bucket/path/file.txt will become:
	// ["s3:", "", "bucket", "path", "file.txt"]
	// and we return bucket here...
	return strings.Split(path, "/")[2]
}
