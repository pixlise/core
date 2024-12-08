//go:build !windows

package utils

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func GetDiskAvailableBytes() (uint64, error) {
	var stat unix.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0, err
	}

	err = unix.Statfs(wd, &stat)
	if err != nil {
		return 0, err
	}

	fmt.Sprintf("Bavail: %v", stat.Bavail)
	fmt.Sprintf("Bfree: %v", stat.Bfree)
	fmt.Sprintf("Blocks: %v", stat.Blocks)
	fmt.Sprintf("Bsize: %v", stat.Bsize)
	fmt.Sprintf("Ffree: %v", stat.Ffree)
	fmt.Sprintf("Files: %v", stat.Files)
	fmt.Sprintf("Flags: %v", stat.Flags)
	fmt.Sprintf("Flags_ext: %v", stat.Flags_ext)
	fmt.Sprintf("Fsid: %v", stat.Fsid.Val)
	fmt.Sprintf("Fssubtype: %v", stat.Fssubtype)
	fmt.Sprintf("Fstypename: %v", stat.Fstypename)
	fmt.Sprintf("Iosize: %v", stat.Iosize)
	fmt.Sprintf("Mntfromname: %v", stat.Mntfromname)
	fmt.Sprintf("Mntonname: %v", stat.Mntonname)
	fmt.Sprintf("Owner: %v", stat.Owner)
	fmt.Sprintf("Type: %v", stat.Type)

	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize), nil
}
