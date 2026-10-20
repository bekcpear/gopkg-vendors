package packer

import (
	"debug/elf"
	"log"
)

func fileIsELFOrFatal(filePath string) {
	f, err := elf.Open(filePath)
	if err != nil {
		log.Fatalf("%s is not an ELF binary! %v", filePath, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("%s is not an ELF binary! Close: %v", filePath, err)
	}
}
