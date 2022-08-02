// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

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
