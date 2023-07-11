package wstestlib

import (
	"fmt"
)

func Example_doReqReplacements() {
	savedLookup := map[string]string{"theName": "NOW", "moreVar": "More"}
	fmt.Println(doReqReplacements("here\nis some\ntext", savedLookup))
	fmt.Println()
	fmt.Println(doReqReplacements("here ${IDLOAD=theName} is\nsome ${IDLOAD=moreVar}\ntext", savedLookup))
	fmt.Println(doReqReplacements("here ${IDLOAD=non-existant} it fails", savedLookup))
	fmt.Println(doReqReplacements("here ${IDCHK=theName} it fails", savedLookup))
	fmt.Println(doReqReplacements("here ${ID=theName unfinished", savedLookup))

	// Output:
	// here
	// is some
	// text <nil>
	//
	// here NOW is
	// some More
	// text <nil>
	//  IDLOAD: No replacement text named: non-existant for request message: here ${IDLOAD=non-existant} it fails
	//  Unknown definition used on request message: IDCHK
	//  failed to find closing token for "}" in "here ${ID=theName unfinished"
}
