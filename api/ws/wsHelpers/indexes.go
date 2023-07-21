package wsHelpers

import (
	"errors"
	"fmt"
)

// Converts encoded indexes into a list of indexes
// For example, if the total list has 100 entries, send
// up 0, -1, 99 and the API will interpret this
// as needing indexes: 0, 1, 2, ... 98, 99
// Of course, multiple runs can be encoded, so 0, -1, 4, 50, -1, 55
// should mean 0, 1, 2, 3, 4, 50, 51, 52, 53, 54, 55
// NOTE: arraySize is the size of the array these indexes are
// reading from, so if any index is >= arraySize it's reading past
// the array bounds!

// Returns a list of unsigned indexes, and an error if:
// - A negative value is seen that is not -1
// - <start idx>, -1, <end idx which is <= start idx>
func MakeIndexList(encodedIndexes []int32, arraySize int) ([]uint32, error) {
	if len(encodedIndexes) <= 0 {
		return []uint32{}, nil
	}

	// Defining a range, fill the gap...
	if encodedIndexes[0] == -1 {
		// Can't have -1 at the start, we don't have the starting
		// number then!
		return nil, errors.New("indexes start with -1")
	} else if encodedIndexes[len(encodedIndexes)-1] == -1 {
		// Can't look ahead, we're at th end!
		return nil, errors.New("indexes end with -1")
	}

	result := []uint32{}
	for c, idx := range encodedIndexes {
		if idx == -1 {
			// Find the last value (noting it was already added!)
			startIdx := encodedIndexes[c-1]
			endIdx := encodedIndexes[c+1]

			if int(endIdx) >= arraySize {
				return nil, fmt.Errorf("index %v out of bounds: %v", uint32(endIdx), arraySize)
			}

			// Ensure there is a valid range between these numbers
			if endIdx <= startIdx+1 {
				return nil, fmt.Errorf("invalid range: %v->%v", startIdx, endIdx)
			}

			for iFill := startIdx + 1; iFill < endIdx; iFill++ {
				result = append(result, uint32(iFill))
			}
		} else if idx < -1 {
			return nil, fmt.Errorf("invalid index: %v", idx)
		} else {
			if int(idx) >= arraySize {
				return nil, fmt.Errorf("index %v out of bounds: %v", idx, arraySize)
			}
			result = append(result, uint32(idx))
		}
	}

	return result, nil
}
