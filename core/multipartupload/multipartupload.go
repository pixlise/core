package multipartupload

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
)

type MultiPartUploadHandler struct {
	bucket        string
	partStorePath string

	completedFileGCTimeoutSec  uint64
	incompleteFileGCTimeoutSec uint64

	fs  fileaccess.FileAccess
	ts  timestamper.ITimeStamper
	log logger.ILogger
}

type MultiPartStatus struct {
	UploadStartTimeUnixSec uint64
	LastUploadTimeUnixSec  uint64
	CompleteTimeUnixSec    uint64
	TotalBytes             uint64
}

var statusFileName = "status.json"
var MaxPartSize = 1024 * 1024 * 21

func MakeMultiPartUploadHandler(
	bucket string,
	partStorePath string,
	completedFileGCTimeoutSec uint64,
	incompleteFileGCTimeoutSec uint64,
	fs fileaccess.FileAccess,
	ts timestamper.ITimeStamper,
	l logger.ILogger) *MultiPartUploadHandler {
	return &MultiPartUploadHandler{
		bucket:        bucket,
		partStorePath: partStorePath,

		completedFileGCTimeoutSec:  completedFileGCTimeoutSec,
		incompleteFileGCTimeoutSec: incompleteFileGCTimeoutSec,

		fs:  fs,
		ts:  ts,
		log: l,
	}
}

// Call this when starting an upload. It checks to see how many bytes we may have received already (in case it's a resume)
// otherwise returns 0
func (mp *MultiPartUploadHandler) StartUpload(fileName string, totalBytes uint64) (uint64, bool, error) {
	sz, comp, err := mp.internalStartUpload(fileName, totalBytes)
	if err != nil {
		err = fmt.Errorf("StartUpload for %v (%v bytes) error: %v", fileName, totalBytes, err)
		mp.log.Errorf("%v", err)
	} else {
		mp.log.Infof("StartUpload for %v (%v bytes): uploaded %v bytes, complete: %v", fileName, totalBytes, sz, comp)
	}
	return sz, comp, err
}

func (mp *MultiPartUploadHandler) internalStartUpload(fileName string, totalBytes uint64) (uint64, bool, error) {
	if totalBytes == 0 {
		return 0, false, fmt.Errorf("StartUpload for file %v rejected as totalBytes is 0", fileName)
	}

	// Check if status file exists
	s3Path := mp.getPartUploadPath(fileName)
	statusPath := path.Join(s3Path, statusFileName)

	status := MultiPartStatus{}
	if err := mp.fs.ReadJSON(mp.bucket, statusPath, &status, true); err != nil {
		// Something went wrong
		return 0, false, err
	}

	// If what we read is empty...
	if status.TotalBytes == 0 && status.UploadStartTimeUnixSec == 0 {
		// Treat it like this hasn't been started yet
		status = MultiPartStatus{
			UploadStartTimeUnixSec: uint64(mp.ts.GetTimeNowSec()),
			TotalBytes:             totalBytes,
		}

		return 0, false, mp.fs.WriteJSON(mp.bucket, statusPath, &status)
	}

	// Check that it's not yet complete
	if status.CompleteTimeUnixSec > 0 {
		return 0, true, nil //fmt.Errorf("Upload start for %v: already complete at %v", fileName, status.CompleteTimeUnixSec)
	}

	// It's been started. Check the total bytes match!
	if status.TotalBytes != totalBytes {
		// TODO: Clear existing attempt and start fresh??
		// Maybe this should just be a warning/log message??
		return 0, false, fmt.Errorf("Upload start failed due to total bytes mismatch. Expecting %v bytes, now starting with %v bytes", status.TotalBytes, totalBytes)
	}

	// Return how many bytes we already have
	sz, _, _, err := mp.uploadedBytesSoFar(fileName, false)
	return sz, false, err
}

func (mp *MultiPartUploadHandler) uploadedBytesSoFar(fileName string, getNamesAndSizes bool) (uint64, []string, []uint64, error) {
	names := []string{}
	sizes := []uint64{}

	// Take a listing of the uploaded data so far and find the total byte size
	s3Path := mp.getPartUploadPath(fileName)
	files, err := mp.fs.ListObjects(mp.bucket, s3Path)
	if err != nil {
		return 0, names, sizes, err
	}

	// Get the size of each file
	bytesSoFar := uint64(0)
	for _, f := range files {
		// Don't add our suffix file bytes to it!!
		if !strings.HasSuffix(f, statusFileName) {
			exists, sz, err := mp.fs.ObjectExists(mp.bucket, f)
			if err != nil {
				return 0, names, sizes, err
			}

			if !exists {
				return 0, names, sizes, fmt.Errorf("File part named %v not found", f)
			}

			if sz < 0 {
				return 0, names, sizes, fmt.Errorf("File part named %v unexpected size: %v", f, sz)
			}

			sz64 := uint64(sz)
			bytesSoFar += sz64

			// If we have names & sizes to report, do it
			if getNamesAndSizes {
				names = append(names, f)
				sizes = append(sizes, sz64)
			}
		}
	}

	return bytesSoFar, names, sizes, nil
}

func (mp *MultiPartUploadHandler) getPartUploadPath(fileName string) string {
	return path.Join(mp.partStorePath, fileName)
}

// When we receive a part of a file, this does all the processing.
// Expects:
// - File name
// - Bytes uploaded so far (previous parts, NOT including this data size)
// - Bytes of this file part
// If upload position doesn't match, we return an error
// Returns bytesReceived (uint64), isComplete (bool), error
func (mp *MultiPartUploadHandler) UploadPart(fileName string, totalUploadedBytes uint64, data []byte) (uint64, bool, error) {
	sz, comp, totalSz, err := mp.internalUploadPart(fileName, totalUploadedBytes, data)
	if err != nil {
		err = fmt.Errorf("UploadPart for %v (%v bytes so far) error: %v", fileName, totalUploadedBytes, err)
		mp.log.Errorf("%v", err)
	} else {
		mp.log.Infof("UploadPart for %v received %.1f%% %v bytes, complete: %v", fileName, 100*float32(sz)/float32(totalSz), sz, comp)
	}
	return sz, comp, err
}

func (mp *MultiPartUploadHandler) internalUploadPart(fileName string, totalUploadedBytes uint64, data []byte) (uint64, bool, uint64, error) {
	errIdentifier := fmt.Sprintf("Uploading part of file %v, total bytes so far %v", fileName, totalUploadedBytes)

	if len(data) <= 0 || len(data) >= MaxPartSize {
		// Can't upload this many bytes!
		return 0, false, 0, fmt.Errorf("%v: part size unexpected: %v", errIdentifier, len(data))
	}

	// Check that the upload has started
	s3Path := mp.getPartUploadPath(fileName)
	statusPath := path.Join(s3Path, statusFileName)

	status := MultiPartStatus{}
	if err := mp.fs.ReadJSON(mp.bucket, statusPath, &status, false); err != nil {
		// Something went wrong
		return 0, false, 0, fmt.Errorf("%v: failed to read status: %v", errIdentifier, err)
	}

	// Make sure status is valid too
	if status.TotalBytes == 0 {
		return 0, false, 0, fmt.Errorf("%v: unexpected status total bytes is 0", errIdentifier)
	}

	if status.UploadStartTimeUnixSec == 0 {
		return 0, false, 0, fmt.Errorf("%v: unexpected status start time stamp is 0", errIdentifier)
	}

	if status.CompleteTimeUnixSec > 0 {
		return 0, false, 0, fmt.Errorf("%v: unexpected status already complete at %v", errIdentifier, status.CompleteTimeUnixSec)
	}

	// Check that the total bytes uploaded so far matches what we have too
	if expectedTotalBytesSoFar, _, _, err := mp.uploadedBytesSoFar(fileName, false); err != nil {
		return 0, false, 0, fmt.Errorf("%v: Failed to read total bytes so far: %v", errIdentifier, err)
	} else if expectedTotalBytesSoFar != totalUploadedBytes {
		return 0, false, 0, fmt.Errorf("%v: Expected total bytes so far of %v, got %v", errIdentifier, expectedTotalBytesSoFar, totalUploadedBytes)
	}

	// Check if this completes the upload or not
	totalUploadedIncludingNow := totalUploadedBytes + uint64(len(data))
	if totalUploadedIncludingNow > status.TotalBytes {
		return 0, false, 0, fmt.Errorf("%v: uploaded too many bytes. Expected %v, got %v", errIdentifier, status.TotalBytes, totalUploadedIncludingNow)
	}

	// If it matches exactly, we're complete
	status.LastUploadTimeUnixSec = uint64(mp.ts.GetTimeNowSec())
	if totalUploadedIncludingNow == status.TotalBytes {
		status.CompleteTimeUnixSec = status.LastUploadTimeUnixSec
	}

	// Write status
	if err := mp.fs.WriteJSON(mp.bucket, statusPath, status); err != nil {
		return 0, false, 0, fmt.Errorf("%v: failed to store status: %v", errIdentifier, err)
	}

	// Write uploaded part to S3
	partPath := path.Join(s3Path, fmt.Sprintf("%012d.part", totalUploadedBytes))
	if err := mp.fs.WriteObject(mp.bucket, partPath, data); err != nil {
		return 0, false, 0, fmt.Errorf("%v: failed to store: %v", errIdentifier, err)
	}

	return totalUploadedIncludingNow, status.CompleteTimeUnixSec > 0, status.TotalBytes, nil
}

// TODO: Return an io.Reader? For now we download it to the local disk and return a file, but this may prove too
// prohibitive, eg what if they're uploading many files concurrently - our local disk may fill up! What if this
// is read from a separate machine though, eg a new "importing" node in docker? No issues there!
// TODO: Provide an expected checksum?
func (mp *MultiPartUploadHandler) ReadCompletedUpload(fileName string) (string, error) {
	mp.log.Infof("ReadCompletedUpload for %v...", fileName)

	f, err := mp.internalReadCompletedUpload(fileName)
	if err != nil {
		err = fmt.Errorf("ReadCompletedUpload for %v error: %v", fileName, err)
		mp.log.Errorf("%v", err)
	} else {
		mp.log.Infof("ReadCompletedUpload for %v returning file...", fileName)
	}

	return f, err
}

func (mp *MultiPartUploadHandler) internalReadCompletedUpload(fileName string) (string, error) {
	errIdentifier := fmt.Sprintf("Reading completed upload of multi-part file %v", fileName)

	// Try to read all chunks, ensure expected bytes matches
	s3Path := mp.getPartUploadPath(fileName)
	statusPath := path.Join(s3Path, statusFileName)

	status := MultiPartStatus{}
	if err := mp.fs.ReadJSON(mp.bucket, statusPath, &status, false); err != nil {
		// Something went wrong
		return "", fmt.Errorf("%v: failed to read status: %v", errIdentifier, err)
	}

	// Ensure it's complete
	if status.TotalBytes == 0 {
		return "", fmt.Errorf("%v: status total bytes was 0", errIdentifier)
	}

	if status.CompleteTimeUnixSec == 0 {
		return "", fmt.Errorf("%v: not complete", errIdentifier)
	}

	// Ensure it's the right size
	uploadedTotalBytes, listingNames, _, err := mp.uploadedBytesSoFar(fileName, true)
	if err != nil {
		return "", fmt.Errorf("%v: failed to calculate total uploaded size: %v", errIdentifier, err)
	} else if uploadedTotalBytes != status.TotalBytes {
		return "", fmt.Errorf("%v: actual uploaded size %v bytes, expected %v bytes", errIdentifier, uploadedTotalBytes, status.TotalBytes)
	}

	// Read each file from S3 and build a local file we can provide
	sort.Strings(listingNames) // Ensure they're sorted (it's just byte start, so should sort correctly this way)

	fComplete, err := os.CreateTemp("", filepath.Base(fileName)+"*")
	if err != nil {
		return "", fmt.Errorf("%v: failed to create uploaded file: %v", errIdentifier, err)
	}
	defer fComplete.Close()

	returnPath := fComplete.Name()

	// Write each piece to the file
	for i, fPart := range listingNames {
		mp.log.Infof(" Reading %v (%v/%v)...", fPart, i, len(listingNames))

		fData, err := mp.fs.ReadObject(mp.bucket, fPart)
		if err != nil {
			return "", fmt.Errorf("%v: failed to read file part: %v. Error: %v", errIdentifier, fPart, err)
		}

		_, err = fComplete.Write(fData)
		if err != nil {
			return "", fmt.Errorf("%v: failed to write file part: %v. Error: %v", errIdentifier, fPart, err)
		}
	}

	return returnPath, nil
}

// Run through old/complete uploads and delete ones we aren't using any more
func (mp *MultiPartUploadHandler) GarbageCollect() error {
	// Delete stuff from S3 that has completed a while ago or hasn't had updates in a while (time-out)
	listing, err := mp.fs.ListObjects(mp.bucket, mp.partStorePath)
	if err != nil {
		return fmt.Errorf("MultiPartFileUpload GC: Failed to list objects: %v", err)
	}

	mp.log.Infof("MultiPartFileUpload Garbage Collector running...")

	nowUnixSec := mp.ts.GetTimeNowSec()
	freed := 0

	for _, f := range listing {
		if strings.HasSuffix(f, statusFileName) {
			// It's a status file, read it and see if this can be deleted!
			status := MultiPartStatus{}
			err = mp.fs.ReadJSON(mp.bucket, f, &status, false)
			if err != nil {
				// Just log this and continue!
				mp.log.Errorf("- MultiPartFileUpload GC: Failed to read status file %v: %v", f, err)
			} else {
				// Check if it's old
				deletePath := ""
				lastTS := int64(status.LastUploadTimeUnixSec)
				if lastTS <= 0 {
					lastTS = int64(status.UploadStartTimeUnixSec)
				}

				if status.CompleteTimeUnixSec > 0 && nowUnixSec-int64(status.CompleteTimeUnixSec) > int64(mp.completedFileGCTimeoutSec) {
					deletePath = path.Dir(f)
					mp.log.Infof("- Freeing s3://%v/%v as it completed %v sec ago", mp.bucket, deletePath, nowUnixSec-int64(status.CompleteTimeUnixSec))
				} else if nowUnixSec-lastTS > int64(mp.incompleteFileGCTimeoutSec) {
					deletePath = path.Dir(f)
					mp.log.Infof("- Freeing s3://%v/%v as it was last uploaded to %v sec ago", mp.bucket, deletePath, nowUnixSec-int64(status.LastUploadTimeUnixSec))
				}

				if len(deletePath) > 0 {
					err = mp.fs.EmptyObjects(mp.bucket, deletePath)
					if err != nil {
						mp.log.Infof("MultiPartFileUpload GC: Failed to free s3://%v/%v: %v", mp.bucket, deletePath, err)
					}
					err = mp.fs.DeleteObject(mp.bucket, deletePath)
					if err != nil {
						mp.log.Infof("MultiPartFileUpload GC: Failed to free root s3://%v/%v: %v", mp.bucket, deletePath, err)
					}
					freed++
				}
			}
		}
	}

	return nil
}
