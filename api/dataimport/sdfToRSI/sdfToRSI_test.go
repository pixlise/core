package sdfToRSI

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
)

func Example_ConvertSDFtoRSI() {
	ensureSDFRawExists()

	fmt.Printf("mkdir worked: %v\n", os.MkdirAll("./output", 0644) == nil)
	files, rtts, err := ConvertSDFtoRSIs("./test-data/sdf_raw.txt", "./output/")
	fmt.Printf("%v, %v: %v\n", files, rtts, err)

	// Output:
	// mkdir worked: true
	// [RSI-208536069.csv RSI-208601602.csv], [208536069 208601602]: <nil>
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
