package packer

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bradfitz/monogok/internal/build"
)

func kernelGoarch(hdr []byte) string {
	const (
		arm32Magic       = 0x016f2818
		arm32MagicOffset = 0x24
		arm64Magic       = 0x644d5241
		arm64MagicOffset = 0x38
		x86Magic            = 0xaa55
		x86MagicOffset      = 0x1fe
		x86XloadflagsOffset = 0x236
	)
	if len(hdr) >= arm64MagicOffset+4 && binary.LittleEndian.Uint32(hdr[arm64MagicOffset:]) == arm64Magic {
		return "arm64"
	}
	if len(hdr) >= arm32MagicOffset+4 && binary.LittleEndian.Uint32(hdr[arm32MagicOffset:]) == arm32Magic {
		return "arm"
	}
	if len(hdr) >= x86XloadflagsOffset+2 && binary.LittleEndian.Uint16(hdr[x86MagicOffset:]) == x86Magic {
		if hdr[x86XloadflagsOffset]&1 != 0 {
			return "amd64"
		} else {
			return "386"
		}
	}
	return ""
}

func (pack *Pack) validateTargetArchMatchesKernel() error {
	cfg := pack.Cfg
	kernelDir, err := pack.PackageDir(cfg.KernelPackageOrDefault())
	if err != nil {
		return err
	}
	kernelPath := filepath.Join(kernelDir, "vmlinuz")
	k, err := os.Open(kernelPath)
	if err != nil {
		return err
	}
	defer k.Close()
	hdr := make([]byte, 1<<10)
	if _, err := io.ReadFull(k, hdr); err != nil {
		return err
	}
	kernelArch := kernelGoarch(hdr)
	if kernelArch == "" {
		return fmt.Errorf("kernel %v architecture in %s not detected", cfg.KernelPackageOrDefault(), kernelPath)
	}
	targetArch := build.TargetArch()
	if kernelArch != targetArch {
		return fmt.Errorf("target architecture %q (GOARCH) doesn't match the %s kernel type %q",
			targetArch,
			cfg.KernelPackageOrDefault(),
			kernelArch)
	}
	return nil
}
