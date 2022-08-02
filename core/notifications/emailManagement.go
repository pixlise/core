// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package notifications

import (
	"bytes"
	"fmt"
	"github.com/pixlise/core/core/notifications/templates"
	"html/template"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pixlise/core/core/awsutil"
)

const (
	charSet = "UTF-8"
)

// Method - Notification Methods
type Method struct {
	UI    bool `json:"ui"`
	Sms   bool `json:"sms"`
	Email bool `json:"email"`
}

// NotificationConfig - Config specifically for notifications
type NotificationConfig struct {
	Method `json:"method"`
}

// Topics - Notification Topics'
type Topics struct {
	Name   string             `json:"name"`
	Config NotificationConfig `json:"config"`
}

// Notifications - Object for notification settings
type Notifications struct {
	Topics          []Topics            `json:"topics"`
	Hints           []string            `json:"hints"`
	UINotifications []UINotificationObj `json:"uinotifications"`
}

// Config - config options for user
type Config struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Cell           string `json:"cell"`
	DataCollection string `json:"data_collection"`
}

// UserStruct - Structure for user configuration
type UserStruct struct {
	Userid        string `json:"userid"`
	Notifications `json:"notifications"`
	Config        `json:"userconfig"`
}

//TemplateContents - Structure for template injection
type TemplateContents struct {
	ContentMap  UserStruct
	TemplateMap map[string]interface{}
}

//SendGlobalEmail - Send an email to all users.
func (stack *NotificationStack) SendGlobalEmail(content string, subject string) error {

	users, err := stack.GetAllUsers()
	if err != nil{
		return err
	}
	var bcc []string
	for _, u := range users {
		bcc = append(bcc, u.Email)
	}

	awsutil.SESSendEmail("info@mail.pixlise.org", "UTF-8", content,
		"", subject,
		"info@mail.pixlise.org", []string{}, bcc)

	return nil
}

//SendEmail - Send an email for a topic type
func (stack *NotificationStack) SendEmail(topic string, templateInput map[string]interface{}, userOverride []string, userOverrideEmails []string, topiclookupoverride string, includeadmin bool) error {
	var subscribers []UserStruct
	var err error
	var searchtopic = topic
	if topiclookupoverride != "" {
		searchtopic = topiclookupoverride
	}
		if userOverride != nil {
			subscribers, err = stack.GetSubscribersByTopicID(userOverride, searchtopic)
		} else if userOverrideEmails != nil {
			subscribers, err = stack.GetSubscribersByEmailTopicID(userOverride, searchtopic)
		} else {
			subscribers, err = stack.GetSubscribersByTopic(searchtopic)
		}

	if err != nil {
		return err
	}
	fmt.Println("Email Subs found: " + strconv.Itoa(len(subscribers)))
	for i, sub := range subscribers {
		stack.Logger.Debugf("Sub %v : %v, enabled: %v", i, sub.Name, sub.Topics[0].Config.Method.Email)
		if sub.Topics[0].Config.Method.Email == true {
			fmt.Println("Generating Email Content")
			html, e := generateEmailContent(sub, topic, templateInput, "HTML")
			if e != nil {
				fmt.Printf("Error Found: %v", e.Error())
				return e
			}
			fmt.Println("Generating Text Email Content")
			text, f := generateEmailContent(sub, topic, templateInput, "TXT")
			if f != nil {
				return f
			}

			fmt.Println("Setting subject and sender")
			subject := fmt.Sprintf("%v", templateInput["subject"])
			sender := fmt.Sprintf("%v", "info@mail.pixlise.org")
			fmt.Printf("Sending %v, to %v", subject, sender)
			//Needs extracting to config
			var adminaddresses []string
			if includeadmin {
				adminaddresses = stack.AdminEmails
			}
			awsutil.SESSendEmail(sub.Email, charSet, text, html, subject, sender, adminaddresses, []string{})
		}
	}

	return nil
}

// SendUI Send a notification to the UI stack.
func (stack *NotificationStack) SendUI(topic string, templateInput map[string]interface{}, userOverride []string, topiclookupoverride string) error {
	var subscribers []UserStruct
	var err error
	var searchtopic = topic
	if topiclookupoverride != "" {
		searchtopic = topiclookupoverride
	}
	if userOverride != nil {
		subscribers, err = stack.GetSubscribersByTopicID(userOverride, searchtopic)
	} else {
		subscribers, err = stack.GetSubscribersByTopic(searchtopic)
	}
	if err != nil {
		return err
	}
	fmt.Println("UI Subs found: " + strconv.Itoa(len(subscribers)))

	for _, sub := range subscribers {
		if sub.Topics[0].Config.Method.UI == true {
			text, err := generateEmailContent(sub, topic, templateInput, "UI")
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
		}
	}
	return nil
}

// SendAll - Generic function to trigger the same notification across all services to users who are signed up to receive them.
func (stack *NotificationStack) SendAll(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool) error {
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
func (stack *NotificationStack) SendAllDataSource(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool, topicoverride string) error {
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

func generateEmailContent(subscriber UserStruct, templateName string, templateInput map[string]interface{}, format string) (string, error) {
	path, err := os.Getwd()
	if err != nil {
		log.Println(path)
	}
	t := textTemplates.GetTemplates()
	var templates = template.Must(template.New(templateName).Parse(t[templateName+"-"+format]))
	if err != nil {
		fmt.Println("Failed to read template strings")
		return "", err
	}

	//fmt.Println("Generating Contents")
	inv := TemplateContents{ContentMap: subscriber, TemplateMap: templateInput}
	var tpl bytes.Buffer
	//fmt.Printf("Executing Template: %v, %v", templateName, inv)
	err = templates.ExecuteTemplate(&tpl, templateName, inv)
	if err != nil {
		fmt.Printf("Failed to generate template: %v \n", err.Error())
		return "", err
	}
	result := tpl.String()
	return result, nil

}
