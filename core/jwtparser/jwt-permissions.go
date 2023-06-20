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

package jwtparser

import "fmt"

func ReadPermissions(claims map[string]interface{}) (map[string]bool, error) {
	result := map[string]bool{}

	claimPermissions, ok := claims["permissions"].([]interface{}) // example of casting interface to something concrete
	if !ok {
		return result, fmt.Errorf("Failed to get permissions from request JWT")
	}

	for _, claimPerm := range claimPermissions {
		// Get it as a string
		claimPermStr, ok := claimPerm.(string)
		if ok {
			result[claimPermStr] = true
		}
	}

	return result, nil
}
