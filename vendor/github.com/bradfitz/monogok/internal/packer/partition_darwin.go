package packer

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	DKIOCGETBLOCKCOUNT = 0x40086419
	DKIOCGETBLOCKSIZE  = 0x40046418
)

func deviceSize(fd uintptr) (uint64, error) {
	var (
		blocksize  uint32
		blockcount uint64
	)
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, DKIOCGETBLOCKSIZE, uintptr(unsafe.Pointer(&blocksize))); errno != 0 {
		return 0, errno
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, DKIOCGETBLOCKCOUNT, uintptr(unsafe.Pointer(&blockcount))); errno != 0 {
		return 0, errno
	}

	return uint64(blocksize) * blockcount, nil
}

func rereadPartitions(f *os.File) error {
	return nil // not needed on macOS
}
