/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2022 WireGuard LLC. All Rights Reserved.
 */

package tun

import (
	"os"
)

type Event int

const (
	EventUp = 1 << iota
	EventDown
	EventMTUUpdate
)

type Device interface {
	// File returns the file descriptor of the device.
	File() *os.File

	// Read one or more packets from the Device (without any additional headers).
	// On a successful read it returns the number of packets read, and sets
	// packet lengths within the sizes slice. len(sizes) must be >= len(buffs).
	// A nonzero offset can be used to instruct the Device on where to begin
	// reading into each element of the buffs slice.
	Read(buffs [][]byte, sizes []int, offset int) (n int, err error)

	// Write one or more packets to the device (without any additional headers).
	// On a successful write it returns the number of packets written. A nonzero
	// offset can be used to instruct the Device on where to begin writing from
	// each packet contained within the buffs slice.
	Write(buffs [][]byte, offset int) (int, error)

	// MTU returns the MTU of the Device.
	MTU() (int, error)

	// Name returns the current name of the Device.
	Name() (string, error)

	// Events returns a channel of type Event, which is fed Device events.
	Events() <-chan Event

	// Close stops the Device and closes the Event channel.
	Close() error

	// BatchSize returns the preferred/max number of packets that can be read or
	// written in a single read/write call. If offloading is disabled, the batch
	// size will be 1. BatchSize must not change over the lifetime of a Device,
	// except to DisableOffload() (if Device is a DisableOffloader) prior to
	// read/write operations commencing.
	BatchSize() int
}

// DisableOffloader is a type that may be supported by Device implementations if
// they support offloading, and support offloading being disabled.
type DisableOffloader interface {
	DisableOffload() error
}
