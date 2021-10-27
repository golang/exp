// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"path"
	"reflect"
	"testing"

	"golang.org/x/tools/go/packages/packagestest"
)

// TestImportsOnly checks for module and imports graph correctness
// for the Config.ImportsOnly=true mode. The inlined test code has
// the following package (left) and module (right) imports graphs:
//
//       entry/x        entry/y                     entry
//              \     /        \                   /     \
//            amod/avuln      zmod/z           amod       zmod
//                |                              |
//              wmod/w                         wmod
//                |                              |
//            bmod/bvuln                       bmod
//
// Packages ending in "vuln" have some known vulnerabilities.
func TestImportsOnly(t *testing.T) {
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "golang.org/entry",
			Files: map[string]interface{}{
				"x/x.go": `
			package x

			import "golang.org/amod/avuln"

			func X() {
				avuln.VulnData{}.Vuln1()
			}
			`,
				"y/y.go": `
			package y

			import (
				"golang.org/amod/avuln"
				"golang.org/zmod/z"
			)

			func Y() {
				avuln.VulnData{}.Vuln2()
				z.Z()
			}
		`}},
		{
			Name: "golang.org/zmod@v0.0.0",
			Files: map[string]interface{}{"z/z.go": `
			package z

			func Z() {}
			`},
		},
		{
			Name: "golang.org/amod@v1.1.3",
			Files: map[string]interface{}{"avuln/avuln.go": `
			package avuln

			import "golang.org/wmod/w"

			type VulnData struct {}
			func (v VulnData) Vuln1() { w.W() }
			func (v VulnData) Vuln2() {}
			`},
		},
		{
			Name: "golang.org/bmod@v0.5.0",
			Files: map[string]interface{}{"bvuln/bvuln.go": `
			package bvuln

			func Vuln() {}
			`},
		},
		{
			Name: "golang.org/wmod@v0.0.0",
			Files: map[string]interface{}{"w/w.go": `
			package w

			import "golang.org/bmod/bvuln"

			func W() { bvuln.Vuln() }
			`},
		},
	})
	defer e.Cleanup()

	// Make sure local vulns can be loaded.
	fetchingInTesting = true
	// Load x and y as entry packages.
	pkgs, err := loadPackages(e, path.Join(e.Temp(), "entry/x"), path.Join(e.Temp(), "entry/y"))
	if err != nil {
		t.Fatal(err)
	}

	if len(pkgs) != 2 {
		t.Fatal("failed to load x and y test packages")
	}

	cfg := &Config{
		Client:      testClient,
		ImportsOnly: true,
	}
	result, err := Source(pkgs, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(zpavlinovic): add test below for module graph too.

	// Check that we find the right number of vulnerabilities.
	// There should be three entries as there are three vulnerable
	// symbols in the two import-reachable OSVs.
	if len(result.Vulns) != 3 {
		t.Errorf("want 3 Vulns, got %d", len(result.Vulns))
	}

	// Check that vulnerabilities are connected to the imports graph.
	for _, v := range result.Vulns {
		if v.ImportSink == 0 {
			t.Errorf("want ImportSink !=0 for vuln %v:%v; got 0", v.Symbol, v.PkgPath)
		}
	}

	// The slice should include import chains:
	//   x -> avuln -> w -> bvuln
	//         |
	//   y ---->
	// That is, z package shoud not appear in the slice.
	wantImports := map[string][]string{
		"golang.org/entry/x":    {"golang.org/amod/avuln"},
		"golang.org/entry/y":    {"golang.org/amod/avuln"},
		"golang.org/amod/avuln": {"golang.org/wmod/w"},
		"golang.org/wmod/w":     {"golang.org/bmod/bvuln"},
	}

	if igStrMap := impGraphToStrMap(result.Imports); !reflect.DeepEqual(wantImports, igStrMap) {
		t.Errorf("want %v imports graph; got %v", wantImports, igStrMap)
	}
}
