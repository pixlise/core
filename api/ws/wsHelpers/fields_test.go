package wsHelpers

import "fmt"

func Example_wsHelpers_MaskSecretField() {
	fmt.Println(MaskSecretField(""))
	fmt.Println(MaskSecretField("a"))
	fmt.Println(MaskSecretField("aB"))
	fmt.Println(MaskSecretField("aBc"))
	fmt.Println(MaskSecretField("aBcD"))
	fmt.Println(MaskSecretField("aBcDe"))
	fmt.Println(MaskSecretField("aBcDeF"))
	fmt.Println(MaskSecretField("helloworld99"))

	// Output:
	// ***
	// ***
	// ***
	// ***
	// ***D
	// ***De
	// ***DeF
	// *********d99
}
