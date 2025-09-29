package imgFormat

import (
	"fmt"
	"os"
)

func Example_readUNCFile() {
	imgFileBytes, err := os.ReadFile("./test-data/jpg_0777241795_1ADA0201_000004.unc")
	if err != nil {
		return
	}

	w, h, d, err := ReadUNCFile(imgFileBytes)
	fmt.Printf("%v|%v|%v|%v,%v,%v,%v|%v,%v,%v,%v|%v", w, h, len(d), d[0], d[1], d[2], d[3], d[84000], d[84001], d[84002], d[84003], err)

	// Output:
	// 752|580|1744640|48,48,48,255|49,49,49,255|<nil>
}

func Example_extractJPGFromUNCFile() {
	imgFileBytes, err := os.ReadFile("./test-data/jpg_0777241795_1ADA0201_000004.unc")
	if err != nil {
		return
	}

	err = ExtractJPGFromUNCFile(imgFileBytes, "test-output.jpg")
	fmt.Printf("%v", err)

	// Output:
	// <nil>
}
