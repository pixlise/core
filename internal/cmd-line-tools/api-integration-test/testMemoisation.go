package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
)

func testMemoisation(apiHost string, jwt string) {
	testMemoisationGet_BadKey(apiHost, jwt)
	testMemoisationGet_NoKey(apiHost, jwt)
	testMemoisationWrite_NoKey(apiHost, jwt)
	testMemoisationWrite_NoData(apiHost, jwt)
	testMemoisationWrite_KeyNotMatched(apiHost, jwt)
	testMemoisationWrite_WriteReadRead(apiHost, jwt)
}

func testMemoisationGet_BadKey(apiHost string, jwt string) {
	key := utils.RandStringBytesMaskImpr(10)
	status, body, err := doHTTPRequest("http", "GET", apiHost, "memoise", "key="+key, nil, jwt)

	failIf(err != nil, err)
	failIf(string(body) != key+" not found\n" || status != 404, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))
}

func testMemoisationGet_NoKey(apiHost string, jwt string) {
	status, body, err := doHTTPRequest("http", "GET", apiHost, "memoise", "key=", nil, jwt)

	failIf(err != nil, err)
	failIf(string(body) != "Key is too short\n" || status != 400, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))
}

func testMemoisationWrite_NoKey(apiHost string, jwt string) {
	status, body, err := doHTTPRequest("http", "PUT", apiHost, "memoise", "key=", nil, jwt)

	failIf(err != nil, err)
	failIf(string(body) != "Key is too short\n" || status != 400, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))
}

func testMemoisationWrite_NoData(apiHost string, jwt string) {
	key := utils.RandStringBytesMaskImpr(10)

	status, body, err := doHTTPRequest("http", "PUT", apiHost, "memoise", "key="+key, nil, jwt)

	failIf(err != nil, err)
	failIf(string(body) != "Missing data field\n" || status != 400, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))
}

func testMemoisationWrite_KeyNotMatched(apiHost string, jwt string) {
	key := utils.RandStringBytesMaskImpr(10)

	item := &protos.MemoisedItem{
		Key:      "anotherKey",
		Data:     []byte{1, 3, 5, 7},
		ScanId:   "scan123",
		DataSize: 4,
	}

	uploadBody, err := proto.Marshal(item)
	if err != nil {
		log.Fatalln(err)
	}

	status, body, err := doHTTPRequest("http", "PUT", apiHost, "memoise", "key="+key, bytes.NewBuffer(uploadBody), jwt)

	failIf(err != nil, err)
	failIf(string(body) != "Memoisation item key doesn't match query parameter\n" || status != 400, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))
}

func testMemoisationWrite_WriteReadRead(apiHost string, jwt string) {
	key := utils.RandStringBytesMaskImpr(10)

	// Write:
	item := &protos.MemoisedItem{
		Key:      key,
		Data:     []byte{1, 3, 5, 7},
		ScanId:   "scan123",
		DataSize: 3,
	}

	uploadBody, err := proto.Marshal(item)
	if err != nil {
		log.Fatalln(err)
	}

	status, body, err := doHTTPRequest("http", "PUT", apiHost, "memoise", "key="+key, bytes.NewBuffer(uploadBody), jwt)

	failIf(err != nil, err)
	failIf(status != 200, fmt.Errorf("Unexpected memoisation response! Status %v, body: %v", status, string(body)))

	// We should have a time stamp
	ts, err := strconv.ParseInt(string(body), 10, 32)
	failIf(err != nil, err)

	failIf(ts < 1742956321, fmt.Errorf("Invalid timestamp: %v", ts))

	// Read (ensure the fields that the API should set are set - different to what we passed in above)
	status, body, err = doHTTPRequest("http", "GET", apiHost, "memoise", "key="+key, nil, jwt)

	failIf(err != nil, err)

	readItem := &protos.MemoisedItem{}
	err = proto.Unmarshal(body, readItem)
	failIf(err != nil, err)

	failIf(readItem.Key != key, errors.New("Memoisation read: key mismatch"))
	failIf(readItem.ScanId != item.ScanId, errors.New("Memoisation read: ScanId mismatch"))
	failIf(int64(readItem.MemoTimeUnixSec) != ts, errors.New("Memoisation read: MemoTimeUnixSec mismatch"))
	failIf(int64(readItem.LastReadTimeUnixSec) != ts, errors.New("Memoisation read: LastReadTimeUnixSec mismatch"))
	failIf(readItem.ExprId != "", errors.New("Memoisation read: ExprId mismatch"))
	failIf(readItem.QuantId != "", errors.New("Memoisation read: QuantId mismatch"))
	failIf(!utils.SlicesEqual(readItem.Data, item.Data), errors.New("Memoisation read: data mismatch"))
	failIf(readItem.DataSize != 4, errors.New("Memoisation read: DataSize mismatch"))

	// Wait over a second and read again - the last read timestamp should be different
	time.Sleep(time.Second * 2)

	status, body, err = doHTTPRequest("http", "GET", apiHost, "memoise", "key="+key, nil, jwt)

	failIf(err != nil, err)

	readItem = &protos.MemoisedItem{}
	err = proto.Unmarshal(body, readItem)
	failIf(err != nil, err)

	failIf(readItem.Key != key, errors.New("Memoisation read 2: key mismatch"))
	failIf(readItem.ScanId != item.ScanId, errors.New("Memoisation read 2: ScanId mismatch"))
	failIf(int64(readItem.MemoTimeUnixSec) != ts, errors.New("Memoisation read 2: MemoTimeUnixSec mismatch"))
	failIf(int64(readItem.LastReadTimeUnixSec) <= ts, errors.New("Memoisation read 2: LastReadTimeUnixSec mismatch"))
	failIf(readItem.ExprId != "", errors.New("Memoisation read 2: ExprId mismatch"))
	failIf(readItem.QuantId != "", errors.New("Memoisation read 2: QuantId mismatch"))
	failIf(!utils.SlicesEqual(readItem.Data, item.Data), errors.New("Memoisation read 2: data mismatch"))
	failIf(readItem.DataSize != 4, errors.New("Memoisation read 2: DataSize mismatch"))
}
