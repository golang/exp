package vulncheck

import (
	"reflect"
	"strings"
	"testing"
)

// chainsToString converts map Vuln:chains to Vuln.PkgPath:["pkg1->...->pkgN", ...]
// string representation.
func chainsToString(chains map[*Vuln][]ImportChain) map[string][]string {
	m := make(map[string][]string)
	for v, chs := range chains {
		var chsStr []string
		for _, ch := range chs {
			var chStr []string
			for _, imp := range ch {
				chStr = append(chStr, imp.Path)
			}
			chsStr = append(chsStr, strings.Join(chStr, "->"))
		}
		m[v.PkgPath] = chsStr
	}
	return m
}

func TestImportChains(t *testing.T) {
	// Package import structure for the test program
	//    entry1  entry2
	//      |       |
	//    interm1   |
	//      |    \  |
	//      |   interm2
	//      |   /     |
	//     vuln1    vuln2
	e1 := &PkgNode{ID: 1, Path: "entry1"}
	e2 := &PkgNode{ID: 2, Path: "entry2"}
	i1 := &PkgNode{ID: 3, Path: "interm1", ImportedBy: []int{1}}
	i2 := &PkgNode{ID: 4, Path: "interm2", ImportedBy: []int{2, 3}}
	v1 := &PkgNode{ID: 5, Path: "vuln1", ImportedBy: []int{3, 4}}
	v2 := &PkgNode{ID: 6, Path: "vuln2", ImportedBy: []int{4}}

	ig := &ImportGraph{
		Packages: map[int]*PkgNode{1: e1, 2: e2, 3: i1, 4: i2, 5: v1, 6: v2},
		Entries:  []int{1, 2},
	}
	vuln1 := &Vuln{ImportSink: 5, PkgPath: "vuln1"}
	vuln2 := &Vuln{ImportSink: 6, PkgPath: "vuln2"}
	res := &Result{Imports: ig, Vulns: []*Vuln{vuln1, vuln2}}

	// The chain entry1->interm1->interm2->vuln1 is not reported
	// as there exist a shorter trace going from entry1 to vuln1
	// via interm1.
	want := map[string][]string{
		"vuln1": {"entry1->interm1->vuln1", "entry2->interm2->vuln1"},
		"vuln2": {"entry2->interm2->vuln2", "entry1->interm1->interm2->vuln2"},
	}

	chains := ImportChains(res)
	if got := chainsToString(chains); !reflect.DeepEqual(want, got) {
		t.Errorf("want %v; got %v", want, got)
	}
}
