package packer

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"
	"time"

	"github.com/bradfitz/monogok/internal/build"
	"github.com/bradfitz/monogok/internal/config"
	"github.com/bradfitz/monogok/internal/updateflag"
)

func (pack *Pack) logicBuild(bindir string) error {
	log := pack.Env.Logger()

	cfg := pack.Cfg
	args := cfg.Packages
	log.Printf("Building %d Go packages:", len(args))
	log.Printf("")
	for _, pkg := range args {
		log.Printf("  %s", pkg)
		for _, configFile := range packageConfigFiles[pkg] {
			log.Printf("    will %s", configFile.kind)
			if configFile.path != "" {
				log.Printf("      from %s", configFile.path)
			}
			if !configFile.lastModified.IsZero() {
				log.Printf("      last modified: %s (%s ago)",
					configFile.lastModified.Format(time.RFC3339),
					time.Since(configFile.lastModified).Round(1*time.Second))
			}
		}
		log.Printf("")
	}

	pkgs := append([]string{}, cfg.GokrazyPackagesOrDefault()...)
	pkgs = append(pkgs, cfg.Packages...)
	pkgs = append(pkgs, build.InitDeps(cfg.InternalCompatibilityFlags.InitPkg)...)
	noBuildPkgs := []string{
		cfg.KernelPackageOrDefault(),
	}
	if fw := cfg.FirmwarePackageOrDefault(); fw != "" {
		noBuildPkgs = append(noBuildPkgs, fw)
	}
	if e := cfg.EEPROMPackageOrDefault(); e != "" {
		noBuildPkgs = append(noBuildPkgs, e)
	}
	setUmask()

	buildEnv := &build.BuildEnv{
		BuildDir: pack.BuildDir,
	}
	var buildErr error
	trace.WithRegion(context.Background(), "build", func() {
		buildErr = buildEnv.Build(bindir, pkgs, pack.packageBuildFlags, pack.packageBuildTags, pack.packageBuildEnv, noBuildPkgs, pack.basenames)
	})
	if buildErr != nil {
		return buildErr
	}

	log.Printf("")

	var err error
	trace.WithRegion(context.Background(), "validate", func() {
		err = pack.validateTargetArchMatchesKernel()
	})
	if err != nil {
		return err
	}

	var (
		root      *FileInfo
		foundBins []foundBin
	)
	trace.WithRegion(context.Background(), "findbins", func() {
		root, foundBins, err = findBins(cfg, buildEnv, bindir, pack.basenames, pack.moduleRoot)
	})
	if err != nil {
		return err
	}
	pack.root = root

	packageConfigFiles = make(map[string][]packageConfigFile)

	var extraFiles map[string][]*FileInfo
	trace.WithRegion(context.Background(), "findextrafiles", func() {
		extraFiles, err = FindExtraFiles(cfg)
	})
	if err != nil {
		return err
	}
	for _, packageExtraFiles := range extraFiles {
		for _, ef := range packageExtraFiles {
			for _, de := range ef.Dirents {
				if de.Filename != "perm" {
					continue
				}
				return fmt.Errorf("invalid ExtraFilePaths or ExtraFileContents: cannot write extra files to user-controlled /perm partition")
			}
		}
	}

	if len(packageConfigFiles) > 0 {
		log.Printf("Including extra files for Go packages:")
		log.Printf("")
		for _, pkg := range args {
			if len(packageConfigFiles[pkg]) == 0 {
				continue
			}
			log.Printf("  %s", pkg)
			for _, configFile := range packageConfigFiles[pkg] {
				log.Printf("    will %s", configFile.kind)
				log.Printf("      from %s", configFile.path)
				log.Printf("      last modified: %s (%s ago)",
					configFile.lastModified.Format(time.RFC3339),
					time.Since(configFile.lastModified).Round(1*time.Second))
			}
			log.Printf("")
		}
	}

	if cfg.InternalCompatibilityFlags.InitPkg == "" {
		gokrazyInit := &gokrazyInit{
			root:             root,
			flagFileContents: pack.flagFileContents,
			envFileContents:  pack.envFileContents,
			buildTimestamp:   pack.buildTimestamp,
			dontStart:        pack.dontStart,
			waitForClock:     pack.waitForClock,
			basenames:        pack.basenames,
		}
		if cfg.InternalCompatibilityFlags.OverwriteInit != "" {
			return gokrazyInit.dump(cfg.InternalCompatibilityFlags.OverwriteInit)
		}

		var tmpdir string
		trace.WithRegion(context.Background(), "buildinit", func() {
			tmpdir, err = gokrazyInit.build(pack.moduleRoot)
		})
		if err != nil {
			return err
		}
		pack.initTmp = tmpdir

		initPath := filepath.Join(tmpdir, "init")

		fileIsELFOrFatal(initPath)

		gokrazy := root.mustFindDirent("gokrazy")
		gokrazy.Dirents = append(gokrazy.Dirents, &FileInfo{
			Filename: "init",
			FromHost: initPath,
		})
	}

	defaultPassword, updateHostname := updateflag.Value{
		Update: cfg.InternalCompatibilityFlags.Update,
	}.GetUpdateTarget(cfg.Hostname)
	update, err := cfg.Update.WithFallbackToHostSpecific(cfg.Hostname)
	if err != nil {
		return err
	}

	if update.HTTPPort == "" {
		update.HTTPPort = "80"
	}

	if update.HTTPSPort == "" {
		update.HTTPSPort = "443"
	}

	if update.Hostname == "" {
		update.Hostname = updateHostname
	}

	if update.HTTPPassword == "" && !update.NoPassword {
		pw, err := ensurePasswordFileExists(updateHostname, defaultPassword)
		if err != nil {
			return err
		}
		update.HTTPPassword = pw
	}

	pack.update = update

	for _, dir := range []string{"bin", "dev", "etc", "proc", "sys", "tmp", "perm", "lib", "run", "mnt"} {
		root.Dirents = append(root.Dirents, &FileInfo{
			Filename: dir,
		})
	}

	root.Dirents = append(root.Dirents, &FileInfo{
		Filename:    "var",
		SymlinkDest: "/perm/var",
	})

	mnt := root.mustFindDirent("mnt")
	for _, md := range cfg.MountDevices {
		if !strings.HasPrefix(md.Target, "/mnt/") {
			continue
		}
		rest := strings.TrimPrefix(md.Target, "/mnt/")
		rest = strings.TrimSuffix(rest, "/")
		if strings.Contains(rest, "/") {
			continue
		}
		mnt.Dirents = append(mnt.Dirents, &FileInfo{
			Filename: rest,
		})
	}

	// include lib/modules from kernelPackage dir, if present
	kernelDir, err := pack.PackageDir(cfg.KernelPackageOrDefault())
	if err != nil {
		return err
	}
	pack.kernelDir = kernelDir
	modulesDir := filepath.Join(kernelDir, "lib", "modules")
	if _, err := os.Stat(modulesDir); err == nil {
		log.Printf("Including loadable kernel modules from:")
		log.Printf("  %s", modulesDir)
		modules := &FileInfo{
			Filename: "modules",
		}
		trace.WithRegion(context.Background(), "kernelmod", func() {
			_, err = addToFileInfo(modules, modulesDir)
		})
		if err != nil {
			return err
		}
		lib := root.mustFindDirent("lib")
		lib.Dirents = append(lib.Dirents, modules)
	}

	etc := root.mustFindDirent("etc")
	tmpdir, err := os.MkdirTemp("", "gokrazy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	hl, err := hostLocaltime(tmpdir)
	if err != nil {
		return err
	}
	if hl != "" {
		etc.Dirents = append(etc.Dirents, &FileInfo{
			Filename: "localtime",
			FromHost: hl,
		})
	}
	etc.Dirents = append(etc.Dirents, &FileInfo{
		Filename:    "resolv.conf",
		SymlinkDest: "/tmp/resolv.conf",
	})
	etc.Dirents = append(etc.Dirents, &FileInfo{
		Filename: "hosts",
		FromLiteral: `127.0.0.1 localhost
::1 localhost
`,
	})
	etc.Dirents = append(etc.Dirents, &FileInfo{
		Filename:    "hostname",
		FromLiteral: cfg.Hostname,
	})

	ssl := &FileInfo{Filename: "ssl"}
	ssl.Dirents = append(ssl.Dirents, &FileInfo{
		Filename:    "ca-bundle.pem",
		FromLiteral: pack.systemCertsPEM,
	})

	pack.schema = "http"
	if update.CertPEM == "" || update.KeyPEM == "" {
		deployCertFile, deployKeyFile, err := getCertificate(cfg)
		if err != nil {
			return err
		}

		if deployCertFile != "" {
			b, err := os.ReadFile(deployCertFile)
			if err != nil {
				return err
			}
			update.CertPEM = strings.TrimSpace(string(b))

			b, err = os.ReadFile(deployKeyFile)
			if err != nil {
				return err
			}
			update.KeyPEM = strings.TrimSpace(string(b))
		}
	}
	if update.CertPEM != "" && update.KeyPEM != "" {
		pack.schema = "https"

		ssl.Dirents = append(ssl.Dirents, &FileInfo{
			Filename:    "gokrazy-web.pem",
			FromLiteral: update.CertPEM,
		})
		ssl.Dirents = append(ssl.Dirents, &FileInfo{
			Filename:    "gokrazy-web.key.pem",
			FromLiteral: update.KeyPEM,
		})
	}

	etc.Dirents = append(etc.Dirents, ssl)

	if !update.NoPassword {
		etc.Dirents = append(etc.Dirents, &FileInfo{
			Filename:    "gokr-pw.txt",
			Mode:        0400,
			FromLiteral: update.HTTPPassword,
		})
	}

	etc.Dirents = append(etc.Dirents, &FileInfo{
		Filename:    "http-port.txt",
		FromLiteral: update.HTTPPort,
	})

	etc.Dirents = append(etc.Dirents, &FileInfo{
		Filename:    "https-port.txt",
		FromLiteral: update.HTTPSPort,
	})

	var sbom []byte
	var sbomWithHash SBOMWithHash
	trace.WithRegion(context.Background(), "sbom", func() {
		sbom, sbomWithHash, err = generateSBOM(pack.FileCfg, foundBins)
	})
	if err != nil {
		return err
	}
	pack.sbom = sbom
	pack.sbomWithHash = sbomWithHash

	etcGokrazy := &FileInfo{Filename: "gokrazy"}
	etcGokrazy.Dirents = append(etcGokrazy.Dirents, &FileInfo{
		Filename:    "sbom.json",
		FromLiteral: string(sbom),
	})
	mountdevices, err := json.Marshal(cfg.MountDevices)
	if err != nil {
		return err
	}
	etcGokrazy.Dirents = append(etcGokrazy.Dirents, &FileInfo{
		Filename:    "mountdevices.json",
		FromLiteral: string(mountdevices),
	})
	etc.Dirents = append(etc.Dirents, etcGokrazy)

	empty := &FileInfo{Filename: ""}
	if paths := getDuplication(root, empty); len(paths) > 0 {
		return fmt.Errorf("root file system contains duplicate files: your config contains multiple packages that install %s", paths)
	}

	for pkg1, fs := range extraFiles {
		for _, fs1 := range fs {
			if paths := getDuplication(root, fs1); len(paths) > 0 {
				return fmt.Errorf("extra files of package %s collides with root file system: %v", pkg1, paths)
			}

			for pkg2, fs := range extraFiles {
				for _, fs2 := range fs {
					if pkg1 == pkg2 {
						continue
					}

					if paths := getDuplication(fs1, fs2); len(paths) > 0 {
						return fmt.Errorf("extra files of package %s collides with package %s: %v", pkg1, pkg2, paths)
					}
				}
			}

			if err := root.combine(fs1); err != nil {
				return fmt.Errorf("failed to add extra files from package %s: %v", pkg1, err)
			}
		}
	}

	return nil
}

func findBins(cfg *config.Struct, buildEnv *build.BuildEnv, bindir string, basenames map[string]string, moduleRoot string) (*FileInfo, []foundBin, error) {
	var found []foundBin
	result := FileInfo{Filename: ""}

	gokrazyMainPkgs, err := buildEnv.MainPackages(cfg.GokrazyPackagesOrDefault())
	if err != nil {
		return nil, nil, err
	}
	gokrazy := FileInfo{Filename: "gokrazy"}
	for _, pkg := range gokrazyMainPkgs {
		binPath := filepath.Join(bindir, pkg.Basename())
		fileIsELFOrFatal(binPath)
		gokrazy.Dirents = append(gokrazy.Dirents, &FileInfo{
			Filename: pkg.Basename(),
			FromHost: binPath,
		})
		found = append(found, foundBin{
			gokrazyPath: "/gokrazy/" + pkg.Basename(),
			hostPath:    binPath,
		})
	}

	if cfg.InternalCompatibilityFlags.InitPkg != "" {
		initMainPkgs, err := buildEnv.MainPackages([]string{cfg.InternalCompatibilityFlags.InitPkg})
		if err != nil {
			return nil, nil, err
		}
		for _, pkg := range initMainPkgs {
			binPath := filepath.Join(bindir, pkg.Basename())
			fileIsELFOrFatal(binPath)
			gokrazy.Dirents = append(gokrazy.Dirents, &FileInfo{
				Filename: pkg.Basename(),
				FromHost: binPath,
			})
			found = append(found, foundBin{
				gokrazyPath: "/gokrazy/init",
				hostPath:    binPath,
			})
		}
	}
	result.Dirents = append(result.Dirents, &gokrazy)

	mainPkgs, err := buildEnv.MainPackages(cfg.Packages)
	if err != nil {
		return nil, nil, err
	}
	user := FileInfo{Filename: "user"}
	for _, pkg := range mainPkgs {
		basename := pkg.Basename()
		if basenameOverride, ok := basenames[pkg.ImportPath]; ok {
			basename = basenameOverride
		}
		binPath := filepath.Join(bindir, basename)
		fileIsELFOrFatal(binPath)
		user.Dirents = append(user.Dirents, &FileInfo{
			Filename: basename,
			FromHost: binPath,
		})
		found = append(found, foundBin{
			gokrazyPath: "/user/" + basename,
			hostPath:    binPath,
		})
	}
	result.Dirents = append(result.Dirents, &user)
	return &result, found, nil
}

type archiveExtraction struct {
	dirs map[string]*FileInfo
}

func (ae *archiveExtraction) mkdirp(dir string) {
	if dir == "/" {
		return
	}
	parts := strings.Split(strings.TrimPrefix(dir, "/"), "/")
	parent := ae.dirs["."]
	for idx, part := range parts {
		path := strings.Join(parts[:1+idx], "/")
		if dir, ok := ae.dirs[path]; ok {
			parent = dir
			continue
		}
		subdir := &FileInfo{
			Filename: part,
		}
		parent.Dirents = append(parent.Dirents, subdir)
		ae.dirs[path] = subdir
		parent = subdir
	}
}

func (ae *archiveExtraction) extractArchive(path string) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	defer f.Close()
	rd := tar.NewReader(f)

	var latestTime time.Time
	for {
		header, err := rd.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return time.Time{}, err
		}

		filename := strings.TrimSuffix(header.Name, "/")

		fi := &FileInfo{
			Filename: filepath.Base(filename),
			Mode:     os.FileMode(header.Mode),
		}

		if latestTime.Before(header.ModTime) {
			latestTime = header.ModTime
		}

		dir := filepath.Dir(filename)
		ae.mkdirp(dir)
		parent := ae.dirs[dir]
		parent.Dirents = append(parent.Dirents, fi)

		switch header.Typeflag {
		case tar.TypeSymlink:
			fi.SymlinkDest = header.Linkname

		case tar.TypeDir:
			ae.dirs[filename] = fi

		default:
			b, err := io.ReadAll(rd)
			if err != nil {
				return time.Time{}, err
			}
			fi.FromLiteral = string(b)
		}
	}

	return latestTime, nil
}

// findExtraFilesInDir probes for extrafiles .tar files (possibly with an
// architecture suffix like _amd64), or whether dir itself exists.
func findExtraFilesInDir(dir string) (string, error) {
	targetArch := build.TargetArch()

	var err error
	for _, p := range []string{
		dir + "_" + targetArch + ".tar",
		dir + ".tar",
		dir,
	} {
		_, err = os.Stat(p)
		if err == nil {
			return p, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", err
}

func addExtraFilesFromDir(pkg, dir string, fi *FileInfo) error {
	ae := archiveExtraction{
		dirs: make(map[string]*FileInfo),
	}
	ae.dirs["."] = fi

	targetArch := build.TargetArch()

	effectivePath := dir + "_" + targetArch + ".tar"
	latestModTime, err := ae.extractArchive(effectivePath)
	if err != nil {
		return err
	}
	if len(fi.Dirents) == 0 {
		effectivePath = dir + ".tar"
		latestModTime, err = ae.extractArchive(effectivePath)
		if err != nil {
			return err
		}
	}
	if len(fi.Dirents) == 0 {
		effectivePath = dir
		latestModTime, err = addToFileInfo(fi, effectivePath)
		if err != nil {
			return err
		}
		if len(fi.Dirents) == 0 {
			return nil
		}
	}

	packageConfigFiles[pkg] = append(packageConfigFiles[pkg], packageConfigFile{
		kind:         "include extra files in the root file system",
		path:         effectivePath,
		lastModified: latestModTime,
	})

	return nil
}

func mkdirp(root *FileInfo, dir string) *FileInfo {
	if dir == "/" {
		return root
	}
	parts := strings.Split(strings.TrimPrefix(dir, "/"), "/")
	parent := root
	for _, part := range parts {
		subdir := &FileInfo{
			Filename: part,
		}
		parent.Dirents = append(parent.Dirents, subdir)
		parent = subdir
	}
	return parent
}

func FindExtraFiles(cfg *config.Struct) (map[string][]*FileInfo, error) {
	result := make(map[string][]*FileInfo)
	for pkg, pc := range cfg.PackageConfig {
		var fileInfos []*FileInfo

		for dest, path := range pc.ExtraFilePaths {
			root := &FileInfo{}
			if st, err := os.Stat(path); err == nil && st.Mode().IsRegular() {
				var err error
				path, err = filepath.Abs(path)
				if err != nil {
					return nil, err
				}
				// Copy a file from the host
				dir := mkdirp(root, filepath.Dir(dest))
				dir.Dirents = append(dir.Dirents, &FileInfo{
					Filename: filepath.Base(dest),
					FromHost: path,
				})
				packageConfigFiles[pkg] = append(packageConfigFiles[pkg], packageConfigFile{
					kind:         "include extra files in the root file system",
					path:         path,
					lastModified: st.ModTime(),
				})
			} else {
				// Check if the ExtraFilePaths entry refers to an extrafiles
				// .tar archive or an existing directory. If nothing can be
				// found, report the error so the user can fix their config.
				_, err := findExtraFilesInDir(path)
				if err != nil {
					return nil, fmt.Errorf("ExtraFilePaths of %s: %v", pkg, err)
				}
				// Copy a tarball or directory from the host
				dir := mkdirp(root, dest)
				if err := addExtraFilesFromDir(pkg, path, dir); err != nil {
					return nil, err
				}
			}

			fileInfos = append(fileInfos, root)
		}

		for dest, contents := range pc.ExtraFileContents {
			root := &FileInfo{}
			dir := mkdirp(root, filepath.Dir(dest))
			dir.Dirents = append(dir.Dirents, &FileInfo{
				Filename:    filepath.Base(dest),
				FromLiteral: contents,
			})
			packageConfigFiles[pkg] = append(packageConfigFiles[pkg], packageConfigFile{
				kind: "include extra files in the root file system",
			})
			fileInfos = append(fileInfos, root)
		}

		result[pkg] = fileInfos
	}
	return result, nil
}
