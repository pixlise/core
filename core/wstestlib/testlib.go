package wstestlib

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
)

// An individual action that can be taken
// NOTE: one of these fields must be set
type actionItem struct {
	annotation string // Purely for logging of test
	defLine    string // source file+line that created this action

	// Individual actions that can happen:
	connect *client.ConnectInfo

	disconnect bool

	sendReq string
	// As part of sending, we can specify an expected response
	expectedResp string
	// And any other expected messages (updates?)
	expectedMsgs []string

	waitMs uint32
}

// After a group of actions is executed, we can specify
// multiple expected responses (or updates) for cases
// where we may want to send multiple requests out and
// capture all the randomly-ordered responses/updates
type actionGroup struct {
	actions          []actionItem
	expectedMessages []string
	timeoutMs        int
}

type ScriptedTestUser struct {
	auth0Params  client.Auth0Info
	user         *client.SocketConn
	actionGroups []actionGroup

	tempGroup *actionGroup

	groupIdx  int
	actionIdx int

	userNameConnected string
}

var savedItems = map[string]string{}

func MakeScriptedTestUser(auth0Params client.Auth0Info) ScriptedTestUser {
	return ScriptedTestUser{
		auth0Params:  auth0Params,
		user:         &client.SocketConn{},
		actionGroups: []actionGroup{},
	}
}
func (s *ScriptedTestUser) GetUserId() string {
	return s.user.UserId
}

// Use to reset a user, fails if called before all existing groups are complete
func (s *ScriptedTestUser) ClearActions() {
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

// Adding various action types
func (s *ScriptedTestUser) AddConnectAction(annotation string, params *client.ConnectInfo) {
	s.addAction(actionItem{annotation: annotation, connect: params})
}

func (s *ScriptedTestUser) AddDisonnectAction(annotation string) {
	s.addAction(actionItem{annotation: annotation, disconnect: true})
}

func (s *ScriptedTestUser) AddSendReqAction(annotation string, sendReq string, expectedResp string) {
	s.addAction(actionItem{annotation: annotation, sendReq: sendReq, expectedResp: expectedResp})
}

func (s *ScriptedTestUser) AddSleepAction(annotation string, sleepMs uint32) {
	s.addAction(actionItem{annotation: annotation, waitMs: sleepMs})
}

func (s *ScriptedTestUser) addAction(action actionItem) {
	// NOTE: at this point we assume we're called from 1 public function of this package, which itself
	// is called from somewhere important that we need to remember...
	action.defLine = GetCaller(3)

	if s.tempGroup == nil {
		s.tempGroup = &actionGroup{
			actions:          []actionItem{},
			expectedMessages: []string{},
		}
	}

	s.tempGroup.actions = append(s.tempGroup.actions, action)
}

func (s *ScriptedTestUser) CloseActionGroup(expectedMsgs []string, timeoutMs int) {
	// Close a group
	if s.tempGroup == nil {
		log.Fatal("Cannot add expected responses")
	}

	// Add responses to the group
	s.tempGroup.expectedMessages = expectedMsgs
	s.tempGroup.timeoutMs = timeoutMs

	// Also add the expected messages from each action
	for _, action := range s.tempGroup.actions {
		if len(action.expectedResp) > 0 {
			s.tempGroup.expectedMessages = append(s.tempGroup.expectedMessages, action.expectedResp)
		}

		if action.expectedMsgs != nil {
			s.tempGroup.expectedMessages = append(s.tempGroup.expectedMessages, action.expectedMsgs...)
		}
	}

	// Save the group
	s.actionGroups = append(s.actionGroups, *s.tempGroup)

	// Clear it
	s.tempGroup = nil
}

func GetIdCreated(name string) string {
	if val, ok := savedItems[name]; ok {
		return val
	}
	log.Fatalf("Failed to find saved ID named: %v", name)
	return ""
}

// Returns false if finished, error if there's an error
func (s *ScriptedTestUser) RunNextAction() (bool, error) {
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

func (s *ScriptedTestUser) runSpecificAction(action actionItem, which string) error {
	which = action.defLine + " " + which

	if action.connect != nil {
		s.printAction(which, action.annotation, fmt.Sprintf("Connecting to host: %v as user %v", action.connect.Host, action.connect.User))
		s.userNameConnected = action.connect.User
		return s.user.Connect(*action.connect, s.auth0Params)
	}

	if action.disconnect {
		s.printAction(which, action.annotation, fmt.Sprintf("Disconnecting user %v", s.userNameConnected))
		return s.user.Disconnect()
	}

	if action.waitMs > 0 {
		s.printAction(which, action.annotation, fmt.Sprintf("Waiting %vms", action.waitMs))
		time.Sleep(time.Millisecond * time.Duration(action.waitMs))
		return nil
	}

	if len(action.sendReq) > 0 {
		// Replace anything we need to before marshalling into proto bytes
		sendReqReplaced, err := doReqReplacements(action.sendReq, savedItems)
		if err != nil {
			log.Fatalln(err)
		}
		// Snip out the first line to send
		sendSnippet := sendReqReplaced
		linePos := strings.Index(sendSnippet, "\n")
		if linePos > 0 {
			sendSnippet = sendSnippet[0:linePos]
		}
		s.printAction(which, action.annotation, "" /*fmt.Sprintf("Sending req %v", sendSnippet)*/)

		wsmsg := protos.WSMessage{}
		err = protojson.Unmarshal([]byte(sendReqReplaced), &wsmsg)
		if err != nil {
			log.Fatalln(fmt.Errorf("Failed to parse request to be sent: %v.\nAction: %v\nRequest was: %v", err, action.annotation, sendReqReplaced))
		}
		return s.user.SendMessage(&wsmsg)
	}

	return errors.New("No action to take")
}

func (s *ScriptedTestUser) printAction(which string, annotation string, desc string) {
	if len(desc) > 0 {
		desc = "(" + desc + ")"
	}
	fmt.Printf("   - [%v] %v%v...\n", which, annotation, desc)
}

func (s *ScriptedTestUser) printGroup(desc string) {
	fmt.Printf(" => %v...\n", desc)
}

func (s *ScriptedTestUser) completeGroup(group actionGroup) error {
	s.printGroup(fmt.Sprintf("Waiting for %v messages or timeout of %vms", len(group.expectedMessages), group.timeoutMs))

	// Check that the expected msgs match
	msgs := s.user.WaitForMessages(len(group.expectedMessages), time.Duration(group.timeoutMs)*time.Millisecond)

	if len(group.expectedMessages) > 0 && len(msgs) <= 0 {
		return errors.New("No messages received")
	} else {
		// In case we got something more than the expected count of messages, put in a small wait here
		msgs2 := s.user.WaitForMessages(0, time.Duration(50)*time.Millisecond)
		if len(msgs2) > 0 {
			fmt.Printf("Received %v more messages than expected!\n", len(msgs2))
			msgs = append(msgs, msgs2...)
		}
	}

	// For debugger inspection only
	matchedMsgs := []string{}

	for _, msg := range msgs {
		// Compare resp to an expected one
		b, err := protojson.Marshal(msg)
		if err != nil {
			return err
		}
		msgStr := string(b)

		if len(group.expectedMessages) <= 0 {
			// If we've run out of expected messages in the mean time...
			return fmt.Errorf("Received unexpected message: %v\n", msgStr)
		}

		var matched bool
		var prettyReceivedMsgStr string
		var idMatched bool

		matchErrors := []error{}

		for c, expStr := range group.expectedMessages {
			prettyReceivedMsgStr, err, idMatched = checkMatch(expStr, msgStr, s.user.UserId, savedItems)

			if err != nil {
				// If ids were matched, we know not to scan any further
				if idMatched {
					matched = true
					fmt.Printf("idMatched with error: %v\n", err)
					break
				} else {
					matchErrors = append(matchErrors, err)
				}

				// Otherwise, we continue looping because things may have arrived out of order
			} else {
				// Found a match, remove it
				matched = true
				group.expectedMessages = append(group.expectedMessages[:c], group.expectedMessages[c+1:]...)
				matchedMsgs = append(matchedMsgs, expStr)
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
		return fmt.Errorf("Failed to find match for %v expected messages:\n%v", len(group.expectedMessages), strings.Join(group.expectedMessages, "\n"))
	}

	return nil
}
