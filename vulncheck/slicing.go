// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

// forwardReachableFrom computes the set of functions forward reachable from `sources`.
// A function f is reachable from a function g if f is an anonymous function defined
// in g or a function called in g as given by the callgraph `cg`.
func forwardReachableFrom(sources map[*ssa.Function]bool, cg *callgraph.Graph) map[*ssa.Function]bool {
	m := make(map[*ssa.Function]bool)
	for s := range sources {
		forward(s, cg, m)
	}
	return m
}

func forward(f *ssa.Function, cg *callgraph.Graph, seen map[*ssa.Function]bool) {
	if seen[f] {
		return
	}
	seen[f] = true
	var buf [10]*ssa.Value // avoid alloc in common case
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			switch i := instr.(type) {
			case ssa.CallInstruction:
				for _, c := range siteCallees(i, cg) {
					forward(c, cg, seen)
				}
			default:
				for _, op := range i.Operands(buf[:0]) {
					if fn, ok := (*op).(*ssa.Function); ok {
						forward(fn, cg, seen)
					}
				}
			}
		}
	}
}

// pruneSet removes functions in `set` that are in `toPrune`.
func pruneSet(set, toPrune map[*ssa.Function]bool) {
	for f := range set {
		if !toPrune[f] {
			delete(set, f)
		}
	}
}
