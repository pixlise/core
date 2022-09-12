package endpoints

import (
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/cloudwatch"
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
