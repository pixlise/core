package wsHandler

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/semanticversion"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func HandleDataModuleGetReq(req *protos.DataModuleGetReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.DataModuleDB](false, req.Id, protos.ObjectType_OT_DATA_MODULE, dbCollections.ModulesName, hctx)
	if err != nil {
		return nil, err
	}

	module := &protos.DataModule{
		Id:              dbItem.Id,
		Name:            dbItem.Name,
		Comments:        dbItem.Comments,
		ModifiedUnixSec: dbItem.ModifiedUnixSec,
	}

	module.Creator = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	// Get the version requested...
	verRequested := req.Version

	if verRequested == nil {
		// Version is not supplied, get latest
		verRequested, err = getLatestModuleVersion(req.Id, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, err
		}
	}

	// Get this specific version
	moduleVersion, err := getModuleVersion(req.Id, verRequested, hctx.Svcs.MongoDB)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Id + ", version: " + semanticversion.SemanticVersionToString(verRequested))
		}
		return nil, fmt.Errorf("Failed to get version: %v for module: %v. Error: %v", semanticversion.SemanticVersionToString(verRequested), req.Id, err)
	}

	versions, err := getModuleVersions(req.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// Add all previous versions
	module.Versions = versions
	// Find the requested version and replace it with the one we got if it exists
	fetchedSemanticVersion := semanticversion.SemanticVersionToString(moduleVersion.Version)

	replacedFetchedVersion := false
	for i, ver := range module.Versions {
		if semanticversion.SemanticVersionToString(ver.Version) == fetchedSemanticVersion {
			module.Versions[i] = moduleVersion
			replacedFetchedVersion = true
			break
		}
	}

	// If we didn't find the version we fetched, add it to the end
	if !replacedFetchedVersion {
		module.Versions = append(module.Versions, moduleVersion)
	}

	return &protos.DataModuleGetResp{
		Module: module,
	}, nil
}

func HandleDataModuleListReq(req *protos.DataModuleListReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_DATA_MODULE, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ModulesName)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.DataModuleDB{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	// And for each module, we list all versions. Note, that we're returning a map of modules by module ID
	itemMap := map[string]*protos.DataModule{}
	for _, item := range items {
		versions, err := getModuleVersions(item.Id, hctx.Svcs.MongoDB)

		if err != nil {
			return nil, fmt.Errorf("Failed to query versions for module %v. Error: %v", item.Id, err)
		}

		// If we didn't get any versions returned, this is an error!
		if len(versions) <= 0 {
			return nil, fmt.Errorf("No versions for module %v", item.Id)
		}

		// Deep copy :(
		itemWire := &protos.DataModule{
			Id:              item.Id,
			Name:            item.Name,
			Comments:        item.Comments,
			ModifiedUnixSec: item.ModifiedUnixSec,
			Versions:        versions,
		}

		if owner, ok := idToOwner[item.Id]; ok {
			itemWire.Creator = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}

		itemMap[item.Id] = itemWire
	}

	return &protos.DataModuleListResp{
		Modules: itemMap,
	}, nil
}

func getModuleVersion(moduleID string, version *protos.SemanticVersion, db *mongo.Database) (*protos.DataModuleVersion, error) {
	// NOTE: This was initially built with a query:
	// filter := bson.D{primitive.E{Key: "moduleid", Value: moduleID}, primitive.E{Key: "version", Value: version}}
	// But now ID is composed of these fields so it's more direct to query by ID
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ModuleVersionsName)

	result := &protos.DataModuleVersion{}
	id := moduleID + "-v" + semanticversion.SemanticVersionToString(version)
	verResult := coll.FindOne(ctx, bson.M{"_id": id})

	if verResult.Err() != nil {
		return nil, verResult.Err()
	}

	// Read the module item
	err := verResult.Decode(&result)
	return result, err
}

func getLatestModuleVersion(moduleID string, db *mongo.Database) (*protos.SemanticVersion, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ModuleVersionsName)
	cursor, err := coll.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "moduleid", Value: moduleID}}}},
		bson.D{
			{Key: "$sort",
				Value: bson.D{
					{Key: "version.major", Value: -1},
					{Key: "version.minor", Value: -1},
					{Key: "version.patch", Value: -1},
				},
			},
		},
		bson.D{{Key: "$limit", Value: 1}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "version", Value: 1}}}},
	})

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)
	ver := &protos.DataModuleVersion{}
	for cursor.Next(ctx) {
		err = cursor.Decode(ver)
		break
	}

	return ver.Version, err
}

func getModuleVersions(moduleID string, db *mongo.Database) ([]*protos.DataModuleVersion, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ModuleVersionsName)

	filter := bson.D{primitive.E{Key: "moduleid", Value: moduleID}}
	opts := options.Find().SetProjection(bson.D{
		{Key: "version", Value: true},
		{Key: "tags", Value: true},
		{Key: "comments", Value: true},
		{Key: "timestampunixsec", Value: true},
	})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	allVersions := []*protos.DataModuleVersionDB{}
	err = cursor.All(ctx, &allVersions)
	if err != nil {
		return nil, err
	}

	result := []*protos.DataModuleVersion{}

	for _, versionDB := range allVersions {
		ver := &protos.DataModuleVersion{
			Version:          versionDB.Version,
			Tags:             versionDB.Tags,
			Comments:         versionDB.Comments,
			TimeStampUnixSec: versionDB.TimeStampUnixSec,
		}
		result = append(result, ver)
	}
	return result, nil
}

// Just validates the basics around a module, can be called from create or update
func validateModule(name string, comments string) error {
	if !isValidModuleName(name) {
		return fmt.Errorf("Invalid module name: %v", name)
	}

	if err := wsHelpers.CheckStringField(&comments, "Comments", 0, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return err
	}
	return nil
}

func createModule(name string, comments string, intialSourceCode string, tags []string, hctx wsHelpers.HandlerContext) (*protos.DataModule, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateModule(name, comments)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	modId := hctx.Svcs.IDGen.GenObjectID()

	module := &protos.DataModuleDB{
		Id:       modId,
		Name:     name,
		Comments: comments,
	}

	// We need to create an ownership item along with it
	ownerItem, err := wsHelpers.MakeOwnerForWrite(modId, protos.ObjectType_OT_DATA_MODULE, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())
	if err != nil {
		return nil, err
	}

	module.ModifiedUnixSec = ownerItem.CreatedUnixSec

	// Create the initial version
	saveVer := &protos.SemanticVersion{Major: 0, Minor: 0, Patch: 1}
	verId := modId + "-v" + semanticversion.SemanticVersionToString(saveVer)
	version := &protos.DataModuleVersionDB{
		Id:               verId,
		ModuleId:         modId,
		Version:          saveVer,
		SourceCode:       intialSourceCode,
		Comments:         comments,
		Tags:             tags,
		TimeStampUnixSec: ownerItem.CreatedUnixSec,
		// TODO: doi metadata ??
	}

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.ModulesName).InsertOne(sessCtx, module)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.ModuleVersionsName).InsertOne(sessCtx, version)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem)
		if _err != nil {
			return nil, _err
		}
		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)

	if err != nil {
		return nil, err
	}

	// Make the return struct
	moduleWire := &protos.DataModule{
		Id:              module.Id,
		Name:            module.Name,
		Comments:        module.Comments,
		Creator:         wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper),
		ModifiedUnixSec: module.ModifiedUnixSec,
		Versions: []*protos.DataModuleVersion{
			{
				Version:          version.Version,
				Tags:             version.Tags,
				Comments:         version.Comments,
				TimeStampUnixSec: version.TimeStampUnixSec,
				SourceCode:       version.SourceCode,
			},
		},
	}

	return moduleWire, nil
}

func updateModule(id string, name string, comments string, hctx wsHelpers.HandlerContext) (*protos.DataModule, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.DataModuleDB](true, id, protos.ObjectType_OT_DATA_MODULE, dbCollections.ModulesName, hctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	update := bson.D{}
	if len(name) > 0 {
		dbItem.Name = name
		update = append(update, bson.E{Key: "name", Value: name})
	}

	if len(comments) > 0 {
		dbItem.Comments = comments
		update = append(update, bson.E{Key: "comments", Value: comments})
	}

	// Validate it
	err = validateModule(dbItem.Name, dbItem.Comments)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Update modified time
	dbItem.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update = append(update, bson.E{Key: "modifiedunixsec", Value: dbItem.ModifiedUnixSec})

	// It's valid, update the DB
	dbResult, err := hctx.Svcs.MongoDB.Collection(dbCollections.ModulesName).UpdateByID(ctx, id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if dbResult.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("DataModule UpdateByID result had unexpected counts %+v id: %v", dbResult, id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	result := &protos.DataModule{
		Id:              dbItem.Id,
		Name:            dbItem.Name,
		Comments:        dbItem.Comments,
		ModifiedUnixSec: dbItem.ModifiedUnixSec,
		Creator:         wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper),
	}

	return result, nil
}

func HandleDataModuleWriteReq(req *protos.DataModuleWriteReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleWriteResp, error) {
	var item *protos.DataModule
	var err error

	if len(req.Id) <= 0 {
		item, err = createModule(req.Name, req.Comments, req.InitialSourceCode, req.InitialTags, hctx)
	} else {
		// Ensure no source code is set
		if len(req.InitialSourceCode) > 0 {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("InitialSourceCode must not be set for module updates, only name and comments allowed to change"))
		}
		item, err = updateModule(req.Id, req.Name, req.Comments, hctx)
	}
	if err != nil {
		return nil, err
	}

	return &protos.DataModuleWriteResp{
		Module: item,
	}, nil
}

// Some validation functions
func isValidModuleName(name string) bool {
	// Limit length to something reasonable (this will be used in code so we don't want huge module names)
	if len(name) > 20 {
		return false
	}

	// Names must be valid Lua variable names...
	match, err := regexp.MatchString("^[A-Za-z]$|^[A-Za-z_]+[A-Za-z0-9_]*[A-Za-z0-9]$", name)
	if err != nil {
		return false
	}
	return match
}

func HandleDataModuleAddVersionReq(req *protos.DataModuleAddVersionReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleAddVersionResp, error) {
	// Check that the version update field is a valid value
	if !utils.ItemInSlice(req.VersionUpdate, []protos.VersionField{protos.VersionField_MV_MAJOR, protos.VersionField_MV_MINOR, protos.VersionField_MV_PATCH}) {
		return nil, fmt.Errorf("Invalid version update field: %v", req.VersionUpdate)
	}

	// Validate the rest
	if err := wsHelpers.CheckStringField(&req.ModuleId, "ModuleId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.SourceCode, "SourceCode", 1, wsHelpers.SourceCodeMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Comments, "Comments", 0, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckFieldLength(req.Tags, "Tags", 0, wsHelpers.TagListMaxLength); err != nil {
		return nil, err
	}

	// Check that the module exists
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.DataModuleDB](false, req.ModuleId, protos.ObjectType_OT_DATA_MODULE, dbCollections.ModulesName, hctx)
	if err != nil {
		return nil, err
	}

	module := &protos.DataModule{
		Id:              dbItem.Id,
		Name:            dbItem.Name,
		Comments:        dbItem.Comments,
		ModifiedUnixSec: dbItem.ModifiedUnixSec,
	}

	module.Creator = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	ver, err := getLatestModuleVersion(req.ModuleId, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// Increment the version as needed
	if req.VersionUpdate == protos.VersionField_MV_MAJOR {
		ver.Major++
		ver.Minor = 0
		ver.Patch = 0
	} else if req.VersionUpdate == protos.VersionField_MV_MINOR {
		ver.Minor++
		ver.Patch = 0
	} else {
		ver.Patch++
	}

	// Write out the new version
	verId := req.ModuleId + "-v" + semanticversion.SemanticVersionToString(ver)
	nowUnix := hctx.Svcs.TimeStamper.GetTimeNowSec()
	verRec := &protos.DataModuleVersionDB{
		Id:               verId,
		ModuleId:         req.ModuleId,
		SourceCode:       req.SourceCode,
		Version:          ver,
		Tags:             req.Tags,
		Comments:         req.Comments,
		TimeStampUnixSec: uint32(nowUnix),
		//DOIMetadata:      input.DOIMetadata,
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ModuleVersionsName)

	insertResult, err := coll.InsertOne(ctx, verRec)
	if err != nil {
		return nil, err
	}
	if insertResult.InsertedID != verId {
		hctx.Svcs.Log.Errorf("CreateModule (version): Expected Mongo insert to return ID %v, got %v", verId, insertResult.InsertedID)
	}

	// Add all previous versions
	versions, err := getModuleVersions(req.ModuleId, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	module.Versions = versions

	// Find the requested version and replace it with the one we got if it exists
	fetchedSemanticVersion := semanticversion.SemanticVersionToString(verRec.Version)

	returnVersion := &protos.DataModuleVersion{
		Version:          verRec.Version,
		Tags:             verRec.Tags,
		Comments:         verRec.Comments,
		TimeStampUnixSec: verRec.TimeStampUnixSec,
		SourceCode:       verRec.SourceCode,
	}

	replacedFetchedVersion := false
	for i, ver := range module.Versions {
		if semanticversion.SemanticVersionToString(ver.Version) == fetchedSemanticVersion {
			module.Versions[i] = returnVersion
			replacedFetchedVersion = true
			break
		}
	}

	// If we didn't find the version we fetched, add it to the end
	if !replacedFetchedVersion {
		module.Versions = append(module.Versions, returnVersion)
	}

	return &protos.DataModuleAddVersionResp{
		Module: module,
	}, nil
}
