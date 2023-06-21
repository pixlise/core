package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/api/endpoints"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/idgen"
	"github.com/pixlise/core/v3/core/jwtparser"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/mongoDBConnection"
	"github.com/pixlise/core/v3/core/timestamper"
	"github.com/pixlise/core/v3/core/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// This was added for a profiler to be able to connect, otherwise uses no reasources really
	go func() {
		http.ListenAndServe(":1234", nil)
	}()
	// This is for prometheus
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	rand.Seed(time.Now().UnixNano())

	cfg := loadConfig()
	svcs := initServices(cfg)

	////////////////////////////////////////////////////
	// Set up WebSocket server
	m := melody.New()
	ws := ws.MakeWSHandler(m, svcs)

	// Create event handlers for websocket
	m.HandleConnect(ws.HandleConnect)
	m.HandleDisconnect(ws.HandleDisconnect)
	//m.HandleMessage(ws.HandleMessage) <-- For now we don't accept text messages in web socket, all protobuf binary!
	m.HandleMessageBinary(ws.HandleMessage)

	////////////////////////////////////////////////////
	// Set up HTTP server

	muxRouter := mux.NewRouter() //.StrictSlash(true)
	// Should we use StrictSlash??

	router := apiRouter.NewAPIRouter(svcs, muxRouter)

	// Root request which shows status HTML page
	router.AddPublicHandler("/", "GET", endpoints.RootRequest)

	// User requesting version as protobuf
	router.AddPublicHandler("/version-binary", "GET", endpoints.GetVersionProtobuf)
	// User requesting version as JSON
	router.AddPublicHandler("/version-json", "GET", endpoints.GetVersionJSON)

	// WS initiation - token retrieval to be allowed to create socket
	router.AddGenericHandler("/ws-connect", apiRouter.MakeMethodPermission("GET", permission.PermPublic), ws.HandleBeginWSConnection)

	// Actual web socket creation, expects the HTTP upgrade header
	router.AddPublicHandler("/ws", "GET", ws.HandleSocketCreation)

	// Setup middleware
	routePermissions := router.GetPermissions()
	printRoutePermissions(routePermissions)

	jwtValidator := svcs.JWTReader.GetValidator()
	authware := endpoints.AuthMiddleWareData{
		RoutePermissionsRequired: routePermissions,
		JWTValidator:             jwtValidator,
		Logger:                   svcs.Log,
	}
	logware := endpoints.LoggerMiddleware{
		APIServices:  svcs,
		JwtValidator: jwtValidator,
	}

	promware := endpoints.PrometheusMiddleware

	router.Router.Use(authware.Middleware, logware.Middleware, promware)

	// Now also log this to the world...
	svcs.Log.Infof("API version \"%v\" started...", services.ApiVersion)

	log.Fatal(
		http.ListenAndServe(":8080",
			handlers.CORS(
				handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
				handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
				handlers.AllowedOrigins([]string{"*"}))(router.Router)))
}

func loadConfig() config.APIConfig {
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Something went wrong with API config. Error: %v\n", err)
	}

	// Show the config
	cfgJSON, err := json.MarshalIndent(cfg, "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		log.Fatalf("Error trying to display config\n")
	}

	// Core count can't be 0!
	if cfg.CoresPerNode <= 0 {
		cfg.CoresPerNode = 6 // Reasonable, our laptops have 6...
	}

	if cfg.MaxQuantNodes <= 0 {
		cfg.MaxQuantNodes = 40
	}

	cfgStr := string(cfgJSON)
	log.Println(cfgStr)
	return cfg
}

func initServices(cfg config.APIConfig) *services.APIServices {
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

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(cfg.LogLevel)

	// Connect to mongo
	mongoClient, err := mongoDBConnection.Connect(sess, cfg.MongoSecret, iLog)
	if err != nil {
		log.Fatal(err)
	}

	// Get handle to the DB
	dbName := mongoDBConnection.GetDatabaseName("pixlise", cfg.EnvironmentName)
	db := mongoClient.Database(dbName)

	// Authenticaton for endpoints
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

	// Set up services
	svcs := &services.APIServices{
		Config:      cfg,
		Log:         iLog,
		FS:          fs,
		JWTReader:   jwt,
		IDGen:       &idgen.IDGen{},
		TimeStamper: &timestamper.UnixTimeNowStamper{},
		MongoDB:     db,
	}

	return svcs
}
