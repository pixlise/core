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

type NotificationManager interface {
	// User specific
	GetUser(userid string) (UserStruct, error)
	GetUserEnsureExists(userid string, name string, email string) (UserStruct, error)
	WriteUser(data UserStruct) error

	// Sending notifications
	SendGlobalEmail(content string, subject string) error
	SendAll(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool) error
	SendAllDataSource(topic string, templateInput map[string]interface{}, userOverride []string, includeadmin bool, topicoverride string) error

	// UI-specific notifications
	SendUINotification(newNotification UINotificationItem) error
	GetUINotifications(userid string) ([]UINotificationItem, error)

	// Run-time cached list of users who are tracked
	GetTrack(userid string) (bool, bool)
	SetTrack(userid string, track bool)

	//AddNotification(UINotificationItem)
}
