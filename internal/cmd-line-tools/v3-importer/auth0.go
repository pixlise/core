package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/auth0.v4/management"
)

func migrateAuth0UserGroups(auth0Domain string, auth0ClientId string, auth0Secret string, dest *mongo.Database) (map[string]string, error) {
	result := map[string]string{}
	coll := dest.Collection(dbCollections.UserGroupsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return result, err
	}

	/*roleToGroup, allGroups, userToGroup, err := readFromAuth0(auth0Domain, auth0ClientId, auth0Secret)
	if err != nil {
		return err
	}*/
	_, allGroups, userToGroup := dummyRead()

	// Form user groups
	for group := range allGroups {
		groupMembers := []string{}

		// Run through all users, and add the ones that are in this group
		for user, groups := range userToGroup {
			if utils.ItemInSlice(group, groups) {
				groupMembers = append(groupMembers, user)
			}
		}

		group = strings.TrimPrefix(group, "access:")

		dbGroup := &protos.UserGroupDB{
			Id:   makeID(),
			Name: group,
			//CreatedUnixSec: ,
			Viewers: &protos.UserGroupList{
				UserIds:  []string{},
				GroupIds: []string{},
			},
			Members: &protos.UserGroupList{
				UserIds:  groupMembers,
				GroupIds: []string{},
			},
			AdminUserIds: []string{},
		}

		_, err := coll.InsertOne(context.TODO(), dbGroup)
		if err != nil {
			return result, err
		}

		// Remember this group
		result[group] = dbGroup.Id
	}

	fmt.Printf("Created %v user groups\n", len(allGroups))

	return result, nil
}

func readFromAuth0(auth0Domain string, auth0ClientId string, auth0Secret string) (map[string][]string, map[string]bool, map[string][]string, error) {
	roleToGroup := map[string][]string{}
	allGroups := map[string]bool{}
	userToGroup := map[string][]string{}

	api, err := management.New(auth0Domain, auth0ClientId, auth0Secret)

	if err != nil {
		return roleToGroup, allGroups, userToGroup, err
	}

	// Get all roles
	roles, err := api.Role.List()
	if err != nil {
		return roleToGroup, allGroups, userToGroup, err
	}

	// Generate a list of user IDs to groups they belong to
	for _, role := range roles.Roles {
		roleToGroup[*role.ID] = []string{}

		perm, err := api.Role.Permissions(*role.ID)
		if err != nil {
			return roleToGroup, allGroups, userToGroup, err
		}

		for _, p := range perm.Permissions {
			permName := p.GetName()
			if strings.Contains(permName, "access:") {
				roleToGroup[*role.ID] = append(roleToGroup[*role.ID], permName)
				allGroups[permName] = true
			}
		}
	}

	// Find what roles each user has
	var userPage int
	for {
		userList, err := api.User.List(
			management.Query(""),
			management.Page(userPage),
		)
		if err != nil {
			return roleToGroup, allGroups, userToGroup, err
		}

		for _, u := range userList.Users {
			userId := u.GetID()
			gotRoles, err := api.User.Roles(userId)
			if err != nil {
				return roleToGroup, allGroups, userToGroup, err
			}

			for _, role := range gotRoles.Roles {
				roleId := role.GetID()
				if group, ok := roleToGroup[roleId]; !ok {
					return roleToGroup, allGroups, userToGroup, fmt.Errorf("User: %v failed to get role: %v", userId, roleId)
				} else {
					userToGroup[userId] = append(userToGroup[userId], group...)
				}
			}
		}

		if !userList.HasNext() {
			break
		}
		userPage++
	}

	return roleToGroup, allGroups, userToGroup, nil
}
