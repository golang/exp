// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"reflect"
	"testing"

	"golang.org/x/vulndb/osv"
)

func TestFetchVulnerabilities(t *testing.T) {
	mc := &mockClient{
		ret: map[string][]*osv.Entry{
			"example.mod/a": {{ID: "a", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/a"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "2.0.0"}}}}}}}},
			"example.mod/b": {{ID: "b", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/b"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "1.1.1"}}}}}}}},
			"example.mod/d": {{ID: "c", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/d"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "2.0.0"}}}}}}}},
			"example.mod/e": {{ID: "e", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/e"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "2.2.0"}}}}}}}},
		},
	}

	mv, err := fetchVulnerabilities(mc, []*Module{
		{Path: "example.mod/a", Dir: modCacheDirectory(), Version: "v1.0.0"},
		{Path: "example.mod/b", Dir: modCacheDirectory(), Version: "v1.0.4"},
		{Path: "example.mod/c", Replace: &Module{Path: "example.mod/d", Dir: modCacheDirectory(), Version: "v1.0.0"}, Version: "v2.0.0"},
		{Path: "example.mod/e", Replace: &Module{Path: "../local/example.mod/d", Dir: modCacheDirectory(), Version: "v1.0.1"}, Version: "v2.1.0"},
	})
	if err != nil {
		t.Fatalf("FetchVulnerabilities failed: %s", err)
	}

	expected := moduleVulnerabilities{
		{
			mod: &Module{Path: "example.mod/a", Dir: modCacheDirectory(), Version: "v1.0.0"},
			vulns: []*osv.Entry{
				{ID: "a", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/a"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "2.0.0"}}}}}}},
			},
		},
		{
			mod: &Module{Path: "example.mod/b", Dir: modCacheDirectory(), Version: "v1.0.4"},
			vulns: []*osv.Entry{
				{ID: "b", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/b"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "1.1.1"}}}}}}},
			},
		},
		{
			mod: &Module{Path: "example.mod/c", Replace: &Module{Path: "example.mod/d", Dir: modCacheDirectory(), Version: "v1.0.0"}, Version: "v2.0.0"},
			vulns: []*osv.Entry{
				{ID: "c", Affected: []osv.Affected{{Package: osv.Package{Name: "example.mod/d"}, Ranges: osv.Affects{{Type: osv.TypeSemver, Events: []osv.RangeEvent{{Fixed: "2.0.0"}}}}}}},
			},
		},
	}
	if !reflect.DeepEqual(mv, expected) {
		t.Fatalf("FetchVulnerabilities returned unexpected results, got:\n%s\nwant:\n%s", moduleVulnerabilitiesToString(mv), moduleVulnerabilitiesToString(expected))
	}
}
