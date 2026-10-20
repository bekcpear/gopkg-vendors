package gokrazy

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tailscale/ts-gokrazy/internal/rootdev"
	"golang.org/x/sys/unix"
)

type rebootOpts struct {
	tryKexec                 bool
	kexecMergeCurrentCmdline bool
}

func kexecReboot(opts rebootOpts) error {
	tmpdir, err := ioutil.TempDir("", "kexec")
	if err != nil {
		return err
	}

	if err := syscall.Mount(rootdev.Partition(rootdev.Boot), tmpdir, "vfat", 0, ""); err != nil {
		return err
	}

	kernel, err := os.Open(filepath.Join(tmpdir, "vmlinuz"))
	if err != nil {
		return err
	}
	defer kernel.Close()
	cmdline, err := ioutil.ReadFile(filepath.Join(tmpdir, "cmdline.txt"))
	if err != nil {
		return err
	}
	if opts.kexecMergeCurrentCmdline {
		currentCmdline, err := ioutil.ReadFile("/proc/cmdline")
		if err != nil {
			return err
		}
		cmdline = []byte(mergeKexecCmdline(string(cmdline), string(currentCmdline)))
	}

	flags := unix.KEXEC_ARCH_DEFAULT | unix.KEXEC_FILE_NO_INITRAMFS
	if err := unix.KexecFileLoad(int(kernel.Fd()), 0, string(cmdline), flags); err != nil {
		// err is syscall.ENOSYS on kernels without CONFIG_KEXEC_FILE_LOAD=y
		return err
	}

	return unix.Reboot(unix.LINUX_REBOOT_CMD_KEXEC)
}

func reboot(opts rebootOpts) error {
	if opts.tryKexec {
		if err := kexecReboot(opts); err != nil {
			log.Printf("kexec reboot failed: %v", err)
		}
	}
	return unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART)
}

// mergeKexecCmdline returns a kexec cmdline that keeps target-owned boot
// selection parameters from targetCmdline while carrying runtime parameters
// from currentCmdline through to the kexec'd kernel.
//
// This is useful for test and VM environments that boot gokrazy with
// platform-specific kernel arguments supplied by the VM runner. After an
// update, kexec normally uses the new boot partition's cmdline.txt, which does
// not necessarily contain those live runtime arguments.
func mergeKexecCmdline(targetCmdline, currentCmdline string) string {
	var merged []string
	have := map[string]bool{}
	for f := range strings.FieldsSeq(currentCmdline) {
		if isTargetOwnedCmdlineArg(f) {
			continue
		}
		merged = append(merged, f)
		have[cmdlineArgKey(f)] = true
	}
	for f := range strings.FieldsSeq(targetCmdline) {
		if isTargetOwnedCmdlineArg(f) {
			merged = append(merged, f)
			have[cmdlineArgKey(f)] = true
			continue
		}
		if !have[cmdlineArgKey(f)] {
			merged = append(merged, f)
			have[cmdlineArgKey(f)] = true
		}
	}
	return strings.Join(merged, " ")
}

func isTargetOwnedCmdlineArg(arg string) bool {
	return strings.HasPrefix(arg, "root=") ||
		strings.HasPrefix(arg, "gokrazy.try_boot_inactive=") ||
		strings.HasPrefix(arg, "gokrazy.switch_on_boot=")
}

func cmdlineArgKey(arg string) string {
	k, _, _ := strings.Cut(arg, "=")
	return k
}
