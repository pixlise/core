// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package runner

import (
	"fmt"
	"os"

	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	cmap "github.com/orcaman/concurrent-map"
	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	apiNotifications "github.com/pixlise/core/v2/core/notifications"
	"github.com/pixlise/core/v2/data-import/output"
)

// getUpdateNotificationType - Get the notificationtype for a dataset update
func getUpdateNotificationType(datasetID string, bucket string, fs fileaccess.FileAccess) (string, error) {
	datasetSummary, err := datasetModel.ReadDataSetSummary(fs, bucket, datasetID)
	if err != nil {
		return "", err
	}

	diff, err := output.SummaryDiff(datasetSummary, bucket, fs)
	if err != nil {
		return "unknown", err
	}
	if diff.MaxSpectra > 0 || diff.BulkSpectra > 0 || diff.DwellSpectra > 0 || diff.NormalSpectra > 0 {
		return "spectra", nil
	} else if diff.ContextImages > 0 {
		return "image", nil
	} else if diff.DriveID > 0 || diff.Site != "" || diff.Target != "" || diff.Title != "" {
		return "housekeeping", nil
	}
	return "unknown", nil
}

// makeNotificationStack - Create a notification stack
func makeNotificationStack(fs fileaccess.FileAccess, log logger.ILogger) apiNotifications.NotificationManager {
	if os.Getenv("MongoSecret") != "" {
		seccache, err := secretcache.New()

		mongo := apiNotifications.MongoUtils{
			SecretsCache:     seccache,
			ConnectionSecret: os.Getenv("MongoSecret"),
			MongoUsername:    os.Getenv("MongoUsername"),
			MongoEndpoint:    os.Getenv("MongoEndpoint"),
			Log:              log,
		}
		err = mongo.Connect()
		if err != nil {
			fmt.Printf("Couldn't connect to mongodb: %v", err)
		}
		return &apiNotifications.NotificationStack{
			Notifications: []apiNotifications.UINotificationObj{},
			FS:            fs,
			Track:         cmap.New(), //make(map[string]bool),
			Bucket:        os.Getenv("notificationBucket"),
			Environment:   "prod",
			MongoUtils:    &mongo,
			Logger:        log,
		}
	} else {
		return nil
	}
}
