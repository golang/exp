// go:build ignore

package B

import (
	"a.org/A"
)

type internal struct{}

func (i internal) Vuln() {}

func B1() {
	A.A1() // transitive vuln use but should not be reported
	var i A.I
	i = internal{}
	i.Vuln() // no vuln use
}

// Part of a test program consisting of packages found in
// vuln.go, a_dep.go, and b_dep.go. For more details, see
// testProgAndEnv function in helpers_test.go.
