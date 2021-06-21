// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"golang.org/x/tools/go/packages"
)

// Returns module version of a package pkg. If the version is "" and the module is
// replaced by another module with the same path, replaced module version is returned.
// TODO(zpavlinovic): check if this is complete/correct.
func version(pkg *packages.Package) string {
	module := pkg.Module
	if module == nil {
		return ""
	}

	if module.Version != "" {
		return module.Version
	}

	if module.Replace == nil || module.Replace.Path != module.Path {
		return ""
	}
	return module.Replace.Version
}

// populateVersionInfo recursively populates pkgVersions for the input package pkg and its transitive dependencies.
func populatePkgVersions(pkg *packages.Package, pkgVersions map[string]string, seen map[string]bool) {
	if _, ok := seen[pkg.PkgPath]; ok {
		return
	}
	seen[pkg.PkgPath] = true

	version := version(pkg)
	if version != "" {
		pkgVersions[pkg.PkgPath] = version
	}

	for _, imp := range pkg.Imports {
		populatePkgVersions(imp, pkgVersions, seen)
	}
}

// PackageVersions computes a map from a path of every package in pkgs and
// its transitive dependencies to their module version. If module or its
// version are not present, the corresponding package is not in the map.
//
// Does not check for well-formedness of version strings. If such strings
// exist, the produced map can lead to confusing results down the line.
// (Well-formedness of version strings should be checked by external tools,
// such as using golang.org/x/tools/go/packages.Load to construct pkgs.)
func PackageVersions(pkgs []*packages.Package) map[string]string {
	pkgVersions := make(map[string]string)
	seen := make(map[string]bool)
	for _, pkg := range pkgs {
		populatePkgVersions(pkg, pkgVersions, seen)
	}
	return pkgVersions
}
