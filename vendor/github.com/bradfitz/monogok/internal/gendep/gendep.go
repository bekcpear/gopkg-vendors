// Package gendep generates a gokrazydeps.go file listing all packages
// from config.json as blank imports, for the purpose of making `go mod tidy`
// aware of the gokrazy packages in a monorepo.
package gendep

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bradfitz/monogok/internal/config"
)

func Generate(configPath string) error {
	cfg, err := config.ReadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %v", err)
	}

	var pkgs []string
	pkgs = append(pkgs, cfg.GokrazyPackagesOrDefault()...)
	pkgs = append(pkgs, cfg.Packages...)
	if cfg.KernelPackage != nil && *cfg.KernelPackage != "" {
		pkgs = append(pkgs, *cfg.KernelPackage)
	} else {
		pkgs = append(pkgs, "github.com/gokrazy/kernel.rpi")
	}
	if cfg.FirmwarePackage != nil && *cfg.FirmwarePackage != "" {
		pkgs = append(pkgs, *cfg.FirmwarePackage)
	}
	if cfg.EEPROMPackage != nil && *cfg.EEPROMPackage != "" {
		pkgs = append(pkgs, *cfg.EEPROMPackage)
	}
	// Always include gokrazy itself (for the init template)
	pkgs = append(pkgs, "github.com/gokrazy/gokrazy")

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, p := range pkgs {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		unique = append(unique, p)
	}
	sort.Strings(unique)

	var sb strings.Builder
	sb.WriteString("//go:build ignore\n\n")
	sb.WriteString("package gokrazydeps\n\n")
	sb.WriteString("import (\n")
	for _, p := range unique {
		sb.WriteString(fmt.Sprintf("\t_ %q\n", p))
	}
	sb.WriteString(")\n")

	outPath := filepath.Join(filepath.Dir(configPath), "gokrazydeps.go")
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing %s: %v", outPath, err)
	}
	fmt.Printf("Wrote %s\n", outPath)
	return nil
}
