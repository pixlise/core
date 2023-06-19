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

package fileaccess

import (
	"os"
	"syscall"
)

func (fs *FSAccess) IsNotFoundError(err error) bool {
	// See https://stackoverflow.com/questions/24043781/idiomatic-way-to-get-os-err-after-call
	if perr, ok := err.(*os.PathError); ok {
		switch perr.Err.(syscall.Errno) {
		case syscall.ERROR_PATH_NOT_FOUND: // Windows
			return true
		case syscall.ERROR_FILE_NOT_FOUND: // Windows
			return true
		}
	}

	return false
}
