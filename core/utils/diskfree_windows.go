package utils

import "golang.org/x/sys/windows"

func GetDiskAvailableBytes() (uint64, error) {
	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr("C:"),
		&freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, err
	}
	return freeBytesAvailable, nil
}
