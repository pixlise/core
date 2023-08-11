package wsHelpers

import "fmt"

func Example_orderCacheItems() {
	items := map[string]datasetCacheItem{
		"one": {
			localPath:        "path/one.bin",
			fileSize:         10 * 1024 * 1024,
			timestampUnixSec: 1234567891,
		},
		"three": {
			localPath:        "path/three.bin",
			fileSize:         30 * 1024 * 1024,
			timestampUnixSec: 1234567893,
		},
		"two": {
			localPath:        "path/two.bin",
			fileSize:         20 * 1024 * 1024,
			timestampUnixSec: 1234567892,
		},
		"four": {
			localPath:        "path/four.bin",
			fileSize:         40 * 1024 * 1024,
			timestampUnixSec: 1234567894,
		},
	}

	byAge, totalSize := orderCacheItems(items)
	fmt.Printf("Total size: %v\n", totalSize)

	for _, i := range byAge {
		fmt.Printf("path: %v, ts: %v, size: %v\n", i.localPath, i.timestampUnixSec, i.fileSize)
	}

	// Output:
	// Total size: 104857600
	// path: path/four.bin, ts: 1234567894, size: 41943040
	// path: path/three.bin, ts: 1234567893, size: 31457280
	// path: path/two.bin, ts: 1234567892, size: 20971520
	// path: path/one.bin, ts: 1234567891, size: 10485760
}
