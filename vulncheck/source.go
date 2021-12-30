// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"context"
	"runtime"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/vuln/osv"
)

// Source detects vulnerabilities in pkgs and computes slices of
//  - imports graph related to an import of a package with some
//    known vulnerabilities
//  - requires graph related to a require of a module with a
//    package that has some known vulnerabilities
//  - call graph leading to the use of a known vulnerable function
//    or method
func Source(ctx context.Context, pkgs []*Package, cfg *Config) (*Result, error) {
	modVulns, err := fetchVulnerabilities(ctx, cfg.Client, extractModules(pkgs))
	if err != nil {
		return nil, err
	}
	modVulns = modVulns.Filter(lookupEnv("GOOS", runtime.GOOS), lookupEnv("GOARCH", runtime.GOARCH))

	result := &Result{
		Imports:  &ImportGraph{Packages: make(map[int]*PkgNode)},
		Requires: &RequireGraph{Modules: make(map[int]*ModNode)},
		Calls:    &CallGraph{Functions: make(map[int]*FuncNode)},
	}

	vulnPkgModSlice(pkgs, modVulns, result)

	if cfg.ImportsOnly {
		return result, nil
	}

	prog, ssaPkgs := buildSSA(pkgs)
	entries := entryPoints(ssaPkgs)
	cg := callGraph(prog, entries)
	vulnCallGraphSlice(entries, modVulns, cg, result)

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
func vulnPkgModSlice(pkgs []*Package, modVulns moduleVulnerabilities, result *Result) {
	// analyzedPkgs contains information on packages analyzed thus far.
	// If a package is mapped to nil, this means it has been visited
	// but it does not lead to a vulnerable imports. Otherwise, a
	// visited package is mapped to Imports package node.
	analyzedPkgs := make(map[*Package]*PkgNode)
	for _, pkg := range pkgs {
		// Top level packages that lead to vulnerable imports are
		// stored as result.Imports graph entry points.
		if e := vulnImportSlice(pkg, modVulns, result, analyzedPkgs); e != nil {
			result.Imports.Entries = append(result.Imports.Entries, e.ID)
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
func vulnImportSlice(pkg *Package, modVulns moduleVulnerabilities, result *Result, analyzed map[*Package]*PkgNode) *PkgNode {
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

			var symbols []string
			if len(affected.EcosystemSpecific.Symbols) != 0 {
				symbols = affected.EcosystemSpecific.Symbols
			} else {
				symbols = allSymbols(pkg.Pkg)
			}

			for _, symbol := range symbols {
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

	// Add entry module IDs.
	seenEntries := make(map[int]bool)
	for _, epID := range result.Imports.Entries {
		entryModID := moduleNodeID(result.Imports.Packages[epID], result, modNodeIDs)
		if seenEntries[entryModID] {
			continue
		}
		seenEntries[entryModID] = true
		result.Requires.Entries = append(result.Requires.Entries, entryModID)
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

func vulnCallGraphSlice(entries []*ssa.Function, modVulns moduleVulnerabilities, cg *callgraph.Graph, result *Result) {
	// analyzedFuncs contains information on functions analyzed thus far.
	// If a function is mapped to nil, this means it has been visited
	// but it does not lead to a vulnerable call. Otherwise, a visited
	// function is mapped to Calls function node.
	analyzedFuncs := make(map[*ssa.Function]*FuncNode)
	for _, entry := range entries {
		// Top level entries that lead to vulnerable calls
		// are stored as result.Calls graph entry points.
		if e := vulnCallSlice(entry, modVulns, cg, result, analyzedFuncs); e != nil {
			result.Calls.Entries = append(result.Calls.Entries, e.ID)
		}
	}
}

// funID is an id counter for nodes of Calls graph.
var funID int = 0

func nextFunID() int {
	funID++
	return funID
}

// vulnCallSlice checks if f has some vulnerabilities or transitively calls
// a function with known vulnerabilities. If so, populates result.Calls
// graph with this reachability information and returns the result.Call
// function node. Otherwise, returns nil.
func vulnCallSlice(f *ssa.Function, modVulns moduleVulnerabilities, cg *callgraph.Graph, result *Result, analyzed map[*ssa.Function]*FuncNode) *FuncNode {
	if fn, ok := analyzed[f]; ok {
		return fn
	}

	fn := cg.Nodes[f]
	if fn == nil {
		return nil
	}

	// Check if f has known vulnerabilities.
	vulns := modVulns.VulnsForSymbol(f.Package().Pkg.Path(), dbFuncName(f))

	var funNode *FuncNode
	// If there are vulnerabilities for f, create node for f and
	// save it immediately. This allows us to include F in the
	// slice when analyzing chain V -> F -> V where V is vulnerable.
	if len(vulns) > 0 {
		funNode = funcNode(f)
	}
	analyzed[f] = funNode

	// Recursively compute which callees lead to a call of a
	// vulnerable function. Remember the nodes of such callees.
	type siteNode struct {
		call ssa.CallInstruction
		fn   *FuncNode
	}
	var onSlice []siteNode
	for _, edge := range fn.Out {
		if calleeNode := vulnCallSlice(edge.Callee.Func, modVulns, cg, result, analyzed); calleeNode != nil {
			onSlice = append(onSlice, siteNode{call: edge.Site, fn: calleeNode})
		}
	}

	// If f is not vulnerable nor it transitively leads
	// to vulnerable calls, jump out.
	if len(onSlice) == 0 && len(vulns) == 0 {
		return nil
	}

	// If f is not vulnerable, then at this point it has
	// to be on the path leading to a vulnerable call.
	if funNode == nil {
		funNode = funcNode(f)
		analyzed[f] = funNode
	}
	result.Calls.Functions[funNode.ID] = funNode

	// Save node predecessor information.
	for _, calleeSliceInfo := range onSlice {
		call, node := calleeSliceInfo.call, calleeSliceInfo.fn
		cs := &CallSite{
			Parent:   funNode.ID,
			Name:     call.Common().Value.Name(),
			RecvType: callRecvType(call),
			Resolved: resolved(call),
			Pos:      instrPosition(call),
		}
		node.CallSites = append(node.CallSites, cs)
	}

	// Populate CallSink field for each detected vuln symbol.
	for _, osv := range vulns {
		for _, affected := range osv.Affected {
			if affected.Package.Name != funNode.PkgPath {
				continue
			}
			addCallSinkForVuln(funNode.ID, osv, dbFuncName(f), funNode.PkgPath, result)
		}
	}
	return funNode
}

func funcNode(f *ssa.Function) *FuncNode {
	id := nextFunID()
	return &FuncNode{
		ID:       id,
		Name:     f.Name(),
		PkgPath:  f.Package().Pkg.Path(),
		RecvType: funcRecvType(f),
		Pos:      funcPosition(f),
	}
}

// addCallSinkForVuln adds callID as call sink to vuln of result.Vulns
// identified with <osv, symbol, pkg>.
func addCallSinkForVuln(callID int, osv *osv.Entry, symbol, pkg string, result *Result) {
	for _, vuln := range result.Vulns {
		if vuln.OSV == osv && vuln.Symbol == symbol && vuln.PkgPath == pkg {
			vuln.CallSink = callID
			return
		}
	}
}
