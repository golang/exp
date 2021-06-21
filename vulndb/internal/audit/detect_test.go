// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"testing"

	"golang.org/x/vulndb/osv"
)

var testVulnerabilities = []*osv.Entry{
	{
		Package: osv.Package{
			Name: "xyz.org/vuln",
		},
		Affects: osv.Affects{
			Ranges: []osv.AffectsRange{
				{
					Type:       osv.TypeSemver,
					Introduced: "v1.0.1",
					Fixed:      "v3.2.6",
				},
			},
		},
		EcosystemSpecific: osv.GoSpecific{
			Symbols: []string{"foo", "bar"},
			GOOS:    []string{"amd64"},
			GOARCH:  []string{"linux"},
		},
	},
	{
		Package: osv.Package{
			Name: "xyz.org/vuln",
		},
		Affects: osv.Affects{
			Ranges: []osv.AffectsRange{
				{
					Type:  osv.TypeSemver,
					Fixed: "v4.0.0",
				},
			},
		},
		EcosystemSpecific: osv.GoSpecific{
			Symbols: []string{"foo"},
		},
	},
	{
		Package: osv.Package{
			Name: "abc.org/morevuln",
		},
	},
}

func TestPackageVulnCreationAndChecking(t *testing.T) {
	pkgVulns := createPkgVulns(testVulnerabilities)
	if len(pkgVulns) != 2 {
		t.Errorf("want 2 package paths; got %d", len(pkgVulns))
	}

	for _, test := range []struct {
		path    string
		version string
		os      string
		arch    string
		noVulns int
	}{
		// xyz.org/vuln has foo and bar vulns for linux, and just foo for windows.
		{"xyz.org/vuln", "v1.0.1", "amd64", "linux", 2},
		{"xyz.org/vuln", "v1.0.1", "amd64", "windows", 1},
		{"xyz.org/vuln", "v2.4.5", "amd64", "linux", 2},
		{"xyz.org/vuln", "v3.2.7", "amd64", "linux", 1},
		// foo for linux must be at version before v4.0.0.
		{"xyz.org/vuln", "v5.4.5", "amd64", "linux", 0},
		// abc.org/morevuln has vulnerabilities for any symbol, platform, and version
		{"abc.org/morevuln", "v11.0.1", "amd64", "linux", 1},
		{"abc.org/morevuln", "v300.0.1", "i386", "windows", 1},
	} {
		if vulns := pkgVulns.vulnerabilities(test.path, test.version, test.arch, test.os); len(vulns) != test.noVulns {
			t.Errorf("want %d vulnerabilities for %s (v:%s, o:%s, a:%s); got %d",
				test.noVulns, test.path, test.version, test.os, test.path, len(vulns))
		}
	}
}

func TestSymbolVulnCreationAndChecking(t *testing.T) {
	symVulns := createSymVulns(testVulnerabilities)
	if len(symVulns) != 2 {
		t.Errorf("want 2 package paths; got %d", len(symVulns))
	}

	for _, test := range []struct {
		symbol   string
		path     string
		version  string
		os       string
		arch     string
		numVulns int
	}{
		// foo appears twice as a vulnerable symbol for "xyz.org/vuln" and bar once.
		{"foo", "xyz.org/vuln", "v1.0.1", "amd64", "linux", 2},
		{"bar", "xyz.org/vuln", "v1.0.1", "amd64", "linux", 1},
		// foo and bar detected vulns should go down by one for windows platform as well as i386 architecture.
		{"foo", "xyz.org/vuln", "v1.0.1", "amd64", "windows", 1},
		{"bar", "xyz.org/vuln", "v1.0.1", "i386", "linux", 0},
		// There should be no findings for foo and bar at module version v5.0.0.
		{"foo", "xyz.org/vuln", "v5.0.0", "amd64", "linux", 0},
		{"bar", "xyz.org/vuln", "v5.0.0", "amd64", "linux", 0},
		// symbol is not a vulnerable symbol for xyz.org/vuln and bogus package is not in the database.
		{"symbol", "xyz.org/vuln", "v1.0.1", "amd64", "linux", 0},
		{"foo", "bogus", "v1.0.1", "amd64", "linux", 0},
		// abc.org/morevuln has vulnerabilities for any symbol, platform, and version
		{"symbol", "abc.org/morevuln", "v2.0.1", "amd64", "linux", 1},
		{"lobmys", "abc.org/morevuln", "v300.0.1", "i386", "windows", 1},
	} {
		if vulns := symVulns.vulnerabilities(test.symbol, test.path, test.version, test.arch, test.os); len(vulns) != test.numVulns {
			t.Errorf("want %d vulnerabilities for %s (p:%s v:%s, o:%s, a:%s); got %d",
				test.numVulns, test.symbol, test.path, test.version, test.os, test.arch, len(vulns))
		}
	}
}
