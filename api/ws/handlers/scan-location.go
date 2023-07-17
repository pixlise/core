package wsHandler

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleScanLocationReq(req *protos.ScanLocationReq, hctx wsHelpers.HandlerContext) (*protos.ScanLocationResp, error) {
	exprPB, startLocIdx, endLocIdx, err := beginDatasetFileReq(req.ScanId, req.StartingLocation, req.LocationCount, hctx)
	if err != nil {
		return nil, err
	}

	locs := []*protos.Location{}
	for c := startLocIdx; c < endLocIdx; c++ {
		loc := exprPB.Locations[c]

		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert PMC %v to int while reading scan location %v", loc.Id, c)
		}

		locSave := &protos.Location{
			Id: int32(pmc),
		}

		// Add physical location if we have one defined
		if loc.Beam != nil {
			locSave.X = loc.Beam.X
			locSave.Y = loc.Beam.Y
			locSave.Z = loc.Beam.Z
		}

		// Add meta fields
		if len(loc.Meta) > 0 {
			locSave.Meta = map[int32]*protos.ScanMetaDataItem{}
			for _, meta := range loc.Meta {
				if meta.LabelIdx >= int32(len(exprPB.MetaLabels)) {
					return nil, fmt.Errorf("LabelIdx %v out of range when reading meta for location at idx: %v", meta.LabelIdx, c)
				}

				label := exprPB.MetaLabels[meta.LabelIdx]

				// Check that this slot isn't taken
				if _, ok := locSave.Meta[meta.LabelIdx]; ok {
					return nil, fmt.Errorf("Conflicting label index %v when reading spectrum meta for spectrum at idx: %v", meta.LabelIdx, c)
				}

				mSave := &protos.ScanMetaDataItem{}
				if t := exprPB.MetaTypes[meta.LabelIdx]; t == protos.Experiment_MT_STRING {
					mSave.Value = &protos.ScanMetaDataItem_Svalue{Svalue: meta.Svalue}
				} else if t == protos.Experiment_MT_INT {
					mSave.Value = &protos.ScanMetaDataItem_Ivalue{Ivalue: meta.Ivalue}
				} else if t == protos.Experiment_MT_FLOAT {
					mSave.Value = &protos.ScanMetaDataItem_Fvalue{Fvalue: meta.Fvalue}
				} else {
					return nil, fmt.Errorf("Unknown type %v for meta label: %v when reading spectrum type for spectrum at idx: %v", t, label, c)
				}

				locSave.Meta[meta.LabelIdx] = mSave
			}
		}

		locs = append(locs, locSave)
	}

	// Read out the scan data
	return &protos.ScanLocationResp{
		Locations: locs,
	}, nil
}
