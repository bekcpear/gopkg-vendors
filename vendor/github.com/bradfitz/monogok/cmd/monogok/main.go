// monogok builds gokrazy images from a monorepo using a single go.mod.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradfitz/monogok/internal/config"
	"github.com/bradfitz/monogok/internal/gendep"
	"github.com/bradfitz/monogok/internal/packer"
	"github.com/spf13/cobra"
)

var configFlag string

func main() {
	rootCmd := &cobra.Command{
		Use:   "monogok",
		Short: "Build gokrazy images using a monorepo's single go.mod",
	}
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "path to config.json (default: walk up from cwd)")

	rootCmd.AddCommand(overwriteCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(gendepCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*config.Struct, string, error) {
	cfgPath, err := packer.FindConfig(configFlag)
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.ReadFromFile(cfgPath)
	if err != nil {
		return nil, "", err
	}
	moduleRoot, err := packer.FindModuleRoot(filepath.Dir(cfgPath))
	if err != nil {
		return nil, "", err
	}
	cfg.ApplyEnvironment()
	return cfg, moduleRoot, nil
}

func overwriteCmd() *cobra.Command {
	var (
		overwritePath    string
		overwriteBoot    string
		overwriteRoot    string
		overwriteMBR     string
		targetBytes      int
		sudo             string
	)

	cmd := &cobra.Command{
		Use:   "overwrite",
		Short: "Build all packages and write a full disk image",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, moduleRoot, err := loadConfig()
			if err != nil {
				return err
			}

			if overwritePath == "" {
				return fmt.Errorf("--full flag (path to device or file) is required")
			}

			cfg.InternalCompatibilityFlags.Overwrite = overwritePath
			cfg.InternalCompatibilityFlags.OverwriteBoot = overwriteBoot
			cfg.InternalCompatibilityFlags.OverwriteRoot = overwriteRoot
			cfg.InternalCompatibilityFlags.OverwriteMBR = overwriteMBR
			cfg.InternalCompatibilityFlags.TargetStorageBytes = targetBytes
			if sudo != "" {
				cfg.InternalCompatibilityFlags.Sudo = sudo
			}

			pack := packer.NewPack(cfg, moduleRoot)
			return pack.RunOverwrite(context.Background())
		},
	}

	cmd.Flags().StringVar(&overwritePath, "full", "", "path to SD card device or image file to overwrite")
	cmd.Flags().StringVar(&overwriteBoot, "boot", "", "path to overwrite just the boot partition")
	cmd.Flags().StringVar(&overwriteRoot, "root", "", "path to overwrite just the root partition")
	cmd.Flags().StringVar(&overwriteMBR, "mbr", "", "path to overwrite just the MBR")
	cmd.Flags().IntVar(&targetBytes, "target_storage_bytes", 0, "target storage size in bytes (for file images)")
	cmd.Flags().StringVar(&sudo, "sudo", "", "sudo mode: auto, always, or never")

	return cmd
}

func updateCmd() *cobra.Command {
	var (
		insecure bool
		testboot bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Build and deploy over the network",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, moduleRoot, err := loadConfig()
			if err != nil {
				return err
			}

			cfg.InternalCompatibilityFlags.Update = "yes"
			cfg.InternalCompatibilityFlags.Insecure = insecure
			cfg.InternalCompatibilityFlags.Testboot = testboot

			pack := packer.NewPack(cfg, moduleRoot)
			return pack.RunUpdate(context.Background())
		},
	}

	cmd.Flags().BoolVar(&insecure, "insecure", false, "use HTTP instead of HTTPS for updates")
	cmd.Flags().BoolVar(&testboot, "testboot", false, "test-boot instead of permanently switching")

	return cmd
}

func gendepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gendep",
		Short: "Generate gokrazydeps.go next to config.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := packer.FindConfig(configFlag)
			if err != nil {
				return err
			}
			return gendep.Generate(cfgPath)
		},
	}

	return cmd
}
