// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"go/token"
	"reflect"
	"sort"
	"testing"

	"golang.org/x/vulndb/osv"
)

func TestSymbolVulnDetectionVTA(t *testing.T) {
	pkgs, modVulns := testContext(t)
	got := projectFindings(VulnerableSymbols(pkgs, modVulns))

	// There should be four call chains reported with VTA-VTA version, in the following order:
	//   T:T1() -> vuln.VG                                     [use of global at line 4]
	//   T:T1() -> A:A1() -> vuln.VulnData.Vuln()              [call at A.go:14]
	//   T:T2() -> vuln.Vuln() [approx.resolved] -> vuln.VG    [use of global at vuln.go:4]
	//   T:T1() -> vuln.VulnData.Vuln() [approx. resolved]     [call at testdata.go:13]
	// Without VTA-VTA, we would alse have the following false positive:
	//   T:T2() -> vuln.VulnData.Vuln() [approx. resolved]     [call at testdata.go:26]
	want := []Finding{
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln.VG",
			Trace: []TraceElem{
				{Description: "command-line-arguments.T1(...)", Position: &token.Position{Line: 11, Filename: "T.go"}},
			},
			Type:     GlobalType,
			Position: &token.Position{Line: 5, Filename: "vuln.go"},
			Vulns:    []osv.Entry{{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight:   0,
		},
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln.VulnData.Vuln",
			Trace: []TraceElem{
				{Description: "command-line-arguments.T1(...)", Position: &token.Position{Line: 11, Filename: "T.go"}},
				{Description: "a.org/A.A1(...)", Position: &token.Position{Line: 14, Filename: "T.go"}}},
			Type:     FunctionType,
			Position: &token.Position{Line: 15, Filename: "A.go"},
			Vulns:    []osv.Entry{{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight:   0,
		},
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln.VG",
			Trace: []TraceElem{
				{Description: "command-line-arguments.T2(...)", Position: &token.Position{Line: 20, Filename: "T.go"}},
				{Description: "command-line-arguments.t0(...) [approx. resolved to thirdparty.org/vulnerabilities/vuln.Vuln]", Position: &token.Position{Line: 22, Filename: "T.go"}},
			},
			Type:     GlobalType,
			Position: &token.Position{Line: 5, Filename: "vuln.go"},
			Vulns:    []osv.Entry{{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight:   1,
		},
		{
			Symbol: "thirdparty.org/vulnerabilities/vuln.VulnData.Vuln",
			Trace: []TraceElem{
				{Description: "command-line-arguments.T1(...)", Position: &token.Position{Line: 11, Filename: "T.go"}},
				{Description: "a.org/A.I.Vuln(...) [approx. resolved to (thirdparty.org/vulnerabilities/vuln.VulnData).Vuln]", Position: &token.Position{Line: 14, Filename: "T.go"}}},
			Type:     FunctionType,
			Position: &token.Position{Line: 14, Filename: "T.go"},
			Vulns:    []osv.Entry{{Package: osv.Package{Name: "thirdparty.org/vulnerabilities/vuln"}}},
			weight:   1,
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
