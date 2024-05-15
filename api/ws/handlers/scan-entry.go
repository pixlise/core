package wsHandler

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleScanEntryReq(req *protos.ScanEntryReq, hctx wsHelpers.HandlerContext) ([]*protos.ScanEntryResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	sclkIdx := -1
	readtypeIdx := -1
	for c, label := range exprPB.MetaLabels {
		if label == "SCLK" {
			sclkIdx = c
		} else if label == "READTYPE" {
			readtypeIdx = c
		}

		if readtypeIdx > -1 && sclkIdx > -1 {
			break
		}
	}

	entries := []*protos.ScanEntry{}
	for _, c := range indexes {
		loc := exprPB.Locations[c]

		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert PMC %v to int while reading scan location %v", loc.Id, c)
		}

		timestamp := uint32(0)
		for _, meta := range loc.Meta {
			if meta.LabelIdx == int32(sclkIdx) {
				timestamp = uint32(meta.Ivalue)
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

		locSave.NormalSpectra = 0
		locSave.DwellSpectra = 0
		locSave.BulkSpectra = 0
		locSave.MaxSpectra = 0

		if loc.Detectors != nil {
			for _, detector := range loc.Detectors {
				for _, m := range detector.Meta {
					if m.LabelIdx == int32(readtypeIdx) {
						// Verify type
						if t := exprPB.MetaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
							// These are hard-coded string values
							switch m.Svalue {
							case "BulkSum":
								locSave.BulkSpectra++
							case "MaxValue":
								locSave.MaxSpectra++
							case "Normal":
								locSave.NormalSpectra++
							case "Dwell":
								locSave.DwellSpectra++
							}
						}
					}
				}
			}
		}

		entries = append(entries, locSave)
	}

	return []*protos.ScanEntryResp{&protos.ScanEntryResp{
		Entries: entries,
	}}, nil
}
