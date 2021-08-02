// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"container/list"
	"fmt"
	"go/token"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
)

// VulnerableSymbols returns a list of vulnerability findings for symbols transitively reachable
// through the callgraph built using VTA analysis from the entry points of pkgs, given the
// vulnerability and platform info captured in env.
//
// Returns all findings reachable from pkgs while analyzing each package only once, prefering findings
// of shorter import traces. For instance, given call chains
//   A() -> B() -> V
//   A() -> D() -> B() -> V
//   D() -> B() -> V
// where A and D are top level packages and V is a vulnerable symbol, VulnerableSymbols can return either
//   A() -> B() -> V
// or
//   D() -> B() -> V
// as traces of transitively using a vulnerable symbol V.
//
// Panics if packages in pkgs do not belong to the same program.
func VulnerableSymbols(pkgs []*ssa.Package, modVulns ModuleVulnerabilities) []Finding {
	prog := pkgsProgram(pkgs)
	if prog == nil {
		panic("packages in pkgs must belong to a single common program")
	}
	entries := entryPoints(pkgs)
	callGraph := callGraph(prog, entries)

	queue := list.New()
	for _, entry := range entries {
		queue.PushBack(&callChain{f: entry})
	}

	var findings []Finding
	seen := make(map[*ssa.Function]bool)
	for queue.Len() > 0 {
		front := queue.Front()
		v := front.Value.(*callChain)
		queue.Remove(front)

		if seen[v.f] {
			continue
		}
		seen[v.f] = true

		finds, calls := funcVulnsAndCalls(v, modVulns, callGraph)
		findings = append(findings, finds...)
		for _, call := range calls {
			queue.PushBack(call)
		}
	}

	return findings
}

// callGraph builds a call graph of prog based on VTA analysis.
func callGraph(prog *ssa.Program, entries []*ssa.Function) *callgraph.Graph {
	entrySlice := make(map[*ssa.Function]bool)
	for _, e := range entries {
		entrySlice[e] = true
	}
	initial := cha.CallGraph(prog)
	allFuncs := ssautil.AllFunctions(prog)

	fslice := forwardReachableFrom(entrySlice, initial)
	// Keep only actually linked functions.
	pruneSlice(fslice, allFuncs)
	vtaCg := vta.CallGraph(fslice, initial)

	// Repeat the process once more, this time using
	// the produced VTA call graph as the base graph.
	fslice = forwardReachableFrom(entrySlice, vtaCg)
	pruneSlice(fslice, allFuncs)

	return vta.CallGraph(fslice, vtaCg)
}

func entryPoints(topPackages []*ssa.Package) []*ssa.Function {
	var entries []*ssa.Function
	for _, pkg := range topPackages {
		if pkg.Pkg.Name() == "main" {
			// for "main" packages the only valid entry points are the "main"
			// function and any "init#" functions, even if there are other
			// exported functions or types. similarly to isEntry it should be
			// safe to ignore the validity of the main or init# signatures,
			// since the compiler will reject malformed definitions,
			// and the init function is synthetic
			entries = append(entries, memberFuncs(pkg.Members["main"], pkg.Prog)...)
			for name, member := range pkg.Members {
				if strings.HasPrefix(name, "init#") || name == "init" {
					entries = append(entries, memberFuncs(member, pkg.Prog)...)
				}
			}
			continue
		}
		for _, member := range pkg.Members {
			for _, f := range memberFuncs(member, pkg.Prog) {
				if isEntry(f) {
					entries = append(entries, f)
				}
			}
		}
	}
	return entries
}

func isEntry(f *ssa.Function) bool {
	// it should be safe to ignore checking that the signature of the "init" function
	// is valid, since it is synthetic
	if f.Name() == "init" && f.Synthetic == "package initializer" {
		return true
	}

	return f.Synthetic == "" && f.Object() != nil && f.Object().Exported()
}

// callChain helps doing BFS over package call graph while remembering the call stack.
type callChain struct {
	// nil for entry points of the chain.
	call   ssa.CallInstruction
	f      *ssa.Function
	parent *callChain
}

func (chain *callChain) trace() []TraceElem {
	if chain == nil {
		return nil
	}

	var pos *token.Position
	desc := fmt.Sprintf("%s.%s(...)", pkgPath(chain.f), chain.f.Name())
	if chain.call != nil {
		pos = instrPosition(chain.call)
		if unresolved(chain.call) {
			// In case of a statically unresolved call site, communicate to the client
			// that this was approximatelly resolved to chain.f.
			desc = fmt.Sprintf("%s(...) [approx. resolved to %s]", callName(chain.call), chain.f)
		}
	} else {
		// No call information means the function is an entry point.
		pos = funcPosition(chain.f)
	}

	return append(chain.parent.trace(), TraceElem{Description: desc, Position: pos})
}

// weight computes an approximate measure of how useful the call chain would
// be to the client as a trace. The smaller the value, the more useful the chain.
// Currently defined as the number of unresolved call sites in the chain.
func (chain *callChain) weight() int {
	if chain == nil {
		return 0
	}

	callWeight := 0
	if unresolved(chain.call) {
		callWeight = 1
	}
	return callWeight + chain.parent.weight()
}

// funcVulnsAndCalls returns a list of symbol findings for function at the top
// of chain and next calls to analyze.
func funcVulnsAndCalls(chain *callChain, modVulns ModuleVulnerabilities, callGraph *callgraph.Graph) ([]Finding, []*callChain) {
	var findings []Finding
	var calls []*callChain
	for _, b := range chain.f.Blocks {
		for _, instr := range b.Instrs {
			// First collect all findings for globals except callees in function call statements.
			findings = append(findings, globalFindings(globalUses(instr), chain, modVulns)...)

			// Callees are handled separately to produce call findings rather than global findings.
			site, ok := instr.(ssa.CallInstruction)
			if !ok {
				continue
			}

			callees := siteCallees(site, callGraph)
			for _, callee := range callees {
				c := &callChain{call: site, f: callee, parent: chain}
				calls = append(calls, c)

				if f := callFinding(c, modVulns); f != nil {
					findings = append(findings, *f)
				}
			}
		}
	}
	return findings, calls
}

// globalFindings returns findings for vulnerable globals among globalUses.
// Assumes each use in globalUses is a use of a global variable. Can generate
// duplicates when globalUses contains duplicates.
func globalFindings(globalUses []*ssa.Value, chain *callChain, modVulns ModuleVulnerabilities) []Finding {
	if underRelatedVuln(chain, modVulns) {
		return nil
	}

	var findings []Finding
	for _, o := range globalUses {
		g := (*o).(*ssa.Global)
		vulns := modVulns.VulnsForSymbol(g.Package().Pkg.Path(), g.Name())
		if len(vulns) > 0 {
			findings = append(findings,
				Finding{
					Symbol:   fmt.Sprintf("%s.%s", g.Package().Pkg.Path(), g.Name()),
					Trace:    chain.trace(),
					Position: valPosition(*o, chain.f),
					Type:     GlobalType,
					Vulns:    serialize(vulns),
					weight:   chain.weight()})
		}
	}
	return findings
}

// callFinding returns vulnerability finding for the call made at the top of the chain.
// If there is no vulnerability or no call information, then nil is returned.
// TODO(zpavlinovic): remove ssa info from higher-order calls.
func callFinding(chain *callChain, modVulns ModuleVulnerabilities) *Finding {
	if underRelatedVuln(chain, modVulns) {
		return nil
	}

	callee := chain.f
	call := chain.call
	if callee == nil || call == nil {
		return nil
	}

	vulns := modVulns.VulnsForSymbol(callee.Package().Pkg.Path(), dbFuncName(callee))
	if len(vulns) > 0 {
		c := chain
		if !unresolved(call) {
			// If the last call is a resolved callsite, remove the edge from the trace as that
			// information is provided in the symbol field.
			c = c.parent
		}
		return &Finding{
			Symbol:   fmt.Sprintf("%s.%s", callee.Package().Pkg.Path(), dbFuncName(callee)),
			Trace:    c.trace(),
			Position: instrPosition(call),
			Type:     FunctionType,
			Vulns:    serialize(vulns),
			weight:   c.weight()}
	}

	return nil
}

// Checks if a potential vulnerability in chain.f is analyzed only because
// a previous vulnerability in the same package as chain.f has been seen.
// For instance, for the chain P1:A -> P2:B -> P2:C where both B and C are
// vulnerable, the function returns true since B is already vulnerable and
// has hence been reported. Clients are likely not interested in vulnerabilties
// inside of a function that is already deemed vulnerable. This is an optimization
// step to stop flooding of findings when a package has a lot of known vulnerable
// symbols (e.g., all of them).
//
// Note that for P1:A -> P2:B -> P3:D -> P2:C the function returns false. This
// is because C is called from D that comes from a different package.
func underRelatedVuln(chain *callChain, modVulns ModuleVulnerabilities) bool {
	pkg := pkgPath(chain.f)

	c := chain
	for {
		c = c.parent
		// Analyze the immediate substack related to pkg.
		if c == nil || pkgPath(c.f) != pkg {
			break
		}
		// TODO: can we optimize using the information on findings already reported?
		if len(modVulns.VulnsForSymbol(c.f.Pkg.Pkg.Path(), dbFuncName(c.f))) > 0 {
			return true
		}
	}
	return false
}
