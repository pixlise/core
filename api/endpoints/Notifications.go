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

package endpoints

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	apiNotifications "github.com/pixlise/core/v2/core/notifications"

	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/awsutil"
)

// Method - Subscription methods
type Method struct {
	UI    bool `json:"ui"`
	Sms   bool `json:"sms"`
	Email bool `json:"email"`
}

// Config - List of configurations from App Metadata.
type Config struct {
	Cell    string `json:"cell"`
	Methods Method `json:"method"`
}

//AppData - App data type for JSON conversion
type AppData struct {
	Topics []apiNotifications.Topics `json:"topics"`
}

//HintsData - Hints Object
type HintsData struct {
	Hints []string `json:"hints"`
}

//TestData - JSON Data for test emails
type TestData struct {
	TestType    string `json:"type"`
	TestContact string `json:"contact"`
}

//GlobalData - JSON Data for global emails
type GlobalData struct {
	GlobalContent string `json:"content"`
	GlobalSubject string `json:"subject"`
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
// Notification management.
const alertsPrefix = "notification/alerts"

func registerNotificationHandler(router *apiRouter.ApiObjectRouter) {
	const subscriptionPrefix = "notification/subscriptions"
	const hintsPrefix = "notification/hints"
	const testPrefix = "notification/test"
	const globalPrefix = "notification/global"

	router.AddJSONHandler(handlers.MakeEndpointPath(subscriptionPrefix, "userid"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), listSubscriptions)
	router.AddJSONHandler(handlers.MakeEndpointPath(subscriptionPrefix), apiRouter.MakeMethodPermission("GET", permission.PermPublic), listSubscriptions)

	router.AddJSONHandler(handlers.MakeEndpointPath(subscriptionPrefix, "userid"), apiRouter.MakeMethodPermission("POST", permission.PermWriteUserRoles), updateSubscriptions)
	router.AddJSONHandler(handlers.MakeEndpointPath(subscriptionPrefix), apiRouter.MakeMethodPermission("POST", permission.PermPublic), updateSubscriptions)

	router.AddJSONHandler(handlers.MakeEndpointPath(hintsPrefix), apiRouter.MakeMethodPermission("GET", permission.PermPublic), listHints)
	router.AddJSONHandler(handlers.MakeEndpointPath(hintsPrefix), apiRouter.MakeMethodPermission("POST", permission.PermPublic), updateHints)

	router.AddJSONHandler(handlers.MakeEndpointPath(alertsPrefix), apiRouter.MakeMethodPermission("GET", permission.PermPublic), listAlerts)

	router.AddJSONHandler(handlers.MakeEndpointPath(testPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWritePiquantConfig), executeTest)
	router.AddJSONHandler(handlers.MakeEndpointPath(globalPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWritePiquantConfig), globalNotification)
}

func globalNotification(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req GlobalData
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}
	err = params.Svcs.Notifications.SendGlobalEmail(req.GlobalContent, req.GlobalSubject)

	return nil, err
}

func executeTest(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req TestData
	err = json.Unmarshal(body, &req)
	if req.TestType == "sms" {
		err := awsutil.SNSSendSms(req.TestContact, "THIS IS A TEST SMS MESSAGE. PLEASE DISREGARD")
		if err != nil {
			fmt.Printf("%v", err)
		}

	} else if req.TestType == "email" {
		awsutil.SESSendEmail(req.TestContact, "UTF-8", "TEST EMAIL PLEASE DISREGARD",
			"<html><p>TEST EMAIL PLEASE DISREGARD</p></html>", "PIXLISE TEST EMAIL",
			"info@mail.pixlise.org", []string{}, []string{})
	}
	return nil, nil
}

func listAlerts(params handlers.ApiHandlerParams) (interface{}, error) {
	notification, err := params.Svcs.Notifications.GetUINotifications(params.UserInfo.UserID)

	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return []apiNotifications.UINotificationObj{}, nil
		}
		return nil, err
	}
	if notification == nil {
		s := []string{}
		return s, nil
	}
	return notification, nil
}

func listHints(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)

	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return []string{}, nil
		}
		return nil, err
	}

	return HintsData{user.Hints}, err
}

func updateHints(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req HintsData
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)

	if err != nil {
		if params.Svcs.Config.MongoEndpoint == "" && params.Svcs.FS.IsNotFoundError(err) {
			user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		} else if params.Svcs.Config.MongoEndpoint != "" {
			user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		}

		if err != nil {
			return nil, err
		}
	}

	user.Hints = req.Hints
	err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)

	if err != nil {
		return nil, err
	}

	return HintsData{user.Hints}, nil
}

func listSubscriptions(params handlers.ApiHandlerParams) (interface{}, error) {
	if val, ok := params.PathParams["userid"]; ok {
		if perm, ok := params.UserInfo.Permissions["read:user-roles"]; ok {
			if perm == true {
				user, err := params.Svcs.Notifications.FetchUserObject(val, true, params.UserInfo.Name, params.UserInfo.Email)
				if err != nil {
					if params.Svcs.FS.IsNotFoundError(err) {
						return AppData{}, nil
					}
					return nil, err
				}
				return user.Topics, nil
			}

			return "Unable to lookup userid by user, check your permissions", nil
		}

		return "Unable to lookup userid by user, check your permissions", nil
	}
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return AppData{}, nil
		}
		return nil, err
	}
	return AppData{user.Topics}, nil
}

func updateSubscriptions(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}
	var req AppData
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	if val, ok := params.PathParams["userid"]; ok {
		if perm, ok := params.UserInfo.Permissions[permission.PermWriteUserRoles]; ok {
			if perm == true {
				user, err := params.Svcs.Notifications.FetchUserObject(val, true, params.UserInfo.Name, params.UserInfo.Email)
				if err != nil {
					if params.Svcs.Config.MongoEndpoint == "" && params.Svcs.FS.IsNotFoundError(err) {
						user, err = params.Svcs.Notifications.CreateUserObject(val, params.UserInfo.Name, params.UserInfo.Email)
					} else if params.Svcs.Config.MongoEndpoint != "" {
						user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
					}

					if err != nil {
						return nil, err
					}
				}
				user.Topics = req.Topics

				err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)
				return user.Topics, nil
			}

			return "Unable to lookup userid by user, check your permissions", nil
		}

		return "Unable to lookup userid by user, check your permissions", nil
	}

	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		params.Svcs.Log.Errorf("Error Fetching User Object: %v", err)
		err = nil
		if params.Svcs.Config.MongoEndpoint == "" && params.Svcs.FS.IsNotFoundError(err) {
			user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		} else if params.Svcs.Config.MongoEndpoint != "" {
			user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		}

		if err != nil {
			//return nil, err
			params.Svcs.Log.Errorf("Error Fetching User Object: %v", err)
		}
	} else if user.Userid == "" {
		user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
	}
	user.Topics = req.Topics
	err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)
	if err != nil {
		return nil, err
	}

	return AppData{user.Topics}, nil
}
