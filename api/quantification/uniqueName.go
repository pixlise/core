package quantification

import (
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
)

func checkQuantificationNameExists(name string, scanId string, svcs *services.APIServices, sessUser sessionuser.SessionUser) bool {
	items, _, err := ListUserQuants(nil, svcs, sessUser)
	if err != nil {
		svcs.Log.Errorf("checkQuantificationNameExists: Failed to list user quants, allowing quant creation for name: %v, scan: %v. Error was %v", name, scanId, err)
		return false
	}

	// Check if it exists
	for _, item := range items {
		if item.Params.UserParams.Name == name {
			return true
		}
	}

	// Not found!
	return false
}
