package quantification

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RunAutoQuantifications(scanId string, svcs *services.APIServices, onlyIfNotExists bool) {
	svcs.Log.Infof("Request to run auto-quantifications for scan: %v", scanId)

	// TODO: Make these configuration parameters!
	quantNames := []string{"AutoQuant-PDS", "AutoQuant-PIXL"}
	quantModes := []string{quantModeCombinedAB, quantModeSeparateAB}
	quantElements := [][]string{
		// PDS: intended "Na2O", "MgO", "Al2O3", "SiO2", "P2O5", "SO3", "Cl", "K2O", "CaO", "TiO2", "Cr2O3", "MnO", "FeO-T", "NiO", "ZnO", "Br"
		// But we must specify elements only! Expecting PIQUANT to determine the oxide states to write
		{"Na", "Mg", "Al", "Si", "P", "S", "Cl", "K", "Ca", "Ti", "Cr", "Mn", "Fe", "Ni", "Zn", "Br"},
		// PIXL: intended "Na2O", "MgO", "Al2O3", "SiO2", "P2O5", "SO3", "Cl", "K2O", "CaO", "TiO2", "Cr2O3", "MnO", "FeO-T", "NiO", "ZnO", "GeO", "Br", "Rb2O", "SrO", "Y2O3", "ZrO2"
		// But we must specify elements only! Expecting PIQUANT to determine the oxide states to write
		{"Na", "Mg", "Al", "Si", "P", "S", "Cl", "K", "Ca", "Ti", "Cr", "Mn", "Fe", "Ni", "Zn", "Ge", "Br", "Rb", "Sr", "Y", "Zr"},
	}
	detector := "PIXL/PiquantConfigs/v7"

	allNames := []string{}
	for _, name := range quantNames {
		for _, m := range quantModes {
			allNames = append(allNames, makeAutoQuantName(name, m))
		}
	}

	existingAutoQuants, err := getExistingAutoQuants(scanId, allNames, svcs.MongoDB)
	if err != nil {
		svcs.Log.Errorf("%v", err)
		return
	}

	// If we only want to run when there is no existing one yet
	if len(existingAutoQuants) > 0 {
		if onlyIfNotExists {
			svcs.Log.Errorf("AutoQuant detected existing quantifications: %v. Skipping auto-quantification", strings.Join(existingAutoQuants, ","))
			return
		} else {
			svcs.Log.Infof("AutoQuant detected existing quantifications: %v. Running anyway...", strings.Join(existingAutoQuants, ","))
		}
	} else {
		svcs.Log.Infof("AutoQuant detected no existing auto-quants. Starting...")
	}

	exprPB, err := wsHelpers.ReadDatasetFile(scanId, svcs)
	if err != nil {
		svcs.Log.Errorf("AutoQuant failed to read scan %v to determine PMC list: %v", scanId, err)
		return
	}

	pmcs, err := readQuantifiablePMCs(exprPB, scanId, svcs.Log)
	if err != nil {
		svcs.Log.Errorf("%v", scanId, err)
		return
	}

	// Start all the quants
	for c, name := range quantNames {
		for _, m := range quantModes {
			params := &protos.QuantCreateParams{
				Command:        "map",
				Name:           makeAutoQuantName(name, m),
				ScanId:         scanId,
				Pmcs:           pmcs,
				Elements:       quantElements[c],
				DetectorConfig: detector,
				Parameters:     "-Fe,1",
				RunTimeSec:     300,
				QuantMode:      m,
				RoiIDs:         []string{},
				IncludeDwells:  false,
			}

			i := MakeQuantJobUpdater(params, nil, svcs.Notifier, svcs.MongoDB)
			_, err := CreateJob(params, specialUserIds.PIXLISESystemUserId, svcs, nil, nil, i.SendQuantJobUpdate)
			if err != nil {
				svcs.Log.Errorf("AutoQuant failed to create quant job: %v. Error: %v", params.Name, err)
				return
			}
		}
	}
}

func makeAutoQuantName(name string, quantMode string) string {
	return fmt.Sprintf("%v (%v)", name, quantMode)
}

func getExistingAutoQuants(scanId string, allNames []string, mongoDB *mongo.Database) ([]string, error) {
	// Check if we have auto quant already, if not, run one
	ctx := context.TODO()
	coll := mongoDB.Collection(dbCollections.QuantificationsName)

	filter := bson.M{"scanid": scanId, "params.userparams.name": bson.M{"$in": allNames}}
	opt := options.Find()
	cursor, err := coll.Find(ctx, filter, opt)

	if err != nil {
		return []string{}, fmt.Errorf("AutoQuant failed to read existing quantifications: %v", err)
	}

	result := []*protos.QuantificationSummary{}
	err = cursor.All(ctx, &result)
	if err != nil {
		return []string{}, fmt.Errorf("AutoQuant failed to decode existing quantifications: %v", err)
	}

	existingNames := []string{}
	for _, item := range result {
		existingNames = append(existingNames, fmt.Sprintf("%v (id: %v)", item.Params.UserParams.Name, item.Id))
	}

	return existingNames, nil
}

func readQuantifiablePMCs(exprPB *protos.Experiment, scanId string, logger logger.ILogger) ([]int32, error) {
	pmcs := []int32{}

	for _, loc := range exprPB.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			logger.Errorf("AutoQuant: Failed to read PMC %v from scan %v. Skipping...", loc.Id, scanId)
			continue
		}

		if len(loc.PseudoIntensities) > 0 && len(loc.PseudoIntensities[0].ElementIntensities) > 0 {
			// We have quantifiable data for this
			pmcs = append(pmcs, int32(pmc))
		}
	}

	return pmcs, nil
}
