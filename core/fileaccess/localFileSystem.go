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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/core/utils"
)

// Implementation of file access using local file system
type FSAccess struct {
}

func (fs *FSAccess) ListObjects(rootPath string, prefix string) ([]string, error) {
	result := []string{}

	rootOnly := filepath.Join(rootPath) // Using filepath.Join to make it match the fullPath cleans off ./ for example
	fullPath := fs.filePath(rootPath, prefix)

	// To have common behaviour on S3 vs local file system, here we check if the path exists
	// because user may be querying for files with a given path prefix, so we my have to go up
	// one directory and walk those files WITH a prefix check
	filePrefix := ""
	fullPathExists, err := fs.ObjectExists(fullPath, "")
	if err != nil {
		return result, err
	}

	if !fullPathExists {
		// Try go up a directory
		filePrefix = filepath.Base(fullPath)
		fullPath = filepath.Dir(fullPath)

		// Check this directory exists...
		fullPathExists, err = fs.ObjectExists(fullPath, "")
		if err != nil {
			return result, err
		}

		// If this doesn't exist, no files found like this...
		if !fullPathExists {
			return result, nil
		}
	}

	err = filepath.Walk(fullPath, func(pathFound string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {

			// If we are dealing with a file prefix check, do the check here, where we have
			// the entire path to look at
			if len(filePrefix) > 0 {
				prefixToCheck := filepath.Join(fullPath, filePrefix)
				if !strings.HasPrefix(pathFound, prefixToCheck) {
					// Doesn't have the prefix, so we're not saving this one!
					return nil
				}
			}

			// Copy out the file names only. This may be too limiting and we may need to return
			// some kind of structs but enough for now
			// Also note pathFound contains the root directory, so we chop it off
			toSave := pathFound
			if strings.HasPrefix(toSave, rootOnly) {
				toSave = toSave[len(rootOnly)+1:]
			}

			// Force paths to be / on Windows, mainly so Example style unit tests pass
			if os.PathSeparator == '\\' {
				toSave = strings.ReplaceAll(toSave, "\\", "/")
			}

			result = append(result, toSave)
		}
		return nil
	})

	return result, err
}

func (fs *FSAccess) ObjectExists(rootPath string, path string) (bool, error) {
	fullPath := fs.filePath(rootPath, path)
	_, err := os.Stat(fullPath)

	// If we got a not exist error, file doesn't exist, and this is not an error...
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	// Otherwise, return the bool flag and error itself
	return err == nil, err
}

func (fs *FSAccess) ReadObject(rootPath string, path string) ([]byte, error) {
	fullPath := fs.filePath(rootPath, path)
	return os.ReadFile(fullPath)
}

func (fs *FSAccess) WriteObject(rootPath string, path string, data []byte) error {
	fullPath := fs.filePath(rootPath, path)

	// Ensure any subdirs in between are created
	createPath := filepath.Dir(fullPath)
	err := os.MkdirAll(createPath, 0777)
	if err != nil {
		return err
	}

	// Write the file out, this will create if needed else truncate and write
	return os.WriteFile(fullPath, data, 0777)
}

func (fs *FSAccess) ReadJSON(rootPath string, s3Path string, itemsPtr interface{}, emptyIfNotFound bool) error {
	fileData, err := fs.ReadObject(rootPath, s3Path)

	// If we got an error, and it's an S3 key not found, and we're told to ignore these and return empty data, then do so
	if err != nil {
		if emptyIfNotFound && fs.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(fileData, itemsPtr)
}

func (fs *FSAccess) WriteJSON(rootPath string, s3Path string, itemsPtr interface{}) error {
	fileData, err := json.MarshalIndent(itemsPtr, "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return err
	}

	return fs.WriteObject(rootPath, s3Path, fileData)
}

func (fs *FSAccess) WriteJSONNoIndent(rootPath string, s3Path string, itemsPtr interface{}) error {
	fileData, err := json.Marshal(itemsPtr)
	if err != nil {
		return err
	}

	return fs.WriteObject(rootPath, s3Path, fileData)
}

func (fs *FSAccess) DeleteObject(rootPath string, path string) error {
	fullPath := fs.filePath(rootPath, path)
	return os.Remove(fullPath)
}

func (fs *FSAccess) CopyObject(srcRootPath string, srcPath string, dstRootPath string, dstPath string) error {
	srcFullPath := fs.filePath(srcRootPath, srcPath)

	fin, err := os.Open(srcFullPath)
	if err != nil {
		return err
	}
	defer fin.Close()

	dstFullPath := fs.filePath(dstRootPath, dstPath)
	fout, err := os.Create(dstFullPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	return err
}

func (fs *FSAccess) EmptyObjects(rootPath string) error {
	// Found we had a function floating around already that does this
	// and it doesn't delete the original dir, so doesn't need Mkdir as below
	d, err := os.Open(rootPath)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(rootPath, name))
		if err != nil {
			return err
		}
	}
	return nil
	/*
		// Here we remove all files/dirs under it (including itself), then recreate it to have it there as an empty dir
		// to match AWS implementation, where you empty the bucket but it's still there
		// This way unit tests pass too!
		err := os.RemoveAll(rootPath)
		if err != nil {
			return err
		}
		return os.Mkdir(rootPath, 777)*/
}

func (fs *FSAccess) filePath(rootPath string, filePath string) string {
	// Return them joined, but obey ./ at the start as this will be required for running local tests for example
	result := filepath.Join(rootPath, filePath)
	if strings.HasPrefix(rootPath, "./") {
		result = "./" + result
	}

	return result
}

// Creates a directory under the specified root, ensures it's empty (eg if it already existed)
func MakeEmptyLocalDirectory(root string, subdir string) (string, error) {
	emptyDirPath := filepath.Join(root, subdir)

	// Create and make sure it's empty
	err := os.MkdirAll(emptyDirPath, os.ModePerm)
	if err != nil {
		return emptyDirPath, fmt.Errorf("Failed to create directory %v for importer: %v", emptyDirPath, err)
	}

	localFS := FSAccess{}
	err = localFS.EmptyObjects(emptyDirPath)
	if err != nil {
		return emptyDirPath, fmt.Errorf("Failed to clear directory %v for importer: %v", emptyDirPath, err)
	}

	return emptyDirPath, nil
}

func CopyFileLocally(srcPath string, dstPath string) error {
	// Read all content of src to data
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Write data to dst
	return os.WriteFile(dstPath, data, 0644)
}
