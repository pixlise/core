package wstestlib

import (
	"fmt"
	"math"
	"strings"
	"time"

	protos "github.com/pixlise/core/v3/generated-protos"
)

type NoAccessTestccessCallbackFunc func(string)
type AccessTestccessCallbackFunc func(string, string, *protos.UserGroupList, *protos.UserGroupList, []*protos.UserGroupDB, bool)

// Sets up users/groups for different scenarios and calls the callback function for each iteration. Note: noAccessCallback can be nil if not required
// Group depth defines the max number of groups chained together as part of the tests
func RunFullAccessTest(apiHost string, userId string, groupDepth int, noAccessCallback NoAccessTestccessCallbackFunc, accessCheckCallback AccessTestccessCallbackFunc) {
	viewEditTxt := []string{"viewer", "editor"}
	viewMemberTxt := []string{"viewer", "member"}

	loginCount := 0
	for groupLevel := 0; groupLevel <= groupDepth; groupLevel++ {
		viewEditCount := int(math.Pow(2, float64(groupLevel+1)))
		for viewOrEdit := 0; viewOrEdit < viewEditCount; viewOrEdit++ {
			// Call back for no access test - in case we want to try the "cleared" scenario - without access permissions set up
			if noAccessCallback != nil {
				noAccessCallback(apiHost)
			}

			what := ""

			// Now set up ownership object for scan
			ownership := []*protos.UserGroupList{
				{
					UserIds:  []string{},
					GroupIds: []string{},
				},
				{
					UserIds:  []string{},
					GroupIds: []string{},
				},
			}

			//fmt.Printf("viewOrEdit: %v\n", viewOrEdit)
			ownerViewOrEdit := viewOrEdit & 1
			ownershipListToEdit := ownership[ownerViewOrEdit]
			what += fmt.Sprintf("(%v)", viewEditTxt[ownerViewOrEdit])

			groups := []*protos.UserGroupDB{}
			switch groupLevel {
			case 0:
				what = "owner" + what + ".user"
				// User is assigned directly
				ownershipListToEdit.UserIds = append(ownershipListToEdit.UserIds, userId)
			default:
				trace := []string{}
				lastGroupId := ""
				for g := 0; g < groupLevel; g++ {
					group := &protos.UserGroupDB{
						Id:             fmt.Sprintf("user-group-%v", g),
						Name:           fmt.Sprintf("Group-%v", g),
						CreatedUnixSec: uint32(1234567890 + g),
					}

					groupList := &protos.UserGroupList{
						UserIds:  []string{},
						GroupIds: []string{},
					}

					attached := ""
					if len(lastGroupId) == 0 {
						groupList.UserIds = append(groupList.UserIds, userId)
						attached = "userId"
					} else {
						groupList.GroupIds = append(groupList.GroupIds, lastGroupId)
						attached = "groupId=" + lastGroupId
					}

					// Decide if this one will be a viewer or member
					mask := 0x1 << (g + 1)
					viewOrMember := 0
					if (viewOrEdit & mask) > 0 {
						viewOrMember = 1
					}

					if viewOrMember == 0 {
						group.Viewers = groupList
					} else {
						group.Members = groupList
					}

					trace = append(trace, fmt.Sprintf(" %v(%v).%v", group.Id, viewMemberTxt[viewOrMember], attached))

					groups = append(groups, group)
					lastGroupId = group.Id
				}

				what = "owner" + what + ".group=" + lastGroupId + " " + strings.Join(trace, " ")
				ownershipListToEdit.GroupIds = append(ownershipListToEdit.GroupIds, lastGroupId)
			}

			// Call back for the actual access test
			accessCheckCallback(apiHost, what, ownership[0], ownership[1], groups, ownerViewOrEdit == 1)

			// Occasionally pause to not trip auth0 login frequency
			if loginCount > 9 {
				fmt.Println("Wait to not rate limit Auth0...")
				time.Sleep(time.Duration(13) * time.Second)
			}

			loginCount += 2

			fmt.Printf("Login count: %v...\n", loginCount)
		}
	}
}
