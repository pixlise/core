package wsHandler

import (
	"fmt"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleScanEntryMetadataReq(req *protos.ScanEntryMetadataReq, hctx wsHelpers.HandlerContext) ([]*protos.ScanEntryMetadataResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}
	/*
		sclkIdx := -1
		for c, label := range exprPB.MetaLabels {
			if label == "SCLK" {
				sclkIdx = c
				break
			}
		}
	*/
	entryMetas := []*protos.ScanEntryMetadata{}
	for _, c := range indexes {
		loc := exprPB.Locations[c]
		metaSave := &protos.ScanEntryMetadata{Meta: map[int32]*protos.ScanMetaDataItem{}}

		for _, meta := range loc.Meta {
			/*
					if meta.LabelIdx == int32(sclkIdx) {
				 		// We already included SCLK as a field on ScanEntry
						continue
					}
			*/

			if meta.LabelIdx >= int32(len(exprPB.MetaLabels)) {
				return nil, fmt.Errorf("LabelIdx %v out of range when reading meta for location at idx: %v", meta.LabelIdx, c)
			}

			label := exprPB.MetaLabels[meta.LabelIdx]

			// Check that this slot isn't taken
			if _, ok := metaSave.Meta[meta.LabelIdx]; ok {
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

			metaSave.Meta[meta.LabelIdx] = mSave
		}

		entryMetas = append(entryMetas, metaSave)
	}

	return []*protos.ScanEntryMetadataResp{&protos.ScanEntryMetadataResp{
		Entries: entryMetas,
	}}, nil
}
