package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/api/endpoints"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/ws"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/jwtparser"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	m := melody.New()

	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Something went wrong with API config. Error: %v\n", err)
	}

	// Get a session for the bucket region
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS session. Error: %v", err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

	// Public API version page
	http.HandleFunc("/", endpoints.GetAboutPage)

	// Public endpoints
	http.HandleFunc("/version-binary", endpoints.GetVersion)
	http.HandleFunc("/version-json", endpoints.GetVersionJSON)

	// Authenticated endpoints
	jwtValidator, err := jwtparser.InitJWTValidator(
		cfg.Auth0Domain,
		cfg.ConfigBucket,
		filepaths.GetConfigFilePath(filepaths.Auth0PemFileName),
		fs,
	)

	if err != nil {
		log.Fatalf("Failed to init JWT validator. Error: %v", err)
	}

	jwt := jwtparser.RealJWTReader{Validator: jwtValidator}

	// Websocket handling
	ws := ws.MakeWSHandler(jwt, m)
	http.HandleFunc("/ws-connect", ws.BeginWSConnection)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleConnect(ws.HandleConnect)
	m.HandleDisconnect(ws.HandleDisconnect)
	m.HandleMessage(ws.HandleMessage)
	m.HandleMessageBinary(ws.HandleMessage)

	http.ListenAndServe(":8080", nil)
}
