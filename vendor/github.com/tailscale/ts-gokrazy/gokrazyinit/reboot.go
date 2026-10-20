//go:build !amd64
// +build !amd64

package gokrazy

import "golang.org/x/sys/unix"

type rebootOpts struct {
	tryKexec                 bool
	kexecMergeCurrentCmdline bool
}

func reboot(opts rebootOpts) error {
	return unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART)
}
