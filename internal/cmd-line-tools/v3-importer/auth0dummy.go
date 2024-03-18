package main

func dummyRead() (map[string][]string, map[string]bool, map[string][]string) {
	// This dummys up a call to auth0 so our migration tool can work often and not hit auth0 api rate limits
	// this stuff doesn't change much really... here is what it dummys up:

	// The roles we find... so id to list of groups
	roleToGroup := map[string][]string{
		"role_id": {"access:JPL Breadboard", "access:PIXL-EM", "access:PIXL-FM"},
	}

	// Group membership
	userToGroup := map[string][]string{
		"auth0_user_id": {"access:JPL Breadboard", "access:PIXL-EM", "access:PIXL-FM", "access:Stony Brook Breadboard"},
	}

	// Just a straight list of the groups
	allGroups := map[string]bool{
		"access:JPL Breadboard": true,
	}

	return roleToGroup, allGroups, userToGroup
}
