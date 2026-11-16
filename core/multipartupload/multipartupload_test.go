package multipartupload

import (
	"fmt"
	"log"
	"os"

	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
)

func Example_upload() {
	if err := os.RemoveAll("./test-bucket"); err != nil {
		log.Fatal(err)
	}

	fs := fileaccess.MakeFSAccessS3Simulator("./test-bucket")
	ts := timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{
		1793713867, 1793713868, 1793713869, 1793713870, 1793713871, 1793713872, 1793713873, 1793713895}}
	l := logger.StdOutLoggerForTest{}
	mp := MakeMultiPartUploadHandler("bucket", "part-path", 20, 10, fs, &ts, &l)

	sz, comp, err := mp.StartUpload("new.png", 10)
	fmt.Printf("start: %v|%v|%v\n", sz, comp, err)

	sz, comp, err = mp.StartUpload("new.png", 12)
	fmt.Printf("start_diff_size: %v|%v|%v\n", sz, comp, err)

	sz, comp, err = mp.StartUpload("new.png", 9)
	fmt.Printf("start_diff_size2: %v|%v|%v\n", sz, comp, err)

	completed := false
	sz, completed, err = mp.UploadPart("not-started.png", 0, []byte{1, 2})
	fmt.Printf("upload-part-fail: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 0, []byte{})
	fmt.Printf("upload-part-err: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 0, []byte{1, 2})
	fmt.Printf("upload-part: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 0, []byte{3, 4, 5})
	fmt.Printf("upload-part2-fail: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 2, []byte{3, 4, 5})
	fmt.Printf("upload-part2: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 5, []byte{6, 7, 8, 9, 10, 11, 12, 13})
	fmt.Printf("upload-part-3-toobig: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 5, []byte{6, 7, 8})
	fmt.Printf("upload-part-3: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 9, []byte{9})
	fmt.Printf("upload-part-4-fail: %v|%v|%v\n", sz, completed, err)

	sz, comp, err = mp.StartUpload("new.png", 10)
	fmt.Printf("start_resume: %v|%v|%v\n", sz, comp, err)

	sz, completed, err = mp.UploadPart("new.png", 8, []byte{9})
	fmt.Printf("upload-part-4: %v|%v|%v\n", sz, completed, err)

	sz, completed, err = mp.UploadPart("new.png", 9, []byte{10})
	fmt.Printf("upload-part-5: %v|%v|%v\n", sz, completed, err)

	sz, comp, err = mp.StartUpload("new.png", 10)
	fmt.Printf("start_resume-fail: %v|%v|%v\n", sz, comp, err)

	err = mp.GarbageCollect()
	fmt.Printf("GC: %v\n", err)

	fmt.Println(l.LastLogLine())

	err = mp.GarbageCollect()
	fmt.Printf("GC2: %v\n", err)

	fmt.Println(l.LastLogLine())

	_, err = os.Stat("./test-bucket/bucket/part-path/new.png")
	fmt.Printf("new.png stat err: %v\n", err)

	_, err = os.Stat("./test-bucket/bucket/part-path")
	fmt.Printf("part-path stat err: %v\n", err)

	// Output:
	// start: 0|false|<nil>
	// start_diff_size: 0|false|StartUpload for new.png (12 bytes) error: Upload start failed due to total bytes mismatch. Expecting 10 bytes, now starting with 12 bytes
	// start_diff_size2: 0|false|StartUpload for new.png (9 bytes) error: Upload start failed due to total bytes mismatch. Expecting 10 bytes, now starting with 9 bytes
	// upload-part-fail: 0|false|UploadPart for not-started.png (0 bytes so far) error: Uploading part of file not-started.png, total bytes so far 0: failed to read status: open ./test-bucket/bucket/part-path/not-started.png/status.json: no such file or directory
	// upload-part-err: 0|false|UploadPart for new.png (0 bytes so far) error: Uploading part of file new.png, total bytes so far 0: part size unexpected: 0
	// upload-part: 2|false|<nil>
	// upload-part2-fail: 0|false|UploadPart for new.png (0 bytes so far) error: Uploading part of file new.png, total bytes so far 0: Expected total bytes so far of 2, got 0
	// upload-part2: 5|false|<nil>
	// upload-part-3-toobig: 0|false|UploadPart for new.png (5 bytes so far) error: Uploading part of file new.png, total bytes so far 5: uploaded too many bytes. Expected 10, got 13
	// upload-part-3: 8|false|<nil>
	// upload-part-4-fail: 0|false|UploadPart for new.png (9 bytes so far) error: Uploading part of file new.png, total bytes so far 9: Expected total bytes so far of 8, got 9
	// start_resume: 8|false|<nil>
	// upload-part-4: 9|false|<nil>
	// upload-part-5: 10|true|<nil>
	// start_resume-fail: 0|true|<nil>
	// GC: <nil>
	// INFO: MultiPartFileUpload Garbage Collector running...
	// GC2: <nil>
	// INFO: - Freeing s3://bucket/part-path/new.png as it completed 23 sec ago
	// new.png stat err: stat ./test-bucket/bucket/part-path/new.png: no such file or directory
	// part-path stat err: <nil>
}
