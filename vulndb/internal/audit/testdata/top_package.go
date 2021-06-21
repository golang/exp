// go:build ignore

package T

import (
	"a.org/A"
	"b.org/B"
	"thirdparty.org/vulnerabilities/vuln"
)

func T1(x bool) {
	print(vuln.VG) // vuln use
	if x {
		A.A1().Vuln() // vuln use
	} else {
		B.B1() // no vuln use
	}
}

func T2(x bool) {
	if x {
		A.A2()() // vuln use. The return value of A.A2() is stored in register t0
	} else {
		A.A3()()
		w := A.A4(benign)
		w.V.Vuln() // no vuln use with vta-vta
	}
}

func benign(i A.I) {}

// Part of a test program consisting of packages found in
// vuln.go, a_dep.go, and b_dep.go. For more details,
// see testProgAndEnv function in helpers_test.go
