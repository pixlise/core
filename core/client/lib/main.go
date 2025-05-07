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

/*
//export getSpectrum
func getSpectrum(scanId string, pmc int, spectrumType int, detector string) *C.char {
	if socket == nil {
		return C.CString("Not authenticated")
	}

	fmt.Printf("getSpectrum called!\n")
	fmt.Printf("scanId: \"%v\"\n", scanId)
	fmt.Printf("pmc: \"%v\"\n", pmc)
	fmt.Printf("spectrumType: \"%v\"\n", spectrumType)
	fmt.Printf("detector: \"%v\"\n", detector)

	counts, err := apiClient.GetSpectrum(socket, scanId, pmc, protos.SpectrumType(spectrumType), detector)
	if err != nil {
		return C.CString(fmt.Sprintf("%v", err))
	}

	mem := allocInts(len(counts))
	for c, v := range counts {
		mem[c] = int64(v)
	}

	return emptyCString
}

//export testStrings
func testStrings(scanId string, pmc int) *C.char {
	r := fmt.Sprintf("testStrings %v, %v", scanId, pmc)
	fmt.Println(r)
	return C.CString(r) // Something will need to call C.free(unsafe.Pointer(...))
}

//export testIntArray
func testIntArray(scanId string, pmc int) {
	r := fmt.Sprintf("testIntArray %v, %v", scanId, pmc)
	fmt.Println(r)

	result := []int32{20, 30, 40}
	mem := allocInts(len(result))

	fmt.Printf("mem: %v\n", len(mem))

	for c, v := range result {
		mem[c] = int64(v)
	}

	// Python can access it in its own array... return uintptr(unsafe.Pointer(&result[0])) //<-- note: result[0]
}*/

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

//export getScanEntryDataColumn
func getScanEntryDataColumn(scanId string, columnName string) *C.char {
	return processRequest("getScanEntryDataColumn", func() (proto.Message, error) { return apiClient.GetScanEntryDataColumn(scanId, columnName) })
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

//export getDiffractionPeaks
func getDiffractionPeaks(scanId string) *C.char {
	return processRequest("getDiffractionPeaks", func() (proto.Message, error) { return apiClient.GetDiffractionPeaks(scanId) })
}

//export getQuantColumns
func getQuantColumns(quantId string) *C.char {
	return processRequest("getQuantColumns", func() (proto.Message, error) { return apiClient.GetQuantColumns(quantId) })
}

//export getQuantColumn
func getQuantColumn(quantId string, columnName string, detector string) *C.char {
	return processRequest("getQuantColumn", func() (proto.Message, error) { return apiClient.GetQuantColumn(quantId, columnName, detector) })
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

func main() {
}

/* What to implement:

From expression language:
[x] Reading Quant columns as map
[x] Reading Housekeeping columns as map
[ ] Read diffraction as map
[ ] Read roughness as map
[x] Read x,y,z

Other ideas:
[x] List scans
[x] List scan PMCs
[x] List meta fields in scans
[x] List quants per scan
[x] List columns in quants
[x] List images per scan
[x] Get image beam versions
[x] Get i,j per image
[x] List ROIs per scan
[x] Get ROI

[x] Read spectrum
[ ] Set/get spectrum calibration?
[ ] Read spectrum as a range map

Saving new stuff:
[x] Create ROI
[ ] Create Quant?
[ ] Create beam locations?
[ ] Create a named map? Update UI if map changes?

[ ] Make sure client has some kind of rate limit so people can't update stuff?
[ ] Maybe make a sleep-based soft limit for rate limit of gets, and exception for hard-limit
[x] Make public endpoint to get client connection details for auth0 (and api host address)
[x] Make authenticate() not take parameters, just look in predefined path + env var for user/pass/host
[ ] Maybe add some python-specific helper stuff to turn maps into numpy/pandas thingies
[ ] If UI viewing ROI, update if ROI changes?

*/
