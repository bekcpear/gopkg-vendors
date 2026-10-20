//go:build unix

package packer

import "syscall"

func setUmask() {
	syscall.Umask(0022)
}
