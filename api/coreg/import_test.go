package coreg

import "fmt"

func Example_readWarpedImageTransform() {
	xform, name, err := readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png")
	fmt.Printf("%+v|%v|%v", xform, name, err)

	// Output:
	// xOffset:40 yOffset:519 xScale:4.478153 yScale:4.478153|coreg-40_519-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png|<nil>
}
