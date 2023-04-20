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

package quantModel

import (
	"sync"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
)

func CheckQuantificationNameExists(name string, datasetID string, userID string, svcs *services.APIServices) bool {
	userQuantSummaryPrefixedPath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.QuantSummaryFilePrefix)

	// Do these in parallel
	userQuants := []string{}

	var wg sync.WaitGroup
	wg.Add(2)

	matched := make(chan bool, 1)

	// This just gets the list of summaries
	go func() {
		defer wg.Done()

		userQuants, _ = svcs.FS.ListObjects(svcs.Config.UsersBucket, userQuantSummaryPrefixedPath)
	}()

	// This gets the list of active quant jobs and checks, writes any matches to the channel
	go func() {
		defer wg.Done()

		processing, err := ListQuantJobsForDataset(svcs, userID, datasetID)
		if err == nil {
			for _, inProgItem := range processing {
				if name == inProgItem.Params.Name {
					matched <- true
					return
				}
			}
		}

		matched <- false
	}()

	// Wait for the above 2 to finish. If we still haven't found a match, we need to run through each quant summary returned...
	wg.Wait()
	close(matched)

	// See if any found a match
	for res := range matched {
		if res == true {
			return true
		}
	}

	// Now we have to check each individual quant summary file
	return checkNameMatchesOneSummaries(name, userQuants, svcs)
}

// Downloads job summary file with path: s3://PiquantJobsBucket/RootJobSummaries/<dataset-id>JobSummarySuffix
// Returns only jobs created by userID that are NOT complete (still in progress/failed)
func ListQuantJobsForDataset(svcs *services.APIServices, userID string, datasetID string) ([]JobSummaryItem, error) {
	jobsList := []JobSummaryItem{}

	// Run through all jobs in the summary JSON & put the ones for this dataset here
	jobsPath := filepaths.GetJobSummaryPath(datasetID)
	var jobsMap JobSummaryMap
	err := svcs.FS.ReadJSON(svcs.Config.PiquantJobsBucket, jobsPath, &jobsMap, false)
	if err != nil {
		// We failed to get the jobs list, so it probably doesn't exist (yet), so just return an empty joblist
		svcs.Log.Infof("Failed to download job list: s3://%v/%v, assuming there are just no jobs to report", svcs.Config.PiquantJobsBucket, jobsPath)
		return jobsList, nil
	}

	// Run through each item in map, and save in list of jobs
	for _, item := range jobsMap {
		if item.Params.Creator.UserID == userID && item.Status != JobComplete {
			jobsList = append(jobsList, SetMissingSummaryFields(item))
		}
	}

	return jobsList, nil
}

func checkNameMatchesOneSummaries(name string, userQuants []string, svcs *services.APIServices) bool {
	results := make(chan bool, len(userQuants))

	// Download each one (parallel), if any matches, save this in channel
	var wg sync.WaitGroup
	for _, item := range userQuants {
		//fmt.Println(item)
		wg.Add(1)

		go func(path string) {
			defer wg.Done()

			summary := JobSummaryItem{}
			err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, path, &summary, false)
			//fmt.Println(summary.Params.Name)
			if err == nil && summary.Params.JobStartingParameters != nil && name == summary.Params.Name {
				results <- true
			} else {
				results <- false
			}
		}(item)
	}

	// Wait for all
	wg.Wait()
	close(results)

	// See if any found a match
	for res := range results {
		if res == true {
			return true
		}
	}

	return false
}
