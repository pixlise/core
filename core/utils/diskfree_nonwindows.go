//go:build !windows

package utils

import (
	"os"

	"golang.org/x/sys/unix"
)

func GetDiskAvailableBytes() (uint64, error) {
	var stat unix.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0, err
	}

	unix.Statfs(wd, &stat)

	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize), nil
}
