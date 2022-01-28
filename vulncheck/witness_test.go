// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"reflect"
	"strings"
	"testing"
)

// chainsToString converts map Vuln:chains to Vuln.PkgPath:["pkg1->...->pkgN", ...]
// string representation.
func chainsToString(chains map[*Vuln][]ImportChain) map[string][]string {
	m := make(map[string][]string)
	for v, chs := range chains {
		var chsStr []string
		for _, ch := range chs {
			var chStr []string
			for _, imp := range ch {
				chStr = append(chStr, imp.Path)
			}
			chsStr = append(chsStr, strings.Join(chStr, "->"))
		}
		m[v.PkgPath] = chsStr
	}
	return m
}

// stacksToString converts map *Vuln:stacks to Vuln.Symbol:["f1->...->fN", ...]
// string representation.
func stacksToString(stacks map[*Vuln][]CallStack) map[string][]string {
	m := make(map[string][]string)
	for v, sts := range stacks {
		var stsStr []string
		for _, st := range sts {
			var stStr []string
			for _, call := range st {
				stStr = append(stStr, call.Function.Name)
			}
			stsStr = append(stsStr, strings.Join(stStr, "->"))
		}
		m[v.Symbol] = stsStr
	}
	return m
}

func TestImportChains(t *testing.T) {
	// Package import structure for the test program
	//    entry1  entry2
	//      |       |
	//    interm1   |
	//      |    \  |
	//      |   interm2
	//      |   /     |
	//     vuln1    vuln2
	e1 := &PkgNode{ID: 1, Path: "entry1"}
	e2 := &PkgNode{ID: 2, Path: "entry2"}
	i1 := &PkgNode{ID: 3, Path: "interm1", ImportedBy: []int{1}}
	i2 := &PkgNode{ID: 4, Path: "interm2", ImportedBy: []int{2, 3}}
	v1 := &PkgNode{ID: 5, Path: "vuln1", ImportedBy: []int{3, 4}}
	v2 := &PkgNode{ID: 6, Path: "vuln2", ImportedBy: []int{4}}

	ig := &ImportGraph{
		Packages: map[int]*PkgNode{1: e1, 2: e2, 3: i1, 4: i2, 5: v1, 6: v2},
		Entries:  []int{1, 2},
	}
	vuln1 := &Vuln{ImportSink: 5, PkgPath: "vuln1"}
	vuln2 := &Vuln{ImportSink: 6, PkgPath: "vuln2"}
	res := &Result{Imports: ig, Vulns: []*Vuln{vuln1, vuln2}}

	// The chain entry1->interm1->interm2->vuln1 is not reported
	// as there exist a shorter trace going from entry1 to vuln1
	// via interm1.
	want := map[string][]string{
		"vuln1": {"entry1->interm1->vuln1", "entry2->interm2->vuln1"},
		"vuln2": {"entry2->interm2->vuln2", "entry1->interm1->interm2->vuln2"},
	}

	chains := ImportChains(res)
	if got := chainsToString(chains); !reflect.DeepEqual(want, got) {
		t.Errorf("want %v; got %v", want, got)
	}
}

func TestCallStacks(t *testing.T) {
	// Call graph structure for the test program
	//    entry1      entry2
	//      |           |
	//    interm1(std)  |
	//      |    \     /
	//      |   interm2(interface)
	//      |   /     |
	//     vuln1    vuln2
	e1 := &FuncNode{ID: 1, Name: "entry1"}
	e2 := &FuncNode{ID: 2, Name: "entry2"}
	i1 := &FuncNode{ID: 3, Name: "interm1", PkgPath: "net/http", CallSites: []*CallSite{&CallSite{Parent: 1, Resolved: true}}}
	i2 := &FuncNode{ID: 4, Name: "interm2", CallSites: []*CallSite{&CallSite{Parent: 2, Resolved: true}, &CallSite{Parent: 3, Resolved: true}}}
	v1 := &FuncNode{ID: 5, Name: "vuln1", CallSites: []*CallSite{&CallSite{Parent: 3, Resolved: true}, &CallSite{Parent: 4, Resolved: false}}}
	v2 := &FuncNode{ID: 6, Name: "vuln2", CallSites: []*CallSite{&CallSite{Parent: 4, Resolved: false}}}

	cg := &CallGraph{
		Functions: map[int]*FuncNode{1: e1, 2: e2, 3: i1, 4: i2, 5: v1, 6: v2},
		Entries:   []int{1, 2},
	}
	vuln1 := &Vuln{CallSink: 5, Symbol: "vuln1"}
	vuln2 := &Vuln{CallSink: 6, Symbol: "vuln2"}
	res := &Result{Calls: cg, Vulns: []*Vuln{vuln1, vuln2}}

	want := map[string][]string{
		"vuln1": {"entry2->interm2->vuln1", "entry1->interm1->vuln1"},
		"vuln2": {"entry2->interm2->vuln2", "entry1->interm1->interm2->vuln2"},
	}

	stacks := CallStacks(res)
	if got := stacksToString(stacks); !reflect.DeepEqual(want, got) {
		t.Errorf("want %v; got %v", want, got)
	}
}
