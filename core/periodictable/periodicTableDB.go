package periodictable

import (
	"strconv"

	"github.com/pixlise/core/v4/core/logger"
)

type PeriodicTableItem struct {
	Name       string
	AtomicMass float64
	Z          int
	Symbol     string
}

type ElementOxidationState struct {
	Formula                      string
	Element                      string
	Z                            int
	IsElement                    bool    // Fe would contain true, Fe2O3 would contain false
	ConversionToElementWeightPct float64 // For example, the number to divide Fe2O3 weight % to get Fe weight %
}

type PeriodicTableDB struct {
	periodicTable []PeriodicTableItem
	symbolToIdx   map[string]int
	log           logger.ILogger
}

func MakePeriodicTable(l logger.ILogger) *PeriodicTableDB {
	//items := []PeriodicTableItem{}
	items := FillTable()

	// Make the lookup table
	symbolLookup := map[string]int{}

	for c, item := range items {
		symbolLookup[item.Symbol] = c
	}

	return &PeriodicTableDB{items, symbolLookup, l}
}

func (pdb *PeriodicTableDB) getElementBySymbol(formula string) *PeriodicTableItem {
	idx := pdb.getElementIndex(formula)
	if idx == -1 {
		return nil
	}
	return &pdb.periodicTable[idx]
}

func (pdb *PeriodicTableDB) GetMolecularMass(formula string) float64 {
	// Allows finding the molecular mass not just of an individual element, but of an entire formula, eg SiO2
	// This needs to find elements and the multipliciation factor in the formula
	weight := float64(0)

	// Special case - if it's FeO-T, we worked correctly before, but had pointless warning messages for O-T
	// because it was read as "O-" which wasn't a valid element, after which "O" was tried, and worked... so here
	// we snip off O-T in this case
	if formula == "FeO-T" {
		formula = "FeO"
	}

	formulaRemainder := formula
	for c := 0; c < len(formula); c++ {
		thisWeight := float64(0)

		elem := pdb.getFirstElement(formulaRemainder)
		if len(elem) > 0 {
			elemItem := pdb.getElementBySymbol(elem)
			if elemItem != nil {
				thisWeight = elemItem.AtomicMass
			} else {
				thisWeight = 0
			}

			// consume these chars
			formulaRemainder = formulaRemainder[len(elem):]
		} else {
			// Special case for FeO-T style "totals". Initially was only for FeO but can happen for others now
			// such as FeCO3-T
			if formulaRemainder == "-T" /* && formula == "FeO-T"*/ {
				formulaRemainder = ""
				break
			}

			pdb.log.Errorf("GetMolecularMass: Failed to find element in \"%v\" part of formula: \"%v\"", formulaRemainder, formula)
			weight = 0
			break
		}

		// See if it is followed by a multiplier, eg CO2 - O finding it's x2
		mult := pdb.getMultiplier(formulaRemainder)

		if mult > 0 {
			thisWeight *= mult

			// Consume the characters
			snip := 1
			if mult > 9 {
				snip = 2
			}
			formulaRemainder = formulaRemainder[snip:]
		}

		weight += thisWeight

		if len(formulaRemainder) <= 0 {
			break
		}
	}

	return weight
}

// Returns the first element in the formula
func (pdb *PeriodicTableDB) getFirstElement(formula string) string {
	// See if we can find an element on the first 2 chars
	if len(formula) > 1 {
		elem := formula[0:2]
		state := pdb.GetElementOxidationState(elem)
		if state != nil && state.IsElement {
			return state.Element
		}
	}

	// If not, maybe it's only the first char
	if len(formula) > 0 {
		elem := formula[0:1]
		state := pdb.GetElementOxidationState(elem)
		if state != nil && state.IsElement {
			return state.Element
		}
	}

	// Nope, return nothing
	return ""
}

// Returns the multiplier at the start of the formula string (or 0 if none)
// Really just returns a number if string starts with one, used when parsing
// a chemical formula
func (pdb *PeriodicTableDB) getMultiplier(formula string) float64 {
	// Read numbers until we are out
	numChars := ""
	for _, ch := range formula {
		if ch < '0' || ch > '9' {
			break
		}
		numChars += string(ch)
	}

	num, err := strconv.Atoi(numChars)
	if err != nil {
		return 0
	}

	if num <= 0 {
		return 0
	}

	return float64(num)
}

func (pdb *PeriodicTableDB) getElementIndex(formula string) int {
	// NOTE: here a symbol was originally an element, but PIQUANT has since been modified
	// to return quantifications with oxides or carbonates of an element, so the symbol
	// might be Fe, but could be Fe2O3.
	// To work around this, here we first try look up by the first 2 letters, then by 1 letter
	// to cover the case of 1 char elements. If both fail, then return null
	// NOTE 2: PIQUANT has since been modified to also output (in particular with Iron) -T meaning
	// total. So this also has to handle that case

	if len(formula) < 1 {
		return -1
	}

	endIdx := 2
	if len(formula) < 2 {
		endIdx = 1
	}

	idx, ok := pdb.symbolToIdx[formula[0:endIdx]]
	if !ok {
		if endIdx > 1 {
			idx, ok = pdb.symbolToIdx[formula[0:1]]
			if !ok {
				idx = -1
			}
		} else {
			idx = -1
		}
	}

	return idx
}

func (pdb *PeriodicTableDB) GetElementOxidationState(formula string) *ElementOxidationState {
	var result *ElementOxidationState

	idx := pdb.getElementIndex(formula)
	if idx != -1 {
		// We found an element in it, check if this formula is longer than just the symbol and form
		// the result
		elem := pdb.periodicTable[idx]
		result = &ElementOxidationState{
			Formula:                      formula,
			Element:                      elem.Symbol,
			Z:                            elem.Z,
			IsElement:                    len(formula) <= len(elem.Symbol),
			ConversionToElementWeightPct: pdb.getFormulaToElementConversionFactor(elem.Symbol, formula),
		}
	}

	return result
}

func (pdb *PeriodicTableDB) getFormulaToElementConversionFactor(element string, formula string) float64 {
	result := float64(1)

	elementData := pdb.getElementBySymbol(element)
	if elementData != nil {
		elementMass := elementData.AtomicMass

		// Find if it's multiplied by something
		pos := len(element)

		otherElementMass := float64(0)
		lastElementMass := float64(0)

		tmp := ""
		for pos < len(formula) {
			ch := formula[pos : pos+1]

			// Add to the last one
			tmp += ch

			// See if it's an element
			tmpData := pdb.getElementBySymbol(tmp)
			if tmpData != nil {
				// If we already have a mass read, add it
				otherElementMass += lastElementMass

				// Remember it
				lastElementMass = tmpData.AtomicMass

				tmp = ""
			} else {
				// See if it's a multiplier
				mult, err := strconv.Atoi(tmp)
				if err == nil && mult > 0 {
					if lastElementMass == 0 {
						// It's a multiplier for the element...
						elementMass *= float64(mult)
					}

					massToAdd := float64(mult) * lastElementMass
					otherElementMass += massToAdd
					lastElementMass = 0
					tmp = ""
				} else {
					// Special case for handling FeO-T and other "-T" totals...
					if formula[pos:] == "-T" /*&& formula == "FeO-T"*/ {
						break
					}

					pdb.log.Errorf("getFormulaToElementConversionFactor: Failed while parsing %v in: %v", ch, formula)
				}
			}

			pos++
		}

		if lastElementMass != 0 {
			otherElementMass += lastElementMass
		}

		if otherElementMass != 0 {
			// We have the element mass (eg Ti), and the mass of "the other stuff" (eg O2 from TiO2). Get the mass of the element
			// as a percentage of the total mass, giving us a factor to multiply the total weight% (eg of TiO2), that allows us to
			// extract just the weight% of Ti. Our resultant value is the factor, and that multiplication with total weight% is elsewhere
			result = elementMass / (elementMass + otherElementMass)
		}
	}

	return result
}
