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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pixlise/core/v2/api/endpoints"
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/expressions/expressions"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/utils"
)

// What to do:
// - Get list of files & last mod time from old bucket
// - Get mapping of auth0 nickname to auth0 username (maybe actually using auth0 user id really!)
// - Move ViewState/*.json into ViewState/Last/*.json
// - Edit all .json files creator info, add correct auth0 user name and last modified date
// - Add blank tags field to those that need it (?)

// Files to edit:
// /UserContent/UserID/ElementSets.json
// /UserContent/UserID/DataExpressions.json
// /UserContent/UserID/RGBMixes.json

// /UserContent/UserID/DatasetID/ROI.json
// /UserContent/UserID/DatasetID/SpectrumAnnotation.json
// /UserContent/UserID/DatasetID/multi-quant-z-stack.json
// /UserContent/UserID/DatasetID/Quantifications/summary-<Id>.json
// /UserContent/UserID/DatasetID/ViewState/Workspaces/Id.json
// /UserContent/UserID/DatasetID/ViewState/WorkspaceCollections/Id.json

// /UserContent/shared/ElementSets.json
// /UserContent/shared/DataExpressions.json
// /UserContent/shared/RGBMixes.json

// /UserContent/shared/DatasetID/ROI.json
// /UserContent/shared/DatasetID/ViewState/Workspaces/Id.json
// /UserContent/shared/DatasetID/ViewState/WorkspaceCollections/Id.json

func main() {
	fmt.Println("==============================")
	fmt.Println("=  PIXLISE data fixer        =")
	fmt.Println("==============================")

	//ilog := &logger.StdOutLogger{}

	var cmd = flag.String("cmd", "getdates", "Command to run. getdates, writedates")
	var oldBucket = flag.String("oldbucket", "", "Name of old copy of data bucket for getdates")
	var newBucket = flag.String("newbucket", "", "Name of new users data bucket for getdates")
	//var localUserDataPath = flag.String("userdatapath_in", "", "Path to local copy of user data files")
	//var localUserDataOutputPath = flag.String("userdatapath_out", "", "Path to write user data files amended with timestamp fields")
	var savedTimes = flag.String("savedtimes", "", "Path to saved user file creation times CSV")
	var resumeFromUserID = flag.String("resumeuser", "", "User ID to resume from")
	var resumeFromDatasetID = flag.String("resumedataset", "", "Dataset ID to resume from")
	flag.Parse()

	var err error

	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("AWS GetSession failed: %v", err)
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("AWS GetS3 failed: %v", err)
	}

	//localFS := &fileaccess.FSAccess{}
	//remoteFS := fileaccess.MakeS3Access(svc)

	if *cmd == "getdates" {
		getDates(svc, *oldBucket, *newBucket)
	} else if *cmd == "writedates" {
		//writeDates(svc, *localUserDataPath, *localUserDataOutputPath)
		updateDates(svc, *savedTimes, *newBucket, *resumeFromUserID, *resumeFromDatasetID)
	} else {
		log.Fatalf("Unknown command: %v", *cmd)
	}
}

func updateDates(svc s3iface.S3API, savedTimesPath string, userBucket string, resumeFromUserID string, resumeFromDatasetID string) {
	remoteFS := fileaccess.MakeS3Access(svc)

	// Read all paths and form a map, while getting all user and dataset IDs
	datasetIDs, userIDs, fileCreateLookup, workspaceFiles, collectionFiles := makeLookups(savedTimesPath)

	// Now run through the files we need to edit and add the creation time
	sortedDatasetIDs := []string{}
	for datasetID := range datasetIDs {
		sortedDatasetIDs = append(sortedDatasetIDs, datasetID)
	}
	sort.Strings(sortedDatasetIDs)

	sortedUserIDs := []string{}
	for userID := range userIDs {
		if userID != "notifications" {
			sortedUserIDs = append(sortedUserIDs, userID)
		}
	}
	sort.Strings(sortedUserIDs)

	// Per-user (and shared files)
	sortedUserIDs = append(sortedUserIDs, pixlUser.ShareUserID)

	for _, userID := range sortedUserIDs {
		if len(resumeFromUserID) > 0 {
			if userID != resumeFromUserID {
				fmt.Printf("Skipping user ID: %v\n", userID)
				continue
			} else {
				// we've found it, clear the user ID we're searching for
				resumeFromUserID = ""
			}
		}
		fmt.Printf("Editing for user ID: %v\n", userID)

		// Local to user files, here we just set the same create time on each object in the file
		// /UserContent/UserID/ElementSets.json
		editElementSets(remoteFS, userBucket, userID, fileCreateLookup)

		// /UserContent/UserID/DataExpressions.json
		editExpressions(remoteFS, userBucket, userID, fileCreateLookup)

		// /UserContent/UserID/RGBMixes.json
		editRGBMixes(remoteFS, userBucket, userID, fileCreateLookup)

		for _, datasetID := range sortedDatasetIDs {
			if len(resumeFromDatasetID) > 0 {
				if datasetID != resumeFromDatasetID {
					fmt.Printf("Skipping dataset ID: %v\n", datasetID)
					continue
				} else {
					// we've found it, clear the user ID we're searching for
					resumeFromDatasetID = ""
				}
			}

			fmt.Printf("Editing for user ID: %v, dataset ID: %v\n", userID, datasetID)

			// /UserContent/UserID/DatasetID/ROI.json
			editROIs(remoteFS, userBucket, userID, datasetID, fileCreateLookup)

			// /UserContent/UserID/DatasetID/SpectrumAnnotation.json
			editSpectrumAnnotations(remoteFS, userBucket, userID, datasetID, fileCreateLookup)

			// /UserContent/UserID/DatasetID/multi-quant-z-stack.json
			// /UserContent/UserID/DatasetID/Quantifications/summary-<Id>.json

			// Check if there are workspace files we have timestamps for
			workspaceKey := userID + "_" + datasetID

			filePaths, ok := workspaceFiles[workspaceKey]
			if ok {
				for _, workspacePath := range filePaths {
					editWorkspaces(remoteFS, userBucket, userID, workspacePath, fileCreateLookup)
				}
			}

			filePaths, ok = collectionFiles[workspaceKey]
			if ok {
				for _, collectionPath := range filePaths {
					editCollections(remoteFS, userBucket, userID, collectionPath, fileCreateLookup)
				}
			}
		}
	}
}

func editElementSets(remoteFS fileaccess.FileAccess, userBucket string, userID string, fileCreateLookup map[string]int) {
	filePath := filepaths.GetElementSetPath(userID)
	itemLookup := endpoints.ElementSetLookup{}

	editArrayFile(remoteFS, userBucket, userID, filePath, fileCreateLookup, itemLookup)
}

func editExpressions(remoteFS fileaccess.FileAccess, userBucket string, userID string, fileCreateLookup map[string]int) {
	filePath := filepaths.GetExpressionPath(userID)
	itemLookup := expressions.DataExpressionLookup{}

	editArrayFile(remoteFS, userBucket, userID, filePath, fileCreateLookup, itemLookup)
}

func editRGBMixes(remoteFS fileaccess.FileAccess, userBucket string, userID string, fileCreateLookup map[string]int) {
	filePath := filepaths.GetRGBMixPath(userID)
	itemLookup := endpoints.RGBMixLookup{}

	editArrayFile(remoteFS, userBucket, userID, filePath, fileCreateLookup, itemLookup)
}

func editROIs(remoteFS fileaccess.FileAccess, userBucket string, userID string, datasetID string, fileCreateLookup map[string]int) {
	filePath := filepaths.GetROIPath(userID, datasetID)
	itemLookup := roiModel.ROILookup{}

	editArrayFile(remoteFS, userBucket, userID, filePath, fileCreateLookup, itemLookup)
}

func editSpectrumAnnotations(remoteFS fileaccess.FileAccess, userBucket string, userID string, datasetID string, fileCreateLookup map[string]int) {
	filePath := filepaths.GetAnnotationsPath(userID, datasetID)
	itemLookup := endpoints.AnnotationLookup{}

	editArrayFile(remoteFS, userBucket, userID, filePath, fileCreateLookup, itemLookup)
}

func editWorkspaces(remoteFS fileaccess.FileAccess, userBucket string, userID string, workspacePath string, fileCreateLookup map[string]int) {
	itemLookup := endpoints.Workspace{}

	editItemFile(remoteFS, userBucket, userID, workspacePath, fileCreateLookup, itemLookup)
}

func editCollections(remoteFS fileaccess.FileAccess, userBucket string, userID string, collectionPath string, fileCreateLookup map[string]int) {
	itemLookup := endpoints.WorkspaceCollection{}

	editItemFile(remoteFS, userBucket, userID, collectionPath, fileCreateLookup, itemLookup)
}

type setTimeItem interface {
	endpoints.SpectrumAnnotationLine | expressions.DataExpression | endpoints.RGBMix | endpoints.ElementSet | roiModel.ROISavedItem | endpoints.Workspace | endpoints.WorkspaceCollection
	SetTimes(string, int64)
}

func SetTimes[T setTimeItem](x *T, userID string, t int64) {
	(*x).SetTimes(userID, t)
}

func editArrayFile[K comparable, V setTimeItem]( // endpoints.SpectrumAnnotationLine | expressions.DataExpression | endpoints.RGBMix | endpoints.ElementSet | roiModel.ROISavedItem](
	remoteFS fileaccess.FileAccess,
	userBucket string,
	userID string,
	filePath string,
	fileCreateLookup map[string]int,
	itemLookup map[K]V) {
	createTime, ok := fileCreateLookup[filePath]
	if !ok {
		log.Printf("No saved create time for: %v\n", filePath)
		return
	}

	// Need to save a pre-edited copy too, so we go the long way...
	//err := remoteFS.ReadJSON(userBucket, filePath, &item, true)
	fileData, err := getS3Data(remoteFS, userBucket, filePath)
	if err != nil {
		log.Fatalf("Failed to download: %v. Error: %v", filePath, err)
	}

	err = json.Unmarshal(fileData, &itemLookup)

	if err != nil {
		log.Fatalf("Failed to read: %v. Error: %v", filePath, err)
	}

	if len(itemLookup) > 0 {
		// Read them all and add create time field
		for _, item := range itemLookup {
			SetTimes(&item, userID, int64(createTime))
		}

		writeEdited(remoteFS, userBucket, filePath, &itemLookup)
	}
}

func editItemFile[V setTimeItem]( // endpoints.SpectrumAnnotationLine | expressions.DataExpression | endpoints.RGBMix | endpoints.ElementSet | roiModel.ROISavedItem](
	remoteFS fileaccess.FileAccess,
	userBucket string,
	userID string,
	filePath string,
	fileCreateLookup map[string]int,
	item V) {
	createTime, ok := fileCreateLookup[filePath]
	if !ok {
		log.Printf("No saved create time for: %v\n", filePath)
		return
	}

	// Need to save a pre-edited copy too, so we go the long way...
	//err := remoteFS.ReadJSON(userBucket, filePath, &item, true)
	fileData, err := getS3Data(remoteFS, userBucket, filePath)
	if err != nil {
		log.Fatalf("Failed to download: %v. Error: %v", filePath, err)
	}

	err = json.Unmarshal(fileData, &item)
	if err != nil {
		log.Fatalf("Failed to read: %v. Error: %v", filePath, err)
	}

	SetTimes(&item, userID, int64(createTime))
	writeEdited(remoteFS, userBucket, filePath, &item)
}

func getS3Data(remoteFS fileaccess.FileAccess, userBucket string, filePath string) ([]byte, error) {
	fileData, err := remoteFS.ReadObject(userBucket, filePath)
	if err != nil {
		return nil, err
	}

	// Save a local copy
	err = os.MkdirAll("./output/pre-edit/"+path.Dir(filePath), 0777)
	if err != nil {
		return nil, err
	}

	fs := &fileaccess.FSAccess{}
	filePath = "./output/pre-edit/" + filePath

	err = fs.WriteObject(filePath, "", fileData)
	if err != nil {
		return nil, err
	}

	return fileData, nil
}

func writeEdited(remoteFS fileaccess.FileAccess, userBucket string, filePath string, itemPtr interface{}) {
	// For now we write to local file system only
	err := os.MkdirAll("./output/edited/"+path.Dir(filePath), 0777)
	if err != nil {
		log.Fatal(err)
	}

	fs := &fileaccess.FSAccess{}
	filePathLocal := "./output/edited/" + filePath

	err = fs.WriteJSON(filePathLocal, "", itemPtr)
	if err != nil {
		log.Fatalf("Failed to write: %v. Error: %v", filePathLocal, err)
	}

	// Now write to AWS too
	err = remoteFS.WriteJSON(userBucket, filePath, itemPtr)
	if err != nil {
		log.Fatalf("Failed to write to S3: %v. Error: %v", filePath, err)
	}
}

func makeLookups(savedTimesPath string) (map[string]bool, map[string]bool, map[string]int, map[string][]string, map[string][]string) {
	savedLines, err := utils.ReadFileLines(savedTimesPath)
	if err != nil {
		log.Fatal(err)
	}

	datasetIDs := map[string]bool{}
	userIDs := map[string]bool{}
	fileCreateLookup := map[string]int{}
	workspaceFiles := map[string][]string{}
	collectionFiles := map[string][]string{}

	for c, line := range savedLines {
		if c == 0 {
			continue // header row
		}

		// Break it into parts, expect stuff like:
		// "UserContent/5de45d85ca40070f421a3a34/012221_83_sand/ViewState","contextImage-analysis.json",1618310969,605,605
		// so there should ONLY be 5 parts, if not, complain
		bits := []string{}
		pos := strings.Index(line[0:], "\",")
		bits = append(bits, line[0:pos+1])

		pos2 := pos + 2 + strings.Index(line[pos+2:], "\",")
		bits = append(bits, line[pos+2:pos2+1])

		endBits := strings.Split(line[pos2+2:], ",")
		bits = append(bits, endBits...)

		if len(bits) != 5 {
			log.Printf("(%v): Unexpected comma count: %v\n", c+1, line)
			continue
		}

		// Get them with "" stripped off
		filePath := bits[0][1 : len(bits[0])-1]
		fileName := bits[1][1 : len(bits[1])-1]

		// Save this in our create time lookup
		createTime, err := strconv.Atoi(bits[2])
		if err != nil {
			log.Printf("(%v): Unexpected create time: %v\n", c+1, line)
			continue
		}

		// Check it's in the right subdir
		if !strings.HasPrefix(filePath, filepaths.RootUserContent) {
			log.Printf("(%v): Skipping non-user-content %v...\n", c+1, filePath)
			continue
		}

		// We're only editing JSON files
		if !strings.HasSuffix(fileName, ".json") {
			log.Printf("(%v): Skipping non-JSON %v...\n", c+1, fileName)
			continue
		}

		pathBits := strings.Split(filePath, "/")
		userID := ""
		datasetID := ""

		// Get the user ID (2nd part)
		if len(pathBits) > 1 && pathBits[1] != "notifications" {
			userID = pathBits[1]
			userIDs[userID] = true
		}

		// Dataset ID is the 3rd part
		if len(pathBits) > 2 {
			datasetID = pathBits[2]
			datasetIDs[datasetID] = true
		}

		fullPath := filepath.Join(filePath, fileName)
		fileCreateLookup[fullPath] = createTime
		workspaceKey := userID + "_" + datasetID

		if len(userID) > 0 && len(datasetID) > 0 {
			// Save view state related paths because their IDs are not predictable like user/dataset IDs
			if strings.Contains(fullPath, "/ViewState/Workspaces/") {
				if _, ok := workspaceFiles[workspaceKey]; !ok {
					workspaceFiles[workspaceKey] = []string{}
				}
				workspaceFiles[workspaceKey] = append(workspaceFiles[workspaceKey], fullPath)
			}

			if strings.Contains(fullPath, "/ViewState/WorkspaceCollections/") {
				if _, ok := collectionFiles[workspaceKey]; !ok {
					collectionFiles[workspaceKey] = []string{}
				}
				collectionFiles[workspaceKey] = append(collectionFiles[workspaceKey], fullPath)
			}
		}
	}

	return datasetIDs, userIDs, fileCreateLookup, workspaceFiles, collectionFiles
}

/*
	func writeDates(svc s3iface.S3API, localUserFilesPath string, localUserFileOutputPath string) {
		localFS := &fileaccess.FSAccess{}
		files, err := localFS.ListObjects(localUserFilesPath, "")
		if err != nil {
			log.Fatal(err)
		}

		for _, filePath := range files {
			pathBits := strings.Split(filePath, "/")

			// Must be in UserContent dir
			if pathBits[0] != filepaths.RootUserContent {
				log.Printf("Skipping non-user-content %v...\n", filePath)
				continue
			}

			// We're only editing JSON files
			if !strings.HasSuffix(filePath, ".json") {
				log.Printf("Skipping non-json %v...\n", filePath)
				continue
			}

			// Edit the known files we're interested in
			if pathBits[1] == pixlUser.ShareUserID {
				// It's a shared file, check which
				if pathBits[2] == "ElementSets.json" ||
					pathBits[2] == "DataExpressions.json" ||
					pathBits[2] == "RGBMixes.json" {
					log.Printf("Skipping: %v\n", filePath)
				} else {
					// /UserContent/shared/ElementSets.json
					// /UserContent/shared/DataExpressions.json
					// /UserContent/shared/RGBMixes.json

					// /UserContent/shared/DatasetID/ROI.json
					// /UserContent/shared/DatasetID/ViewState/Workspaces/Id.json
					// /UserContent/shared/DatasetID/ViewState/WorkspaceCollections/Id.json

				}
			} else {
			}

// /UserContent/UserID/ElementSets.json
// /UserContent/UserID/DataExpressions.json
// /UserContent/UserID/RGBMixes.json

// /UserContent/UserID/DatasetID/ROI.json
// /UserContent/UserID/DatasetID/SpectrumAnnotation.json
// /UserContent/UserID/DatasetID/multi-quant-z-stack.json
// /UserContent/UserID/DatasetID/Quantifications/summary-<Id>.json
// /UserContent/UserID/DatasetID/ViewState/Workspaces/Id.json
// /UserContent/UserID/DatasetID/ViewState/WorkspaceCollections/Id.json

		}
	}
*/
func getDates(svc s3iface.S3API, oldBucket string, newBucket string) {
	err := os.Mkdir("./output", 0777)
	if err != nil {
		log.Fatal(err)
	}

	// Get a listing of the data buckets
	oldDateLookup, err := getListingWithDates(svc, oldBucket, "UserContent/")
	if err != nil {
		log.Fatal(err)
	}
	saveFileListingCSV("oldbucket-listing.csv", oldDateLookup, false)

	newDateLookup, err := getListingWithDates(svc, newBucket, "UserContent/")
	if err != nil {
		log.Fatal(err)
	}
	saveFileListingCSV("newbucket-listing.csv", newDateLookup, false)

	// Form a single lookup - use new file dates, and overwrite with old file dates if they exist
	overallLookup := map[string][]int64{}
	for key, data := range newDateLookup {
		overallLookup[key] = []int64{data[0], data[1], 0}
	}
	for key, data := range oldDateLookup {
		newData, exists := overallLookup[key]

		if !exists {
			// Old file that no longer exists, save it and mark new size as -1
			overallLookup[key] = []int64{data[0], -1, data[1]}
		} else {
			if newData[0] < data[0] {
				fmt.Printf("Anomaly: new date of file %v is %v, old is %v\n", key, newData[0], data[0])
			}
			overallLookup[key] = []int64{data[0], newData[1], data[1]}
		}
	}
	saveFileListingCSV("all-listing.csv", overallLookup, true)
}

func saveFileListingCSV(outFileName string, dateLookup map[string][]int64, showOldSize bool) {
	fmt.Printf("Saving %v lines into: %v\n", len(dateLookup), outFileName)

	// Write this out as a CSV for our own purposes, break path into path+filename so we can filter easier
	csv := "path,filename,lastmodified,size"
	if showOldSize {
		csv += ",oldsize"
	}
	csv += "\n"

	// Write in sorted order
	keys := []string{}
	for filePath := range dateLookup {
		keys = append(keys, filePath)
	}
	sort.Strings(keys)

	for _, filePath := range keys {
		//for filePath, fileData := range dateLookup {
		fileData := dateLookup[filePath]
		end := ""
		if showOldSize {
			// assume old size is the last in the array
			end = fmt.Sprintf(",%v", fileData[2])
		}
		csv += fmt.Sprintf("\"%v\",\"%v\",%v,%v%v\n", path.Dir(filePath), path.Base(filePath), fileData[0], fileData[1], end)
	}

	localFS := &fileaccess.FSAccess{}
	err := localFS.WriteObject("./output/", outFileName, []byte(csv))
	if err != nil {
		log.Fatal(err)
	}
}

func getListingWithDates(s3Api s3iface.S3API, bucket string, prefix string) (map[string][]int64, error) {
	continuationToken := ""
	result := map[string][]int64{}

	params := s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	for true {
		// If we have a continuation token, add it to the parameters we send...
		if len(continuationToken) > 0 {
			params.ContinuationToken = aws.String(continuationToken)
		}

		listing, err := s3Api.ListObjectsV2(&params)

		if err != nil {
			return result, err
		}

		// Save the returned items...
		for _, item := range listing.Contents {
			// We filter out paths that end in / from S3, these are pointless but can happen if
			// something was made via the web console with create directory, it creates these empty objects...
			if !strings.HasSuffix(*item.Key, "/") {
				result[*item.Key] = []int64{item.LastModified.Unix(), *item.Size}
			}
		}

		if listing.IsTruncated != nil && *listing.IsTruncated && listing.NextContinuationToken != nil {
			continuationToken = *listing.NextContinuationToken
		} else {
			break
		}
	}
	return result, nil
}
