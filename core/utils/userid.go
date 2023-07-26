package utils

import "strings"

func FixUserId(userId string) string {
	if len(userId) > 0 && !strings.HasPrefix(userId, "auth0|") {
		return "auth0|" + userId
	}
	return userId
}