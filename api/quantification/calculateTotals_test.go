package quantification

import (
	"fmt"
	"os"

	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/proto"
)

func readQuantificationFile(filePath string) (*protos.Quantification, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	quantPB := &protos.Quantification{}
	err = proto.Unmarshal(fileBytes, quantPB)
	if err != nil {
		return nil, err
	}
	return quantPB, nil
}

func Example_calculateTotals_AB_NeedsCombined() {
	q, err := readQuantificationFile("./testdata/AB.bin")
	fmt.Printf("%v\n", err)

	if err == nil {
		result, err := calculateTotals(q, []int{90, 91, 95})

		fmt.Printf("%v|%v\n", result, err)
	}

	// Output:
	// <nil>
	// map[]|Quantification must be for Combined detectors
}

func Example_calculateTotals_NoPMC() {
	q, err := readQuantificationFile("./testdata/combined.bin")
	fmt.Printf("%v\n", err)

	if err == nil {
		result, err := calculateTotals(q, []int{68590, 68591, 68595})

		fmt.Printf("%v|%v\n", result, err)
	}

	// Output:
	// <nil>
	// map[]|Quantification had no valid data for ROI PMCs
}

func Example_calculateTotals_Success() {
	q, err := readQuantificationFile("./testdata/combined.bin")
	fmt.Printf("%v\n", err)

	if err == nil {
		result, err := calculateTotals(q, []int{90, 91, 95})

		fmt.Printf("%v|%v\n", result, err)
	}

	// Output:
	// <nil>
	// map[CaO_%:7.5057006 FeO-T_%:10.621034 SiO2_%:41.48377 TiO2_%:0.7424]|<nil>
}
