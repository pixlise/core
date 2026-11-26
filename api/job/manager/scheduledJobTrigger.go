package jobmanager

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job/jobrunner"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/memoisation"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (jm *JobManager) RunScheduledPostImportJobs(scan *protos.ScanItem) ([]string, error) {
	jobIds := []string{}
	jm.svcs.Log.Infof("RunScheduledPostImportJobs for scan %v with instrument %v", scan.Id, scan.Instrument)

	jobs, err := jm.ListScheduledJobs()
	if err != nil {
		return jobIds, err
	}

	// Find all jobs relevant to this scan
	chosenJobs := []*protos.ScheduledJob{}
	for _, job := range jobs {
		if job.ScheduleType == protos.ScheduledJob_AFTER_IMPORT && job.Instrument == scan.Instrument {
			// It's a job for our instrument and it's mean to run after import! Add to the list
			// which we will have to sort by order next
			chosenJobs = append(chosenJobs, job)
		}
	}

	if len(chosenJobs) <= 0 {
		jm.svcs.Log.Infof("RunScheduledPostImportJobs: No jobs found for scan %v with instrument %v", scan.Id, scan.Instrument)
		return jobIds, nil
	}

	jm.svcs.Log.Infof("RunScheduledPostImportJobs found %v jobs for scan %v with instrument %v", len(chosenJobs), scan.Id, scan.Instrument)

	sort.Slice(chosenJobs, func(i, j int) bool { return chosenJobs[i].JobOrder < chosenJobs[j].JobOrder })

	// Run the jobs one-by-one
	for _, job := range chosenJobs {
		if jobId, err := jm.RunScheduledJob(job, scan); err != nil {
			return jobIds, err
		} else {
			jobIds = append(jobIds, jobId)
		}
	}

	return jobIds, nil
}

func (jm *JobManager) getProcessedJobParams(names []string, job *protos.ScheduledJob, importedScanId string) (map[string]string, error) {
	result := map[string]string{}

	resolveQuantName := ""

	for _, name := range names {
		value, ok := job.JobParameters[name]
		if !ok {
			return result, fmt.Errorf("Missing parameter: %v", name)
		}

		pos := strings.Index(value, ":")
		if pos < 0 {
			result[name] = value
		} else {
			// We have a prefix, do what's needed here to form a single string to store
			prefix := value[0 : pos+1]
			suffix := value[pos+1:]

			if name == "quant" {
				if prefix == QUANT_BY_NAME_PREFIX {
					resolveQuantName = suffix
					continue
				} else if prefix == QUANT_BY_ID_PREFIX {
					result["quantId"] = suffix
					continue
				} else {
					return result, fmt.Errorf("%v: unknown prefix: %v", name, prefix)
				}
			} else if name == "elements" {
				if prefix == ELEMS_BY_SET {
					// Look up this element set by name, find an id for it
					item := &protos.ElementSet{}
					err := expressionrunner.ReadOne(dbCollections.ElementSetsName, bson.M{"name": suffix}, &item, jm.svcs.MongoDB)
					if err != nil {
						return result, fmt.Errorf("Failed to read element set \"%v\" for job parameter: %v. Error: %v", suffix, name, err)
					}

					// Form the element list here
					elemList := map[string]bool{}
					for _, z := range item.Lines {
						elem := expressionrunner.PTable.GetSymbol(int(z.Z))
						if len(elem) <= 0 {
							return result, fmt.Errorf("Element set \"%v\" for job parameter: %v has unknown element: %v", suffix, name, z.Z)
						}
						elemList[elem] = true
					}

					elemListStr := utils.GetMapKeys(elemList)
					result[name] = strings.Join(elemListStr, ",")

					continue
				} else if prefix == ELEMS_BY_LIST {
					result[name] = suffix
					continue
				} else {
					return result, fmt.Errorf("%v: unknown prefix: %v", name, prefix)
				}
			}

			// Assume it can just be transferred
			result[name] = value
		}
	}

	scanId := result["scanId"]
	if scanId == SCAN_ID_NONE {
		scanId = ""
	}
	if scanId == SCAN_ID_AUTO_IMPORTED {
		// Fall back on the one that was specified to this function
		scanId = importedScanId
		// Make sure the specified one is what's set!
		result["scanId"] = importedScanId
	}

	// If we've still got to resolve the quant name, do it here
	if len(resolveQuantName) > 0 {
		if len(scanId) <= 0 {
			return result, errors.New("No scan id specified, cannot search for quant by name")
		}

		// Look up this quant by name, find an id for it
		/*filter := bson.M{"$and": []bson.M{
			bson.M{"params.userparams.name": resolveQuantName},
			bson.M{"scanId": scanId},
		}}*/
		filter := bson.D{
			{Key: "$and",
				Value: bson.A{
					bson.D{{Key: "params.userparams.name", Value: resolveQuantName}},
					bson.D{{Key: "scanid", Value: scanId}},
				},
			},
		}

		ctx := context.TODO()
		coll := jm.svcs.MongoDB.Collection(dbCollections.QuantificationsName)

		cursor, err := coll.Find(ctx, filter, options.Find())
		if err != nil {
			return result, fmt.Errorf("Failed to find quantification \"%v\" for scan %v: %v", resolveQuantName, scanId, err)
		}

		quants := []*protos.QuantificationSummary{}
		err = cursor.All(ctx, &quants)
		if err != nil {
			return result, fmt.Errorf("Failed to decode quantification(s) \"%v\" for scan %v: %v", resolveQuantName, scanId, err)
		}

		// If there is more than one, complain
		if len(quants) == 0 {
			return result, fmt.Errorf("No quantifications named \"%v\" found for scan %v", resolveQuantName, scanId)
		} else if len(quants) > 1 {
			return result, fmt.Errorf("Found more than one quantification \"%v\" for scan %v", resolveQuantName, scanId)
		}

		quant := quants[0]
		/*
			quant := &protos.QuantificationSummary{}
			err := expressionrunner.ReadOne(dbCollections.QuantificationsName, filter, &quant, jm.svcs.MongoDB)
			if err != nil {
				//return result, fmt.Errorf("Failed to find quantification \"%v\" for scan %v", resolveQuantName, scanId)
				return result, fmt.Errorf("Failed to find quantification \"%v\" for scan %v: %v", resolveQuantName, scanId, err)
			}
		*/
		// Set the quant id
		result["quantId"] = quant.Id
	}

	return result, nil
}

func (jm *JobManager) RunScheduledJob(job *protos.ScheduledJob, scan *protos.ScanItem) (string, error) {
	var jobStatus *protos.JobStatus
	if job.JobType == protos.JobType_JT_RUN_EXPRESSION {
		if scan == nil {
			return "", errors.New("No scanId specified")
		}

		// Get parameters
		params, err := jm.getProcessedJobParams([]string{"scanId", "quant", "expressionId"}, job, scan.Id)
		if err != nil {
			return "", err
		}

		scanId := params["scanId"]
		roiId := ""
		units := protos.DataUnit_UNIT_DEFAULT

		exprItem := &protos.DataExpression{}
		err = expressionrunner.ReadOne(dbCollections.ExpressionsName, bson.M{"_id": params["expressionId"]}, &exprItem, jm.svcs.MongoDB)
		if err != nil {
			return "", fmt.Errorf("Could not read expression %v: %v", params["expressionId"], err)
		}

		// Form a mem cache key
		cacheKey, err := memoisation.MakeCacheKey(scan, exprItem, params["quantId"], roiId, units)
		if err != nil {
			return "", err
		}

		jm.svcs.Log.Infof("RunScheduledJob submitting expression job %v...", job.Id)
		jobStatus, err = jm.SubmitExpressionJob(scanId, params["quantId"], params["expressionId"], roiId, cacheKey, nil, nil)
		return jobStatus.JobId, err
	} else if job.JobType == protos.JobType_JT_RUN_QUANT {
		if scan == nil {
			return "", errors.New("No scanId specified")
		}

		// Get parameters
		params, err := jm.getProcessedJobParams([]string{"scanId", "elements", "quantName", "configName", "quantMode"}, job, scan.Id)
		if err != nil {
			return "", err
		}

		scanId := params["scanId"]

		// Get the list of PMCs to quantify
		exprPB, err := wsHelpers.ReadDatasetFile(scanId, jm.svcs)
		if err != nil {
			return "", fmt.Errorf("AutoQuant failed to read scan %v to determine PMC list: %v", scanId, err)
		}

		pmcs, err := quantification.ReadQuantifiablePMCs(exprPB, scanId, jm.svcs.Log)
		if err != nil {
			return "", fmt.Errorf("Failed to read PMCs from %v: %v", scanId, err)
		}

		createParams := &protos.QuantCreateParams{
			Command:        "map",
			Name:           params["quantName"],
			ScanId:         scanId,
			Pmcs:           pmcs,
			Elements:       strings.Split(params["elements"], ","),
			DetectorConfig: params["configName"],
			Parameters:     "-Fe,1",
			RunTimeSec:     300,
			QuantMode:      params["quantMode"],
			RoiIDs:         []string{},
			IncludeDwells:  false,
		}

		jm.svcs.Log.Infof("RunScheduledJob submitting quant job %v...", job.Id)
		jobStatus, err = jm.SubmitQuantJob(createParams, nil, nil)
		return jobStatus.JobId, err
	} else if job.JobType == protos.JobType_JT_RUN_PYTHON_SCRIPT {
		// Get parameters
		importedScanId := ""
		if scan != nil {
			importedScanId = scan.Id
		}

		expectedParams := []string{"repositoryId", jobrunner.ArgBranchName, jobrunner.ArgExecFileName, "scanId", "quant", "clientAuthId"}
		params, err := jm.getProcessedJobParams(expectedParams, job, importedScanId)
		if err != nil {
			return "", err
		}

		jm.svcs.Log.Infof("RunScheduledJob submitting python job %v...", job.Id)
		jobStatus, err = jm.SubmitPythonJob(params["repositoryId"], params[jobrunner.ArgBranchName], params[jobrunner.ArgExecFileName], params["scanId"], params["quantId"], params["clientAuthId"], nil, nil)
		return jobStatus.JobId, err
	}

	return "", fmt.Errorf("RunScheduledJob cannot run job %v of type %v", job.Id, job.JobType)
}
