package expressionrunner

import (
	"strings"

	"github.com/pixlise/core/v4/core/utils"
)

// Returns pureElementColumnLookup, elementColumns
func buildPureElementLookup(quantLabels []string) (map[string]string, map[string][]string) {
	pureElementColumnLookup := map[string]string{}
	elementColumns := map[string][]string{}

	// Loop through all column names that may contain element information and store these names so we
	// can easily find them at runtime
	columnTypesFound := map[string]bool{}
	elements := map[string]bool{}

	for _, label := range quantLabels {
		labelBits := strings.Split(label, "_")
		if len(labelBits) == 2 {
			if labelBits[1] == "%" || labelBits[1] == "err" || labelBits[1] == "int" {
				// Remember it as a column type
				columnTypesFound[labelBits[1]] = true

				// Remember the element we found
				elements[labelBits[0]] = true
			}
		}
	}

	for elem := range elements {
		colTypes := utils.GetMapKeys(columnTypesFound)
		elementColumns[elem] = colTypes

		if utils.ItemInSlice("%", colTypes) {
			// If we have a weight % column, and it's not an element, but a carbonate/oxide, we need to add
			// just weight % for the element
			elemState := PTable.GetElementOxidationState(elem)
			if elemState != nil && !elemState.IsElement {
				elementColumns[elemState.Element] = []string{"%"}

				// Also remember that this can be calculated
				pureElementColumnLookup[elemState.Element+"_%"] = elem + "_%"
			}
		}
	}

	return pureElementColumnLookup, elementColumns
}
