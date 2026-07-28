package utils

import (
	"fmt"
	"strings"
)

// Reads a list of key=value strings. Provide they key names, and it returns a lookup for the values
// Make sure they're all supplied and not empty
func ReadKeyValueList(expectedKeys []string, toRead []string) (map[string]string, error) {
	result := map[string]string{}

	for _, kv := range toRead {
		bits := strings.Split(kv, "=")
		if len(bits) != 2 || len(bits[0]) <= 0 || len(bits[1]) <= 0 {
			return result, fmt.Errorf("Invalid key/value pair specified: \"%v\"", kv)
		}

		result[bits[0]] = bits[1]
	}

	for _, k := range expectedKeys {
		if _, ok := result[k]; !ok {
			return result, fmt.Errorf("No value supplied for key: \"%v\"", k)
		}
	}

	return result, nil
}
