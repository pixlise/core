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
	"errors"
	"time"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"

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

// NotificationStack - Controller for UI Notifications
type NotificationStack struct {
	Notifications []UINotificationObj
	FS            fileaccess.FileAccess
	Bucket        string
	Track         cmap.ConcurrentMap //map[string]bool
	AdminEmails   []string
	Environment   string
	Logger        logger.ILogger
	MongoUtils    *MongoUtils
}

// UINotificationObj - UI Notification Object
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

// SendUINotification - Dispatch notification to the stack
func (stack *NotificationStack) SendUINotification(newNotification UINotificationObj) error {
	//Add time of arrival
	newNotification.Timestamp = time.Now()

	if stack.MongoUtils == nil {
		return errors.New("Mongo not configured")
	}

	return stack.MongoUtils.InsertUINotification(newNotification)
}

// GetUINotifications - Return Notifications for user and remove from stack
func (stack *NotificationStack) GetUINotifications(userid string) ([]UINotificationObj, error) {
	if stack.MongoUtils == nil {
		return nil, errors.New("Mongo not configured")
	}

	if stack.MongoUtils.MongoEndpoint == "" {
		return []UINotificationObj{}, nil
	}
	obj, err := stack.MongoUtils.GetUINotifications(userid)
	if err != nil {
		return nil, err
	}
	err = stack.MongoUtils.DeleteUINotifications(userid)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (stack *NotificationStack) GetAllUsers() ([]UserStruct, error) {
	if stack.MongoUtils == nil {
		return nil, errors.New("Mongo not configured")
	}

	return stack.MongoUtils.GetAllMongoUsers(stack.Logger)
}

func (stack *NotificationStack) GetSubscribersByTopicID(useroverride []string, topic string) ([]UserStruct, error) {
	if stack.MongoUtils == nil {
		return nil, errors.New("Mongo not configured")
	}

	return stack.MongoUtils.GetMongoSubscribersByTopicID(useroverride, topic, stack.Logger)
}

func (stack *NotificationStack) GetSubscribersByEmailTopicID(useroverride []string, topic string) ([]UserStruct, error) {
	if stack.MongoUtils == nil {
		return nil, errors.New("Mongo not configured")
	}

	return stack.MongoUtils.GetMongoSubscribersByEmailTopicID(useroverride, topic, stack.Logger)
}

func (stack *NotificationStack) GetSubscribersByTopic(topic string) ([]UserStruct, error) {
	if stack.MongoUtils == nil {
		return nil, errors.New("Mongo not configured")
	}

	return stack.MongoUtils.GetMongoSubscribersByTopic(topic, stack.Logger)
}

func (stack *NotificationStack) CreateUserObject(userid string, name string, email string) (UserStruct, error) {
	us := stack.InitUser(name, email, userid)

	if stack.MongoUtils == nil {
		return us, errors.New("Mongo not configured")
	}

	return us, stack.MongoUtils.CreateMongoUserObject(us)
}

func (stack *NotificationStack) FetchUserObject(userid string, createIfNotExist bool, name string, email string) (UserStruct, error) {
	if stack.MongoUtils == nil {
		return UserStruct{}, errors.New("Mongo not configured")
	}

	return stack.MongoUtils.FetchMongoUserObject(userid, createIfNotExist, name, email)
}

func (stack *NotificationStack) UpdateUserConfigFile(userid string, data UserStruct) error {
	if stack.MongoUtils == nil {
		return errors.New("Mongo not configured")
	}

	return stack.MongoUtils.UpdateMongoUserConfig(userid, data)
}

func (stack *NotificationStack) InitUser(name string, email string, userid string) UserStruct {
	conf := Config{DataCollection: "unknown", Name: name, Email: email, Cell: ""}
	ncations := Notifications{Topics: []Topics{}, Hints: []string{}, UINotifications: []UINotificationObj{}}
	user := UserStruct{Userid: userid, Notifications: ncations, Config: conf}
	return user
}
