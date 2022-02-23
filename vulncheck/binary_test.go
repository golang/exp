// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/tools/go/packages/packagestest"
)

// TODO: we build binary programatically, so what if the underlying tool chain changes?
func TestBinary(t *testing.T) {
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "golang.org/entry",
			Files: map[string]interface{}{
				"main.go": `
			package main

			import (
				"golang.org/cmod/c"
				"golang.org/bmod/bvuln"
			)

			func main() {
				c.C()
				bvuln.NoVuln() // no vuln use
				print("done")
			}
			`,
			}},
		{
			Name: "golang.org/cmod@v1.1.3",
			Files: map[string]interface{}{"c/c.go": `
			package c

			import (
				"golang.org/amod/avuln"
			)

			//go:noinline
			func C() {
				v := avuln.VulnData{}
				v.Vuln1() // vuln use
			}
			`},
		},
		{
			Name: "golang.org/amod@v1.1.3",
			Files: map[string]interface{}{"avuln/avuln.go": `
			package avuln

			type VulnData struct {}

			//go:noinline
			func (v VulnData) Vuln1() {}

			//go:noinline
			func (v VulnData) Vuln2() {}
			`},
		},
		{
			Name: "golang.org/bmod@v0.5.0",
			Files: map[string]interface{}{"bvuln/bvuln.go": `
			package bvuln

			//go:noinline
			func Vuln() {}

			//go:noinline
			func NoVuln() {}
			`},
		},
	})
	defer e.Cleanup()

	// Make sure local vulns can be loaded.
	fetchingInTesting = true

	cmd := exec.Command("go", "build")
	cmd.Dir = e.Config.Dir
	cmd.Env = e.Config.Env
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) > 0 {
		t.Fatalf("failed to build the binary %v %v", err, string(out))
	}

	binExt := ""
	// TODO: is there a better way to do this?
	if runtime.GOOS == "windows" {
		binExt = ".exe"
	}

	bin, err := os.Open(filepath.Join(e.Config.Dir, "entry"+binExt))
	if err != nil {
		t.Fatalf("failed to access the binary %v", err)
	}
	defer bin.Close()

	// Test imports only mode
	cfg := &Config{
		Client:      testClient,
		ImportsOnly: true,
	}
	res, err := Binary(context.Background(), bin, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// In importsOnly mode, all three vulnerable symbols
	// {avuln.VulnData.Vuln1, avuln.VulnData.Vuln2, bvuln.Vuln}
	// should be detected.
	if len(res.Vulns) != 3 {
		t.Errorf("expected 3 vuln symbols; got %d", len(res.Vulns))
	}

	// Test the symbols (non-import mode)
	cfg = &Config{Client: testClient}
	res, err = Binary(context.Background(), bin, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// In non-importsOnly mode, only one symbol avuln.VulnData.Vuln1 should be detected.
	if len(res.Vulns) != 1 {
		t.Errorf("expected 1 vuln symbols got %d", len(res.Vulns))
	}
}
