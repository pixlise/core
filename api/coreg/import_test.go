package coreg

import "fmt"

func Example_readWarpedImageTransform() {
	xform, err := readWarpedImageTransform("warped-zoom_4.478153138946561-win_519_40_1232_1183-SN100D0-SC3_0921_0748732957_027RAS_N0450000SRLC11373_0000LMJ01-A.png")
	fmt.Printf("%+v|%v", xform, err)

	// Output:
	// xOffset:519  yOffset:1232  xScale:0.22330634  yScale:0.22330634|<nil>
}
