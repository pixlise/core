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

package quantModel

import (
	"io/ioutil"

	"github.com/pixlise/core/api/services"
	protos "github.com/pixlise/core/generated-protos"
	"google.golang.org/protobuf/proto"
)

func GetQuantification(svcs *services.APIServices, s3Path string) (*protos.Quantification, error) {
	bytes, err := svcs.FS.ReadObject(svcs.Config.UsersBucket, s3Path)
	if err != nil {
		return nil, err
	}

	quantPB := &protos.Quantification{}
	err = proto.Unmarshal(bytes, quantPB)
	if err != nil {
		return nil, err
	}

	return quantPB, nil
}

func ReadQuantificationFile(path string) (*protos.Quantification, error) {
	qbytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	q := &protos.Quantification{}
	err = proto.Unmarshal(qbytes, q)
	if err != nil {
		return nil, err
	}
	return q, nil
}
