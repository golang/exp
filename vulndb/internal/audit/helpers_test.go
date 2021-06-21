// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"go/token"
	"io/ioutil"
	"os"
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
//
// The produced environment is based on testdata/dbs vulnerability databases.
func testProgAndEnv(t *testing.T) ([]*ssa.Package, Env) {
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

	_, ssaPkgs, pkgs, err := loadAndBuildPackages(e, "/vulntest/T/T.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(ssaPkgs) != 1 {
		t.Errorf("want 1 top level SSA package; got %d", len(ssaPkgs))
	}

	vulnsToLoad := []string{"thirdparty.org/vulnerabilities", "bogus.org/module"}
	dbSources := []string{fileSource(t, "testdata/dbs/bogus.db.org"), fileSource(t, "testdata/dbs/golang.deepgo.org")}
	vulns, err := LoadVulnerabilities(dbSources, vulnsToLoad)
	if err != nil {
		t.Fatal(err)
	}

	return ssaPkgs, Env{OS: "linux", Arch: "amd64", Vulns: vulns, PkgVersions: PackageVersions(pkgs)}
}

func loadAndBuildPackages(e *packagestest.Exported, file string) (*ssa.Program, []*ssa.Package, []*packages.Package, error) {
	e.Config.Mode |= packages.NeedModule | packages.LoadAllSyntax
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

// projectVulns simplifies vulnerabilities for testing comparison purposes
// to only package path.
func projectVulns(vulns []osv.Entry) []osv.Entry {
	var nv []osv.Entry
	for _, v := range vulns {
		nv = append(nv, osv.Entry{Package: osv.Package{Name: v.Package.Name}})
	}
	return nv
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
			Vulns:    projectVulns(f.Vulns),
			weight:   f.weight,
		}
		nfs = append(nfs, nf)
	}
	return nfs
}

// fileSource creates a file URI for a database path `db`. If `db` is
// relative, the source is made absolute w.r.t. the current directory.
func fileSource(t *testing.T, db string) string {
	cd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return "file://" + path.Join(cd, db)
}

func readFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load code from `%v`: %v", path, err)
	}
	return strings.ReplaceAll(string(content), "// go:build ignore", "")
}
