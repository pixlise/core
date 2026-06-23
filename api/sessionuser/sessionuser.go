package sessionuser

import protos "github.com/pixlise/core/v4/generated-protos"

type SessionUser struct {
	SessionId              string
	User                   *protos.UserInfo
	Permissions            map[string]bool
	MemberOfGroupIds       []string
	ViewerOfGroupIds       []string
	NotificationSubscribed bool
}
