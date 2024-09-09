package wsHandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/auth0login"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/auth0.v4/management"
)

// /////////////////////////////////////////////////////////////////////
// Listing users from Auth0
func HandleUserListReq(req *protos.UserListReq, hctx wsHelpers.HandlerContext) (*protos.UserListResp, error) {
	auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
	if err != nil {
		return nil, err
	}

	var page int

	users := []*protos.Auth0UserDetails{}

	for {
		userList, err := auth0API.User.List(
			management.Query(""), //`logins_count:{100 TO *]`),
			management.Page(page),
		)
		if err != nil {
			return nil, err
		}

		users = append(users, makeUserList(userList, hctx.Svcs.MongoDB)...)

		if !userList.HasNext() {
			break
		}
		page++
	}

	return &protos.UserListResp{Details: users}, err
}

// /////////////////////////////////////////////////////////////////////
// Getting a users roles
func HandleUserRolesListReq(req *protos.UserRolesListReq, hctx wsHelpers.HandlerContext) (*protos.UserRolesListResp, error) {
	if err := wsHelpers.CheckStringField(&req.UserId, "UserId", 5, wsHelpers.Auth0UserIdFieldMaxLength); err != nil {
		return nil, err
	}

	auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
	if err != nil {
		return nil, err
	}

	gotRoles, err := auth0API.User.Roles(req.UserId)
	if err != nil {
		return nil, err
	}

	return &protos.UserRolesListResp{
		Roles: makeRoleList(gotRoles),
	}, nil
}

// /////////////////////////////////////////////////////////////////////
// Managing a users roles
func HandleUserAddRoleReq(req *protos.UserAddRoleReq, hctx wsHelpers.HandlerContext) (*protos.UserAddRoleResp, error) {
	if err := wsHelpers.CheckStringField(&req.UserId, "UserId", 5, wsHelpers.Auth0UserIdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.RoleId, "RoleId", 5, 32); err != nil {
		return nil, err
	}

	auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
	if err != nil {
		return nil, err
	}

	userID := req.UserId
	roleID := req.RoleId

	unassignNeeded := false
	unassignedNewUserRoleID := hctx.Svcs.Config.Auth0NewUserRoleID

	if roleID != unassignedNewUserRoleID {
		// If the user has the role "Unassigned New User" and is being assigned another role, we clear
		// Unassigned New User because an admin user may not know to remove it and it would confuse other things
		roleResp, err := auth0API.User.Roles(userID)
		if err != nil {
			hctx.Svcs.Log.Errorf("Failed to query user roles when new role being assigned: %v", err)
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
		hctx.Svcs.Log.Infof("User %v is being assigned role %v. The existing \"Unassigned New User\" role is being automatically removed", userID, roleID)

		roleToUnassign := unassignedNewUserRoleID
		err = auth0API.User.RemoveRoles(userID, &management.Role{ID: &roleToUnassign})
		if err != nil {
			hctx.Svcs.Log.Errorf("Failed to remove \"Unassigned New User\" role when user role is changing: %v", err)
		}

		// Don't flood Auth0 with requests!
		time.Sleep(1200 * time.Millisecond)
	}

	err = auth0API.User.AssignRoles(userID, &management.Role{ID: &roleID})
	return nil, err
}

func HandleUserDeleteRoleReq(req *protos.UserDeleteRoleReq, hctx wsHelpers.HandlerContext) (*protos.UserDeleteRoleResp, error) {
	if err := wsHelpers.CheckStringField(&req.UserId, "UserId", 5, wsHelpers.Auth0UserIdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.RoleId, "RoleId", 5, 32); err != nil {
		return nil, err
	}

	auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
	if err != nil {
		return nil, err
	}

	err = auth0API.User.RemoveRoles(req.UserId, &management.Role{ID: &req.RoleId})
	return nil, err
}

// /////////////////////////////////////////////////////////////////////
// Getting all user roles
func HandleUserRoleListReq(req *protos.UserRoleListReq, hctx wsHelpers.HandlerContext) (*protos.UserRoleListResp, error) {
	auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
	if err != nil {
		return nil, err
	}

	// Get roles for each
	gotRoles, err := auth0API.Role.List()
	if err != nil {
		return nil, err
	}

	return &protos.UserRoleListResp{
		Roles: makeRoleList(gotRoles),
	}, nil
}

// /////////////////////////////////////////////////////////////////////
// Utility functions

func makeRoleList(from *management.RoleList) []*protos.Auth0UserRole {
	roles := []*protos.Auth0UserRole{}

	for _, r := range from.Roles {
		role := &protos.Auth0UserRole{
			Id:          r.GetID(),
			Name:        r.GetName(),
			Description: r.GetDescription(),
		}
		roles = append(roles, role)
	}

	return roles
}

func makeUserList(from *management.UserList, db *mongo.Database) []*protos.Auth0UserDetails {
	users := []*protos.Auth0UserDetails{}

	for _, u := range from.Users {
		user := makeUser(u, db)
		users = append(users, user)
	}

	return users
}

func makeUser(from *management.User, db *mongo.Database) *protos.Auth0UserDetails {
	userID := from.GetID()
	userName := from.GetName()
	userEmail := from.GetEmail()

	user := protos.Auth0UserDetails{
		Auth0User: &protos.UserInfo{
			Id:      userID,
			Name:    userName,
			Email:   userEmail,
			IconURL: from.GetPicture(),
		},
	}

	// These may not be there...
	if from.CreatedAt != nil {
		user.CreatedUnixSec = uint32(from.GetCreatedAt().Unix())
	}
	if from.LastLogin != nil {
		user.LastLoginUnixSec = uint32(from.GetLastLogin().Unix())
	}

	userDBItem, err := wsHelpers.GetDBUser(userID, db)
	if err != nil {
		fmt.Printf("Failed to get user details for Auth0 user id: %v\n", userID)
	} else if userDBItem != nil {
		user.PixliseUser = userDBItem.Info
	}

	return &user
}

func HandleUserImpersonateReq(req *protos.UserImpersonateReq, hctx wsHelpers.HandlerContext) (*protos.UserImpersonateResp, error) {
	// NOTE: we set this up in the DB, page refresh will cause it to be applied
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserImpersonatorsName)
	ctx := context.TODO()

	realUserId, ok := hctx.Session.Get("realUserId")

	// If we're impersonating a user, make sure the one requested isn't ours
	if ok && realUserId == req.UserId || req.UserId == hctx.SessUser.User.Id {
		return nil, errors.New("User cannot impersonate themself")
	}

	if len(req.UserId) <= 0 {
		// Delete any impersonation entries
		if !ok {
			return nil, errors.New("Failed to get real user id so cannot remove impersonation setting")
		}

		delResult, err := coll.DeleteOne(ctx, bson.M{"_id": realUserId}, options.Delete())
		if err != nil {
			return nil, fmt.Errorf("Failed to delete impersonation setting: %v", err)
		}

		if delResult.DeletedCount <= 0 {
			return nil, errors.New("No impersonation settings were removed")
		}

		return &protos.UserImpersonateResp{}, nil
	}
	// ELSE...

	// Validate the user id
	userToImpersonate, err := wsHelpers.GetDBUser(req.UserId, hctx.Svcs.MongoDB)

	if err != nil {
		return nil, fmt.Errorf("Failed to find user to impersonate by id: %v. Error: %v", req.UserId, err)
	}

	// Add impersonation entry
	// NOTE: we ensure we are not currently impersonating...
	userId := hctx.SessUser.User.Id
	if ok {
		// User is currently impersonating already, so make sure we book their real user name in!
		userId = fmt.Sprintf("%v", realUserId) // or could maybe use realUserId.(string)
	}

	saveItem := wsHelpers.UserImpersonationItem{
		Id:               userId,
		ImpersonatedId:   req.UserId,
		TimeStampUnixSec: uint32(hctx.Svcs.TimeStamper.GetTimeNowSec()),
	}
	filter := bson.M{"_id": userId}
	updResult, err := coll.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: &saveItem}}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("Failed to save impersonation setting: %v", err)
	}

	if updResult.UpsertedCount == 0 && updResult.ModifiedCount == 0 {
		hctx.Svcs.Log.Errorf("Unexpected update result for user impersonation of %v by %v: %+v", req.UserId, userId, updResult)
	}

	/*
		// User attached to this session wants to become the user id specified OR stop impersonating if incoming ID is blank
		if len(req.UserId) <= 0 {
			// Ensure we have a "original" user stored
			if orig, ok := hctx.Session.Get("originalUser"); !ok {
				return nil, fmt.Errorf("Failed to find original user details, existing session may not be impersonating a user")
			} else {
				hctx.Session.Set("user", orig)
				hctx.Session.UnSet("originalUser")
			}
		} else {
			// They're becoming a user - store original if needed
			if _, ok := hctx.Session.Get("originalUser"); !ok {
				if u, okReal := hctx.Session.Get("user"); okReal {
					hctx.Session.Set("originalUser", u)
				} else {
					return nil, fmt.Errorf("Failed to backup real user details")
				}
			}

			// "Become" the new user
			newUser, err := wsHelpers.MakeSessionUser(hctx.SessUser.SessionId, req.UserId, hctx.SessUser.Permissions, hctx.Svcs.MongoDB)
			if err != nil {
				return nil, fmt.Errorf("Failed to impersonate user: %v. Error was: %v", req.UserId, err)
			}

			hctx.Session.Set("user", newUser)
			hctx.SessUser = *newUser
		}*/

	return &protos.UserImpersonateResp{
		SessionUser: &protos.UserInfo{
			Id:    userToImpersonate.Id,
			Name:  userToImpersonate.Info.Name,
			Email: userToImpersonate.Info.Email,
		},
	}, nil
}

func HandleUserImpersonateGetReq(req *protos.UserImpersonateGetReq, hctx wsHelpers.HandlerContext) (*protos.UserImpersonateGetResp, error) {
	// If user is not impersonating someone, just return an empty message
	_, ok := hctx.Session.Get("realUserId")
	if !ok {
		// No impersonation...
		return &protos.UserImpersonateGetResp{}, nil
	}

	// We have the real user id, but what's in our session is the impersonated info, return that
	return &protos.UserImpersonateGetResp{
		SessionUser: hctx.SessUser.User,
	}, nil
}
