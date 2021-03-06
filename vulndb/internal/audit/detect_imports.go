// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"container/list"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// VulnerableImports returns a list of vulnerability findings for packages imported by `pkgs`
// given the vulnerability and platform info captured in `env`.
//
// Returns all findings reachable from `pkgs` while analyzing each package only once, prefering
// findings of shorter import traces. For instance, given import chains
//   A -> B -> V
//   A -> D -> B -> V
//   D -> B -> V
// where A and D are top level packages and V is a vulnerable package, VulnerableImports can return either
//   A -> B -> V
// or
//   D -> B -> V
// as traces of importing a vulnerable package V.
func VulnerableImports(pkgs []*ssa.Package, env Env) []Finding {
	pkgVulns := createPkgVulns(env.Vulns)

	var findings []Finding
	seen := make(map[string]bool)
	queue := list.New()
	for _, pkg := range pkgs {
		queue.PushBack(&importChain{pkg: pkg.Pkg})
	}

	for queue.Len() > 0 {
		front := queue.Front()
		v := front.Value.(*importChain)
		queue.Remove(front)

		pkg := v.pkg
		if pkg == nil {
			continue
		}

		if seen[pkg.Path()] {
			continue
		}
		seen[pkg.Path()] = true

		for _, imp := range pkg.Imports() {
			vulns := queryPkgVulns(imp.Path(), env, pkgVulns)
			if len(vulns) > 0 {
				findings = append(findings,
					Finding{
						Symbol: imp.Path(),
						Type:   ImportType,
						Trace:  v.trace(),
						Vulns:  serialize(vulns),
						weight: len(v.trace())})
			}
			queue.PushBack(&importChain{pkg: imp, parent: v})
		}
	}

	return findings
}

// importChain helps doing BFS over package imports while remembering import chains.
type importChain struct {
	pkg    *types.Package
	parent *importChain
}

func (chain *importChain) trace() []TraceElem {
	if chain == nil {
		return nil
	}
	return append(chain.parent.trace(), TraceElem{Description: chain.pkg.Path()})
}
