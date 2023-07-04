package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
)

type connectInfo struct {
	host string
	user string
	pass string
}

// An individual action that can be taken
// NOTE: one of these fields must be set
type actionItem struct {
	annotation string // Purely for logging of test

	connect    *connectInfo
	disconnect bool
	sendReq    string
	waitMs     uint32
}

// After a group of actions is executed, we can specify
// multiple expected responses (or updates) for cases
// where we may want to send multiple requests out and
// capture all the randomly-ordered responses/updates
type actionGroup struct {
	actions          []actionItem
	expectedMessages []string
	waitTimeMs       int
}

type scriptedTestUser struct {
	user         *socketConn
	actionGroups []actionGroup

	tempGroup *actionGroup

	groupIdx  int
	actionIdx int

	userNameConnected string
	idsCreated        map[string]string
}

func makeScriptedTestUser() scriptedTestUser {
	return scriptedTestUser{
		user:         &socketConn{},
		actionGroups: []actionGroup{},
		idsCreated:   map[string]string{},
	}
}

// Use to reset a user, fails if called before all existing groups are complete
func (s *scriptedTestUser) clearActions() {
	if s.groupIdx != len(s.actionGroups) {
		log.Fatalf("Unexpected call to clearActions on user: %v", s.userNameConnected)
	}

	// Reset testing stuff
	s.actionGroups = []actionGroup{}
	s.tempGroup = nil

	s.groupIdx = 0
	s.actionIdx = 0

	// NOTE: don't reset idsCreated - almost the whole point of this function is to be able
	// to reuse vars that were in previous tests!
}

// Annotation could just be specified as part of the action item, but we force it to be
// a separate parameter to catch missing annotations in the compiler
func (s *scriptedTestUser) addAction(annotation string, action actionItem) {
	if len(action.annotation) > 0 {
		log.Fatalf("Action annotation %v would overwrite: %v", annotation, action.annotation)
	}
	action.annotation = annotation

	if s.tempGroup == nil {
		s.tempGroup = &actionGroup{
			actions:          []actionItem{},
			expectedMessages: []string{},
		}
	}

	s.tempGroup.actions = append(s.tempGroup.actions, action)
}

func (s *scriptedTestUser) addExpectedMessages(resps []string, waitTimeMs int) {
	// Close a group
	if s.tempGroup == nil {
		log.Fatal("Cannot add expected responses")
	}

	// Add responses to the group
	s.tempGroup.expectedMessages = resps
	s.tempGroup.waitTimeMs = waitTimeMs

	// Save the group
	s.actionGroups = append(s.actionGroups, *s.tempGroup)

	// Clear it
	s.tempGroup = nil
}

func (s *scriptedTestUser) getIdCreated(name string) string {
	if val, ok := s.idsCreated[name]; ok {
		return val
	}
	log.Fatalf("Failed to find saved ID named: %v", name)
	return ""
}

// Returns false if finished, error if there's an error
func (s *scriptedTestUser) runNextAction() (bool, error) {
	// Run the next action
	if s.groupIdx >= len(s.actionGroups) {
		return false, errors.New("Tests already finished")
	}

	if s.actionIdx >= len(s.actionGroups[s.groupIdx].actions) {
		// Start the next group
		s.actionIdx = 0
		s.groupIdx++

		// If we just got to the end, we're finished with this group
		if s.groupIdx >= len(s.actionGroups) {
			err := s.completeGroup(s.actionGroups[s.groupIdx-1])
			return false, err
		}
	}

	// Run the action we're pointing to
	if s.actionIdx == 0 {
		s.printGroup(fmt.Sprintf("Running group %v (%v actions)", s.groupIdx, len(s.actionGroups[s.groupIdx].actions)))
	}
	err := s.runSpecificAction(s.actionGroups[s.groupIdx].actions[s.actionIdx], fmt.Sprintf("g%v,a%v", s.groupIdx, s.actionIdx))
	s.actionIdx++
	return true, err
}

func (s *scriptedTestUser) runSpecificAction(action actionItem, which string) error {
	if action.connect != nil {
		s.printAction(which, action.annotation, fmt.Sprintf("Connecting to host: %v as user %v", action.connect.host, action.connect.user))
		s.userNameConnected = action.connect.user
		return s.user.connect(action.connect.host, action.connect.user, action.connect.pass)
	}

	if action.disconnect {
		s.printAction(which, action.annotation, fmt.Sprintf("Disconnecting user %v", s.userNameConnected))
		return s.user.disconnect()
	}

	if action.waitMs > 0 {
		s.printAction(which, action.annotation, fmt.Sprintf("Waiting %vms", action.waitMs))
		time.Sleep(time.Millisecond * time.Duration(action.waitMs))
		return nil
	}

	if len(action.sendReq) > 0 {
		// Snip out the first line to send
		sendSnippet := action.sendReq
		linePos := strings.Index(sendSnippet, "\n")
		if linePos > 0 {
			sendSnippet = sendSnippet[0:linePos]
		}
		s.printAction(which, action.annotation, fmt.Sprintf("Sending req %v", sendSnippet))

		wsmsg := protos.WSMessage{}
		err := protojson.Unmarshal([]byte(action.sendReq), &wsmsg)
		if err != nil {
			log.Fatalln(fmt.Errorf("Failed to parse request to be sent: %v.\nAction: %v\nRequest was: %v", err, action.annotation, action.sendReq))
		}
		return s.user.sendMessage(&wsmsg)
	}

	return errors.New("No action to take")
}

func (s *scriptedTestUser) printAction(which string, annotation string, desc string) {
	fmt.Printf("   - [%v] %v (%v)...\n", which, annotation, desc)
}

func (s *scriptedTestUser) printGroup(desc string) {
	fmt.Printf(" => %v...\n", desc)
}

func (s *scriptedTestUser) completeGroup(group actionGroup) error {
	s.printGroup(fmt.Sprintf("Waiting %vms for messages", group.waitTimeMs))

	// Check that the expected msgs match
	msgs := s.user.waitForMessages(len(group.expectedMessages), time.Duration(group.waitTimeMs)*time.Millisecond)

	if len(msgs) <= 0 {
		return errors.New("No messages received")
	} else {
		// In case we got something more than the expected count of messages, put in a small wait here
		msgs2 := s.user.waitForMessages(0, time.Duration(50)*time.Millisecond)
		if len(msgs2) > 0 {
			fmt.Printf("Received %v more messages than expected!\n", len(msgs2))
			msgs = append(msgs, msgs2...)
		}
	}

	for _, msg := range msgs {
		// Compare resp to an expected one
		b, err := protojson.Marshal(msg)
		if err != nil {
			return err
		}
		msgStr := string(b)

		var matched bool
		var prettyReceivedMsgStr string
		var idMatched bool
		var ids map[string]string

		matchErrors := []error{}

		for c, expStr := range group.expectedMessages {
			ids, prettyReceivedMsgStr, err, idMatched = checkMatch(expStr, msgStr, s.user.userId)

			// Save the ids it may have found
			for k, v := range ids {
				if _, exists := s.idsCreated[k]; exists {
					return fmt.Errorf("Already have a saved value for id name: %v. Existing value: %v, new value: %v", k, s.idsCreated[k], v)
				}
				s.idsCreated[k] = v
			}

			if err != nil {
				// If ids were matched, we know not to scan any further
				if idMatched {
					matched = true
					fmt.Printf("%v\n", err)
					break
				} else {
					matchErrors = append(matchErrors, err)
				}

				// Otherwise, we continue looping because things may have arrived out of order
			} else {
				// Found a match, remove it
				matched = true
				group.expectedMessages = append(group.expectedMessages[:c], group.expectedMessages[c+1:]...)
				break
			}
		}

		if !matched {
			return fmt.Errorf("Received unmatched message: %v\nErrors encountered:\n%v\n", prettyReceivedMsgStr, matchErrors)
		}
	}

	s.printGroup(fmt.Sprintf("Matched %v messages", len(msgs)))

	// Should have none left
	if len(group.expectedMessages) > 0 {
		return fmt.Errorf("Failed to find match for %v expected messages", len(group.expectedMessages))
	}

	return nil
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
