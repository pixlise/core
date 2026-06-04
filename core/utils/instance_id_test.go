package utils

import (
	"fmt"
	"strings"
)

func Example_utils_GetInstanceId() {
	i, e := GetInstanceId()
	fmt.Println(len(i) > 0)
	fmt.Println(strings.HasPrefix(e.Error(), "Failed to fetch instance ID from IMDS: operation error ec2imds: GetMetadata, "))

	// Output:
	// true
	// true
}
