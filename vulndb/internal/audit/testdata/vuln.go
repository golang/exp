// go:build ignore

package vuln

var VG int

type VulnData struct{}

func (v VulnData) Vuln() {}

func Vuln() {
	print(VG)
}

// Part of a test program consisting of packages found in
// top_package.go, a_dep.go, and b_dep.go. For more details,
// see testProgAndEnv function in helpers_test.go.
