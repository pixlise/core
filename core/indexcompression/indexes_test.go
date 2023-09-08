package indexcompression

import (
	"fmt"
	"math"
)

func Example_DecodeIndexList() {
	fmt.Println(DecodeIndexList([]int32{5}, 4))
	fmt.Println(DecodeIndexList([]int32{5}, 5))
	fmt.Println(DecodeIndexList([]int32{5}, 6))
	fmt.Println(DecodeIndexList([]int32{}, 1))
	fmt.Println(DecodeIndexList([]int32{1, 5, 7, 12, 4, 10, 14}, 50))
	fmt.Println(DecodeIndexList([]int32{1, 5, -1, 8, 12, 4, 10, 14}, 50))
	fmt.Println(DecodeIndexList([]int32{1, 5, -4, 8, 12, 4, 10, 14}, 50))
	fmt.Println(DecodeIndexList([]int32{-1, 5, -1, 8, 12, 4, 10, 14}, 50))
	fmt.Println(DecodeIndexList([]int32{1, 5, -1, 8, 12, 4, 10, 14, -1}, 50))
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, -1, 8}, 50))
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, 6, -1, 5}, 50))
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, 6, -1, 6}, 50))
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, 6, -1, 7}, 50))
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, 6, -1, 8}, 50))
	// We don't ensure there are no repeats or they're in order...
	fmt.Println(DecodeIndexList([]int32{1, -1, 4, 2, -1, 8, 11, 13, -1, 16}, 50))
	// Check that expansion going past array size is found
	fmt.Println(DecodeIndexList([]int32{1, -1, 12}, 10))
	// And specified value going past
	fmt.Println(DecodeIndexList([]int32{1, 3, 5, 12, 2}, 10))
	// Verify arraySizeOptional is optional!
	fmt.Println(DecodeIndexList([]int32{1, 3, 5, 12, 2}, -1))

	// Output:
	// [] index 5 out of bounds: 4
	// [] index 5 out of bounds: 5
	// [5] <nil>
	// [] <nil>
	// [1 5 7 12 4 10 14] <nil>
	// [1 5 6 7 8 12 4 10 14] <nil>
	// [] invalid index: -4
	// [] indexes start with -1
	// [] indexes end with -1
	// [1 2 3 4 5 6 7 8] <nil>
	// [] invalid range: 6->5
	// [] invalid range: 6->6
	// [] invalid range: 6->7
	// [1 2 3 4 6 7 8] <nil>
	// [1 2 3 4 2 3 4 5 6 7 8 11 13 14 15 16] <nil>
	// [] index 12 out of bounds: 10
	// [] index 12 out of bounds: 10
	// [1 3 5 12 2] <nil>
}

func Example_EncodeIndexList() {
	fmt.Println(EncodeIndexList([]uint32{9, 22, 8}))
	fmt.Println(EncodeIndexList([]uint32{9, 7, 9, 8}))
	fmt.Println(EncodeIndexList([]uint32{9, 7, 8, 7}))
	fmt.Println(EncodeIndexList([]uint32{9, 7, 8, 8}))
	fmt.Println(EncodeIndexList([]uint32{1002, 14, 1005, 15, 1003, 1004, 13, 16}))
	fmt.Println(EncodeIndexList([]uint32{7, 8}))
	fmt.Println(EncodeIndexList([]uint32{9, 7, 8}))
	fmt.Println(EncodeIndexList([]uint32{1, 3, 5, 12, 2}))
	fmt.Println(EncodeIndexList([]uint32{}))
	fmt.Println(EncodeIndexList([]uint32{7}))
	fmt.Println(EncodeIndexList([]uint32{7, 8, 2, 9, 10, 11, 12, 14, 13, 16}))
	fmt.Println(EncodeIndexList([]uint32{1002, 14, 1005, 15, 1003, 1004, 13, 1100, 16}))
	fmt.Println(EncodeIndexList([]int32{1002, 14, 1005, 15, 1003, 1004, 13, 1100, 16}))
	fmt.Println(EncodeIndexList([]uint32{9, math.MaxInt32, 8}))
	fmt.Println(EncodeIndexList([]uint32{9, math.MaxInt32 + 1, 8}))

	// Output:
	// [8 9 22] <nil>
	// [7 -1 9] <nil>
	// [7 -1 9] <nil>
	// [7 -1 9] <nil>
	// [13 -1 16 1002 -1 1005] <nil>
	// [7 8] <nil>
	// [7 -1 9] <nil>
	// [1 -1 3 5 12] <nil>
	// [] <nil>
	// [7] <nil>
	// [2 7 -1 14 16] <nil>
	// [13 -1 16 1002 -1 1005 1100] <nil>
	// [13 -1 16 1002 -1 1005 1100] <nil>
	// [8 9 2147483647] <nil>
	// [] index list had value > maxint
}
