// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command govulncheck reports known vulnerabilities filed in a vulnerability database
// (see https://golang.org/design/draft-vulndb) that affect a given package or binary.
//
// It uses static analysis or the binary's symbol table to narrow down reports to only
// those that potentially affect the application.
//
// WARNING WARNING WARNING
//
// govulncheck is still experimental and neither its output or the vulnerability
// database should be relied on to be stable or comprehensive. It also performs no
// caching of vulnerability database entries.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/exp/vulndb/internal/audit"
	"golang.org/x/exp/vulndb/internal/binscan"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	jsonFlag    = flag.Bool("json", false, "")
	verboseFlag = flag.Bool("verbose", false, "")
	importsFlag = flag.Bool("imports", false, "")
)

const usage = `govulncheck: identify known vulnerabilities by call graph traversal.

Usage:

	govulncheck [-imports] {package pattern...}

	govulncheck {binary path}

Flags:

	-imports   Perform a broad scan with more false positives, which reports all
	           vulnerabilities found in any transitively imported package, regardless
	           of whether they are reachable.

	-json  	   Print vulnerability findings in JSON format.

	-verbose   Print progress information.

govulncheck can be used with either one or more package patterns (i.e. golang.org/x/crypto/...
or ./...) or with a single path to a Go binary. In the latter case module and symbol
information will be extracted from the binary in order to detect vulnerable symbols
and the -imports flag is disregarded.

The environment variable GOVULNDB can be set to a comma-separate list of vulnerability
database URLs, with http://, https://, or file:// protocols. Entries from multiple
databases are merged.
`

func main() {
	flag.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	dbs := []string{"https://storage.googleapis.com/go-vulndb"}
	if GOVULNDB := os.Getenv("GOVULNDB"); GOVULNDB != "" {
		dbs = strings.Split(GOVULNDB, ",")
	}

	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedModule,
	}

	findings, err := run(cfg, flag.Args(), *importsFlag, dbs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}

	sort.SliceStable(findings, func(i int, j int) bool { return audit.FindingCompare(findings[i], findings[j]) })
	presentTo(os.Stdout, findings)
}

// presentTo pretty-prints findings to out.
func presentTo(out io.Writer, findings []audit.Finding) {
	if !*jsonFlag {
		for _, finding := range findings {
			finding.Write(out)
			out.Write([]byte{'\n'})
		}
		return
	}
	b, err := json.MarshalIndent(findings, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}
	out.Write(b)
	out.Write([]byte{'\n'})
}

// allPkgPaths computes a list of all packages, in
// the form of their paths, reachable from pkgs.
func allPkgPaths(pkgs []*packages.Package) []string {
	paths := make(map[string]bool)
	for _, pkg := range pkgs {
		pkgPaths(pkg, paths)
	}

	var ps []string
	for p := range paths {
		ps = append(ps, p)
	}
	return ps
}

func pkgPaths(pkg *packages.Package, paths map[string]bool) {
	if _, ok := paths[pkg.PkgPath]; ok {
		return
	}
	paths[pkg.PkgPath] = true
	for _, imp := range pkg.Imports {
		pkgPaths(imp, paths)
	}
}

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func run(cfg *packages.Config, patterns []string, importsOnly bool, dbs []string) ([]audit.Finding, error) {
	if len(patterns) == 1 && isFile(patterns[0]) {
		packages, symbols, err := binscan.ExtractPackagesAndSymbols(patterns[0])
		if err != nil {
			return nil, err
		}

		paths := make([]string, 0, len(packages))
		for pkg := range packages {
			paths = append(paths, pkg)
		}

		vulns, err := audit.LoadVulnerabilities(dbs, paths)
		if err != nil {
			return nil, fmt.Errorf("failed to load vulnerability dbs: %v", err)
		}
		env := audit.Env{OS: runtime.GOOS, Arch: runtime.GOARCH, PkgVersions: packages, Vulns: vulns}

		return audit.VulnerablePackageSymbols(symbols, env), nil
	}

	// Load packages.
	if *verboseFlag {
		fmt.Println("loading packages...")
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("packages contain errors")
	}
	if *verboseFlag {
		fmt.Printf("\t%d loaded packages\n", len(pkgs))
	}

	// Load database.
	if *verboseFlag {
		fmt.Println("loading database...")
	}
	vulns, err := audit.LoadVulnerabilities(dbs, allPkgPaths(pkgs))
	if err != nil {
		return nil, fmt.Errorf("failed to load vulnerability dbs: %v", err)
	}

	if *verboseFlag {
		fmt.Printf("\t%d known vulnerabilities.\n", len(vulns))
	}

	// Load package versions.
	pkgVersions := audit.PackageVersions(pkgs)

	// Load SSA.
	if *verboseFlag {
		fmt.Println("building ssa...")
	}
	prog, ssaPkgs := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	if *verboseFlag {
		fmt.Println("\tbuilt ssa.")
	}

	// Compute the findings.
	if *verboseFlag {
		fmt.Println("detecting vulnerabilities...")
	}
	var findings []audit.Finding
	env := audit.Env{OS: runtime.GOOS, Arch: runtime.GOARCH, PkgVersions: pkgVersions, Vulns: vulns}
	if importsOnly {
		findings = audit.VulnerableImports(ssaPkgs, env)
	} else {
		findings = audit.VulnerableSymbols(ssaPkgs, env)
	}
	if *verboseFlag {
		fmt.Printf("\t%d detected findings.\n", len(findings))
	}
	return findings, nil
}
