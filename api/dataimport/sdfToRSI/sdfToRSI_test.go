package sdfToRSI

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Example_ConvertSDFtoRSI() {
	ensureSDFRawExists()

	fmt.Printf("mkdir worked: %v\n", os.MkdirAll("./output", 0777) == nil) // other than 0777 fails in unit tests :(
	wd, err := os.Getwd()
	fmt.Printf("Getwd: %v\n", err == nil)
	p := filepath.Join(wd, "output")
	files, rtts, err := ConvertSDFtoRSIs("./test-data/sdf_raw.txt", p)
	fmt.Printf("%v, %v: %v\n", files, rtts, err)

	// Output:
	// mkdir worked: true
	// Getwd: true
	// [RSI-208536069.csv HK-208536069.csv RSI-208601602.csv HK-208601602.csv], [208536069 208601602]: <nil>
}

func ensureSDFRawExists() {
	// If it's not here, unzip it
	_, err := os.Stat("./test-data/sdf_raw.txt")
	if err != nil {
		archive, err := zip.OpenReader("./test-data/sdf_raw.zip")
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		for _, f := range archive.File {
			if len(archive.File) != 1 || f.Name != "sdf_raw.txt" {
				log.Fatalln("Expected sdf_raw.zip to only contain one file: sdf_raw.txt")
			}

			dstFile, err := os.OpenFile("./test-data/sdf_raw.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				log.Fatalf("Failed to unzip sdf_raw.txt: %v\n", err)
			}
			defer dstFile.Close()

			fileInArchive, err := f.Open()
			if err != nil {
				log.Fatalf("Failed to open sdf_raw.txt for writing: %v\n", err)
			}
			defer fileInArchive.Close()

			if _, err := io.Copy(dstFile, fileInArchive); err != nil {
				log.Fatalf("Failed to wwrite sdf_raw.txt: %v\n", err)
			}
		}
	}
}

func Example_readRTT() {
	r, e := readRTT("")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("123")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("948427324")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("0x1C38D33E")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("1C38D33E") // If lacking the 0x, we should be trying it as hex just in case, some files come with 000001C5 for RTT=453
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("1234/0x333")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("1234/0x4D2")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("0x7F9G03")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("Aword")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("123/456")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("22/0xHello")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("/")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("1/2/3")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("345/245H")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("345/")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("/45")
	fmt.Printf("%v|%v\n", r, e)
	r, e = readRTT("/0x7E")
	fmt.Printf("%v|%v\n", r, e)

	// Output:
	// 0|Failed to read RTT from empty string
	// 123|<nil>
	// 948427324|<nil>
	// 473486142|<nil>
	// 473486142|<nil>
	// 0|Read RTT where int didn't match hex value: "1234/0x333".
	// 1234|<nil>
	// 0|Failed to read hex RTT: "0x7F9G03". Error: strconv.ParseInt: parsing "7F9G03": invalid syntax
	// 0|Failed to read integer RTT: "Aword". Error: strconv.ParseInt: parsing "Aword": invalid syntax
	// 0|Expected hex rtt after / for RTT: "123/456"
	// 0|Failed to read hex part of RTT: "22/0xHello". Error: strconv.ParseInt: parsing "Hello": invalid syntax
	// 0|Failed to read integer part of RTT: "/". Error: strconv.ParseInt: parsing "": invalid syntax
	// 0|Invalid RTT read: "1/2/3"
	// 0|Expected hex rtt after / for RTT: "345/245H"
	// 0|Expected hex rtt after / for RTT: "345/"
	// 0|Failed to read integer part of RTT: "/45". Error: strconv.ParseInt: parsing "": invalid syntax
	// 0|Failed to read integer part of RTT: "/0x7E". Error: strconv.ParseInt: parsing "": invalid syntax
}
