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
	"log"
	"os"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/exp/vulndb/internal/audit"
	"golang.org/x/exp/vulndb/internal/binscan"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/vulndb/client"
	"golang.org/x/vulndb/osv"
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

type results struct {
	Modules  []*packages.Module
	Vulns    []*osv.Entry
	Findings []audit.Finding
}

func (r *results) unreachable() []*osv.Entry {
	seen := map[string]bool{}
	for _, f := range r.Findings {
		for _, v := range f.Vulns {
			seen[v.ID] = true
		}
	}
	unseen := []*osv.Entry{}
	for _, v := range r.Vulns {
		if seen[v.ID] {
			continue
		}
		unseen = append(unseen, v)
	}
	return unseen
}

// presentTo pretty-prints results to out.
func (r *results) presentTo(out io.Writer) {
	sort.Slice(r.Vulns, func(i, j int) bool { return r.Vulns[i].ID < r.Vulns[j].ID })
	sort.SliceStable(r.Findings, func(i int, j int) bool { return audit.FindingCompare(r.Findings[i], r.Findings[j]) })
	if !*jsonFlag {
		for _, finding := range r.Findings {
			finding.Write(out)
			out.Write([]byte{'\n'})
		}
		if unreachable := r.unreachable(); len(unreachable) > 0 {
			fmt.Fprintf(out, "The following %d vulnerabilities don't affect this project:\n", len(unreachable))
			for _, u := range unreachable {
				var aliases string
				if len(u.Aliases) > 0 {
					aliases = fmt.Sprintf(" (%s)", strings.Join(u.Aliases, ", "))
				}
				fmt.Fprintf(out, "- %s%s (package imported, but vulnerable symbol is not reachable)\n", u.ID, aliases)
			}
		}
		return
	}
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}
	out.Write(b)
	out.Write([]byte{'\n'})
}

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

	r, err := run(cfg, flag.Args(), *importsFlag, dbs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}

	r.presentTo(os.Stdout)
}

func extractModules(pkgs []*packages.Package) []*packages.Module {
	modMap := map[*packages.Module]bool{}
	seen := map[*packages.Package]bool{}
	var extract func(*packages.Package, map[*packages.Module]bool)
	extract = func(pkg *packages.Package, modMap map[*packages.Module]bool) {
		if pkg == nil || seen[pkg] {
			return
		}
		if pkg.Module != nil {
			modMap[pkg.Module] = true
		}
		seen[pkg] = true
		for _, imp := range pkg.Imports {
			extract(imp, modMap)
		}
	}
	for _, pkg := range pkgs {
		extract(pkg, modMap)
	}
	modules := []*packages.Module{}
	for mod := range modMap {
		modules = append(modules, mod)
	}
	return modules
}

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func run(cfg *packages.Config, patterns []string, importsOnly bool, dbs []string) (*results, error) {
	r := &results{}
	if len(patterns) == 1 && isFile(patterns[0]) {
		modules, symbols, err := binscan.ExtractPackagesAndSymbols(patterns[0])
		if err != nil {
			return nil, err
		}

		dbClient, err := client.NewClient(dbs, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to create database client: %s", err)
		}
		vulns, err := audit.FetchVulnerabilities(dbClient, modules)
		if err != nil {
			return nil, fmt.Errorf("failed to load vulnerability dbs: %v", err)
		}
		vulns = vulns.Filter(runtime.GOOS, runtime.GOARCH)

		r.Findings = audit.VulnerablePackageSymbols(symbols, vulns)
		return r, nil
	}

	// Load packages.
	if *verboseFlag {
		log.Println("loading packages...")
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("packages contain errors")
	}
	if *verboseFlag {
		log.Printf("\t%d loaded packages\n", len(pkgs))
	}

	// Load database.
	if *verboseFlag {
		log.Println("loading database...")
	}
	r.Modules = extractModules(pkgs)
	dbClient, err := client.NewClient(dbs, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %s", err)
	}
	modVulns, err := audit.FetchVulnerabilities(dbClient, r.Modules)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vulnerabilities: %v", err)
	}
	modVulns = modVulns.Filter(runtime.GOOS, runtime.GOARCH)

	if *verboseFlag {
		log.Printf("\t%d known vulnerabilities.\n", modVulns.Num())
	}

	// Load SSA.
	if *verboseFlag {
		log.Println("building ssa...")
	}
	prog, ssaPkgs := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	if *verboseFlag {
		log.Println("\tbuilt ssa")
	}

	// Compute the findings.
	if *verboseFlag {
		log.Println("detecting vulnerabilities...")
	}
	var findings []audit.Finding
	if importsOnly {
		r.Findings = audit.VulnerableImports(ssaPkgs, modVulns)
	} else {
		r.Findings = audit.VulnerableSymbols(ssaPkgs, modVulns)
	}
	if *verboseFlag {
		log.Printf("\t%d detected findings.\n", len(findings))
	}

	return r, nil
}
