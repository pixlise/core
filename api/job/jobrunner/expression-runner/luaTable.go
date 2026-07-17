package expressionrunner

import (
	"math"

	lua "github.com/yuin/gopher-lua"
)

func makeLuaTable(data PMCDataValues) *lua.LTable {
	pmcs := &lua.LTable{}
	values := &lua.LTable{}
	for _, item := range data.Values {
		pmcs.Append(lua.LNumber(item.PMC))

		// NOTE: Lua doesn't support nil values in tables. https://www.lua.org/manual/5.3/manual.html#2.1
		// so here we specify an undefined value as a NaN so it doesn't break. May need to consider just
		// excluding those PMCs completely, however then the maps wont be the same size in Lua land...
		v := item.Value
		if item.IsUndefined {
			v = math.NaN()
		}
		values.Append(lua.LNumber(v))
	}

	result := &lua.LTable{}
	result.Append(pmcs)
	result.Append(values)

	return result
}

func checkMakeLuaArray(L *lua.LState, obj interface{}) *lua.LTable {
	if t := makeLuaArrayInternal[int32](L, obj); t != nil {
		return t
	}
	if t := makeLuaArrayInternal[int64](L, obj); t != nil {
		return t
	}
	if t := makeLuaArrayInternal[int](L, obj); t != nil {
		return t
	}
	if t := makeLuaArrayInternal[float32](L, obj); t != nil {
		return t
	}
	if t := makeLuaArrayInternal[float64](L, obj); t != nil {
		return t
	}

	if t, ok := obj.([]interface{}); ok {
		result := L.NewTable()
		for _, v := range t {
			lv := makeLuaValue(L, v)
			if lv == nil {
				return nil
			}
			result.Append(lv)
		}
		return result
	}
	return nil
}

func makeLuaArrayInternal[T float32 | float64 | int | int32 | int64](L *lua.LState, obj interface{}) *lua.LTable {
	var result *lua.LTable

	if t, ok := obj.([]T); ok {
		result = L.NewTable()
		for _, v := range t {
			result.Append(lua.LNumber(v))
		}
	}

	return result
}

func checkMakeLuaField(obj interface{}) *lua.LNumber {
	if v := makeLuaFieldInternal[int32](obj); v != nil {
		return v
	}
	if v := makeLuaFieldInternal[int64](obj); v != nil {
		return v
	}
	if v := makeLuaFieldInternal[int](obj); v != nil {
		return v
	}
	if v := makeLuaFieldInternal[float32](obj); v != nil {
		return v
	}
	if v := makeLuaFieldInternal[float64](obj); v != nil {
		return v
	}
	return nil
}

func makeLuaFieldInternal[T float32 | float64 | int | int32 | int64](obj interface{}) *lua.LNumber {
	if v, ok := obj.(T); ok {
		n := lua.LNumber(v)
		return &n
	}
	return nil
}

/*
	func [T float32|float64|int|int32|int64]makeLuaField(L *lua.LState v T) *lua.LTable {
		result := L.NewTable()

		return result
	}
*/

func makeLuaValue(L *lua.LState, obj interface{}) lua.LValue {
	// Check if it's a table
	if t := checkMakeLuaArray(L, obj); t != nil {
		return t
	}

	// Check if it's a field
	if n := checkMakeLuaField(obj); n != nil {
		return *n
	}

	if s, ok := obj.(string); ok {
		return lua.LString(s)
	}

	return nil
}

func makeLuaTableGeneric(L *lua.LState, data map[string]interface{}) *lua.LTable {
	result := L.NewTable()

	for k, val := range data {
		// Check if it's a map, recurse if so!
		if m, ok := val.(map[string]interface{}); ok {
			ltable := makeLuaTableGeneric(L, m)
			result.RawSetString(k, ltable)
			continue
		}

		// Check if it's a table of tables (how we store maps!)
		if t, ok := val.([]interface{}); ok {
			subTable := L.NewTable()
			for _, v := range t {
				lv := makeLuaValue(L, v)
				if lv == nil {
					return nil
				}
				subTable.Append(lv)
			}
			result.RawSetString(k, subTable)
			continue
		}

		// Maybe it's just a field
		lv := makeLuaValue(L, val)
		if lv == nil {
			return nil
		}
		result.RawSetString(k, lv)
	}

	return result
}

func readLuaTable(L *lua.LState, t *lua.LTable) (map[string]interface{}, []interface{}) { //map[interface{}]interface{} {
	//result := map[interface{}]interface{}{}
	result := map[string]interface{}{}
	resultArray := []interface{}{}
	lastKey := 0
	isArray := true

	t.ForEach(func(k lua.LValue, v lua.LValue) {
		// To detect if we're reading an array, we check if we're reading just numbers
		readKNum, ok := k.(lua.LNumber)
		if ok {
			if readKNum != lua.LNumber(lastKey+1) {
				isArray = false
			}
			lastKey = int(readKNum)
		} else {
			isArray = false
		}

		var value interface{}

		vTable, ok := v.(*lua.LTable)
		if ok && vTable != nil {
			readTable, readArray := readLuaTable(L, vTable)

			if len(readArray) > 0 {
				value = readArray
			} else {
				value = readTable
			}
		} else {
			readNum, ok := v.(lua.LNumber)
			if ok {
				value = readNum
			} else {
				value = v.String()
			}
		}

		// if we may still be an array, write to it
		// otherwise just write to the map
		result[k.String()] = value
		if isArray {
			resultArray = append(resultArray, value)
		}
	})

	// If what we read was an array

	return result, resultArray
}
