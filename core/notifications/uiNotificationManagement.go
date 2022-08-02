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
	"fmt"
	"time"

	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"

	cmap "github.com/orcaman/concurrent-map"
)

type NotificationManager interface {
	FetchUserObject(userid string, createIfNotExist bool, name string, email string) (UserStruct, error)
	CreateUserObject(userid string, name string, email string) (UserStruct, error)
	UpdateUserConfigFile(userid string, data UserStruct) error
	SendEmail(topic string, templateInput map[string]interface{}, userOverride []string, userOverrideEmails []string, topiclookupoverride string, includeadmin bool) error
	SendUI(topic string, templateInput map[string]interface{}, userOverride []string, topiclookupoverride string) error
	SendAll(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool) error
	SendAllDataSource(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool, topicoverride string) error
	SendUINotification(newNotification UINotificationObj) error
	GetUINotifications(userid string) ([]UINotificationObj, error)
	GetTrack(userid string) (bool, bool)
	SetTrack(userid string, track bool)
	AddNotification(UINotificationObj)
	SendGlobalEmail(content string, subject string) error
}

//NotificationStack - Controller for UI Notifications
type NotificationStack struct {
	Notifications []UINotificationObj
	FS            fileaccess.FileAccess
	Bucket        string
	Track         cmap.ConcurrentMap //map[string]bool
	AdminEmails   []string
	Environment   string
	Logger        logger.ILogger
	Backend       string
}

//UINotificationObj - UI Notification Object
type UINotificationObj struct {
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userid"`
}

func (stack *NotificationStack) SetTrack(userid string, track bool) {
	//stack.Track[userid] = track
	stack.Track.Set(userid, track)
}
func (stack *NotificationStack) GetTrack(userid string) (bool, bool) {
	if val, ok := stack.Track.Get(userid); ok {
		return val.(bool), ok
	}
	return false, false
}

func (stack *NotificationStack) AddNotification(obj UINotificationObj) {
	stack.Notifications = append(stack.Notifications, obj)
}

//SendUINotification - Dispatch notification to the stack
func (stack *NotificationStack) SendUINotification(newNotification UINotificationObj) error {
	//Add time of arrival
	newNotification.Timestamp = time.Now()

	if stack.Backend == "S3" || stack.Backend == "" {
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
	} else if stack.Backend == "MONGO"{
		m := mongoutils{}
		return m.InsertUINotification(newNotification)
	}

	return nil
}

//GetUINotifications - Return Notifications for user and remove from stack
func (stack *NotificationStack) GetUINotifications(userid string) ([]UINotificationObj, error) {
	// REFACTOR: These paths should be coming from filepaths, we don't want random paths being built around the place, want to centralise so can document/edit easily
	path := "UserContent/notifications/" + userid + ".json"

	n := UserStruct{}
	if stack.Backend == "S3" || stack.Backend == "" {
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
	} else{
		m := mongoutils{}
		obj, err := m.GetUINotifications(userid)
		if err != nil{
			return nil, err
		}
		err = m.DeleteUINotifications(userid)
		if err != nil{
			return nil, err
		}
		return obj, nil
	}
}

func (stack *NotificationStack) GetAllUsers()([]UserStruct, error){
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.GetAllS3Users(stack.Environment, stack.Logger)
	} else{
		m := mongoutils{}
		return m.GetAllMongoUsers(stack.Logger)
	}

}

func (stack *NotificationStack) GetSubscribersByTopicID(useroverride []string, topic string)([]UserStruct, error){
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.GetS3SubscribersByTopicID(useroverride, topic, stack.Environment, stack.Logger)
	} else{
		m := mongoutils{}
		return m.GetMongoSubscribersByTopicID(useroverride, topic, stack.Logger)
	}

}

func (stack *NotificationStack) GetSubscribersByEmailTopicID(useroverride []string, topic string)([]UserStruct, error){
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.GetS3SubscribersByEmailTopicID(useroverride, topic, stack.Environment, stack.Logger)
	} else{
		m := mongoutils{}
		return m.GetMongoSubscribersByEmailTopicID(useroverride, topic, stack.Logger)
	}

}
func (stack *NotificationStack) GetSubscribersByTopic(topic string)([]UserStruct, error){
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.GetS3SubscribersByTopic(topic, stack.Environment, stack.Logger)
	} else{
		m := mongoutils{}
		return m.GetMongoSubscribersByTopic(topic, stack.Logger)
	}

}

func (stack *NotificationStack) CreateUserObject(userid string, name string, email string) (UserStruct, error){

	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.CreateS3UserObject(stack.InitUser(name, email, userid), stack.Bucket, stack.FS)
	} else{
		m := mongoutils{}
		us := stack.InitUser(name, email, userid)
		return us, m.CreateMongoUserObject(us)
	}
}

func (stack *NotificationStack) FetchUserObject(userid string, createIfNotExist bool, name string, email string) (UserStruct, error){
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.FetchS3UserObject(stack.FS, stack.Bucket, userid, createIfNotExist, name, email)
	} else{
		m := mongoutils{}
		return m.FetchMongoUserObject(userid, createIfNotExist, name, email)
	}
}

func (stack *NotificationStack) UpdateUserConfigFile(userid string, data UserStruct) error{
	if stack.Backend == "S3" || stack.Backend == ""{
		u := s3utils{}
		return u.UpdateS3UserConfigFile(stack.FS, stack.Bucket, userid, data)
	} else{
		m := mongoutils{}
		return m.UpdateMongoUserConfig(userid, data)
	}
}

func (stack *NotificationStack) InitUser(name string, email string, userid string) UserStruct {
	conf := Config{DataCollection: "unknown", Name: name, Email: email, Cell: ""}
	ncations := Notifications{Topics: []Topics{}, Hints: []string{}, UINotifications: []UINotificationObj{}}
	user := UserStruct{Userid: userid, Notifications: ncations, Config: conf}
	return user
}
