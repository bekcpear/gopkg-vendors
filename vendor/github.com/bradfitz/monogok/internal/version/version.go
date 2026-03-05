package version

import (
	"runtime/debug"
	"strings"
)

func readParts() (revision string, modified, ok bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false, false
	}
	settings := make(map[string]string)
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if rev, ok := settings["vcs.revision"]; ok {
		return rev, settings["vcs.modified"] == "true", true
	}
	v := info.Main.Version
	if idx := strings.LastIndexByte(v, '-'); idx > -1 {
		return v[idx+1:], false, true
	}
	return "<BUG>", false, false
}

func Read() string {
	revision, modified, ok := readParts()
	if !ok {
		return "<not okay>"
	}
	modifiedSuffix := ""
	if modified {
		modifiedSuffix = " (modified)"
	}

	return "https://github.com/bradfitz/monogok/commit/" + revision + modifiedSuffix
}

func ReadBrief() string {
	revision, modified, ok := readParts()
	if !ok {
		return "<not okay>"
	}
	modifiedSuffix := ""
	if modified {
		modifiedSuffix = "+"
	}
	if len(revision) > 6 {
		revision = revision[:6]
	}
	return "g" + revision + modifiedSuffix
}
