package packer

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradfitz/monogok/internal/build"
	"github.com/bradfitz/monogok/internal/config"
	"github.com/bradfitz/monogok/internal/deviceconfig"
)

const programName = "monogok"

// OutputType distinguishes between full disk image and gaf.
type OutputType string

const (
	OutputTypeFull OutputType = "full"
	OutputTypeGaf  OutputType = "gaf"
)

// Output holds overwrite output config.
type Output struct {
	Type OutputType
	Path string
}

// Env holds the environment for a Pack operation.
type Env struct {
	logger Logger
}

// Logger is the logger interface used by the packer.
type Logger interface {
	Printf(msg string, a ...any)
	Output(calldepth int, s string) error
}

func (e *Env) Logger() Logger {
	if e.logger == nil {
		return log.Default()
	}
	return e.logger
}

func NewEnv(logger Logger) Env {
	return Env{logger: logger}
}

type BuildTimestampOverrideKey struct{}

var BuildTimestampOverride = BuildTimestampOverrideKey{}

// Pack is the core packer state.
type Pack struct {
	Env     Env
	Cfg     *config.Struct
	FileCfg *config.Struct // cfg before internal modifications
	Pack    PackerPack
	Output  *Output

	// moduleRoot is the Go module root directory.
	moduleRoot string

	root             *FileInfo
	update           *config.UpdateStruct
	rootDeviceFiles  []deviceconfig.RootFile
	kernelDir        string
	initTmp          string
	schema           string
	buildTimestamp   string
	systemCertsPEM   string

	packageBuildFlags map[string][]string
	packageBuildTags  map[string][]string
	packageBuildEnv   map[string][]string
	flagFileContents  map[string][]string
	envFileContents   map[string][]string
	dontStart         map[string]bool
	waitForClock      map[string]bool
	basenames         map[string]string

	firstPartitionOffsetSectors int64

	sbom         []byte
	sbomWithHash SBOMWithHash
}

// NewPack creates a new Pack with the given config and module root.
func NewPack(cfg *config.Struct, moduleRoot string) *Pack {
	fileCfg := *cfg // shallow copy for SBOM purposes
	return &Pack{
		Cfg:        cfg,
		FileCfg:    &fileCfg,
		moduleRoot: moduleRoot,
		Env:        NewEnv(nil),
	}
}

// BuildDir returns the module root, regardless of the package.
func (pack *Pack) BuildDir(pkg string) (string, error) {
	return pack.moduleRoot, nil
}

func (pack *Pack) PackageDir(pkg string) (string, error) {
	return build.PackageDir(pack.moduleRoot, pkg)
}

// FindConfig discovers config.json by walking up from cwd (or using the explicit path).
func FindConfig(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file %s: %v", explicit, err)
		}
		return explicit, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, "config.json")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no config.json found (walked up to /)")
		}
		dir = parent
	}
}

// FindModuleRoot walks up from dir to find go.mod.
func FindModuleRoot(dir string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found (walked up to /)")
		}
		dir = parent
	}
}

func (pack *Pack) RunOverwrite(ctx context.Context) error {
	if err := pack.logicPrepare(ctx); err != nil {
		return err
	}

	bindir, err := os.MkdirTemp("", "monogok-bins")
	if err != nil {
		return err
	}
	defer os.RemoveAll(bindir)

	if err := pack.logicBuild(bindir); err != nil {
		return err
	}

	dnsCheck := make(chan error, 1)
	dnsCheck <- nil

	return pack.logicWrite(dnsCheck)
}

func (pack *Pack) RunUpdate(ctx context.Context) error {
	if err := pack.logicPrepare(ctx); err != nil {
		return err
	}

	bindir, err := os.MkdirTemp("", "monogok-bins")
	if err != nil {
		return err
	}
	defer os.RemoveAll(bindir)

	if err := pack.logicBuild(bindir); err != nil {
		return err
	}

	dnsCheck := make(chan error, 1)
	dnsCheck <- nil

	return pack.logicWrite(dnsCheck)
}

func filterGoEnv(env []string) []string {
	var result []string
	for _, e := range env {
		if strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "GOOS=") {
			result = append(result, e)
		}
	}
	return result
}

func verifyNotMounted(dev string) error {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil // best-effort
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.HasPrefix(fields[0], dev) {
			return fmt.Errorf("%s is mounted as %s", fields[0], fields[1])
		}
	}
	return nil
}
