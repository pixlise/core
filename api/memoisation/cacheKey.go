package memoisation

import (
	"fmt"

	protos "github.com/pixlise/core/v4/generated-protos"
)

// Keys are of the form:
// {"scanId":"602735105","exprId":"q2ns80oc4452eldt","quantId":"quant-aqpxxfk6i05gcsy3","roiId":"AllPoints-602735105","units":0},Resp:false,exprMod:1772129285,spectra:3298,90,0
// So we need scan summary details and the expression last modified time
func MakeCacheKey(scanItem *protos.ScanItem, exprItem *protos.DataExpression, quantId, roiId string, units protos.DataUnit) (string, error) {
	normalSpectraCount := scanItem.ContentCounts["NormalSpectra"]
	dwellSpectraCount := scanItem.ContentCounts["DwellSpectra"]

	spectrumTimeStamp := 0 // Comes from SpectrumResp.timeStampUnixSec, seems to always be 0 for now??

	memCacheKey := fmt.Sprintf(
		`{"scanId":"%v","exprId":"%v","quantId":"%v","roiId":"%v","units":%v},Resp:false,exprMod:%v,spectra:%v,%v,%v`,
		scanItem.Id,
		exprItem.Id,
		quantId,
		roiId,
		units.Number(),
		exprItem.ModifiedUnixSec,
		normalSpectraCount,
		dwellSpectraCount,
		spectrumTimeStamp,
	)

	return memCacheKey, nil
}
