// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gorelease is an experimental tool that helps module authors avoid common
// problems before releasing a new version of a module.
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
// gorelease accepts the following flags:
//
// -base=version: The version that the current version of the module will be
// compared against. The version must be a semantic version (for example,
// "v2.3.4") or "none". If the version is "none", gorelease will not compare the
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

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/zip"

	"golang.org/x/exp/apidiff"
	"golang.org/x/tools/go/packages"
)

// IDEAS:
// * Should we suggest versions at all or should -version be mandatory?
// * 'gorelease path1@version1 path2@version2' should compare two arbitrary
//   modules. Useful for comparing differences in forks.
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
// * Allow -base to be an arbitrary revision name that resolves to a version
//   or pseudo-version.
// * Don't accept -version that increments minor or patch version by more than 1
//   or increments the minor version without zeroing the patch, compared with
//   existing versions. Note that -base may be distant from -version,
//   for example, when reverting an incompatible change accidentally released.
// * Report errors when packages can't be loaded without replace / exclude.
// * Clean up overuse of fmt.Errorf.
// * Support -json output.
// * Don't suggest a release tag that already exists.
// * Suggest a minor release if dependency has been bumped by minor version.
// * Updating go version, either in the main module or in a dependency that
//   provides packages transitively imported by non-internal, non-test packages
//   in the main module, should require a minor version bump.
// * Support migration to modules after v2.x.y+incompatible. Requires comparing
//   packages with different module paths.
// * Error when packages import from earlier major version of same module.
//   (this may be intentional; look for real examples first).
// * Check that proposed prerelease will not sort below pseudo-versions.
// * Error messages point to HTML documentation.
// * Positional arguments should specify which packages to check. Without
//   these, we check all non-internal packages in the module.
// * Mechanism to suppress error messages.
// * Check that the main module does not transitively require a newer version
//   of itself.
// * Invalid file names and import paths should be reported sensibly.
//   golang.org/x/mod/zip should return structured errors for this.

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
	if baseVersion != "" && baseVersion != "none" {
		if c := semver.Canonical(baseVersion); c != baseVersion {
			return false, usageErrorf("base version %q is not a canonical semantic version", baseVersion)
		}
	}
	if releaseVersion != "" {
		if c := semver.Canonical(releaseVersion); c != releaseVersion {
			return false, usageErrorf("release version %q is not a canonical semantic version", releaseVersion)
		}
	}
	if baseVersion != "" && baseVersion != "none" && releaseVersion != "" {
		if cmp := semver.Compare(baseVersion, releaseVersion); cmp == 0 {
			return false, usageErrorf("-base and -version must be different")
		} else if cmp > 0 {
			return false, usageErrorf("base version (%q) must be lower than release version (%q)", baseVersion, releaseVersion)
		}
	}

	modRoot, err := findModuleRoot(dir)
	if err != nil {
		return false, err
	}
	repoRoot := findRepoRoot(modRoot)
	if repoRoot == "" {
		repoRoot = modRoot
	}

	report, err := makeReleaseReport(modRoot, repoRoot, baseVersion, releaseVersion)
	if err != nil {
		return false, err
	}
	if err := report.Text(w); err != nil {
		return false, err
	}
	return report.isSuccessful(), nil
}

// makeReleaseReport returns a report comparing the current version of a
// module with a previously released version. The report notes any backward
// compatible and incompatible changes in the module's public API. It also
// diagnoses common problems, such as go.mod or go.sum being incomplete.
// The report recommends or validates a release version and indicates a
// version control tag to use (with an appropriate prefix, for modules not
// in the repository root directory).
//
// modRoot is the directory containing the module's go.mod file. It must not
// be "".
//
// repoRoot the root directory of the version control repository containing
// modRoot. It must not be ""; if there is no known repository, repoRoot
// should be set to modRoot.
//
// baseVersion is a previously released version of the module to compare.
// If baseVersion is "", a base version will be detected automatically, based
// on releaseVersion or the latest available version of the module.
// If baseVersion is "none", no comparison will be performed, and
// the returned report will only describe problems with the release version.
//
// releaseVersion is the proposed version for the module in dir.
// If releaseVersion is "", the report will suggest a release version based on
// changes to the public API.
func makeReleaseReport(modRoot, repoRoot, baseVersion, releaseVersion string) (report, error) {
	if !hasFilePathPrefix(modRoot, repoRoot) {
		// runRelease should always make sure this is true.
		return report{}, fmt.Errorf("module root %q is not in repository root %q", modRoot, repoRoot)
	}

	// Read the module path from the go.mod file.
	goModPath := filepath.Join(modRoot, "go.mod")
	goModData, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return report{}, err
	}
	modFile, err := modfile.ParseLax(goModPath, goModData, nil)
	if err != nil {
		return report{}, err
	}
	if modFile.Module == nil {
		return report{}, fmt.Errorf("%s: module directive is missing", goModPath)
	}
	modPath := modFile.Module.Mod.Path
	if err := checkModPath(modPath); err != nil {
		return report{}, err
	}
	_, modPathMajor, ok := module.SplitPathVersion(modPath)
	if !ok {
		// we just validated the path above.
		panic(fmt.Sprintf("could not find version suffix in module path %q", modPath))
	}

	baseVersionInferred := baseVersion == ""
	if baseVersionInferred {
		if baseVersion, err = inferBaseVersion(modPath, releaseVersion); err != nil {
			return report{}, err
		}
	}
	if baseVersion != "none" {
		if err := module.Check(modPath, baseVersion); err != nil {
			return report{}, fmt.Errorf("can't compare major versions: base version %s does not belong to module %s", baseVersion, modPath)
		}
	}
	// releaseVersion is checked by report.validateVersion.

	// Check if a go version is present in go.mod.
	var diagnostics []string
	if modFile.Go == nil {
		diagnostics = append(diagnostics, "go.mod: go directive is missing")
	}

	// Determine the version tag prefix for the module within the repository.
	tagPrefix := ""
	if modRoot != repoRoot {
		if strings.HasPrefix(modPathMajor, ".") {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: module path starts with gopkg.in and must be declared in the root directory of the repository", modPath))
		} else {
			codeDir := filepath.ToSlash(modRoot[len(repoRoot)+1:])
			var altGoModPath string
			if modPathMajor == "" {
				// module has no major version suffix.
				// codeDir must be a suffix of modPath.
				// tagPrefix is codeDir with a trailing slash.
				if strings.HasSuffix(modPath, "/"+codeDir) {
					tagPrefix = codeDir + "/"
				} else {
					diagnostics = append(diagnostics, fmt.Sprintf("%s: module path must end with %[2]q, since it is in subdirectory %[2]q", modPath, codeDir))
				}
			} else {
				if strings.HasSuffix(modPath, "/"+codeDir) {
					// module has a major version suffix and is in a major version subdirectory.
					// codeDir must be a suffix of modPath.
					// tagPrefix must not include the major version.
					tagPrefix = codeDir[:len(codeDir)-len(modPathMajor)+1]
					altGoModPath = modRoot[:len(modRoot)-len(modPathMajor)+1] + "go.mod"
				} else if strings.HasSuffix(modPath, "/"+codeDir+modPathMajor) {
					// module has a major version suffix and is not in a major version subdirectory.
					// codeDir + modPathMajor is a suffix of modPath.
					// tagPrefix is codeDir with a trailing slash.
					tagPrefix = codeDir + "/"
					altGoModPath = filepath.Join(modRoot, modPathMajor[1:], "go.mod")
				} else {
					diagnostics = append(diagnostics, fmt.Sprintf("%s: module path must end with %[2]q or %q, since it is in subdirectory %[2]q", modPath, codeDir, codeDir+modPathMajor))
				}
			}

			// Modules with major version suffixes can be defined in two places
			// (e.g., sub/go.mod and sub/v2/go.mod). They must not be defined in both.
			if altGoModPath != "" {
				if data, err := ioutil.ReadFile(altGoModPath); err == nil {
					if altModPath := modfile.ModulePath(data); modPath == altModPath {
						goModRel, _ := filepath.Rel(repoRoot, goModPath)
						altGoModRel, _ := filepath.Rel(repoRoot, altGoModPath)
						diagnostics = append(diagnostics, fmt.Sprintf("module is defined in two locations:\n\t%s\n\t%s", goModRel, altGoModRel))
					}
				}
			}
		}
	}

	// Load the base version of the module.
	// We download it into the module cache, then create a go.mod in a temporary
	// directory that requires it. It's important that we don't load the module
	// as the main module so that replace and exclude directives are not applied.
	var basePkgs []*packages.Package
	if baseVersion != "none" {
		baseMod := module.Version{Path: modPath, Version: baseVersion}
		baseModRoot, err := downloadModule(baseMod)
		if err != nil {
			return report{}, err
		}
		baseLoadDir, goModData, goSumData, err := prepareExternalDirForBase(modPath, baseVersion, baseModRoot)
		if err != nil {
			return report{}, err
		}
		defer os.RemoveAll(baseLoadDir)
		if basePkgs, _, err = loadPackages(modPath, baseModRoot, baseLoadDir, goModData, goSumData); err != nil {
			return report{}, err
		}
	}

	// Load the release version of the module.
	// We pack it into a zip file and extract it to a temporary directory as if
	// it were published and downloaded. We'll detect any errors that would occur
	// (for example, invalid file name). Again, we avoid loading it as the
	// main module.
	releaseModRoot, err := copyModuleToTempDir(modPath, modRoot)
	if err != nil {
		return report{}, err
	}
	defer os.RemoveAll(releaseModRoot)
	releaseLoadDir, goModData, goSumData, err := prepareExternalDirForRelease(modFile, modPath, releaseModRoot)
	if err != nil {
		return report{}, nil
	}
	releasePkgs, loadDiagnostics, err := loadPackages(modPath, releaseModRoot, releaseLoadDir, goModData, goSumData)
	if err != nil {
		return report{}, err
	}
	diagnostics = append(diagnostics, loadDiagnostics...)

	// Compare each pair of packages.
	// Ignore internal packages.
	// If we don't have a base version to compare against,
	// just check the new packages for errors.
	shouldCompare := baseVersion != "none"
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
		modulePath:          modPath,
		baseVersion:         baseVersion,
		baseVersionInferred: baseVersionInferred,
		releaseVersion:      releaseVersion,
		tagPrefix:           tagPrefix,
		diagnostics:         diagnostics,
	}
	for _, pair := range zipPackages(basePkgs, releasePkgs) {
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

	if releaseVersion != "" {
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

// inferBaseVersion returns an appropriate base version if one was not
// specified explicitly.
//
// If releaseVersion is not "", inferBaseVersion returns the highest available
// release version of the module lower than releaseVersion.
// Otherwise, inferBaseVersion returns the highest available release version.
// Pre-release versions are not considered. If there is no available version,
// and releaseVersion appears to be the first release version (for example,
// "v0.1.0", "v2.0.0"), "none" is returned.
func inferBaseVersion(modPath, releaseVersion string) (baseVersion string, err error) {
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
			(releaseVersion == "" || semver.Compare(v, releaseVersion) < 0) {
			return v, nil
		}
	}

	if releaseVersion == "" || maybeFirstVersion(releaseVersion) {
		return "none", nil
	}
	return "", fmt.Errorf("no versions found lower than %s", releaseVersion)
}

// loadVersions loads the list of versions for the given module using
// 'go list -m -versions'. The returned versions are sorted in ascending
// semver order.
func loadVersions(modPath string) ([]string, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpDir)
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

// hasPathPrefix reports whether the slash-separated path s
// begins with the elements in prefix.
// Copied from cmd/go/internal/str.HasPathPrefix.
func hasPathPrefix(s, prefix string) bool {
	if len(s) == len(prefix) {
		return s == prefix
	}
	if prefix == "" {
		return true
	}
	if len(s) > len(prefix) {
		if prefix[len(prefix)-1] == '/' || s[len(prefix)] == '/' {
			return s[:len(prefix)] == prefix
		}
	}
	return false
}

// hasFilePathPrefix reports whether the filesystem path s
// begins with the elements in prefix.
// Copied from cmd/go/internal/str.HasFilePathPrefix.
func hasFilePathPrefix(s, prefix string) bool {
	sv := strings.ToUpper(filepath.VolumeName(s))
	pv := strings.ToUpper(filepath.VolumeName(prefix))
	s = s[len(sv):]
	prefix = prefix[len(pv):]
	switch {
	default:
		return false
	case sv != pv:
		return false
	case len(s) == len(prefix):
		return s == prefix
	case prefix == "":
		return true
	case len(s) > len(prefix):
		if prefix[len(prefix)-1] == filepath.Separator {
			return strings.HasPrefix(s, prefix)
		}
		return s[len(prefix)] == filepath.Separator && s[:len(prefix)] == prefix
	}
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
			return "", xerr
		}
		return "", jsonErr
	}
	if parsed.Error != "" {
		return "", errors.New(parsed.Error)
	}
	if xerr != nil {
		return "", xerr
	}
	return parsed.Dir, nil
}

// prepareExternalDirForBase creates a temporary directory and a go.mod file
// that requires the module at the given version. go.sum is copied if present.
func prepareExternalDirForBase(modPath, version, modRoot string) (dir string, goModData, goSumData []byte, err error) {
	dir, err = ioutil.TempDir("", "gorelease-base")
	if err != nil {
		return "", nil, nil, err
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, `module gorelease-base-module

require %s %s
`, modPath, version)
	goModData = buf.Bytes()
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

// prepareExternalDirForRelease creates a temporary directory and a go.mod file
// that requires the module and replaces it with modRoot. go.sum is copied
// if present.
func prepareExternalDirForRelease(modFile *modfile.File, modPath, modRoot string) (dir string, goModData, goSumData []byte, err error) {
	dir, err = ioutil.TempDir("", "gorelease-release")
	if err != nil {
		return "", nil, nil, err
	}

	version := "v0.0.0-gorelease"
	if _, pathMajor, _ := module.SplitPathVersion(modPath); pathMajor != "" {
		version = pathMajor[1:] + ".0.0-gorelease"
	}

	f := &modfile.File{}
	f.AddModuleStmt("gorelease-release-module")
	f.AddRequire(modPath, version)
	f.AddReplace(modPath, version, modRoot, "")
	for _, r := range modFile.Require {
		f.AddRequire(r.Mod.Path, r.Mod.Version)
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
	format := fmt.Sprintf(`{{if .Module}}{{if eq .Module.Path %q}}{{.ImportPath}}{{end}}{{end}}`, modPath)
	cmd := exec.Command("go", "list", "-e", "-f", format, "--", modPath+"/...")
	cmd.Dir = loadDir
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
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

	// Trim modRoot from file paths in errors.
	prefix := modRoot + string(os.PathSeparator)
	for _, pkg := range pkgs {
		for i := range pkg.Errors {
			pkg.Errors[i].Pos = strings.TrimPrefix(pkg.Errors[i].Pos, prefix)
		}
	}

	// Report new requirements in go.mod.
	goModPath := filepath.Join(loadDir, "go.mod")
	loadReqs := func(data []byte) (string, error) {
		buf := &bytes.Buffer{}
		modFile, err := modfile.ParseLax(goModPath, data, nil)
		if err != nil {
			return "", err
		}
		for _, req := range modFile.Require {
			fmt.Fprintf(buf, "%v\n", req.Mod)
		}
		return buf.String(), nil
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

	goModChanged := oldReqs != newReqs
	if goModChanged {
		diagnostics = append(diagnostics, "go.mod: requirements are incomplete.\nRun 'go mod tidy' to add missing requirements.")
	}

	if !goModChanged {
		newGoSumData, err := ioutil.ReadFile(filepath.Join(loadDir, "go.sum"))
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
		if !bytes.Equal(goSumData, newGoSumData) {
			diagnostics = append(diagnostics, "go.sum: one or more sums are missing.\nRun 'go mod tidy' to add missing sums.")
		}
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
