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

package expressions

import (
	"github.com/pixlise/core/v2/core/expressions/modules"
	"github.com/pixlise/core/v2/core/pixlUser"
)

type ExpressionDB interface {
	ListModules() (modules.DataModuleWireLookup, error)
	GetModule(moduleID string, version string) (modules.DataModuleSpecificVersionWire, error)
	CreateModule(input modules.DataModuleInput, creator pixlUser.UserInfo) (modules.DataModuleSpecificVersionWire, error)
	AddModuleVersion(moduleID string, input modules.DataModuleInput) (modules.DataModuleSpecificVersionWire, error)
}