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
	"fmt"
	"github.com/pixlise/core/core/logger"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/utils"
)

func downloadDirectoryZip(s3bucket string, s3path string, fs fileaccess.FileAccess) (string, error) {
	os.MkdirAll(localInputPath, os.ModePerm)

	bytes, err := fs.ReadObject(s3bucket, s3path)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile(localInputPath, "zip")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return "", err
	}
	f.Sync()

	_, err = utils.UnzipDirectory(f.Name(), localUnzipPath)
	if err != nil {
		return "", err
	}

	err = os.Remove(f.Name())
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func uploadDirectoryToAllEnvironments(fs fileaccess.FileAccess, root string, datasetID string, artifactBucket string, envBuckets []string, jobLog logger.ILogger) error {
	var uploadError error = nil

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				jobLog.Infof("FAILED to read file for upload: %v\n", path)
				uploadError = err
			} else {
				uploadPath := filepath.Join(datasetID, filepath.Base(path))

				jobLog.Infof("Uploading %v to S3://%v/%v\n", path, artifactBucket, uploadPath)
				err = fs.WriteObject(artifactBucket, uploadPath, data)
				if err != nil {
					jobLog.Infof("Failed to upload to s3://%v/%v: %v\n", artifactBucket, uploadPath, err)
					uploadError = err
				}

				// For saving to env buckets, we need to put it relative to Datasets/
				uploadPath = filepath.Join("Datasets", uploadPath)

				for _, envBucket := range envBuckets {
					jobLog.Infof("Uploading %v to S3://%v/%v\n", path, envBucket, uploadPath)
					err = fs.WriteObject(envBucket, uploadPath, data)

					if err != nil {
						jobLog.Infof("Failed to upload to s3://%v/%v: %v\n", envBucket, uploadPath, err)
						uploadError = err
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Print(err)
		return err
	}

	// If we encountered an upload error, this'll return it
	return uploadError
}
