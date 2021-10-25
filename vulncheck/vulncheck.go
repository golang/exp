// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vulncheck detects uses of known vulnerabilities
// in Go binaries and source code.
package vulncheck

import (
	"go/token"

	"golang.org/x/vulndb/client"
	"golang.org/x/vulndb/osv"
)

// Config is used for configuring vulncheck algorithms.
type Config struct {
	// ImportsOnly flag, if true, signals vulncheck to analyze import chains only.
	// Otherwise, call chains are analyzed too.
	ImportsOnly bool
	// Client is used for querying data from a vulnerability database.
	Client client.Client
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

// CallGraph whose sinks are vulnerable functions and sources are entry points of user
// packages. CallGraph is backwards directed, i.e., from a function node to the place
// where the function is called.
type CallGraph struct {
	// Funcs contains all call graph nodes as a map: func node id -> func node.
	Funcs map[int]*FuncNode
	// Entries are a subset of Funcs representing vulncheck entry points.
	Entries []*FuncNode
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

// RequireGraph models part of module requires graph where sinks are modules with
// some known vulnerabilities and sources are modules of user entry packages.
// RequireGraph is backwards directed, i.e., from a module to the set of modules
// it is required by.
type RequireGraph struct {
	// Modules contains all module nodes as a map: module node id -> module node.
	Modules map[int]*ModNode
	// Entries are a subset of Modules representing modules of vulncheck entry points.
	Entries []*ModNode
}

type ModNode struct {
	Path    string
	Version string
	Replace *ModNode
	// RequiredBy contains IDs of the modules requiring this module.
	RequiredBy []int
}

// ImportGraph models part of package import graph where sinks are packages with
// some known vulnerabilities and sources are user specified packages. The graph
// is backwards directed, i.e., from a package to the set of packages importing it.
type ImportGraph struct {
	// Packages contains all package nodes as a map: package node id -> package node.
	Packages map[int]*PkgNode
	// Entries are a subset of Packages representing packages of vulncheck entry points.
	Entries []*PkgNode
}

type PkgNode struct {
	// Name is the package identifier as it appears in the source code.
	Name string
	Path string
	// Module holds ID of the corresponding module (node) in Requires graph.
	Module int
	// ImportedBy contains IDs of packages directly importing this package.
	ImportedBy []int
}
