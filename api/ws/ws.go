package ws

import (
	"context"
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dbCollections"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

type connectToken struct {
	expiryUnixSec int64
	userInfo      jwtparser.JWTUserInfo
}

type WSHandler struct {
	melody *melody.Melody
	svcs   *services.APIServices
}

func MakeWSHandler(m *melody.Melody, svcs *services.APIServices) *WSHandler {
	ws := WSHandler{
		melody: m,
		svcs:   svcs,
	}
	return &ws
}

func (ws *WSHandler) HandleBeginWSConnection(params apiRouter.ApiHandlerGenericParams) error {
	token := wsHelpers.CreateConnectToken(params.Svcs, params.UserInfo)

	result := &protos.BeginWSConnectionResponse{}
	result.ConnToken = token
	utils.SendProtoBinary(params.Writer, result)

	fmt.Printf("Generated WS token %v for user %v (%v)\n", token, params.UserInfo.UserID, params.UserInfo.Name)
	return nil
}

func (ws *WSHandler) HandleSocketCreation(params apiRouter.ApiHandlerGenericPublicParams) error {
	if err := ws.melody.HandleRequest(params.Writer, params.Request); err != nil {
		// Added to help debug load balancer behaviour
		fmt.Printf("HandleSocketCreation BadRequest error=\"%v\" from host: %v, method: %v, url: %v, agent: %v\n", err, params.Request.Host, params.Request.Method, params.Request.URL, params.Request.UserAgent())
		return errorwithstatus.MakeBadRequestError(err)
	}

	return nil
}

func (ws *WSHandler) HandleConnect(s *melody.Session) {
	// NOTE: we get passed the initial GET websocket upgrade request here!
	// We require a token as a query param, which we validate against previous
	// calls to /ws-connect. If token isn't valid, we reject
	// To know user info, we can use the token to look it up. We then store
	// it in the session for the life of this session
	var connectingUser jwtparser.JWTUserInfo

	queryParams := s.Request.URL.Query()
	if token, ok := queryParams["token"]; !ok {
		fmt.Println("WS connect failed due to missing token")
		s.CloseWithMsg([]byte("--Missing token"))
		return
	} else {
		// Validate the token
		if len(token) != 1 {
			fmt.Printf("WS connect failed for token: %v\n", token)
			s.CloseWithMsg([]byte("--Multiple tokens provided"))
			return
		}

		var err error
		connectingUser, err = wsHelpers.CheckConnectToken(token[0], ws.svcs)

		if err != nil {
			s.CloseWithMsg([]byte("--" + err.Error()))
			return
		}
	}

	// Look up user info
	sessId := utils.RandStringBytesMaskImpr(32)

	sessionUserId := connectingUser.UserID

	// Check if we're set to impersonate another user
	if ws.svcs.Config.ImpersonateEnabled {
		coll := ws.svcs.MongoDB.Collection(dbCollections.UserImpersonatorsName)
		ctx := context.TODO()
		impersonateResult := coll.FindOne(ctx, bson.M{"_id": connectingUser.UserID}, options.FindOne())
		if impersonateResult.Err() != nil {
			if impersonateResult.Err() != mongo.ErrNoDocuments {
				msg := fmt.Sprintf("Error checking for user impersonation setting: %v", impersonateResult.Err())
				fmt.Printf(msg)
				s.CloseWithMsg([]byte("--" + msg))
				return
			}
		} else {
			// We got impersonation info, find the user id we want to pretend to be
			item := wsHelpers.UserImpersonationItem{}
			err := impersonateResult.Decode(&item)

			if err != nil {
				msg := fmt.Sprintf("Failed to read user impersonation setting: %v", err)
				fmt.Printf(msg)
				s.CloseWithMsg([]byte("--" + msg))
				return
			}

			sessionUserId = item.ImpersonatedId

			// Set the "real" user id so we can know who is impersonating what
			s.Set("realUserId", connectingUser.UserID)
		}
	}

	sessionUser, err := wsHelpers.MakeSessionUser(sessId, sessionUserId, connectingUser.Permissions, ws.svcs.MongoDB)
	if err != nil {
		// If we have no record of this user, add it
		if err == mongo.ErrNoDocuments {
			sessionUser, err = wsHelpers.CreateDBUser(sessId, connectingUser, ws.svcs.MongoDB, ws.svcs.Config.DefaultUserGroupId, ws.svcs.Log)
			if err != nil {
				fmt.Printf("WS connect failed for user: %v (%v) - failed to read/create user in DB\n", connectingUser.UserID, connectingUser.Name)
				s.CloseWithMsg([]byte("--Failed to validate session user"))
				return
			}
		}
	}

	// Store the connection info!
	s.Set("user", *sessionUser)

	if sessionUserId == connectingUser.UserID {
		fmt.Printf("Connect user: %v (%v), session: %v\n", connectingUser.UserID, connectingUser.Name, sessId)
	} else {
		fmt.Printf("Connect user: %v (%v), session: %v, impersonating user: %v (%v)\n", connectingUser.UserID, connectingUser.Name, sessId, sessionUserId, sessionUser.User.Name)
	}

	// And we're connected, nothing more to do but wait for requests!
}

func (ws *WSHandler) HandleDisconnect(s *melody.Session) {
	connectingUser, err := wsHelpers.GetSessionUser(s)
	if err != nil {
		fmt.Printf("Disconnect failed to get session info: %v\n", err)
		return
	}

	fmt.Printf("Disconnect user: %v, session: %v\n", connectingUser.User.Id, connectingUser.SessionId)
}

func (ws *WSHandler) HandleMessage(s *melody.Session, msg []byte) {
	// We got something, decode it as WSMessage
	wsmsg := protos.WSMessage{}
	err := proto.Unmarshal(msg, &wsmsg)
	if err != nil {
		fmt.Printf("HandleMessage: Error while decoding msg %v\n", err)
		return
	}

	user, err := wsHelpers.GetSessionUser(s)
	if err != nil {
		fmt.Printf("HandleMessage: Error while retrieving session user: %v\n", err)
		return
	}

	ctx := wsHelpers.HandlerContext{
		Session:  s,
		SessUser: user,
		Melody:   ws.melody,
		Svcs:     ws.svcs,
	}

	resp, err := ws.dispatchWSMessage(&wsmsg, ctx)
	if err != nil {
		fmt.Printf("HandleMessage: %v\n", err)
	}

	if resp != nil {
		// Set incoming message ID on the outgoing one
		resp.MsgId = wsmsg.MsgId

		// Print out errors, except common ones (cache misses)
		if len(resp.ErrorText) > 0 && resp.GetMemoiseGetResp() == nil {
			fmt.Printf("Sending Response Error: %v\n", resp.String())
		}

		// Send
		wsHelpers.SendForSession(s, resp)
	} else {
		fmt.Printf("WARNING: No response generated for request: %+v\n", resp)
	}
}

// For a list of user ids, this returns all the sessions we have for them, and a list of user ids we didn't find sessions for
func (ws *WSHandler) GetSessionForUsersIfExists(userIds []string) ([]*melody.Session, []string) {
	result := []*melody.Session{}
	sessions, err := ws.melody.Sessions()
	if err != nil {
		return []*melody.Session{}, []string{}
	}

	// Remember who we found...
	usersWithSessions := map[string]bool{}
	for _, userId := range userIds {
		usersWithSessions[userId] = false
	}

	for _, session := range sessions {
		sessUser, err := wsHelpers.GetSessionUser(session)
		if err == nil {
			if utils.ItemInSlice(sessUser.User.Id, userIds) {
				// We found one, add it to our output list!
				result = append(result, session)
				usersWithSessions[sessUser.User.Id] = true
			}
		}
	}

	// Build the list of users we didn't find sessions for
	noSessionFoundUserIds := []string{}
	for userId, hasSession := range usersWithSessions {
		if !hasSession {
			noSessionFoundUserIds = append(noSessionFoundUserIds, userId)
		}
	}

	return result, noSessionFoundUserIds
}
