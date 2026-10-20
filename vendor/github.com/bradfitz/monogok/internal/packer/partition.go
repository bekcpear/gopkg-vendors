// Package packer builds and deploys a gokrazy image.
package packer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"io"
	"log"
	"os"
	"unicode/utf16"
)

// PackerPack represents one pack process (partition-level).
type PackerPack struct {
	Partuuid       uint32
	UsePartuuid    bool
	UseGPTPartuuid bool
	UseGPT         bool
	ExistingEEPROM struct {
		PieepromSHA256 string
		VL805SHA256    string
	}
	FirstPartitionOffsetSectors int64
}

func NewPackForHost(firstPartitionOffsetSectors int64, hostname string) PackerPack {
	h := fnv.New32a()
	h.Write([]byte(hostname))
	return PackerPack{
		Partuuid:                    h.Sum32(),
		UsePartuuid:                 true,
		UseGPTPartuuid:              true,
		UseGPT:                      true,
		FirstPartitionOffsetSectors: firstPartitionOffsetSectors,
	}
}

func (p *PackerPack) ModifyCmdlineRoot() bool {
	return p.UsePartuuid || p.UseGPTPartuuid
}

func (p *PackerPack) GPTPARTUUID(partition uint16) string {
	const gokrazyGUIDPrefix = "60c24cc1-f3f9-427a-8199"
	return fmt.Sprintf("%s-%08x00%02x",
		gokrazyGUIDPrefix,
		p.Partuuid,
		partition)
}

func (p *PackerPack) Root() string {
	if p.UseGPTPartuuid {
		return fmt.Sprintf("PARTUUID=%s/PARTNROFF=1", p.GPTPARTUUID(1))
	}
	if p.UsePartuuid {
		return fmt.Sprintf("PARTUUID=%08x-02", p.Partuuid)
	}
	return ""
}

func (p *PackerPack) PermUUID() string {
	if p.UseGPTPartuuid {
		return p.GPTPARTUUID(4)
	}
	if p.UsePartuuid {
		return fmt.Sprintf("%08x-04", p.Partuuid)
	}
	return ""
}

var (
	active   = byte(0x80)
	inactive = byte(0x00)

	invalidCHS = [3]byte{0xFE, 0xFF, 0xFF}

	FAT      = byte(0xc)
	Linux    = byte(0x83)

	signature = uint16(0xAA55)
)

const MB = 1024 * 1024

func permSize(firstPartitionOffsetSectors int64, devsize uint64) uint32 {
	permStart := uint32(firstPartitionOffsetSectors + (1100 * MB / 512))
	permSz := uint32((devsize / 512) - uint64(firstPartitionOffsetSectors) - (1100 * MB / 512))
	lastAddressable := uint32((devsize / 512) - 1)
	if lastLBA := uint32(lastAddressable - 33); permStart+permSz >= lastLBA {
		permSz -= (permStart + permSz) - lastLBA
	}
	return permSz
}

func PermSizeInKB(firstPartitionOffsetSectors int64, devsize uint64) uint32 {
	permSizeLBA := permSize(firstPartitionOffsetSectors, devsize)
	permSizeBytes := permSizeLBA * 512
	return permSizeBytes / 1024
}

func writePartitionTable(firstPartitionOffsetSectors int64, w io.Writer) error {
	for _, v := range []interface{}{
		[446]byte{},

		active,
		invalidCHS,
		FAT,
		invalidCHS,
		uint32(firstPartitionOffsetSectors),
		uint32(100 * MB / 512),

		inactive,
		invalidCHS,
		byte(0xEE),
		invalidCHS,
		uint32(1),
		uint32(firstPartitionOffsetSectors - 1),

		[16]byte{},
		[16]byte{},

		signature,
	} {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}

	return nil
}

func writeMBRPartitionTable(firstPartitionOffsetSectors int64, w io.Writer, devsize uint64) error {
	for _, v := range []interface{}{
		[446]byte{},

		active,
		invalidCHS,
		FAT,
		invalidCHS,
		uint32(firstPartitionOffsetSectors),
		uint32(100 * MB / 512),

		inactive,
		invalidCHS,
		Linux,
		invalidCHS,
		uint32(firstPartitionOffsetSectors + 100*MB/512),
		uint32(500 * MB / 512),

		inactive,
		invalidCHS,
		Linux,
		invalidCHS,
		uint32(firstPartitionOffsetSectors + 600*MB/512),
		uint32(500 * MB / 512),

		inactive,
		invalidCHS,
		Linux,
		invalidCHS,
		uint32(firstPartitionOffsetSectors + 1100*MB/512),
		uint32(devsize/512 - uint64(firstPartitionOffsetSectors) - 1100*MB/512),

		signature,
	} {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}

	return nil
}

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

func partitionName(name string) [72]byte {
	r := make([]rune, 0, len(name))
	for _, s := range name {
		r = append(r, rune(s))
	}
	if len(r) > 36 {
		panic(fmt.Sprintf("Cannot use %s as partition name, has %d Unicode code units, maximum size is 36", name, len(r)))
	}
	nameb := utf16.Encode(r)
	var result [72]byte
	for i, u := range nameb {
		pos := i * 2
		binary.LittleEndian.PutUint16(result[pos:pos+2], u)
	}
	return result
}

func (p *PackerPack) writeGPT(w io.Writer, devsize uint64, primary bool) error {
	const (
		partitionTypeEFISystemPartition      = "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
		partitionTypeLinuxFilesystemData     = "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
		partitionTypeLinuxRootPartitionAMD64 = "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
		partitionTypeLinuxRootPartitionARM64 = "B921B045-1DF0-41C3-AF44-4C6F280D3FAE"
		partitionTypeLinuxRootPartitionX86   = "44479540-f297-41b2-9af7-d131d5f0458a"
	)

	type partitionEntry struct {
		TypeGUID   [16]byte
		GUID       [16]byte
		FirstLBA   uint64
		LastLBA    uint64
		Attributes uint64
		Name       [72]byte
	}
	partition0First := uint64(p.FirstPartitionOffsetSectors)
	partition0Last := partition0First + (100 * MB / 512) - 1

	partition1First := partition0Last + 1
	partition1Last := partition1First + (500 * MB / 512) - 1

	partition2First := partition1Last + 1
	partition2Last := partition2First + (500 * MB / 512) - 1

	partition3First := partition2Last + 1
	partition3Last := partition3First + uint64(permSize(p.FirstPartitionOffsetSectors, devsize)) - 1

	var rootType [16]byte
	switch os.Getenv("GOARCH") {
	case "386":
		rootType = mustParseGUID(partitionTypeLinuxRootPartitionX86)
	case "amd64":
		rootType = mustParseGUID(partitionTypeLinuxRootPartitionAMD64)
	default:
		rootType = mustParseGUID(partitionTypeLinuxRootPartitionARM64)
	}

	partitionEntries := []partitionEntry{
		{
			TypeGUID:   mustParseGUID(partitionTypeEFISystemPartition),
			GUID:       mustParseGUID(p.GPTPARTUUID(1)),
			FirstLBA:   partition0First,
			LastLBA:    partition0Last,
			Attributes: 0,
			Name:       partitionName("Microsoft basic data"),
		},
		{
			TypeGUID:   rootType,
			GUID:       mustParseGUID(p.GPTPARTUUID(2)),
			FirstLBA:   partition1First,
			LastLBA:    partition1Last,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
		{
			TypeGUID:   mustParseGUID(partitionTypeLinuxFilesystemData),
			GUID:       mustParseGUID(p.GPTPARTUUID(3)),
			FirstLBA:   partition2First,
			LastLBA:    partition2Last,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
		{
			TypeGUID:   mustParseGUID(partitionTypeLinuxFilesystemData),
			GUID:       mustParseGUID(p.GPTPARTUUID(4)),
			FirstLBA:   partition3First,
			LastLBA:    partition3Last,
			Attributes: 0,
			Name:       partitionName("Linux filesystem"),
		},
	}
	var pbuf bytes.Buffer
	if err := binary.Write(&pbuf, binary.LittleEndian, partitionEntries); err != nil {
		return err
	}
	if _, err := pbuf.Write(bytes.Repeat([]byte{0}, (128-len(partitionEntries))*128)); err != nil {
		return err
	}
	entriesChecksum := crc32.ChecksumIEEE(pbuf.Bytes())

	lastAddressable := (devsize / 512) - 1
	currentLBA := uint64(1)
	backupLBA := lastAddressable
	entriesStart := uint64(2)
	if !primary {
		currentLBA = backupLBA
		entriesStart = backupLBA - 32
		backupLBA = 1
	}

	partitionHeader := struct {
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
		DiskGUID:       mustParseGUID(p.GPTPARTUUID(0)),
		EntriesStart:   entriesStart,
		EntriesCount:   128,
		EntriesSize:    128,
		CRC32Array:     entriesChecksum,
	}
	var hbuf bytes.Buffer
	if err := binary.Write(&hbuf, binary.LittleEndian, partitionHeader); err != nil {
		return err
	}
	if got, want := hbuf.Len(), int(partitionHeader.HeaderSize); got != want {
		return fmt.Errorf("BUG: header size: got %d, want %d", got, want)
	}
	partitionHeader.CRC32Header = crc32.ChecksumIEEE(hbuf.Bytes())

	if !primary {
		if _, err := io.Copy(w, &pbuf); err != nil {
			return err
		}
	}

	if err := binary.Write(w, binary.LittleEndian, partitionHeader); err != nil {
		return err
	}

	for _, v := range []interface{}{
		[420]byte{},
	} {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}

	if primary {
		if _, err := io.Copy(w, &pbuf); err != nil {
			return err
		}
	}

	return nil
}

func (p *PackerPack) Partition(o *os.File, devsize uint64) error {
	minsize := uint64(1100 * MB)
	if devsize < minsize {
		return fmt.Errorf("device is too small (at least %d MB needed, %d MB available)", minsize/MB, devsize/MB)
	}
	if !p.UseGPT {
		return writeMBRPartitionTable(p.FirstPartitionOffsetSectors, o, devsize)
	}

	if err := writePartitionTable(p.FirstPartitionOffsetSectors, o); err != nil {
		return err
	}

	if err := p.writeGPT(o, devsize, true); err != nil {
		return err
	}

	lastAddressable := (devsize / 512) - 1
	lbaMinus33 := lastAddressable - 32

	if _, err := o.Seek(int64(lbaMinus33*512), io.SeekStart); err != nil {
		return err
	}

	if err := p.writeGPT(o, devsize, false); err != nil {
		return err
	}
	return nil
}

func (p *PackerPack) RereadPartitions(o *os.File) error {
	if err := rereadPartitions(o); err != nil {
		log.Printf("Re-reading partition table failed: %v. Remember to unplug and re-plug the SD card before creating a file system for persistent data, if desired.", err)
	}
	return nil
}
