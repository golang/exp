// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"testing"

	"golang.org/x/tools/go/packages/packagestest"
)

func TestPackageVersionInfo(t *testing.T) {
	// Export package testdata with a program depending on a vulnerability package
	// vuln with version "v1.0.1".
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "golang.org/vulntest",
			Files: map[string]interface{}{
				"testdata/testdata.go": `
					package testdata

					import (
						"thirdparty.org/vulnerabilities/vuln"
					)

					func Lib1() {
						vuln.Vuln()
					}
					`,
			},
		},
		{
			Name: "thirdparty.org/vulnerabilities@v1.0.1",
			Files: map[string]interface{}{
				"vuln/vuln.go": `
					package vuln

					import (
						"abc.org/xyz/foo"
					)

					func Vuln() { foo.Foo() }
					`,
			},
		},
		{
			Name: "abc.org/xyz@v0.0.0-20201002170205-7f63de1d35b0",
			Files: map[string]interface{}{
				"foo/foo.go": `
					package foo

					func Foo() { }
					`,
			},
		},
	})
	defer e.Cleanup()

	_, _, pkgs, err := loadAndBuildPackages(e, "/vulntest/testdata/testdata.go")
	if err != nil {
		t.Fatal(err)
	}

	v := PackageVersions(pkgs)
	for _, test := range []struct {
		path    string
		version string
		in      bool
	}{
		{"command-line-arguments", "", false},
		{"thirdparty.org/vulnerabilities/vuln", "v1.0.1", true},
		{"abc.org/xyz/foo", "v0.0.0-20201002170205-7f63de1d35b0", true},
	} {
		if version, ok := v[test.path]; ok != test.in || version != test.version {
			t.Errorf("want package %s at version %s in=%t package-version map; got %s and %t", test.path, test.version, test.in, version, ok)
		}
	}
}
