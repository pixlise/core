package endpoints

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/cloudwatch"
	"github.com/pixlise/core/v3/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Logger

const logLevelId = "logLevel"
const logStreamId = "logStream"

func registerLoggerHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "logger"

	// Adjusting and getting log level
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/level"), apiRouter.MakeMethodPermission("GET", permission.PermReadLogs), getLogLevel)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/level", logLevelId), apiRouter.MakeMethodPermission("PUT", permission.PermWriteLogLevel), putLogLevel)

	// Querying logs
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/fetch", logStreamId), apiRouter.MakeMethodPermission("GET", permission.PermReadLogs), logRequest)
}

func logRequest(params handlers.ApiHandlerParams) (interface{}, error) {
	logGroup := "/dataset-importer/" + params.Svcs.Config.EnvironmentName
	logStream := params.PathParams[logStreamId]

	logs, err := cloudwatch.FetchLogs(params.Svcs, logGroup, logStream)

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == cloudwatchlogs.ErrCodeResourceNotFoundException {
			return nil, api.MakeNotFoundError(logStream)
		}
	}

	return logs, err
}

func getLogLevel(params handlers.ApiHandlerParams) (interface{}, error) {
	return logger.GetLogLevelName(params.Svcs.Log.GetLogLevel())
}

func putLogLevel(params handlers.ApiHandlerParams) (interface{}, error) {
	logLevelName := params.PathParams[logLevelId]

	logLevel, err := logger.GetLogLevel(logLevelName)
	if err != nil {
		return nil, err
	}

	// Also set it on the actual logger
	params.Svcs.Log.SetLogLevel(logLevel)

	// Not really an error, but we log in this level to ensure it always gets printed
	params.Svcs.Log.Errorf("User %v request changed log level to: %v", params.UserInfo.UserID, logLevelName)

	return nil, nil
}
