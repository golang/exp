// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"reflect"
	"sort"
	"testing"

	"golang.org/x/vulndb/osv"
)

func TestImportedPackageVulnDetection(t *testing.T) {
	pkgs, env := testProgAndEnv(t)
	got := projectFindings(VulnerableImports(pkgs, env))

	// There should be two chains reported in the following order:
	//   T -> vuln
	//   T -> A -> vuln
	want := []Finding{
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln",
			Trace:  []TraceElem{{Description: "command-line-arguments"}},
			Type:   ImportType,
			Vulns: []osv.Entry{
				{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}},
				{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight: 1,
		},
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln",
			Trace:  []TraceElem{{Description: "command-line-arguments"}, {Description: "a.org/A"}},
			Type:   ImportType,
			Vulns: []osv.Entry{
				{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}},
				{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight: 2,
		},
	}

	if len(want) != len(got) {
		t.Errorf("want %d findings; got %d", len(want), len(got))
		return
	}

	sort.SliceStable(got, func(i int, j int) bool { return FindingCompare(got[i], got[j]) })
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v findings (projected); got %v", want, got)
	}
}
