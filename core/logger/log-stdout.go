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

import (
	"fmt"
	"log"
)

// StdOutLogger - For mocking out in tests
type StdOutLogger struct {
	logLevel LogLevel
}

func (l *StdOutLogger) Printf(level LogLevel, format string, a ...interface{}) {
	txt := logLevelPrefix[level] + ": " + fmt.Sprintf(format, a...)
	log.Println(txt)
}
func (l *StdOutLogger) Debugf(format string, a ...interface{}) {
	l.Printf(LogDebug, format, a...)
}
func (l *StdOutLogger) Infof(format string, a ...interface{}) {
	l.Printf(LogInfo, format, a...)
}
func (l *StdOutLogger) Errorf(format string, a ...interface{}) {
	l.Printf(LogError, format, a...)
}

func (l *StdOutLogger) SetLogLevel(level LogLevel) {
	l.logLevel = level
}
func (l *StdOutLogger) GetLogLevel() LogLevel {
	return l.logLevel
}
