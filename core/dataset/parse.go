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

package dataset

import (
	"io/ioutil"

	"github.com/pixlise/core/api/services"
	protos "github.com/pixlise/core/generated-protos"
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
