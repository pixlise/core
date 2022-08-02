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

package endpoints

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	apiNotifications "github.com/pixlise/core/core/notifications"

	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/core/awsutil"
)

//Method - Subscription methods
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
		//TODO SWAP FOR MONGO
		if params.Svcs.FS.IsNotFoundError(err) {
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
					//TODO SWAP FOR MONGO
					if params.Svcs.FS.IsNotFoundError(err) {
						user, err = params.Svcs.Notifications.CreateUserObject(val, params.UserInfo.Name, params.UserInfo.Email)
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
		//TODO SWAP FOR MONGO
		if params.Svcs.FS.IsNotFoundError(err) {
			user, err = params.Svcs.Notifications.CreateUserObject(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		}
		if err != nil {
			return nil, err
		}
	}
	user.Topics = req.Topics
	err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)
	if err != nil {
		return nil, err
	}

	return AppData{user.Topics}, nil
}
