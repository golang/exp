// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package audit finds vulnerabilities affecting Go packages.
package audit

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/vulndb/osv"
)

// Preamble with types and common functionality used by vulnerability detection mechanisms in detect_*.go files.

// SearchType represents a type of an audit search: call graph, imports, or binary.
type SearchType int

// enum values for SearchType.
const (
	CallGraphSearch SearchType = iota
	ImportsSearch
	BinarySearch
)

// Results contains the information on findings and identified vulnerabilities by audit search.
type Results struct {
	SearchMode SearchType

	// TODO: identify vulnerability with <ID, package, symbol>?
	// Vulnerabilities in dependent modules.
	Vulnerabilities []osv.Entry

	VulnFindings map[string][]Finding // vuln.ID -> findings
}

// String method for results.
func (r Results) String() string {
	sort.Slice(r.Vulnerabilities, func(i, j int) bool { return r.Vulnerabilities[i].ID < r.Vulnerabilities[j].ID })

	rStr := ""
	for _, v := range r.Vulnerabilities {
		findings := r.VulnFindings[v.ID]
		if len(findings) == 0 {
			// TODO: add messages for such cases too?
			continue
		}

		var alias string
		if len(v.Aliases) == 0 {
			alias = v.EcosystemSpecific.URL
		} else {
			alias = strings.Join(v.Aliases, ", ")
		}
		rStr += fmt.Sprintf("Findings for vulnerability: %s (of package %s):\n\n", alias, v.Package.Name)

		for _, finding := range findings {
			rStr += finding.String() + "\n"
		}
	}
	return rStr
}

// addFindings adds a findings `f` for vulnerability `v`.
func (r Results) addFinding(v osv.Entry, f Finding) {
	r.VulnFindings[v.ID] = append(r.VulnFindings[v.ID], f)
}

// sort orders findings for each vulnerability based on its
// perceived usefulness to the user.
func (r Results) sort() {
	for _, fs := range r.VulnFindings {
		sort.SliceStable(fs, func(i int, j int) bool { return findingCompare(&fs[i], &fs[j]) })
	}
}

// Finding represents a finding for the use of a vulnerable symbol or an imported vulnerable package.
// Provides info on symbol location and the trace leading up to the symbol use.
type Finding struct {
	Symbol   string
	Position *token.Position `json:",omitempty"`
	Type     SymbolType
	Trace    []TraceElem

	// Approximate measure for indicating how understandable the finding is to the client.
	// The smaller the weight, the more understandable is the finding.
	weight int

	// Approximate measure for indicating confidence in finding being a true positive. The
	// smaller the value, the bigger the confidence.
	confidence int
}

// String method for findings.
func (f Finding) String() string {
	traceStr := traceString(f.Trace)

	var pos string
	if f.Position != nil {
		pos = fmt.Sprintf(" (%s)", f.Position)
	}

	return fmt.Sprintf("Trace:\n%s%s\n%s\n", f.Symbol, pos, traceStr)
}

func traceString(trace []TraceElem) string {
	// traces are typically short, so string builders are not necessary
	traceStr := ""
	for i := len(trace) - 1; i >= 0; i-- {
		traceStr += trace[i].String() + "\n"
	}
	return traceStr
}

// SymbolType represents a type of a symbol use: function, global, or an import statement.
type SymbolType int

// enum values for SymbolType.
const (
	FunctionType SymbolType = iota
	ImportType
	GlobalType
)

// TraceElem represents an entry in the finding trace. Represents a function call or an import statement.
type TraceElem struct {
	Description string
	Position    *token.Position `json:",omitempty"`
}

// String method for trace elements.
func (e TraceElem) String() string {
	if e.Position == nil {
		return fmt.Sprintf("%s", e.Description)
	}
	return fmt.Sprintf("%s (%s)", e.Description, e.Position)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (s SymbolType) MarshalText() ([]byte, error) {
	var name string
	switch s {
	default:
		name = "unrecognized"
	case FunctionType:
		name = "function"
	case ImportType:
		name = "import"
	case GlobalType:
		name = "global"
	}
	return []byte(name), nil
}

type modVulns struct {
	mod   *packages.Module
	vulns []*osv.Entry
}

type ModuleVulnerabilities []modVulns

func matchesPlatform(os, arch string, e osv.GoSpecific) bool {
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

func (mv ModuleVulnerabilities) Filter(os, arch string) ModuleVulnerabilities {
	var filteredMod ModuleVulnerabilities
	for _, mod := range mv {
		module := mod.mod
		modVersion := module.Version
		if module.Replace != nil {
			modVersion = module.Replace.Version
		}
		// TODO: if modVersion == "", try vcs to get the version?
		var filteredVulns []*osv.Entry
		for _, v := range mod.vulns {
			// A module version is affected if
			//  - it is included in one of the affected version ranges
			//  - and module version is not ""
			//  The latter means the module version is not available, so
			//  we don't want to spam users with potential false alarms.
			//  TODO: issue warning for "" cases above?
			affectsVersion := modVersion != "" && v.Affects.AffectsSemver(modVersion)
			if affectsVersion && matchesPlatform(os, arch, v.EcosystemSpecific) {
				filteredVulns = append(filteredVulns, v)
			}
		}
		filteredMod = append(filteredMod, modVulns{
			mod:   module,
			vulns: filteredVulns,
		})
	}
	return filteredMod
}

func (mv ModuleVulnerabilities) Num() int {
	var num int
	for _, m := range mv {
		num += len(m.vulns)
	}
	return num
}

// VulnsForPackage returns the vulnerabilities for the module which is the most
// specific prefix of importPath, or nil if there is no matching module with
// vulnerabilities.
func (mv ModuleVulnerabilities) VulnsForPackage(importPath string) []*osv.Entry {
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
		if v.Package.Name == importPath {
			packageVulns = append(packageVulns, v)
		}
	}
	return packageVulns
}

// VulnsForSymbol returns vulnerabilites for `symbol` in `mv.VulnsForPackage(importPath)`.
func (mv ModuleVulnerabilities) VulnsForSymbol(importPath, symbol string) []*osv.Entry {
	vulns := mv.VulnsForPackage(importPath)
	if vulns == nil {
		return nil
	}

	symbolVulns := []*osv.Entry{}
	for _, v := range vulns {
		if len(v.EcosystemSpecific.Symbols) == 0 {
			symbolVulns = append(symbolVulns, v)
			continue
		}
		for _, s := range v.EcosystemSpecific.Symbols {
			if s == symbol {
				symbolVulns = append(symbolVulns, v)
				break
			}
		}
	}
	return symbolVulns
}

// Vulns returns vulnerabilities for all modules in `mv`.
func (mv ModuleVulnerabilities) Vulns() []*osv.Entry {
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
