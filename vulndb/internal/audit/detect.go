// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package audit finds vulnerabilities affecting Go packages.
package audit

import (
	"fmt"
	"go/token"
	"io"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/vulndb/osv"
)

// Preamble with types and common functionality used by vulnerability detection mechanisms in detect_*.go files.

// Finding represents a finding for the use of a vulnerable symbol or an imported vulnerable package.
// Provides info on symbol location, trace leading up to the symbol use, and associated vulnerabilities.
type Finding struct {
	Symbol   string
	Position *token.Position `json:",omitempty"`
	Type     SymbolType
	Vulns    []osv.Entry
	Trace    []TraceElem

	// Approximate measure for indicating how useful the finding might be to the audit client.
	// The smaller the weight, the more useful is the finding.
	weight int
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

// Write method for findings showing the trace and the associated vulnerabilities.
func (f Finding) Write(w io.Writer) {
	var pos string
	if f.Position != nil {
		pos = fmt.Sprintf(" (%s)", f.Position)
	}
	fmt.Fprintf(w, "Trace:\n%s%s\n", f.Symbol, pos)
	writeTrace(w, f.Trace)
	io.WriteString(w, "\n")
	writeVulns(w, f.Vulns)
	io.WriteString(w, "\n")
}

// writeTrace in reverse order, e.g., entry point is written last.
func writeTrace(w io.Writer, trace []TraceElem) {
	for i := len(trace) - 1; i >= 0; i-- {
		trace[i].Write(w)
		io.WriteString(w, "\n")
	}
}

func writeVulns(w io.Writer, vulns []osv.Entry) {
	fmt.Fprintf(w, "Vulnerabilities:\n")
	for _, v := range vulns {
		fmt.Fprintf(w, "%s (%s)\n", v.Package.Name, v.EcosystemSpecific.URL)
	}
}

func (e TraceElem) Write(w io.Writer) {
	var pos string
	if e.Position != nil {
		pos = fmt.Sprintf(" (%s)", e.Position)
	}
	fmt.Fprintf(w, "%s%s", e.Description, pos)
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
		var filteredVulns []*osv.Entry
		for _, v := range mod.vulns {
			if matchesPlatform(os, arch, v.EcosystemSpecific) {
				filteredVulns = append(filteredVulns, v)
			}
		}
		filteredMod = append(filteredMod, modVulns{
			mod:   mod.mod,
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
// specific prefixof importPath, or nil if there is no matching module with
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
