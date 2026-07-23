package packer

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// overwriteGaf writes a gaf (gokrazy archive format) file by packing build
// artifacts into an uncompressed zip archive.
func (p *Pack) overwriteGaf(root *FileInfo, sbomMarshaled []byte) error {
	dir, err := os.MkdirTemp("", "gokrazy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	tmpMBR, err := os.Create(filepath.Join(dir, "mbr.img"))
	if err != nil {
		return err
	}
	defer os.Remove(tmpMBR.Name())

	tmpBoot, err := os.Create(filepath.Join(dir, "boot.img"))
	if err != nil {
		return err
	}
	defer os.Remove(tmpBoot.Name())

	tmpRoot, err := os.Create(filepath.Join(dir, "root.img"))
	if err != nil {
		return err
	}
	defer os.Remove(tmpRoot.Name())

	tmpSBOM, err := os.Create(filepath.Join(dir, "sbom.json"))
	if err != nil {
		return err
	}
	defer os.Remove(tmpSBOM.Name())

	if err := p.writeBoot(tmpBoot, tmpMBR.Name()); err != nil {
		return err
	}
	if err := p.writeRoot(tmpRoot, root); err != nil {
		return err
	}
	if _, err := tmpSBOM.Write(sbomMarshaled); err != nil {
		return err
	}

	for _, f := range []*os.File{tmpMBR, tmpBoot, tmpRoot, tmpSBOM} {
		if err := f.Close(); err != nil {
			return err
		}
	}

	return writeGafArchive(dir, p.Output.Path)
}

func writeGafArchive(sourceDir, targetFile string) error {
	f, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Method = zip.Store
		header.Name, err = filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		sf, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer sf.Close()

		_, err = io.Copy(headerWriter, sf)
		return err
	})
}
