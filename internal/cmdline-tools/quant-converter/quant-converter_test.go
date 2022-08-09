package main

import (
	"fmt"
)

func Example_makeExpectedMetaList() {
	fmt.Println(makeExpectedMetaList([]string{"column A", "column B", "column C", "column D"}, []string{"column B", "column C"}))
	// Output: [column A column D] <nil>
}
