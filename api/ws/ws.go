package ws

import (
	"fmt"
	"net/http"
	"time"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/jwtparser"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/proto"
)

type connectToken struct {
	expiryUnixSec int64
	userInfo      jwtparser.JWTUserInfo
}

type WSHandler struct {
	connectTokens map[string]connectToken
	jwtReader     jwtparser.RealJWTReader
	melody        *melody.Melody
	svcs          *services.APIServices
}

func MakeWSHandler(jwtValidator jwtparser.RealJWTReader, m *melody.Melody, svcs *services.APIServices) *WSHandler {
	ws := WSHandler{
		connectTokens: map[string]connectToken{},
		jwtReader:     jwtValidator,
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

func (ws *WSHandler) BeginWSConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Expect & read JWT
	usr, err := ws.jwtReader.GetUserInfo(r)

	if err != nil {
		fmt.Printf("Failed to parse JWT, not accepting ws connection")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate a token that is valid for a short time
	token := utils.RandStringBytesMaskImpr(32)

	expirySec := time.Now().Unix() + 10 // TODO: use GetTimeNowSec

	// Clear out old ones, now is a good a time as any!
	ws.clearOldTokens()

	ws.connectTokens[token] = connectToken{expirySec, usr}

	result := &protos.BeginWSConnectionResponse{}
	result.ConnToken = token
	utils.SendProtoBinary(w, result)
}

func (ws *WSHandler) HandleConnect(s *melody.Session) {
	// NOTE: we get passed the initial GET websocket upgrade request here!
	// We require a token as a query param, which we validate against previous
	// calls to /ws-connect. If token isn't valid, we reject
	// To know user info, we can use the token to look it up. We then store
	// it in the session for the life of this session
	// NOTE2: also s.Request is saved, can be accessed later too!
	/*ss, _ := m.Sessions()

	for _, o := range ss {
		value, exists := o.Get("info")

		if !exists {
			continue
		}

		fmt.Printf("Existing user: %v\n", value)

		//info := value.(*GopherInfo)
		//s.Write([]byte("set " + info.ID + " " + info.X + " " + info.Y))
	}*/

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

	// Store the connection info!
	s.Set("user", connectingUser)

	sessId := utils.RandStringBytesMaskImpr(32)
	s.Set("id", sessId)

	fmt.Printf("Connect user: %v, session: %v\n", connectingUser.UserID, sessId)

	// And we're connected, nothing more to do but wait for requests!
}

func (ws *WSHandler) HandleDisconnect(s *melody.Session) {
	var id string
	var connectingUser jwtparser.JWTUserInfo

	if _id, ok := s.Get("id"); !ok {
		fmt.Println("Disconnect MISSING SESSION ID")
		return
	} else {
		id, ok = _id.(string)
		if !ok {
			fmt.Println("Disconnect INVALID SESSION ID TYPE")
			return
		}
	}

	connectingUser, err := getSessionUser(s)
	if err != nil {
		fmt.Printf("Disconnect %v\n", err)
		return
	}

	fmt.Printf("Disconnect user: %v, session: %v\n", connectingUser.UserID, id)
}

func (ws *WSHandler) HandleMessage(s *melody.Session, msg []byte) {
	// We got something, decode it as WSMessage
	wsmsg := protos.WSMessage{}
	err := proto.Unmarshal(msg, &wsmsg)
	if err != nil {
		fmt.Printf("HandleMessage: Error while decoding msg %v\n", err)
		return
	}

	resp, err := ws.dispatchWSMessage(&wsmsg, s)
	if err != nil {
		fmt.Printf("HandleMessage:  %v\n", err)
	} else {
		// Set incoming message ID on the outgoing one
		resp.MsgId = wsmsg.MsgId

		// Send
		sendForSession(s, resp)
	}
}
