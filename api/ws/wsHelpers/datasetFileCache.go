package wsHelpers

import (
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
type datasetCacheItem struct {
	localPath        string
	fileSize         uint64
	timestampUnixSec int64
}

var scanFileCache = map[string]datasetCacheItem{}

const maxDatasetCacheAgeSec = 60 * 5
const maxDatasetCacheSizeBytes = 200 * 1024 * 1024

func ReadDatasetFile(scanId string, svcs *services.APIServices) (*protos.Experiment, error) {
	cacheRoot := os.TempDir()

	var fileBytes []byte
	var err error
	lfs := fileaccess.FSAccess{}

	// Check if it's cached
	if item, ok := scanFileCache[scanId]; ok {
		// We have a cached file, use if not too old
		now := svcs.TimeStamper.GetTimeNowSec()

		if item.timestampUnixSec > now-maxDatasetCacheAgeSec {
			// Read the file from local cache
			fileBytes, err = lfs.ReadObject("", item.localPath)
			if err != nil {
				// Failed to read locally, delete this cache item
				svcs.Log.Errorf("Failed to read locally cached scan data for %v, path: %v, error was: %v. Download will be attempted.", scanId, item.localPath, err)
				delete(scanFileCache, scanId)
				fileBytes = nil
			}
		}
	}

	// If we don't have data by now, download it and add to our cache
	if fileBytes == nil {
		s3Path := filepaths.GetDatasetFilePath(scanId, filepaths.DatasetFileName)
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
		cachePath := filepath.Join(cacheRoot, scanId+"-dataset.bin")
		err = lfs.WriteObject("", cachePath, fileBytes)
		if err != nil {
			svcs.Log.Errorf("Failed to save scan data s3://%v/%v to local file system: %v", svcs.Config.DatasetsBucket, s3Path, err)
			// But don't die here, we can still service the request with the file bytes we downloaded
		} else {
			// Write to cache
			scanFileCache[scanId] = datasetCacheItem{
				localPath:        cachePath,
				fileSize:         uint64(len(fileBytes)),
				timestampUnixSec: svcs.TimeStamper.GetTimeNowSec(),
			}

			// Now we remove files that would make us over-extend our cache space
			removeOldDatasetCacheItems(scanFileCache)
		}
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

func orderCacheItems(cache map[string]datasetCacheItem) ([]datasetCacheItem, uint64) {
	// Calc total cache size and sort by age, oldest last
	totalSize := uint64(0)
	itemsByAge := []datasetCacheItem{}

	for _, item := range cache {
		totalSize += item.fileSize
		itemsByAge = append(itemsByAge, item)
	}

	sort.Slice(itemsByAge, func(i, j int) bool {
		return itemsByAge[i].timestampUnixSec > itemsByAge[j].timestampUnixSec
	})

	return itemsByAge, totalSize
}

func removeOldDatasetCacheItems(cache map[string]datasetCacheItem) {
	itemsByAge, totalSize := orderCacheItems(cache)
	if len(itemsByAge) <= 0 {
		return
	}

	// Loop through, oldest to newest, delete until we satisfy cache size limit
	for c := len(itemsByAge) - 1; c >= 0; c-- {
		if totalSize < maxDatasetCacheSizeBytes {
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
