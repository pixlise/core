package main

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/semanticversion"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The structs the source DB was constructed with
type SrcUserInfo struct {
	Name        string          `json:"name"`
	UserID      string          `json:"user_id"`
	Email       string          `json:"email"`
	Permissions map[string]bool `json:"-" bson:"-"` // This is a lookup - we don't want this in JSON sent out of API though!
}

type SrcAPIObjectItem struct {
	Shared              bool        `json:"shared"`
	Creator             SrcUserInfo `json:"creator"`
	CreatedUnixTimeSec  int64       `json:"create_unix_time_sec,omitempty"`
	ModifiedUnixTimeSec int64       `json:"mod_unix_time_sec,omitempty"`
}

type SrcDataExpressionExecStats struct {
	DataRequired     []string `json:"dataRequired"`
	RuntimeMS        float32  `json:"runtimeMs"`
	TimeStampUnixSec int64    `json:"mod_unix_time_sec,omitempty"`
}

type SrcModuleReference struct {
	ModuleID string `json:"moduleID"`
	Version  string `json:"version"`
}

type SrcDataExpression struct {
	ID               string               `json:"id" bson:"_id"` // Use as Mongo ID
	Name             string               `json:"name"`
	SourceCode       string               `json:"sourceCode"`
	SourceLanguage   string               `json:"sourceLanguage"` // LUA vs PIXLANG
	Comments         string               `json:"comments"`
	Tags             []string             `json:"tags"`
	ModuleReferences []SrcModuleReference `json:"moduleReferences,omitempty" bson:"moduleReferences,omitempty"`
	Origin           SrcAPIObjectItem     `json:"origin"`
	// NOTE: if modifying below, ensure it's in sync with ExpressionDB StoreExpressionRecentRunStats()
	RecentExecStats *SrcDataExpressionExecStats `json:"recentExecStats,omitempty" bson:"recentExecStats,omitempty"`
	DOIMetadata     SrcDOIMetadata              `json:"doiMetadata,omitempty" bson:"doiMetadata,omitempty"`
}

type SrcDOIRelatedIdentifier struct {
	Identifier string `json:"identifier"`
	Relation   string `json:"relation"`
}

type SrcDOICreator struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
	Orcid       string `json:"orcid"`
}

type SrcDOIContributor struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
	Orcid       string `json:"orcid"`
	Type        string `json:"type"`
}

type SrcDOIMetadata struct {
	Title              string                    `json:"title"`
	Creators           []SrcDOICreator           `json:"creators"`
	Description        string                    `json:"description"`
	Keywords           string                    `json:"keywords"`
	Notes              string                    `json:"notes"`
	RelatedIdentifiers []SrcDOIRelatedIdentifier `json:"relatedIdentifiers"`
	Contributors       []SrcDOIContributor       `json:"contributors"`
	References         string                    `json:"references"`
	Version            string                    `json:"version"`
	DOI                string                    `json:"doi"`
	DOIBadge           string                    `json:"doiBadge"`
	DOILink            string                    `json:"doiLink"`
}

func migrateExpressionsDB(
	src *mongo.Database,
	dest *mongo.Database,
	userGroups map[string]string) error {
	pixlFMGroup := userGroups["PIXL-FM"]
	err := migrateExpressionsDBExpressions(src, dest, pixlFMGroup)
	if err != nil {
		return err
	}
	err = migrateExpressionsDBModules(src, dest, pixlFMGroup)
	if err != nil {
		return err
	}
	return migrateExpressionsDBModuleVersions(src, dest)
}

func migrateExpressionsDBExpressions(src *mongo.Database, dest *mongo.Database, pixlFMGroup string) error {
	destColl := dest.Collection(dbCollections.ExpressionsName)
	err := destColl.Drop(context.TODO())
	if err != nil {
		return err
	}

	filter := bson.D{}
	opts := options.Find()
	cursor, err := src.Collection("expressions").Find(context.TODO(), filter, opts)
	if err != nil {
		return err
	}

	srcExprs := []SrcDataExpression{}
	err = cursor.All(context.TODO(), &srcExprs)
	if err != nil {
		return err
	}

	destExprs := []interface{}{} //[]protos.DataExpression{}
	for _, expr := range srcExprs {
		if shouldIgnoreUser(expr.Origin.Creator.UserID) {
			fmt.Printf(" SKIPPING import of expression from user: %v aka %v\n", expr.Origin.Creator.UserID, usersIdsToIgnore[expr.Origin.Creator.UserID])
			continue
		}

		/*
			if expr.Name == "fuzzstring" ||
				expr.SourceCode == "fuzzstring" ||
				expr.SourceLanguage == "fuzzstring" ||
				expr.Comments == "fuzzstring" ||
				expr.Origin.Creator.UserID == "5e3b3bc480ee5c191714d6b7" {
				continue
			}
		*/
		tags := expr.Tags
		if tags == nil {
			tags = []string{}
		}

		// Added duplicate module checks because it appeared there's a bug in data, but turns out this importer was appending the list twice!
		// Left the checking code just in case but it shouldn't happen...
		refs := []*protos.ModuleReference{}
		existingRefs := map[string]bool{}
		existingRefModules := map[string]bool{}
		if expr.ModuleReferences != nil {
			for _, ref := range expr.ModuleReferences {
				refstr := ref.ModuleID + "_" + ref.Version
				if _, exists := existingRefs[refstr]; exists {
					fmt.Printf("  IGNORING duplicate module+version reference: %v on expression %v\n", refstr, expr.ID)
					continue
				}

				if _, exists := existingRefModules[ref.ModuleID]; exists {
					fmt.Printf("  IGNORING duplicate module reference: %v on expression %v\n", ref.ModuleID, expr.ID)
					continue
				}

				ver, err := semanticversion.SemanticVersionFromString(ref.Version)
				if err != nil {
					return err
				}

				refs = append(refs, &protos.ModuleReference{
					ModuleId: ref.ModuleID,
					Version:  ver,
				})
				existingRefs[refstr] = true
				existingRefModules[ref.ModuleID] = true
			}
		}

		destExpr := protos.DataExpression{
			Id:               expr.ID,
			Name:             expr.Name,
			SourceCode:       expr.SourceCode,
			SourceLanguage:   expr.SourceLanguage,
			Comments:         expr.Comments,
			ModifiedUnixSec:  uint32(expr.Origin.ModifiedUnixTimeSec),
			Tags:             tags,
			ModuleReferences: refs,
		}

		// If the expression is shared, we give view access to PIXL-FM group
		shareWithGroupId := ""
		if expr.Origin.Shared {
			shareWithGroupId = pixlFMGroup
		}

		err = saveOwnershipItem(destExpr.Id, protos.ObjectType_OT_EXPRESSION, expr.Origin.Creator.UserID, "", shareWithGroupId, uint32(expr.Origin.CreatedUnixTimeSec), dest)
		if err != nil {
			return err
		}

		if expr.RecentExecStats != nil {
			destExpr.RecentExecStats = &protos.DataExpressionExecStats{
				DataRequired: expr.RecentExecStats.DataRequired,
				// Old field wasn't normalised for how many PMCs we are running it for, so leave them out
				//RuntimeMs:        expr.RecentExecStats.RuntimeMS,
				TimeStampUnixSec: uint32(expr.RecentExecStats.TimeStampUnixSec),
			}
		}

		// TODO: zenodo

		destExprs = append(destExprs, destExpr)
	}

	result, err := destColl.InsertMany(context.TODO(), destExprs)
	if err != nil {
		return err
	}

	fmt.Printf("Expressions inserted: %v\n", len(result.InsertedIDs))

	return err
}

type SrcDataModule struct {
	ID       string           `json:"id" bson:"_id"` // Use as Mongo ID
	Name     string           `json:"name"`
	Comments string           `json:"comments"`
	Origin   SrcAPIObjectItem `json:"origin"`
}

func migrateExpressionsDBModules(src *mongo.Database, dest *mongo.Database, pixlFMGroup string) error {
	destColl := dest.Collection(dbCollections.ModulesName)
	err := destColl.Drop(context.TODO())
	if err != nil {
		return err
	}

	filter := bson.D{}
	opts := options.Find()
	cursor, err := src.Collection("modules").Find(context.TODO(), filter, opts)
	if err != nil {
		return err
	}

	srcModules := []SrcDataModule{}
	err = cursor.All(context.TODO(), &srcModules)
	if err != nil {
		return err
	}

	destModules := []interface{}{}
	for _, mod := range srcModules {
		destMod := protos.DataModuleDB{
			Id:              mod.ID,
			Name:            mod.Name,
			Comments:        mod.Comments,
			ModifiedUnixSec: uint32(mod.Origin.ModifiedUnixTimeSec),
		}

		// NOTE: we give viewer access to PIXL-FM group
		err = saveOwnershipItem(destMod.Id, protos.ObjectType_OT_DATA_MODULE, mod.Origin.Creator.UserID, "", pixlFMGroup, uint32(mod.Origin.CreatedUnixTimeSec), dest)
		if err != nil {
			return err
		}

		destModules = append(destModules, destMod)
	}

	result, err := destColl.InsertMany(context.TODO(), destModules)
	if err != nil {
		return err
	}

	fmt.Printf("Modules inserted: %v\n", len(result.InsertedIDs))

	return err
}

type SrcSemanticVersion struct {
	Major int
	Minor int
	Patch int
}

type SrcDataModuleVersion struct {
	ID               string             `json:"-" bson:"_id"` // Use as Mongo ID
	ModuleID         string             `json:"moduleID"`     // The ID of the module we belong to
	SourceCode       string             `json:"sourceCode"`
	Version          SrcSemanticVersion `json:"version"`
	Tags             []string           `json:"tags"`
	Comments         string             `json:"comments"`
	TimeStampUnixSec int64              `json:"mod_unix_time_sec"`
	DOIMetadata      SrcDOIMetadata     `json:"doiMetadata,omitempty" bson:"doiMetadata,omitempty"`
}

func migrateExpressionsDBModuleVersions(src *mongo.Database, dest *mongo.Database) error {
	destColl := dest.Collection(dbCollections.ModuleVersionsName)
	err := destColl.Drop(context.TODO())
	if err != nil {
		return err
	}

	filter := bson.D{}
	opts := options.Find()
	cursor, err := src.Collection("moduleVersions").Find(context.TODO(), filter, opts)
	if err != nil {
		return err
	}

	srcModuleVers := []SrcDataModuleVersion{}
	err = cursor.All(context.TODO(), &srcModuleVers)
	if err != nil {
		return err
	}

	destModuleVers := []interface{}{}
	for _, modVer := range srcModuleVers {
		tags := modVer.Tags
		if tags == nil {
			tags = []string{}
		}

		destModVer := protos.DataModuleVersionDB{
			Id:       modVer.ID,
			ModuleId: modVer.ModuleID,
			Version: &protos.SemanticVersion{
				Major: int32(modVer.Version.Major),
				Minor: int32(modVer.Version.Minor),
				Patch: int32(modVer.Version.Patch),
			},
			Tags:             tags,
			Comments:         modVer.Comments,
			TimeStampUnixSec: uint32(modVer.TimeStampUnixSec),
			SourceCode:       modVer.SourceCode,
		}

		// TODO: zenodo

		destModuleVers = append(destModuleVers, destModVer)
	}

	result, err := destColl.InsertMany(context.TODO(), destModuleVers)
	if err != nil {
		return err
	}

	fmt.Printf("Module Versions inserted: %v\n", len(result.InsertedIDs))

	return err
}
