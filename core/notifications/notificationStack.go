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

	"go.mongodb.org/mongo-driver/mongo"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/timestamper"
)

// NotificationStack - Thing that manages user notifications on UI and email including preferences
type NotificationStack struct {
	//Notifications []UINotificationItem
	Track       cmap.ConcurrentMap //map[string]bool
	adminEmails []string
	//Environment   string

	mongoClient *mongo.Client

	userDatabase           *mongo.Database
	userCollection         *mongo.Collection
	notificationCollection *mongo.Collection

	timestamper timestamper.ITimeStamper // So we can mock time.Now()

	log logger.ILogger
}

func MakeNotificationStack(mongoClient *mongo.Client, envName string, timestamper timestamper.ITimeStamper, log logger.ILogger, adminEmails []string) (*NotificationStack, error) {
	userDatabaseName := "userdatabase"
	if envName != "prod" {
		userDatabaseName += "-" + envName
	}

	userDatabase := mongoClient.Database(userDatabaseName)
	userCollection := userDatabase.Collection("users")
	notificationCollection := userDatabase.Collection("notifications")

	stack := &NotificationStack{
		Track:                  cmap.New(),
		mongoClient:            mongoClient,
		userDatabase:           userDatabase,
		userCollection:         userCollection,
		notificationCollection: notificationCollection,
		timestamper:            timestamper,
		log:                    log,
		adminEmails:            adminEmails,
	}

	return stack, nil
}

func (n *NotificationStack) SetTrack(userid string, track bool) {
	n.Track.Set(userid, track)
}

func (n *NotificationStack) GetTrack(userid string) (bool, bool) {
	if val, ok := n.Track.Get(userid); ok {
		return val.(bool), ok
	}
	return false, false
}

/*
func (n *NotificationStack) AddNotification(obj UINotificationItem) {
	n.Notifications = append(n.Notifications, obj)
}
*/

// SendGlobalEmail - Send an email to all users.
func (n *NotificationStack) SendGlobalEmail(content string, subject string) error {
	users, err := n.getAllUsers()
	if err != nil {
		return err
	}
	var bcc []string
	for _, u := range users {
		bcc = append(bcc, u.Config.Email)
	}

	awsutil.SESSendEmail("info@mail.pixlise.org", "UTF-8", content,
		"", subject,
		"info@mail.pixlise.org", []string{}, bcc)

	return nil
}

// SendEmail - Send an email for a topic type
func (n *NotificationStack) sendEmail(topic string, templateInput map[string]interface{}, userOverride []string, userOverrideEmails []string, topiclookupoverride string, includeadmin bool) error {
	var subscribers []UserStruct
	var err error
	var searchtopic = topic
	if topiclookupoverride != "" {
		searchtopic = topiclookupoverride
	}
	if userOverride != nil {
		subscribers, err = n.getSubscribersByTopicID(userOverride, searchtopic)
	} else if userOverrideEmails != nil {
		subscribers, err = n.getSubscribersByEmailTopicID(userOverride, searchtopic)
	} else {
		subscribers, err = n.getSubscribersByTopic(searchtopic)
	}

	if err != nil {
		return err
	}
	n.log.Debugf("Email Subs found: %v", len(subscribers))
	for i, sub := range subscribers {
		n.log.Debugf("Sub %v : %v, enabled: %v", i, sub.Config.Name, sub.Notifications.Topics[0].Config.Method.Email)
		if sub.Notifications.Topics[0].Config.Method.Email == true {
			n.log.Debugf("Generating Email Content")
			html, e := generateEmailContent(n.log, sub, topic, templateInput, "HTML")
			if e != nil {
				n.log.Errorf("generateEmailContent error: %v", e)
				return e
			}
			n.log.Debugf("Generating Text Email Content")
			text, f := generateEmailContent(n.log, sub, topic, templateInput, "TXT")
			if f != nil {
				return f
			}

			n.log.Debugf("Setting subject and sender")
			subject := fmt.Sprintf("%v", templateInput["subject"])
			sender := fmt.Sprintf("%v", "info@mail.pixlise.org")
			n.log.Infof("Sending %v, to %v", subject, sender)
			//Needs extracting to config
			var adminaddresses []string
			if includeadmin {
				adminaddresses = n.adminEmails
			}
			awsutil.SESSendEmail(sub.Config.Email, charSet, text, html, subject, sender, adminaddresses, []string{})
		}
	}

	return nil
}

// SendUI - Send a notification to the UI specifically
func (n *NotificationStack) sendUI(topic string, templateInput map[string]interface{}, userOverride []string, topiclookupoverride string) error {
	var subscribers []UserStruct
	var err error
	var searchtopic = topic
	if topiclookupoverride != "" {
		searchtopic = topiclookupoverride
	}
	if userOverride != nil {
		subscribers, err = n.getSubscribersByTopicID(userOverride, searchtopic)
	} else {
		subscribers, err = n.getSubscribersByTopic(searchtopic)
	}
	if err != nil {
		return err
	}

	n.log.Debugf("UI Subs found: %v", len(subscribers))

	for _, sub := range subscribers {
		if sub.Notifications.Topics[0].Config.Method.UI == true {
			text, err := generateEmailContent(n.log, sub, topic, templateInput, "UI")
			n.log.Debugf("Adding notification to UI stack: %v", sub.Userid)
			if err != nil {
				return err
			}
			notification := UINotificationItem{
				Topic:     topic,
				Message:   text,
				Timestamp: time.Time{},
				UserID:    sub.Userid,
			}
			err = n.SendUINotification(notification)
		}
	}
	return nil
}

// SendAll - Generic function to trigger the same notification across all services to users who are signed up to receive them.
func (n *NotificationStack) SendAll(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool) error {
	err := n.sendEmail(topic, templateInput, userOverride, nil, "", includeadmin)
	if err != nil {
		return err
	}

	// We used to send SMS here too!

	err = n.sendUI(topic, templateInput, userOverride, "")
	if err != nil {
		return err
	}

	return nil
}

// SendAllDataSource - Slight tweak to allow us to lookup topics for updates by the same tag and define emails instead of userids
func (n *NotificationStack) SendAllDataSource(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool, topicoverride string) error {
	err := n.sendEmail(topic, templateInput, userOverride, nil, topicoverride, includeadmin)
	if err != nil {
		return err
	}

	// We used to send SMS here too!

	err = n.sendUI(topic, templateInput, userOverride, topicoverride)
	if err != nil {
		return err
	}

	return nil
}
