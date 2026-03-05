// Package build provides Go build invocation for monogok.
// Unlike gokrazy's per-package builddir approach, monogok builds all packages
// from the enclosing Go module root.
package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bradfitz/monogok/internal/measure"
	"golang.org/x/sync/errgroup"
)

func DefaultTags() []string {
	return []string{
		"gokrazy",
		"netgo",
		"osusergo",
	}
}

func TargetArch() string {
	if arch := os.Getenv("GOARCH"); arch != "" {
		return arch
	}
	return "arm64" // Raspberry Pi 3, 4, Zero 2 W
}

var (
	envOnce sync.Once
	env     []string
)

func goEnv() []string {
	goarch := TargetArch()

	goos := "linux"
	if e := os.Getenv("GOOS"); e != "" {
		goos = e
	}

	cgoEnabledFound := false
	env := os.Environ()
	for idx, e := range env {
		if strings.HasPrefix(e, "CGO_ENABLED=") {
			cgoEnabledFound = true
		}
		if strings.HasPrefix(e, "GOBIN=") {
			env[idx] = "GOBIN="
		}
	}
	if !cgoEnabledFound {
		env = append(env, "CGO_ENABLED=0")
	}
	return append(env,
		fmt.Sprintf("GOARCH=%s", goarch),
		fmt.Sprintf("GOOS=%s", goos),
		"GOBIN=")
}

func Env() []string {
	envOnce.Do(func() {
		env = goEnv()
	})
	return env
}

func InitDeps(initPkg string) []string {
	if initPkg != "" {
		return []string{initPkg}
	}
	return []string{"github.com/gokrazy/gokrazy"}
}

// BuildEnv holds the build directory (module root) for all build operations.
type BuildEnv struct {
	// BuildDir returns the module root for a given package.
	// In monogok, this always returns the same directory (the module root).
	BuildDir func(string) (string, error)
}

func (be *BuildEnv) Build(bindir string, packages []string, packageBuildFlags, packageBuildTags, packageBuildEnv map[string][]string, noBuildPackages []string, basenames map[string]string) error {
	done := measure.Interactively("building (go compiler)")
	defer done("")

	var eg errgroup.Group
	for _, incompletePkg := range packages {
		mainPkgs, err := be.MainPackages([]string{incompletePkg})
		if err != nil {
			return err
		}
		for _, pkg := range mainPkgs {
			pkg := pkg // copy
			buildDir, err := be.BuildDir(pkg.ImportPath)
			if err != nil {
				return fmt.Errorf("buildDir(%s): %v", pkg.ImportPath, err)
			}
			eg.Go(func() error {
				basename := pkg.Basename()
				if basenameOverride, ok := basenames[pkg.ImportPath]; ok {
					basename = basenameOverride
				}
				args := []string{
					"build",
					"-o", filepath.Join(bindir, basename),
				}
				tags := append(DefaultTags(), packageBuildTags[pkg.ImportPath]...)
				args = append(args, "-tags="+strings.Join(tags, ","))
				if buildFlags := packageBuildFlags[pkg.ImportPath]; len(buildFlags) > 0 {
					args = append(args, buildFlags...)
				}
				args = append(args, pkg.ImportPath)
				cmd := exec.Command("go", args...)
				cmd.Env = append(Env(), packageBuildEnv[pkg.ImportPath]...)
				cmd.Dir = buildDir
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("%v: %v", cmd.Args, err)
				}
				return nil
			})
		}
	}
	return eg.Wait()
}

type Pkg struct {
	Name       string `json:"Name"`
	ImportPath string `json:"ImportPath"`
	Target     string `json:"Target"`
}

func (p *Pkg) Basename() string {
	if p.Target != "" {
		return filepath.Base(p.Target)
	}
	base := path.Base(p.ImportPath)
	if isVersionElement(base) {
		return path.Base(path.Dir(p.ImportPath))
	}
	return base
}

func isVersionElement(s string) bool {
	if len(s) < 2 || s[0] != 'v' || s[1] == '0' || s[1] == '1' && len(s) == 2 {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return false
		}
	}
	return true
}

func (be *BuildEnv) mainPackage(pkg string) ([]Pkg, error) {
	buildDir, err := be.BuildDir(pkg)
	if err != nil {
		return nil, fmt.Errorf("BuildDir(%s): %v", pkg, err)
	}

	var buf bytes.Buffer
	cmd := exec.Command("go", append([]string{"list", "-tags", "gokrazy", "-json"}, pkg)...)
	cmd.Dir = buildDir
	cmd.Env = Env()
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %v", cmd.Args, err)
	}
	var result []Pkg
	dec := json.NewDecoder(&buf)
	for {
		var p Pkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if p.Name != "main" {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

func (be *BuildEnv) MainPackages(pkgs []string) ([]Pkg, error) {
	var (
		eg       errgroup.Group
		resultMu sync.Mutex
		result   []Pkg
	)
	for _, pkg := range pkgs {
		pkg := pkg // copy
		eg.Go(func() error {
			p, err := be.mainPackage(pkg)
			if err != nil {
				return err
			}
			resultMu.Lock()
			defer resultMu.Unlock()
			result = append(result, p...)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Basename() < result[j].Basename()
	})
	return result, nil
}

func PackageDir(buildDir, pkg string) (string, error) {
	cmd := exec.Command("go", "list", "-tags", "gokrazy", "-f", "{{ .Dir }}", pkg)
	cmd.Env = Env()
	cmd.Dir = buildDir
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%v: %v", cmd.Args, err)
	}
	return strings.TrimSpace(string(b)), nil
}

func PackageDirs(buildDir string, pkgs []string) ([]string, error) {
	var eg errgroup.Group
	dirs := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		i := i
		pkg := pkg
		eg.Go(func() error {
			dir, err := PackageDir(buildDir, pkg)
			if err != nil {
				return err
			}
			dirs[i] = dir
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return dirs, nil
}

func PermSizeInKB(firstPartitionOffsetSectors int64, devsize uint64) uint32 {
	permStart := uint32(firstPartitionOffsetSectors + (1100 * 1024 * 1024 / 512))
	permSizeLBA := uint32((devsize / 512) - uint64(firstPartitionOffsetSectors) - (1100 * 1024 * 1024 / 512))
	lastAddressable := uint32((devsize / 512) - 1)
	if lastLBA := uint32(lastAddressable - 33); permStart+permSizeLBA >= lastLBA {
		permSizeLBA -= (permStart + permSizeLBA) - lastLBA
	}
	_ = log.Printf // import used
	return (permSizeLBA * 512) / 1024
}
