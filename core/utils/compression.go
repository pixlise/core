package utils

import "golang.org/x/exp/constraints"

func ZeroRunDecode[T constraints.Integer](data []T) []T {
	result := []T{}

	for c := 0; c < len(data); c++ {
		v := data[c]
		if v != 0 {
			// Just copy it across
			result = append(result, v)
		} else {
			// We found a 0, this is going to be followed by the number of 0's. Read ahead and fill that many
			// 0's in our result
			count := data[c+1]

			for i := T(0); i < count; i++ {
				result = append(result, 0)
			}

			// Skip over the count value next run
			c++
		}
	}

	return result
}

func ZeroRunEncode[T constraints.Integer](data []T) []T {
	encoded := []T{}
	count := 0
	init := false
	for _, val := range data {
		if val != 0 {
			if init {
				encoded = append(encoded, T(0))
				encoded = append(encoded, T(count))
				init = false
			}
			encoded = append(encoded, T(val))

		} else {
			if !init {
				count = 0
				init = true
			}
			count = count + 1
		}
	}

	if init {
		encoded = append(encoded, 0)
		encoded = append(encoded, T(count))
	}
	return encoded
}

/*
func RunLengthEncode(data []int) []int {
	var encoded []int

	count := 0

	last := data[0]

	for _, val := range data {
		if last == val {
			count = count + 1
		} else {
			encoded = append(encoded, last)
			encoded = append(encoded, count)
			last = val
			count = 1
		}
	}

	encoded = append(encoded, last)
	encoded = append(encoded, count)

	return encoded
}
*/
