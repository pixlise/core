package utils

import "fmt"

func Example_utils_ZeroRunDecode_u32() {
	data := []uint32{0, 2, 4, 2, 0, 4, 3, 0, 1}

	decoded := ZeroRunDecode(data)

	fmt.Printf("%+v\n", decoded)

	// Output:
	// [0 0 4 2 0 0 0 0 3 0]
}

func Example_utils_ZeroRunDecode_i32() {
	data := []int32{0, 2, 4, 2, 0, 4, 3, 0, 1}

	decoded := ZeroRunDecode(data)

	fmt.Printf("%+v\n", decoded)

	// Output:
	// [0 0 4 2 0 0 0 0 3 0]
}

func Example_utils_ZeroRunEncode_u32() {
	data := []uint32{0, 0, 4, 2, 0, 0, 0, 0, 3, 0}

	encoded := ZeroRunEncode(data)

	fmt.Printf("%+v\n", encoded)

	// Output:
	// [0 2 4 2 0 4 3 0 1]
}

func Example_utils_ZeroRunEncode_i32() {
	data := []int32{0, 0, 4, 2, 0, 0, 0, 0, 3, 0}

	encoded := ZeroRunEncode(data)

	fmt.Printf("%+v\n", encoded)

	// Output:
	// [0 2 4 2 0 4 3 0 1]
}

/*
func Example_RunLengthEncode() {

	data := []int{0, 0, 4, 2, 2, 2, 3, 0}

	encoded := RunLengthEncode(data)

	fmt.Printf("%+v\n", encoded)

	// Output:
	// [0 2 4 1 2 3 3 1 0 1]
}
*/
