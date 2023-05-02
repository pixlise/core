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

// Implements archiving/retrieval of dataset source zip files as delivered by GDS. When new data arrives from GDS, we receive
// it in snapshot zip files containing what arrived at a given time. These are all stored in an S3 bucket and when required
// they are extracted in timestamp order on top of each other to get a view of the data at a given time. Used to then
// generate a PIXLISE dataset using the dataset importer.
package datasetArchive

import (
	"fmt"
	"path"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
)

func AddToDatasetArchive(remoteFS fileaccess.FileAccess, log logger.ILogger, datasetBucket string, sourceBucket string, sourceFilePath string) (bool, error) {
	log.Debugf("Archiving source file: \"s3://%v/%v\"", sourceBucket, sourceFilePath)

	// Work out the file name
	fileName := path.Base(sourceFilePath)

	// TODO: Check if file exists already in archive, in which case fail, because nothing new to be updated?? Or check file sizes differ or something?
	writePath := path.Join(filepaths.RootArchive, fileName)

	exists, err := remoteFS.ObjectExists(datasetBucket, writePath)

	if err != nil {
		err = fmt.Errorf("Failed to check if incoming file exists in archive. Incoming: \"s3://%v/%v\", Destination: \"s3://%v/%v\"", sourceBucket, sourceFilePath, datasetBucket, writePath)
		log.Errorf("%v", err)
	} else if !exists {
		// If the file is confirmed to exist already, we don't write it to the archive and processing should stop (we return exists flag out of here)
		err = remoteFS.CopyObject(sourceBucket, sourceFilePath, datasetBucket, writePath)
		if err != nil {
			err = fmt.Errorf("Failed to archive incoming file: \"s3://%v/%v\"", sourceBucket, sourceFilePath)
		}
	}

	return exists, err
}
