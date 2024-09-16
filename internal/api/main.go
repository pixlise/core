package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/dataimport"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/endpoints"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/notificationSender"
	"github.com/pixlise/core/v4/api/permission"
	"github.com/pixlise/core/v4/api/quantification"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/singleinstance"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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

	// Deprecated now: rand.Seed(time.Now().UnixNano())

	// Invent an instance ID
	instanceId := utils.RandStringBytesMaskImpr(16)

	// Turn off date+time prefixing of log msgs, we have timestamps captured in other ways
	log.SetFlags(0)

	cfg := loadConfig()
	svcs := initServices(cfg, instanceId)

	////////////////////////////////////////////////////
	// Set up WebSocket server
	// Looks like the default config for melody is to expect a ping at least every 54seconds
	m := melody.New()

	// Set web socket configs
	if cfg.WSWriteWaitMs > 0 {
		m.Config.WriteWait = time.Duration(cfg.WSWriteWaitMs) * time.Millisecond
	}
	if cfg.WSPongWaitMs > 0 {
		m.Config.PongWait = time.Duration(cfg.WSPongWaitMs) * time.Millisecond
	}
	if cfg.WSPingPeriodMs > 0 {
		m.Config.PingPeriod = time.Duration(cfg.WSPingPeriodMs) * time.Millisecond
	}
	if cfg.WSMaxMessageSize > 0 {
		m.Config.MaxMessageSize = int64(cfg.WSMaxMessageSize)
	}
	if cfg.WSMessageBufferSize > 0 {
		m.Config.MessageBufferSize = int(cfg.WSMessageBufferSize)
	}
	if cfg.MaxFileCacheAgeSec > 0 {
		wsHelpers.MaxFileCacheAgeSec = int64(cfg.MaxFileCacheAgeSec)
	}
	if cfg.MaxFileCacheSizeBytes > 0 {
		wsHelpers.MaxFileCacheSizeBytes = uint64(cfg.MaxFileCacheSizeBytes)
	}

	fmt.Printf("Web socket config: %+v\n", m.Config)
	ws := ws.MakeWSHandler(m, svcs)

	svcs.Notifier = notificationSender.MakeNotificationSender(instanceId, svcs.MongoDB, svcs.IDGen, svcs.TimeStamper, svcs.Log, getPIXLISELinkBase(svcs.Config.EnvironmentName), ws, m)

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

	// Requesting images
	router.AddCacheControlledStreamHandler(
		apiRouter.MakeEndpointPath("/images/"+apiRouter.UrlStreamDownloadIndicator, endpoints.ScanIdentifier, endpoints.FileNameIdentifier),
		apiRouter.MakeMethodPermission("GET", permission.PermPublic),
		endpoints.GetImage,
	)

	router.AddGenericHandler("/images", apiRouter.MakeMethodPermission("PUT", permission.PermPublic), endpoints.PutImage)

	router.AddGenericHandler("/scan", apiRouter.MakeMethodPermission("PUT", permission.PermPublic), endpoints.PutScanData)

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

	// Start listening for job status changes for auto-import-*
	handler := autoImportHandler{
		instanceId,
		svcs,
	}

	go job.ListenForExternalTriggeredJobs(dataimport.JobIDAutoImportPrefix, handler.handleAutoImportJobStatus, svcs.MongoDB, svcs.Log)

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
	log.Println("API startup configuration:")
	log.Println(cfgStr)
	return cfg
}

func initServices(cfg config.APIConfig, apiInstanceId string) *services.APIServices {
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
	mongoClient, mongoDetails, err := mongoDBConnection.Connect(sess, cfg.MongoSecret, iLog)
	if err != nil {
		log.Fatal(err)
	}

	// Get handle to the DB
	dbName := mongoDBConnection.GetDatabaseName("pixlise", cfg.EnvironmentName)

	db := mongoClient.Database(dbName)
	/*
		// If we're in the unit test environment, drop the database so we start from scratch each time
		if cfg.EnvironmentName == "unittest" {
			log.Printf("NOTE: Environment is \"%v\", so dropping database to start fresh\n", cfg.EnvironmentName)
			db.Drop(context.TODO())
		}
	*/

	// Ensure prod doesn't have restore and impersonate enabled, so we don't overwrite it via the UI
	if strings.Contains(strings.ToLower(cfg.EnvironmentName), "prod") {
		cfg.RestoreEnabled = false
		cfg.ImpersonateEnabled = false
	}

	dbCollections.InitCollections(db, iLog, cfg.EnvironmentName)

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

	snsSvc := sns.New(sess)
	sqsSvc := sqs.New(sess)

	// Set up services
	svcs := &services.APIServices{
		Config:       cfg,
		Log:          iLog,
		S3:           s3svc,
		SNS:          awsutil.RealSNS{SNS: snsSvc},
		SQS:          awsutil.RealSQS{SQS: sqsSvc},
		FS:           fs,
		JWTReader:    jwt,
		IDGen:        &idgen.IDGen{},
		TimeStamper:  &timestamper.UnixTimeNowStamper{},
		MongoDB:      db,
		MongoDetails: mongoDetails,
		// Notifier is configured after ws is created
		InstanceId: apiInstanceId,
	}

	return svcs
}

func getPIXLISELinkBase(env string) string {
	prefix := env
	if strings.Contains(env, "prod") {
		prefix = "www"
	}

	return fmt.Sprintf("https://%v.pixlise.org/", prefix)
}

type autoImportHandler struct {
	instanceId string
	svcs       *services.APIServices
}

func (h autoImportHandler) handleAutoImportJobStatus(status *protos.JobStatus) {
	// If we find a job has completed, check that it's an auto-triggered job, and if so, we can notify out to all our connected
	// clients about this jobs creation. We will also want to email all PIXLISE users interested in this, but for that we have
	// to resolve which of our multiple API instances will do that!

	if !strings.HasPrefix(status.JobId, dataimport.JobIDAutoImportPrefix) {
		h.svcs.Log.Errorf("handleAutoImportJobStatus got unexpected non-external id: %v", status.JobId)
		return // Just in case...
	}

	// Make sure it's a scan!
	if status.JobType != protos.JobStatus_JT_IMPORT_SCAN && status.JobType != protos.JobStatus_JT_REIMPORT_SCAN {
		h.svcs.Log.Errorf("handleAutoImportJobStatus got unexpected job type: %+v", status)
		return
	}

	if status.Status == protos.JobStatus_COMPLETE {
		h.svcs.Notifier.SysNotifyScanChanged(status.JobItemId)

		// Read the scan...
		scan, err := scan.ReadScanItem(status.JobItemId, h.svcs.MongoDB)
		if err != nil {
			h.svcs.Log.Errorf("handleAutoImportJobStatus failed to read scan for id: %v, job id: %v", status.JobItemId, status.JobId)
			return
		}

		// If the scan is not yet "complete", don't notify. For example, if we got a downlink without all spectra delivered (partial downlinks)
		if normalSpectraCount, ok := scan.ContentCounts["NormalSpectra"]; !ok {
			h.svcs.Log.Errorf("handleAutoImportJobStatus failed to get NormalSpectra count for scan: %v, job id: %v", status.JobItemId, status.JobId)
			return
		} else {
			if pseudoCount, ok2 := scan.ContentCounts["PseudoIntensities"]; !ok2 {
				h.svcs.Log.Errorf("handleAutoImportJobStatus failed to get PseudoIntensities count for scan: %v, job id: %v", status.JobItemId, status.JobId)
			} else {
				// Only FM and EM datasets will have pseudo intensities, if it's one of those, we check that we have all normal spectra downloaded
				// NOTE: we're expecting normal spectra to be twice that of psuedo-intensities, because we have A and B detectors!
				if (scan.Instrument == protos.ScanInstrument_PIXL_FM || scan.Instrument == protos.ScanInstrument_PIXL_EM) && normalSpectraCount/2 != pseudoCount {
					h.svcs.Log.Errorf("handleAutoImportJobStatus scan %v not complete, pseudo intensity count: %v, normal spectra count: %v, job id: %v. New scan notification not sent", status.JobItemId, pseudoCount, normalSpectraCount, status.JobId)
					return
				}
			}
		}

		// Determine if it's a new scan or an update
		if status.JobType == protos.JobStatus_JT_IMPORT_SCAN {
			// Is this an updated scan?
			if len(scan.PreviousImportTimesUnixSec) == 0 {
				// No previous times, must be new
				h.svcs.Notifier.NotifyNewScan(scan.Title, scan.Id)
			} else {
				// There are previous times, must be an update
				h.svcs.Notifier.NotifyUpdatedScan(scan.Title, scan.Id)
			}

			// If this is the first time the scan was found to be complete (we have all spectra), run auto quants
			// NOTE: We have to ensure this is only done by 1 active API instance, so we don't end up running the quant on each instance!
			h.svcs.Log.Infof("Scan complete detected, checking if auto-quantification needed...")
			singleinstance.HandleOnce(scan.Id+"-quant", h.instanceId, func(sourceId string) {
				// We ask it to only run if it hasn't got auto-quants already
				quantification.RunAutoQuantifications(scan.Id, h.svcs, true)
			}, h.svcs.MongoDB, h.svcs.TimeStamper, h.svcs.Log)
		} else if status.JobType == protos.JobStatus_JT_REIMPORT_SCAN {
			h.svcs.Notifier.NotifyUpdatedScan(scan.Title, scan.Id)
		}

		// Make sure we're not caching up older versions of the bin file locally
		wsHelpers.ClearCacheForScanId(scan.Id, h.svcs.TimeStamper, h.svcs.Log)
	}
}
