package wstestlib

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
)

type WSMessageHeader struct {
	MsgId int `json:"msgId"`
}

// Returns the map of ids found (to be able to parse future msgs), the pretty-printed version
// of received msg that we're parsing, any errors (including for matching), and a bool indicating
// if the message was id-matched. If it wasn't id-matched, the caller can still do something
// because we may be matching update msgs which have no id set
func checkMatch(expectedMsg string, receivedMsg string, userId string, savedItems map[string]string) (string, error, bool) {
	idMatched := false

	// Pretty print them both
	// Lots of vars to make this easy to debug
	var expectedMsgBytes bytes.Buffer
	var receivedMsgBytes bytes.Buffer
	err := json.Indent(&expectedMsgBytes, []byte(expectedMsg), "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return receivedMsg, fmt.Errorf("Failed to process expected response: %v. Error: %v", expectedMsg, err), idMatched
	}
	err = json.Indent(&receivedMsgBytes, []byte(receivedMsg), "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		return receivedMsg, fmt.Errorf("Failed to process received response: %v. Error: %v", receivedMsg, err), idMatched
	}

	prettyExpectedMsgStr := expectedMsgBytes.String()
	prettyReceivedMsgStr := receivedMsgBytes.String()

	// Limit length
	const lengthLimit = 10000
	if len(prettyReceivedMsgStr) > lengthLimit {
		prettyReceivedMsgStr = prettyReceivedMsgStr[0:lengthLimit]
	}

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

	ctx := compareParams{
		userId:                 userId,
		savedItems:             savedItems,
		allowSaveItemOverwrite: false,
	}
	err = compare(received, expected, ctx)

	if err != nil {
		errTxt := err.Error()
		return prettyReceivedMsgStr, fmt.Errorf(`Match FAILED %v
====================================
Expected: %v
------------------------------------
Received: %v
====================================
`, errTxt, prettyExpectedMsgStr, prettyReceivedMsgStr), idMatched
	}

	// It is matched!
	return prettyReceivedMsgStr, nil, idMatched
}

type compareParams struct {
	userId                 string
	savedItems             map[string]string
	allowSaveItemOverwrite bool // If true, allow IDSAVE= to overwrite an existing one
}

func compare(received any, expected any, ctx compareParams) error {
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
		// Check for IGNORE here so we can ignore not just strings but anything on the received msg
		defMap, preDef, postDef, err := parseDefinitions(expVal)
		if err != nil {
			return err
		}

		if len(defMap) == 1 && len(preDef) == 0 && len(postDef) == 0 {
			for cmd, params := range defMap {
				if cmd == "IGNORE" && len(params) <= 0 {
					// We're ignoring whatever came in on the right, so not even checking the type
					return nil
				}
			}
		}

		expToCompare := ""
		recValAsStr := ""
		switch recVal := received.(type) {
		case float64:
			// NOTE: timestamps were uint64 but because the JS proto serialiser creates strings from them
			// (since float64 doesn't have enough precision to store it...) we were handling time as strings
			// for a while, but now switched all our timestamps to uint32 because who knows who will be around
			// in 2106. Sorry! Maybe JS will support an integer type by then? :)
			if strings.Contains(expVal, "SECAGO") || strings.Contains(expVal, "SECAFTER") {
				recValAsStr = fmt.Sprintf("%d", int64(recVal))
				expToCompare, err = compareExpectedString(expVal, recValAsStr, ctx)
				if err != nil {
					return err
				}
			}
		case string:
			recValAsStr = recVal
			if len(defMap) > 0 && len(preDef) == 0 && len(postDef) == 0 {
				for cmd := range defMap {
					// If valid, we'll see a map of ZIPCMP:<nothing> and PATH:<path> or other parameters along side it in defMap
					if cmd == "ZIPCMP" {
						// We have to unzip the result and compare to a directory specified
						return zipCompare(defMap, recValAsStr)
					}
				}
			}

			expToCompare, err = compareExpectedString(expVal, recVal, ctx)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}

		if expToCompare != recValAsStr {
			// They don't match!
			if expToCompare == expVal {
				return fmt.Errorf(`expected "%v", received "%v"`, expToCompare, recValAsStr)
			} else {
				// The expected string has changed while being processed
				return fmt.Errorf(`expected "%v" (raw string: "%v"), received "%v"`, expToCompare, expVal, recValAsStr)
			}
		}

		// If we got this far, they are considered a match
		return nil
	case map[string]any:
		switch recVal := received.(type) {
		case map[string]any:
			// Get keys from both
			recKeys := utils.GetMapKeys(recVal)
			expKeys := utils.GetMapKeys(expVal)

			// Compare in sorted order
			sort.Strings(recKeys)
			sort.Strings(expKeys)

			missingKeys := []string{}
			if len(recKeys) != len(expKeys) {
				// Find which expected key is missing
				for _, expKey := range expKeys {
					ignoreField := false

					// Check if it's in the received keys
					if !utils.ItemInSlice(expKey, recKeys) {
						// We need to check if we can ignore this key
						expectedKeyValue := expVal[expKey]
						expectedKeyValueStr, ok := expectedKeyValue.(string)
						if ok {
							defMap, preDef, postDef, err := parseDefinitions(expectedKeyValueStr)
							if err != nil {
								return err
							}

							if len(defMap) == 1 && len(preDef) == 0 && len(postDef) == 0 {
								for cmd, params := range defMap {
									if cmd == "IGNORE" && len(params) <= 0 {
										// If we're ignoring this field, remove it from expected keys
										// so we don't report it as missing
										expKeys = utils.RemoveItemFromSlice(expKey, expKeys)
										ignoreField = true
									}
								}
							}
						}

						if !ignoreField {
							missingKeys = append(missingKeys, expKey)
						}
					}
				}
				if len(missingKeys) > 0 {
					return fmt.Errorf("mismatch in structure, expected %v fields, received %v. Expected keys: [%v]. Missing from Received: [%v]", len(expKeys), len(recKeys), strings.Join(expKeys, ", "), strings.Join(missingKeys, ", "))
				}

				// If we got keys that were not expected, that's an error too
				if len(expKeys) < len(recKeys) {
					return fmt.Errorf("mismatch in structure, expected %v fields, received %v. Expected keys: [%v]. Received keys: [%v]", len(expKeys), len(recKeys), strings.Join(expKeys, ", "), strings.Join(recKeys, ", "))
				}
			}

			for c, expKey := range expKeys {
				recKey := recKeys[c]

				// expKey can be just a field name:
				// "somefield"
				// If so, we just string compare
				//
				// Or it can have a definition at the end:
				// "somefield${A=B,C=D}"
				// If so, we assume it's a key name with special parsing params after it
				//
				// Or no text at the start:
				// "${A=B}"
				// If so, we assume it's just a complex string match, only allowing ONE A=B...
				defMap, expKeyCompare, postDef, err := parseDefinitions(expKey)
				if err != nil {
					return err
				}

				// Never want text AFTER it...
				if len(postDef) > 0 {
					return fmt.Errorf("Unexpected text after definition: %v", expKey)
				}

				var keyErr error
				if len(defMap) <= 0 {
					// Just a straight comparison, first compare the field names
					if expKeyCompare != recKey {
						return fmt.Errorf(`expected key: "%v", received key: "%v"`, expKeyCompare, recKey)
					}

					keyErr = compare(recVal[recKey], expVal[expKey], ctx)
				} else if len(expKeyCompare) > 0 && len(defMap) > 0 {
					// Second style, first compare the field names
					if expKeyCompare != recKey {
						return fmt.Errorf(`expected key: "%v", received key: "%v"`, expKeyCompare, recKey)
					}

					// Now check the special parsing params
					processed := false
					for key, val := range defMap {
						if key == "LIST" && len(val) <= 0 {
							// We're doing list processing!
							delete(defMap, key)
							keyErr = compareList(defMap, recVal[recKey], expVal[expKey], ctx)
							processed = true
							break
						}
					}
					if !processed {
						keyErr = errors.New("Failed to find relevant sub-list definition")
					}
				} else if len(expKeyCompare) == 0 && len(defMap) == 1 {
					// Third style, do a comparison
					expKeyCompare, keyErr = compareExpectedString(expKey, recKey, ctx)

					if expKeyCompare != recKey {
						return fmt.Errorf(`expected key: "%v", received key: "%v"`, expKeyCompare, recKey)
					}

					// Now do the comparison as normal
					keyErr = compare(recVal[recKey], expVal[expKey], ctx)
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

			// Lengths are equal, compare each
			for c, expItem := range expVal {
				recItem := recVal[c]
				keyErr := compare(recItem, expItem, ctx)

				if keyErr != nil {
					// Stop here
					return fmt.Errorf("\"%v\": %v", expItem, keyErr)
				}
			}
		default:
			return fmt.Errorf(`expected "%v", received "%v"`, expVal, recVal)
		}
	default:
		return fmt.Errorf("unexpected type: %T for defined expected data", expected)
	}
	return nil
}

func getDefinitionBetween(str string, start string, end string) (string, string, string, error) {
	startPos := strings.Index(str, start)
	if startPos < 0 {
		// No definitions in this string so put the entire thing in to "pre" def in result
		return "", str, "", nil
	}

	// Find closing part
	strNoStart := str[startPos+len(start):]
	length := strings.Index(strNoStart, end)
	if length < 0 {
		return "", "", "", fmt.Errorf(`failed to find closing token for "%v" in "%v"`, end, str)
	}

	// Found it!
	return strNoStart[0:length], str[0:startPos], str[startPos+len(start)+length+len(end):], nil
}

// Parses expected string and finds a cmd and a value if defined
func parseDefinitions(str string) (map[string]string, string, string, error) {
	result := map[string]string{}
	def, pre, post, err := getDefinitionBetween(str, "${", "}")
	if err != nil || len(def) <= 0 {
		return result, pre, post, err
	}

	parts := strings.Split(def, ",")

	// We always end up with at least one... so anything is valid at this point
	// Now parse each one
	for _, exp := range parts {
		expParts := strings.Split(exp, "=")
		switch len(expParts) {
		case 1:
			// Add to the map with no params defined
			result[strings.Trim(expParts[0], "\t ")] = ""
		case 2:
			// Add as key-value pair to map
			result[strings.Trim(expParts[0], "\t ")] = strings.Trim(expParts[1], "\t ")
		}
	}

	err = nil
	if len(result) <= 0 {
		err = fmt.Errorf(`failed to parse cmd/var from "%v"`, str)
	}

	return result, pre, post, err
}

// Compares expected string to received string taking into account any embedded comparison operators
func compareExpectedString(expStr string, recvStr string, ctx compareParams) (string, error) {
	defMap, preDef, postDef, err := parseDefinitions(expStr)
	if err != nil {
		// Error parsing the expected string, stop right here
		return "", err
	}

	if len(defMap) == 0 {
		// No defs to process, so it's just a string compare
		return preDef, nil
	}

	if len(defMap) == 1 {
		// Here if there is a definition we expect it to take up the whole string
		if len(postDef) > 0 || len(preDef) > 0 {
			return "", fmt.Errorf("Unexpected text around definition: %v", expStr)
		}

		for cmd, param := range defMap {
			if len(cmd) <= 0 && len(param) <= 0 {
				// Just a simple string compare
				return expStr, nil
			}

			// Work out what our cmd/val combo means and if it's valid
			if cmd == "USERID" && param == "" {
				// We just want to see if the response has the user id
				return ctx.userId, nil
			} else if cmd == "IGNORE" && len(param) <= 0 {
				// We are ignoring, so return the received string so we don't trigger
				return recvStr, nil
			} else if cmd == "IDSAVE" && len(param) > 0 {
				// Saving an ID (we only check that it's not an empty string here)
				if !ctx.allowSaveItemOverwrite {
					// Check that it doesn't exist already, allowing overwrites would confuse things
					if savedVal, ok := ctx.savedItems[param]; ok && savedVal != recvStr {
						// This is not a failure of matching, so if we're comparing 2 lists 1 level up, it wouldn't even print this out! We have
						// to print it here
						err := fmt.Errorf("saved id for %v already exists: %v, doesn't match save attempt: %v", param, savedVal, recvStr)
						fmt.Println(err)
						return "", err
					}
				}

				if len(recvStr) <= 0 {
					return "", fmt.Errorf(`received empty string when trying to save id as "%v"`, param)
				}

				// Save the ID
				ctx.savedItems[param] = recvStr

				// Return the received str so we don't trigger
				return recvStr, nil
			} else if cmd == "IDCHK" && len(param) > 0 {
				// Check that the saved id with given name is what we received
				if savedId, ok := ctx.savedItems[param]; !ok {
					return "", fmt.Errorf("failed to find defined id name to compare: %v", expStr)
				} else {
					// We return the saved id value to be compared
					return savedId, nil
				}
			} else if cmd == "REGEXMATCH" {
				// Regex match the received value string
				match, err := regexp.Match(param, []byte(recvStr))
				if err != nil {
					return "", fmt.Errorf(`regex match "%v" failed on received "%v". Error: %v`, param, recvStr, err)
				}
				if !match {
					return "", fmt.Errorf(`received "%v" did not match regex "%v"`, recvStr, param)
				}

				// We accept it, don't trip out
				return recvStr, nil
			} else if cmd == "SECAGO" || cmd == "SECAFTER" {
				// Parse the parameter and received string as ints
				var iParam, iRecv int
				var parseErr error
				if iParam, parseErr = strconv.Atoi(param); parseErr != nil {
					return "", fmt.Errorf(`failed to parse param "%v" for "%v" as int: %v`, param, cmd, parseErr)
				}

				if iRecv, parseErr = strconv.Atoi(recvStr); parseErr != nil {
					return "", fmt.Errorf(`failed to parse received "%v" for "%v" comparison as int: %v`, recvStr, cmd, parseErr)
				}

				// Check the received value against the time stamp
				if cmd == "SECAGO" {
					if iParam < 0 {
						// Can't be asking for a future time!
						return "", fmt.Errorf(`invalid value for SECAGO: "%v"`, expStr)
					}
					secondsAgoCheck := int(time.Now().Unix()) - iParam

					if iRecv < secondsAgoCheck {
						return "", fmt.Errorf(`received time stamp %v is %v seconds too old`, iRecv, secondsAgoCheck-iRecv)
					}

					// We accept it, don't trip out
					return recvStr, nil
				}

				// SECAFTER
				if iRecv < iParam {
					return "", fmt.Errorf(`received time stamp %v is before expected %v`, iRecv, iParam)
				}

				// We accept it, don't trip out
				return recvStr, nil
			}
		}
	}

	// If we got this far, we couldn't interpret the inputs
	return "", fmt.Errorf("Unknown matching cmd/param combination: %v", expStr)
}

// List comparison, can have a few items defined:
// minlength - min length of received list
// mode - how we compare the lists
func compareList(defMap map[string]string, received any, expected any, ctx compareParams) error {
	// Check params
	var mode string
	minLength := -1
	expLength := -1

	const def_Mode = "MODE"
	const def_MinLength = "MINLENGTH"
	const def_Length = "LENGTH"

	allDefs := []string{def_MinLength, def_Length, def_Mode}
	for key := range defMap {
		if !utils.ItemInSlice(key, allDefs) {
			return fmt.Errorf("unrecognised list spec: %v", key)
		}
	}

	if _mode, ok := defMap[def_Mode]; ok {
		mode = strings.ToLower(_mode)
	}
	if _minLen, ok := defMap[def_MinLength]; ok {
		var err error
		minLength, err = strconv.Atoi(_minLen)
		if err != nil || minLength < 1 {
			return fmt.Errorf("minimum length invalid in list compare specifications: %v", _minLen)
		}
	}
	if _len, ok := defMap[def_Length]; ok {
		var err error
		expLength, err = strconv.Atoi(_len)
		if err != nil || expLength < 1 {
			return fmt.Errorf("length invalid in list compare specifications: %v", expLength)
		}
	}

	// If we have unrecognised items, stop here

	switch expVal := expected.(type) {
	case []any:
		switch recVal := received.(type) {
		case []any:
			if minLength > -1 && len(recVal) < minLength {
				return fmt.Errorf("expected at least %v list items, received %v", minLength, len(recVal))
			}
			if expLength > -1 && len(recVal) != expLength {
				return fmt.Errorf("expected exactly %v list items, received %v", expLength, len(recVal))
			}

			// Compare based on what was asked
			if mode == "length" {
				// We're only comparing lengths, if we have a min length specified too, we use that
				if minLength < 0 && expLength < 0 && len(recVal) != len(expVal) {
					return fmt.Errorf("expected %v list items, received %v", len(expVal), len(recVal))
				}
			} else if mode == "contains" {
				// We only want the received list to contain the items specified, don't care about the rest...
				for _, expItem := range expVal {
					found := false
					for _, recItem := range recVal {
						// We compare what we have to anything that came in, errors don't matter, we just want to find a match

						// NOTE: if we're here, any IDSAVE= below us should not fail if item exists, because the one we find to match
						// will only match the one we desire to save! So here we loosen the requirement for save item overwriting
						ctx2 := compareParams{
							userId:                 ctx.userId,
							savedItems:             ctx.savedItems,
							allowSaveItemOverwrite: true,
						}
						err := compare(recItem, expItem, ctx2)
						if err == nil {
							found = true
							break
						}
					}

					if !found {
						return fmt.Errorf(`expected list to contain item "%v"`, expItem)
					}
				}
			} else {
				return fmt.Errorf("invalid mode in list compare specifications: %v", defMap)
			}
		default:
			// Received item is NOT a list
			return fmt.Errorf(`expected list compatible with parse spec "%v", received "%v"`, defMap, recVal)
		}
	default:
		// Parse spec said list but had something else defined
		return fmt.Errorf(`expected list for list parse spec "%v"`, defMap)
	}

	// We see them as a match...
	return nil
}

func zipCompare(defMap map[string]string, resultStr string) error {
	expPath, ok := defMap["PATH"]
	if !ok {
		return errors.New("Expected PATH def for ZIPCMP")
	}

	// Unzip the result data so we can compare contents
	resultDir, err := os.MkdirTemp("", "zipCompareData")
	if err != nil {
		return err
	}

	resultName := "resultZip"
	resultZipPath := filepath.Join(resultDir, resultName+".zip")
	// Result is base64 encoded... for some reason...
	zipData, err := base64.StdEncoding.DecodeString(resultStr)
	if err != nil {
		return err
	}
	err = os.WriteFile(resultZipPath, zipData, 0777)
	if err != nil {
		return err
	}

	resultUnzipDir := filepath.Join(resultDir, resultName)
	resultFiles, err := utils.UnzipDirectory(resultZipPath, resultUnzipDir, false)
	if err != nil {
		return err
	}

	// Compare the output files with what's in our expected dir
	fs := fileaccess.FSAccess{}
	expFiles, err := fs.ListObjects(expPath, "")
	if err != nil {
		return err
	}

	if len(resultFiles) != len(expFiles) {
		return fmt.Errorf("Expected %v files, unzipped %v files", len(expFiles), len(resultFiles))
	}

	// For now, this only works for the simple case of one file to compare! This is because the resultant file name might include
	// an id in it so won't match the expected file exactly, so if there's only one file, we compare that and we're done
	// but if there are multiple files we'll need more information to match the result vs expected file.
	if len(resultFiles) != 1 {
		return fmt.Errorf("For now, wstestlib only supports comparing zip outputs if there is just one file in the zip file.")
	}

	resultBytes, err := os.ReadFile(resultFiles[0])
	if err != nil {
		return fmt.Errorf("Failed to open result file from zip. Name: %v. Error: %v", resultFiles[0], err)
	}

	expFilePath := filepath.Join(expPath, expFiles[0])
	expBytes, err := os.ReadFile(expFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open expected file for compare. Name: %v. Error: %v", expFilePath, err)
	}

	if len(expBytes) != len(resultBytes) {
		return fmt.Errorf("Expected data does not match result data for file: %v, exp file: %v", resultFiles[0], expFilePath)
	}

	expStr := string(expBytes)
	resultStrData := string(resultBytes)

	// If we were told how many lines to skip in the comparison, do it
	compareStartLine := 0
	skipStr, ok := defMap["SKIPCSVLINES"]
	if ok {
		var convErr error
		compareStartLine, convErr = strconv.Atoi(skipStr)
		if convErr != nil {
			return fmt.Errorf("SKIPCSVLINES contained invalid value: %v", skipStr)
		}
	}

	expStrLines := strings.Split(expStr, "\n")
	resultStrLines := strings.Split(resultStrData, "\n")

	for c, expLine := range expStrLines {
		if c < compareStartLine {
			continue
		}

		if expLine != resultStrLines[c] {
			return fmt.Errorf("ZipCompare failed for expected file: %v, result file: %v at expected file line %v.\nExpected \"%v\"\nResult   \"%v\"", expFilePath, resultFiles[0], c, expLine, resultStrLines[c])
		}
	}

	return nil
}
