package utils

import (
	"fmt"
	"strings"
)

func Example_utils_GetInstanceId() {
	i, b, e := GetInstanceId()
	fmt.Printf("id not empty: %v\n", len(i) > 0)

	// We want this test to pass if we're in AWS or not, as it looks like
	// our github actions are now getting instance IDs all of a sudden!
	expectedErrPrefix := "Failed to fetch instance ID from IMDS: operation error ec2imds: GetMetadata, "

	if b && e == nil || !b && strings.HasPrefix(e.Error(), expectedErrPrefix) {
		fmt.Printf("isEC2 and error is OK")
	}

	// Output:
	// id not empty: true
	// isEC2 and error is OK
}
