package utils

import (
	"fmt"
	"strings"
)

func Example_utils_GetInstanceId() {
	i, b, e := GetInstanceId()
	fmt.Printf("id not empty: %v\n", len(i) > 0)
	fmt.Printf("isEC2: %v\n", b)
	if e != nil {
		fmt.Printf("expected error: %v\n", strings.HasPrefix(e.Error(), "Failed to fetch instance ID from IMDS: operation error ec2imds: GetMetadata, "))
	} else {
		// Prevent crash... Github actions started doing this now, maybe they were moved to AWS?!
		fmt.Println("e is nil")
	}

	// Output:
	// id not empty: true
	// isEC2: false
	// expected error: true
}
