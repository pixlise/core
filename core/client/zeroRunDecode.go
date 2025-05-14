package client

func zeroRunDecode(data []uint32) []uint32 {
	result := []uint32{}

	for c := 0; c < len(data); c++ {
		v := data[c]
		if v != 0 {
			// Just copy it across
			result = append(result, v)
		} else {
			// We found a 0, this is going to be followed by the number of 0's. Read ahead and fill that many
			// 0's in our result
			count := data[c+1]

			for i := uint32(0); i < count; i++ {
				result = append(result, 0)
			}

			// Skip over the count value next run
			c++
		}
	}

	return result
}
