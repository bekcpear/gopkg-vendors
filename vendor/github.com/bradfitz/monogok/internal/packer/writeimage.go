package packer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"

	"github.com/bradfitz/monogok/internal/config"
	"github.com/bradfitz/monogok/internal/deviceconfig"
	"github.com/bradfitz/monogok/internal/updateflag"
)

func (pack *Pack) logicWrite(dnsCheck chan error) error {
	log := pack.Env.Logger()

	newInstallation := pack.Cfg.InternalCompatibilityFlags.Update == ""

	log.Printf("")
	log.Printf("Feature summary:")
	log.Printf("  use GPT: %v", pack.Pack.UseGPT)
	log.Printf("  use PARTUUID: %v", pack.Pack.UsePartuuid)
	log.Printf("  use GPT PARTUUID: %v", pack.Pack.UseGPTPartuuid)

	cfg := pack.Cfg
	root := pack.root
	var (
		isDev                    bool
		tmpBoot, tmpRoot, tmpMBR *os.File
	)
	switch {
	case cfg.InternalCompatibilityFlags.Overwrite != "" ||
		(pack.Output != nil && pack.Output.Type == OutputTypeFull && pack.Output.Path != ""):

		st, err := os.Stat(cfg.InternalCompatibilityFlags.Overwrite)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		isDev = err == nil && st.Mode()&os.ModeDevice == os.ModeDevice

		if isDev {
			if err := pack.overwriteDevice(cfg.InternalCompatibilityFlags.Overwrite, root, pack.rootDeviceFiles); err != nil {
				return err
			}
			log.Printf("To boot gokrazy, plug the SD card into a supported device (see https://gokrazy.org/platforms/)")
			log.Printf("")
		} else {
			lower := 1200*MB + int(pack.firstPartitionOffsetSectors)

			if cfg.InternalCompatibilityFlags.TargetStorageBytes == 0 {
				return fmt.Errorf("--target_storage_bytes is required (e.g. --target_storage_bytes=%d) when using overwrite with a file", lower)
			}
			if cfg.InternalCompatibilityFlags.TargetStorageBytes%512 != 0 {
				return fmt.Errorf("--target_storage_bytes must be a multiple of 512 (sector size), use e.g. %d", lower)
			}
			if cfg.InternalCompatibilityFlags.TargetStorageBytes < lower {
				return fmt.Errorf("--target_storage_bytes must be at least %d (for boot + 2 root file systems + 100 MB /perm)", lower)
			}

			if _, _, err := pack.overwriteFile(root, pack.rootDeviceFiles, pack.firstPartitionOffsetSectors); err != nil {
				return err
			}

			log.Printf("To boot gokrazy, copy %s to an SD card and plug it into a supported device (see https://gokrazy.org/platforms/)", cfg.InternalCompatibilityFlags.Overwrite)
			log.Printf("")
		}

	default:
		if cfg.InternalCompatibilityFlags.OverwriteBoot != "" {
			mbrfn := cfg.InternalCompatibilityFlags.OverwriteMBR
			if cfg.InternalCompatibilityFlags.OverwriteMBR == "" {
				var err error
				tmpMBR, err = os.CreateTemp("", "gokrazy")
				if err != nil {
					return err
				}
				defer os.Remove(tmpMBR.Name())
				mbrfn = tmpMBR.Name()
			}
			if err := pack.writeBootFile(cfg.InternalCompatibilityFlags.OverwriteBoot, mbrfn); err != nil {
				return err
			}
		}

		if cfg.InternalCompatibilityFlags.OverwriteRoot != "" {
			var rootErr error
			trace.WithRegion(context.Background(), "writeroot", func() {
				rootErr = pack.writeRootFile(cfg.InternalCompatibilityFlags.OverwriteRoot, root)
			})
			if rootErr != nil {
				return rootErr
			}
		}

		if cfg.InternalCompatibilityFlags.OverwriteBoot == "" && cfg.InternalCompatibilityFlags.OverwriteRoot == "" {
			var err error
			tmpMBR, err = os.CreateTemp("", "gokrazy")
			if err != nil {
				return err
			}
			defer os.Remove(tmpMBR.Name())

			tmpBoot, err = os.CreateTemp("", "gokrazy")
			if err != nil {
				return err
			}
			defer os.Remove(tmpBoot.Name())

			if err := pack.writeBoot(tmpBoot, tmpMBR.Name()); err != nil {
				return err
			}

			tmpRoot, err = os.CreateTemp("", "gokrazy")
			if err != nil {
				return err
			}
			defer os.Remove(tmpRoot.Name())

			if err := pack.writeRoot(tmpRoot, root); err != nil {
				return err
			}
		}
	}

	log.Printf("")
	log.Printf("Build complete!")

	if err := pack.printHowToInteract(cfg); err != nil {
		return err
	}

	if err := <-dnsCheck; err != nil {
		log.Printf("WARNING: if the above URL does not work, perhaps name resolution (DNS) is broken")
		log.Printf("in your local network? Resolving your hostname failed: %v", err)
		log.Printf("")
	}

	if newInstallation {
		return nil
	}

	// For update mode, we would need the updater package which depends on
	// gokrazy/internal. For now, return an error for updates that need HTTP.
	_ = isDev
	_ = tmpBoot
	_ = tmpRoot
	_ = tmpMBR
	return fmt.Errorf("HTTP-based update not yet implemented in monogok standalone; use overwrite mode")
}

func (p *Pack) overwriteDevice(dev string, root *FileInfo, rootDeviceFiles []deviceconfig.RootFile) error {
	log := p.Env.Logger()

	if err := verifyNotMounted(dev); err != nil {
		return err
	}
	parttable := "GPT + Hybrid MBR"
	if !p.Pack.UseGPT {
		parttable = "no GPT, only MBR"
	}
	log.Printf("partitioning %s (%s)", dev, parttable)

	f, err := p.partition(dev)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(p.Pack.FirstPartitionOffsetSectors*512, io.SeekStart); err != nil {
		return err
	}

	if err := p.writeBoot(f, ""); err != nil {
		return err
	}

	if err := p.writeMBR(p.Pack.FirstPartitionOffsetSectors, &offsetReadSeeker{f, p.Pack.FirstPartitionOffsetSectors * 512}, f, p.Pack.Partuuid); err != nil {
		return err
	}

	if _, err := f.Seek((p.Pack.FirstPartitionOffsetSectors+(100*MB/512))*512, io.SeekStart); err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "gokr-packer")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if err := p.writeRoot(tmp, root); err != nil {
		return err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := io.Copy(f, tmp); err != nil {
		return err
	}

	if err := p.writeRootDeviceFiles(f, rootDeviceFiles); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	fmt.Println("If your applications need to store persistent data, unplug and re-plug the SD card, then create a file system using e.g.:")
	fmt.Println()
	partition := partitionPath(dev, "4")
	if p.Pack.ModifyCmdlineRoot() {
		partition = fmt.Sprintf("/dev/disk/by-partuuid/%s", p.Pack.PermUUID())
	} else {
		if target, err := filepath.EvalSymlinks(dev); err == nil {
			partition = partitionPath(target, "4")
		}
	}
	fmt.Printf("\tmkfs.ext4 %s\n", partition)
	fmt.Println()

	return nil
}

func partitionPath(base, num string) string {
	if strings.HasPrefix(base, "/dev/mmcblk") ||
		strings.HasPrefix(base, "/dev/loop") {
		return base + "p" + num
	} else if strings.HasPrefix(base, "/dev/disk") ||
		strings.HasPrefix(base, "/dev/rdisk") {
		return base + "s" + num
	}
	return base + num
}

type offsetReadSeeker struct {
	io.ReadSeeker
	offset int64
}

func (ors *offsetReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		return ors.ReadSeeker.Seek(offset+ors.offset, io.SeekStart)
	}
	return ors.ReadSeeker.Seek(offset, whence)
}

type countingWriter int64

func (cw *countingWriter) Write(p []byte) (n int, err error) {
	*cw += countingWriter(len(p))
	return len(p), nil
}

func (p *Pack) overwriteFile(root *FileInfo, rootDeviceFiles []deviceconfig.RootFile, firstPartitionOffsetSectors int64) (bootSize int64, rootSize int64, err error) {
	f, err := os.Create(p.Cfg.InternalCompatibilityFlags.Overwrite)
	if err != nil {
		return 0, 0, err
	}

	if err := f.Truncate(int64(p.Cfg.InternalCompatibilityFlags.TargetStorageBytes)); err != nil {
		return 0, 0, err
	}

	if err := p.Pack.Partition(f, uint64(p.Cfg.InternalCompatibilityFlags.TargetStorageBytes)); err != nil {
		return 0, 0, err
	}

	if _, err := f.Seek(p.Pack.FirstPartitionOffsetSectors*512, io.SeekStart); err != nil {
		return 0, 0, err
	}
	var bs countingWriter
	if err := p.writeBoot(io.MultiWriter(f, &bs), ""); err != nil {
		return 0, 0, err
	}

	if err := p.writeMBR(p.Pack.FirstPartitionOffsetSectors, &offsetReadSeeker{f, p.Pack.FirstPartitionOffsetSectors * 512}, f, p.Pack.Partuuid); err != nil {
		return 0, 0, err
	}

	if _, err := f.Seek(p.Pack.FirstPartitionOffsetSectors*512+100*MB, io.SeekStart); err != nil {
		return 0, 0, err
	}

	tmp, err := os.CreateTemp("", "gokr-packer")
	if err != nil {
		return 0, 0, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if err := p.writeRoot(tmp, root); err != nil {
		return 0, 0, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}

	var rs countingWriter
	if _, err := io.Copy(io.MultiWriter(f, &rs), tmp); err != nil {
		return 0, 0, err
	}

	if err := p.writeRootDeviceFiles(f, rootDeviceFiles); err != nil {
		return 0, 0, err
	}

	fmt.Println("If your applications need to store persistent data, create a file system using e.g.:")
	fmt.Printf("\t/sbin/mkfs.ext4 -F -E offset=%v %s %v\n", p.Pack.FirstPartitionOffsetSectors*512+1100*MB, p.Cfg.InternalCompatibilityFlags.Overwrite, PermSizeInKB(firstPartitionOffsetSectors, uint64(p.Cfg.InternalCompatibilityFlags.TargetStorageBytes)))
	fmt.Println()

	return int64(bs), int64(rs), f.Close()
}

func (pack *Pack) printHowToInteract(cfg *config.Struct) error {
	log := pack.Env.Logger()
	update := pack.update

	updateFlag := pack.Cfg.InternalCompatibilityFlags.Update
	if updateFlag == "" {
		updateFlag = "yes"
	}
	updateBaseUrl, err := updateflag.Value{
		Update: updateFlag,
	}.BaseURL(update.HTTPPort, update.HTTPSPort, pack.schema, update.Hostname, update.HTTPPassword)
	if err != nil {
		return err
	}

	log.Printf("")
	log.Printf("To interact with the device, gokrazy provides a web interface reachable at:")
	log.Printf("")
	log.Printf("\t%s", updateBaseUrl.String())
	log.Printf("")
	log.Printf("In addition, the following Linux consoles are set up:")
	log.Printf("")
	if cfg.SerialConsoleOrDefault() != "disabled" {
		log.Printf("\t1. foreground Linux console on the serial port (115200n8)")
		log.Printf("\t2. secondary Linux framebuffer console on HDMI")
	} else {
		log.Printf("\t1. foreground Linux framebuffer console on HDMI")
	}
	log.Printf("")
	if pack.schema == "https" {
		certObj, err := getCertificateFromString(update.CertPEM)
		if err != nil {
			return fmt.Errorf("error loading certificate: %v", err)
		}
		log.Printf("TLS Certificate fingerprint: %x", getCertificateFingerprintSHA1(certObj))
		log.Printf("Certificate valid until: %s", certObj.NotAfter.String())
	}
	return nil
}
