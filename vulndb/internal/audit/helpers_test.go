// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"go/token"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/packages/packagestest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/vulndb/osv"
)

// Loads test program and environment with the following import structure
//                 T
//              /  |  \
//             A   |   B
//             \   |   /
//              \  |  A
//               \ | /
//               vuln
// where `vuln` is a package containing some vulnerabilities. The definition
// of T can be found in testdata/top_package.go, A is in testdata/a_dep.go,
// B is in testdata/b_dep.go, and vuln is in testdata/vuln.go.
//
// The program has the following vulnerabilities that should be reported
//   T:T1() -> vuln.VG
//   T:T1() -> A:A1() -> vuln.VulnData.Vuln()
//   T:T2() -> vuln.Vuln() [approx.resolved] -> vuln.VG
//   T:T1() -> vuln.VulnData.Vuln() [approx. resolved]
//
// The following vulnerability should not be reported as it is redundant:
//   T:T1() -> A:A1() -> B:B1() -> vuln.VulnData.Vuln()
func testContext(t *testing.T) ([]*ssa.Package, ModuleVulnerabilities) {
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name:  "golang.org/vulntest",
			Files: map[string]interface{}{"T/T.go": readFile(t, "testdata/top_package.go")},
		},
		{
			Name:  "a.org@v1.1.1",
			Files: map[string]interface{}{"A/A.go": readFile(t, "testdata/a_dep.go")},
		},
		{
			Name:  "b.org@v1.2.2",
			Files: map[string]interface{}{"B/B.go": readFile(t, "testdata/b_dep.go")},
		},
		{
			Name:  "thirdparty.org/vulnerabilities@v1.0.1",
			Files: map[string]interface{}{"vuln/vuln.go": readFile(t, "testdata/vuln.go")},
		},
	})
	defer e.Cleanup()

	_, ssaPkgs, _, err := loadAndBuildPackages(e, "/vulntest/T/T.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(ssaPkgs) != 1 {
		t.Errorf("want 1 top level SSA package; got %d", len(ssaPkgs))
	}

	modVulns := ModuleVulnerabilities{
		{
			mod: &packages.Module{Path: "thirdparty.org/vulnerabilities", Version: "v1.0.1"},
			vulns: []*osv.Entry{
				{
					ID: "V1",
					Affected: []osv.Affected{{
						Package:           osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"},
						Ranges:            osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Introduced: "1.0.0"}, {Fixed: "1.0.4"}, {Introduced: "1.1.2"}}}},
						EcosystemSpecific: osv.EcosystemSpecific{Symbols: []string{"VulnData.Vuln", "VulnData.VulnOnPtr"}},
					}},
				},
				{
					ID: "V2",
					Affected: []osv.Affected{{
						Package:           osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"},
						Ranges:            osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Introduced: "1.0.1"}, {Fixed: "1.0.2"}}}},
						EcosystemSpecific: osv.EcosystemSpecific{Symbols: []string{"VG"}},
					}},
				},
			},
		},
	}

	return ssaPkgs, modVulns
}

func loadAndBuildPackages(e *packagestest.Exported, file string) (*ssa.Program, []*ssa.Package, []*packages.Package, error) {
	e.Config.Mode |= packages.NeedModule | packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps
	// Get the path to the test file.
	filepath := path.Join(e.Temp(), file)
	pkgs, err := packages.Load(e.Config, filepath)
	if err != nil {
		return nil, nil, nil, err
	}

	prog, ssaPkgs := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	return prog, ssaPkgs, pkgs, nil
}

// projectPosition simplifies position to only filename and location info.
func projectPosition(pos *token.Position) *token.Position {
	if pos == nil {
		return nil
	}
	fname := pos.Filename
	if fname != "" {
		fname = filepath.Base(fname)
	}
	return &token.Position{Line: pos.Line, Filename: fname}
}

// projectTrace simplifies traces for testing comparison purposes
// by simplifying position info.
func projectTrace(trace []TraceElem) []TraceElem {
	var nt []TraceElem
	for _, e := range trace {
		nt = append(nt, TraceElem{Description: e.Description, Position: projectPosition(e.Position)})
	}
	return nt
}

// projectFindings simplifies findings for testing comparison purposes. Traces
// are removed their position info, finding's position only contains file and
// line info, and vulnerabilities only have package path.
func projectFindings(findings []Finding) []Finding {
	var nfs []Finding
	for _, f := range findings {
		nf := Finding{
			Type:     f.Type,
			Symbol:   f.Symbol,
			Position: projectPosition(f.Position),
			Trace:    projectTrace(f.Trace),
			weight:   f.weight,
		}
		nfs = append(nfs, nf)
	}
	return nfs
}

func readFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load code from `%v`: %v", path, err)
	}
	return strings.ReplaceAll(string(content), "// go:build ignore", "")
}
