package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v2/core/logger"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/utils"
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

	// Clear, not used after this
	bytes = []byte{}

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

func uploadDirectoryToAllEnvironments(fs fileaccess.FileAccess, root string, datasetID string, envBuckets []string, jobLog logger.ILogger) error {
	var uploadError error = nil

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				jobLog.Infof("FAILED to read file for upload: %v\n", path)
				uploadError = err
			} else {
				uploadPath := filepath.Join("Datasets", datasetID, filepath.Base(path))

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
