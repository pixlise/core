package main

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Based on: https://fluhus.github.io/snopher/
// and: https://medium.com/analytics-vidhya/running-go-code-from-python-a65b3ae34a2d

// NOTE: the comment before import "C" can contain code such as helpers to be included in the generated C directly

/*
#include <stdlib.h>
#include <stdint.h>

typedef void* (*alloc_f)(char* t, int64_t n);
static void* call_alloc_f(alloc_f f, char* t, int64_t n) {return f(t,n);}
*/
import "C"

///////////////////////////////////////////////////////////////////////
// Allocation of memory in Python for use in Go
///////////////////////////////////////////////////////////////////////

// Calls the Python alloc callback and returns the allocated buffer
// as a slice.
// Type codes: https://docs.python.org/3/library/array.html
func allocSlice[T any](alloc C.alloc_f, n int, typeCode string) []T {
	t := C.CString(typeCode)                      // Make a c-string type code.
	ptr := C.call_alloc_f(alloc, t, C.int64_t(n)) // Allocate the buffer.
	C.free(unsafe.Pointer(t))                     // Release c-string.
	return unsafe.Slice((*T)(ptr), n)             // Wrap with a go-slice.
}

// Some convenience functions using the above

// BROKEN: Has some kind of alignment problem. Allocates, but when written to in
//         go it seems to still be treated as int64s or something? Or is this in
//         python - it's making all ints 64bit?
// func allocInt32s(alloc C.alloc_f, n int) []int32 {
// 	return allocSlice[int32](alloc, n, "l")
// }

func allocInts(alloc C.alloc_f, n int) []int64 {
	return allocSlice[int64](alloc, n, "q")
}

func allocBytes(alloc C.alloc_f, n int) []byte {
	return allocSlice[byte](alloc, n, "B")
}

func allocString(alloc C.alloc_f, s string) {
	b := allocBytes(alloc, len(s))
	copy(b, s)
}

///////////////////////////////////////////////////////////////////////

var apiClient *client.APIClient
var emptyCString = C.CString("")
var allocFn C.alloc_f

//export authenticate
func authenticate(allocFunc C.alloc_f) *C.char {
	var err error

	// Try to load the config file
	apiClient, err = client.Authenticate()
	if err != nil {
		return C.CString(fmt.Sprintf("Authentication error: %v", err))
	}

	fmt.Println("Authenticated!")
	allocFn = allocFunc
	return emptyCString
}

func serialiseForPython(msg proto.Message) *C.char {
	// Write it to python buffer
	buf, err := proto.Marshal(msg)
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to marshal response: %v", err))
	}

	mem := allocBytes(allocFn, len(buf))
	for c, v := range buf {
		mem[c] = v
	}

	return emptyCString
}

type clientRequest func() (proto.Message, error)

func processRequest(reqName string, reqFunc clientRequest) *C.char {
	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	resp, err := reqFunc()
	if err != nil {
		return C.CString(fmt.Sprintf("%v error: %v", reqName, err))
	}

	return serialiseForPython(resp)
}

//export getScanSpectrum
func getScanSpectrum(scanId string, pmc int32, spectrumType int, detector string) *C.char {
	return processRequest("getScanSpectrum", func() (proto.Message, error) {
		return apiClient.GetScanSpectrum(scanId, pmc, protos.SpectrumType(spectrumType), detector)
	})
}

//export getScanSpectrumRangeAsMap
func getScanSpectrumRangeAsMap(scanId string, channelStart int32, channelEnd int32, detector string) *C.char {
	return processRequest("getScanSpectrumRangeAsMap", func() (proto.Message, error) {
		return apiClient.GetScanSpectrumRangeAsMap(scanId, channelStart, channelEnd, detector)
	})
}

//export listScans
func listScans(scanId string) *C.char {
	return processRequest("listScans", func() (proto.Message, error) { return apiClient.ListScans(scanId) })
}

//export getScanMetaList
func getScanMetaList(scanId string) *C.char {
	return processRequest("getScanMetaList", func() (proto.Message, error) { return apiClient.GetScanMetaList(scanId) })
}

//export getScanMetaData
func getScanMetaData(scanId string) *C.char {
	return processRequest("getScanMetaData", func() (proto.Message, error) { return apiClient.GetScanMetaData(scanId) })
}

//export getScanEntryDataColumns
func getScanEntryDataColumns(scanId string) *C.char {
	return processRequest("getScanEntryDataColumns", func() (proto.Message, error) { return apiClient.GetScanEntryDataColumns(scanId) })
}

//export getScanEntryDataColumnAsMap
func getScanEntryDataColumnAsMap(scanId string, columnName string) *C.char {
	return processRequest("getScanEntryDataColumnAsMap", func() (proto.Message, error) { return apiClient.GetScanEntryDataColumnAsMap(scanId, columnName) })
}

//export listScanQuants
func listScanQuants(scanId string) *C.char {
	return processRequest("listScanQuants", func() (proto.Message, error) { return apiClient.ListScanQuants(scanId) })
}

//export getQuant
func getQuant(quantId string, summaryOnly bool) *C.char {
	return processRequest("getQuant", func() (proto.Message, error) { return apiClient.GetQuant(quantId, summaryOnly) })
}

//export listScanImages
func listScanImages(scanIds string, mustIncludeAll bool) *C.char {
	scanIdList := []string{}
	if strings.Contains(scanIds, "|") {
		scanIdList = strings.Split(scanIds, "|")
	} else {
		// Treat it as one id
		scanIdList = append(scanIdList, scanIds)
	}

	return processRequest("listScanImages", func() (proto.Message, error) { return apiClient.ListScanImages(scanIdList, mustIncludeAll) })
}

//export listScanROIs
func listScanROIs(scanId string) *C.char {
	return processRequest("listScanROIs", func() (proto.Message, error) { return apiClient.ListScanROIs(scanId) })
}

//export getROI
func getROI(id string, isMist bool) *C.char {
	return processRequest("getROI", func() (proto.Message, error) { return apiClient.GetROI(id, isMist) })
}

//export deleteROI
func deleteROI(id string) *C.char {
	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	err := apiClient.DeleteROI(id)
	if err != nil {
		return C.CString(fmt.Sprintf("deleteROI error: %v", err))
	}

	return emptyCString
}

//export getScanBeamLocations
func getScanBeamLocations(scanId string) *C.char {
	return processRequest("getScanBeamLocations", func() (proto.Message, error) { return apiClient.GetScanBeamLocations(scanId) })
}

//export getScanEntries
func getScanEntries(scanId string) *C.char {
	return processRequest("getScanEntries", func() (proto.Message, error) { return apiClient.GetScanEntries(scanId) })
}

//export getScanImageBeamLocationVersions
func getScanImageBeamLocationVersions(imageName string) *C.char {
	return processRequest("getScanImageBeamLocationVersions", func() (proto.Message, error) { return apiClient.GetScanImageBeamLocationVersions(imageName) })
}

//export getScanImageBeamLocations
func getScanImageBeamLocations(imageName string, scanId string, version int32) *C.char {
	return processRequest("getScanImageBeamLocations", func() (proto.Message, error) { return apiClient.GetScanImageBeamLocations(imageName, scanId, version) })
}

//export setUserScanCalibration
func setUserScanCalibration(scanId string, detector string, starteV float32, perChanneleV float32) *C.char {
	return processRequest("setUserScanCalibration", func() (proto.Message, error) {
		return apiClient.SetUserScanCalibration(scanId, detector, starteV, perChanneleV)
	})
}

//export getScanBulkSumCalibration
func getScanBulkSumCalibration(scanId string) *C.char {
	return processRequest("getScanBulkSumCalibration", func() (proto.Message, error) { return apiClient.GetScanBulkSumCalibration(scanId) })
}

//export getDiffractionPeaks
func getDiffractionPeaks(scanId string, calibrationSource int) *C.char {
	return processRequest("getDiffractionPeaks", func() (proto.Message, error) {
		return apiClient.GetDiffractionPeaks(scanId, protos.EnergyCalibrationSource(calibrationSource))
	})
}

//export getDiffractionAsMap
func getDiffractionAsMap(scanId string, calibrationSource int, channelStart int32, channelEnd int32) *C.char {
	return processRequest("getDiffractionAsMap", func() (proto.Message, error) {
		return apiClient.GetDiffractionAsMap(scanId, protos.EnergyCalibrationSource(calibrationSource), channelStart, channelEnd)
	})
}

//export getRoughnessAsMap
func getRoughnessAsMap(scanId string, calibrationSource int) *C.char {
	return processRequest("getRoughnessAsMap", func() (proto.Message, error) {
		return apiClient.GetRoughnessAsMap(scanId, protos.EnergyCalibrationSource(calibrationSource))
	})
}

//export getQuantColumns
func getQuantColumns(quantId string) *C.char {
	return processRequest("getQuantColumns", func() (proto.Message, error) { return apiClient.GetQuantColumns(quantId) })
}

//export getQuantColumnAsMap
func getQuantColumnAsMap(quantId string, columnName string, detector string) *C.char {
	return processRequest("getQuantColumnAsMap", func() (proto.Message, error) { return apiClient.GetQuantColumnAsMap(quantId, columnName, detector) })
}

//export createROI
func createROI(roiBuff string, isMist bool) *C.char {
	// Here we can read the roi string as a protobuf message and create the right structure
	roiItem := &protos.ROIItem{}
	err := protojson.Unmarshal([]byte(roiBuff), roiItem)
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to decode ROI: %v", err))
	}

	return processRequest("createROI", func() (proto.Message, error) { return apiClient.CreateROI(roiItem, isMist) })
}

//export saveMapData
func saveMapData(key string, dataBuff string) *C.char {
	// Here we can read the data string as a protobuf message and create the right structure
	mapItem := &protos.ClientMap{}
	err := protojson.Unmarshal([]byte(dataBuff), mapItem)
	if err != nil {
		return C.CString(fmt.Sprintf("saveMapData: Failed to decode data: %v", err))
	}

	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	err = apiClient.SaveMapData(key, mapItem)
	if err != nil {
		return C.CString(fmt.Sprintf("saveMapData error: %v", err))
	}

	return emptyCString
}

//export loadMapData
func loadMapData(key string) *C.char {
	return processRequest("loadMapData", func() (proto.Message, error) { return apiClient.LoadMapData(key) })
}

//export uploadImage
func uploadImage(imageUpload string) *C.char {
	// Here we can read the data string as a protobuf message and create the right structure
	upload := &protos.ImageUploadHttpRequest{}
	err := protojson.Unmarshal([]byte(imageUpload), upload)
	if err != nil {
		return C.CString(fmt.Sprintf("uploadImage: Failed to decode imageUpload: %v", err))
	}

	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	err = apiClient.UploadImage(upload)
	if err != nil {
		return C.CString(fmt.Sprintf("uploadImage error: %v", err))
	}

	return emptyCString
}

//export deleteImage
func deleteImage(imageName string) *C.char {
	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	err := apiClient.DeleteImage(imageName)
	if err != nil {
		return C.CString(fmt.Sprintf("deleteImage error: %v", err))
	}

	return emptyCString
}

//export getTag
func getTag(tagId string) *C.char {
	return processRequest("getTag", func() (proto.Message, error) { return apiClient.GetTag(tagId) })
}

//export getTagByName
func getTagByName(tagName string) *C.char {
	return processRequest("getTagByName", func() (proto.Message, error) { return apiClient.GetTagByName(tagName) })
}

//export uploadImageBeamLocations
func uploadImageBeamLocations(imageName string, locationsForScanBuff string) *C.char {
	// Here we can read the data string as a protobuf message and create the right structure
	locForScan := &protos.ImageLocationsForScan{}
	err := protojson.Unmarshal([]byte(locationsForScanBuff), locForScan)
	if err != nil {
		return C.CString(fmt.Sprintf("uploadImageBeamLocations: Failed to decode locationsForScanBuff: %v", err))
	}

	if apiClient == nil {
		return C.CString("Not authenticated")
	}

	err = apiClient.UploadImageBeamLocations(imageName, locForScan)
	if err != nil {
		return C.CString(fmt.Sprintf("uploadImageBeamLocations error: %v", err))
	}

	return emptyCString
}

func main() {
}
