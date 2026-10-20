// Package deviceconfig contains any device-specific configuration.
package deviceconfig

// RootFile represents a file that is stored on a raw disk device.
type RootFile struct {
	Name      string
	Offset    int64
	MaxLength int64
}

type DeviceConfig struct {
	MBROnlyWithoutGPT    bool
	RootDeviceFiles      []RootFile
	Slug                 string
	BootPartitionStartLBA int64
}

const (
	sectorSize                         = 512
	DefaultBootPartitionStartLBA int64 = 8192
)

var (
	DeviceConfigs = map[string]DeviceConfig{
		"Hardkernel Odroid HC1": {
			MBROnlyWithoutGPT: true,
			RootDeviceFiles: []RootFile{
				{"bl1.bin", 1 * sectorSize, 30 * sectorSize},
				{"bl2.bin", 31 * sectorSize, 32 * sectorSize},
				{"u-boot.bin", 63 * sectorSize, 1440 * sectorSize},
				{"tzsw.bin", 1503 * sectorSize, 512 * sectorSize},
			},
			Slug: "odroidhc1",
		},
		"QEMU testing MBR": {
			MBROnlyWithoutGPT: true,
			Slug:              "qemumbrtesting",
		},
		"Pine64 Rock64": {
			RootDeviceFiles: []RootFile{
				{"u-boot-rockchip.bin", 64 * sectorSize, 32704 * sectorSize},
			},
			BootPartitionStartLBA: 32768,
			Slug:                  "rock64",
		},
		"FriendlyElec NanoPi Neo": {
			MBROnlyWithoutGPT: true,
			RootDeviceFiles: []RootFile{
				{"u-boot-sunxi-with-spl.bin", 16 * sectorSize, 2032 * sectorSize},
			},
			BootPartitionStartLBA: 2048,
			Slug:                  "nanopi_neo",
		},
	}
)

func GetDeviceConfigBySlug(slug string) (DeviceConfig, bool) {
	for _, cfg := range DeviceConfigs {
		if cfg.Slug == slug {
			return cfg, true
		}
	}
	return DeviceConfig{}, false
}
