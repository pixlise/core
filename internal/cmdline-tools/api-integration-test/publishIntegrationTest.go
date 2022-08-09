package main

import (
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/session"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	apiNotifications "gitlab.com/pixlise/pixlise-go-api/core/notifications"
	"gitlab.com/pixlise/pixlise-go-api/core/pixlUser"
	quant "gitlab.com/pixlise/pixlise-go-api/core/quantModel"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func runPublishTests() error {
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

	config := quant.PublisherConfig{
		KubernetesLocation:      "external",
		QuantDestinationPackage: "f1d96d04-5bba-426d-a451-05085218df9f",
		QuantObjectType:         "m20-soas-pixl-spec-eng",
		PosterImage:             "registry.gitlab.com/pixlise/ocs-poster:latest",
		DatasetsBucket:          "artifactsstack-artifactstestdatasourcepixliseorg0-1e6ukb3gjcp3e",
		EnvironmentName:         "dev",
		Kubeconfig:              *kubeconfig,
		UsersBucket:             "", // TODO: this doesn't make sense... the data bucket is set to artifacts above? What do we put here then? Real code set the data bucket to the actual dataset bucket!
	}

	creator := pixlUser.UserInfo{
		Name:        "testuser",
		UserID:      "testid",
		Email:       "testuser@jpl.nasa.gov",
		Permissions: nil,
	}

	var log = logger.StdOutLogger{}
	dataset := "130744834"
	job := "lrk6t6etwxmfs6dn"
	var notes []apiNotifications.UINotificationObj

	notificationStack := &apiNotifications.DummyNotificationStack{
		Notifications: notes,
		FS:            fs,
		Bucket:        os.Getenv("notificationBucket"),
		Track:         make(map[string]bool),
		Environment:   "prod",
		Logger:        logger.NullLogger{},
	}
	return quant.PublishQuant(fs, config, creator, log, dataset, job, notificationStack)
}
