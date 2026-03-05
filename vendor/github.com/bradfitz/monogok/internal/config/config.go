// Package config allows reading gokrazy instance configuration from config.json.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InternalCompatibilityFlags keep older gokr-packer behavior or user interface
// working, but should never be set by users in config.json manually.
type InternalCompatibilityFlags struct {
	// These have become 'gok overwrite' flags:
	Overwrite          string `json:",omitempty"` // -overwrite
	OverwriteBoot      string `json:",omitempty"` // -overwrite_boot
	OverwriteMBR       string `json:",omitempty"` // -overwrite_mbr
	OverwriteRoot      string `json:",omitempty"` // -overwrite_root
	Sudo               string `json:",omitempty"` // -sudo
	TargetStorageBytes int    `json:",omitempty"` // -target_storage_bytes

	// These have become 'gok update' flags:
	Update   string `json:",omitempty"` // -update
	Insecure bool   `json:",omitempty"` // -insecure
	Testboot bool   `json:",omitempty"` // -testboot

	// These will likely not be carried over from gokr-packer to gok because of
	// a lack of usage.
	InitPkg       string `json:",omitempty"` // -init_pkg
	OverwriteInit string `json:",omitempty"` // -overwrite_init
}

type UpdateStruct struct {
	// Hostname (in UpdateStruct) overrides Struct.Hostname, but only for
	// deploying the update via HTTP, not in the generated image.
	Hostname string `json:",omitempty"`

	UseTLS string `json:",omitempty"` // -tls

	NoPassword bool `json:",omitempty"`

	HTTPPort     string `json:",omitempty"` // -http_port
	HTTPSPort    string `json:",omitempty"` // -https_port
	HTTPPassword string `json:",omitempty"` // http-password.txt
	CertPEM      string `json:",omitempty"` // cert.pem
	KeyPEM       string `json:",omitempty"` // key.pem
}

func (u *UpdateStruct) WithFallbackToHostSpecific(host string) (*UpdateStruct, error) {
	if u == nil {
		u = &UpdateStruct{}
	}
	result := UpdateStruct{
		Hostname:   u.Hostname,
		NoPassword: u.NoPassword,
	}

	if u.HTTPPort != "" {
		result.HTTPPort = u.HTTPPort
	} else {
		port, err := HostnameSpecific(host).ReadFile("http-port.txt")
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		result.HTTPPort = port
	}

	if u.HTTPSPort != "" {
		result.HTTPSPort = u.HTTPSPort
	} else {
		result.HTTPSPort = result.HTTPPort
	}

	if !u.NoPassword {
		if u.HTTPPassword != "" {
			result.HTTPPassword = u.HTTPPassword
		} else {
			pw, err := HostnameSpecific(host).ReadFile("http-password.txt")
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			result.HTTPPassword = pw
		}
	}

	return &result, nil
}

type PackageConfig struct {
	GoBuildEnvironment []string            `json:",omitempty"`
	GoBuildFlags       []string            `json:",omitempty"`
	GoBuildTags        []string            `json:",omitempty"`
	ExtraFilePaths     map[string]string   `json:",omitempty"`
	ExtraFileContents  map[string]string   `json:",omitempty"`
	Basename           string              `json:",omitempty"`
	Environment        []string            `json:",omitempty"`
	CommandLineFlags   []string            `json:",omitempty"`
	DontStart          bool                `json:",omitempty"`
	WaitForClock       bool                `json:",omitempty"`
}

type MountDevice struct {
	Source  string
	Type    string
	Target  string
	Options string
}

type Struct struct {
	Hostname   string        // -hostname
	DeviceType string        `json:",omitempty"` // -device_type
	Update     *UpdateStruct `json:",omitempty"`

	Environment []string `json:",omitempty"`

	Packages []string // flag.Args()

	PackageConfig map[string]PackageConfig `json:",omitempty"`

	SerialConsole string `json:",omitempty"`

	GokrazyPackages *[]string `json:",omitempty"` // -gokrazy_pkgs
	KernelPackage   *string   `json:",omitempty"` // -kernel_package
	FirmwarePackage *string   `json:",omitempty"` // -firmware_package
	EEPROMPackage   *string   `json:",omitempty"` // -eeprom_package

	KernelExtraArgs       []string `json:",omitempty"`
	BootloaderExtraLines  []string `json:",omitempty"`
	BootloaderExtraEEPROM []string `json:",omitempty"`

	MountDevices []MountDevice `json:",omitempty"`

	InternalCompatibilityFlags *InternalCompatibilityFlags `json:",omitempty"`

	Meta struct {
		Instance     string
		Path         string
		LastModified time.Time
	} `json:"-"` // omit from JSON
}

func (s *Struct) SerialConsoleOrDefault() string {
	if s.SerialConsole == "" {
		return "serial0,115200"
	}
	return s.SerialConsole
}

func (s *Struct) GokrazyPackagesOrDefault() []string {
	if s.GokrazyPackages == nil {
		return []string{
			"github.com/gokrazy/gokrazy/cmd/dhcp",
			"github.com/gokrazy/gokrazy/cmd/ntp",
			"github.com/gokrazy/gokrazy/cmd/randomd",
			"github.com/gokrazy/gokrazy/cmd/heartbeat",
		}
	}
	return *s.GokrazyPackages
}

func (s *Struct) KernelPackageOrDefault() string {
	if s.KernelPackage == nil {
		return "github.com/gokrazy/kernel.rpi"
	}
	return *s.KernelPackage
}

func (s *Struct) FirmwarePackageOrDefault() string {
	if s.FirmwarePackage == nil {
		return "github.com/gokrazy/firmware"
	}
	return *s.FirmwarePackage
}

func (s *Struct) EEPROMPackageOrDefault() string {
	if s.EEPROMPackage == nil {
		return "github.com/gokrazy/rpi-eeprom"
	}
	return *s.EEPROMPackage
}

func (s *Struct) ApplyEnvironment() {
	for _, kv := range s.Environment {
		key, value, _ := strings.Cut(kv, "=")
		os.Setenv(key, value)
	}
}

func (s *Struct) FormatForFile() ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	return b, nil
}

func (i *InternalCompatibilityFlags) SudoOrDefault() string {
	if i.Sudo == "" {
		return "auto"
	}
	return i.Sudo
}

func ReadFromFile(fn string) (*Struct, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var cfg Struct
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("decoding %s: %v", fn, err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	if cfg.Update == nil {
		cfg.Update = &UpdateStruct{}
	}
	if cfg.InternalCompatibilityFlags == nil {
		cfg.InternalCompatibilityFlags = &InternalCompatibilityFlags{}
	}
	cfg.Meta.Path = fn
	cfg.Meta.LastModified = st.ModTime()
	return &cfg, nil
}

func validate(cfg *Struct) error {
	for _, kv := range cfg.Environment {
		if _, _, ok := strings.Cut(kv, "="); !ok {
			return fmt.Errorf("malformed Environment entry %q, expected key=value", kv)
		}
	}
	return nil
}

func userConfigDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return d
}

func gokrazyConfigDir() string {
	return filepath.Join(userConfigDir(), "gokrazy")
}

func Gokrazy() string { return gokrazyConfigDir() }

type HostnameDir string

func (h HostnameDir) ReadFile(configBaseName string) (string, error) {
	b, err := os.ReadFile(filepath.Join(string(h), configBaseName))
	if err != nil {
		b, err = os.ReadFile(filepath.Join(gokrazyConfigDir(), configBaseName))
		if err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(string(b)), nil
}

func HostnameSpecific(hostname string) HostnameDir {
	return HostnameDir(filepath.Join(gokrazyConfigDir(), "hosts", hostname))
}
