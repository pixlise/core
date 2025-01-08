package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var t0 = time.Now().UnixMilli()

var destMongoSecret string
var dbName string

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're doing the cleanup in")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		dbName,
	}
	checkNotEmptyName := []string{
		"dbName",
	}

	for c, s := range checkNotEmpty {
		if len(s) <= 0 {
			log.Fatalf("Parameter: %v was empty", checkNotEmptyName[c])
		}
	}

	// Get a session for the bucket region
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS session. Error: %v", err)
	}

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to mongo
	destMongoClient, _, err := mongoDBConnection.Connect(sess, destMongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	// Destination DB is the new pixlise one
	db := destMongoClient.Database(dbName) //mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	listMultiVersionImages(db, iLog)

	printFinishStats()
}

func readScans(db *mongo.Database) map[string]*protos.ScanItem {
	// Read all scans
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ScansName)
	cursor, err := coll.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		fatalError(err)
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(ctx, &scans)
	if err != nil {
		fatalError(err)
	}

	scanMap := map[string]*protos.ScanItem{}
	for _, scan := range scans {
		if _, ok := scanMap[scan.Id]; ok {
			fatalError(errors.New("Duplicate scan id: " + scan.Id))
		}
		scanMap[scan.Id] = scan
	}

	return scanMap
}

func readImages(db *mongo.Database) map[string]*protos.ScanImage {
	// Read all images
	ctx := context.TODO()
	opts := options.Find() /*.SetProjection(bson.D{
		{Key: "id", Value: true},
		{Key: "originscanid", Value: true},
	})*/
	coll := db.Collection(dbCollections.ImagesName)
	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		fatalError(err)
	}

	images := []*protos.ScanImage{}
	err = cursor.All(ctx, &images)
	if err != nil {
		fatalError(err)
	}

	imageLookup := map[string]*protos.ScanImage{}
	for i, img := range images {
		if _, ok := imageLookup[img.ImagePath]; ok {
			fmt.Printf("Duplicate image: %v, %v\n", i, img.ImagePath)
			continue
		}

		imageLookup[img.ImagePath] = img
	}

	fmt.Printf("Total images: %v\n", len(imageLookup))
	return imageLookup
}

func readAndVerifyImageBeams(db *mongo.Database) map[string]*protos.ImageLocations {
	// Read all beam locations
	ctx := context.TODO()
	opts := options.Find().SetProjection(bson.D{
		{Key: "id", Value: true},
		{Key: "locationperscan.scanid", Value: true},
		{Key: "locationperscan.beamversion", Value: true},
		{Key: "locationperscan.instrument", Value: true},
	})
	coll := db.Collection(dbCollections.ImageBeamLocationsName)
	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		fatalError(err)
	}

	beamLocs := []*protos.ImageLocations{}
	err = cursor.All(ctx, &beamLocs)
	if err != nil {
		fatalError(err)
	}

	fmt.Printf("Total beam locations: %v\n", len(beamLocs))
	beamLocLookup := map[string]*protos.ImageLocations{}
	for _, loc := range beamLocs {
		l := len(path.Dir(loc.ImageName))
		if l < 0 {
			fmt.Printf("No scan id in image beam location key: %v\n", loc.ImageName)
			continue
		}

		if _, ok := beamLocLookup[loc.ImageName]; ok {
			fmt.Printf("Duplicate beam loc entry for image: %v\n", loc.ImageName)
			continue
		}

		// Sanity check
		if len(loc.LocationPerScan) < 1 {
			fmt.Printf("No LocationPerScan in beam location for image: %v\n", loc.ImageName)
			continue
		}

		seenVersions := map[uint32]bool{}
		for _, locForScan := range loc.LocationPerScan {
			if locForScan.BeamVersion < 1 || locForScan.BeamVersion > 3 {
				fmt.Printf("Invalid beam version %v for image: %v\n", locForScan.BeamVersion, loc.ImageName)
				continue
			}

			if _, ok := seenVersions[locForScan.BeamVersion]; ok {
				fmt.Printf("Duplicate version %v beam loc entry for image: %v\n", locForScan.BeamVersion, loc.ImageName)
				continue
			}
			seenVersions[locForScan.BeamVersion] = true

			// Ensure scan id is same as image name prefix
			imgPathPrefix := loc.ImageName[0:l]
			if locForScan.ScanId != imgPathPrefix {
				fmt.Printf("Beam loc entry for image: %v, version: %v has invalid scan id: %v\n", loc.ImageName, locForScan.BeamVersion, locForScan.ScanId)
				continue
			}
		}

		beamLocLookup[loc.ImageName] = loc
	}

	return beamLocLookup
}

func collapseImageVersions(imageLookup map[string]*protos.ScanImage, scanMap map[string]*protos.ScanItem) map[string]map[string][]*protos.ScanImage {
	// Find all images that are the same version and group them together
	imageSansVersion := map[string]map[string][]*protos.ScanImage{}

	imagesWithBadOriginScanId := 0

	for _, img := range imageLookup {
		len := len(filepath.Dir(img.ImagePath))
		if len < 0 {
			//fmt.Printf("Ignoring (no scan prefix in image path): %v\n", img.ImagePath)
		} else {
			imgPathPrefix := img.ImagePath[0:len]

			scan, ok := scanMap[imgPathPrefix]
			if !ok {
				fmt.Printf("  No scan for image path prefix: %v, imagePath: %v\n", imgPathPrefix, img.ImagePath)
				continue
			}

			if scan.Instrument != protos.ScanInstrument_PIXL_FM {
				//fmt.Printf("Ignoring (not PIXL FM): %v\n", img.ImagePath)
				continue
			}

			meta, err := gdsfilename.ParseFileName(img.ImagePath)
			if err != nil {
				//fmt.Printf("Ignoring (non-FM file name): %v\n", img.ImagePath)
			} else {
				// Check if it's a VIS or MSA image, which we ignore here
				if meta.ProdType == "VIS" || meta.ProdType == "MSA" {
					continue
				}

				// Ensure it matches the origin scan id
				if img.OriginScanId != imgPathPrefix {
					//fatalError(fmt.Errorf("Image origin scan doesnt match stored value: %v vs %v", img.ImagePath, img.OriginScanId))
					fmt.Printf("  Image origin scan doesnt match stored value: \"%v\" vs \"%v\"\n", img.ImagePath, img.OriginScanId)
					imagesWithBadOriginScanId++
					continue
				}

				// Clear the version out and store (or add to stored one)
				meta.SetVersionStr("__")

				// First check we have an entry for this scan
				if _, ok := imageSansVersion[img.OriginScanId]; !ok {
					imageSansVersion[img.OriginScanId] = map[string][]*protos.ScanImage{}
				}

				// Add this image if we dont have it
				sansVer := filepath.Join(img.OriginScanId, meta.ToString(false, false))

				if _, ok := imageSansVersion[img.OriginScanId][sansVer]; !ok {
					imageSansVersion[img.OriginScanId][sansVer] = []*protos.ScanImage{img}
				} else {
					imageSansVersion[img.OriginScanId][sansVer] = append(imageSansVersion[img.OriginScanId][sansVer], img)
				}
			}
		}
	}

	fmt.Printf("Images with bad origin scan id: %v\n", imagesWithBadOriginScanId)

	return imageSansVersion
}

func deleteImagesWithSingleVersions(imageSansVersion map[string]map[string][]*protos.ScanImage) {
	// Drop all ones with just one image in it
	for scanId, imgs := range imageSansVersion {
		for k, imgVers := range imgs {
			if len(imgVers) < 1 {
				fatalError(fmt.Errorf("Scan %v Image %v has no entries", scanId, k))
			}

			if len(imgVers) == 1 {
				//fmt.Printf("Ignoring (only one image): %v\n", k)
				delete(imgs, k)

				if len(imgs) <= 0 {
					delete(imageSansVersion, scanId)
				}
			}
		}
	}
}

func printImageVersionBeamVersionList(scanIds []string, imageSansVersion map[string]map[string][]*protos.ScanImage, beamLocLookup map[string]*protos.ImageLocations) {
	for _, scanId := range scanIds {
		imgs := imageSansVersion[scanId]
		fmt.Printf("Scan: %v\n", scanId)

		for k, imgVers := range imgs {
			versions := []int{}
			versionedNames := map[int]string{}
			beamsVersionsForImageVersions := map[int][]uint32{}
			for _, v := range imgVers {
				meta, err := gdsfilename.ParseFileName(v.ImagePath)
				if err != nil {
					fatalError(fmt.Errorf("%v: %v", v.ImagePath, err))
				}

				vNum, err := meta.Version()
				if err != nil {
					fatalError(fmt.Errorf("%v: %v", v.ImagePath, err))
				}

				versions = append(versions, int(vNum))
				versionedNames[int(vNum)] = v.ImagePath

				// Look up what beam location versions are available
				beamLocs := beamLocLookup[v.ImagePath]
				if beamLocs != nil {
					if _, ok := beamsVersionsForImageVersions[int(vNum)]; !ok {
						beamsVersionsForImageVersions[int(vNum)] = []uint32{}
					}

					for _, beamLoc := range beamLocs.LocationPerScan {
						beamsVersionsForImageVersions[int(vNum)] = append(beamsVersionsForImageVersions[int(vNum)], beamLoc.BeamVersion)
					}
				}
			}

			fmt.Printf("  Image %v:\n", k)

			slices.Sort(versions)
			for _, v := range versions {
				fmt.Printf("   %v: %v\n", v, versionedNames[v])
				if beamsVersionsForImageVersions[v] == nil {
					fmt.Printf("   NO BEAM VERSIONS!\n")
				} else {
					vers := []string{}
					for _, bVer := range beamsVersionsForImageVersions[v] {
						vers = append(vers, fmt.Sprintf("%v", bVer))
					}
					fmt.Printf("      Beam versions: [%v]\n", strings.Join(vers, ", "))
				}
			}
		}
	}
}

func listMultiVersionImages(db *mongo.Database, iLog logger.ILogger) {
	scanMap := readScans(db)
	//scanIds := makeSortedScanIds(scanMap)

	imageLookup := readImages(db)
	beamLocLookup := readAndVerifyImageBeams(db)

	imageSansVersion := collapseImageVersions(imageLookup, scanMap)
	//deleteImagesWithSingleVersions(imageSansVersion)
	scanIds := utils.GetMapKeys(imageSansVersion)
	sort.Strings(scanIds)

	fmt.Printf("Listing %v scans containing images with multiple versions present\n", len(imageSansVersion))

	printImageVersionBeamVersionList(scanIds, imageSansVersion, beamLocLookup)

	// Pull together all images
	scanImgs := []*protos.ScanImage{}
	for _, imgs := range imageSansVersion["284951045"] {
		scanImgs = append(scanImgs, imgs...)
	}

	latestScanImgs, err := wsHelpers.GetLatestImagesOnly(scanImgs)
	if err != nil {
		fatalError(err)
	}

	fmt.Printf("%v\n", latestScanImgs)
}

func fatalError(err error) {
	printFinishStats()
	log.Fatal(err)
}

func printFinishStats() {
	t1 := time.Now().UnixMilli()
	sec := (t1 - t0) / 1000
	fmt.Printf("Runtime %v seconds\n", sec)
}
