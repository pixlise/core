package quantification

import (
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func checkQuantificationNameExists(name string, scanId string, hctx wsHelpers.HandlerContext) bool {
	items, _, err := ListUserQuants(nil, hctx)
	if err != nil {
		hctx.Svcs.Log.Errorf("checkQuantificationNameExists: Failed to list user quants, allowing quant creation for name: %v, scan: %v. Error was %v", name, scanId, err)
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
