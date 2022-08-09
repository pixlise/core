package endpoints

import (
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/core/cloudwatch"
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
