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

package timestamper

import "time"

type ITimeStamper interface {
	GetTimeNowSec() int64
}

type UnixTimeNowStamper struct {
}

// GetTimeNowSec - Returns unix time now in seconds
func (ts *UnixTimeNowStamper) GetTimeNowSec() int64 {
	return time.Now().Unix()
}

type MockTimeNowStamper struct {
	QueuedTimeStamps []int64
}

// GetTimeNowSec - Returns unix time now in seconds
func (ts *MockTimeNowStamper) GetTimeNowSec() int64 {
	val := ts.QueuedTimeStamps[0]
	ts.QueuedTimeStamps = ts.QueuedTimeStamps[1:]
	return val
}
