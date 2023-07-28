package main

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
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

func migrateExpressionsDB(src *mongo.Database, dest *mongo.Database) error {
	err := migrateExpressionsDBExpressions(src, dest)
	if err != nil {
		return err
	}
	err = migrateExpressionsDBModules(src, dest)
	if err != nil {
		return err
	}
	return migrateExpressionsDBModuleVersions(src, dest)
}

func migrateExpressionsDBExpressions(src *mongo.Database, dest *mongo.Database) error {
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
			fmt.Printf("Skipping import of expression from user: %v aka %v\n", expr.Origin.Creator.UserID, usersIdsToIgnore[expr.Origin.Creator.UserID])
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
		destExpr := protos.DataExpression{
			Id:             expr.ID,
			Name:           expr.Name,
			SourceCode:     expr.SourceCode,
			SourceLanguage: expr.SourceLanguage,
			Comments:       expr.Comments,
			Tags:           tags,
		}

		err = saveOwnershipItem(destExpr.Id, protos.ObjectType_OT_ROI, expr.Origin.Creator.UserID, uint32(expr.Origin.CreatedUnixTimeSec), dest)
		if err != nil {
			return err
		}

		if expr.RecentExecStats != nil {
			destExpr.RecentExecStats = &protos.DataExpressionExecStats{
				DataRequired:     expr.RecentExecStats.DataRequired,
				RuntimeMs:        expr.RecentExecStats.RuntimeMS,
				TimeStampUnixSec: uint32(expr.RecentExecStats.RuntimeMS),
			}
		}

		for _, modRef := range expr.ModuleReferences {
			destExpr.ModuleReferences = append(destExpr.ModuleReferences, &protos.ModuleReference{
				ModuleId: modRef.ModuleID,
				Version:  modRef.Version,
			})
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

func migrateExpressionsDBModules(src *mongo.Database, dest *mongo.Database) error {
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
		destMod := protos.DataModule{
			Id:       mod.ID,
			Name:     mod.Name,
			Comments: mod.Comments,
		}

		err = saveOwnershipItem(destMod.Id, protos.ObjectType_OT_ROI, mod.Origin.Creator.UserID, uint32(mod.Origin.CreatedUnixTimeSec), dest)
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

		destModVer := protos.DataModuleVersion{
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
