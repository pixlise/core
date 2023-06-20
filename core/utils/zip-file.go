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

package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func AddFilesToZip(w *zip.Writer, basePath, baseInZip string) {
	// Open the Directory
	files, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Println(err)
	}

	for _, file := range files {
		fmt.Println(basePath + file.Name())
		if !file.IsDir() {
			dat, err := os.ReadFile(basePath + file.Name())
			if err != nil {
				fmt.Println(err)
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				fmt.Println(err)
			}
			_, err = f.Write(dat)
			if err != nil {
				fmt.Println(err)
			}
		} else if file.IsDir() {

			// Recurse
			newBase := basePath + file.Name() + "/"
			fmt.Println("Recursing and Adding SubDir: " + file.Name())
			fmt.Println("Recursing and Adding SubDir: " + newBase)

			AddFilesToZip(w, newBase, baseInZip+file.Name()+"/")
		}
	}
}

func UnzipDirectory(src string, dest string, flattenPaths bool) ([]string, error) {
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		// If the zip path starts with __MACOSX, ignore it, it's garbage that a mac laptop has included...
		if strings.HasPrefix(f.Name, "__MACOSX") {
			continue
		}

		thisPath := f.Name
		if flattenPaths {
			// This may end in a /, in which case there's nothing to write
			if strings.HasSuffix(thisPath, "/") {
				continue
			}
			thisPath = path.Base(thisPath)
		}

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, thisPath)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

// ZipDirectory - zips a whole directory and its contents (NOT recursive!)
// See: https://golang.org/pkg/archive/zip/#example_Writer
func ZipDirectory(dirPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Recurse for all files in the dir
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		filePath := filepath.Join(dirPath, file.Name())

		// Create file in zip
		header := &zip.FileHeader{
			Name:     file.Name(),
			Method:   zip.Deflate,
			Modified: time.Now(),
		}

		f, err := w.CreateHeader(header)
		if err != nil {
			return nil, err
		}

		// Read file contents
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		// Write file contents to zip
		_, err = f.Write(contents)
		if err != nil {
			return nil, err
		}
	}

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return nil, err
	}

	// Return zip data
	return buf.Bytes(), nil
}
