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
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/jcxplorer/cwlogger"
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

// ILogger - Generic logger interface
type ILogger interface {
	Printf(level LogLevel, format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// Logger - Structure holding API logger internals
type Logger struct {
	logger   *cwlogger.Logger
	logLevel LogLevel
}

// DefaultGroup - name to use, usually logs go here, except for things like quant
// jobs where we want to track the workings of an individual job
const DefaultGroup = "API"

// Init - initialises the logger, given settings and AWS session
func Init(logGroupName string, logLevel LogLevel, environmentName string, sess *session.Session) (Logger, error) {
	var result Logger

	// The actual log group name is prefixed by env so we never confuse them...
	theLogGroup := fmt.Sprintf("/api/%v-%v", environmentName, logGroupName)

	// Here we actually init a cwlogger in the background
	logger, err := cwlogger.New(&cwlogger.Config{
		LogGroupName: theLogGroup,
		Client:       cloudwatchlogs.New(sess),
	})

	if err != nil {
		return result, err
	}

	// Setup result
	result.logger = logger
	result.logLevel = logLevel

	return result, nil
}

// Printf - Print to log, with format string and log level
func (l Logger) Printf(level LogLevel, format string, a ...interface{}) {
	// If we're not on this log level, skip
	if l.logLevel > level {
		return
	}

	txt := logLevelPrefix[level] + ": " + fmt.Sprintf(format, a...)

	// Write to the cloudwatch logger
	l.logger.Log(time.Now(), txt)

	// Also write to local stdout
	log.Println(txt)
}

// Debugf - Print debug to log, with format string
func (l Logger) Debugf(format string, a ...interface{}) {
	l.Printf(LogDebug, format, a...)
}

// Infof - Print info to log, with format string
func (l Logger) Infof(format string, a ...interface{}) {
	l.Printf(LogInfo, format, a...)
}

// Errorf - Print error to log, with format string
func (l Logger) Errorf(format string, a ...interface{}) {
	l.Printf(LogError, format, a...)
}

// StdOutLogger - For mocking out in tests
type StdOutLogger struct {
	logs []string
}

func (l StdOutLogger) Printf(level LogLevel, format string, a ...interface{}) {
	txt := logLevelPrefix[level] + ": " + fmt.Sprintf(format, a...)
	l.logs = append(l.logs, txt)
	log.Println(txt)
}
func (l StdOutLogger) Debugf(format string, a ...interface{}) {
	l.Printf(LogDebug, format, a...)
}
func (l StdOutLogger) Infof(format string, a ...interface{}) {
	l.Printf(LogInfo, format, a...)
}
func (l StdOutLogger) Errorf(format string, a ...interface{}) {
	l.Printf(LogError, format, a...)
}

// NullLogger - For mocking out in tests
type NullLogger struct {
}

func (l NullLogger) Printf(level LogLevel, format string, a ...interface{}) {
	// We do nothing!
}
func (l NullLogger) Debugf(format string, a ...interface{}) {
	// We do nothing!
}
func (l NullLogger) Infof(format string, a ...interface{}) {
	// We do nothing!
}
func (l NullLogger) Errorf(format string, a ...interface{}) {
	// We do nothing!
}
