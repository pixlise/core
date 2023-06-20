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
	"strings"
)

// StdOutLoggerForTest - For mocking out in tests, but saves logs so they are searchable
type StdOutLoggerForTest struct {
	logs     []string
	logLevel LogLevel
}

func (l *StdOutLoggerForTest) Printf(level LogLevel, format string, a ...interface{}) {
	txt := logLevelPrefix[level] + ": " + fmt.Sprintf(format, a...)
	l.logs = append(l.logs, txt)
	log.Println(txt)
}
func (l *StdOutLoggerForTest) Debugf(format string, a ...interface{}) {
	if l.logLevel <= LogDebug {
		l.Printf(LogDebug, format, a...)
	}
}
func (l *StdOutLoggerForTest) Infof(format string, a ...interface{}) {
	if l.logLevel <= LogInfo {
		l.Printf(LogInfo, format, a...)
	}
}
func (l *StdOutLoggerForTest) Errorf(format string, a ...interface{}) {
	l.Printf(LogError, format, a...)
}

func (l *StdOutLoggerForTest) SetLogLevel(level LogLevel) {
	l.logLevel = level
}
func (l *StdOutLoggerForTest) GetLogLevel() LogLevel {
	return l.logLevel
}

// Checking logs (for tests)
func (l *StdOutLoggerForTest) LastLogLine() string {
	if len(l.logs) <= 0 {
		return ""
	}
	return l.logs[len(l.logs)-1]
}

func (l *StdOutLoggerForTest) LogContains(txt string) bool {
	for _, line := range l.logs {
		if strings.Contains(line, txt) {
			return true
		}
	}
	return false
}

func (l *StdOutLoggerForTest) Close() {
}
