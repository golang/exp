// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/vulndb/osv"
)

func moduleVulnerabilitiesToString(mv ModuleVulnerabilities) string {
	var s string
	for _, m := range mv {
		s += fmt.Sprintf("mod: %v\n", m.mod)
		for _, v := range m.vulns {
			s += fmt.Sprintf("\t%v\n", v)
		}
	}
	return s
}

func TestFilterVulns(t *testing.T) {
	mv := ModuleVulnerabilities{
		{
			mod: &packages.Module{
				Path:    "example.mod/a",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "a", Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Fixed: "v2.0.0"}}}},
				{ID: "b", EcosystemSpecific: osv.GoSpecific{GOOS: []string{"windows", "linux"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v1.0.1"}}}},
				{ID: "c", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"arm64", "amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v1.0.0", Fixed: "v1.0.1"}}}},
				{ID: "d", EcosystemSpecific: osv.GoSpecific{GOOS: []string{"windows"}}},
			},
		},
		{
			mod: &packages.Module{
				Path:    "example.mod/b",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "e", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"arm64"}}},
				{ID: "f", EcosystemSpecific: osv.GoSpecific{GOOS: []string{"linux"}}},
				{ID: "g", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v0.0.1", Fixed: "v2.0.1"}}}},
				{ID: "h", EcosystemSpecific: osv.GoSpecific{GOOS: []string{"windows"}, GOARCH: []string{"amd64"}}},
			},
		},
		{
			mod: &packages.Module{
				Path: "example.mod/c",
			},
			vulns: []*osv.Entry{
				{ID: "i", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v0.0.0"}}}},
				{ID: "j", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Fixed: "v3.0.0"}}}},
				{ID: "k"},
			},
		},
	}

	filtered := mv.Filter("linux", "amd64")

	expected := ModuleVulnerabilities{
		{
			mod: &packages.Module{
				Path:    "example.mod/a",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "a", Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Fixed: "v2.0.0"}}}},
				{ID: "c", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"arm64", "amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v1.0.0", Fixed: "v1.0.1"}}}},
			},
		},
		{
			mod: &packages.Module{
				Path:    "example.mod/b",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "f", EcosystemSpecific: osv.GoSpecific{GOOS: []string{"linux"}}},
				{ID: "g", EcosystemSpecific: osv.GoSpecific{GOARCH: []string{"amd64"}}, Affects: osv.Affects{Ranges: []osv.AffectsRange{{Type: osv.TypeSemver, Introduced: "v0.0.1", Fixed: "v2.0.1"}}}},
			},
		},
		{
			mod: &packages.Module{
				Path: "example.mod/c",
			},
		},
	}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("Filter returned unexpected results, got:\n%s\nwant:\n%s", moduleVulnerabilitiesToString(filtered), moduleVulnerabilitiesToString(expected))
	}
}

func vulnsToString(vulns []*osv.Entry) string {
	var s string
	for _, v := range vulns {
		s += fmt.Sprintf("\t%v\n", v)
	}
	return s
}

func TestVulnsForPackage(t *testing.T) {
	mv := ModuleVulnerabilities{
		{
			mod: &packages.Module{
				Path:    "example.mod/a",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "a", Package: osv.Package{Name: "example.mod/a/b/c"}},
			},
		},
		{
			mod: &packages.Module{
				Path:    "example.mod/a/b",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "b", Package: osv.Package{Name: "example.mod/a/b/c"}},
			},
		},
		{
			mod: &packages.Module{
				Path:    "example.mod/d",
				Version: "v0.0.1",
			},
			vulns: []*osv.Entry{
				{ID: "d", Package: osv.Package{Name: "example.mod/d"}},
			},
		},
	}

	filtered := mv.VulnsForPackage("example.mod/a/b/c")
	expected := []*osv.Entry{
		{ID: "b", Package: osv.Package{Name: "example.mod/a/b/c"}},
	}

	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("VulnsForPackage returned unexpected results, got:\n%s\nwant:\n%s", vulnsToString(filtered), vulnsToString(expected))
	}
}

func TestVulnsForPackageReplaced(t *testing.T) {
	mv := ModuleVulnerabilities{
		{
			mod: &packages.Module{
				Path:    "example.mod/a",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "a", Package: osv.Package{Name: "example.mod/a/b/c"}},
			},
		},
		{
			mod: &packages.Module{
				Path: "example.mod/a/b",
				Replace: &packages.Module{
					Path: "example.mod/b",
				},
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "c", Package: osv.Package{Name: "example.mod/b/c"}},
			},
		},
	}

	filtered := mv.VulnsForPackage("example.mod/a/b/c")
	expected := []*osv.Entry{
		{ID: "c", Package: osv.Package{Name: "example.mod/b/c"}},
	}

	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("VulnsForPackage returned unexpected results, got:\n%s\nwant:\n%s", vulnsToString(filtered), vulnsToString(expected))
	}
}

func TestVulnsForSymbol(t *testing.T) {
	mv := ModuleVulnerabilities{
		{
			mod: &packages.Module{
				Path:    "example.mod/a",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "a", Package: osv.Package{Name: "example.mod/a/b/c"}},
			},
		},
		{
			mod: &packages.Module{
				Path:    "example.mod/a/b",
				Version: "v1.0.0",
			},
			vulns: []*osv.Entry{
				{ID: "b", Package: osv.Package{Name: "example.mod/a/b/c"}, EcosystemSpecific: osv.GoSpecific{Symbols: []string{"a"}}},
				{ID: "c", Package: osv.Package{Name: "example.mod/a/b/c"}, EcosystemSpecific: osv.GoSpecific{Symbols: []string{"b"}}},
			},
		},
	}

	filtered := mv.VulnsForSymbol("example.mod/a/b/c", "a")
	expected := []*osv.Entry{
		{ID: "b", Package: osv.Package{Name: "example.mod/a/b/c"}, EcosystemSpecific: osv.GoSpecific{Symbols: []string{"a"}}},
	}

	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("VulnsForPackage returned unexpected results, got:\n%s\nwant:\n%s", vulnsToString(filtered), vulnsToString(expected))
	}
}
