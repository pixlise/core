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
	"io/ioutil"
	"time"

	apiNotifications "github.com/pixlise/core/core/notifications"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"gopkg.in/auth0.v4/management"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// User management, by some sort of Admin users. Ability to see all users & assign roles to them

type dataCollection struct {
	Collect string `json:"collect"`
}

// UserInfo - the structure describing a user from our management API. This largly reflects
// the fields in Auth0's User structure
type auth0UserInfo struct {
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	CreatedUnixSec   int64    `json:"created_at"`
	LastLoginUnixSec int64    `json:"last_login"`
	Picture          string   `json:"picture"`
	Roles            []string `json:"roles,omitempty"`
}

type roleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const userIDIdentifier = "user_id"
const unassignedNewUserRoleID = "rol_BDm6RvOwIGqxSbYt" // "Unassigned New User" role

func registerUserManagementHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "user"

	// Various ways of querying users
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/query"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userListQuery)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/by-role", idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userListByRole)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/by-id", idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/roles", userIDIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userGetRoles)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/roles", userIDIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteUserRoles), userPostRoles)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/roles", userIDIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteUserRoles), userDeleteRoles)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/config"), apiRouter.MakeMethodPermission("POST", permission.PermReadUserRoles), userPostConfig)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/config"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userGetConfig)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/data-collection"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), userGetDataCollection)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/data-collection"), apiRouter.MakeMethodPermission("POST", permission.PermReadUserRoles), userPostDataCollection)
	// Simply retrieves roles
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/all-roles"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), roleList)
}

func roleList(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	// Get roles for each
	gotRoles, err := api.Role.List()
	if err != nil {
		return nil, err
	}

	roles := makeRoleList(gotRoles)
	return roles, nil
}

func userListByRole(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	id := params.PathParams[idIdentifier]
	result := []auth0UserInfo{}

	// Slow for now, but works... don't have a lot of users so probably all on 1 page!
	// TODO: if we have speed issues, paginate our own API
	var page int
	for {
		users, err := api.Role.Users(id, management.Page(page))
		if err != nil {
			return nil, err
		}

		result = append(result, makeUserList(users)...)

		if !users.HasNext() {
			break
		}
		page++
	}

	return result, err
}

func userGetDataCollection(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}

	return dataCollection{Collect: user.Config.DataCollection}, nil
}

func userPostDataCollection(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req dataCollection
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	user.Config.DataCollection = req.Collect
	err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)
	if err != nil {
		return nil, err
	}

	if req.Collect == "true" {
		params.Svcs.Notifications.SetTrack(params.UserInfo.UserID, true)
	} else {
		params.Svcs.Notifications.SetTrack(params.UserInfo.UserID, false)
	}

	return "Success", nil
}

func userGet(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	id := params.PathParams[idIdentifier]
	user, err := api.User.Read(id)
	if err != nil {
		return nil, err
	}

	return makeUser(user), nil
}

func userGetRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	userID := params.PathParams[userIDIdentifier]
	gotRoles, err := api.User.Roles(userID)
	if err != nil {
		return nil, err
	}

	roles := makeRoleList(gotRoles)
	return roles, nil
}

func userPostRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	roleID := params.PathParams[idIdentifier]
	userID := params.PathParams[userIDIdentifier]

	unassignNeeded := false

	if roleID != unassignedNewUserRoleID {
		// If the user has the role "Unassigned New User" and is being assigned another role, we clear
		// Unassigned New User because an admin user may not know to remove it and it would confuse other things
		roleResp, err := api.User.Roles(userID)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to query user roles when new role being assigned: %v", err)
		} else {
			for _, r := range roleResp.Roles {
				if r.GetID() == unassignedNewUserRoleID {
					// Yes, we do need to unassign the existing role
					unassignNeeded = true
				}
			}
		}

		// Don't flood Auth0 with requests!
		time.Sleep(1200 * time.Millisecond)
	}

	if unassignNeeded {
		params.Svcs.Log.Infof("User %v is being assigned role %v. The existing \"Unassigned New User\" role is being automatically removed", userID, roleID)

		roleToUnassign := unassignedNewUserRoleID
		err = api.User.RemoveRoles(userID, &management.Role{ID: &roleToUnassign})
		if err != nil {
			params.Svcs.Log.Errorf("Failed to remove \"Unassigned New User\" role when user role is changing: %v", err)
		}

		// Don't flood Auth0 with requests!
		time.Sleep(1200 * time.Millisecond)
	}

	err = api.User.AssignRoles(userID, &management.Role{ID: &roleID})
	return nil, err
}

func userDeleteRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	roleID := params.PathParams[idIdentifier]
	userID := params.PathParams[userIDIdentifier]
	err = api.User.RemoveRoles(userID, &management.Role{ID: &roleID})
	return nil, err
}

func userListQuery(params handlers.ApiHandlerParams) (interface{}, error) {
	api, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	var page int
	result := []auth0UserInfo{}

	for {
		l, err := api.User.List(
			management.Query(""), //`logins_count:{100 TO *]`),
			management.Page(page),
		)
		if err != nil {
			return nil, err
		}

		result = append(result, makeUserList(l)...)

		if !l.HasNext() {
			break
		}
		page++
	}

	return result, err
}

func userGetConfig(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}
	return user.Config, nil
}

func userPostConfig(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Notifications.FetchUserObject(params.UserInfo.UserID, true, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}
	var req apiNotifications.Config
	err = json.Unmarshal(body, &req)

	user.Config = req
	err = params.Svcs.Notifications.UpdateUserConfigFile(params.UserInfo.UserID, user)

	return req, err
}

func makeUserList(from *management.UserList) []auth0UserInfo {
	users := []auth0UserInfo{}

	for _, u := range from.Users {
		user := makeUser(u)
		users = append(users, user)
	}

	return users
}

func makeUser(from *management.User) auth0UserInfo {
	user := auth0UserInfo{
		UserID:  from.GetID(),
		Name:    from.GetName(),
		Email:   from.GetEmail(),
		Picture: from.GetPicture(),
		//Roles - not returned here
	}

	// These may not be there...
	if from.CreatedAt != nil {
		user.CreatedUnixSec = from.GetCreatedAt().Unix()
	}
	if from.LastLogin != nil {
		user.LastLoginUnixSec = from.GetLastLogin().Unix()
	}

	return user
}

func makeRoleList(from *management.RoleList) []roleInfo {
	var roles []roleInfo

	for _, r := range from.Roles {
		role := roleInfo{
			ID:          r.GetID(),
			Name:        r.GetName(),
			Description: r.GetDescription(),
		}
		roles = append(roles, role)
	}

	return roles
}

// InitAuth0ManagementAPI - bootstrap auth0
func InitAuth0ManagementAPI(cfg config.APIConfig) (*management.Management, error) {
	api, err := management.New(cfg.Auth0Domain, cfg.Auth0ManagementClientID, cfg.Auth0ManagementSecret)
	return api, err
}
