// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vulncheck detects uses of known vulnerabilities
// in Go binaries and source code.
package vulncheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/vuln/client"
	"golang.org/x/vuln/osv"
)

// Config is used for configuring vulncheck algorithms.
type Config struct {
	// ImportsOnly flag, if true, signals vulncheck to analyze import chains only.
	// Otherwise, call chains are analyzed too.
	ImportsOnly bool
	// Client is used for querying data from a vulnerability database.
	Client client.Client
}

// Package models Go package for vulncheck analysis. A version
// of packages.Package trimmed down to reduce memory consumption.
type Package struct {
	Name      string
	PkgPath   string
	Imports   []*Package
	Pkg       *types.Package
	Fset      *token.FileSet
	Syntax    []*ast.File
	TypesInfo *types.Info
	Module    *Module
}

// Module models Go module for vulncheck analysis.
type Module struct {
	Path    string
	Version string
	Dir     string
	Replace *Module
}

// Convert converts a slice of packages.Package to
// a slice of corresponding vulncheck.Package.
func Convert(pkgs []*packages.Package) []*Package {
	ms := make(map[*packages.Module]*Module)
	var mod func(*packages.Module) *Module
	mod = func(m *packages.Module) *Module {
		if m == nil {
			return nil
		}
		if vm, ok := ms[m]; ok {
			return vm
		}
		vm := &Module{
			Path:    m.Path,
			Version: m.Version,
			Dir:     m.Dir,
			Replace: mod(m.Replace),
		}
		ms[m] = vm
		return vm
	}

	ps := make(map[*packages.Package]*Package)
	var pkg func(*packages.Package) *Package
	pkg = func(p *packages.Package) *Package {
		if vp, ok := ps[p]; ok {
			return vp
		}

		vp := &Package{
			Name:      p.Name,
			PkgPath:   p.PkgPath,
			Pkg:       p.Types,
			Fset:      p.Fset,
			Syntax:    p.Syntax,
			TypesInfo: p.TypesInfo,
			Module:    mod(p.Module),
		}
		ps[p] = vp

		for _, i := range p.Imports {
			vp.Imports = append(vp.Imports, pkg(i))
		}
		return vp
	}

	var vpkgs []*Package
	for _, p := range pkgs {
		vpkgs = append(vpkgs, pkg(p))
	}
	return vpkgs
}

// Result contains information on which vulnerabilities are potentially affecting
// user code and how are they affecting it via call graph, package imports graph,
// and module requires graph.
type Result struct {
	// Calls is a call graph whose roots are program entry functions/methods and
	// sinks are vulnerable functions/methods. Empty when Config.ImportsOnly=true
	// or when no vulnerable symbols are reachable via program call graph.
	Calls *CallGraph
	// Imports is a package dependency graph whose roots are entry user packages
	// and sinks are the packages with some vulnerable symbols. Empty when no
	// packages with some vulnerabilities are imported in the program.
	Imports *ImportGraph
	// Requires is a module dependency graph whose roots are entry user modules
	// and sinks are modules with some vulnerable packages. Empty when no modules
	// with some vulnerabilities are required by the program.
	Requires *RequireGraph

	// Vulns contains information on detected vulnerabilities and their place in
	// the above graphs. Only vulnerabilities whose symbols are reachable in Calls,
	// or whose packages are imported in Imports, or whose modules are required in
	// Requires, have an entry in Vulns.
	Vulns []*Vuln
}

// Vuln provides information on how a vulnerability is affecting user code by
// connecting it to the Result.{Calls,Imports,Requires} graphs. Vulnerabilities
// detected in Go binaries do not have a place in the Result graphs.
type Vuln struct {
	// The next four fields identify a vulnerability. Note that *osv.Entry
	// describes potentially multiple symbols from multiple packages.

	// OSV contains information on detected vulnerability in the shared
	// vulnerability format.
	OSV *osv.Entry
	// Symbol is the name of the detected vulnerable function or method.
	Symbol string
	// PkgPath is the package path of the detected Symbol.
	PkgPath string
	// ModPath is the module path corresponding to PkgPath.
	ModPath string

	// CallSink is the ID of the sink node in Calls graph corresponding to
	// the use of Symbol. ID is not available (denoted with 0) in binary mode,
	// or if Symbol is not reachable, or if Config.ImportsOnly=true.
	CallSink int
	// ImportSink is the ID of the sink node in the Imports graph corresponding
	// to the import of PkgPath. ID is not available (denoted with 0) in binary
	// mode or if PkgPath is not imported.
	ImportSink int
	// RequireSink is the ID of the sink node in Requires graph corresponding
	// to the require statement of ModPath. ID is not available (denoted with 0)
	// in binary mode.
	RequireSink int
}

// CallGraph is a slice of a full program call graph whose sinks are conceptually
// vulnerable functions and sources are entry points of user packages. In order to
// support succinct traversal of the slice related to a particular vulnerability,
// CallGraph is technically backwards directed, i.e., from a vulnerable function
// towards the program entry functions (see FuncNode).
type CallGraph struct {
	// Functions contains all call graph nodes as a map: func node id -> func node.
	Functions map[int]*FuncNode
	// Entries are IDs of a subset of Functions representing vulncheck entry points.
	Entries []int
}

type FuncNode struct {
	ID   int
	Name string
	// RecvType is the receiver object type of this function, if any.
	RecvType string
	PkgPath  string
	Pos      *token.Position
	// CallSites is a set of call sites where this function is called.
	CallSites []*CallSite
}

func (fn *FuncNode) String() string {
	if fn.RecvType == "" {
		return fmt.Sprintf("%s.%s", fn.PkgPath, fn.Name)
	}
	return fmt.Sprintf("%s.%s", fn.RecvType, fn.Name)
}

type CallSite struct {
	// Parent is ID of the enclosing function where the call is made.
	Parent int
	// Name stands for the name of the function (variable) being called.
	Name string
	// RecvType is the full path of the receiver object type, if any.
	RecvType string
	Pos      *token.Position
	// Resolved indicates if the called function can be statically resolved.
	Resolved bool
}

// RequireGraph is a slice of a full program module requires graph whose sinks
// are conceptually modules with some known vulnerabilities and sources are modules
// of user entry packages. In order to support succinct traversal of the slice
// related to a particular vulnerability, RequireGraph is technically backwards
// directed, i.e., from a vulnerable module towards the program entry modules (see ModNode).
type RequireGraph struct {
	// Modules contains all module nodes as a map: module node id -> module node.
	Modules map[int]*ModNode
	// Entries are IDs of a subset of Modules representing modules of vulncheck entry points.
	Entries []int
}

type ModNode struct {
	ID      int
	Path    string
	Version string
	// Replace is the ID of the replacement module node, if any.
	Replace int
	// RequiredBy contains IDs of the modules requiring this module.
	RequiredBy []int
}

// ImportGraph is a slice of a full program package import graph whose sinks are
// conceptually packages with some known vulnerabilities and sources are user
// specified packages. In order to support succinct traversal of the slice related
// to a particular vulnerability, ImportGraph is technically backwards directed,
// i.e., from a vulnerable package towards the program entry packages (see PkgNode).
type ImportGraph struct {
	// Packages contains all package nodes as a map: package node id -> package node.
	Packages map[int]*PkgNode
	// Entries are IDs of a subset of Packages representing packages of vulncheck entry points.
	Entries []int
}

type PkgNode struct {
	ID int
	// Name is the package identifier as it appears in the source code.
	Name string
	Path string
	// Module holds ID of the corresponding module (node) in Requires graph.
	Module int
	// ImportedBy contains IDs of packages directly importing this package.
	ImportedBy []int

	// pkg is used for connecting package node to module and call graph nodes.
	pkg *Package
}

// moduleVulnerabilities is an internal structure for
// holding and querying vulnerabilities provided by a
// vulnerability database client.
type moduleVulnerabilities []modVulns

// modVulns groups vulnerabilities per module.
type modVulns struct {
	mod   *Module
	vulns []*osv.Entry
}

func (mv moduleVulnerabilities) Filter(os, arch string) moduleVulnerabilities {
	var filteredMod moduleVulnerabilities
	for _, mod := range mv {
		module := mod.mod
		modVersion := module.Version
		if module.Replace != nil {
			modVersion = module.Replace.Version
		}
		// TODO(https://golang.org/issues/49264): if modVersion == "", try vcs?
		var filteredVulns []*osv.Entry
		for _, v := range mod.vulns {
			var filteredAffected []osv.Affected
			for _, a := range v.Affected {
				// A module version is affected if
				//  - it is included in one of the affected version ranges
				//  - and module version is not ""
				//  The latter means the module version is not available, so
				//  we don't want to spam users with potential false alarms.
				//  TODO: issue warning for "" cases above?
				affected := modVersion != "" && a.Ranges.AffectsSemver(modVersion) && matchesPlatform(os, arch, a.EcosystemSpecific)
				if affected {
					filteredAffected = append(filteredAffected, a)
				}
			}
			if len(filteredAffected) == 0 {
				continue
			}
			// save the non-empty vulnerability with only
			// affected symbols.
			newV := *v
			newV.Affected = filteredAffected
			filteredVulns = append(filteredVulns, &newV)
		}
		filteredMod = append(filteredMod, modVulns{
			mod:   module,
			vulns: filteredVulns,
		})
	}
	return filteredMod
}

func matchesPlatform(os, arch string, e osv.EcosystemSpecific) bool {
	matchesOS := len(e.GOOS) == 0
	matchesArch := len(e.GOARCH) == 0
	for _, o := range e.GOOS {
		if os == o {
			matchesOS = true
			break
		}
	}
	for _, a := range e.GOARCH {
		if arch == a {
			matchesArch = true
			break
		}
	}
	return matchesOS && matchesArch
}
func (mv moduleVulnerabilities) Num() int {
	var num int
	for _, m := range mv {
		num += len(m.vulns)
	}
	return num
}

// VulnsForPackage returns the vulnerabilities for the module which is the most
// specific prefix of importPath, or nil if there is no matching module with
// vulnerabilities.
func (mv moduleVulnerabilities) VulnsForPackage(importPath string) []*osv.Entry {
	var mostSpecificMod *modVulns
	for _, mod := range mv {
		md := mod
		if strings.HasPrefix(importPath, md.mod.Path) {
			if mostSpecificMod == nil || len(mostSpecificMod.mod.Path) < len(md.mod.Path) {
				mostSpecificMod = &md
			}
		}
	}

	if mostSpecificMod == nil {
		return nil
	}

	if mostSpecificMod.mod.Replace != nil {
		importPath = fmt.Sprintf("%s%s", mostSpecificMod.mod.Replace.Path, strings.TrimPrefix(importPath, mostSpecificMod.mod.Path))
	}
	vulns := mostSpecificMod.vulns
	packageVulns := []*osv.Entry{}
	for _, v := range vulns {
		for _, a := range v.Affected {
			if a.Package.Name == importPath {
				packageVulns = append(packageVulns, v)
				break
			}
		}
	}
	return packageVulns
}

// VulnsForSymbol returns vulnerabilities for `symbol` in `mv.VulnsForPackage(importPath)`.
func (mv moduleVulnerabilities) VulnsForSymbol(importPath, symbol string) []*osv.Entry {
	vulns := mv.VulnsForPackage(importPath)
	if vulns == nil {
		return nil
	}

	symbolVulns := []*osv.Entry{}
	for _, v := range vulns {
	vulnLoop:
		for _, a := range v.Affected {
			if a.Package.Name != importPath {
				continue
			}
			if len(a.EcosystemSpecific.Symbols) == 0 {
				symbolVulns = append(symbolVulns, v)
				continue vulnLoop
			}
			for _, s := range a.EcosystemSpecific.Symbols {
				if s == symbol {
					symbolVulns = append(symbolVulns, v)
					continue vulnLoop
				}
			}
		}
	}
	return symbolVulns
}

// Vulns returns vulnerabilities for all modules in `mv`.
func (mv moduleVulnerabilities) Vulns() []*osv.Entry {
	var vulns []*osv.Entry
	seen := make(map[string]bool)
	for _, mv := range mv {
		for _, v := range mv.vulns {
			if !seen[v.ID] {
				vulns = append(vulns, v)
				seen[v.ID] = true
			}
		}
	}
	return vulns
}
