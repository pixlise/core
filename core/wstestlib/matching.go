package wstestlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/utils"
)

type WSMessageHeader struct {
	MsgId int `json:"msgId"`
}

// Returns the map of ids found (to be able to parse future msgs), the pretty-printed version
// of received msg that we're parsing, any errors (including for matching), and a bool indicating
// if the message was id-matched. If it wasn't id-matched, the caller can still do something
// because we may be matching update msgs which have no id set
func checkMatch(expectedMsg string, receivedMsg string, userId string) (map[string]string, string, error, bool) {
	idsCreated := map[string]string{}
	idMatched := false

	// Pretty print them both
	// Lots of vars to make this easy to debug
	var expectedMsgBytes bytes.Buffer
	var receivedMsgBytes bytes.Buffer
	err := json.Indent(&expectedMsgBytes, []byte(expectedMsg), "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return idsCreated, receivedMsg, fmt.Errorf("Failed to process expected response: %v. Error: %v", expectedMsg, err), idMatched
	}
	err = json.Indent(&receivedMsgBytes, []byte(receivedMsg), "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return idsCreated, receivedMsg, fmt.Errorf("Failed to process received response: %v. Error: %v", receivedMsg, err), idMatched
	}

	prettyExpectedMsgStr := expectedMsgBytes.String()
	prettyReceivedMsgStr := receivedMsgBytes.String()

	// Now read both as generic JSON objects and compare what we got
	var received map[string]any
	err = json.Unmarshal(receivedMsgBytes.Bytes(), &received)
	if err != nil {
		log.Fatalf("Failed to parse received JSON: %v", prettyReceivedMsgStr)
	}

	var expected map[string]any
	err = json.Unmarshal(expectedMsgBytes.Bytes(), &expected)
	if err != nil {
		log.Fatalf("Failed to parse expected JSON: %v", prettyExpectedMsgStr)
	}

	// Parse both as a message header, so we can read the msg id too
	var expHeader WSMessageHeader
	var recvHeader WSMessageHeader

	err = json.Unmarshal(receivedMsgBytes.Bytes(), &recvHeader)
	if err != nil {
		log.Fatalf("Failed to parse received JSON as WSMessage header: %v", prettyReceivedMsgStr)
	}
	err = json.Unmarshal(expectedMsgBytes.Bytes(), &expHeader)
	if err != nil {
		log.Fatalf("Failed to parse expected JSON as WSMessage header: %v", prettyExpectedMsgStr)
	}

	// If both have a msg id, we can know for sure if we're supposed to be compared
	idMatched = expHeader.MsgId == recvHeader.MsgId && expHeader.MsgId > 0

	err = compare(received, expected, userId, idsCreated)

	if err != nil {
		errTxt := err.Error()
		return idsCreated, prettyReceivedMsgStr, fmt.Errorf(`Match FAILED %v
====================================
Expected: %v
------------------------------------
Received: %v
====================================
`, errTxt, prettyExpectedMsgStr, prettyReceivedMsgStr), idMatched
	}

	// It is matched!
	return idsCreated, prettyReceivedMsgStr, nil, idMatched
}

func compare(received any, expected any, userId string, idsCreated map[string]string) error {
	switch expVal := expected.(type) {
	case nil:
		switch recVal := received.(type) {
		case nil:
			return nil // They match
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	case float64:
		switch recVal := received.(type) {
		case float64:
			// Check values
			if recVal != expVal {
				return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
			}
			return nil // They match
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	case bool:
		switch recVal := received.(type) {
		case bool:
			// Check values
			if recVal != expVal {
				return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
			}
			return nil // They match
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	case string:
		const SECAGO_START = "$SECAGO="

		secondsAgoCheck := 0
		if expVal == "$IGNORE$" {
			// We're choosing to deliberately ignore the received value here
			return nil
		} else if strings.HasPrefix(expVal, SECAGO_START) && strings.HasSuffix(expVal, "$") {
			// Work out what time stamp to compare
			secondsStr := expVal[len(SECAGO_START) : len(expVal)-1]
			seconds, _err := strconv.Atoi(secondsStr)
			if _err != nil {
				return fmt.Errorf("failed to read defined seconds ago: %v", secondsStr)
			}

			secondsAgoCheck = int(time.Now().Unix()) - seconds
		}

		switch recVal := received.(type) {
		case float64:
			// NOTE: it looks like time stamps (which are uint64) get converted to string instead of int
			// because that wouldn't fit into a javascript "number" (aka float64). So we have a check here but
			// also in string.

			// If we had a string specified, the only way the received thing having a number
			// is valid is if it's a timestamp comparison
			if secondsAgoCheck <= 0 {
				return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
			}

			if recVal < float64(secondsAgoCheck) {
				return fmt.Errorf(`received time stamp %v is %v seconds too old`, recVal, float64(secondsAgoCheck)-recVal)
			}

			// We accept the time stamp match...
			return nil
		case string:
			if secondsAgoCheck > 0 {
				// We're doing a timestamp age check, so convert the received value to a number
				seconds, _err := strconv.Atoi(recVal)
				if _err != nil {
					return fmt.Errorf("failed to read timestamp from string value: %v. Error was: %v", recVal, _err)
				}

				if seconds < secondsAgoCheck {
					return fmt.Errorf(`received time stamp %v is %v seconds too old`, seconds, secondsAgoCheck-seconds)
				}

				// Otherwise, we're happy wit it
				return nil
			}

			const ID_START = "$ID="

			// Check values, there may be some specific overrides here...
			if expVal == "$USERID$" {
				// We just want to see if the response has the user id
				if recVal != userId {
					return fmt.Errorf(`expected user id "%v", received "%v"`, userId, recVal)
				}
			} else if strings.HasPrefix(expVal, ID_START) && strings.HasSuffix(expVal, "$") {
				idName := expVal[len(ID_START) : len(expVal)-1]
				if len(idName) <= 0 {
					return fmt.Errorf("failed to read defined id name to save: %v", expVal)
				}

				// Save the ID
				idsCreated[idName] = recVal
			} else if recVal != expVal {
				return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
			}

			// If we got this far, they are considered a match
			return nil
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	case map[string]any:
		switch recVal := received.(type) {
		case map[string]any:
			// Get keys from both
			recKeys := utils.GetMapKeys(recVal)
			expKeys := utils.GetMapKeys(expVal)

			// Compare in sorted order
			sort.Strings(recKeys)
			sort.Strings(expKeys)

			if len(recKeys) != len(expKeys) {
				return fmt.Errorf("mismatch in structure, expected %v fields, received %v", len(expKeys), len(recKeys))
			}

			for c, expKey := range expKeys {
				recKey := recKeys[c]

				// Check if we have any specs as to how to interpret the received side...
				expKeyCompare := expKey
				var expSpecs string

				if strings.HasSuffix(expKey, "#") {
					specPos := strings.Index(expKey, "#")
					if specPos > 0 {
						// we have specs, so cut back what we're comparing
						expKeyCompare = expKey[0:specPos]
						expSpecs = expKey[specPos+1 : len(expKey)-1]
					}
				}

				if expKeyCompare != recKey {
					return fmt.Errorf(`expected field name: "%v", received: "%v"`, expKeyCompare, recKey)
				}

				// Key matches, compare sub-structure, depending on specs specified
				var keyErr error
				if len(expSpecs) > 0 {
					specParams := strings.Split(expSpecs, ",")
					if len(specParams) < 2 {
						return fmt.Errorf(`unknown expected parse spec: "%v"`, expSpecs)
					}

					if specParams[0] == "LIST" {
						keyErr = compareList(specParams[1:], recVal[recKey], expVal[expKey])
					}
				} else {
					keyErr = compare(recVal[recKey], expVal[expKey], userId, idsCreated)
				}

				if keyErr != nil {
					// Stop here
					return fmt.Errorf("\"%v\": %v", expKey, keyErr)
				}
			}
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	case []any:
		switch recVal := received.(type) {
		case []any:
			// They're both lists, compare length first
			if len(recVal) != len(expVal) {
				return fmt.Errorf("expected %v list items, received %v", len(expVal), len(recVal))
			}
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	default:
		log.Fatalf("unexpected type: %T", expected)
	}
	return nil
}

func compareList(params []string, received any, expected any) error {
	paramsForErr := strings.Join(params, ",")

	// Check params
	var mode string
	minLength := -1

	for _, param := range params {
		bits := strings.Split(param, "=")
		if len(bits) != 2 {
			return fmt.Errorf("failed to parse list compare specifications: %v", paramsForErr)
		}
		if strings.ToLower(bits[0]) == "mode" {
			mode = strings.ToLower(bits[1])
		} else if strings.ToLower(bits[0]) == "minlength" {
			var err error
			minLength, err = strconv.Atoi(bits[1])
			if err != nil || minLength < 1 {
				return fmt.Errorf("minimum length invalid in list compare specifications: %v", paramsForErr)
			}
		} else {
			return fmt.Errorf(`unknown item "%v" in list compare specifications: %v`, bits[0], paramsForErr)
		}
	}

	switch expVal := expected.(type) {
	case []any:
		switch recVal := received.(type) {
		case []any:
			if minLength > -1 && len(recVal) < minLength {
				return fmt.Errorf("expected at least %v list items, received %v", minLength, len(recVal))
			}

			// Compare based on what was asked
			if mode == "length" {
				// We're only comparing lengths, if we have a min length specified too, we use that
				if minLength < 0 && len(recVal) != len(expVal) {
					return fmt.Errorf("expected %v list items, received %v", len(expVal), len(recVal))
				}
			} else if mode == "contains" {
				// We only want the received list to contain the items specified, don't care about the rest...
				for _, expItem := range expVal {
					found := false
					for _, recItem := range recVal {
						if expItem == recItem {
							found = true
							break
						}
					}

					if !found {
						return fmt.Errorf(`expected list to contain item "%v"`, expItem)
					}
				}
			} else if mode == "sorted" {
				// Sort both lists before comparing
				return fmt.Errorf(`SORTED LIST COMPARE NOT YET IMPLEMENTED`)
			} else {
				return fmt.Errorf("invalid mode in list compare specifications: %v", paramsForErr)
			}
		default:
			// Received item is NOT a list
			return fmt.Errorf(`expected list compatible with parse spec "%v", received "%v"`, paramsForErr, recVal)
		}
	default:
		// Parse spec said list but had something else defined
		return fmt.Errorf(`expected list for list parse spec "%v"`, paramsForErr)
	}

	// We see them as a match...
	return nil
}
