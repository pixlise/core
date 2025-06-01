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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/auth0.v4/management"
)

// Batch-replacement of roles in Auth0 to update user roles to newly adjusted role scheme
// Allows us to run through all users and replace a set of roles with others

// Defined roles:
// General: rol_Ufb1Yb6biX61zJdz
// Creator: rol_zlh4KM1IMuXrUOGE
// Editor: rol_8C61CvTRe336hIPX
// Sharer: rol_Zhs0ISBNvqgNt1Eq
// Admin: rol_IDt2L7nT031U7GPx
func main() {
	var cmd string
	var cmdFile string
	var roleToAdd string
	var auth0Domain, auth0ManagementClientID, auth0ManagementSecret string

	flag.StringVar(&cmd, "cmd", "", "Command")
	flag.StringVar(&cmdFile, "cmdFile", "", "Command File")
	flag.StringVar(&roleToAdd, "roleToAdd", "", "Role to add")

	flag.StringVar(&auth0Domain, "auth0Domain", "", "Username")
	flag.StringVar(&auth0ManagementClientID, "auth0ManagementClientID", "", "Password")
	flag.StringVar(&auth0ManagementSecret, "auth0ManagementSecret", "", "Client ID")

	flag.Parse()

	api, err := management.New(auth0Domain, auth0ManagementClientID, auth0ManagementSecret)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if cmd == "listRoles" {
		roles, err := api.Role.List()
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Printf("%v\n", roles.Roles)
	} else if cmd == "listUserRoles" {
		var page int
		var count int

		fmt.Println("Idx, User ID, User Name, Role Names, Role IDs")

		for {
			userList, err := api.User.List(
				management.Query(""), //`logins_count:{100 TO *]`),
				management.Page(page),
			)
			if err != nil {
				log.Fatalf("get users page %v: %v", page, err)
				log.Fatalf("%v", err)
			}

			for _, u := range userList.Users {
				id := u.GetID()
				roles, err := api.User.Roles(id)
				if err != nil {
					log.Fatalf("get roles for user %v: %v", id, err)
				}

				roleIDs := []string{}
				roleNames := []string{}

				for _, role := range roles.Roles {
					roleIDs = append(roleIDs, role.GetID())
					roleNames = append(roleNames, role.GetName())
				}

				fmt.Printf("%v, %v, %v, %v, %v\n", count, id, u.GetName(), strings.Join(roleNames, " "), strings.Join(roleIDs, " "))
				count++
			}

			if !userList.HasNext() {
				break
			}
			page++
		}
	} else if cmd == "changeRoles" {
		cmdFileBytes, err := os.ReadFile(cmdFile)
		if err != nil {
			log.Fatalf("Failed to read command file %v: %v", cmdFile, err)
		}

		if len(roleToAdd) <= 0 {
			log.Fatalf("Didn't specify argument for roleToAdd")
		}

		// Should have the following columns
		cmdLines := strings.Split(string(cmdFileBytes), "\n")
		expCSVHeader := "Idx, User ID, User Name, Role Names, Role IDs"
		if cmdLines[0] != expCSVHeader {
			log.Fatalf("Command file for %v expected header: %v", cmd, expCSVHeader)
		}

		// Run through
		userIds := []string{}
		roleIdListToRemove := []string{}

		for lineNo, line := range cmdLines {
			if lineNo == 0 {
				continue
			}

			if len(line) > 0 {
				parts := strings.Split(line, ",")
				if len(parts) != 5 {
					log.Fatalf("Expected 5 columns on line %v, got: %v", lineNo, line)
				}

				userIds = append(userIds, strings.Trim(parts[1], " "))
				roleIdListToRemove = append(roleIdListToRemove, strings.Trim(parts[4], " "))
			}
		}

		// Now we run through and do the calls
		for c, userId := range userIds {
			fmt.Printf("Editing user: %v\n", userId)
			roleIdsToRemove := strings.Split(roleIdListToRemove[c], " ")
			for _, roleIdToRemove := range roleIdsToRemove {
				if len(roleIdToRemove) > 0 {
					fmt.Printf("  - Removing role: %v\n", roleIdToRemove)
					if err := api.User.RemoveRoles(userId, &management.Role{ID: &roleIdToRemove}); err != nil {
						log.Fatalf("Failed to remove role %v from user %v: %v", roleIdToRemove, userId, err)
					}
				}
			}

			fmt.Printf("  - Assigning role: %v\n", roleToAdd)
			if err := api.User.AssignRoles(userId, &management.Role{ID: &roleToAdd}); err != nil {
				log.Fatalf("Failed to add role %v from user %v: %v", roleToAdd, userId, err)
			}
		}
	}
}
