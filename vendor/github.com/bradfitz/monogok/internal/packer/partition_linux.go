package packer

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func deviceSize(fd uintptr) (uint64, error) {
	var devsize uint64
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, unix.BLKGETSIZE64, uintptr(unsafe.Pointer(&devsize))); errno != 0 {
		return 0, errno
	}
	return devsize, nil
}

func rereadPartitions(f *os.File) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKRRPART, 0); errno != 0 {
		return errno
	}
	return nil
}
