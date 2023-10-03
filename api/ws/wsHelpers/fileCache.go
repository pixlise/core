package wsHelpers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/proto"
)

// This uses a cache as it may be reading the same thing many times in bursts.
// Cache is updated upon user info change though
type fileCacheItem struct {
	localPath        string
	fileSize         uint64
	timestampUnixSec int64
}

var fileCache = map[string]fileCacheItem{}

var MaxFileCacheAgeSec = int64(60 * 5)
var MaxFileCacheSizeBytes = uint64(200 * 1024 * 1024)

func ReadDatasetFile(scanId string, svcs *services.APIServices) (*protos.Experiment, error) {
	cacheId := "scan-" + scanId
	fileBytes := checkCache(cacheId, "scan", svcs)

	// If we don't have data by now, download it and add to our cache
	var err error
	if fileBytes == nil {
		s3Path := filepaths.GetDatasetFilePath(scanId, filepaths.DatasetFileName)
		fmt.Printf("Downloading file: s3://%v/%v\n", svcs.Config.DatasetsBucket, s3Path)
		fileBytes, err = svcs.FS.ReadObject(svcs.Config.DatasetsBucket, s3Path)
		if err != nil {
			// Doesn't seem to exist?
			if svcs.FS.IsNotFoundError(err) {
				return nil, errorwithstatus.MakeNotFoundError(scanId)
			}

			svcs.Log.Errorf("Failed to load scan data for %v, from: s3://%v/%v, error was: %v.", scanId, svcs.Config.DatasetsBucket, s3Path, err)
			return nil, err
		}

		// Write locally
		addToCache(cacheId, "-dataset.bin", fmt.Sprintf("s3://%v/%v", svcs.Config.DatasetsBucket, s3Path), fileBytes, svcs)
	}

	// Now decode the data & return it
	datasetPB := &protos.Experiment{}
	err = proto.Unmarshal(fileBytes, datasetPB)
	if err != nil {
		svcs.Log.Errorf("Failed to decode scan data for scan: %v. Error: %v", scanId, err)
		return nil, err
	}

	return datasetPB, nil
}

func ReadQuantificationFile(quantId string, quantPath string, svcs *services.APIServices) (*protos.Quantification, error) {
	cacheId := "quant-" + quantId
	fileBytes := checkCache(cacheId, "quant", svcs)

	// If we don't have data by now, download it and add to our cache
	var err error
	if fileBytes == nil {
		fmt.Printf("Downloading file: s3://%v/%v\n", svcs.Config.DatasetsBucket, quantPath)
		fileBytes, err = svcs.FS.ReadObject(svcs.Config.UsersBucket, quantPath)
		if err != nil {
			// Doesn't seem to exist?
			if svcs.FS.IsNotFoundError(err) {
				return nil, errorwithstatus.MakeNotFoundError(quantId)
			}

			svcs.Log.Errorf("Failed to load quant data for %v, from: s3://%v/%v, error was: %v.", quantId, svcs.Config.UsersBucket, quantPath, err)
			return nil, err
		}

		// Write locally
		addToCache(cacheId, "-quant.bin", fmt.Sprintf("s3://%v/%v", svcs.Config.UsersBucket, quantPath), fileBytes, svcs)
	}

	// Now decode the data & return it
	quantPB := &protos.Quantification{}
	err = proto.Unmarshal(fileBytes, quantPB)
	if err != nil {
		svcs.Log.Errorf("Failed to decode quant data for scan: %v. Error: %v", quantId, err)
		return nil, err
	}

	return quantPB, nil
}

func ReadDiffractionFile(scanId string, svcs *services.APIServices) (*protos.Diffraction, error) {
	cacheId := "diffraction-" + scanId
	fileBytes := checkCache(cacheId, "diffraction", svcs)

	// If we don't have data by now, download it and add to our cache
	var err error
	if fileBytes == nil {
		s3Path := filepaths.GetDatasetFilePath(scanId, filepaths.DiffractionDBFileName)
		fmt.Printf("Downloading file: s3://%v/%v\n", svcs.Config.DatasetsBucket, s3Path)
		fileBytes, err = svcs.FS.ReadObject(svcs.Config.DatasetsBucket, s3Path)
		if err != nil {
			// Doesn't seem to exist?
			if svcs.FS.IsNotFoundError(err) {
				return nil, errorwithstatus.MakeNotFoundError(scanId)
			}

			svcs.Log.Errorf("Failed to load diffraction data for %v, from: s3://%v/%v, error was: %v.", scanId, svcs.Config.DatasetsBucket, s3Path, err)
			return nil, err
		}

		// Write locally
		addToCache(cacheId, "-diffraction.bin", fmt.Sprintf("s3://%v/%v", svcs.Config.DatasetsBucket, s3Path), fileBytes, svcs)
	}

	// Now decode the data & return it
	diffPB := &protos.Diffraction{}
	err = proto.Unmarshal(fileBytes, diffPB)
	if err != nil {
		svcs.Log.Errorf("Failed to decode diffraction data for scan: %v. Error: %v", scanId, err)
		return nil, err
	}

	return diffPB, nil
}

func checkCache(id string, fileTypeName string, svcs *services.APIServices) []byte {
	var fileBytes []byte
	var err error
	lfs := fileaccess.FSAccess{}

	// Check if it's cached
	if item, ok := fileCache[id]; ok {
		// We have a cached file, use if not too old
		now := svcs.TimeStamper.GetTimeNowSec()

		if item.timestampUnixSec > now-MaxFileCacheAgeSec {
			// Read the file from local cache
			fmt.Printf("Reading local file: %v\n", item.localPath)
			fileBytes, err = lfs.ReadObject("", item.localPath)
			if err != nil {
				// Failed to read locally, delete this cache item
				svcs.Log.Errorf("Failed to read locally cached scan %v for %v, path: %v, error was: %v. Download will be attempted.", fileTypeName, id, item.localPath, err)
				delete(fileCache, id)
				fileBytes = nil
			}
		}
	}

	return fileBytes
}

func addToCache(id string, fileSuffix string, srcPath string, fileBytes []byte, svcs *services.APIServices) {
	cacheRoot := os.TempDir()
	cachePath := filepath.Join(cacheRoot, id+fileSuffix)
	lfs := fileaccess.FSAccess{}
	err := lfs.WriteObject("", cachePath, fileBytes)
	if err != nil {
		svcs.Log.Errorf("Failed to cache %v to local file system: %v", srcPath, err)
		// But don't die here, we can still service the request with the file bytes we downloaded
	} else {
		// Write to cache
		fileCache[id] = fileCacheItem{
			localPath:        cachePath,
			fileSize:         uint64(len(fileBytes)),
			timestampUnixSec: svcs.TimeStamper.GetTimeNowSec(),
		}

		// Now we remove files that would make us over-extend our cache space
		removeOldFileCacheItems(fileCache)
	}
}

func orderCacheItems(cache map[string]fileCacheItem) ([]fileCacheItem, uint64) {
	// Calc total cache size and sort by age, oldest last
	totalSize := uint64(0)
	itemsByAge := []fileCacheItem{}

	for _, item := range cache {
		totalSize += item.fileSize
		itemsByAge = append(itemsByAge, item)
	}

	sort.Slice(itemsByAge, func(i, j int) bool {
		return itemsByAge[i].timestampUnixSec > itemsByAge[j].timestampUnixSec
	})

	return itemsByAge, totalSize
}

func removeOldFileCacheItems(cache map[string]fileCacheItem) {
	itemsByAge, totalSize := orderCacheItems(cache)
	if len(itemsByAge) <= 0 {
		return
	}

	// Loop through, oldest to newest, delete until we satisfy cache size limit
	for c := len(itemsByAge) - 1; c >= 0; c-- {
		if totalSize < MaxFileCacheSizeBytes {
			// Cache is small enough now, stop here
			break
		}

		// Try delete this file
		item := itemsByAge[c]

		// Delete it
		err := os.Remove(item.localPath)
		if err != nil {
			// If that worked, remember our cache is smaller now
			totalSize -= item.fileSize

			// And remove it from cache too
			itemsByAge = append(itemsByAge[0:c], itemsByAge[c+1:]...)
		}
	}
}
