package ws

import (
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/core/jwtparser"
)

func getSessionUser(s *melody.Session) (jwtparser.JWTUserInfo, error) {
	var sessionID = ""
	var connectingUser jwtparser.JWTUserInfo

	if _id, ok := s.Get("id"); ok {
		_idStr, ok := _id.(string)
		if ok {
			sessionID = _idStr
		}
	}

	if _connectingUser, ok := s.Get("user"); !ok {
		return connectingUser, fmt.Errorf("User not found on session %v", sessionID)
	} else {
		connectingUser, ok = _connectingUser.(jwtparser.JWTUserInfo)
		if !ok {
			return connectingUser, fmt.Errorf("User details corrupt on session %v", sessionID)
		}
	}

	return connectingUser, nil
}