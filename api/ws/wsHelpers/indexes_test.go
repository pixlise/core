package wsHelpers

import "fmt"

func Example_MakeIndexList() {
	fmt.Println(MakeIndexList([]int32{5}, 4))
	fmt.Println(MakeIndexList([]int32{5}, 5))
	fmt.Println(MakeIndexList([]int32{5}, 6))
	fmt.Println(MakeIndexList([]int32{}, 1))
	fmt.Println(MakeIndexList([]int32{1, 5, 7, 12, 4, 10, 14}, 50))
	fmt.Println(MakeIndexList([]int32{1, 5, -1, 8, 12, 4, 10, 14}, 50))
	fmt.Println(MakeIndexList([]int32{1, 5, -4, 8, 12, 4, 10, 14}, 50))
	fmt.Println(MakeIndexList([]int32{-1, 5, -1, 8, 12, 4, 10, 14}, 50))
	fmt.Println(MakeIndexList([]int32{1, 5, -1, 8, 12, 4, 10, 14, -1}, 50))
	fmt.Println(MakeIndexList([]int32{1, -1, 4, -1, 8}, 50))
	fmt.Println(MakeIndexList([]int32{1, -1, 4, 6, -1, 5}, 50))
	fmt.Println(MakeIndexList([]int32{1, -1, 4, 6, -1, 6}, 50))
	fmt.Println(MakeIndexList([]int32{1, -1, 4, 6, -1, 7}, 50))
	fmt.Println(MakeIndexList([]int32{1, -1, 4, 6, -1, 8}, 50))
	// We don't ensure there are no repeats or they're in order...
	fmt.Println(MakeIndexList([]int32{1, -1, 4, 2, -1, 8, 11, 13, -1, 16}, 50))
	// Check that expansion going past array size is found
	fmt.Println(MakeIndexList([]int32{1, -1, 12}, 10))
	// And specified value going past
	fmt.Println(MakeIndexList([]int32{1, 3, 5, 12, 2}, 10))

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
}
