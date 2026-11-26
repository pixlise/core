package jobrunner

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
)

// Downloads the repository and returns:
// - The path to the directory the files are in
// - The path of the script file requested
// - The original working directory
// - An error if there is one

func downloadRepository(url, user, secret, jobId, expectedScriptName string, jobLog logger.ILogger) (string, string, string, error) {
	// Get working dir
	origWD, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("Failed to get working directory: %v", err)
	}

	// Make temp path to download/unzip to
	localPath := filepath.Join(os.TempDir(), "repo-"+jobId)
	err = os.MkdirAll(localPath, 0777)
	if err != nil {
		return localPath, "", "", fmt.Errorf("Failed to make working directory for repository download: %v", err)
	}

	// Change to the new dir
	err = os.Chdir(localPath)
	if err != nil {
		return localPath, "", origWD, fmt.Errorf("Failed to chdir to \"%v\" for repo files: %v", localPath, err)
	}

	zipName := "../" + jobId + ".zip"
	f, err := os.Create(zipName)
	if err != nil {
		return localPath, "", origWD, err
	}
	defer f.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return localPath, "", origWD, err
	}

	req.SetBasicAuth(user, secret)

	resp, err := client.Do(req)
	if err != nil {
		return localPath, "", origWD, err
	}

	defer resp.Body.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return localPath, "", origWD, err
	}
	if n <= 0 {
		return localPath, "", origWD, errors.New("No bytes downloaded for source zip")
	}

	// Unzip the file
	unzippedFileNames, err := utils.UnzipDirectory(f.Name(), localPath, false)
	if err != nil {
		return localPath, "", origWD, err
	}

	// Delete the original file
	err = os.Remove(zipName)
	if err != nil {
		jobLog.Errorf("Failed to delete downloaded repository zip: %v", err)
	}

	// Should have at least one file!
	if len(unzippedFileNames) <= 0 {
		return localPath, "", origWD, errors.New("No files extracted from source zip")
	}

	// Ensure the named one exists
	scriptPath := ""
	for _, p := range unzippedFileNames {
		if strings.HasSuffix(p, expectedScriptName) {
			// Yep, found it! Return this path relative to the local path
			// we unzipped to
			scriptPath, err = filepath.Rel(localPath, p)
			if err != nil {
				return localPath, "", origWD, fmt.Errorf("Failed to find relative path to script: %v", err)
			}
			break
		}
	}

	// If not found, bail and delete the whole thing
	if len(scriptPath) <= 0 {
		err = os.RemoveAll(localPath)
		if err != nil {
			jobLog.Errorf("Failed to delete downloaded repository zip: %v", err)
		}

		return localPath, "", origWD, fmt.Errorf("Repository does not contain a script named %v", expectedScriptName)
	}

	// If the named script is in a sub-dir, change to be in that as the root directory
	scriptDir := filepath.Dir(scriptPath)
	if len(scriptDir) > 0 {
		err = os.Chdir(scriptDir)
		if err != nil {
			jobLog.Errorf("Failed to chdir to script directory \"%v\": %v", scriptDir, err)
		}

		// Now the file is just local
		scriptPath = filepath.Base(scriptPath)
	}

	return localPath, scriptPath, origWD, nil
}
