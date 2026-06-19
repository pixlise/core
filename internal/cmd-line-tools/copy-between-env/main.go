package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pixlise/core/v4/api/dataimport/scanOwner"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	fmt.Printf("Copy between PIXLISE environments: \"%v\"...\n", services.ApiVersion)

	// Read args
	var scanId, shareWithGroup,
		srcAWSProfile, srcAWSRegion, srcMongoSecret, srcMongoDBName,
		destAWSProfile, destAWSRegion, destMongoSecret, destMongoDBName,
		srcDataBucket, srcUserBucket,
		destDataBucket, destUserBucket string

	flag.StringVar(&scanId, "scanId", "", "Scan ID to copy")
	flag.StringVar(&shareWithGroup, "shareWithGroup", "", "Group ID to share the copied scan with")

	flag.StringVar(&srcAWSProfile, "srcAWSProfile", "", "Source AWS Profile")
	flag.StringVar(&srcAWSRegion, "srcAWSRegion", "", "Source AWS Region")
	flag.StringVar(&srcMongoSecret, "srcMongoSecret", "", "Source Mongo Secret")
	flag.StringVar(&srcMongoDBName, "srcMongoDBName", "", "Source Mongo database name")
	flag.StringVar(&destAWSProfile, "destAWSProfile", "", "Destination AWS Profile")
	flag.StringVar(&destAWSRegion, "destAWSRegion", "", "Destination AWS Region")
	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination Mongo Secret")
	flag.StringVar(&destMongoDBName, "destMongoDBName", "", "Source Mongo database name")

	flag.StringVar(&srcDataBucket, "srcDataBucket", "", "Bucket to read data from")
	flag.StringVar(&srcUserBucket, "srcUserBucket", "", "Bucket to read quants from")
	flag.StringVar(&destDataBucket, "destDataBucket", "", "Bucket to write data to")
	flag.StringVar(&destUserBucket, "destUserBucket", "", "Bucket to write quants to")

	flag.Parse()

	// Check params
	flag.VisitAll(func(f *flag.Flag) {
		if f.Name != "destMongoSecret" && len(f.Value.String()) <= 0 {
			log.Fatalf("Arg: %v not set", f.Name)
		}
	})

	// Connect to the DBs and init remote FS's
	fmt.Println("Connecting to DB and AWS...")
	l := &logger.StdOutLogger{}
	srcFS, srcDB, err := getMongoAndFS(srcAWSProfile, srcAWSRegion, srcMongoSecret, srcMongoDBName, l)
	if err != nil {
		log.Fatalf("Failed to connect to source DB/AWS: %v", err)
	}

	destFS, destDB, err := getMongoAndFS(destAWSProfile, destAWSRegion, destMongoSecret, destMongoDBName, l)
	if err != nil {
		log.Fatalf("Failed to connect to destination DB/AWS: %v", err)
	}

	ctx := context.TODO()

	// Read the scan and its images
	fmt.Println("Reading source DB...")
	scanItem, err := scan.ReadScanItem(scanId, srcDB)
	if err != nil {
		log.Fatalf("Failed to read scan: %v", err)
	}
	scanDefaultImageItem, err := readScanDefaultImage(scanId, srcDB)
	if err != nil {
		log.Fatalf("Failed to read scan default image: %v", err)
	}
	tags, err := readTags(scanItem.Tags, srcDB)
	if err != nil {
		log.Fatalf("Failed to read tags %v: %v", strings.Join(scanItem.Tags, ","), err)
	}
	detectorConfig, err := piquant.GetDetectorConfig(scanItem.InstrumentConfig, srcDB)
	if err != nil {
		log.Fatalf("Failed to read detector config %v: %v", scanItem.InstrumentConfig, err)
	}
	images, err := readImages(scanId, srcDB)
	if err != nil {
		log.Fatalf("Failed to read images: %v", err)
	}
	quants, err := readQuants(scanId, srcDB)
	if err != nil {
		log.Fatalf("Failed to read images: %v", err)
	}

	fmt.Println("Reading source files...")
	scanFiles, err := srcFS.ListObjects(srcDataBucket, filepaths.GetScanFilePath(scanId, ""))
	if err != nil {
		log.Fatalf("Failed to list scan files: %v", err)
	}
	imageFiles, err := srcFS.ListObjects(srcDataBucket, path.Join(filepaths.DatasetImagesRoot, scanId))
	if err != nil {
		log.Fatalf("Failed to list image files: %v", err)
	}
	quantFiles, err := srcFS.ListObjects(srcUserBucket, filepaths.GetQuantPath(sessionuser.PIXLISESystemUserId, scanId, ""))
	if err != nil {
		log.Fatalf("Failed to list quant files: %v", err)
	}

	// List what we're reading
	fmt.Println("Dumping reads...")
	scanJ, _ := json.MarshalIndent(scanItem, "", utils.PrettyPrintIndentForJSON)
	fmt.Printf("Reading scan: %v\nDefault Image: %v\nImages:\n", string(scanJ), scanDefaultImageItem.DefaultImageFileName)
	for c, img := range images {
		fmt.Printf(" %v: %v\n", c+1, img.ImagePath)
	}

	fmt.Printf("Quantifications:\n")
	for c, q := range quants {
		fmt.Printf(" %v: %v (%v)\n", c+1, q.Params.UserParams.Name, q.Id)
	}

	fmt.Printf("Scan Files:\n")
	for c, f := range scanFiles {
		fmt.Printf(" %v: %v\n", c+1, f)
	}

	fmt.Printf("Image Files:\n")
	for c, f := range imageFiles {
		fmt.Printf(" %v: %v\n", c+1, f)
	}

	fmt.Printf("Quant Files:\n")
	for c, f := range quantFiles {
		fmt.Printf(" %v: %v\n", c+1, f)
	}

	// Verify nothing is missing
	warn := false

	// Scan Files
	for _, expfile := range []string{filepaths.DatasetFileName, filepaths.DiffractionDBFileName} {
		expPath := fmt.Sprintf("%v/%v/%v", filepaths.DatasetScansRoot, scanId, expfile)
		if !utils.ItemInSlice(expPath, scanFiles) {
			fmt.Printf("WARNING: Missing %v from expected scan files\n", expPath)
			warn = true
		}
	}

	// Images
	dbImageNames := []string{}
	for _, img := range images {
		dbImageNames = append(dbImageNames, path.Join(filepaths.DatasetImagesRoot, img.ImagePath))
	}

	if len(dbImageNames) != len(imageFiles) {
		fmt.Printf("WARNING: Number of images in DB (%v) didn't match number of image files: %v\n", len(dbImageNames), len(imageFiles))
		warn = true
	}

	for _, imgFile := range imageFiles {
		if !utils.ItemInSlice(imgFile, dbImageNames) {
			fmt.Printf("WARNING: Failed to match image file: %v to DB image list\n", imgFile)
			warn = true
		}
	}

	// Quants
	if len(quantFiles) != len(quants)*2 {
		fmt.Printf("WARNING: Number of quant files (%v) doesn't match 2x DB quants: %v\n", len(quantFiles), len(quants))
		warn = true
	}

	for _, q := range quants {
		// Check that .csv and .bin exists of this one
		found := 0
		for _, qf := range quantFiles {
			if strings.HasSuffix(qf, q.Id+".csv") || strings.HasSuffix(qf, q.Id+".bin") {
				found = found + 1
			}
		}

		if found != 2 {
			fmt.Printf("WARNING: Failed to match all files for quant %v\n", q.Id)
			warn = true
		}
	}

	if warn {
		fmt.Println("Quitting due to WARNINGS...")
		return
	}

	// Now we copy
	fmt.Println("Copying items...")

	// What ownership we're assigning to each item we're creating...
	ownership := &protos.ScanAutoShareEntry{
		Viewers: &protos.UserGroupList{UserIds: []string{}, GroupIds: []string{}},
		Editors: &protos.UserGroupList{UserIds: []string{sessionuser.PIXLISESystemUserId}, GroupIds: []string{shareWithGroup}},
	}

	// Scan:
	fmt.Println("Scan DB ownership and scan item...")
	// Write an ownership item
	err = scanOwner.WriteAutoSharedOwnership(scanId, protos.ObjectType_OT_SCAN, ownership, sessionuser.PIXLISESystemUserId, int64(scanItem.CompleteTimeStampUnixSec), destDB, l)
	if err != nil {
		log.Fatalf("Failed to write scan ownership: %v", err)
	}

	// And the scan itself
	_, err = destDB.Collection(dbCollections.ScansName).UpdateOne(ctx, bson.D{{Key: "_id", Value: scanId}}, bson.D{{Key: "$set", Value: scanItem}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatalf("Failed to write scan item: %v", err)
	}
	_, err = destDB.Collection(dbCollections.ScanDefaultImagesName).UpdateOne(ctx, bson.D{{Key: "_id", Value: scanId}}, bson.D{{Key: "$set", Value: scanDefaultImageItem}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatalf("Failed to write scan default image: %v", err)
	}

	// Tags
	for _, tag := range tags {
		_, err = destDB.Collection(dbCollections.TagsName).UpdateOne(ctx, bson.D{{Key: "_id", Value: tag.Id}}, bson.D{{Key: "$set", Value: tag}}, options.Update().SetUpsert(true))
		if err != nil {
			log.Fatalf("Failed to write tag %v: %v", tag.Id, err)
		}
	}

	// Detector config
	_, err = destDB.Collection(dbCollections.DetectorConfigsName).UpdateOne(ctx, bson.D{{Key: "_id", Value: detectorConfig.Id}}, bson.D{{Key: "$set", Value: detectorConfig}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatalf("Failed to write detector config %v: %v", detectorConfig.Id, err)
	}

	// Copy scan files
	fmt.Println("Scan files...")
	for _, scanFile := range scanFiles {
		if err = copyBetweenBuckets(srcFS, srcDataBucket, scanFile, destFS, destDataBucket, scanFile); err != nil {
			log.Fatal(err)
		}
	}

	// Quants:
	fmt.Println("Quant DB Items...")
	for _, quant := range quants {
		err = scanOwner.WriteAutoSharedOwnership(quant.Id, protos.ObjectType_OT_QUANTIFICATION, ownership, sessionuser.PIXLISESystemUserId, int64(scanItem.CompleteTimeStampUnixSec), destDB, l)
		if err != nil {
			log.Fatalf("Failed to write quant %v ownership: %v", quant.Id, err)
		}

		_, err = destDB.Collection(dbCollections.QuantificationsName).UpdateOne(ctx, bson.D{{Key: "_id", Value: quant.Id}}, bson.D{{Key: "$set", Value: quant}}, options.Update().SetUpsert(true))
		if err != nil {
			log.Fatalf("Failed to write quant %v: %v", quant.Id, err)
		}
	}

	fmt.Println("Quant files...")
	for _, quantFile := range quantFiles {
		if err = copyBetweenBuckets(srcFS, srcUserBucket, quantFile, destFS, destUserBucket, quantFile); err != nil {
			log.Fatal(err)
		}
	}

	// Images:
	fmt.Println("Image DB items...")
	for _, imgItem := range images {
		// Images back onto scan ownership so we don't write a new one out!
		_, err = destDB.Collection(dbCollections.ImagesName).UpdateOne(ctx, bson.D{{Key: "_id", Value: imgItem.ImagePath}}, bson.D{{Key: "$set", Value: imgItem}}, options.Update().SetUpsert(true))
		if err != nil {
			log.Fatalf("Failed to write image %v: %v", imgItem.ImagePath, err)
		}
	}

	fmt.Println("Images Files...")
	for _, imgFile := range imageFiles {
		fmt.Printf("  %v file copy...\n", imgFile)
		if err = copyBetweenBuckets(srcFS, srcDataBucket, imgFile, destFS, destDataBucket, imgFile); err != nil {
			log.Fatal(err)
		}
	}

	// Other things to copy:
	// - ScanDefaultImages
	// - Tags (so our tag ids arent dangling references)
	// DetectorConfigs
	// Workspaces involving the scan?? That ends up needing expressions, expression groups, rois, etc
}

func copyBetweenBuckets(
	srcFS fileaccess.FileAccess, srcBucket, srcPath string,
	destFS fileaccess.FileAccess, destBucket, destPath string) error {
	d, err := srcFS.ReadObject(srcBucket, srcPath)
	if err != nil {
		return fmt.Errorf("Failed to read file %v: %v", srcPath, err)
	}

	err = destFS.WriteObject(destBucket, destPath, d)
	if err != nil {
		return fmt.Errorf("Failed to write file %v: %v", destPath, err)
	}
	return nil
}

func readScanDefaultImage(scanId string, db *mongo.Database) (*protos.ScanImageDefaultDB, error) {
	ctx := context.TODO()
	result := db.Collection(dbCollections.ScanDefaultImagesName).FindOne(ctx, bson.M{"_id": scanId})
	if result.Err() != nil {
		return nil, result.Err()
	}

	item := &protos.ScanImageDefaultDB{}
	err := result.Decode(item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func readTags(tagIds []string, db *mongo.Database) ([]*protos.TagDB, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.TagsName)

	filter := bson.M{"_id": bson.M{"$in": tagIds}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	tags := []*protos.TagDB{}
	err = cursor.All(context.TODO(), &tags)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func readImages(scanId string, db *mongo.Database) ([]*protos.ScanImage, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagesName)

	//filter := bson.M{"originscanid": scanId}
	//filter := bson.M{"_id": bson.M{"$in": []string{scanId}}}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$regex", Value: fmt.Sprintf("%v/.*", scanId)}}}}
	cursor, err := coll.Find(ctx, filter)

	if err != nil {
		return nil, err
	}

	scanImages := []*protos.ScanImage{}
	err = cursor.All(context.TODO(), &scanImages)
	if err != nil {
		return nil, err
	}

	return scanImages, nil
}

func readQuants(scanId string, db *mongo.Database) ([]*protos.QuantificationSummary, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.QuantificationsName)

	filter := bson.M{"scanid": scanId, "params.requestoruserid": sessionuser.PIXLISESystemUserId}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := []*protos.QuantificationSummary{}
	err = cursor.All(ctx, &items)
	return items, err
}

func getMongoAndFS(awsProfile string, awsRegion string, mongoSecretName string, mongoDatabaseName string, l logger.ILogger) (fileaccess.FileAccess, *mongo.Database, error) {
	sess, err := session.NewSessionWithOptions(
		session.Options{
			Profile: awsProfile,
			Config: aws.Config{
				Region: aws.String(awsRegion),
			},
		},
	)
	if err != nil {
		log.Fatalln(err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalln(err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

	db, _, err := mongoDBConnection.ConnectToMongo(sess, mongoSecretName, l, false)
	if err != nil {
		return nil, nil, err
	}

	client := db.Database(mongoDatabaseName)
	return fs, client, nil
}
