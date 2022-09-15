package endpoints

import (
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/cloudwatch"
	"github.com/pixlise/core/v2/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Logger

const logLevelId = "logLevel"

func registerLoggerHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "logger"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/fetch", "logGroup"), apiRouter.MakeMethodPermission("GET", permission.PermWriteMetrics), logRequest)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/level"), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), getLogLevel)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/level", logLevelId), apiRouter.MakeMethodPermission("PUT", permission.PermWriteMetrics), putLogLevel)
}

func logRequest(params handlers.ApiHandlerParams) (interface{}, error) {
	logGroup := "/api/prod-" + params.PathParams["logGroup"]
	logs, err := cloudwatch.FetchLogs(logGroup, params.Svcs)
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
