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

// LogLevel - log level type
type LogLevel int

const (

	// LogDebug - DEBUG log level
	LogDebug LogLevel = iota

	// LogInfo - INFO log level
	LogInfo LogLevel = iota

	// LogError - ERROR log level (does not call os.Exit!)
	LogError LogLevel = iota
)

var logLevelPrefix = map[LogLevel]string{
	LogDebug: "DEBUG",
	LogInfo:  "INFO",
	LogError: "ERROR",
}

// ILogger - Generic logger interface
type ILogger interface {
	Printf(level LogLevel, format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}
