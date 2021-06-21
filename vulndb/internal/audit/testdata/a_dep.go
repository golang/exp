// go:build ignore

package A

import (
	"thirdparty.org/vulnerabilities/vuln"
)

type I interface {
	Vuln()
}

func A1() I {
	v := vuln.VulnData{}
	v.Vuln() // vuln use
	return v
}

func A2() func() {
	return vuln.Vuln
}

func A3() func() {
	return func() {}
}

type vulnWrap struct {
	V I
}

func A4(f func(i I)) vulnWrap {
	f(vuln.VulnData{})
	return vulnWrap{}
}

func doWrap(i I) {
	w := vulnWrap{}
	w.V = i
}

// Part of a test program consisting of packages found
// in top_package.go, b_dep.go, and vuln.go. For more
// details, see testProgAndEnv function in helpers_test.go.
