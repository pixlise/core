package sessionuser

import "github.com/pixlise/core/v4/core/utils"

var PIXLISESystemUserId = "PIXLISEImport"
var JPLImport = "JPLImport"
var SBUImport = "SBUImport"
var BrukerImport = "BrukerImport"

var syntheticUserIds = []string{PIXLISESystemUserId, JPLImport, SBUImport, BrukerImport}

func IsSystemUser(id string) bool {
	return utils.ItemInSlice(id, syntheticUserIds)
}
