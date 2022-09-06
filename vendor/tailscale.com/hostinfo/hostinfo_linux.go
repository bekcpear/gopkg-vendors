// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android
// +build linux,!android

package hostinfo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/sys/unix"
	"tailscale.com/util/lineread"
	"tailscale.com/util/strs"
	"tailscale.com/version/distro"
)

func init() {
	osVersion = osVersionLinux
	packageType = packageTypeLinux

	if v := linuxDeviceModel(); v != "" {
		SetDeviceModel(v)
	}
}

func linuxDeviceModel() string {
	for _, path := range []string{
		// First try the Synology-specific location.
		// Example: "DS916+-j"
		"/proc/sys/kernel/syno_hw_version",

		// Otherwise, try the Devicetree model, usually set on
		// ARM SBCs, etc.
		// Example: "Raspberry Pi 4 Model B Rev 1.2"
		// Example: "WD My Cloud Gen2: Marvell Armada 375"
		"/sys/firmware/devicetree/base/model", // Raspberry Pi 4 Model B Rev 1.2"
	} {
		b, _ := os.ReadFile(path)
		if s := strings.Trim(string(b), "\x00\r\n\t "); s != "" {
			return s
		}
	}
	return ""
}

func getQnapQtsVersion(versionInfo string) string {
	for _, field := range strings.Fields(versionInfo) {
		if suffix, ok := strs.CutPrefix(field, "QTSFW_"); ok {
			return "QTS " + suffix
		}
	}
	return ""
}

func osVersionLinux() string {
	// TODO(bradfitz,dgentry): cache this, or make caller(s) cache it.
	dist := distro.Get()
	propFile := "/etc/os-release"
	switch dist {
	case distro.Synology:
		propFile = "/etc.defaults/VERSION"
	case distro.OpenWrt:
		propFile = "/etc/openwrt_release"
	case distro.WDMyCloud:
		slurp, _ := ioutil.ReadFile("/etc/version")
		return fmt.Sprintf("%s", string(bytes.TrimSpace(slurp)))
	case distro.QNAP:
		slurp, _ := ioutil.ReadFile("/etc/version_info")
		return getQnapQtsVersion(string(slurp))
	}

	m := map[string]string{}
	lineread.File(propFile, func(line []byte) error {
		eq := bytes.IndexByte(line, '=')
		if eq == -1 {
			return nil
		}
		k, v := string(line[:eq]), strings.Trim(string(line[eq+1:]), `"'`)
		m[k] = v
		return nil
	})

	var un unix.Utsname
	unix.Uname(&un)

	var attrBuf strings.Builder
	attrBuf.WriteString("; kernel=")
	attrBuf.WriteString(unix.ByteSliceToString(un.Release[:]))
	if inContainer() {
		attrBuf.WriteString("; container")
	}
	if env := GetEnvType(); env != "" {
		fmt.Fprintf(&attrBuf, "; env=%s", env)
	}
	attr := attrBuf.String()

	id := m["ID"]

	switch id {
	case "debian":
		slurp, _ := ioutil.ReadFile("/etc/debian_version")
		return fmt.Sprintf("Debian %s (%s)%s", bytes.TrimSpace(slurp), m["VERSION_CODENAME"], attr)
	case "ubuntu":
		return fmt.Sprintf("Ubuntu %s%s", m["VERSION"], attr)
	case "", "centos": // CentOS 6 has no /etc/os-release, so its id is ""
		if cr, _ := ioutil.ReadFile("/etc/centos-release"); len(cr) > 0 { // "CentOS release 6.10 (Final)
			return fmt.Sprintf("%s%s", bytes.TrimSpace(cr), attr)
		}
		fallthrough
	case "fedora", "rhel", "alpine", "nixos":
		// Their PRETTY_NAME is fine as-is for all versions I tested.
		fallthrough
	default:
		if v := m["PRETTY_NAME"]; v != "" {
			return fmt.Sprintf("%s%s", v, attr)
		}
	}
	switch dist {
	case distro.Synology:
		return fmt.Sprintf("Synology %s%s", m["productversion"], attr)
	case distro.OpenWrt:
		return fmt.Sprintf("OpenWrt %s%s", m["DISTRIB_RELEASE"], attr)
	case distro.Gokrazy:
		return fmt.Sprintf("Gokrazy%s", attr)
	}
	return fmt.Sprintf("Other%s", attr)
}

func packageTypeLinux() string {
	// Report whether this is in a snap.
	// See https://snapcraft.io/docs/environment-variables
	// We just look at two somewhat arbitrarily.
	if os.Getenv("SNAP_NAME") != "" && os.Getenv("SNAP") != "" {
		return "snap"
	}
	return ""
}
