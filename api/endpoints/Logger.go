package endpoints

import (
	"gitlab.com/pixlise/pixlise-go-api/api/handlers"
	"gitlab.com/pixlise/pixlise-go-api/api/permission"
	apiRouter "gitlab.com/pixlise/pixlise-go-api/api/router"
	"gitlab.com/pixlise/pixlise-go-api/core/cloudwatch"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Logger

func registerLoggerHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "logger"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, "logGroup"), apiRouter.MakeMethodPermission("GET", permission.PermWriteMetrics), logRequest)
}

func logRequest(params handlers.ApiHandlerParams) (interface{}, error) {
	logGroup := "/api/prod-" + params.PathParams["logGroup"]
	logs, err := cloudwatch.FetchLogs(logGroup, params.Svcs)
	return logs, err
}
