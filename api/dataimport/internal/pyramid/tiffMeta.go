package pyramid

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Implemented using: https://docs.fileformat.com/image/tiff/

func readTiffMeta(imagePath string) (map[string]string, error) {
	result := map[string]string{}

	f, err := os.OpenFile(imagePath, os.O_RDONLY, 0777)
	if err != nil {
		return result, err
	}

	header := make([]byte, 8)
	readCount, err := f.Read(header)
	if err != nil {
		return result, fmt.Errorf("failed to read tiff header for: %v, error: %v", imagePath, err)
	}

	if readCount != 8 {
		return result, fmt.Errorf("failed to read tiff header for: %v", imagePath)
	}

	// I for "Intel" vs M for "Motorola"
	if header[0] != 'I' && header[0] != 'M' || header[0] != header[1] {
		return result, fmt.Errorf("failed to read endian of tiff: %v", imagePath)
	}

	littleEndian := header[0] == 'I' && header[1] == 'I'

	// Read magic number
	magic := readUint16(littleEndian, header[2:4])

	// Must be 42 for normal tiff or 43 for "big tiff" (over 2^32 bytes in size)
	if magic != 42 && magic != 43 {
		return result, fmt.Errorf("invalid tiff header: %v", imagePath)
	}

	if magic == 43 {
		result, err = readBigTiff(f, littleEndian, header[4:])

	} else {
		result, err = readOGTiff(f, littleEndian, header[4:])
	}

	if err != nil {
		return result, fmt.Errorf("Error reading tiff %v: %v", imagePath, err)
	}

	return result, nil
}

func readBigTiff(f *os.File, isLittleEndian bool, header []byte) (map[string]string, error) {
	fmt.Printf("Reading BigTIF little_endian=%v...\n", isLittleEndian)

	result := map[string]string{}

	offsetSize := readUint16(isLittleEndian, header[0:2])
	if offsetSize != 8 {
		return result, fmt.Errorf("expected bigtiff offset size to be 8")
	}

	empty := readUint16(isLittleEndian, header[2:4])
	if empty != 0 {
		return result, fmt.Errorf("expected 0 after bigtiff offset")
	}

	offsetBuf := make([]byte, 8)
	sz, err := f.Read(offsetBuf)
	if err != nil {
		return result, err
	}
	if sz != 8 {
		return result, fmt.Errorf("failed to read IFD offset bytes")
	}

	//offset := readUint64(isLittleEndian, offsetBuf)

	return result, nil
}

func readOGTiff(f *os.File, isLittleEndian bool, header []byte) (map[string]string, error) {
	fmt.Printf("Reading TIF little_endian=%v...\n", isLittleEndian)

	result := map[string]string{}

	// We're reading an OG tif file, so the last 4 bytes of the 8-byte header are the offset of the first IFD
	offset := readUint32(isLittleEndian, header)

	_, err := f.Seek(int64(offset), 0)
	if err != nil {
		return result, err
	}

	for {
		// Read the IFD:
		// WORD NumEntries
		// List of tags
		// DWORD NextIFDOffset or 0 if no more are available
		numTags, err := readFileUint16(f, isLittleEndian)
		if err != nil {
			return result, err
		}

		for c := 0; c < int(numTags); c++ {
			// Read each 12 byte tag
			tag, err := readFileUint16(f, isLittleEndian)
			if err != nil {
				return result, err
			}
			dataType, err := readFileUint16(f, isLittleEndian)
			if err != nil {
				return result, err
			}
			dataSize, err := readFileUint32(f, isLittleEndian)
			if err != nil {
				return result, err
			}
			dataOffset, err := readFileUint32(f, isLittleEndian)
			if err != nil {
				return result, err
			}

			tagName := ""
			if n, ok := tagNames[TagType(tag)]; ok {
				tagName = n
			} else {
				tagName = fmt.Sprintf("TAG=%v", tag)
			}

			tagTypeName := ""
			if n, ok := tagDataType[DataType(dataType)]; ok {
				tagTypeName = n
			} else {
				tagTypeName = fmt.Sprintf("TYPE=%v", dataType)
			}

			fmt.Printf("IFD tag=%v (%v), type=%v (%v), count=%v, offset=%v\n", tagName, tag, tagTypeName, dataType, dataSize, dataOffset)

			if dataType != uint16(DataType_ASCII) {
				continue
			}

			// Jump and read the value
			_, err = f.Seek(int64(dataOffset), 0)
			if err != nil {
				return result, err
			}

			tagData, err := readFileBytes(f, int(dataSize))
			if err != nil {
				return result, err
			}

			// Strip the null
			tagData = tagData[0 : len(tagData)-1]
			fmt.Printf("  Data: %v\n", string(tagData))

			result[tagName] = string(tagData)

			// Hop back to read the next tag
			_, err = f.Seek(int64(offset)+12, 0)
			if err != nil {
				return result, err
			}
		}

		// At this point, we should find the offset to the next IFD
		nextIFDOffset, err := readFileUint32(f, isLittleEndian)
		if err != nil {
			return result, err
		}

		if nextIFDOffset == 0 {
			fmt.Println("End of IFDs")
			break
		}

		_, err = f.Seek(int64(nextIFDOffset), 0)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func readUint16(isLittleEndian bool, b []byte) uint16 {
	if isLittleEndian {
		return binary.LittleEndian.Uint16(b)
	}
	return binary.BigEndian.Uint16(b)
}

func readUint32(isLittleEndian bool, b []byte) uint32 {
	if isLittleEndian {
		return binary.LittleEndian.Uint32(b)
	}
	return binary.BigEndian.Uint32(b)
}

func readUint64(isLittleEndian bool, b []byte) uint64 {
	if isLittleEndian {
		return binary.LittleEndian.Uint64(b)
	}
	return binary.BigEndian.Uint64(b)
}

func readFileBytes(f *os.File, bytes int) ([]byte, error) {
	buf := make([]byte, bytes)
	n, err := f.Read(buf)
	if err != nil {
		return buf, err
	}
	if n != bytes {
		offset, _ := f.Seek(0, io.SeekCurrent)
		return buf, fmt.Errorf("failed to read %v bytes from file %v at offset %v", bytes, f.Name(), offset)
	}

	return buf, nil
}

func readFileUint16(f *os.File, isLittleEndian bool) (uint16, error) {
	buf, err := readFileBytes(f, 2)
	if err != nil {
		return 0, err
	}
	return readUint16(isLittleEndian, buf), nil
}

func readFileUint32(f *os.File, isLittleEndian bool) (uint32, error) {
	buf, err := readFileBytes(f, 4)
	if err != nil {
		return 0, err
	}
	return readUint32(isLittleEndian, buf), nil
}

func readFileUint64(f *os.File, isLittleEndian bool) (uint64, error) {
	buf, err := readFileBytes(f, 8)
	if err != nil {
		return 0, err
	}
	return readUint64(isLittleEndian, buf), nil
}
