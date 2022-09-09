package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/profile"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
	apiNotifications "github.com/pixlise/core/core/notifications"
	"github.com/pixlise/core/core/utils"
	"github.com/pixlise/core/data-converter/importer"
	"github.com/pixlise/core/data-converter/importer/msatestdata"
	"github.com/pixlise/core/data-converter/importer/pixlfm"
	"github.com/pixlise/core/data-converter/output"
)

func createDatasourceEvent(inpath string) DatasourceEvent {
	return DatasourceEvent{
		Inpath:         inpath,
		Rangespath:     "DatasetConfig/StandardPseudoIntensities.csv",
		Outpath:        tmpprefix,
		DatasetID:      "",
		DetectorConfig: "PIXL",
	}
}

var o interface{ Stop() }

// JobInit - Create name, Filesystem Access, Notification Stack
func jobinit(inpath string, log logger.ILogger) (DatasourceEvent, fileaccess.S3Access, apiNotifications.NotificationManager, error) {
	o = profile.Start(profile.MemProfile, profile.ProfilePath("/tmp/profile"))
	name := createDatasourceEvent(inpath)
	sess, err := awsutil.GetSession()
	svc, err := awsutil.GetS3(sess)
	if err != nil {
		return DatasourceEvent{}, fileaccess.S3Access{}, nil, err
	}
	fs := fileaccess.MakeS3Access(svc)
	ns := makeNotificationStack(fs, log)
	return name, fs, ns, err
}

// processS3 - If the message received is an S3 trigger, then process the S3 trigger
func processS3(makeLog bool, record awsutil.Record) (string, error) {
	jobLog := logger.StdOutLogger{}

	jobLog.Infof("=========================================")
	jobLog.Infof("=  PIXLISE dataset importer process S3  =")
	jobLog.Infof("=========================================")

	jobLog.Infof("HandleRequest for: \"%v\"\n", record)
	jobLog.Infof("Key: \"%v\"\n", record.S3.Object.Key)

	targetbucket := os.Getenv("DATASETS_BUCKET")

	if strings.Contains(record.S3.Object.Key, "dataset-addons") {
		jobLog.Infof("Re-processing dataset due to file: \"%v\"\n", record.S3.Object.Key)
		splits := strings.Split(record.S3.Object.Key, "/")
		name, fs, ns, err := jobinit(record.S3.Object.Key, jobLog)
		if err != nil {
			return "", err
		}
		s, err := executeReprocess(splits[1], name, time.Now().Unix(), fs, ns, targetbucket, jobLog)
		stopProfiler(fs)
		return s, err
	} else {
		jobLog.Infof("Datasource Path: " + record.S3.Object.Key)
		name, fs, ns, err := jobinit(record.S3.Object.Key, jobLog)
		sourcebucket := record.S3.Bucket.Name
		str, err := executePipeline(name, fs, ns, time.Now().Unix(), sourcebucket, targetbucket, jobLog)
		if err != nil {
			jobLog.Infof("Error in processing data: %v\n", err.Error())
			_, err = triggerErrorNotifications(ns)
			jobLog.Errorf("Could not trigger error notification: %v", err)
		}
		err = fs.DeleteObject(record.S3.Bucket.Name, record.S3.Object.Key)
		if err != nil {
			_, err = triggerErrorNotifications(ns)
			jobLog.Errorf("Could not trigger error notification: %v", err)
		}
		stopProfiler(fs)
		return str, err
	}
}

// processSNS - If the message received is an SNS message, then process the SNS message
func processSns(makeLog bool, record awsutil.Record) (string, error) {
	message := record.SNS.Message
	jobLog := logger.StdOutLogger{}

	jobLog.Infof("==========================================")
	jobLog.Infof("=  PIXLISE dataset importer process SNS  =")
	jobLog.Infof("==========================================")

	targetbucket := os.Getenv("DATASETS_BUCKET")

	jobLog.Infof("Message: %v", message)

	if strings.HasPrefix(message, `{"datasetaddons":`) {
		var snsMsg APISnsMessage
		err := json.Unmarshal([]byte(message), &snsMsg)
		if err != nil {
			fmt.Printf("error unmarshalling message: %v", err)
		}
		fmt.Printf("Re-processing dataset due to file: \"%v\"\n", message)
		fmt.Printf("Key: \"%v\"\n", snsMsg.Key.Dir)
		name, fs, ns, err := jobinit(snsMsg.Key.Dir, jobLog)

		jobLog.Infof("Key: \"%v\"\n", snsMsg.Key.Dir)
		jobLog.Infof("Re-processing dataset due to file: \"%v\"\n", message)

		splits := strings.Split(snsMsg.Key.Dir, "/")

		s, err := executeReprocess(splits[1], name, time.Now().Unix(), fs, ns, targetbucket, jobLog)
		stopProfiler(fs)
		return s, err
	}
	var e awsutil.Event
	err := e.UnmarshalJSON([]byte(message))
	if err != nil {
		jobLog.Errorf("Issue decoding message: %v", err)
	}
	if e.Records[0].EventSource == "aws:s3" {
		name, fs, ns, err := jobinit(e.Records[0].S3.Object.Key, jobLog)
		if err != nil {
			return "", err
		}
		keys, err := checkExisting(getDatasourceBucket(), e.Records[0].S3.Object.Key, fs, jobLog)
		if err != nil {
			return "", err
		}
		if len(keys) > 0 {
			jobLog.Infof("File already exists, not reprocessing")
			return "", nil
		}
		sourcebucket := e.Records[0].S3.Bucket.Name
		jobLog.Infof("Sourcebucket: " + sourcebucket)
		s, err := executePipeline(name, fs, ns, time.Now().Unix(), sourcebucket, targetbucket, jobLog)
		stopProfiler(fs)
		return s, err
	} else if strings.HasPrefix(message, "datasource:") {
		// run execution
	} else {
		jobLog.Infof("Re-processing dataset due to SNS request: \"%v\"\n", record.SNS.Message)
		name, fs, ns, err := jobinit("", jobLog)
		if err != nil {
			fmt.Printf("error initialising job: %v", err)
		}
		s, err := executeReprocess(record.SNS.Message, name, time.Now().Unix(), fs, ns, targetbucket, jobLog)
		stopProfiler(fs)
		return s, err
	}
	return "", nil
}

func stopProfiler(fs fileaccess.S3Access) {
	o.Stop()
	dir, err := utils.ZipDirectory("/tmp/profile")
	if err != nil {
		fmt.Println("Failed to zip profile")
	}
	fs.WriteObject("devpixlise-manualuploadf14e9f17-x59rn61oxeh0", "/profile.zip", dir)
}

// executeReprocess - If the request is to reprocess an existing data source, then execute the reprocess pipeline and download the existing files
func executeReprocess(rtt string, name DatasourceEvent, creationUnixTimeSec int64, fs fileaccess.FileAccess, ns apiNotifications.NotificationManager, targetbucket string, jobLog logger.ILogger) (string, error) {
	_ = fetchRanges(getConfigBucket(), name.Rangespath, fs)
	var allthefiles []string
	updateExisting := false
	importers := map[string]importer.Importer{"test-msa": msatestdata.MSATestData{}, "pixl-fm": pixlfm.PIXLFM{}}
	// TODO: make this work --> importerNames := utils.GetStringMapKeys(importers)
	var importerNames []string
	for k := range importers {
		importerNames = append(importerNames, k)
	}
	files, err := checkExistingArchive(allthefiles, rtt, &updateExisting, fs, jobLog)
	allthefiles = append(files)
	for _, p := range allthefiles {
		if strings.HasSuffix(p, ".zip") || strings.Contains(p, "zip") {
			jobLog.Infof("Preparing to unzip %v\n", p)
			_, err := utils.UnzipDirectory(p, localUnzipPath)
			if err != nil {
				return "", err
			}
			inpath := localUnzipPath
			name.Inpath = inpath
		}
	}
	err = downloadExtraFiles(rtt, fs)
	if err != nil {
		return "", err
	}
	r, err := processFiles(localUnzipPath, name, importers, creationUnixTimeSec, true, fs, ns, targetbucket, jobLog)
	return r, err
}

// executePipeline - Run the full pipeline
func executePipeline(name DatasourceEvent, fs fileaccess.FileAccess, ns apiNotifications.NotificationManager, creationUnixTimeSec int64, sourcebucket string, targetbucket string, jobLog logger.ILogger) (string, error) {
	if jobLog == nil {
		jobLog = logger.NullLogger{}
	}
	err := os.MkdirAll(localUnzipPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	localFS := fileaccess.FSAccess{}
	err = localFS.EmptyObjects(localUnzipPath)
	if err != nil {
		return "", err
	}
	importers := map[string]importer.Importer{"test-msa": msatestdata.MSATestData{}, "pixl-fm": pixlfm.PIXLFM{}}
	// TODO: make this work --> importerNames := utils.GetStringMapKeys(importers)
	var importerNames []string
	for k := range importers {
		importerNames = append(importerNames, k)
	}

	jobLog.Infof("----- Importing pseudo-intensity ranges -----\n")
	err = fetchRanges(getConfigBucket(), name.Rangespath, fs)
	if err != nil {
		return "", err
	}

	inpath := localUnzipPath
	updateExisting := false
	allthefiles := []string{}
	//allthefiles = append(allthefiles, inpath)
	// As this datasource is now in the process flow, copy to the archive folder for re-processing and historical purposes
	jobLog.Infof("----- Copying file %v %v to archive: %v %v -----\n", sourcebucket, name.Inpath, getConfigBucket(), "Datasets/archive/"+name.Inpath)
	err = fs.CopyObject(sourcebucket, name.Inpath, getDatasourceBucket(), "Datasets/archive/"+name.Inpath)
	if err != nil {
		return "", err
	}
	// Lookup other parts of the dataset that have already been processed
	splits := strings.Split(name.Inpath, "-")
	rtt := splits[0]
	check, err := checkExistingArchive(allthefiles, rtt, &updateExisting, fs, jobLog)
	//allthefiles = append(files)
	if err != nil {
		return "", err
	}
	if len(check) > 0 {
		updateExisting = true
	}

	// Download the input file from the preprocess bucket -- Should be with the existing archive to ensure order
	jobLog.Infof("----- Importing file %v -----\n", name.Inpath)
	_, err = downloadDirectoryZip(sourcebucket, name.Inpath, fs)
	if err != nil {
		return "", err
	}
	//Unzip the files into the same folder
	jobLog.Infof("Unzipping all archives...")
	for _, p := range allthefiles {
		if strings.HasSuffix(p, ".zip") || strings.Contains(p, "zip") {
			//fmt.Printf("Preparing to unzip %v\n", p)
			_, err := utils.UnzipDirectory(p, localUnzipPath)
			if err != nil {
				jobLog.Errorf("Unzip failed for \"%v\". Error: \"%v\"\n", p, err)
				return "", err
			}
			name.Inpath = inpath
		}
	}

	// Get the extra manual stuff
	err = downloadExtraFiles(rtt, fs)
	if err != nil {
		return "", err
	}
	r, err := processFiles(inpath, name, importers, creationUnixTimeSec, updateExisting, fs, ns, targetbucket, jobLog)
	return r, err

}

// processFiles - Once files have been downloasded, process the files to generate the datasource and upload the results
func processFiles(inpath string, name DatasourceEvent, importers map[string]importer.Importer,
	creationUnixTimeSec int64, updateExisting bool, fs fileaccess.FileAccess, ns apiNotifications.NotificationManager, targetbucket string, jobLog logger.ILogger) (string, error) {
	argFormat := "pixl-fm"
	var argInPath = inpath
	var argOutPath = name.Outpath
	argOutDatasetIDOverride, err := computeName()

	flag.Parse()

	importing, ok := importers[argFormat]
	if !ok {
		s := fmt.Sprintf("Importer for format \"%v\" not found\n", argFormat)
		return s, nil
	}

	fmt.Printf("----- Importing %v dataset: %v -----\n", argFormat, argInPath)

	data, contextImageSrcPath, err := importing.Import(argInPath, localRangesCSVPath, jobLog)
	if err != nil {
		s := fmt.Sprintf("IMPORT ERROR: %v\n", err)
		return s, err
	}

	// Override dataset ID for output if required
	if argOutDatasetIDOverride != "" && len(argOutDatasetIDOverride) > 0 && argOutDatasetIDOverride != " " {
		data.DatasetID = argOutDatasetIDOverride
	}

	customDetectorVal, err := customDetector(data.Meta.SOL)
	if err != nil {
		return "", err
	}
	if customDetectorVal == "" {
		customDetectorVal = name.DetectorConfig
	}

	// Override detector config if required
	if customDetectorVal != "" && len(customDetectorVal) > 0 {
		data.DetectorConfig = customDetectorVal
	}

	customGroupVal, err := customGroup(customDetectorVal)
	if err != nil {
		return "", err
	}
	data.Group = customGroupVal
	// Form the output path
	outPath := path.Join(argOutPath, data.DatasetID)

	jobLog.Infof("----- Checking for Quick/Autolooks: %v -----\n", outPath)

	importAutoQuickLook(outPath)

	jobLog.Infof("----- Writing Dataset to: %v -----\n", outPath)
	saver := output.PIXLISEDataSaver{}
	err = saver.Save(*data, contextImageSrcPath, outPath, creationUnixTimeSec, jobLog)
	if err != nil {
		s := fmt.Sprintf("WRITE ERROR: %v\n", err)
		return s, err
	}

	jobLog.Infof("----- Calculating Diffraction Peaks: %v -----\n", data.DatasetID)

	err = createPeakDiffractionDB(path.Join(outPath, filepaths.DatasetFileName), path.Join(outPath, filepaths.DiffractionDBFileName), jobLog)

	if err != nil {
		return "", err
	}

	err = copyAdditionalDirectories(outPath, jobLog)
	if err != nil {
		return "", err
	}
	if targetbucket == "" {
		jobLog.Errorf("No Target Bucket Defined, exiting")
	} else {
		err = uploadDirectoryToAllEnvironments(fs, outPath, data.DatasetID, []string{targetbucket}, jobLog)
	}
	if err != nil {
		return "", err
	}

	jobLog.Infof("------ Triggering Notifications ------\n")
	update, err := overrideUpdateType()
	if update != "trvial" {
		updatenotificationtype, err := getUpdateNotificationType(data.DatasetID, getDatasourceBucket(), fs)
		if err != nil {
			return "", err
		}
		_, err = triggernotifications(fs, updateExisting, updatenotificationtype, ns, jobLog)
		if err != nil {
			return "", err
		}
	}

	return "", nil
}
