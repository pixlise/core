package coreg

import (
	"fmt"
	"log"

	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
)

func printWarpXform(xform *protos.ImageMatchTransform, name string, err error) {
	fmt.Printf("%v|%v\n", name, err)

	if b, err := protojson.Marshal(xform); err != nil {
		log.Fatalln(err)
	} else {
		// Proto isn't deterministic, so we read as JSON and write it back out again
		fmt.Printf("%v\n", utils.MakeDeterministicJSON(b, false))
	}
}

func Example_readWarpedImageTransform() {
	printWarpXform(readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png"))

	printWarpXform(readWarpedImageTransform("warped-zoom_1.1359177671479777-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01-A.png"))

	printWarpXform(readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01.png"))

	printWarpXform(readWarpedImageTransform("warped-zoom_1.1359177671479777-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png"))

	printWarpXform(readWarpedImageTransform("warped-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png"))

	printWarpXform(readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RASS_N0450000SRLC11373_0000LMJ01.png"))

	// Output:
	// coreg-40_519-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png|<nil>
	// {
	//  "xOffset": 40,
	//  "xScale": 4.478153138946561,
	//  "yOffset": 519,
	//  "yScale": 4.478153138946561
	// }
	// coreg-186_216-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01-A.png|<nil>
	// {
	//  "xOffset": 186,
	//  "xScale": 1.1359177671479777,
	//  "yOffset": 216,
	//  "yScale": 1.1359177671479777
	// }
	// coreg-40_519-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01.png|<nil>
	// {
	//  "xOffset": 40,
	//  "xScale": 4.478153138946561,
	//  "yOffset": 519,
	//  "yScale": 4.478153138946561
	// }
	// coreg-186_216-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png|<nil>
	// {
	//  "xOffset": 186,
	//  "xScale": 1.1359177671479777,
	//  "yOffset": 216,
	//  "yScale": 1.1359177671479777
	// }
	// |Warped image name does not have expected components
	// {}
	// |Failed to find GDS file name section in image name: SC3_0921_0748732957_027RASS_N0450000SRLC11373_0000LMJ01. Error: Failed to parse meta from file name
	// {}
}
