package coreg

import "fmt"

func Example_readWarpedImageTransform() {
	xform, name, err := readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	xform, name, err = readWarpedImageTransform("warped-zoom_1.1359177671479777-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01-A.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	xform, name, err = readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	xform, name, err = readWarpedImageTransform("warped-zoom_1.1359177671479777-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	xform, name, err = readWarpedImageTransform("warped-win_216_186_167_183-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	xform, name, err = readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RASS_N0450000SRLC11373_0000LMJ01.png")
	fmt.Printf("%+v|%v|%v\n", xform, name, err)

	// Output:
	// xOffset:40  yOffset:519  xScale:4.478153  yScale:4.478153|coreg-40_519-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png|<nil>
	// xOffset:186  yOffset:216  xScale:1.1359178  yScale:1.1359178|coreg-186_216-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01-A.png|<nil>
	// xOffset:40  yOffset:519  xScale:4.478153  yScale:4.478153|coreg-40_519-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01.png|<nil>
	// xOffset:186  yOffset:216  xScale:1.1359178  yScale:1.1359178|coreg-186_216-PCB_0921_0748739251_000RAS_N045000032302746300020LUJ01.png|<nil>
	// <nil>||Warped image name does not have expected components
	// <nil>||Failed to find GDS file name section in image name: SC3_0921_0748732957_027RASS_N0450000SRLC11373_0000LMJ01. Error: Failed to parse meta from file name
}
