package client

import "fmt"

func Example_zeroRunDecode() {
	data := []uint32{0, 2, 4, 2, 0, 4, 3, 0, 1}

	decoded := zeroRunDecode(data)

	fmt.Printf("%+v\n", decoded)

	// Output:
	// [0 0 4 2 0 0 0 0 3 0]
}
