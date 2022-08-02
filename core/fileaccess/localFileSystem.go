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
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pixlise/core/core/utils"
)

// Implementation of file access using local file system
type FSAccess struct {
}

func (fs *FSAccess) ListObjects(rootPath string, prefix string) ([]string, error) {
	result := []string{}

	rootOnly := path.Join(rootPath) // Using path.Join to make it match the fullPath cleans off ./ for example
	fullPath := fs.filePath(rootPath, prefix)

	err := filepath.Walk(fullPath, func(pathFound string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Copy out the file names only. This may be too limiting and we may need to return
			// some kind of structs but enough for now
			// Also note pathFound contains the root directory, so we chop it off
			toSave := pathFound
			if strings.HasPrefix(toSave, rootOnly) {
				toSave = toSave[len(rootOnly)+1:]
			}
			result = append(result, toSave)
		}
		return nil
	})

	return result, err
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

func (fs *FSAccess) IsNotFoundError(err error) bool {
	// See https://stackoverflow.com/questions/24043781/idiomatic-way-to-get-os-err-after-call
	if perr, ok := err.(*os.PathError); ok {
		switch perr.Err.(syscall.Errno) {
		case syscall.ENOENT:
			return true
		}
	}

	return false
}

func (fs *FSAccess) filePath(rootPath string, filePath string) string {
	return path.Join(rootPath, filePath)
}
