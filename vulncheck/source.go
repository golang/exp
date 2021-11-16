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
	vulnPkgModSlice(pkgs, modVulns, result)
	return result, nil
}

// pkgID is an id counter for nodes of Imports graph.
var pkgID int = 0

func nextPkgID() int {
	pkgID++
	return pkgID
}

// vulnPkgModSlice computes the slice of pkgs imports and requires graph
// leading to imports/requires of vulnerable packages/modules in modVulns
// and stores the computed slices to result.
func vulnPkgModSlice(pkgs []*packages.Package, modVulns moduleVulnerabilities, result *Result) {
	// analyzedPkgs contains information on packages analyzed thus far.
	// If a package is mapped to nil, this means it has been visited
	// but it does not lead to a vulnerable imports. Otherwise, a
	// visited package is mapped to Imports package node.
	analyzedPkgs := make(map[*packages.Package]*PkgNode)
	for _, pkg := range pkgs {
		// Top level packages that lead to vulnerable imports are
		// stored as result.Imports graph entry points.
		if e := vulnImportSlice(pkg, modVulns, result, analyzedPkgs); e != nil {
			result.Imports.Entries = append(result.Imports.Entries, e)
		}
	}

	// Populate module requires slice as an overlay
	// of package imports slice.
	vulnModuleSlice(result)
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
	id := nextPkgID()
	pkgNode := &PkgNode{
		ID:   id,
		Name: pkg.Name,
		Path: pkg.PkgPath,
		pkg:  pkg,
	}
	analyzed[pkg] = pkgNode

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
					ImportSink: id,
				}
				result.Vulns = append(result.Vulns, vuln)
			}
		}
	}
	return pkgNode
}

// vulnModuleSlice populates result.Requires as an overlay
// of result.Imports.
func vulnModuleSlice(result *Result) {
	// Map from module nodes, identified with their
	// path and version, to their unique ids.
	modNodeIDs := make(map[string]int)
	// We first collect inverse requires by (predecessor)
	// relation on module node ids.
	modPredRelation := make(map[int]map[int]bool)
	for _, pkgNode := range result.Imports.Packages {
		// Create or get module node for pkgNode.
		pkgModID := moduleNodeID(pkgNode, result, modNodeIDs)
		pkgNode.Module = pkgModID

		// Get the set of predecessors.
		predSet := make(map[int]bool)
		for _, predPkgID := range pkgNode.ImportedBy {
			predModID := moduleNodeID(result.Imports.Packages[predPkgID], result, modNodeIDs)
			predSet[predModID] = true
		}
		modPredRelation[pkgModID] = predSet
	}

	// Store the predecessor requires relation to result.
	for modID := range modPredRelation {
		if modID == 0 {
			continue
		}

		var predIDs []int
		for predID := range modPredRelation[modID] {
			predIDs = append(predIDs, predID)
		}
		modNode := result.Requires.Modules[modID]
		modNode.RequiredBy = predIDs
	}

	// And finally update Vulns with module information.
	for _, vuln := range result.Vulns {
		pkgNode := result.Imports.Packages[vuln.ImportSink]
		modNode := result.Requires.Modules[pkgNode.Module]

		vuln.RequireSink = pkgNode.Module
		vuln.ModPath = modNode.Path
	}
}

// modID is an id counter for nodes of Requires graph.
var modID int = 0

func nextModID() int {
	modID++
	return modID
}

// moduleNode creates a module node associated with pkgNode, if one does
// not exist already, and returns id of the module node. The actual module
// node is stored to result.
func moduleNodeID(pkgNode *PkgNode, result *Result, modNodeIDs map[string]int) int {
	mod := pkgNode.pkg.Module
	if mod == nil {
		return 0
	}

	mk := modKey(mod)
	if id, ok := modNodeIDs[mk]; ok {
		return id
	}

	id := nextModID()
	n := &ModNode{
		ID:      id,
		Path:    mod.Path,
		Version: mod.Version,
	}
	result.Requires.Modules[id] = n
	modNodeIDs[mk] = id

	// Create a replace module too when applicable.
	if mod.Replace != nil {
		rmk := modKey(mod.Replace)
		if rid, ok := modNodeIDs[rmk]; ok {
			n.Replace = rid
		} else {
			rid := nextModID()
			rn := &ModNode{
				Path:    mod.Replace.Path,
				Version: mod.Replace.Version,
			}
			result.Requires.Modules[rid] = rn
			modNodeIDs[rmk] = rid
			n.Replace = rid
		}
	}
	return id
}
