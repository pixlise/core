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

package pixlUser

// UserInfo - Anything we need to identify a user
type UserInfo struct {
	Name        string          `json:"name"`
	UserID      string          `json:"user_id"`
	Email       string          `json:"email"`
	Permissions map[string]bool `json:"-"` // This is a lookup - we don't want this in JSON sent out of API though!
}

// APIObjectItem API endpoints send around versions of this struct (with extra fields depending on the data type)
// TODO: maybe need to move this to its own API structures place? It's currently used in more places than just API handlers though.
type APIObjectItem struct {
	Shared  bool     `json:"shared"`
	Creator UserInfo `json:"creator"`
}

// A special shared user ID so code knows if it's referring to this...
const ShareUserID = "shared"
