package wstestlib

import (
	"encoding/base64"
	"fmt"
	"os"
)

// Replaces anything in the request that needs replacing. At time of writing
// this only involves the ability to use $IDLOAD=name$, where we look up the
// value from the package global "savedItems"
func doReqReplacements(req string, savedItemLookup map[string]string) (string, error) {
	reqResult := req

	for {
		defMap, pre, post, err := parseDefinitions(reqResult)
		if err != nil {
			return "", err
		}

		// Expecting ONE or NO defs
		if len(defMap) == 0 {
			// Nothing here, stop
			break
		} else if len(defMap) == 1 {
			// Process this item
			for key, value := range defMap {
				if key == "IDLOAD" {
					if len(value) <= 0 {
						return "", fmt.Errorf("IDLOAD: Missing replacement id name")
					}

					if replaceWith, ok := savedItemLookup[value]; !ok {
						return "", fmt.Errorf("IDLOAD: No replacement text named: %v for request message: %v", value, req)
					} else {
						// Found it, do the replacement!
						reqResult = pre + replaceWith + post
					}
				} else if key == "FILEBYTES" {
					if len(value) <= 0 {
						return "", fmt.Errorf("FILEBYTES: Missing file path")
					}

					// Read the file specified, include the bytes in the field encoded as base64
					data, err := os.ReadFile(value)
					if err != nil {
						return "", fmt.Errorf(`Failed to read file into message field: %v`, req)
					}

					reqResult = pre + base64.StdEncoding.EncodeToString(data) + post
				} else {
					return "", fmt.Errorf("Unknown definition used on request message: %v", key)
				}
			}
		} else {
			return "", fmt.Errorf("Unexpected number of definitions (%v) on request line: %v", len(defMap), reqResult)
		}
	}

	return reqResult, nil
}
