package ws

import (
	"fmt"
	"time"

	"github.com/olahol/melody"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/jwtparser"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/proto"
)

type connectToken struct {
	expiryUnixSec int64
	userInfo      jwtparser.JWTUserInfo
}

type WSHandler struct {
	connectTokens map[string]connectToken
	melody        *melody.Melody
	svcs          *services.APIServices
}

func MakeWSHandler(m *melody.Melody, svcs *services.APIServices) *WSHandler {
	ws := WSHandler{
		connectTokens: map[string]connectToken{},
		melody:        m,
		svcs:          svcs,
	}
	return &ws
}

func (ws *WSHandler) clearOldTokens() {
	nowSec := time.Now().Unix() // TODO: use GetTimeNowSec
	for token, usr := range ws.connectTokens {
		if usr.expiryUnixSec < nowSec {
			delete(ws.connectTokens, token)
		}
	}
}

func (ws *WSHandler) HandleBeginWSConnection(params apiRouter.ApiHandlerGenericParams) error {
	// Generate a token that is valid for a short time
	token := utils.RandStringBytesMaskImpr(32)

	expirySec := time.Now().Unix() + 10 // TODO: use GetTimeNowSec

	// Clear out old ones, now is a good a time as any!
	ws.clearOldTokens()

	ws.connectTokens[token] = connectToken{expirySec, params.UserInfo}

	result := &protos.BeginWSConnectionResponse{}
	result.ConnToken = token
	utils.SendProtoBinary(params.Writer, result)
	return nil
}

func (ws *WSHandler) HandleSocketCreation(params apiRouter.ApiHandlerGenericPublicParams) error {
	ws.melody.HandleRequest(params.Writer, params.Request)
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
		s.CloseWithMsg([]byte("Missing token"))
		return
	} else {
		// Validate the token
		if len(token) != 1 {
			s.CloseWithMsg([]byte("Multiple tokens provided"))
			return
		}

		if conn, ok := ws.connectTokens[token[0]]; !ok {
			s.CloseWithMsg([]byte("Invalid token"))
			return
		} else {
			// Check that it hasn't expired
			nowSec := time.Now().Unix() // TODO: use GetTimeNowSec
			if conn.expiryUnixSec < nowSec {
				s.CloseWithMsg([]byte("Expired token"))
				return
			} else {
				connectingUser = conn.userInfo

				// We no longer need this item in our connection token map
				delete(ws.connectTokens, token[0])
			}
		}
	}

	// Look up user info
	sessId := utils.RandStringBytesMaskImpr(32)

	sessionUser, err := wsHelpers.MakeSessionUser(sessId, connectingUser, ws.svcs.MongoDB)
	if err != nil {
		// If we have no record of this user, add it
		if err == mongo.ErrNoDocuments {
			sessionUser, err = wsHelpers.CreateDBUser(sessId, connectingUser, ws.svcs.MongoDB)
			if err != nil {
				s.CloseWithMsg([]byte("Failed to validate session user"))
				return
			}
		}
	}

	// Store the connection info!
	s.Set("user", *sessionUser)

	fmt.Printf("Connect user: %v, session: %v\n", connectingUser.UserID, sessId)

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

		// Send
		wsHelpers.SendForSession(s, resp)
	} else {
		fmt.Printf("WARNING: No response generated for request: %+v\n", resp)
	}
}