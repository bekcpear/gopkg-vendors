//go:build !linux && !darwin

package packer

import (
	"fmt"
	"os"
	"runtime"
)

func deviceSize(fd uintptr) (uint64, error) {
	return 0, fmt.Errorf("gokrazy is currently missing code for getting device sizes on your operating system")
}

func rereadPartitions(f *os.File) error {
	return fmt.Errorf("gokrazy is currently missing code for re-reading partition tables on your operating system")
}

func (p *Pack) SudoPartition(path string) (*os.File, error) {
	return nil, fmt.Errorf("gokrazy is currently missing code for elevating privileges on %s", runtime.GOOS)
}
