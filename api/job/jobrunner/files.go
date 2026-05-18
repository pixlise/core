package jobrunner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

const dirperm = 0777

func downloadFile(jobLog logger.ILogger, remoteFS fileaccess.FileAccess, bucket string, remotePathAndFile string, localPath string) error {
	jobLog.Debugf("Download \"s3://%v/%v\" -> \"%v\":", bucket, remotePathAndFile, localPath)

	if len(bucket) <= 0 {
		return fmt.Errorf("No bucket specified")
	}
	if len(remotePathAndFile) <= 0 {
		return fmt.Errorf("No remotePathAndFile specified")
	}
	if len(localPath) <= 0 {
		return fmt.Errorf("No localPath specified")
	}

	// Ensure local path exists
	localPathOnly := filepath.Dir(localPath)
	localPathForLog := ""
	if len(localPathOnly) > 0 && localPathOnly != "." {
		err := os.MkdirAll(localPathOnly, dirperm)
		if err != nil {
			return fmt.Errorf("Failed to create local path: \"%v\". Error: %v", localPathOnly, err)
		} else {
			jobLog.Debugf(" Path created: \"%v\"", localPathOnly)
		}
	} else {
		// Write where we'll put the local file...
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("Failed to get working directory: %v", err)
		}

		localPathForLog = "<CWD>/" + localPath
		jobLog.Debugf(" Local path is %v", localPathForLog)
		localPath = filepath.Join(wd, localPath)
	}

	data, err := remoteFS.ReadObject(bucket, remotePathAndFile)
	if err != nil {
		if remoteFS.IsNotFoundError(err) {
			return fmt.Errorf("Failed to download s3://%v/%v: Not found", bucket, remotePathAndFile)
		}
		return fmt.Errorf("Failed to download s3://%v/%v: %v", bucket, remotePathAndFile, err)
	} else {
		jobLog.Debugf(" Downloaded %v bytes", len(data))
	}

	// Save to the file
	err = os.WriteFile(localPath, data, dirperm)
	if err != nil {
		return fmt.Errorf("Failed to write %v byte local file: %v. Error: %v", len(data), localPath, err)
	} else {
		jobLog.Debugf(" Wrote file: %v", localPathForLog)
	}
	return nil
}

func uploadFile(jobLog logger.ILogger, remoteFS fileaccess.FileAccess, localPath string, bucket string, remotePath string) error {
	jobLog.Debugf("Upload %v -> s3://%v/%v", localPath, bucket, remotePath)
	bytes, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	return remoteFS.WriteObject(bucket, remotePath, bytes)
}
