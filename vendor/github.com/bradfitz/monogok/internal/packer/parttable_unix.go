//go:build unix

package packer

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func (p *Pack) SudoPartition(path string) (*os.File, error) {
	if fd, err := strconv.Atoi(os.Getenv("GOKR_PACKER_FD")); err == nil {
		// child process
		conn := mustUnixConn(uintptr(fd))
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		if err := p.partitionDevice(f, path); err != nil {
			return nil, err
		}
		_, _, err = conn.WriteMsgUnix(nil, syscall.UnixRights(int(f.Fd())), nil)
		return nil, err
	}

	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	syscall.CloseOnExec(pair[0])

	if exe, err := os.Executable(); err == nil {
		os.Args[0] = exe
	}

	cmd := exec.Command("sudo", append([]string{"--preserve-env"}, os.Args...)...)
	cmd.Env = []string{
		"GOKR_PACKER_FD=1",
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("GOKRAZY_PARENT_DIR=%s", os.Getenv("GOKRAZY_PARENT_DIR")),
	}
	cmd.Stdout = os.NewFile(uintptr(pair[1]), "")
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}()

	conn := mustUnixConn(uintptr(pair[0]))
	oob := make([]byte, 32)
	_, oobn, _, _, err := conn.ReadMsgUnix(nil, oob)
	if err != nil {
		return nil, err
	}
	if oobn <= 0 {
		return nil, fmt.Errorf("ReadMsgUnix: oobn <= 0")
	}

	scm, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, err
	}
	if got, want := len(scm), 1; got != want {
		return nil, fmt.Errorf("SCM message: got %d, want %d", got, want)
	}

	fds, err := syscall.ParseUnixRights(&scm[0])
	if err != nil {
		return nil, err
	}
	if got, want := len(fds), 1; got != want {
		return nil, fmt.Errorf("ParseUnixRights: got %d fds, want %d fds", got, want)
	}

	return os.NewFile(uintptr(fds[0]), ""), nil
}
