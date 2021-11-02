// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"fmt"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/packages/packagestest"
	"golang.org/x/vulndb/osv"
)

type mockClient struct {
	ret map[string][]*osv.Entry
}

func (mc *mockClient) GetByModule(a string) ([]*osv.Entry, error) {
	return mc.ret[a], nil
}

func (mc *mockClient) GetByID(a string) (*osv.Entry, error) {
	return nil, nil
}

// testClient contains the following test vulnerabilities
//   golang.org/amod/avuln.{VulnData.Vuln1, vulnData.Vuln2}
//   golang.org/bmod/bvuln.{Vuln}
var testClient = &mockClient{
	ret: map[string][]*osv.Entry{
		"golang.org/amod": []*osv.Entry{
			{
				ID: "VA",
				Affected: []osv.Affected{{
					Package:           osv.Package{Name: "golang.org/amod/avuln"},
					Ranges:            osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Introduced: "1.0.0"}, {Fixed: "1.0.4"}, {Introduced: "1.1.2"}}}},
					EcosystemSpecific: osv.EcosystemSpecific{Symbols: []string{"VulnData.Vuln1", "VulnData.Vuln2"}},
				}},
			},
		},
		"golang.org/bmod": []*osv.Entry{
			{
				ID: "VB",
				Affected: []osv.Affected{{
					Package:           osv.Package{Name: "golang.org/bmod/bvuln"},
					Ranges:            osv.Affects{{Type: osv.TypeSemver}},
					EcosystemSpecific: osv.EcosystemSpecific{Symbols: []string{"Vuln"}},
				}},
			},
		},
	},
}

func moduleVulnerabilitiesToString(mv moduleVulnerabilities) string {
	var s string
	for _, m := range mv {
		s += fmt.Sprintf("mod: %v\n", m.mod)
		for _, v := range m.vulns {
			s += fmt.Sprintf("\t%v\n", v)
		}
	}
	return s
}

func vulnsToString(vulns []*osv.Entry) string {
	var s string
	for _, v := range vulns {
		s += fmt.Sprintf("\t%v\n", v)
	}
	return s
}

func impGraphToStrMap(ig *ImportGraph) map[string][]string {
	m := make(map[string][]string)
	for _, n := range ig.Packages {
		for _, predId := range n.ImportedBy {
			pred := ig.Packages[predId]
			m[pred.Path] = append(m[pred.Path], n.Path)
		}
	}
	return m
}

func reqGraphToStrMap(rg *RequireGraph) map[string][]string {
	m := make(map[string][]string)
	for _, n := range rg.Modules {
		for _, predId := range n.RequiredBy {
			pred := rg.Modules[predId]
			m[pred.Path] = append(m[pred.Path], n.Path)
		}
	}
	return m
}

func loadPackages(e *packagestest.Exported, patterns ...string) ([]*packages.Package, error) {
	e.Config.Mode |= packages.NeedModule | packages.NeedName | packages.NeedFiles |
		packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes |
		packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps
	return packages.Load(e.Config, patterns...)
}
