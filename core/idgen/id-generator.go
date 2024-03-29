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

package idgen

import "github.com/pixlise/core/v4/core/utils"

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Generation of random string IDs

// IDGenerator - Generates ID strings
type IDGenerator interface {
	GenObjectID() string
}

// IDGen - Implementation of ID generator interface
type IDGen struct {
}

// GenObjectID - Implementation of ID generator interface
func (i *IDGen) GenObjectID() string {
	return utils.RandStringBytesMaskImpr(16)
}

// Here we really just expose some test helpers
type MockIDGenerator struct {
	IDs []string
}

func (m *MockIDGenerator) GenObjectID() string {
	if len(m.IDs) > 0 {
		id := m.IDs[0]
		m.IDs = m.IDs[1:]
		return id
	}
	return "NO_ID_DEFINED"
}
