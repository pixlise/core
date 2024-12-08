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

	fmt.Printf("Bavail: %v\n", stat.Bavail)
	fmt.Printf("Bfree: %v\n", stat.Bfree)
	fmt.Printf("Blocks: %v\n", stat.Blocks)
	fmt.Printf("Bsize: %v\n", stat.Bsize)
	fmt.Printf("Ffree: %v\n", stat.Ffree)
	fmt.Printf("Files: %v\n", stat.Files)
	fmt.Printf("Flags: %v\n", stat.Flags)
	fmt.Printf("Flags_ext: %v\n", stat.Flags_ext)
	fmt.Printf("Fsid: %v\n", stat.Fsid.Val)
	fmt.Printf("Fssubtype: %v\n", stat.Fssubtype)
	fmt.Printf("Fstypename: %v\n", stat.Fstypename)
	fmt.Printf("Iosize: %v\n", stat.Iosize)
	fmt.Printf("Mntfromname: %v\n", stat.Mntfromname)
	fmt.Printf("Mntonname: %v\n", stat.Mntonname)
	fmt.Printf("Owner: %v\n", stat.Owner)
	fmt.Printf("Type: %v\n", stat.Type)

	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize), nil
}
