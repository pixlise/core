package quantRunner

import "fmt"

/*
5x11, 4035 PMCs, 9 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:23 (with params)
	=> 4035*2=8070 spectra on 160 cores in 203 sec => 50.44 spectra/core in 203 sec => 4.02sec/spectra
5x11, 4035 PMCs, 6 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:47 (with params)
	=> 4035*2=8070 spectra on 160 cores in 167 sec => 50.44 spectra/core in 167 sec => 3.31sec/spectra
5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:52 (with params)
	=> 4035*2=8070 spectra on 160 cores in 112 sec => 50.44 spectra/core in 112 sec => 2.22sec/spectra

5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 10 nodes. Runtime: 3:32 (with params)
	=> 4035*2=8070 spectra on 80 cores in 212 sec => 100.88 spectra/core in 212 sec => 2.10 sec/spectra

3 elements => 2.10sec/spectra
3 elements => 2.22sec/spectra
	3 elems jumped 1.09sec
6 elements => 3.31sec/spectra
	3 elems jumped 0.71sec
9 elements => 4.02sec/spectra

Assumptions:
  - Lets make this calcatable: 9elem=4sec/spectra, 3elem = 2sec/spectra, linearly interpolate in this range
  - Works out to elements = 3*sec - 3
  - To calculate node count, we are given Core count, Runtime desired, Spectra count, Element count
  - Using the above:
    Runtime = Spectra*SpectraRuntime / (Core*Nodes)
    Nodes = Spectra*SpectraRuntime / (Runtime * Core)

    SpectraRuntime is calculated using the above formula:
    Elements = 3 * Sec - 3
    SpectraRuntime = (Elements+3) / 3

    Nodes = Spectra*((Elements + 3) / 3) / (RuntimeDesired * Cores)
    Nodes = Spectra*(Elements+3) / 3*(RuntimeDesired * Cores)

    Example using the values from above:
    Nodes = 8070*(3+3)/(3*120*8)
    Nodes = 8070*6/5088 = 9.5, close to 10

    Nodes = 8070*(9+3)/(3*203*8)
    Nodes = 96840 / 4872 = 19.9, close to 20

    Nodes = 8070*(6+3)/(3*167*8)
    Nodes = 72630 / 4008 = 18.12, close to 20

    If we're happy to run 6 elems, 8070 spectra, 8 cores in 5 minutes:
    Nodes = 8070*(6+3) / (3*300*8)
    Nodes = 72630 / 7200 = 10 nodes... seems reasonable
*/

func Example_estimateNodeCount() {
	// Based on experimental runs in: https://github.com/pixlise/core/-/issues/113

	// Can only use the ones where we had allcoation of 7.5 set in kubernetes, because the others weren't maxing out cores

	// 5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 10 nodes. Runtime: 3:22
	fmt.Println(EstimateNodeCount(4035*2, 3, 3*60+22, 8, 50))
	// 5x11, 4035 PMCs, 3 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:52 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 3, 60+52, 8, 50))
	// 5x11, 4035 PMCs, 4 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:11 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 4, 2*60+11, 8, 50))
	// 5x11, 4035 PMCs, 5 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:26 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 5, 2*60+26, 8, 50))
	// 5x11, 4035 PMCs, 6 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:47 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 6, 2*60+47, 8, 50))
	// 5x11, 4035 PMCs, 7 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 2:55 (no params though)
	fmt.Println(EstimateNodeCount(4035*2, 7, 2*60+55, 8, 50))
	// 5x11, 4035 PMCs, 8 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:12 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 8, 3*60+12, 8, 50))
	// 5x11, 4035 PMCs, 9 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:23 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 9, 3*60+23, 8, 50))
	// 5x11, 4035 PMCs, 10 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:35 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 10, 3*60+35, 8, 50))
	// 5x11, 4035 PMCs, 11 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 3:46 (with params)
	fmt.Println(EstimateNodeCount(4035*2, 11, 3*60+46, 8, 50))

	// 5x5, 1769 PMCs, 11 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 1:47 (with params)
	fmt.Println(EstimateNodeCount(1769*2, 11, 60+47, 8, 50))

	// 5x5, 1769 PMCs, 4 elements, 8 cores (7.5 allocation in kubernetes), 20 nodes. Runtime: 0:59 (with params)
	fmt.Println(EstimateNodeCount(1769*2, 4, 59, 8, 50))

	// Ensure the max cores have an effect
	fmt.Println(EstimateNodeCount(1769*2, 4, 59, 8, 6))

	// It's a bit unfortunate we ran all but 1 tests on the same number of cores, but
	// the above data varies the spectra count, element count and expected runtime

	// Below we'd expect 20 for all answers except the first one, but there's a slight (and
	// allowable) drift because we're not exactly spot on with our estimate, and there's
	// fixed overhead time we aren't even calculating properly

	// Output:
	// 10
	// 18
	// 18
	// 18
	// 18
	// 19
	// 19
	// 20
	// 20
	// 21
	// 19
	// 17
	// 6
}

func Example_filesPerNode() {
	fmt.Println(FilesPerNode(8088, 5))
	fmt.Println(FilesPerNode(8068, 3))

	// Output:
	// 1619
	// 2690
}
