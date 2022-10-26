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
	"errors"
	"fmt"
)

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

func GetLogLevelName(level LogLevel) (string, error) {
	name, ok := logLevelPrefix[level]
	if !ok {
		return "", fmt.Errorf("Invalid log level: %v", level)
	}
	return name, nil
}

func GetLogLevel(name string) (LogLevel, error) {
	for lev, levName := range logLevelPrefix {
		if levName == name {
			return lev, nil
		}
	}

	return LogDebug, errors.New("Invalid log level name: " + name)
}

// ILogger - Generic logger interface
type ILogger interface {
	Printf(level LogLevel, format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
	SetLogLevel(level LogLevel)
	GetLogLevel() LogLevel
	Close()
}

// This can be called from anywhere that wants to handle a panic gracefully and ensure that
// any logging is done (eg completing sending to cloudwatch)
func HandlePanicWithLog(withLog ILogger) {
	err := recover()
	if err != nil {
		withLog.Errorf("PANIC %v", err)

		// Wait for the above and other msgs to get sent to cloudwatch
		withLog.Close()

		panic(err)
		//os.Exit(1)
	}
}()