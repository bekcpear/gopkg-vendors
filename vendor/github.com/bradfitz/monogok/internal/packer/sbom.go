package packer

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/bradfitz/monogok/internal/build"
	"github.com/bradfitz/monogok/internal/buildid"
	"github.com/bradfitz/monogok/internal/config"
	"golang.org/x/sync/errgroup"
)

type FileHash struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

type GoPackage struct {
	Path      string `json:"path"`
	BuildID   string
	BuildInfo string
}

type SBOM struct {
	ConfigHash      FileHash   `json:"config_hash"`
	ExtraFileHashes []FileHash `json:"extra_file_hashes"`
	GoPackages      []GoPackage `json:"go_packages"`
}

type SBOMWithHash struct {
	SBOMHash string `json:"sbom_hash"`
	SBOM     SBOM   `json:"sbom"`
}

func readBuildID(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	const readSize = 32 * 1024
	data := make([]byte, readSize)
	_, err := io.ReadFull(f, data)
	if err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return "", err
	}
	return buildid.ReadELF(f.Name(), f, data)
}

func generateSBOM(cfg *config.Struct, foundBins []foundBin) ([]byte, SBOMWithHash, error) {
	formattedCfg, err := cfg.FormatForFile()
	if err != nil {
		return nil, SBOMWithHash{}, err
	}

	result := SBOM{
		ConfigHash: FileHash{
			Path: cfg.Meta.Path,
			Hash: fmt.Sprintf("%x", sha256.Sum256([]byte(string(formattedCfg)))),
		},
	}

	var (
		eg           errgroup.Group
		goPackagesMu sync.Mutex
	)
	for _, bin := range foundBins {
		eg.Go(func() error {
			f, err := os.Open(bin.hostPath)
			if err != nil {
				return err
			}
			info, err := buildinfo.Read(f)
			if err != nil {
				return err
			}
			id, err := readBuildID(f)
			if err != nil {
				return err
			}

			goPackagesMu.Lock()
			defer goPackagesMu.Unlock()
			result.GoPackages = append(result.GoPackages, GoPackage{
				Path:      bin.gokrazyPath,
				BuildID:   id,
				BuildInfo: info.String(),
			})
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, SBOMWithHash{}, err
	}

	extraFiles, err := FindExtraFiles(cfg)
	if err != nil {
		return nil, SBOMWithHash{}, err
	}

	packages := append(getGokrazySystemPackages(cfg), cfg.Packages...)

	for _, pkgAndVersion := range packages {
		pkg := pkgAndVersion
		if idx := strings.IndexByte(pkg, '@'); idx > -1 {
			pkg = pkg[:idx]
		}

		files := append([]*FileInfo{}, extraFiles[pkg]...)
		if len(files) == 0 {
			continue
		}

		for len(files) > 0 {
			fi := files[0]
			files = files[1:]
			files = append(files, fi.Dirents...)
			if fi.FromHost == "" {
				continue
			}

			b, err := os.ReadFile(fi.FromHost)
			if err != nil {
				return nil, SBOMWithHash{}, err
			}
			result.ExtraFileHashes = append(result.ExtraFileHashes, FileHash{
				Path: fi.FromHost,
				Hash: fmt.Sprintf("%x", sha256.Sum256(b)),
			})
		}
	}

	sort.Slice(result.GoPackages, func(i, j int) bool {
		return result.GoPackages[i].Path < result.GoPackages[j].Path
	})

	sort.Slice(result.ExtraFileHashes, func(i, j int) bool {
		return result.ExtraFileHashes[i].Path < result.ExtraFileHashes[j].Path
	})

	b, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return nil, SBOMWithHash{}, err
	}
	b = append(b, '\n')

	sH := SBOMWithHash{
		SBOMHash: fmt.Sprintf("%x", sha256.Sum256(b)),
		SBOM:     result,
	}

	sM, err := json.MarshalIndent(sH, "", "    ")
	if err != nil {
		return nil, SBOMWithHash{}, err
	}
	sM = append(sM, '\n')

	return sM, sH, nil
}

func getGokrazySystemPackages(cfg *config.Struct) []string {
	pkgs := append([]string{}, cfg.GokrazyPackagesOrDefault()...)
	pkgs = append(pkgs, build.InitDeps(cfg.InternalCompatibilityFlags.InitPkg)...)
	pkgs = append(pkgs, cfg.KernelPackageOrDefault())
	if fw := cfg.FirmwarePackageOrDefault(); fw != "" {
		pkgs = append(pkgs, fw)
	}
	if e := cfg.EEPROMPackageOrDefault(); e != "" {
		pkgs = append(pkgs, e)
	}
	return pkgs
}
