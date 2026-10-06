// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/exp/apidiff"
)

// TestModuleExportDataRoundTrip checks that writing a module's export
// data and reading it back yields the same set of packages with the
// same API. In particular, dependencies of the module's packages must
// not appear as packages of the module, and main packages must not be
// dropped; either would be reported as a spurious change when the file
// is compared against the module loaded from source.
func TestModuleExportDataRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go command not available: %v", err)
	}

	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.24\n",
		"a/a.go": `package a

import "fmt"

type T struct{ S fmt.Stringer }

type G[E any] struct{ Elem E }

type Alias = G[int]
`,
		"b/b.go": `package b

import (
	"strings"

	"example.com/m/a"
)

func F(t a.T) a.Alias { return a.Alias{} }

var R *strings.Reader
`,
		"cmd/c/main.go": `package main

import "example.com/m/b"

var _ = b.F

func main() {}
`,
	} {
		filename := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	fset := token.NewFileSet()
	loaded, err := loadModule(fset, ".")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(t.TempDir(), "export")
	if err := writeModuleExportData(fset, loaded, file); err != nil {
		t.Fatal(err)
	}
	read, err := readModuleExportData(token.NewFileSet(), file)
	if err != nil {
		t.Fatal(err)
	}

	if read.Path != loaded.Path {
		t.Errorf("module path: got %q, want %q", read.Path, loaded.Path)
	}
	paths := func(m *apidiff.Module) []string {
		var paths []string
		for _, pkg := range m.Packages {
			paths = append(paths, pkg.Path())
		}
		slices.Sort(paths)
		return paths
	}
	if got, want := paths(read), paths(loaded); !slices.Equal(got, want) {
		t.Errorf("packages: got %q, want %q", got, want)
	}

	// Compare in both directions.
	for _, report := range []apidiff.Report{
		apidiff.ModuleChanges(loaded, read),
		apidiff.ModuleChanges(read, loaded),
	} {
		for _, c := range report.Changes {
			t.Errorf("unexpected change: %s", c.Message)
		}
	}
}
