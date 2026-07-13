package expressionrunner

import (
	"fmt"
	"math"

	lua "github.com/Shopify/go-lua"
	"github.com/pixlise/core/v4/core/scan"
)

type PMCDataValue struct {
	// This is a single value for a PMC. Initially it was just a number, however with multi-quant
	// we ended up needing "undefined" values because that element simply has no value quantified.
	// After some consultation it seems we still want to treat these as 0 in the case of calculations
	// because if we add 2 maps together where one has some undefined values in it, we want the new
	// map to only contain the other value.
	//
	// Initially wanted to use JS undefined, but undefined+12==undefined.
	//
	// Thought of defining a "special" undefined value but that means any math done will have to check
	// for this.
	//
	// Finally, it seems a separate isUndefined flag is the easiest way to go. When element() reads it,
	// some values may have this set to true, but after arithmatic it will become false.
	//
	// Also this way if something displaying the data wants to treat them as 0's it doesn't need
	// modification, but if it wants to treat them otherwise it has a separate variable to look at.
	// isUndefined defaults to false because we rarely actually want to create an undefined value!
	PMC         int
	Value       float64
	IsUndefined bool
	Label       string
}

func MakePMCDataValue(pmc int, value float64, isUndefined bool, label string) PMCDataValue {
	if isUndefined && value != 0 {
		fmt.Printf("PMC: %v is undefined, but value is: %v", pmc, value)
		value = 0
	}

	return PMCDataValue{
		PMC:         pmc,
		Value:       value,
		IsUndefined: isUndefined,
		Label:       label,
	}
}

type PMCDataValues struct {
	ValueRange scan.MinMax
	Values     []PMCDataValue
	IsBinary   bool
	Warning    string
}

func (p *PMCDataValues) AddValue(v PMCDataValue) {
	if !v.IsUndefined {
		p.ValueRange.Expand(v.Value)
	}
	p.Values = append(p.Values, v)

	if v.Value != 0 && v.Value != 1 {
		p.IsBinary = false
	}
}

func (p *PMCDataValues) SetValues(values []PMCDataValue) {
	if len(values) <= 0 {
		return
	}

	p.Values = values
	p.ValueRange = scan.MinMax{}
	p.IsBinary = true // if we see anything that's not 0 or 1, we mark this as false

	for _, item := range values {
		if !item.IsUndefined {
			p.ValueRange.Expand(item.Value)
		}

		if item.Value != 0 && item.Value != 1 {
			p.IsBinary = false
		}
	}

	if p.ValueRange.GetRange() == 0 {
		// It's not actually binary...
		p.IsBinary = false
	}
}

/*
	func MakePMCDataValuesWithValues(values []PMCDataValue) PMCDataValues {
		result := PMCDataValues{}
		result.SetValues(values)
		return result
	}
*/
func makePMCDataValuesWithMinMax(values []PMCDataValue, valRange scan.MinMax, isBinary bool) PMCDataValues {
	result := PMCDataValues{
		ValueRange: valRange,
		Values:     values,
		IsBinary:   isBinary,
	}
	return result
}

func pushAsLuaTable(data PMCDataValues, L *lua.State) {
	// Make the overall table
	L.NewTable()

	// Push PMC table
	L.NewTable()

	// Push each PMC in at consecutive indexes
	for c, item := range data.Values {
		L.PushInteger(item.PMC)
		L.RawSetInt(-2, c+1)
	}

	// Set this table as a field in the overall table
	L.RawSetInt(-2, 1)

	// Push the values table
	L.NewTable()

	for c, item := range data.Values {
		v := item.Value

		// NOTE: Lua doesn't support nil values in tables. https://www.lua.org/manual/5.3/manual.html#2.1
		// so here we specify an undefined value as a NaN so it doesn't break. May need to consider just
		// excluding those PMCs completely, however then the maps wont be the same size in Lua land...
		if item.IsUndefined {
			v = math.NaN()
		}

		L.PushNumber(v)
		L.RawSetInt(-2, c+1)
	}

	L.RawSetInt(-2, 2)
}
