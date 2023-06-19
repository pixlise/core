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

package permission

// This is a list of all the objects that are publicly accessible. This is used to
// determine whether a user has access to an object or not. If the object is not in
// this list, then the user must have access to it in order to see it.
type PublicObjectsAuth struct {
	Datasets        []string // This is a list of all datasets that are public AND have public objects in them
	ROIs            []string
	Expressions     []string
	Modules         []string
	RGBMixes        []string
	Quantifications []string
	Collections     []string
	Workspaces      []string
}

// DatasetAuthInfo - Structure of dataset auth JSON files
// This is used to check whether an individual dataset CAN be public or not
type DatasetAuthInfo struct {
	DatasetID               string `json:"dataset_id"`
	Public                  bool   `json:"public"`
	PublicReleaseUTCTimeSec int64  `json:"public_release_utc_time_sec"`
	Sol                     string `json:"sol"`
}

// DatasetsAuth - Structure of dataset auth JSON files
// This is used to check the public status of all datasets
type DatasetsAuth map[string]DatasetAuthInfo

// These enums keep track of the different types of objects that can be public
type PublicObjectEnumType int64

const (
	PublicObjectDataset PublicObjectEnumType = iota
	PublicObjectROI
	PublicObjectExpression
	PublicObjectModule
	PublicObjectRGBMix
	PublicObjectQuantification
	PublicObjectCollection
	PublicObjectWorkspace
)
