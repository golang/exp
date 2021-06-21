// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages/packagestest"
	"golang.org/x/tools/go/ssa"
)

// funcsToString returns a set of function names for `funcs`.
func funcsToString(funcs map[*ssa.Function]bool) map[string]bool {
	fs := make(map[string]bool)
	for f := range funcs {
		fs[dbFuncName(f)] = true
	}
	return fs
}

func TestSlicing(t *testing.T) {
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name:  "some/module",
			Files: map[string]interface{}{"slice/slice.go": readFile(t, "testdata/slice.go")},
		},
	})
	prog, pkgs, _, err := loadAndBuildPackages(e, "/module/slice/slice.go")
	if err != nil {
		t.Fatal(err)
	}

	pkg := pkgs[0]
	sources := map[*ssa.Function]bool{pkg.Func("Apply"): true, pkg.Func("Do"): true}
	fs := funcsToString(forwardReachableFrom(sources, cha.CallGraph(prog)))
	want := map[string]bool{
		"Apply":   true,
		"Apply$1": true,
		"X":       true,
		"Y":       true,
		"Do":      true,
		"Do$1":    true,
		"Do$1$1":  true,
		"debug":   true,
		"A.Foo":   true,
		"B.Foo":   true,
	}
	if !reflect.DeepEqual(want, fs) {
		t.Errorf("want %v; got %v", want, fs)
	}
}
