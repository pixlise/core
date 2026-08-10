package quantRunner

import "fmt"

func Example_estimateNodeCount() {
	// PIXLISE "cheaper" envs and new job runner now changes this:
	runtimeSecs := []uint{30, 60, 120, 180, 300, 600, 900}
	elems := []uint{1, 4, 23}
	for _, elemCount := range elems {
		for _, runtimeSec := range runtimeSecs {
			nodes := EstimateNodeCount(1377, elemCount, runtimeSec, 150)
			fmt.Printf("EstimateNodeCount(1377 spectra, %v elems, %v sec, 50 maxNodes): %v\n", elemCount, runtimeSec, nodes)
		}
	}

	// Output:
	// EstimateNodeCount(1377 spectra, 1 elems, 30 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 1 elems, 60 sec, 50 maxNodes): 80
	// EstimateNodeCount(1377 spectra, 1 elems, 120 sec, 50 maxNodes): 40
	// EstimateNodeCount(1377 spectra, 1 elems, 180 sec, 50 maxNodes): 26
	// EstimateNodeCount(1377 spectra, 1 elems, 300 sec, 50 maxNodes): 16
	// EstimateNodeCount(1377 spectra, 1 elems, 600 sec, 50 maxNodes): 8
	// EstimateNodeCount(1377 spectra, 1 elems, 900 sec, 50 maxNodes): 5
	// EstimateNodeCount(1377 spectra, 4 elems, 30 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 4 elems, 60 sec, 50 maxNodes): 127
	// EstimateNodeCount(1377 spectra, 4 elems, 120 sec, 50 maxNodes): 63
	// EstimateNodeCount(1377 spectra, 4 elems, 180 sec, 50 maxNodes): 42
	// EstimateNodeCount(1377 spectra, 4 elems, 300 sec, 50 maxNodes): 25
	// EstimateNodeCount(1377 spectra, 4 elems, 600 sec, 50 maxNodes): 12
	// EstimateNodeCount(1377 spectra, 4 elems, 900 sec, 50 maxNodes): 8
	// EstimateNodeCount(1377 spectra, 23 elems, 30 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 23 elems, 60 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 23 elems, 120 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 23 elems, 180 sec, 50 maxNodes): 150
	// EstimateNodeCount(1377 spectra, 23 elems, 300 sec, 50 maxNodes): 113
	// EstimateNodeCount(1377 spectra, 23 elems, 600 sec, 50 maxNodes): 56
	// EstimateNodeCount(1377 spectra, 23 elems, 900 sec, 50 maxNodes): 37
}

func Example_filesPerNode() {
	fmt.Printf("FilesPerNode 8088, 5 = %v\n", FilesPerNode(8088, 5))
	fmt.Printf("FilesPerNode 8068, 3 = %v\n", FilesPerNode(8068, 3))

	// Output:
	// FilesPerNode 8088, 5 = 1618
	// FilesPerNode 8068, 3 = 2689
}
