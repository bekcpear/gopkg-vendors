//go:build !darwin && !linux
// +build !darwin,!linux

package core

import (
	"syscall"
)

// WARNING: This context is used both by net.Dialer and net.Listen in tcp.go

func (t *linkTCP) tcpContext(network, address string, c syscall.RawConn) error {
	return nil
}

func (t *linkTCP) getControl(sintf string) func(string, string, syscall.RawConn) error {
	return t.tcpContext
}
