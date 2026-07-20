package periodictable

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_periodictable_getFormulaToElementConversionFactor() {
	p := MakePeriodicTable(&logger.StdOutLoggerForTest{})

	fmt.Printf("Ca->Ca0: %v\n", p.getFormulaToElementConversionFactor("Ca", "CaO"))
	fmt.Printf("Ca->CaCO3: %v\n", p.getFormulaToElementConversionFactor("Ca", "CaCO3"))
	fmt.Printf("Fe->FeCO3: %v\n", p.getFormulaToElementConversionFactor("Fe", "FeCO3"))
	fmt.Printf("Ti->TiO2: %v\n", p.getFormulaToElementConversionFactor("Ti", "TiO2"))
	fmt.Printf("Cr->Cr2O3: %v\n", p.getFormulaToElementConversionFactor("Cr", "Cr2O3"))
	fmt.Printf("Fe->FeO-T: %v\n", p.getFormulaToElementConversionFactor("Fe", "FeO-T"))

	// Output:
	// Ca->Ca0: 0.714700941878836
	// Ca->CaCO3: 0.400442805017924
	// Fe->FeCO3: 0.4820372150994077
	// Ti->TiO2: 0.5994081032764639
	// Cr->Cr2O3: 0.6842020077610267
	// Fe->FeO-T: 0.7773110413326207
}

func Example_periodictable_getFirstElement() {
	p := MakePeriodicTable(&logger.StdOutLoggerForTest{})

	test := []string{"O", "TiO2", "CaO3", "CO", "FeO-T"}
	for _, t := range test {
		fmt.Printf("%v -> %v\n", t, p.getFirstElement(t))
	}

	// Output:
	// O -> O
	// TiO2 -> Ti
	// CaO3 -> Ca
	// CO -> C
	// FeO-T -> Fe
}

func Example_periodictable_getMultiplier() {
	p := MakePeriodicTable(&logger.StdOutLoggerForTest{})

	test := []string{"K", "-T", "3", "3K", "36", "Cr2O3"}
	for _, t := range test {
		fmt.Printf("multiplier of %v = %v\n", t, p.getMultiplier(t))
	}

	// Output:
	// multiplier of K = 0
	// multiplier of -T = 0
	// multiplier of 3 = 3
	// multiplier of 3K = 3
	// multiplier of 36 = 36
	// multiplier of Cr2O3 = 0
}

func Example_periodictable_getElementIndex() {
	p := MakePeriodicTable(&logger.StdOutLoggerForTest{})

	test := []string{"O", "TiO2", "CaO3", "FeO-T"}
	for _, t := range test {
		fmt.Printf("Z(%v)=%v\n", t, p.getElementIndex(t))
	}

	// Output:
	// Z(O)=7
	// Z(TiO2)=21
	// Z(CaO3)=19
	// Z(FeO-T)=25
}

func Example_periodictable_GetMolecularMass() {
	p := MakePeriodicTable(&logger.StdOutLoggerForTest{})

	test := []string{"O", "Ca", "CaO", "TiO2", "CaCO3", "Cr2O3", "FeO-T", "", "4", "Dx", "Cz2O3", "Cr2D3", "Ca0", "Ca-3", "Ca,Fe"}
	expected := []float64{15.9994, 40.08, 40.08 + 15.9994, 47.88 + 15.9994*2, 40.08 + 12.011 + 15.9994*3, 51.996*2 + 15.9994*3, 55.847 + 15.9994, 0, 0, 0, 0, 0, 0, 0, 0}
	for c, t := range test {
		fmt.Printf("mass of \"%v\" = %v, expected = %v\n", t, p.GetMolecularMass(t), expected[c])
	}

	// Output:
	// mass of "O" = 15.9994, expected = 15.9994
	// mass of "Ca" = 40.08, expected = 40.08
	// mass of "CaO" = 56.0794, expected = 56.0794
	// mass of "TiO2" = 79.8788, expected = 79.8788
	// mass of "CaCO3" = 100.08919999999999, expected = 100.0892
	// mass of "Cr2O3" = 151.99020000000002, expected = 151.9902
	// mass of "FeO-T" = 71.8464, expected = 71.8464
	// mass of "" = 0, expected = 0
	// mass of "4" = 0, expected = 0
	// mass of "Dx" = 0, expected = 0
	// mass of "Cz2O3" = 0, expected = 0
	// mass of "Cr2D3" = 0, expected = 0
	// mass of "Ca0" = 0, expected = 0
	// mass of "Ca-3" = 0, expected = 0
	// mass of "Ca,Fe" = 0, expected = 0
}
