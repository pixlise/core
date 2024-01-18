package wsHelpers

import (
	"strings"

	protos "github.com/pixlise/core/v4/generated-protos"
)

func HasPermission(userPermissions map[string]bool, toCheck protos.Permission) bool {
	// Get the permission as a string, but snip off the prefix
	permName := strings.TrimPrefix(toCheck.String(), "PERM_")
	return userPermissions[permName]
}
