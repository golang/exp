// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gorelease is an experimental tool that helps module authors avoid common
// problems before releasing a new version of a module.
//
// Usage:
//
//    gorelease [-base={version|none}] [-version=version]
//
// Examples:
//
//    # Compare with the latest version and suggest a new version.
//    gorelease
//
//    # Compare with a specific version and suggest a new version.
//    gorelease -base=v1.2.3
//
//    # Compare with the latest version and check a specific new version for compatibility.
//    gorelease -version=v1.3.0
//
//    # Compare with a specific version and check a specific new version for compatibility.
//    gorelease -base=v1.2.3 -version=v1.3.0
//
// gorelease analyzes changes in the public API and dependencies of the main
// module. It compares a base version (set with -base) with the currently
// checked out revision. Given a proposed version to release (set with
// -version), gorelease reports whether the changes are consistent with
// semantic versioning. If no version is proposed with -version, gorelease
// suggests the lowest version consistent with semantic versioning.
//
// If there are no visible changes in the module's public API, gorelease
// accepts versions that increment the minor or patch version numbers. For
// example, if the base version is "v2.3.1", gorelease would accept "v2.3.2" or
// "v2.4.0" or any prerelease of those versions, like "v2.4.0-beta". If no
// version is proposed, gorelease would suggest "v2.3.2".
//
// If there are only backward compatible differences in the module's public
// API, gorelease only accepts versions that increment the minor version. For
// example, if the base version is "v2.3.1", gorelease would accept "v2.4.0"
// but not "v2.3.2".
//
// If there are incompatible API differences for a proposed version with
// major version 1 or higher, gorelease will exit with a non-zero status.
// Incompatible differences may only be released in a new major version, which
// requires creating a module with a different path. For example, if
// incompatible changes are made in the module "example.com/mod", a
// new major version must be released as a new module, "example.com/mod/v2".
// For a proposed version with major version 0, which allows incompatible
// changes, gorelease will describe all changes, but incompatible changes
// will not affect its exit status.
//
// For more information on semantic versioning, see https://semver.org.
//
// Note: gorelease does not accept build metadata in releases (like
// v1.0.0+debug). Although it is valid semver, the Go tool and other tools in
// the ecosystem do not support it, so its use is not recommended.
//
// gorelease accepts the following flags:
//
// -base=version: The version that the current version of the module will be
// compared against. This may be a version like "v1.5.2", a version query like
// "latest", or "none". If the version is "none", gorelease will not compare the
// current version against any previous version; it will only validate the
// current version. This is useful for checking the first release of a new major
// version. If -base is not specified, gorelease will attempt to infer a base
// version from the -version flag and available released versions.
//
// -version=version: The proposed version to be released. If specified,
// gorelease will confirm whether this version is consistent with changes made
// to the module's public API. gorelease will exit with a non-zero status if the
// version is not valid.
//
// gorelease is eventually intended to be merged into the go command
// as "go release". See golang.org/issues/26420.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/exp/apidiff"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/zip"
	"golang.org/x/tools/go/packages"
)

// IDEAS:
// * Should we suggest versions at all or should -version be mandatory?
// * Verify downstream modules have licenses. May need an API or library
//   for this. Be clear that we can't provide legal advice.
// * Internal packages may be relevant to submodules (for example,
//   golang.org/x/tools/internal/lsp is imported by golang.org/x/tools).
//   gorelease should detect whether this is the case and include internal
//   directories in comparison. It should be possible to opt out or specify
//   a different list of submodules.
// * Decide what to do about build constraints, particularly GOOS and GOARCH.
//   The API may be different on some platforms (e.g., x/sys).
//   Should gorelease load packages in multiple configurations in the same run?
//   Is it a compatible change if the same API is available for more platforms?
//   Is it an incompatible change for fewer?
//   How about cgo? Is adding a new cgo dependency an incompatible change?
// * Support splits and joins of nested modules. For example, if we are
//   proposing to tag a particular commit as both cloud.google.com/go v0.46.2
//   and cloud.google.com/go/storage v1.0.0, we should ensure that the sets of
//   packages provided by those modules are disjoint, and we should not report
//   the packages moved from one to the other as an incompatible change (since
//   the APIs are still compatible, just with a different module split).

// TODO(jayconrod):
// * Clean up overuse of fmt.Errorf.
// * Support migration to modules after v2.x.y+incompatible. Requires comparing
//   packages with different module paths.
// * Error when packages import from earlier major version of same module.
//   (this may be intentional; look for real examples first).
// * Mechanism to suppress error messages.

func main() {
	log.SetFlags(0)
	log.SetPrefix("gorelease: ")
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	success, err := runRelease(os.Stdout, wd, os.Args[1:])
	if err != nil {
		if _, ok := err.(*usageError); ok {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		} else {
			log.Fatal(err)
		}
	}
	if !success {
		os.Exit(1)
	}
}

// runRelease is the main function of gorelease. It's called by tests, so
// it writes to w instead of os.Stdout and returns an error instead of
// exiting.
func runRelease(w io.Writer, dir string, args []string) (success bool, err error) {
	// Validate arguments and flags. We'll print our own errors, since we want to
	// test without printing to stderr.
	fs := flag.NewFlagSet("gorelease", flag.ContinueOnError)
	fs.Usage = func() {}
	fs.SetOutput(ioutil.Discard)
	var baseVersion, releaseVersion string
	fs.StringVar(&baseVersion, "base", "", "previous version to compare against")
	fs.StringVar(&releaseVersion, "version", "", "proposed version to be released")
	if err := fs.Parse(args); err != nil {
		return false, &usageError{err: err}
	}

	if len(fs.Args()) > 0 {
		return false, usageErrorf("no arguments allowed")
	}
	if releaseVersion != "" {
		if semver.Build(releaseVersion) != "" {
			return false, usageErrorf("release version %q is not a canonical semantic version: build metadata is not supported", releaseVersion)
		}
		if c := semver.Canonical(releaseVersion); c != releaseVersion {
			return false, usageErrorf("release version %q is not a canonical semantic version", releaseVersion)
		}
	}
	if baseVersion != "" && semver.Canonical(baseVersion) == baseVersion && releaseVersion != "" {
		if cmp := semver.Compare(baseVersion, releaseVersion); cmp == 0 {
			return false, usageErrorf("-base and -version must be different")
		} else if cmp > 0 {
			return false, usageErrorf("base version (%q) must be lower than release version (%q)", baseVersion, releaseVersion)
		}
	}

	// Find the local module and repository root directories.
	modRoot, err := findModuleRoot(dir)
	if err != nil {
		return false, err
	}
	repoRoot := findRepoRoot(modRoot)

	// Load packages for the version to be released from the local directory.
	release, err := loadLocalModule(modRoot, repoRoot, releaseVersion)
	if err != nil {
		return false, err
	}

	// Find the base version if there is one, download it, and load packages from
	// the module cache.
	baseModPath := release.modPath // TODO(golang.org/issue/39666): allow different module path
	base, err := loadDownloadedModule(baseModPath, baseVersion, releaseVersion)
	if err != nil {
		return false, err
	}

	// Compare packages and check for other issues.
	report, err := makeReleaseReport(base, release)
	if err != nil {
		return false, err
	}
	if err := report.Text(w); err != nil {
		return false, err
	}
	return report.isSuccessful(), nil
}

type moduleInfo struct {
	modRoot         string // module root directory
	repoRoot        string // repository root directory (may be "")
	modPath         string // module path
	version         string // resolved version or "none"
	versionQuery    string // a query like "latest" or "dev-branch", if specified
	versionInferred bool   // true if the version was unspecified and inferred
	modPathMajor    string // major version suffix like "/v3" or ".v2"
	tagPrefix       string // prefix for version tags if module not in repo root

	goModPath string        // file path to go.mod
	goModData []byte        // content of go.mod
	goSumData []byte        // content of go.sum
	goModFile *modfile.File // parsed go.mod file

	diagnostics []string            // problems not related to loading specific packages
	pkgs        []*packages.Package // loaded packages with type information
}

// loadLocalModule loads information about a module and its packages from a
// local directory.
//
// modRoot is the directory containing the module's go.mod file.
//
// repoRoot is the root directory of the repository containing the module or "".
//
// version is a proposed version for the module or "".
func loadLocalModule(modRoot, repoRoot, version string) (m moduleInfo, err error) {
	if repoRoot != "" && !hasFilePathPrefix(modRoot, repoRoot) {
		return moduleInfo{}, fmt.Errorf("module root %q is not in repository root %q", modRoot, repoRoot)
	}

	// Load the go.mod file and check the module path and go version.
	m = moduleInfo{
		modRoot:   modRoot,
		repoRoot:  repoRoot,
		version:   version,
		goModPath: filepath.Join(modRoot, "go.mod"),
	}

	if version != "" && semver.Compare(version, "v0.0.0-99999999999999-zzzzzzzzzzzz") < 0 {
		m.diagnostics = append(m.diagnostics, fmt.Sprintf("Version %s is lower than most pseudo-versions. Consider releasing v0.1.0-0 instead.", version))
	}

	m.goModData, err = ioutil.ReadFile(m.goModPath)
	if err != nil {
		return moduleInfo{}, err
	}
	m.goModFile, err = modfile.ParseLax(m.goModPath, m.goModData, nil)
	if err != nil {
		return moduleInfo{}, err
	}
	if m.goModFile.Module == nil {
		return moduleInfo{}, fmt.Errorf("%s: module directive is missing", m.goModPath)
	}
	m.modPath = m.goModFile.Module.Mod.Path
	if err := checkModPath(m.modPath); err != nil {
		return moduleInfo{}, err
	}
	var ok bool
	_, m.modPathMajor, ok = module.SplitPathVersion(m.modPath)
	if !ok {
		// we just validated the path above.
		panic(fmt.Sprintf("could not find version suffix in module path %q", m.modPath))
	}
	if m.goModFile.Go == nil {
		m.diagnostics = append(m.diagnostics, "go.mod: go directive is missing")
	}

	// Determine the version tag prefix for the module within the repository.
	if repoRoot != "" && modRoot != repoRoot {
		if strings.HasPrefix(m.modPathMajor, ".") {
			m.diagnostics = append(m.diagnostics, fmt.Sprintf("%s: module path starts with gopkg.in and must be declared in the root directory of the repository", m.modPath))
		} else {
			codeDir := filepath.ToSlash(modRoot[len(repoRoot)+1:])
			var altGoModPath string
			if m.modPathMajor == "" {
				// module has no major version suffix.
				// codeDir must be a suffix of modPath.
				// tagPrefix is codeDir with a trailing slash.
				if strings.HasSuffix(m.modPath, "/"+codeDir) {
					m.tagPrefix = codeDir + "/"
				} else {
					m.diagnostics = append(m.diagnostics, fmt.Sprintf("%s: module path must end with %[2]q, since it is in subdirectory %[2]q", m.modPath, codeDir))
				}
			} else {
				if strings.HasSuffix(m.modPath, "/"+codeDir) {
					// module has a major version suffix and is in a major version subdirectory.
					// codeDir must be a suffix of modPath.
					// tagPrefix must not include the major version.
					m.tagPrefix = codeDir[:len(codeDir)-len(m.modPathMajor)+1]
					altGoModPath = modRoot[:len(modRoot)-len(m.modPathMajor)+1] + "go.mod"
				} else if strings.HasSuffix(m.modPath, "/"+codeDir+m.modPathMajor) {
					// module has a major version suffix and is not in a major version subdirectory.
					// codeDir + modPathMajor is a suffix of modPath.
					// tagPrefix is codeDir with a trailing slash.
					m.tagPrefix = codeDir + "/"
					altGoModPath = filepath.Join(modRoot, m.modPathMajor[1:], "go.mod")
				} else {
					m.diagnostics = append(m.diagnostics, fmt.Sprintf("%s: module path must end with %[2]q or %q, since it is in subdirectory %[2]q", m.modPath, codeDir, codeDir+m.modPathMajor))
				}
			}

			// Modules with major version suffixes can be defined in two places
			// (e.g., sub/go.mod and sub/v2/go.mod). They must not be defined in both.
			if altGoModPath != "" {
				if data, err := ioutil.ReadFile(altGoModPath); err == nil {
					if altModPath := modfile.ModulePath(data); m.modPath == altModPath {
						goModRel, _ := filepath.Rel(repoRoot, m.goModPath)
						altGoModRel, _ := filepath.Rel(repoRoot, altGoModPath)
						m.diagnostics = append(m.diagnostics, fmt.Sprintf("module is defined in two locations:\n\t%s\n\t%s", goModRel, altGoModRel))
					}
				}
			}
		}
	}

	// Load the module's packages.
	// We pack the module into a zip file and extract it to a temporary directory
	// as if it were published and downloaded. We'll detect any errors that would
	// occur (for example, invalid file names). We avoid loading it as the
	// main module.
	tmpModRoot, err := copyModuleToTempDir(m.modPath, m.modRoot)
	if err != nil {
		return moduleInfo{}, err
	}
	defer func() {
		if rerr := os.RemoveAll(tmpModRoot); err == nil && rerr != nil {
			err = fmt.Errorf("removing temporary module directory: %v", rerr)
		}
	}()
	tmpLoadDir, tmpGoModData, tmpGoSumData, err := prepareLoadDir(m.goModFile, m.modPath, tmpModRoot, version, false)
	if err != nil {
		return moduleInfo{}, err
	}
	defer func() {
		if rerr := os.RemoveAll(tmpLoadDir); err == nil && rerr != nil {
			err = fmt.Errorf("removing temporary load directory: %v", rerr)
		}
	}()
	var loadDiagnostics []string
	m.pkgs, loadDiagnostics, err = loadPackages(m.modPath, tmpModRoot, tmpLoadDir, tmpGoModData, tmpGoSumData)
	if err != nil {
		return moduleInfo{}, err
	}
	m.diagnostics = append(m.diagnostics, loadDiagnostics...)

	return m, nil
}

// loadDownloadedModule downloads a module and loads information about it and
// its packages from the module cache.
//
// modPath is the module's path.
//
// version is the version to load. It may be "none" (indicating nothing should
// be loaded), "" (the highest available version below max should be used), a
// version query (to be resolved with 'go list'), or a canonical version.
//
// If version is "" and max is not "", available versions greater than or equal
// to max will not be considered. Typically, loadDownloadedModule is used to
// load the base version, and max is the release version.
func loadDownloadedModule(modPath, version, max string) (m moduleInfo, err error) {
	// Check the module path and version.
	// If the version is a query, resolve it to a canonical version.
	m = moduleInfo{modPath: modPath}
	if err := checkModPath(modPath); err != nil {
		return moduleInfo{}, err
	}

	var ok bool
	_, m.modPathMajor, ok = module.SplitPathVersion(m.modPath)
	if !ok {
		// we just validated the path above.
		panic(fmt.Sprintf("could not find version suffix in module path %q", m.modPath))
	}

	if version == "none" {
		// We don't have a base version to compare against.
		m.version = "none"
		return m, nil
	}
	if version == "" {
		// Unspecified version: use the highest version below max.
		m.versionInferred = true
		if m.version, err = inferBaseVersion(modPath, max); err != nil {
			return moduleInfo{}, err
		}
		if m.version == "none" {
			return m, nil
		}
	} else if version != module.CanonicalVersion(version) {
		// Version query: find the real version.
		m.versionQuery = version
		if m.version, err = queryVersion(modPath, version); err != nil {
			return moduleInfo{}, err
		}
		if m.version != "none" && max != "" && semver.Compare(m.version, max) >= 0 {
			// TODO(jayconrod): reconsider this comparison for pseudo-versions in
			// general. A query might match different pseudo-versions over time,
			// depending on ancestor versions, so this might start failing with
			// no local change.
			return moduleInfo{}, fmt.Errorf("base version %s (%s) must be lower than release version %s", m.version, m.versionQuery, max)
		}
	} else {
		// Canonical version: make sure it matches the module path.
		if err := module.CheckPathMajor(version, m.modPathMajor); err != nil {
			// TODO(golang.org/issue/39666): don't assume this is the base version
			// or that we're comparing across major versions.
			return moduleInfo{}, fmt.Errorf("can't compare major versions: base version %s does not belong to module %s", version, modPath)
		}
		m.version = version
	}

	// Load packages.
	v := module.Version{Path: modPath, Version: m.version}
	if m.modRoot, err = downloadModule(v); err != nil {
		return moduleInfo{}, err
	}
	tmpLoadDir, tmpGoModData, tmpGoSumData, err := prepareLoadDir(nil, modPath, m.modRoot, m.version, true)
	if err != nil {
		return moduleInfo{}, err
	}
	defer func() {
		if rerr := os.RemoveAll(tmpLoadDir); err == nil && rerr != nil {
			err = fmt.Errorf("removing temporary load directory: %v", err)
		}
	}()
	if m.pkgs, _, err = loadPackages(modPath, m.modRoot, tmpLoadDir, tmpGoModData, tmpGoSumData); err != nil {
		return moduleInfo{}, err
	}

	// Attempt to load the mod file, if it exists.
	m.goModPath = filepath.Join(m.modRoot, "go.mod")
	if m.goModData, err = ioutil.ReadFile(m.goModPath); err != nil && !os.IsNotExist(err) {
		return moduleInfo{}, fmt.Errorf("reading go.mod: %v", err)
	}
	if err == nil {
		m.goModFile, err = modfile.ParseLax(m.goModPath, m.goModData, nil)
		if err != nil {
			return moduleInfo{}, err
		}
	}
	// The modfile might not exist, leading to err != nil. That's OK - continue.

	return m, nil
}

// makeReleaseReport returns a report comparing the current version of a
// module with a previously released version. The report notes any backward
// compatible and incompatible changes in the module's public API. It also
// diagnoses common problems, such as go.mod or go.sum being incomplete.
// The report recommends or validates a release version and indicates a
// version control tag to use (with an appropriate prefix, for modules not
// in the repository root directory).
func makeReleaseReport(base, release moduleInfo) (report, error) {
	if base.modPath != release.modPath {
		// TODO(golang.org/issue/39666): allow base and release path to be different.
		panic(fmt.Sprintf("base module path %q is different than release module path %q", base.modPath, release.modPath))
	}
	modPath := release.modPath

	// Compare each pair of packages.
	// Ignore internal packages.
	// If we don't have a base version to compare against,
	// just check the new packages for errors.
	shouldCompare := base.version != "none"
	isInternal := func(pkgPath string) bool {
		if !hasPathPrefix(pkgPath, modPath) {
			panic(fmt.Sprintf("package %s not in module %s", pkgPath, modPath))
		}
		for pkgPath != modPath {
			if path.Base(pkgPath) == "internal" {
				return true
			}
			pkgPath = path.Dir(pkgPath)
		}
		return false
	}
	r := report{
		base:    base,
		release: release,
	}
	for _, pair := range zipPackages(base.pkgs, release.pkgs) {
		basePkg, releasePkg := pair.base, pair.release
		switch {
		case releasePkg == nil:
			// Package removed
			if !isInternal(basePkg.PkgPath) || len(basePkg.Errors) > 0 {
				pr := packageReport{
					path:       basePkg.PkgPath,
					baseErrors: basePkg.Errors,
				}
				if !isInternal(basePkg.PkgPath) {
					pr.Report = apidiff.Report{
						Changes: []apidiff.Change{{
							Message:    "package removed",
							Compatible: false,
						}},
					}
				}
				r.addPackage(pr)
			}

		case basePkg == nil:
			// Package added
			if !isInternal(releasePkg.PkgPath) && shouldCompare || len(releasePkg.Errors) > 0 {
				pr := packageReport{
					path:          releasePkg.PkgPath,
					releaseErrors: releasePkg.Errors,
				}
				if !isInternal(releasePkg.PkgPath) && shouldCompare {
					// If we aren't comparing against a base version, don't say
					// "package added". Only report packages with errors.
					pr.Report = apidiff.Report{
						Changes: []apidiff.Change{{
							Message:    "package added",
							Compatible: true,
						}},
					}
				}
				r.addPackage(pr)
			}

		default:
			// Matched packages
			if !isInternal(basePkg.PkgPath) && basePkg.Name != "main" && releasePkg.Name != "main" {
				pr := packageReport{
					path:          basePkg.PkgPath,
					baseErrors:    basePkg.Errors,
					releaseErrors: releasePkg.Errors,
					Report:        apidiff.Changes(basePkg.Types, releasePkg.Types),
				}
				r.addPackage(pr)
			}
		}
	}

	if release.version != "" {
		r.validateVersion()
	} else {
		r.suggestVersion()
	}

	return r, nil
}

// findRepoRoot finds the root directory of the repository that contains dir.
// findRepoRoot returns "" if it can't find the repository root.
func findRepoRoot(dir string) string {
	vcsDirs := []string{".git", ".hg", ".svn", ".bzr"}
	d := filepath.Clean(dir)
	for {
		for _, vcsDir := range vcsDirs {
			if _, err := os.Stat(filepath.Join(d, vcsDir)); err == nil {
				return d
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// findModuleRoot finds the root directory of the module that contains dir.
func findModuleRoot(dir string) (string, error) {
	d := filepath.Clean(dir)
	for {
		if fi, err := os.Stat(filepath.Join(d, "go.mod")); err == nil && !fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return "", fmt.Errorf("%s: cannot find go.mod file", dir)
}

// checkModPath is like golang.org/x/mod/module.CheckPath, but it returns
// friendlier error messages for common mistakes.
//
// TODO(jayconrod): update module.CheckPath and delete this function.
func checkModPath(modPath string) error {
	if path.IsAbs(modPath) || filepath.IsAbs(modPath) {
		// TODO(jayconrod): improve error message in x/mod instead of checking here.
		return fmt.Errorf("module path %q must not be an absolute path.\nIt must be an address where your module may be found.", modPath)
	}
	if suffix := dirMajorSuffix(modPath); suffix == "v0" || suffix == "v1" {
		return fmt.Errorf("module path %q has major version suffix %q.\nA major version suffix is only allowed for v2 or later.", modPath, suffix)
	} else if strings.HasPrefix(suffix, "v0") {
		return fmt.Errorf("module path %q has major version suffix %q.\nA major version may not have a leading zero.", modPath, suffix)
	} else if strings.ContainsRune(suffix, '.') {
		return fmt.Errorf("module path %q has major version suffix %q.\nA major version may not contain dots.", modPath, suffix)
	}
	return module.CheckPath(modPath)
}

// inferBaseVersion returns an appropriate base version if one was not specified
// explicitly.
//
// If max is not "", inferBaseVersion returns the highest available release
// version of the module lower than max. Otherwise, inferBaseVersion returns the
// highest available release version. Pre-release versions are not considered.
// If there is no available version, and max appears to be the first release
// version (for example, "v0.1.0", "v2.0.0"), "none" is returned.
func inferBaseVersion(modPath, max string) (baseVersion string, err error) {
	defer func() {
		if err != nil {
			err = &baseVersionError{err: err}
		}
	}()

	versions, err := loadVersions(modPath)
	if err != nil {
		return "", err
	}

	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if semver.Prerelease(v) == "" &&
			(max == "" || semver.Compare(v, max) < 0) {
			return v, nil
		}
	}

	if max == "" || maybeFirstVersion(max) {
		return "none", nil
	}
	return "", fmt.Errorf("no versions found lower than %s", max)
}

// queryVersion returns the canonical version for a given module version query.
func queryVersion(modPath, query string) (resolved string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("could not resolve version %s@%s: %w", modPath, query, err)
		}
	}()
	if query == "upgrade" || query == "patch" {
		return "", errors.New("query is based on requirements in main go.mod file")
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	defer func() {
		if rerr := os.Remove(tmpDir); rerr != nil && err == nil {
			err = rerr
		}
	}()
	arg := modPath + "@" + query
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "--", arg)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	out, err := cmd.Output()
	if err != nil {
		return "", cleanCmdError(err)
	}
	return strings.TrimSpace(string(out)), nil
}

// loadVersions loads the list of versions for the given module using
// 'go list -m -versions'. The returned versions are sorted in ascending
// semver order.
func loadVersions(modPath string) ([]string, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer func() {
		if rerr := os.Remove(tmpDir); rerr != nil && err == nil {
			err = rerr
		}
	}()
	cmd := exec.Command("go", "list", "-m", "-versions", "--", modPath)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	out, err := cmd.Output()
	if err != nil {
		return nil, cleanCmdError(err)
	}
	versions := strings.Fields(string(out))
	if len(versions) > 0 {
		versions = versions[1:] // skip module path
	}

	// Sort versions defensively. 'go list -m -versions' should always returns
	// a sorted list of versions, but it's fast and easy to sort them here, too.
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})
	return versions, nil
}

// maybeFirstVersion returns whether v appears to be the first version
// of a module.
func maybeFirstVersion(v string) bool {
	major, minor, patch, _, _, err := parseVersion(v)
	if err != nil {
		return false
	}
	if major == "0" {
		return minor == "0" && patch == "0" ||
			minor == "0" && patch == "1" ||
			minor == "1" && patch == "0"
	}
	return minor == "0" && patch == "0"
}

// dirMajorSuffix returns a major version suffix for a slash-separated path.
// For example, for the path "foo/bar/v2", dirMajorSuffix would return "v2".
// If no major version suffix is found, "" is returned.
//
// dirMajorSuffix is less strict than module.SplitPathVersion so that incorrect
// suffixes like "v0", "v02", "v1.2" can be detected. It doesn't handle
// special cases for gopkg.in paths.
func dirMajorSuffix(path string) string {
	i := len(path)
	for i > 0 && ('0' <= path[i-1] && path[i-1] <= '9') || path[i-1] == '.' {
		i--
	}
	if i <= 1 || i == len(path) || path[i-1] != 'v' || (i > 1 && path[i-2] != '/') {
		return ""
	}
	return path[i-1:]
}

// copyModuleToTempDir copies module files from modRoot to a subdirectory of
// scratchDir. Submodules, vendor directories, and irregular files are excluded.
// An error is returned if the module contains any files or directories that
// can't be included in a module zip file (due to special characters,
// excessive sizes, etc.).
func copyModuleToTempDir(modPath, modRoot string) (dir string, err error) {
	// Generate a fake version consistent with modPath. We need a canonical
	// version to create a zip file.
	version := "v0.0.0-gorelease"
	_, majorPathSuffix, _ := module.SplitPathVersion(modPath)
	if majorPathSuffix != "" {
		version = majorPathSuffix[1:] + ".0.0-gorelease"
	}
	m := module.Version{Path: modPath, Version: version}

	zipFile, err := ioutil.TempFile("", "gorelease-*.zip")
	if err != nil {
		return "", err
	}
	defer func() {
		zipFile.Close()
		os.Remove(zipFile.Name())
	}()

	dir, err = ioutil.TempDir("", "gorelease")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
			dir = ""
		}
	}()

	if err := zip.CreateFromDir(zipFile, m, modRoot); err != nil {
		var e zip.FileErrorList
		if errors.As(err, &e) {
			return "", e
		}
		return "", err
	}
	if err := zipFile.Close(); err != nil {
		return "", err
	}
	if err := zip.Unzip(dir, m, zipFile.Name()); err != nil {
		return "", err
	}
	return dir, nil
}

// downloadModule downloads a specific version of a module to the
// module cache using 'go mod download'.
func downloadModule(m module.Version) (modRoot string, err error) {
	defer func() {
		if err != nil {
			err = &downloadError{m: m, err: cleanCmdError(err)}
		}
	}()

	// Run 'go mod download' from a temporary directory to avoid needing to load
	// go.mod from gorelease's working directory (or a parent).
	// go.mod may be broken, and we don't need it.
	// TODO(golang.org/issue/36812): 'go mod download' reads go.mod even though
	// we don't need information about the main module or the build list.
	// If it didn't read go.mod in this case, we wouldn't need a temp directory.
	tmpDir, err := ioutil.TempDir("", "gorelease-download")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpDir)
	cmd := exec.Command("go", "mod", "download", "-json", "--", m.Path+"@"+m.Version)
	cmd.Dir = tmpDir
	out, err := cmd.Output()
	var xerr *exec.ExitError
	if err != nil {
		var ok bool
		if xerr, ok = err.(*exec.ExitError); !ok {
			return "", err
		}
	}

	// If 'go mod download' exited unsuccessfully but printed well-formed JSON
	// with an error, return that error.
	parsed := struct{ Dir, Error string }{}
	if jsonErr := json.Unmarshal(out, &parsed); jsonErr != nil {
		if xerr != nil {
			return "", cleanCmdError(xerr)
		}
		return "", jsonErr
	}
	if parsed.Error != "" {
		return "", errors.New(parsed.Error)
	}
	if xerr != nil {
		return "", cleanCmdError(xerr)
	}
	return parsed.Dir, nil
}

// prepareLoadDir creates a temporary directory and a go.mod file that requires
// the module being loaded. go.sum is copied if present.
//
// modFile is the pre-parsed go.mod file. If non-nil, its requirements and
// go version will be copied so that incomplete and out-of-date requirements
// may be reported later.
//
// modPath is the module's path.
//
// version is the version of the module being loaded. If must be canonical
// for modules loaded from the cache. Otherwise, it may be empty (for example,
// when no release version is proposed).
//
// cached indicates whether the module is being loaded from the module cache.
// If true, the module can be referenced with a simple requirement.
// If false, the module will be referenced with a local replace directive.
func prepareLoadDir(modFile *modfile.File, modPath, modRoot, version string, cached bool) (dir string, goModData, goSumData []byte, err error) {
	if module.Check(modPath, version) != nil {
		// If no version is proposed or if the version isn't valid, use a fake
		// version that matches the module's major version suffix. If the version
		// is invalid, that will be reported elsewhere.
		version = "v0.0.0-gorelease"
		if _, pathMajor, _ := module.SplitPathVersion(modPath); pathMajor != "" {
			version = pathMajor[1:] + ".0.0-gorelease"
		}
	}

	dir, err = ioutil.TempDir("", "gorelease-load")
	if err != nil {
		return "", nil, nil, err
	}

	f := &modfile.File{}
	f.AddModuleStmt("gorelease-load-module")
	f.AddRequire(modPath, version)
	if !cached {
		f.AddReplace(modPath, version, modRoot, "")
	}
	if modFile != nil {
		if modFile.Go != nil {
			f.AddGoStmt(modFile.Go.Version)
		}
		for _, r := range modFile.Require {
			f.AddRequire(r.Mod.Path, r.Mod.Version)
		}
	}
	goModData, err = f.Format()
	if err != nil {
		return "", nil, nil, err
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), goModData, 0666); err != nil {
		return "", nil, nil, err
	}

	goSumData, err = ioutil.ReadFile(filepath.Join(modRoot, "go.sum"))
	if err != nil && !os.IsNotExist(err) {
		return "", nil, nil, err
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "go.sum"), goSumData, 0666); err != nil {
		return "", nil, nil, err
	}

	return dir, goModData, goSumData, nil
}

// loadPackages returns a list of all packages in the module modPath, sorted by
// package path. modRoot is the module root directory, but packages are loaded
// from loadDir, which must contain go.mod and go.sum containing goModData and
// goSumData.
//
// We load packages from a temporary external module so that replace and exclude
// directives are not applied. The loading process may also modify go.mod and
// go.sum, and we want to detect and report differences.
//
// Package loading errors will be returned in the Errors field of each package.
// Other diagnostics (such as the go.sum file being incomplete) will be
// returned through diagnostics.
// err will be non-nil in case of a fatal error that prevented packages
// from being loaded.
func loadPackages(modPath, modRoot, loadDir string, goModData, goSumData []byte) (pkgs []*packages.Package, diagnostics []string, err error) {
	// List packages in the module.
	// We can't just load example.com/mod/... because that might include packages
	// in nested modules. We also can't filter packages from the output of
	// packages.Load, since it doesn't tell us which module they came from.
	//
	// TODO(golang.org/issue/41456): this command fails in -mod=readonly mode
	// if sums are missing, which they always are for downloaded modules. In
	// Go 1.16, -mod=readonly is the default, and -mod=mod may eventually be
	// removed, so we should avoid -mod=mod here. Lazy loading may also require
	// changes to temporary module requirements.
	//
	// Instead of running this command, we should make a list of importable
	// packages by walking the directory tree. With such a list,
	// in prepareLoadDir, we could generate a temporary package that imports
	// all of them, then 'go get -d' that package to ensure no requirements
	// or sums are missing.
	format := fmt.Sprintf(`{{if .Module}}{{if eq .Module.Path %q}}{{.ImportPath}}{{end}}{{end}}`, modPath)
	cmd := exec.Command("go", "list", "-mod=mod", "-e", "-f", format, "--", modPath+"/...")
	cmd.Dir = loadDir
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, cleanCmdError(err)
	}
	var pkgPaths []string
	for len(out) > 0 {
		eol := bytes.IndexByte(out, '\n')
		if eol < 0 {
			eol = len(out)
		}
		pkgPaths = append(pkgPaths, string(out[:eol]))
		out = out[eol+1:]
	}

	// Load packages.
	// TODO(jayconrod): if there are errors loading packages in the release
	// version, try loading in the release directory. Errors there would imply
	// that packages don't load without replace / exclude directives.
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		Dir:  loadDir,
	}
	if len(pkgPaths) > 0 {
		pkgs, err = packages.Load(cfg, pkgPaths...)
		if err != nil {
			return nil, nil, err
		}
	}

	// Sort the returned packages by path.
	// packages.Load makes no guarantee about the order of returned packages.
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].PkgPath < pkgs[j].PkgPath
	})

	// Trim modRoot from file paths in errors.
	prefix := modRoot + string(os.PathSeparator)
	for _, pkg := range pkgs {
		for i := range pkg.Errors {
			pkg.Errors[i].Pos = strings.TrimPrefix(pkg.Errors[i].Pos, prefix)
		}
	}

	// Report new requirements in go.mod.
	goModPath := filepath.Join(loadDir, "go.mod")
	loadReqs := func(data []byte) ([]string, error) {
		modFile, err := modfile.ParseLax(goModPath, data, nil)
		if err != nil {
			return nil, err
		}
		lines := make([]string, len(modFile.Require))
		for i, req := range modFile.Require {
			lines[i] = req.Mod.String()
		}
		sort.Strings(lines)
		return lines, nil
	}

	oldReqs, err := loadReqs(goModData)
	if err != nil {
		return nil, nil, err
	}
	newGoModData, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return nil, nil, err
	}
	newReqs, err := loadReqs(newGoModData)
	if err != nil {
		return nil, nil, err
	}

	oldMap := make(map[string]bool)
	for _, req := range oldReqs {
		oldMap[req] = true
	}
	var missing []string
	for _, req := range newReqs {
		if !oldMap[req] {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("go.mod: the following requirements are needed\n\t%s\nRun 'go mod tidy' to add missing requirements.", strings.Join(missing, "\n\t")))
		return pkgs, diagnostics, nil
	}

	newGoSumData, err := ioutil.ReadFile(filepath.Join(loadDir, "go.sum"))
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	if !bytes.Equal(goSumData, newGoSumData) {
		diagnostics = append(diagnostics, "go.sum: one or more sums are missing.\nRun 'go mod tidy' to add missing sums.")
	}

	return pkgs, diagnostics, nil
}

type packagePair struct {
	base, release *packages.Package
}

// zipPackages combines two lists of packages, sorted by package path,
// and returns a sorted list of pairs of packages with matching paths.
// If a package is in one list but not the other (because it was added or
// removed between releases), a pair will be returned with a nil
// base or release field.
func zipPackages(basePkgs, releasePkgs []*packages.Package) []packagePair {
	baseIndex, releaseIndex := 0, 0
	var pairs []packagePair
	for baseIndex < len(basePkgs) || releaseIndex < len(releasePkgs) {
		var basePkg, releasePkg *packages.Package
		if baseIndex < len(basePkgs) {
			basePkg = basePkgs[baseIndex]
		}
		if releaseIndex < len(releasePkgs) {
			releasePkg = releasePkgs[releaseIndex]
		}

		var pair packagePair
		if basePkg != nil && (releasePkg == nil || basePkg.PkgPath < releasePkg.PkgPath) {
			// Package removed
			pair = packagePair{basePkg, nil}
			baseIndex++
		} else if releasePkg != nil && (basePkg == nil || releasePkg.PkgPath < basePkg.PkgPath) {
			// Package added
			pair = packagePair{nil, releasePkg}
			releaseIndex++
		} else {
			// Matched packages.
			pair = packagePair{basePkg, releasePkg}
			baseIndex++
			releaseIndex++
		}
		pairs = append(pairs, pair)
	}
	return pairs
}
