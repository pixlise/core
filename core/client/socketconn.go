package client

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pixlise/core/v4/core/auth0login"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
	"gopkg.in/square/go-jose.v2/jwt"
)

type ConnectInfo struct {
	Host string
	User string
	Pass string
}

type Auth0Info struct {
	ClientId string `json:"auth0_client"`
	Domain   string `json:"auth0_domain"`
	Audience string `json:"auth0_audience"`
}

type SocketConn struct {
	JWT       string
	UserId    string
	send      chan []byte
	recv      chan []byte
	recvList  [][]byte // msgs received in past
	interrupt chan os.Signal
	done      chan struct{}
	reqCount  uint32
}

const maxResponsesBuffered = 100

// Inspired by: https://tradermade.com/tutorials/golang-websocket-client
func (s *SocketConn) Connect(connectParams ConnectInfo, auth0Params Auth0Info) error {
	token, err := s.getWSConnectToken(connectParams, auth0Params)
	if err != nil {
		return err
	}

	s.send = make(chan []byte)
	s.interrupt = make(chan os.Signal, 1)

	signal.Notify(s.interrupt, os.Interrupt)

	// NOTE: not using wss for local...
	protocol := "ws"
	hostUrl := connectParams.Host

	if strings.HasPrefix(hostUrl, "https://") {
		protocol = "wss"
		hostUrl = strings.TrimPrefix(hostUrl, "https://")
	} else {
		hostUrl = strings.TrimPrefix(hostUrl, "http://")
	}

	hostUrl = strings.TrimSuffix(hostUrl, "/")

	wsUrl := url.URL{Scheme: protocol, Host: hostUrl, Path: "/ws", RawQuery: "token=" + token}
	ws, resp, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		log.Fatalln("WS connection failed:", err)
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Expecting an empty body
	if len(b) > 0 {
		log.Fatalf("Expected empty WS Connection body, got: %v", string(b))
	}

	// When the program closes close the connection
	//defer ws.Close()
	done := make(chan struct{})

	s.recv = make(chan []byte, maxResponsesBuffered)

	// Message receiving thread
	go func() {
		defer close(done)
		for {
			mtype, msgBytes, err := ws.ReadMessage()
			if err != nil {
				log.Fatalf("Error when reading msg from socket: %v\n", err)
			}

			// Check that it's a binary message...
			if mtype != 2 {
				log.Fatalln("Received non-binary message from web socket")
			}

			s.recv <- msgBytes
		}
	}()

	// Message sending thread
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case m := <-s.send:
				err := ws.WriteMessage(websocket.BinaryMessage, []byte(m))
				if err != nil {
					log.Fatalf("Failed to send message: %v\n", err)
				}
			case t := <-ticker.C:
				/*log.Println("Sending ping...")
				err := ws.WriteMessage(websocket.TextMessage, []byte(t.String()))
				if err != nil {
					log.Fatalf("Failed to send ping: %v\n", err)
				}*/
				log.Printf("Skipping Sending ping... %v\n", t)
			case <-s.interrupt:
				log.Println("interrupt")
				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Println("Write close: ", err)
					return
				}

				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}
	}()

	return nil
}

func (s *SocketConn) Disconnect() error {
	s.done <- struct{}{}
	return nil
}

// So we don't spam Auth0 and get rate limited (and thereby fail tests), we store the last known JWT of given connect params
var cachedJWT = map[string]string{}

func ClearJWTCache() {
	cachedJWT = map[string]string{}
}

func GetJWTFromCache(host string, user string, pass string) string {
	cacheKey := host + "-" + user + "-" + pass

	return cachedJWT[cacheKey]
}

func (s *SocketConn) getWSConnectToken(connectParams ConnectInfo, auth0Params Auth0Info) (string, error) {
	// Try find it
	cacheKey := connectParams.Host + "-" + connectParams.User + "-" + connectParams.Pass
	if jwt, ok := cachedJWT[cacheKey]; !ok {
		jwt, err := auth0login.GetJWT(connectParams.User, connectParams.Pass,
			auth0Params.ClientId, auth0Params.Domain, "http://localhost:4200/authenticate", auth0Params.Audience, "openid profile email")
		if err != nil {
			return "", err
		}

		// Cache it!
		cachedJWT[cacheKey] = jwt
		s.JWT = jwt
	} else {
		s.JWT = jwt
	}

	// Parse the JWT to get our user ID
	token, err := jwt.ParseSigned(s.JWT)
	if err != nil {
		return "", err
	}
	var claims map[string]interface{}
	err = token.UnsafeClaimsWithoutVerification(&claims)
	if err != nil {
		return "", err
	}

	s.UserId = fmt.Sprintf("%v", claims["sub"])

	// Get WS connection token
	// NOTE: not using https for local...
	protocol := "http"
	hostUrl := connectParams.Host

	if strings.HasPrefix(hostUrl, "https://") {
		protocol = "https"
		hostUrl = strings.TrimPrefix(hostUrl, "https://")
	} else {
		hostUrl = strings.TrimPrefix(hostUrl, "http://")
	}
	hostUrl = strings.TrimSuffix(hostUrl, "/")

	wsConnectUrl := url.URL{Scheme: protocol, Host: hostUrl, Path: "/ws-connect"}

	client := &http.Client{}
	req, err := http.NewRequest("GET", wsConnectUrl.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.JWT)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	respBody := protos.BeginWSConnectionResponse{}
	err = proto.Unmarshal(b, &respBody)
	if err != nil {
		return "", err
	}

	return respBody.ConnToken, nil
}

func (s *SocketConn) SendMessage(msg *protos.WSMessage) error {
	s.reqCount++

	msg.MsgId = s.reqCount

	bytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	s.send <- bytes
	return nil
}

/*
func (s *SocketConn) waitReceive(expectStatus protos.ResponseStatus, timeoutMs time.Duration) *protos.WSMessage {
	select {
	case r := <-s.recv:
		wsResp := &protos.WSMessage{}
		err := proto.Unmarshal(r, wsResp)
		if err != nil {
			log.Fatalf("Error receiving msg: %v\n", err)
		}
		if wsResp.Status != expectStatus {
			log.Fatalf("Expected response status %v, got msg: %v", expectStatus, wsResp.String())
		}
		return wsResp
	case <-time.After(timeoutMs * time.Millisecond):
		log.Fatalf("Timed out")
		return nil
	}
}
*/

// Parameters define stop conditions, either how many messages or how much time to wait
func (s *SocketConn) WaitForMessages(msgCount int, timeout time.Duration) []*protos.WSMessage {
	msgs := []*protos.WSMessage{}

	running := true
	for i := 0; i < maxResponsesBuffered && running; i++ {
		select {
		case r := <-s.recv:
			wsResp := &protos.WSMessage{}
			err := proto.Unmarshal(r, wsResp)
			if err != nil {
				log.Fatalf("Error receiving msg: %v\n", err)
			}
			msgs = append(msgs, wsResp)

			//log.Printf("Received msgs: %v, latest Id: %v, msg: %+v\n", len(msgs), wsResp.MsgId, wsResp)

			// If we have enough, stop here
			if len(msgs) >= msgCount {
				running = false
			}
		case <-time.After(timeout):
			running = false
		}
	}

	return msgs
}
