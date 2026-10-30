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

func Example_wsHelpers_GenerateAggregateSpectrum() {
	b, err := os.ReadFile("./test-data/naltsos.bin")
	if err != nil {
		log.Fatal(err)
	}

	exprPB := &protos.Experiment{}
	if err = proto.Unmarshal(b, exprPB); err != nil {
		log.Fatal(err)
	}

	// Generate a few aggregate spectra out of this
	fmt.Println("Checking error cases...")
	bulks, maxs, err := GenerateAggregateSpectrum([]uint32{3, 10, 15, 20, 900}, []string{"A", "F"}, protos.SpectrumType_SPECTRUM_NORMAL, true, true, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15, 20, 900}, []string{"A", "F"}, protos.SpectrumType_SPECTRUM_NORMAL, true, true, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15, 20, 900}, []string{"A"}, protos.SpectrumType_SPECTRUM_NORMAL, true, true, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15, 20}, []string{"A"}, protos.SpectrumType_SPECTRUM_DWELL, true, true, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	fmt.Println("Checking valid request items found...")
	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15}, []string{"A"}, protos.SpectrumType_SPECTRUM_NORMAL, true, false, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15}, []string{"A", "B"}, protos.SpectrumType_SPECTRUM_NORMAL, true, false, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	bulks, maxs, err = GenerateAggregateSpectrum([]uint32{10, 15}, []string{"A"}, protos.SpectrumType_SPECTRUM_NORMAL, false, true, exprPB)
	fmt.Printf("%v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))

	indexLists := [][]uint32{{6}, {8}, {100}, {6, 8, 100}, {130}, {130}, {}}
	addToIdx := len(indexLists) - 1
	for i, loc := range exprPB.Locations {
		if loc.PseudoIntensities != nil {
			indexLists[addToIdx] = append(indexLists[addToIdx], uint32(i))
		}
	}
	types := []protos.SpectrumType{
		protos.SpectrumType_SPECTRUM_NORMAL,
		protos.SpectrumType_SPECTRUM_NORMAL,
		protos.SpectrumType_SPECTRUM_NORMAL,
		protos.SpectrumType_SPECTRUM_NORMAL,
		protos.SpectrumType_SPECTRUM_BULK,
		protos.SpectrumType_SPECTRUM_MAX,
		protos.SpectrumType_SPECTRUM_NORMAL}

	for c, l := range indexLists {
		fmt.Printf("Checking spectra outputs for %v, %v...\n", types[c], l)
		bulks, maxs, err = GenerateAggregateSpectrum(l, []string{"A"}, types[c], true, len(l) > 1 && types[c] == protos.SpectrumType_SPECTRUM_NORMAL, exprPB)
		fmt.Printf("- %v|%v|%v\n", err, utils.GetMapKeys(bulks), utils.GetMapKeys(maxs))
		if len(bulks) > 0 {
			fmt.Printf("- bulk-A: %v|%v|counts=%v|maxcount=%v, meta: %v\n", bulks["A"].Detector, bulks["A"].Type, len(bulks["A"].Counts), bulks["A"].MaxCount, bulks["A"].Meta)
			decodedCounts := utils.ZeroRunDecode(bulks["A"].Counts)
			fmt.Printf("- bulk-A counts(%v): %v ... %v\n", len(decodedCounts), decodedCounts[100:106], decodedCounts[400:406])
		}
		if len(maxs) > 0 {
			fmt.Printf("- max-A: %v|%v|counts=%v|maxcount=%v, meta: %v\n", maxs["A"].Detector, maxs["A"].Type, len(maxs["A"].Counts), maxs["A"].MaxCount, maxs["A"].Meta)
			decodedCounts := utils.ZeroRunDecode(maxs["A"].Counts)
			fmt.Printf("- max-A counts(%v): %v ... %v\n", len(decodedCounts), decodedCounts[100:106], decodedCounts[400:406])
		}
	}

	// Output:
	// Checking error cases...
	// Requested detector A, read type Normal not found at location 3|[]|[]
	// Requested detector F, read type Normal not found at location 10|[]|[]
	// Invalid location 900, location count is 133|[]|[]
	// Requested detector A, read type Dwell not found at location 10|[]|[]
	// Checking valid request items found...
	// <nil>|[A]|[]
	// <nil>|[A B]|[]
	// <nil>|[]|[A]
	// Checking spectra outputs for SPECTRUM_NORMAL, [6]...
	// - <nil>|[A]|[]
	// - bulk-A: A|SPECTRUM_NORMAL|counts=1974|maxcount=783, meta: map[119:fvalue:14.121621 120:fvalue:-18.5 123:fvalue:15 124:fvalue:7.862]
	// - bulk-A counts(4096): [0 1 0 0 0 1] ... [4 1 3 7 8 9]
	// Checking spectra outputs for SPECTRUM_NORMAL, [8]...
	// - <nil>|[A]|[]
	// - bulk-A: A|SPECTRUM_NORMAL|counts=2522|maxcount=5576, meta: map[119:fvalue:13.708748 120:fvalue:-18.5 123:fvalue:15 124:fvalue:7.862]
	// - bulk-A counts(4096): [1 0 3 1 3 0] ... [13 13 4 9 3 11]
	// Checking spectra outputs for SPECTRUM_NORMAL, [100]...
	// - <nil>|[A]|[]
	// - bulk-A: A|SPECTRUM_NORMAL|counts=2649|maxcount=3014, meta: map[119:fvalue:13.888511 120:fvalue:-18.5 123:fvalue:15 124:fvalue:7.862]
	// - bulk-A counts(4096): [1 1 1 2 4 2] ... [16 7 8 10 12 8]
	// Checking spectra outputs for SPECTRUM_NORMAL, [6 8 100]...
	// - <nil>|[A]|[A]
	// - bulk-A: A|SPECTRUM_NORMAL|counts=2793|maxcount=9355, meta: map[119:fvalue:41.71888 120:fvalue:-18.5 123:fvalue:45 124:fvalue:7.862]
	// - bulk-A counts(4096): [2 2 4 3 7 3] ... [33 21 15 26 23 28]
	// - max-A: A|SPECTRUM_NORMAL|counts=2793|maxcount=5576, meta: map[119:fvalue:13.906293 120:fvalue:-18.5 123:fvalue:15 124:fvalue:7.862]
	// - max-A counts(4096): [1 1 3 2 4 2] ... [16 13 8 10 12 11]
	// Checking spectra outputs for SPECTRUM_BULK, [130]...
	// - <nil>|[A]|[]
	// - bulk-A: A|SPECTRUM_BULK|counts=3343|maxcount=367901, meta: map[119:fvalue:1712.4017 120:fvalue:-11.8 123:fvalue:1815 124:fvalue:7.9226]
	// - bulk-A counts(4096): [95 119 129 160 178 178] ... [1074 1127 1146 1196 1307 1406]
	// Checking spectra outputs for SPECTRUM_MAX, [130]...
	// - <nil>|[A]|[]
	// - bulk-A: A|SPECTRUM_MAX|counts=3343|maxcount=7017, meta: map[119:fvalue:14.293016 120:fvalue:-11.8 123:fvalue:15 124:fvalue:7.9226]
	// - bulk-A counts(4096): [4 3 4 5 5 5] ... [17 27 20 17 20 24]
	// Checking spectra outputs for SPECTRUM_NORMAL, [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 80 81 82 83 84 85 87 88 89 90 91 92 93 94 95 96 97 98 99 100 101 102 103 104 105 106 107 108 109 110 111 112 113 114 115 116 117 118 119 120 121 122 123 124 125 126 128]...
	// - <nil>|[A]|[A]
	// - bulk-A: A|SPECTRUM_NORMAL|counts=3343|maxcount=367901, meta: map[119:fvalue:1680.6581 120:fvalue:-18.5 123:fvalue:1815 124:fvalue:7.862]
	// - bulk-A counts(4096): [95 119 129 160 178 178] ... [1074 1127 1146 1196 1307 1406]
	// - max-A: A|SPECTRUM_NORMAL|counts=3343|maxcount=7017, meta: map[119:fvalue:13.889736 120:fvalue:-18.5 123:fvalue:15 124:fvalue:7.862]
	// - max-A counts(4096): [4 3 4 5 5 5] ... [17 27 20 17 20 24]
}
