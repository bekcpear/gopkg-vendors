// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package disklayout

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"unicode/utf16"
)

// GokrazyHostname is the default hostname monogok uses when building
// the Tailscale appliance image. Embedded GAFs from pkgs.tailscale.com
// are built with this hostname.
const GokrazyHostname = "tsapp"

// gokrazyGUIDPrefix is the fixed first three groups of every gokrazy
// partition's GPT GUID. Its last group is built from the per-disk
// partuuid and the partition number.
const gokrazyGUIDPrefix = "60c24cc1-f3f9-427a-8199"

// GPT partition type GUIDs.
const (
	gptTypeEFISystem            = "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
	gptTypeLinuxFilesystemData  = "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
	gptTypeLinuxRootPartAMD64   = "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
	gptTypeLinuxRootPartARM64   = "B921B045-1DF0-41C3-AF44-4C6F280D3FAE"
	gptTypeLinuxRootPart386     = "44479540-f297-41b2-9af7-d131d5f0458a"
	gptTypeMicrosoftBasicData   = "EBD0A0A2-B9E5-4433-87C0-68B6B72699C7" // unused; for reference
	gptTypeProtectiveMBREntries = "00000000-0000-0000-0000-000000000000" // unused; for reference
)

// GokrazyPartUUID returns the per-disk partuuid for a gokrazy install
// with the given hostname. It is the FNV-1a 32-bit hash of hostname,
// matching monogok's NewPackForHost.
func GokrazyPartUUID(hostname string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(hostname))
	return h.Sum32()
}

// GokrazyPartitionGUID returns the GPT partition GUID for partition n
// (1-based) on a gokrazy install with the given partuuid. It matches
// monogok's PackerPack.GPTPARTUUID.
func GokrazyPartitionGUID(partUUID uint32, n uint16) string {
	return fmt.Sprintf("%s-%08x00%02x", gokrazyGUIDPrefix, partUUID, n)
}

// ParseCmdlinePartUUID extracts the per-disk partuuid from a gokrazy
// cmdline.txt's root=PARTUUID=...  parameter. cmdline is the full
// cmdline.txt content.
//
// gokrazy cmdlines look like:
//
//	root=PARTUUID=60c24cc1-f3f9-427a-8199-XXXXXXXX00YY/PARTNROFF=1
//
// where XXXXXXXX is the partuuid we want.
func ParseCmdlinePartUUID(cmdline string) (uint32, error) {
	i := strings.Index(cmdline, gokrazyGUIDPrefix+"-")
	if i < 0 {
		return 0, fmt.Errorf("no gokrazy PARTUUID found in cmdline")
	}
	rest := cmdline[i+len(gokrazyGUIDPrefix)+1:]
	if len(rest) < 8 {
		return 0, fmt.Errorf("truncated gokrazy PARTUUID in cmdline")
	}
	v, err := strconv.ParseUint(rest[:8], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid gokrazy PARTUUID %q: %w", rest[:8], err)
	}
	return uint32(v), nil
}

// RootArch is the architecture used to pick the GPT root partition type
// GUID. Use the constants Arch386, ArchAMD64, ArchARM64.
type RootArch string

const (
	Arch386   RootArch = "386"
	ArchAMD64 RootArch = "amd64"
	ArchARM64 RootArch = "arm64"
)

func (a RootArch) typeGUID() string {
	switch a {
	case Arch386:
		return gptTypeLinuxRootPart386
	case ArchAMD64:
		return gptTypeLinuxRootPartAMD64
	default:
		// arm64 (the default for gokrazy's pi builds) and anything else.
		return gptTypeLinuxRootPartARM64
	}
}

// WriteGPT writes the protective MBR (with bootCode in the first 446
// bytes), the primary GPT, and the secondary GPT to f for a fresh
// gokrazy disk install of devsize bytes whose first partition starts
// at firstLBA. partUUID is the per-disk partuuid (see
// GokrazyPartUUID); rootArch picks the root partition type GUID.
//
// WriteGPT does NOT write the boot.img, root.img, or perm partition
// contents; the caller writes those separately at the offsets returned
// by BootStartLBA, RootAStartLBA, etc.
//
// The mbr.img member of a monogok-built GAF is the appropriate
// bootCode (446 bytes). If bootCode is shorter, it is zero-padded; if
// longer, an error is returned.
//
// Each Write call to f is a multiple of 512 bytes so raw block devices
// that require sector-aligned writes (e.g. macOS /dev/rdiskN) work.
func WriteGPT(f io.WriteSeeker, devsize uint64, firstLBA int64, bootCode []byte, partUUID uint32, rootArch RootArch) error {
	if len(bootCode) > 446 {
		return fmt.Errorf("bootCode is %d bytes; expected at most 446", len(bootCode))
	}
	if min := uint64(MinDiskMB) * mb; devsize < min {
		return fmt.Errorf("device size %d bytes is below the gokrazy minimum of %d MB", devsize, MinDiskMB)
	}

	// LBA 0: protective MBR sector.
	mbr := buildProtectiveMBR(bootCode, firstLBA)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write(mbr[:]); err != nil {
		return err
	}

	// LBA 1..33: primary GPT (header + 128 entries).
	primary, err := buildGPTArea(devsize, firstLBA, partUUID, rootArch, true)
	if err != nil {
		return err
	}
	if _, err := f.Seek(int64(sectorSize), io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write(primary); err != nil {
		return err
	}

	// Last LBA - 32 .. last LBA: secondary GPT (entries + header).
	secondary, err := buildGPTArea(devsize, firstLBA, partUUID, rootArch, false)
	if err != nil {
		return err
	}
	tailLBA := int64(devsize/sectorSize) - 33
	if _, err := f.Seek(tailLBA*int64(sectorSize), io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write(secondary); err != nil {
		return err
	}
	return nil
}

// buildProtectiveMBR returns the 512-byte protective MBR sector for a
// GPT-partitioned disk. The first 446 bytes are bootCode (zero-padded
// to 446), the next 64 bytes are a protective MBR partition table
// (one 0xEE entry covering the disk minus the EFI partition window),
// and the last 2 bytes are the 0xAA55 signature.
//
// This matches monogok's writePartitionTable (the GPT path).
func buildProtectiveMBR(bootCode []byte, firstLBA int64) [512]byte {
	var mbr [512]byte
	copy(mbr[:446], bootCode)

	var buf [66]byte
	w := newByteWriter(buf[:0])
	for _, v := range []any{
		mbrActive,
		mbrInvalidCHS,
		mbrPartitionFAT,
		mbrInvalidCHS,
		uint32(firstLBA),
		uint32(BootPartitionSizeMB * mb / sectorSize),

		mbrInactive,
		mbrInvalidCHS,
		byte(0xEE), // protective MBR
		mbrInvalidCHS,
		uint32(1),
		uint32(firstLBA - 1),

		[16]byte{},
		[16]byte{},

		mbrSignature,
	} {
		_ = binary.Write(w, binary.LittleEndian, v)
	}
	copy(mbr[446:], w.Bytes())
	return mbr
}

// buildGPTArea returns the bytes of either the primary GPT (header +
// 420 zeros padding + 32 sectors of entries = 33 sectors) or the
// secondary GPT (entries + header + padding = 33 sectors), depending
// on primary.
//
// Returning a sector-aligned []byte (rather than streaming individual
// fields via binary.Write) lets the caller perform a single
// sector-aligned write to raw block devices that require it (e.g.
// macOS /dev/rdiskN).
//
// This is a port of monogok's PackerPack.writeGPT with the GOARCH env
// dependency replaced by an explicit rootArch parameter and
// FirstPartitionOffsetSectors replaced by firstLBA.
func buildGPTArea(devsize uint64, firstLBA int64, partUUID uint32, rootArch RootArch, primary bool) ([]byte, error) {
	type partitionEntry struct {
		TypeGUID   [16]byte
		GUID       [16]byte
		FirstLBA   uint64
		LastLBA    uint64
		Attributes uint64
		Name       [72]byte
	}

	bootFirst := uint64(firstLBA)
	bootLast := bootFirst + (BootPartitionSizeMB * mb / sectorSize) - 1

	rootAFirst := bootLast + 1
	rootALast := rootAFirst + (RootPartitionSizeMB * mb / sectorSize) - 1

	rootBFirst := rootALast + 1
	rootBLast := rootBFirst + (RootPartitionSizeMB * mb / sectorSize) - 1

	permFirst := rootBLast + 1
	permLast := permFirst + uint64(permSizeClampedForGPT(firstLBA, devsize)) - 1

	rootTypeGUID := mustParseGUID(rootArch.typeGUID())
	entries := []partitionEntry{
		{
			TypeGUID:   mustParseGUID(gptTypeEFISystem),
			GUID:       mustParseGUID(GokrazyPartitionGUID(partUUID, 1)),
			FirstLBA:   bootFirst,
			LastLBA:    bootLast,
			Attributes: 0,
			Name:       partitionName("Microsoft basic data"),
		},
		{
			TypeGUID:   rootTypeGUID,
			GUID:       mustParseGUID(GokrazyPartitionGUID(partUUID, 2)),
			FirstLBA:   rootAFirst,
			LastLBA:    rootALast,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
		{
			TypeGUID:   mustParseGUID(gptTypeLinuxFilesystemData),
			GUID:       mustParseGUID(GokrazyPartitionGUID(partUUID, 3)),
			FirstLBA:   rootBFirst,
			LastLBA:    rootBLast,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
		{
			TypeGUID:   mustParseGUID(gptTypeLinuxFilesystemData),
			GUID:       mustParseGUID(GokrazyPartitionGUID(partUUID, 4)),
			FirstLBA:   permFirst,
			LastLBA:    permLast,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
	}

	var pbuf bytes.Buffer
	if err := binary.Write(&pbuf, binary.LittleEndian, entries); err != nil {
		return nil, err
	}
	if _, err := pbuf.Write(bytes.Repeat([]byte{0}, (128-len(entries))*128)); err != nil {
		return nil, err
	}
	entriesChecksum := crc32.ChecksumIEEE(pbuf.Bytes())

	lastAddressable := (devsize / sectorSize) - 1
	currentLBA := uint64(1)
	backupLBA := lastAddressable
	entriesStart := uint64(2)
	if !primary {
		currentLBA = backupLBA
		entriesStart = backupLBA - 32
		backupLBA = 1
	}

	header := struct {
		Signature      [8]byte
		Revision       uint32
		HeaderSize     uint32
		CRC32Header    uint32
		Reserved       uint32
		CurrentLBA     uint64
		BackupLBA      uint64
		FirstUsableLBA uint64
		LastUsableLBA  uint64
		DiskGUID       [16]byte
		EntriesStart   uint64
		EntriesCount   uint32
		EntriesSize    uint32
		CRC32Array     uint32
	}{
		Signature:      [8]byte{0x45, 0x46, 0x49, 0x20, 0x50, 0x41, 0x52, 0x54},
		Revision:       0x00010000,
		HeaderSize:     92,
		CurrentLBA:     currentLBA,
		BackupLBA:      backupLBA,
		FirstUsableLBA: 34,
		LastUsableLBA:  lastAddressable - 32 - 1,
		DiskGUID:       mustParseGUID(GokrazyPartitionGUID(partUUID, 0)),
		EntriesStart:   entriesStart,
		EntriesCount:   128,
		EntriesSize:    128,
		CRC32Array:     entriesChecksum,
	}
	var hbuf bytes.Buffer
	if err := binary.Write(&hbuf, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	header.CRC32Header = crc32.ChecksumIEEE(hbuf.Bytes())

	// 33 sectors total: header (92) + padding (420) + entries (32*512).
	out := make([]byte, 0, 33*sectorSize)
	w := newByteWriter(out)
	if !primary {
		w.Write(pbuf.Bytes())
	}
	if err := binary.Write(w, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, [420]byte{}); err != nil {
		return nil, err
	}
	if primary {
		w.Write(pbuf.Bytes())
	}
	if len(w.Bytes()) != 33*sectorSize {
		return nil, fmt.Errorf("BUG: GPT area is %d bytes; want %d", len(w.Bytes()), 33*sectorSize)
	}
	return w.Bytes(), nil
}

// permSizeClampedForGPT returns the perm partition size in sectors,
// clamped to leave 33 trailing sectors at the end of the disk for the
// secondary GPT header and entries (which the GPT primary header
// reserves via FirstUsableLBA / LastUsableLBA). This matches what
// monogok's writeGPT writes into the perm partition entry.
func permSizeClampedForGPT(firstLBA int64, devsize uint64) uint32 {
	permStart := PermStartLBA(firstLBA)
	totalLBA := uint32(devsize / sectorSize)
	if totalLBA <= permStart {
		return 0
	}
	permSz := totalLBA - permStart
	lastAddressable := totalLBA - 1
	if lastLBA := lastAddressable - 33; permStart+permSz >= lastLBA {
		permSz -= (permStart + permSz) - lastLBA
	}
	return permSz
}

// mustParseGUID parses a GUID string of the form
// "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX" into the 16-byte little-endian
// representation used in GPT entries.
func mustParseGUID(guid string) [16]byte {
	var (
		timeLow                 uint32
		timeMid                 uint16
		timeHighAndVersion      uint16
		clockSeqHighAndReserved uint8
		clockSeqLow             uint8
		node                    []byte
	)
	_, err := fmt.Sscanf(guid,
		"%08x-%04x-%04x-%02x%02x-%012x",
		&timeLow,
		&timeMid,
		&timeHighAndVersion,
		&clockSeqHighAndReserved,
		&clockSeqLow,
		&node)
	if err != nil {
		panic(err)
	}
	var result [16]byte
	binary.LittleEndian.PutUint32(result[0:4], timeLow)
	binary.LittleEndian.PutUint16(result[4:6], timeMid)
	binary.LittleEndian.PutUint16(result[6:8], timeHighAndVersion)
	result[8] = clockSeqHighAndReserved
	result[9] = clockSeqLow
	copy(result[10:], node)
	return result
}

// partitionName converts a partition name to its 72-byte UTF-16 little-
// endian representation. Names longer than 36 code units cause a panic.
func partitionName(name string) [72]byte {
	r := make([]rune, 0, len(name))
	for _, s := range name {
		r = append(r, rune(s))
	}
	if len(r) > 36 {
		panic(fmt.Sprintf("partition name %q has %d code units; max 36", name, len(r)))
	}
	nameb := utf16.Encode(r)
	var result [72]byte
	for i, u := range nameb {
		pos := i * 2
		binary.LittleEndian.PutUint16(result[pos:pos+2], u)
	}
	return result
}
