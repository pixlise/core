package jobmanager

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/olahol/melody"
	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/job/jobnode"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

func completeExpressionJob(jg *jobconfig.JobGroupConfig, jstatus *protos.JobStatus, session *melody.Session, svcs *services.APIServices) error {
	// Take output csv, validate it, memoise it
	outPath := ""
	outBucket := ""

	for _, cfg := range jg.NodeConfig.OutputFiles {
		filename := path.Base(cfg.RemotePath)
		if filename == jobnode.ExpressionJobOutputFileName {
			outBucket = cfg.RemoteBucket
			outPath = cfg.RemotePath
			break
		}
	}

	if len(outPath) <= 0 || len(outBucket) <= 0 {
		return fmt.Errorf("Failed to find expression output path for job: %v", jg.JobGroupId)
	}

	outCSVData, err := svcs.FS.ReadObject(outBucket, outPath)
	if err != nil {
		return fmt.Errorf("Failed to read expression output s3://%v/%v for job: %v", outBucket, outPath, jg.JobGroupId)
	}

	if len(outCSVData) <= 0 {
		return fmt.Errorf("Expression output s3://%v/%v was empty job: %v", outBucket, outPath, jg.JobGroupId)
	}

	// Read it and memoise it
	outCSV, err := dataImportHelpers.ReadCSVData(bytes.NewReader([]byte(outCSVData)), 0, ',')
	if err != nil {
		return fmt.Errorf("Failed to parse output s3://%v/%v for job: %v", outBucket, outPath, jg.JobGroupId)
	}
	if len(outCSV) < 2 {
		return fmt.Errorf("Result CSV is empty s3://%v/%v for job: %v", outBucket, outPath, jg.JobGroupId)
	}

	// Expect columns
	if outCSV[0][0] != "PMC" && outCSV[0][0] != "value" {
		return fmt.Errorf("Unexpected result CSV format for s3://%v/%v for job: %v", outBucket, outPath, jg.JobGroupId)
	}

	// Notify that job is complete, client should be able to retrieve it via memoisation retrieval
	memoKey := ""
	memoKeyPrefix := "memoKey="
	for _, arg := range jg.NodeConfig.Args {
		if strings.HasPrefix(arg, memoKeyPrefix) {
			memoKey = arg[len(memoKeyPrefix):]
			break
		}
	}

	if len(memoKey) <= 0 {
		return fmt.Errorf("Failed to determine memoisation key for job: %v", jg.JobGroupId)
	}

	// Read args, expect key=value pairs
	argLookup, err := utils.ReadKeyValueList([]string{"scanId", "quantId", "expressionId", "memoKey"}, jg.NodeConfig.Args)
	if err != nil {
		return fmt.Errorf("Failed to memoise job %v result: %v", jg.JobGroupId, err)
	}

	// Convert to the right format
	valueRange := scan.MinMax{}
	memValues := []*protos.MemPMCDataValue{}

	for c, row := range outCSV[1:] {
		pmc, err := strconv.Atoi(row[0])
		if err != nil {
			return fmt.Errorf("Failed to read PMC from row %v [%v] in s3://%v/%v for job: %v. Error: %v", c+1, row[0], outBucket, outPath, jg.JobGroupId, err)
		}

		isUndef := false
		var value float64
		if row[1] == "null" {
			isUndef = true
			value = math.NaN()
		} else {
			value, err = strconv.ParseFloat(row[1], 32)
			if err != nil {
				return fmt.Errorf("Failed to read value from row %v [%v] in s3://%v/%v for job: %v. Error: %v", c+1, row[1], outBucket, outPath, jg.JobGroupId, err)
			}

			valueRange.Expand(value)
		}

		memValues = append(memValues, &protos.MemPMCDataValue{
			Pmc:         uint32(pmc),
			Value:       float32(value),
			IsUndefined: isUndef,
			Label:       "",
		})
	}

	m := &protos.MemPMCDataValues{
		Values:   memValues,
		MinValue: float32(*valueRange.Min),
		MaxValue: float32(*valueRange.Max),
		IsBinary: false,
		Warning:  "",
	}

	// Memoise the result
	_, _, err = memoise(memoKey, argLookup["scanId"], argLookup["quantId"], argLookup["expressionId"], argLookup["roiId"], jstatus.RequestorUserId, m, svcs)

	return err
}

func memoise(memCacheKey string, scanId, quantId, expressionId, roiId, requestorUserId string, m *protos.MemPMCDataValues, svcs *services.APIServices) (*protos.MemoisedItem, *protos.MemDataQueryResult, error) {
	// We just read the expression item here, not checking permissions because we've gotten this
	// far, we can assume they were checked already
	exprItem := &protos.DataExpression{}
	err := expressionrunner.ReadOne(dbCollections.ExpressionsName, bson.M{"_id": expressionId}, exprItem, svcs.MongoDB)
	if err != nil {
		return nil, nil, err
	}

	memResult := &protos.MemDataQueryResult{
		ResultValues: m,
		IsPMCTable:   true,
		Expression:   exprItem,
	}

	/*
		if (result.region) {
			memResult.region = MemRegionSettings.create({
			region: result.region.region,
			displaySettings: ROIItemDisplaySettings.create({
				colour: result.region.displaySettings.colour.asString(),
				shape: result.region.displaySettings.shape,
			}),
			pixelIndexSet: memPixelIndexSet,
			});
		}
	*/

	data, err := proto.Marshal(memResult)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create memoise item for expression: %v. Error: %v", expressionId, err)
	}

	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	opt := options.Update().SetUpsert(true)

	timestamp := uint32(svcs.TimeStamper.GetTimeNowSec())
	item := &protos.MemoisedItem{
		Key:                 memCacheKey,
		MemoTimeUnixSec:     timestamp,
		Data:                data,
		ScanId:              scanId,
		QuantId:             quantId,
		ExprId:              expressionId,
		DataSize:            uint32(len(data)),
		LastReadTimeUnixSec: timestamp, // Right now this is the last time it was accessed. To be updated in future get calls
		MemoWriterUserId:    requestorUserId,
	}

	result, err := coll.UpdateByID(ctx, memCacheKey, bson.D{{Key: "$set", Value: item}}, opt)
	if err != nil {
		return nil, nil, err
	}

	if result.UpsertedCount == 0 && (result.MatchedCount != result.ModifiedCount) {
		svcs.Log.Errorf("memoise expression result for: %v got unexpected DB write result: %+v", memCacheKey, result)
	}

	return item, memResult, nil
}
