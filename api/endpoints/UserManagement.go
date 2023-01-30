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
	"errors"
	"io/ioutil"
	"time"

	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/pixlUser"
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

	// This data isn't from Auth0, but we're joining it in
	UserDetails *pixlUser.UserStruct `json:"user_details,omitempty"`
}

type roleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const userIDIdentifier = "user_id"
const fieldIDIdentifier = "field_id"
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

	// Removed because these were not used in client and didn't have unit tests!
	//router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/data-collection"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserSettings), userGetDataCollection)
	//router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/data-collection"), apiRouter.MakeMethodPermission("POST", permission.PermWriteUserSettings), userPostDataCollection)

	// Simply retrieves roles
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/all-roles"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserRoles), roleList)

	// Setting fields in user config (name, email for now... could use this to set data-collection too).
	// This is required because auth0 only asks for user email, eventually we notice we don't have their name and prompt for it
	// and this is the endpoint that's supposed to fix it! They may also change their emails over time.
	// NOTE: permission is read, but this is because users who edit their own accounts are different from users who have write roles permissions (admins)!
	// TODO: Maybe this needs to be broken out under its own permission
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/field", fieldIDIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteUserSettings), userPutField)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/config"), apiRouter.MakeMethodPermission("POST", permission.PermWriteUserSettings), userPostConfig)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/config"), apiRouter.MakeMethodPermission("GET", permission.PermReadUserSettings), userGetConfig)

	// Admins can edit user names and emails in bulk by uploading a CSV
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/bulk-user-details"), apiRouter.MakeMethodPermission("POST", permission.PermWriteUserRoles), userEditInBulk)
}

func roleList(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	// Get roles for each
	gotRoles, err := auth0API.Role.List()
	if err != nil {
		return nil, err
	}

	roles := makeRoleList(gotRoles)
	return roles, nil
}

func userListByRole(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	id := params.PathParams[idIdentifier]
	result := []auth0UserInfo{}

	// Slow for now, but works... don't have a lot of users so probably all on 1 page!
	// TODO: if we have speed issues, paginate our own API
	var page int
	for {
		users, err := auth0API.Role.Users(id, management.Page(page))
		if err != nil {
			return nil, err
		}

		result = append(result, makeUserList(users, &params.Svcs.Users)...)

		if !users.HasNext() {
			break
		}
		page++
	}

	return result, err
}

/*
Removed because these were not used in client and didn't have unit tests!

	func userGetDataCollection(params handlers.ApiHandlerParams) (interface{}, error) {
		user, err := params.Svcs.Users.GetUserEnsureExists(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		if err != nil {
			return nil, err
		}

		result := dataCollection{
			Collect: user.Config.DataCollection,
		}

		return result, nil
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

		user, err := params.Svcs.Users.GetUserEnsureExists(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
		if err != nil {
			return nil, err
		}

		// Overwrite data collection flag
		user.Config.DataCollection = req.Collect

		// Save user
		err = params.Svcs.Users.WriteUser(user)
		if err != nil {
			return nil, err
		}

		// Also remember in our run-time cache wether this user is allowing tracking or not
		params.Svcs.Notifications.SetTrack(params.UserInfo.UserID, req.Collect == "true")
		return nil, nil
	}
*/

func userGet(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	id := params.PathParams[idIdentifier]
	user, err := auth0API.User.Read(id)
	if err != nil {
		return nil, err
	}

	return makeUser(user, &params.Svcs.Users), nil
}

func userGetRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	userID := params.PathParams[userIDIdentifier]
	gotRoles, err := auth0API.User.Roles(userID)
	if err != nil {
		return nil, err
	}

	roles := makeRoleList(gotRoles)
	return roles, nil
}

func userPostRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	roleID := params.PathParams[idIdentifier]
	userID := params.PathParams[userIDIdentifier]

	unassignNeeded := false

	if roleID != unassignedNewUserRoleID {
		// If the user has the role "Unassigned New User" and is being assigned another role, we clear
		// Unassigned New User because an admin user may not know to remove it and it would confuse other things
		roleResp, err := auth0API.User.Roles(userID)
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
		err = auth0API.User.RemoveRoles(userID, &management.Role{ID: &roleToUnassign})
		if err != nil {
			params.Svcs.Log.Errorf("Failed to remove \"Unassigned New User\" role when user role is changing: %v", err)
		}

		// Don't flood Auth0 with requests!
		time.Sleep(1200 * time.Millisecond)
	}

	err = auth0API.User.AssignRoles(userID, &management.Role{ID: &roleID})
	return nil, err
}

func userDeleteRoles(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	roleID := params.PathParams[idIdentifier]
	userID := params.PathParams[userIDIdentifier]
	err = auth0API.User.RemoveRoles(userID, &management.Role{ID: &roleID})
	return nil, err
}

func userListQuery(params handlers.ApiHandlerParams) (interface{}, error) {
	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	var page int
	result := []auth0UserInfo{}

	for {
		userList, err := auth0API.User.List(
			management.Query(""), //`logins_count:{100 TO *]`),
			management.Page(page),
		)
		if err != nil {
			return nil, err
		}

		result = append(result, makeUserList(userList, &params.Svcs.Users)...)

		if !userList.HasNext() {
			break
		}
		page++
	}

	return result, err
}

func userGetConfig(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Users.GetUserEnsureExists(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}
	return user.Config, nil
}

func userPostConfig(params handlers.ApiHandlerParams) (interface{}, error) {
	user, err := params.Svcs.Users.GetUserEnsureExists(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}
	var req pixlUser.UserDetails
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	user.Config = req
	err = params.Svcs.Users.WriteUser(user)

	return req, err
}

func userPutField(params handlers.ApiHandlerParams) (interface{}, error) {
	fieldName := params.PathParams[fieldIDIdentifier]

	if fieldName != "name" && fieldName != "email" {
		return nil, errors.New("Unrecognised field: " + fieldName)
	}

	// Ensure user is stored already for this
	user, err := params.Svcs.Users.GetUserEnsureExists(params.UserInfo.UserID, params.UserInfo.Name, params.UserInfo.Email)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	value := ""
	err = json.Unmarshal(body, &value)
	if err != nil {
		return nil, err
	}

	if fieldName == "name" {
		user.Config.Name = value
	} else {
		user.Config.Email = value
	}
	err = params.Svcs.Users.WriteUser(user)
	if err != nil {
		return nil, err
	}

	auth0API, err := InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return nil, err
	}

	auth0User := management.User{}
	if fieldName == "name" {
		auth0User.Name = &value
	} else {
		auth0User.Email = &value
	}

	err = auth0API.User.Update("auth0|"+params.UserInfo.UserID, &auth0User)
	return nil, err
}

type UserEditRequest struct {
	UserID string
	Name   string
	Email  string
}

func userEditInBulk(params handlers.ApiHandlerParams) (interface{}, error) {
	// Here the body is expected to be a JSON of user id, and optional name & email that need to be set
	// This only sets it in Mongo! This does NOT edit Auth0
	// Only exists because we had correct names in auth0 but our mongo was not in sync with it
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	editItems := []UserEditRequest{}
	err = json.Unmarshal(body, &editItems)
	if err != nil {
		return nil, err
	}

	params.Svcs.Log.Infof("Editing users in bulk...")

	// Run through each item & edit in mongo
	for c, item := range editItems {
		params.Svcs.Log.Infof(" %v: %v (name: %v, email: %v)", c+1, item.UserID, item.Name, item.Email)

		user, err := params.Svcs.Users.GetUser(item.UserID)
		if err != nil {
			params.Svcs.Log.Errorf(" User does not exist: %v", item.UserID)
			continue
			//return nil, err
		}

		// Set the fields
		if len(item.Name) > 0 {
			user.Config.Name = item.Name
		}
		if len(item.Email) > 0 {
			user.Config.Email = item.Email
		}

		err = params.Svcs.Users.WriteUser(user)
		if err != nil {
			params.Svcs.Log.Errorf(" Failed to write user: %v", item.UserID)
			//return nil, err
		}
	}

	params.Svcs.Log.Infof("User editing complete")
	return nil, nil
}

func makeUserList(from *management.UserList, pixlUsers *pixlUser.UserDetailsLookup) []auth0UserInfo {
	users := []auth0UserInfo{}

	for _, u := range from.Users {
		user := makeUser(u, pixlUsers)
		users = append(users, user)
	}

	return users
}

func makeUser(from *management.User, pixlUsers *pixlUser.UserDetailsLookup) auth0UserInfo {
	userID := from.GetID()
	userName := from.GetName()
	userEmail := from.GetEmail()
	pixlUserInfo, err := pixlUsers.GetUserEnsureExists(userID, userName, userEmail)
	if err != nil {
		panic(err)
	}

	user := auth0UserInfo{
		UserID:  userID,
		Name:    userName,
		Email:   userEmail,
		Picture: from.GetPicture(),
		//Roles - not returned here
		UserDetails: &pixlUserInfo,
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
