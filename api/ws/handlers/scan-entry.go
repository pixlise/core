package wsHandler

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleScanEntryReq(req *protos.ScanEntryReq, hctx wsHelpers.HandlerContext) (*protos.ScanEntryResp, error) {
	exprPB, startLocIdx, endLocIdx, err := beginDatasetFileReq(req.ScanId, req.Entries.FirstEntryIndex, req.Entries.EntryCount, hctx)
	if err != nil {
		return nil, err
	}

	sclkIdx := -1
	for c, label := range exprPB.MetaLabels {
		if label == "SCLK" {
			sclkIdx = c
			break
		}
	}

	entries := []*protos.ScanEntry{}
	for c := startLocIdx; c < endLocIdx; c++ {
		loc := exprPB.Locations[c]

		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert PMC %v to int while reading scan location %v", loc.Id, c)
		}

		timestamp := int32(0)
		for _, meta := range loc.Meta {
			if meta.LabelIdx == int32(sclkIdx) {
				timestamp = meta.Ivalue
				break
			}
		}

		locSave := &protos.ScanEntry{
			Id:        int32(pmc),
			Timestamp: timestamp,
		}

		// Set the counts as needed
		if loc.Beam != nil {
			locSave.Location = true
		}
		if len(loc.ContextImage) > 0 {
			locSave.Images++
		}
		if loc.PseudoIntensities != nil {
			locSave.PseudoIntensities = true
		}
		if len(loc.Meta) > 0 {
			locSave.Meta = true
		}
		if loc.Detectors != nil {
			locSave.Spectra += uint32(len(loc.Detectors))
		}

		entries = append(entries, locSave)
	}

	return &protos.ScanEntryResp{
		Entries: entries,
	}, nil
}
