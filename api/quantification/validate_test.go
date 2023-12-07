package quantification

import "fmt"

func Example_validateParameters() {
	fmt.Printf("%v\n", validateParameters("-b,0,50,2,10 -f"))
	fmt.Printf("%v\n", validateParameters("-b,0,50,2,10.55 -o \"filename.whatever\" -f -Fe,1"))
	fmt.Printf("%v\n", validateParameters("-b,0,50,2,10;ls -al;echo -f"))
	fmt.Printf("%v\n", validateParameters("-b,0,50,2,10&&rm -rf ~/; -f"))

	// Output:
	// <nil>
	// <nil>
	// Invalid parameters passed: -b,0,50,2,10;ls -al;echo -f
	// Invalid parameters passed: -b,0,50,2,10&&rm -rf ~/; -f
}
