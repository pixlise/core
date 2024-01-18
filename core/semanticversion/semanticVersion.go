package semanticversion

import (
	"fmt"
	"strconv"
	"strings"

	protos "github.com/pixlise/core/v4/generated-protos"
)

func SemanticVersionToString(v *protos.SemanticVersion) string {
	if v == nil {
		return "?.?.?"
	}
	return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Patch)
}

func SemanticVersionFromString(v string) (*protos.SemanticVersion, error) {
	result := &protos.SemanticVersion{}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("Invalid semantic version: %v", v)
	}
	nums := []int{}
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return result, fmt.Errorf("Failed to parse version %v, part %v is not a number", v, part)
		}
		nums = append(nums, num)
	}

	result.Major = int32(nums[0])
	result.Minor = int32(nums[1])
	result.Patch = int32(nums[2])

	return result, nil
}
