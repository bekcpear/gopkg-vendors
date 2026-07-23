// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package disklayout describes the gokrazy on-disk partition layout and
// provides helpers for writing it. It is intended for use by tools that
// install a gokrazy image onto a fresh disk (e.g. SD card flashing
// tools), so they can lay out the partition table the same way monogok
// does when it produces a full disk image.
package disklayout

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// DefaultBootPartitionStartLBA is the standard LBA at which gokrazy
// places the first (boot) partition. 8192 sectors of 512 bytes = 4 MiB,
// the recommended alignment for SD cards.
const DefaultBootPartitionStartLBA int64 = 8192

const (
	mb         = 1024 * 1024
	sectorSize = 512

	// BootPartitionSizeMB is the gokrazy boot (FAT) partition size.
	BootPartitionSizeMB = 100
	// RootPartitionSizeMB is the size of each of the two gokrazy root
	// (squashfs) partitions; the two are sized identically so OTA
	// updates can swap between them.
	RootPartitionSizeMB = 500

	// MinDiskMB is the smallest device size in megabytes that can hold
	// a gokrazy install with one byte of perm space, given
	// DefaultBootPartitionStartLBA.
	MinDiskMB = (DefaultBootPartitionStartLBA*sectorSize)/mb + BootPartitionSizeMB + 2*RootPartitionSizeMB + 1
)

const (
	mbrPartitionFAT   = byte(0x0c)
	mbrPartitionLinux = byte(0x83)

	mbrActive   = byte(0x80)
	mbrInactive = byte(0x00)

	mbrSignature = uint16(0xAA55)
)

var mbrInvalidCHS = [3]byte{0xFE, 0xFF, 0xFF}

// BootStartLBA returns the LBA where the boot partition starts.
func BootStartLBA(firstLBA int64) uint32 {
	return uint32(firstLBA)
}

// RootAStartLBA returns the LBA where the root A partition starts.
func RootAStartLBA(firstLBA int64) uint32 {
	return uint32(firstLBA + BootPartitionSizeMB*mb/sectorSize)
}

// RootBStartLBA returns the LBA where the root B partition starts.
func RootBStartLBA(firstLBA int64) uint32 {
	return uint32(firstLBA + (BootPartitionSizeMB+RootPartitionSizeMB)*mb/sectorSize)
}

// PermStartLBA returns the LBA where the perm (writable, ext4) partition
// starts.
func PermStartLBA(firstLBA int64) uint32 {
	return uint32(firstLBA + (BootPartitionSizeMB+2*RootPartitionSizeMB)*mb/sectorSize)
}

// PermSize returns the perm partition size in sectors on a disk of
// devsize bytes whose first partition starts at firstLBA. It is the
// unclamped "fill the rest of the disk" size, matching what monogok
// writes when building a full MBR image. Callers that need to reserve
// trailing space (e.g. for a secondary GPT) must subtract it themselves.
func PermSize(firstLBA int64, devsize uint64) uint32 {
	permStart := PermStartLBA(firstLBA)
	totalLBA := uint32(devsize / sectorSize)
	if totalLBA <= permStart {
		return 0
	}
	return totalLBA - permStart
}

// WritePartitionTable writes the 64-byte gokrazy MBR primary partition
// table to w (not including the 446-byte boot code prefix or the 2-byte
// 0xAA55 signature). devsize is the total disk size in bytes.
//
// firstLBA is typically DefaultBootPartitionStartLBA. devsize must be
// large enough to hold the boot + both root partitions + at least one
// perm sector, or an error is returned.
func WritePartitionTable(w io.Writer, firstLBA int64, devsize uint64) error {
	if min := uint64(MinDiskMB) * mb; devsize < min {
		return fmt.Errorf("device size %d bytes (%.1f MB) is below the gokrazy minimum of %d MB", devsize, float64(devsize)/mb, MinDiskMB)
	}
	for _, v := range []any{
		mbrActive,
		mbrInvalidCHS,
		mbrPartitionFAT,
		mbrInvalidCHS,
		BootStartLBA(firstLBA),
		uint32(BootPartitionSizeMB * mb / sectorSize),

		mbrInactive,
		mbrInvalidCHS,
		mbrPartitionLinux,
		mbrInvalidCHS,
		RootAStartLBA(firstLBA),
		uint32(RootPartitionSizeMB * mb / sectorSize),

		mbrInactive,
		mbrInvalidCHS,
		mbrPartitionLinux,
		mbrInvalidCHS,
		RootBStartLBA(firstLBA),
		uint32(RootPartitionSizeMB * mb / sectorSize),

		mbrInactive,
		mbrInvalidCHS,
		mbrPartitionLinux,
		mbrInvalidCHS,
		PermStartLBA(firstLBA),
		PermSize(firstLBA, devsize),
	} {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}

// BuildMBR returns a complete 512-byte MBR boot sector for a fresh
// gokrazy disk install: 446 bytes of bootCode (zero-padded if shorter;
// rejected if longer), followed by the 64-byte gokrazy partition table,
// followed by the 0xAA55 boot signature.
func BuildMBR(bootCode []byte, firstLBA int64, devsize uint64) ([512]byte, error) {
	var mbr [512]byte
	if len(bootCode) > 446 {
		return mbr, errors.New("boot code is longer than 446 bytes")
	}
	copy(mbr[:446], bootCode)

	var buf [66]byte
	bw := newByteWriter(buf[:0])
	if err := WritePartitionTable(bw, firstLBA, devsize); err != nil {
		return mbr, err
	}
	if err := binary.Write(bw, binary.LittleEndian, mbrSignature); err != nil {
		return mbr, err
	}
	copy(mbr[446:], bw.Bytes())
	return mbr, nil
}

// byteWriter is a tiny io.Writer that appends to a backing slice without
// allocating, used to keep BuildMBR allocation-free.
type byteWriter struct{ b []byte }

func newByteWriter(b []byte) *byteWriter { return &byteWriter{b: b} }
func (w *byteWriter) Bytes() []byte      { return w.b }
func (w *byteWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
