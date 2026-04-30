package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleImageScanEntryDisplayElementsGetReq(req *protos.ImageScanEntryDisplayElementsGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageScanEntryDisplayElementsGetResp, error) {
	// Look up all the data required to calculate these polygons
	// NOTE: we should probably store these longer term as we start dealing with larger data, but for now generation is fine as it was fine for years done client-side!

	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, nil, hctx)
	if err != nil {
		return nil, err
	}

	entries, err := scan.ReadScanEntries(exprPB, indexes)
	if err != nil {
		return nil, err
	}

	beams := scan.ReadXYZ(exprPB, indexes)

	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName)
	scanResult := coll.FindOne(context.TODO(), bson.M{"_id": req.ScanId}, options.FindOne())
	if scanResult.Err() != nil {
		return nil, errorwithstatus.MakeNotFoundError(req.ScanId)
	}

	scanItem := &protos.ScanItem{}
	err = scanResult.Decode(scanItem)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode scan: %v. Error: %v", req.ScanId, err)
	}

	var locs *protos.ImageLocations
	if len(req.ImageName) <= 0 && len(req.ScanId) > 0 {
		// We have no image, so we generate coordinates
		// NOTE: empty image name implies this won't write to DB
		locs, err = wsHelpers.GenerateIJs("", req.ScanId, scanItem.Instrument, hctx.Svcs)
		if err != nil {
			return nil, err
		}
	} else {
		locs, err = wsHelpers.GetImageBeamLocations(hctx, req.ImageName, map[string]uint32{req.ScanId: req.BeamVersion})
		if err != nil {
			return nil, err
		}
	}

	if len(locs.LocationPerScan) <= 0 {
		return nil, fmt.Errorf("Image %v associated with scan: %v has no beam locations", req.ImageName, req.ScanId)
	}

	cfg, err := piquant.GetDetectorConfig(scanItem.InstrumentConfig, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}
	// NOTE: we aren't updating cfg.ElevAngle here, but we don't need it in this anyway

	return scan.GeneratePolygons(req.ImageName, scanItem, entries, beams, &locs.LocationPerScan[0].Locations, cfg)
}
