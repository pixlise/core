package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/kubernetes"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/core/ocs"
	"gitlab.com/pixlise/pixlise-go-api/core/pixlUser"
	apiv1 "k8s.io/api/core/v1"
)

const targetbucket = "artifactsstack-artifactstestdatasourcesopspixlise-1wk6on67z9ar0"
const rtt = "130089473"
const sqsurl = "https://sqs.us-east-1.amazonaws.com/963058736014/ArtifactsStack-artifactsSStageOCSQueueC1B0B9EC-1GNLVUHSA8Y0U"

var l = logger.StdOutLogger{}

// Run an end to end OCS test, posting data to datadrive and reading the notifications coming back.
func runOCSTests() error {
	l.Infof("Starting OCS Integration Test")
	sess, err := session.NewSession()
	if err != nil {
		l.Errorf("%v", err)
	}

	svc, err := awsutil.GetS3(sess)
	fs := fileaccess.MakeS3Access(svc)
	var kubeconfig *string
	home := homeDir()
	kubeconf := filepath.Join(home, ".kube", "config")
	kubeconfig = &kubeconf
	k := kubernetes.KubeHelper{
		Kubeconfig: *kubeconfig,
	}

	k.Bootstrap("external", l)

	err = setupIntegrationTest(sess, fs)
	if err != nil {
		l.Errorf("%v", err)
	}
	uploadDataset(k)
	err = downloadDataset(k, fs)
	if err != nil {
		l.Errorf("%v", err)
	}
	l.Infof("Ending OCS Integration Test")
	return err
}

//Empty the queue and s3 bucket for a clean test
func setupIntegrationTest(sess *session.Session, fs fileaccess.FileAccess) error {
	// Clear out SQS Queue
	l.Infof("Purging SQS Queue")
	err := awsutil.PurgeQueue(*sess, sqsurl)
	if err != nil {
		l.Errorf("%v", err)
	}

	if err != nil {
		return err
	}
	l.Infof("Emptying S3 Bucket")
	err = fs.EmptyObjects(targetbucket)
	l.Infof("Deleted object(s) from bucket: %s", targetbucket)

	if err != nil {
		return err
	}
	return nil
}

//Upload Data to Datadriving using the OCS Poster docker container
func uploadDataset(k kubernetes.KubeHelper) {
	l.Infof("Starting Upload Dataset Process")
	// Fetch dataset and stage to S3
	var filelist = []string{"datadrive-testsource/BGT/PE__0367_0699557777_000BGT__01101081300894730003___J02.CSV"}
	// Use poster functions to post to datadrive

	products := strings.Join(filelist, ",")
	sourceBucket := "artifactsstack-artifactstestdatasourcepixliseorg0-1e6ukb3gjcp3e"
	destinationPackage := "f1d96d04-5bba-426d-a451-05085218df9f"
	objectType := "m20-soas-pixl-spec-eng"
	path := "/ods/surface/sol/00001/soas/rdr/pixl/BGT"
	posterimage := "registry.gitlab.com/pixlise/ocs-poster:latest"
	creator := pixlUser.UserInfo{
		Name:        "testuser",
		UserID:      "testid",
		Email:       "testuser@jpl.nasa.gov",
		Permissions: nil,
	}

	env := make(map[string]string)
	env["venue"] = "sstage"
	env["credss_username"] = "m20-sstage-pixlise"
	env["credss_appaccount"] = "true"
	env["AWS_PROFILE"] = "default"

	volumes := []apiv1.Volume{
		{
			Name: "aws-volume",
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{Name: "aws-pixlise-config"},
				},
			},
		},
	}

	volumemounts := []apiv1.VolumeMount{
		{
			Name:      "aws-volume",
			MountPath: "/root/.aws/credentialstmp",
			SubPath:   "credentials",
		},
	}
	_, err := k.RunPod(nil, ocs.GeneratePosterPodCmd(products, sourceBucket, path, destinationPackage, objectType), env, volumes, volumemounts, posterimage,
		"api", "testposter", generatePodLabels(), creator, l, false)
	if err != nil {
		l.Errorf("Error: %v", err)
	}
	l.Infof("Ending Upload Dataset Process")
}

func generatePodLabels() map[string]string {
	m := make(map[string]string)
	m["datasetId"] = "test"
	m["environment"] = "test"

	return m
}

// Download data from OCS using the OCS Fetcher docker container
func downloadDataset(k kubernetes.KubeHelper, fs fileaccess.FileAccess) error {
	l.Infof("Starting Download Dataset Process")
	// Enable OCS Fetcher
	l := logger.NullLogger{}
	fetcherimage := "registry.gitlab.com/pixlise/ocs-fetcher:latest"
	creator := pixlUser.UserInfo{
		Name:        "testuser",
		UserID:      "testid",
		Email:       "testuser@jpl.nasa.gov",
		Permissions: nil,
	}
	volumes := []apiv1.Volume{
		{
			Name: "aws-volume",
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{Name: "aws-pixlise-config"},
				},
			},
		},
	}

	volumemounts := []apiv1.VolumeMount{
		{
			Name:      "aws-volume",
			MountPath: "/root/.aws/credentialstmp",
			SubPath:   "credentials",
		},
	}
	env := make(map[string]string)
	env["UPLOAD_BUCKET"] = targetbucket
	env["venue"] = "sstage"
	env["credss_username"] = "m20-sstage-pixlise"
	env["credss_appaccount"] = "true"
	env["QUEUE_NAME"] = "ArtifactsStack-artifactsSStageOCSQueueC1B0B9EC-1GNLVUHSA8Y0U"
	env["AWS_PROFILE"] = "csso-sstage"
	env["TEST_BUCKET"] = "m20-sstage-ods"
	env["DATASOURCE_BUCKET"] = targetbucket
	pod, err := k.RunPod(nil, ocs.GenerateFetcherPodCmd(), env, volumes, volumemounts, fetcherimage, "api", "testfetcher", generatePodLabels(), creator, l, true)
	if err != nil {
		l.Errorf("Error detected launching OCS Fetcher: %v", err)
	}
	//
	found := false
	for start := time.Now(); time.Since(start) < 15*time.Minute; {
		l.Infof("Checking for expected data")
		exists, err := checkS3ForFiles(fs, rtt)
		if err != nil {
			return err
		}
		if exists {
			l.Infof("Expected Data Found")
			found = true
			break
		}
		time.Sleep(30 * time.Second)
	}

	err = k.DeletePod("api", pod)
	if err != nil {
		return err
	}

	if !found {
		l.Errorf("Could not find data")
	}

	l.Infof("Ending Download Dataset Process")
	l.Infof("Data found, test complete")
	return nil
}

func checkS3ForFiles(fs fileaccess.FileAccess, folder string) (bool, error) {
	files, err := fs.ListObjects(targetbucket, folder)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}
