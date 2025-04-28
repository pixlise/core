package main

import (
	"fmt"
	"unsafe"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
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

var socket *client.SocketConn

//export authenticate
func authenticate(configFile string) {
	fmt.Printf("configFile: \"%v\"\n", configFile)
	var err error

	// Try to load the config file
	socket, err = client.Authenticate(configFile)
	fmt.Printf("%v\n", err)

	fmt.Printf("Authenticated!\n")
}

//export getSpectrum
func getSpectrum(alloc C.alloc_f, scanId string, pmc int, spectrumType int, detector string) *C.char {
	if socket == nil {
		return C.CString("Not authenticated")
	}

	fmt.Printf("getSpectrum called!\n")
	fmt.Printf("scanId: \"%v\"\n", scanId)
	fmt.Printf("pmc: \"%v\"\n", pmc)
	fmt.Printf("spectrumType: \"%v\"\n", spectrumType)
	fmt.Printf("detector: \"%v\"\n", detector)

	counts, err := client.GetSpectrum(socket, scanId, pmc, protos.SpectrumType(spectrumType), detector)
	if err != nil {
		return C.CString(fmt.Sprintf("%v", err))
	}

	mem := allocInts(alloc, len(counts))
	for c, v := range counts {
		mem[c] = int64(v)
	}

	return C.CString("")
}

//export testStrings
func testStrings(scanId string, pmc int) *C.char {
	r := fmt.Sprintf("testStrings %v, %v", scanId, pmc)
	fmt.Println(r)
	return C.CString(r) // Something will need to call C.free(unsafe.Pointer(...))
}

//export testIntArray
func testIntArray(alloc C.alloc_f, scanId string, pmc int) {
	r := fmt.Sprintf("testIntArray %v, %v", scanId, pmc)
	fmt.Println(r)

	result := []int32{20, 30, 40}
	mem := allocInts(alloc, len(result))

	fmt.Printf("mem: %v\n", len(mem))

	for c, v := range result {
		mem[c] = int64(v)
	}

	// Python can access it in its own array... return uintptr(unsafe.Pointer(&result[0])) //<-- note: result[0]
}

func main() {
}
