package quantification

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RunAutoQuantifications(scanId string, svcs *services.APIServices) {
	svcs.Log.Infof("Running auto-quantifications for scan: %v", scanId)

	// Check if we have auto quant already, if not, run one
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.QuantificationsName)

	// TODO: Make these configuration parameters!
	quantNames := []string{"AutoQuant-PDS", "AutoQuant-PIXL"}
	quantModes := []string{quantModeCombinedAB, quantModeSeparateAB}
	quantElements := [][]string{
		// PDS
		[]string{"Na2O", "MgO", "Al2O3", "SiO2", "P2O5", "SO3", "Cl", "K2O", "CaO", "TiO2", "Cr2O3", "MnO", "FeO-T", "NiO", "ZnO", "Br"},
		[]string{"Na2O", "MgO", "Al2O3", "SiO2", "P2O5", "SO3", "Cl", "K2O", "CaO", "TiO2", "Cr2O3", "MnO", "FeO-T", "NiO", "ZnO", "GeO", "Br", "Rb2O", "SrO", "Y2O3", "ZrO2"},
	}
	detector := "PIXL/v7"

	allNames := []string{}
	for _, name := range quantNames {
		for _, m := range quantModes {
			allNames = append(allNames, makeAutoQuantName(name, m))
		}
	}

	filter := bson.M{"scanid": scanId, "params.name": bson.M{"$in": allNames}}
	opt := options.Find()
	cursor, err := coll.Find(ctx, filter, opt)

	if err != nil {
		svcs.Log.Errorf("AutoQuant failed to read existing quantifications: %v", err)
		return
	}

	result := []*protos.QuantificationSummary{}
	err = cursor.All(ctx, &result)
	if err != nil {
		svcs.Log.Errorf("AutoQuant failed to decode existing quantifications: %v", err)
		return
	}

	exprPB, err := wsHelpers.ReadDatasetFile(scanId, svcs)
	if err != nil {
		svcs.Log.Errorf("AutoQuant failed to read scan %v to determine PMC list: %v", scanId, err)
		return
	}

	pmcs := []int32{}
	for _, loc := range exprPB.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			svcs.Log.Errorf("AutoQuant: Failed to read PMC %v from scan %v. Skipping...", loc.Id, scanId)
			continue
		}

		if len(loc.PseudoIntensities) > 0 && len(loc.PseudoIntensities[0].ElementIntensities) > 0 {
			// We have quantifiable data for this
			pmcs = append(pmcs, int32(pmc))
		}
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
				Parameters:     "",
				RunTimeSec:     0,
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
