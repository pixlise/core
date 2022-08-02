// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package quantModel

import (
	"sync"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
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
