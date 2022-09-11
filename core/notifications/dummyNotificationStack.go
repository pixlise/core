// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package notifications

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
)

// REFACTOR: Top of this file is practically IDENTICAL to uiNotificationmanagement.go ???
// REFACTOR: Bottom of this file is practically IDENTICAL to emailManagement.go ???

type DummyNotificationStack struct {
	Notifications []UINotificationObj
	FS            fileaccess.FileAccess
	Bucket        string
	Track         map[string]bool
	AdminEmails   []string
	Environment   string
	Logger        logger.ILogger
}

func (stack *DummyNotificationStack) FetchUserObject(userid string, createIfNotExist bool, name string, email string) (UserStruct, error) {
	//TODO implement me
	panic("implement me")
}

func (stack *DummyNotificationStack) CreateUserObject(userid string, name string, email string) (UserStruct, error) {
	//TODO implement me
	panic("implement me")
}

func (stack *DummyNotificationStack) UpdateUserConfigFile(userid string, data UserStruct) error {
	//TODO implement me
	panic("implement me")
}

func (stack *DummyNotificationStack) SetTrack(userid string, track bool) {
	stack.Track[userid] = track
}
func (stack *DummyNotificationStack) GetTrack(userid string) (bool, bool) {
	if val, ok := stack.Track[userid]; ok {
		return val, ok
	}
	return false, false
}

func (stack *DummyNotificationStack) AddNotification(obj UINotificationObj) {
	stack.Notifications = append(stack.Notifications, obj)
}

// SendUINotification - Dispatch notification to the stack
func (stack *DummyNotificationStack) SendUINotification(newNotification UINotificationObj) error {
	//Add time of arrival
	newNotification.Timestamp = time.Now()

	path := "UserContent/notifications/" + newNotification.UserID + ".json"

	// REFACTOR: At least comment here why this would be nil?
	if stack.FS != nil {
		n := UserStruct{}
		err := stack.FS.ReadJSON(stack.Bucket, path, &n, false)
		if err != nil {
			return err
		}

		items := n.Notifications.UINotifications
		n.Notifications.UINotifications = append(items, newNotification)

		err = stack.FS.WriteJSONNoIndent(stack.Bucket, path, &n)
		if err != nil {
			return fmt.Errorf("Failed to upload notifications for user: %v", newNotification.UserID)
		}
	}

	return nil
}

// GetUINotifications - Return Notifications for user and remove from stack
func (stack *DummyNotificationStack) GetUINotifications(userid string) ([]UINotificationObj, error) {
	// REFACTOR: These paths should be coming from filepaths, we don't want random paths being built around
	// the place, want to centralise so can document/edit easily
	path := "UserContent/notifications/" + userid + ".json"

	n := UserStruct{}
	err := stack.FS.ReadJSON(stack.Bucket, path, &n, false)
	if err != nil {
		return nil, err
	}

	// Get the existing notifications
	notifications := n.Notifications.UINotifications
	if notifications == nil {
		notifications = []UINotificationObj{}
	}

	// Write an empty file back if it wasn't already empty...
	if len(notifications) > 0 {
		n.Notifications.UINotifications = []UINotificationObj{}

		err = stack.FS.WriteJSONNoIndent(stack.Bucket, path, &n)
		if err != nil {
			fmt.Printf("Failed to write cleared notification file for user: %v. Error: %v\n", userid, err)
		}
	}

	return notifications, nil
}

// SendGlobalEmail - Send an email to all users.
func (stack *DummyNotificationStack) SendGlobalEmail(content string, subject string) error {

	stack.Logger.Infof("Sending dummy email. \n Subject: %v\n Content: %v", content, subject)

	return nil
}

// SendEmail - Send an email for a topic type
func (stack *DummyNotificationStack) SendEmail(topic string, templateInput map[string]interface{}, userOverride []string, userOverrideEmails []string, topiclookupoverride string, includeadmin bool) error {

	sub := UserStruct{
		Userid: "123",
		Notifications: Notifications{
			Topics:          nil,
			Hints:           nil,
			UINotifications: nil,
		},
		Config: Config{
			Name:           "Tom Barber",
			Email:          "tom@spicule.co.uk",
			Cell:           "",
			DataCollection: "",
		},
	}
	text, f := generateEmailContent(sub, topic, templateInput, "TXT")
	if f != nil {
		return f
	}
	stack.Logger.Infof("Generated Email Content. Text: %v", text)
	return nil
}

//SendSms sends an SMS message via AWS SNS
//func (stack *NotificationStack) SendSms(topic string, templateInput map[string]interface{}, userOverride []string) error {
//	var subscribers []UserStruct
//	var err error
//	if userOverride != nil {
//		subscribers, err = GetS3SubscribersByTopicID(userOverride, topic, stack.Environment, stack.Logger)
//	} else {
//		subscribers, err = GetS3SubscribersByTopic(topic, stack.Environment, stack.Logger)
//	}
//
//	if err != nil {
//		return err
//	}
//	fmt.Println("SMS Subs found: " + strconv.Itoa(len(subscribers)))
//
//	for _, sub := range subscribers {
//		if sub.Topics[0].Config.Method.Sms == true {
//			text, err := generateTxtEmailContent(sub, topic+"-sms", templateInput)
//			if err != nil {
//				return err
//			}
//
//			err = awsutil.SNSSendSms(sub.Cell, text)
//			if err != nil {
//				return err
//			}
//		}
//
//	}
//
//	return nil
//}

// SendUI Send a notification to the UI stack.
func (stack *DummyNotificationStack) SendUI(topic string, templateInput map[string]interface{}, userOverride []string, topiclookupoverride string) error {
	sub := UserStruct{
		Userid: "123",
		Notifications: Notifications{
			Topics:          nil,
			Hints:           nil,
			UINotifications: nil,
		},
		Config: Config{
			Name:           "Tom Barber",
			Email:          "tom@spicule.co.uk",
			Cell:           "",
			DataCollection: "",
		},
	}
	text, err := generateEmailContent(sub, topic+"-ui", templateInput, "TXT")
	fmt.Println("Adding notification to UI stack: " + sub.Userid)
	if err != nil {
		return err
	}
	notification := UINotificationObj{
		Topic:     topic,
		Message:   text,
		Timestamp: time.Time{},
		UserID:    sub.Userid,
	}
	err = stack.SendUINotification(notification)

	return nil
}

// SendAll - Generic function to trigger the same notification across all services to users who are signed up to receive them.
func (stack *DummyNotificationStack) SendAll(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool) error {
	err := stack.SendEmail(topic, templateInput, userOverride, nil, "", includeadmin)
	if err != nil {
		return err
	}
	//err = stack.SendSms(topic, templateInput, userOverride)
	//if err != nil {
	//	return err
	//}
	err = stack.SendUI(topic, templateInput, userOverride, "")
	if err != nil {
		return err
	}

	return nil
}

// SendAllDataSource - Slight tweak to allow us to lookup topics for updates by the same tag and define emails instead of userids
func (stack *DummyNotificationStack) SendAllDataSource(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool, topicoverride string) error {
	err := stack.SendEmail(topic, templateInput, userOverride, nil, topicoverride, includeadmin)
	if err != nil {
		return err
	}
	//err = stack.SendSms(topic, templateInput, userOverride)
	//if err != nil {
	//	return err
	//}
	err = stack.SendUI(topic, templateInput, userOverride, topicoverride)
	if err != nil {
		return err
	}

	return nil
}
