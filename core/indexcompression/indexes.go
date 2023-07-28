package indexcompression

import (
	"errors"
	"fmt"
	"math"
	"sort"
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
func DecodeIndexList(encodedIndexes []int32, arraySize int) ([]uint32, error) {
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

// Given a list of unsigned indexes, this SORTS them and encodes them such that runs of consecutive
// numbers like 4, 5, 6, 7 are replaced with the sequence 4, -1, 7 - thereby reducing storage size
// NOTE: even though it takes unsigned int32s, it cannot support values over maxint, it just takes
// unsigned to signify that an array index can't be negative
func EncodeIndexList[T uint32 | int32](indexes []T) ([]int32, error) {
	if len(indexes) == 0 {
		return []int32{}, nil
	}

	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
	result := make([]int32, 0, len(indexes))
	incrCount := 0

	for c, idx := range indexes {
		if idx > math.MaxInt32 {
			return []int32{}, errors.New("index list had value > maxint")
		}
		if c == 0 { // First one is ALWAYS appended!
			result = append(result, int32(idx))
		} else {
			// Check if we're the last of a run
			diffPrev := idx - indexes[c-1]

			if diffPrev == 1 {
				incrCount++
			}

			// A wall for last value to pick up...
			diffNext := math.MaxInt32
			if c < len(indexes)-1 {
				diffNext = int(indexes[c+1] - idx)
			}

			if diffPrev <= 1 && diffNext > 1 {
				// We're the end of a run of incrementing numbers
				if incrCount > 1 {
					result = append(result, -1)
				}
				result = append(result, int32(idx))
				incrCount = 0
			} else if diffPrev > 1 {
				// Bigger leap than 1, so write this value, as it may
				// be the start of a run of incrementing numbers
				result = append(result, int32(idx))
			}
			/*
				// Check if it's an increment
				diff := idx - indexes[c-1]

				if c == len(indexes)-1 {
					// Last one, can't keep looping!
					if diff == 1 {
						incrCount++
					}

					endIncRun(&result, int32(idx), int32(indexes[c-1]), int32(diff), &incrCount)
				} else if diff > 0 {
					if diff == 1 {
						// Only do this if there are more coming...
						incrCount++
					} else {
						endIncRun(&result, int32(idx), int32(indexes[c-1]), int32(diff), &incrCount)
					}
				}
			*/
		}
	}
	return result, nil
}

/*
func endIncRun(result *[]int32, currIdx int32, prevIdx int32, idxDiff int32, incrCount *int) {
	if *incrCount > 1 {
		// Mark this as a continuation of the run
		*result = append(*result, -1)

		// Last one may have been the end of the run, check
		if idxDiff > 1 {
			*result = append(*result, prevIdx)
		}
	}

	// Write current one out
	*result = append(*result, currIdx)
	*incrCount = 0
}
*/
