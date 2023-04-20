package endpoints

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/core/api"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Test endpoints - to simulate different errors coming back so PIXLISE UI can be tested against this

func registerTestHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "test"

	router.AddGenericHandler("/"+path.Join(pathPrefix, "500"), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), test500)
	router.AddGenericHandler("/"+path.Join(pathPrefix, "503"), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), test503)
}

func test500(params handlers.ApiHandlerGenericParams) error {
	//n := 1
	//d := 0
	//return fmt.Errorf("%v", n/d)
	return fmt.Errorf("Server blew up")
}

func test503(params handlers.ApiHandlerGenericParams) error {
	return api.MakeStatusError(http.StatusServiceUnavailable, errors.New("Gateway unavailable"))
}
