// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"golang.org/x/tools/go/packages"
)

// Source detects vulnerabilities in pkgs and computes slices of
//  - imports graph related to an import of a package with some
//    known vulnerabilities
//  - requires graph related to a require of a module with a
//    package that has some known vulnerabilities
//  - call graph leading to the use of a known vulnerable function
//    or method
func Source(pkgs []*packages.Package, cfg *Config) (*Result, error) {
	if !cfg.ImportsOnly {
		panic("call graph feature is currently unsupported")
	}

	modVulns, err := fetchVulnerabilities(cfg.Client, extractModules(pkgs))
	if err != nil {
		return nil, err
	}

	result := &Result{
		Imports:  &ImportGraph{Packages: make(map[int]*PkgNode)},
		Requires: &RequireGraph{Modules: make(map[int]*ModNode)},
	}
	vulnPkgImportSlice(pkgs, modVulns, result)
	// TODO(zpavlinovic): compute module and call graph slice.
	return result, nil
}

// pkgId is an id counter for nodes of Imports graph.
var pkgID int = 0

func nextPkgID() int {
	pkgID += 1
	return pkgID
}

// vulnPkgImportSlice computes the slice of pkg imports graph leading to imports of vulnerable
// packages in modVulns and stores the slice to result.
func vulnPkgImportSlice(pkgs []*packages.Package, modVulns moduleVulnerabilities, result *Result) {
	// analyzed contains information on packages analyzed thus far.
	// If a package is mapped to nil, this means it has been visited
	// but it does not lead to a vulnerable imports. Otherwise, a
	// visited package is mapped to Imports package node.
	analyzed := make(map[*packages.Package]*PkgNode)
	for _, pkg := range pkgs {
		// Top level packages that lead to vulnerable imports are
		// stored as result.Imports graph entry points.
		if e := vulnImportSlice(pkg, modVulns, result, analyzed); e != nil {
			result.Imports.Entries = append(result.Imports.Entries, e)
		}
	}
}

// vulnImportSlice checks if pkg has some vulnerabilities or transitively imports
// a package with known vulnerabilities. If that is the case, populates result.Imports
// graph with this reachability information and returns the result.Imports package
// node for pkg. Otherwise, returns nil.
func vulnImportSlice(pkg *packages.Package, modVulns moduleVulnerabilities, result *Result, analyzed map[*packages.Package]*PkgNode) *PkgNode {
	if pn, ok := analyzed[pkg]; ok {
		return pn
	}
	analyzed[pkg] = nil
	// Recursively compute which direct dependencies lead to an import of
	// a vulnerable package and remember the nodes of such dependencies.
	var onSlice []*PkgNode
	for _, imp := range pkg.Imports {
		if impNode := vulnImportSlice(imp, modVulns, result, analyzed); impNode != nil {
			onSlice = append(onSlice, impNode)
		}
	}

	// Check if pkg has known vulnerabilities.
	vulns := modVulns.VulnsForPackage(pkg.PkgPath)

	// If pkg is not vulnerable nor it transitively leads
	// to vulnerabilities, jump out.
	if len(onSlice) == 0 && len(vulns) == 0 {
		return nil
	}

	// Module id gets populated later.
	pkgNode := &PkgNode{
		Name: pkg.Name,
		Path: pkg.PkgPath,
	}
	analyzed[pkg] = pkgNode

	id := nextPkgID()
	result.Imports.Packages[id] = pkgNode

	// Save node predecessor information.
	for _, impSliceNode := range onSlice {
		impSliceNode.ImportedBy = append(impSliceNode.ImportedBy, id)
	}

	// Create Vuln entry for each symbol of known OSV entries for pkg.
	for _, osv := range vulns {
		for _, affected := range osv.Affected {
			if affected.Package.Name != pkgNode.Path {
				continue
			}
			for _, symbol := range affected.EcosystemSpecific.Symbols {
				vuln := &Vuln{
					OSV:        osv,
					Symbol:     symbol,
					PkgPath:    pkgNode.Path,
					ModPath:    modPath(pkg.Module),
					ImportSink: id,
				}
				result.Vulns = append(result.Vulns, vuln)
			}
		}
	}
	return pkgNode
}
