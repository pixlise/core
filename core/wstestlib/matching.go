package wstestlib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/utils"
)

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

	// Run through line-by-line, compare fields based on comparison specified or exact compare if nothing else
	expMsgLines := strings.Split(prettyExpectedMsgStr, "\n")
	recvMsgLines := strings.Split(prettyReceivedMsgStr, "\n")

	// From here, if we dont detect a match, we set this flag and the error can explain what's wrong
	notMatch := false
	err = nil

	// At this point, if they both have msg IDs, check if they are the same, otherwise there's
	// no sense in trying to match any further
	expMsgId := getMsgId(expMsgLines)
	recvMsgId := getMsgId(recvMsgLines)

	if expMsgId > 0 && recvMsgId > 0 {
		// Both messages have ids, check if they match
		if expMsgId == recvMsgId {
			idMatched = true
		} else {
			// The ids don't match, so stop here, no sense comparing contents of 2 unrelated msgs
			return idsCreated, prettyReceivedMsgStr, errors.New("IDs don't match"), false
		}
	}

	if len(expMsgLines) != len(recvMsgLines) {
		// Don't even bother looping through lines and worrying about indexes...
		notMatch = true
	} else {
		for c, expLine := range expMsgLines {
			recvLine := recvMsgLines[c]
			// Check if we have something in our line to specify how to compare
			// We look for things like "var": "$IGNORE$",
			// and take action on what's between $$
			const expEnd = "$\""
			const expStart = "\"$"

			const recvEnd = "\""
			const recvStart = "\""

			const ID_START = "ID="

			const SECAGO_START = "SECAGO="

			expValue, expValuePos := getTextBetween(expLine, expStart, expEnd)

			if expValuePos >= 5 { // There has to be at LEAST ["a": ] before it
				if expValue == "IGNORE" {
					// Skip over this line
					continue
				} else if expValue == "USERID" {
					// Match user id
					recvValue, recvValuePos := getTextBetween(recvLine, recvStart, recvEnd)
					if recvValuePos > 5 && recvValue != userId {
						notMatch = true
						err = fmt.Errorf("Received unexpected user id: %v", recvLine)
						break
					} // else OK keep going
				} else if strings.HasPrefix(expValue, ID_START) {
					// We want to save this as a named ID so it can be referred to in later bits of test script
					idName := expValue[len(ID_START):]
					if len(idName) <= 0 {
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read ID name to save when reading match string: %v", expValue), idMatched
					}

					// Now get the value to save
					recvValue, recvValuePos := getTextBetween(recvLine, recvStart, recvEnd)
					if recvValuePos < 0 {
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read received line: %v for expected line: %v", recvLine, expLine), idMatched
					}

					if recvValuePos <= 5 { // There has to be at LEAST ["a": ] before it
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read end of received line: %v for expected line: %v", recvLine, expLine), idMatched
					}

					idsCreated[idName] = recvValue

					// NOTHING to match here, we're just accepting the value as an id to save
					continue
				} else if strings.HasPrefix(expValue, SECAGO_START) {
					seconds, _err := strconv.Atoi(expValue[len(SECAGO_START):])
					if _err != nil {
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read seconds for %v, line was: %v", SECAGO_START, expLine), idMatched
					}
					// Generate a time stamp for that many seconds ago
					timestamp := int(time.Now().Unix()) - seconds

					recvValue, recvValuePos := getTextBetween(recvLine, recvStart, recvEnd)
					if recvValuePos < 0 {
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read received line: %v for expected line: %v", recvLine, expLine), idMatched
					}

					// Convert the read value to int too
					secondsRecvd, _err := strconv.Atoi(recvValue)
					if _err != nil {
						return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Failed to read time stamp from received line: %v", recvLine), idMatched
					}

					if secondsRecvd < timestamp {
						notMatch = true
						recvLinePrint := strings.Trim(recvLine, " \t")
						err = fmt.Errorf("Received time stamp %v sec too old. Received line: \"%v\" expected to be greater than: %v", timestamp-secondsRecvd, recvLinePrint, timestamp)
						break
					}
				} else {
					return idsCreated, prettyReceivedMsgStr, fmt.Errorf("Unknown comparison action: %v", expValue), idMatched
				}
			} else {
				// Just compare directly
				if recvLine != expLine {
					notMatch = true
					err = fmt.Errorf("Mismatch on line: %v [%v]", c, recvLine)
					break
				}
			}
		}
	}

	if notMatch {
		errTxt := ""
		if err != nil {
			errTxt = err.Error()
		}
		return idsCreated, prettyReceivedMsgStr, fmt.Errorf(`Match FAILED %v
====================================
Expected (# lines %v): %v
------------------------------------
Received (# lines %v): %v
====================================
`, errTxt, len(expMsgLines), prettyExpectedMsgStr, len(recvMsgLines), prettyReceivedMsgStr), idMatched
	}

	// It is matched!
	return idsCreated, prettyReceivedMsgStr, nil, idMatched
}

// Returns text between startToMatch and strEndToMatch - expecting the string to actually END after strEndToMatch
// otherwise we dont find it. Returns ("", -1) if not found, otherwise the text found and the start idx
func getTextBetween(str string, startToMatch string, strEndToMatch string) (string, int) {
	// If , isn't specifically what we're looking for, remove it here and treat it as
	// optional at the end because some lines may end in , others not, depending on the structure...
	if strEndToMatch != "," {
		str = strings.TrimSuffix(str, ",")
	}

	// Check if it ends how we expect...
	if strings.HasSuffix(str, strEndToMatch) {
		// Get the string
		quotePos := strings.LastIndex(str[0:len(str)-len(strEndToMatch)], startToMatch)
		if quotePos >= 0 {
			// Read the expected value
			readPos := quotePos + len(startToMatch)
			return str[readPos : len(str)-len(strEndToMatch)], readPos
		}
	}

	return "", -1
}

func getMsgId(lines []string) int {
	// Find the msg id
	// Example:
	/*
		{
			"msgId": 3,
			"status": "WS_OK",
			"elementSetGetResp": {
	*/
	// We probably only need to scan the first few lines, because the ID should
	// be there, but just out of paranoia, we can leave it scanning the whole thing
	// for now

	for _ /*c*/, line := range lines {
		/*if c > 3 {
			break
		}*/

		line = strings.Trim(line, "\t ")
		txt, pos := getTextBetween(line, "\"msgId\": ", ",")
		if pos > -1 && len(txt) > 0 {
			id, err := strconv.Atoi(txt)
			if err == nil {
				return id
			}
		}
	}
	return -1
}
