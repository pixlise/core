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

package logger

// NullLogger - For mocking out in tests
type NullLogger struct {
}

func (l *NullLogger) Printf(level LogLevel, format string, a ...interface{}) {
	// We do nothing!
}
func (l *NullLogger) Debugf(format string, a ...interface{}) {
	// We do nothing!
}
func (l *NullLogger) Infof(format string, a ...interface{}) {
	// We do nothing!
}
func (l *NullLogger) Errorf(format string, a ...interface{}) {
	// We do nothing!
}
