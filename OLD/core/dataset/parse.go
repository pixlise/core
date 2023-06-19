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

package dataset

import (
	"io/ioutil"

	"github.com/pixlise/core/v3/api/services"
	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/proto"
)

// GetDataset - returns a dataset proto after downloading from S3
func GetDataset(svcs *services.APIServices, s3Path string) (*protos.Experiment, error) {
	bytes, err := svcs.FS.ReadObject(svcs.Config.DatasetsBucket, s3Path)
	if err != nil {
		return nil, err
	}

	datasetPB := &protos.Experiment{}
	err = proto.Unmarshal(bytes, datasetPB)
	if err != nil {
		return nil, err
	}

	return datasetPB, nil
}

// ReadDatasetFile - reads a dataset proto from local file system
func ReadDatasetFile(path string) (*protos.Experiment, error) {
	dsbytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ds := &protos.Experiment{}
	err = proto.Unmarshal(dsbytes, ds)
	if err != nil {
		return nil, err
	}
	return ds, nil
}
