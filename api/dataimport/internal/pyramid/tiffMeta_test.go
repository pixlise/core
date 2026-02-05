package pyramid

import "fmt"

func Example_pyramid_readTiffMeta() {
	//inputTiff := "./test-data/Big_Import/pyramid/Multi_page24bpp.tif"
	inputTiff := "./test-data/Units/DimensionsMismatch.tif"
	//inputTiff := "./test-data/Units/sample_5mb.tiff"
	//inputTiff := "/home/peter/Documents/RawImageData/largeImport/pyramid/optical_z-stack.tif"

	meta, err := readTiffMeta(inputTiff)
	fmt.Printf("%v|%v\n", err, meta)

	// Output:
	// <nil>|something
}
