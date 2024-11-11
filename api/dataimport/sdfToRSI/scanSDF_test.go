package sdfToRSI

import "fmt"

func Example_scanSDF() {
	ensureSDFRawExists()

	refs, err := scanSDF("./test-data/BadPath.txt")
	fmt.Printf("%v|%v\n", len(refs), err != nil)

	refs, err = scanSDF("./test-data/sdf_raw.txt")
	fmt.Printf("err: %v\n", err)
	for _, ref := range refs {
		fmt.Printf("%d: %v: '%v'\n", ref.Line, ref.What, ref.Value)
	}

	// Output:
	// 0|true
	// err: <nil>
	// 439: start: ''
	// 464: first-time: '2022-301T13:50:28'
	// 6213: dust-cover: 'opening'
	// 6647: dust-cover: 'opened'
	// 9970: dust-cover: 'closing'
	// 10155: dust-cover: 'closed'
	// 13182: dust-cover: 'opening'
	// 13308: dust-cover: 'opened'
	// 17361: new-rtt: '0C6E0204'
	// 19140: science: 'begin'
	// 19358: sci-place: 'Initialize"'
	// 19362: sci-place: 'Move to Crouch"'
	// 20073: sci-place: 'Perform OFS Eval"'
	// 21009: new-rtt: '0C6E0205'
	// 21223: sci-place: 'Move to Focus"'
	// 23322: sci-place: 'Pre Scan"'
	// 25039: sci-place: 'Perform Scan"'
	// 56876: sci-place: 'Post Scan"'
	// 59907: science: 'end'
	// 68018: new-rtt: '0C6F0201'
	// 70049: science: 'begin'
	// 70312: sci-place: 'Initialize"'
	// 70316: sci-place: 'Move to Crouch"'
	// 71984: sci-place: 'Perform OFS Eval"'
	// 72524: new-rtt: '0C6F0202'
	// 72766: sci-place: 'Move to Focus"'
	// 76051: sci-place: 'Pre Scan"'
	// 77735: sci-place: 'Perform Scan"'
	// 271793: sci-place: 'Post Scan"'
	// 276000: science: 'end'
	//
}
