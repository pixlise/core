package sdfToRSI

import "fmt"

func Example_processCentroidLine1() {
	sliNum, x, y, intensity, err := processCentroidLine1(7, "", "3 -- pixel x,y,intensity: [0x7ab6fdcf] [0x37d1f047] [0x03f1] |     411.7626     187.3011")
	fmt.Printf("%v,%v,%v,%v,%v\n", sliNum, x, y, intensity, err)

	// Output:
	// 3,411.7626,187.3011,1009,<nil>
}

func Example_processCentroidLine2() {
	x, y, z, err := processCentroidLine2(7, "2022-301T17:28:38 :    2 CenSLI_struct  0 -- position x,y,z: [0xffffdbb9] [0xffffc999] [0x0000e8e3]  |    -0.009287    -0.013927     0.059619", 0)
	fmt.Printf("%v,%v,%v,%v\n", x, y, z, err)

	// Output:
	// -0.009287,-0.013927,0.059619,<nil>
}

func Example_processCentroidLine3() {
	id, res, err := processCentroidLine3(7, "2022-301T17:28:38 :    2 CenSLI_struct  0 -- ID: 0x4a, Residual: 0x03", 0)
	fmt.Printf("%v,%v,%v\n", id, res, err)

	// Output:
	// 74,3,<nil>
}
