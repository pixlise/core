package wsHelpers

import (
	"fmt"
	"log"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Here we merge existing scan data with spectra from the temp dir and the last downloaded set
// to form a new scan file that can be saved containing everything
// NOTE: We're just replacing the spectra in the passed in experiment data (that is expected to
//
//	be modifiable as it was likely just loaded from a file anyway!)
func Example_wsHelpers_MergeSpectra() {
	b, err := os.ReadFile("./test-data/naltsos.bin")
	if err != nil {
		log.Fatal(err)
	}

	exprPB := &protos.Experiment{}
	if err = proto.Unmarshal(b, exprPB); err != nil {
		log.Fatal(err)
	}

	counts := []uint32{}
	for c := 0; c < 4096; c++ {
		counts = append(counts, uint32(c))
	}

	s := []*protos.Spectra{}
	for c := 0; c < len(exprPB.Locations); c++ {
		abMod := []*protos.Spectrum{}

		if len(exprPB.Locations[c].PseudoIntensities) > 0 {
			abMod = append(abMod, &protos.Spectrum{
				Detector: "A",
				Type:     protos.SpectrumType_SPECTRUM_NORMAL,
				Counts:   counts,
				MaxCount: 4095,
				Meta: map[int32]*protos.ScanMetaDataItem{
					119: { // LIVETIME
						Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 8.8},
					},
					120: { // OFFSET
						Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: -0.95},
					},
					123: { // REALTIME
						Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 8.819},
					},
					124: { // XPERCHAN
						Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 11},
					},
				},
			},
				&protos.Spectrum{
					Detector: "B'",
					Type:     protos.SpectrumType_SPECTRUM_NORMAL,
					Counts:   counts,
					MaxCount: 4095,
					Meta: map[int32]*protos.ScanMetaDataItem{
						119: { // LIVETIME
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 9.8},
						},
						120: { // OFFSET
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: -0.85},
						},
						123: { // REALTIME
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 9.818},
						},
						124: {
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 11.1},
						},
					},
				},
				&protos.Spectrum{
					Detector: "Another",
					Type:     protos.SpectrumType_SPECTRUM_NORMAL,
					Counts:   counts,
					MaxCount: 4095,
					Meta: map[int32]*protos.ScanMetaDataItem{
						119: { // LIVETIME
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 10.8},
						},
						120: { // OFFSET
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: -0.75},
						},
						123: { // REALTIME
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 10.881},
						},
						124: { // XPERCHAN
							Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 11.2},
						},
					},
				},
			)
		}

		s = append(s, &protos.Spectra{
			Spectra: abMod,
		})
	}

	err = MergeSpectra(exprPB, s, true)
	if err != nil {
		log.Fatal(err)
	}

	// Run through and get stats of what we have
	fmt.Printf("Locations: %v\n", len(exprPB.Locations))
	dets := map[string]bool{}
	for _, loc := range exprPB.Locations {
		for _, d := range loc.Detectors {
			for _, v := range d.Meta {
				if v.LabelIdx == 118 {
					dets[v.Svalue] = true
					break
				}
			}
		}
	}

	detList := utils.GetMapKeys(dets)
	slices.Sort(detList)

	fmt.Printf("Detectors: %v\n", strings.Join(detList, ","))

	for i, loc := range exprPB.Locations {
		if i > 5 && i != 130 {
			continue
		}

		for c, d := range loc.Detectors {
			meta := []string{}
			for _, m := range d.Meta {
				b, _ := protojson.Marshal(m)
				x := string(b)

				meta = append(meta, "  "+strings.ReplaceAll(x, " ", ""))
			}
			sort.Strings(meta)
			fmt.Printf("loc %v, detector %v max: %v, meta:\n%v\n", i, c, d.SpectrumMax, strings.Join(meta, "\n"))
		}
	}

	// Output:
	// Locations: 133
	// Detectors: A,A',Another,B,B'
	// loc 5, detector 0 max: 4560, meta:
	//   {"ivalue":678032243}
	//   {"labelIdx":118,"svalue":"A"}
	//   {"labelIdx":119,"fvalue":13.787305}
	//   {"labelIdx":120,"fvalue":-18.5}
	//   {"labelIdx":121,"ivalue":93}
	//   {"labelIdx":122,"svalue":"Normal"}
	//   {"labelIdx":123,"fvalue":15}
	//   {"labelIdx":124,"fvalue":7.862}
	// loc 5, detector 1 max: 5330, meta:
	//   {"ivalue":678032244}
	//   {"labelIdx":118,"svalue":"B"}
	//   {"labelIdx":119,"fvalue":13.760527}
	//   {"labelIdx":120,"fvalue":-22.4}
	//   {"labelIdx":121,"ivalue":93}
	//   {"labelIdx":122,"svalue":"Normal"}
	//   {"labelIdx":123,"fvalue":15}
	//   {"labelIdx":124,"fvalue":7.881}
	// loc 5, detector 2 max: 4095, meta:
	//   {"labelIdx":118,"svalue":"A'"}
	//   {"labelIdx":119,"fvalue":8.8}
	//   {"labelIdx":120,"fvalue":-0.95}
	//   {"labelIdx":122,"svalue":"Normal"}
	//   {"labelIdx":123,"fvalue":8.819}
	//   {"labelIdx":124,"fvalue":11}
	// loc 5, detector 3 max: 4095, meta:
	//   {"labelIdx":118,"svalue":"B'"}
	//   {"labelIdx":119,"fvalue":9.8}
	//   {"labelIdx":120,"fvalue":-0.85}
	//   {"labelIdx":122,"svalue":"Normal"}
	//   {"labelIdx":123,"fvalue":9.818}
	//   {"labelIdx":124,"fvalue":11.1}
	// loc 5, detector 4 max: 4095, meta:
	//   {"labelIdx":118,"svalue":"Another"}
	//   {"labelIdx":119,"fvalue":10.8}
	//   {"labelIdx":120,"fvalue":-0.75}
	//   {"labelIdx":122,"svalue":"Normal"}
	//   {"labelIdx":123,"fvalue":10.881}
	//   {"labelIdx":124,"fvalue":11.2}
	// loc 130, detector 0 max: 367901, meta:
	//   {"labelIdx":118,"svalue":"A"}
	//   {"labelIdx":119,"fvalue":1712.4017}
	//   {"labelIdx":120,"fvalue":-11.8}
	//   {"labelIdx":121,"ivalue":218}
	//   {"labelIdx":122,"svalue":"BulkSum"}
	//   {"labelIdx":123,"fvalue":1815}
	//   {"labelIdx":124,"fvalue":7.9226}
	//   {"labelIdx":125,"svalue":"YY"}
	//   {"labelIdx":126,"svalue":"EMSA/MASspectraldatafile"}
	//   {"labelIdx":127,"svalue":"2"}
	//   {"labelIdx":128,"svalue":"4096"}
	//   {"labelIdx":129,"svalue":"PIXLFlightModel"}
	//   {"labelIdx":130,"svalue":"XRF"}
	//   {"labelIdx":131,"svalue":"N/A"}
	//   {"labelIdx":132,"svalue":"TC202v2.0PIXL"}
	//   {"labelIdx":133,"svalue":"eV"}
	//   {"labelIdx":134,"svalue":"-1.032"}
	//   {"labelIdx":135,"svalue":"COUNTS"}
	// loc 130, detector 1 max: 353738, meta:
	//   {"labelIdx":118,"svalue":"B"}
	//   {"labelIdx":119,"fvalue":1712.503}
	//   {"labelIdx":120,"fvalue":-13.2}
	//   {"labelIdx":121,"ivalue":218}
	//   {"labelIdx":122,"svalue":"BulkSum"}
	//   {"labelIdx":123,"fvalue":1815}
	//   {"labelIdx":124,"fvalue":7.9273}
	//   {"labelIdx":125,"svalue":"YY"}
	//   {"labelIdx":126,"svalue":"EMSA/MASspectraldatafile"}
	//   {"labelIdx":127,"svalue":"2"}
	//   {"labelIdx":128,"svalue":"4096"}
	//   {"labelIdx":129,"svalue":"PIXLFlightModel"}
	//   {"labelIdx":130,"svalue":"XRF"}
	//   {"labelIdx":131,"svalue":"N/A"}
	//   {"labelIdx":132,"svalue":"TC202v2.0PIXL"}
	//   {"labelIdx":133,"svalue":"eV"}
	//   {"labelIdx":134,"svalue":"-1.032"}
	//   {"labelIdx":135,"svalue":"COUNTS"}
	// loc 130, detector 2 max: 7017, meta:
	//   {"labelIdx":118,"svalue":"A"}
	//   {"labelIdx":119,"fvalue":14.293016}
	//   {"labelIdx":120,"fvalue":-11.8}
	//   {"labelIdx":121,"ivalue":218}
	//   {"labelIdx":122,"svalue":"MaxValue"}
	//   {"labelIdx":123,"fvalue":15}
	//   {"labelIdx":124,"fvalue":7.9226}
	//   {"labelIdx":125,"svalue":"YY"}
	//   {"labelIdx":126,"svalue":"EMSA/MASspectraldatafile"}
	//   {"labelIdx":127,"svalue":"2"}
	//   {"labelIdx":128,"svalue":"4096"}
	//   {"labelIdx":129,"svalue":"PIXLFlightModel"}
	//   {"labelIdx":130,"svalue":"XRF"}
	//   {"labelIdx":131,"svalue":"N/A"}
	//   {"labelIdx":132,"svalue":"TC202v2.0PIXL"}
	//   {"labelIdx":133,"svalue":"eV"}
	//   {"labelIdx":134,"svalue":"-1.032"}
	//   {"labelIdx":135,"svalue":"COUNTS"}
	// loc 130, detector 3 max: 6786, meta:
	//   {"labelIdx":118,"svalue":"B"}
	//   {"labelIdx":119,"fvalue":14.27207}
	//   {"labelIdx":120,"fvalue":-13.2}
	//   {"labelIdx":121,"ivalue":218}
	//   {"labelIdx":122,"svalue":"MaxValue"}
	//   {"labelIdx":123,"fvalue":15}
	//   {"labelIdx":124,"fvalue":7.9273}
	//   {"labelIdx":125,"svalue":"YY"}
	//   {"labelIdx":126,"svalue":"EMSA/MASspectraldatafile"}
	//   {"labelIdx":127,"svalue":"2"}
	//   {"labelIdx":128,"svalue":"4096"}
	//   {"labelIdx":129,"svalue":"PIXLFlightModel"}
	//   {"labelIdx":130,"svalue":"XRF"}
	//   {"labelIdx":131,"svalue":"N/A"}
	//   {"labelIdx":132,"svalue":"TC202v2.0PIXL"}
	//   {"labelIdx":133,"svalue":"eV"}
	//   {"labelIdx":134,"svalue":"-1.032"}
	//   {"labelIdx":135,"svalue":"COUNTS"}
	// loc 130, detector 4 max: 495495, meta:
	//   {"labelIdx":118,"svalue":"A'"}
	//   {"labelIdx":119,"fvalue":1064.7992}
	//   {"labelIdx":120,"fvalue":-0.949999}
	//   {"labelIdx":122,"svalue":"BulkSum"}
	//   {"labelIdx":123,"fvalue":1067.0972}
	//   {"labelIdx":124,"fvalue":11}
	// loc 130, detector 5 max: 495495, meta:
	//   {"labelIdx":118,"svalue":"Another"}
	//   {"labelIdx":119,"fvalue":1306.8005}
	//   {"labelIdx":120,"fvalue":-0.75}
	//   {"labelIdx":122,"svalue":"BulkSum"}
	//   {"labelIdx":123,"fvalue":1316.5999}
	//   {"labelIdx":124,"fvalue":11.199994}
	// loc 130, detector 6 max: 495495, meta:
	//   {"labelIdx":118,"svalue":"B'"}
	//   {"labelIdx":119,"fvalue":1185.7999}
	//   {"labelIdx":120,"fvalue":-0.84999907}
	//   {"labelIdx":122,"svalue":"BulkSum"}
	//   {"labelIdx":123,"fvalue":1187.9773}
	//   {"labelIdx":124,"fvalue":11.099987}

}
