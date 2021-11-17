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
// database should be relied on to be stable or comprehensive.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"runtime"
	"strings"

	"golang.org/x/exp/vulndb/internal/audit"
	"golang.org/x/exp/vulndb/internal/binscan"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/vuln/client"
)

var (
	jsonFlag    = flag.Bool("json", false, "")
	verboseFlag = flag.Bool("verbose", false, "")
	importsFlag = flag.Bool("imports", false, "")
	allFlag     = flag.Bool("all", false, "")
	testsFlag   = flag.Bool("tests", false, "")
)

const usage = `govulncheck: identify known vulnerabilities by call graph traversal.

Usage:

	govulncheck [-imports] [-json] [-all] [-tests] [-tags] {package pattern...}

	govulncheck {binary path}

Flags:

	-imports   Perform a broad scan with more false positives, which reports all
	           vulnerabilities found in any transitively imported package, regardless
	           of whether they are reachable.

	-json  	   Print vulnerability findings in JSON format.

	-all       Show all representative findings for each vulnerability. A best effort
		   is made to order findings by relevance. When false [default], show only
		   the most relevant finding.

	-verbose   Print progress information.

	-tags	   Comma-separated list of build tags.

	-tests     Boolean flag indicating if test files should be analyzed too.

govulncheck can be used with either one or more package patterns (i.e. golang.org/x/crypto/...
or ./...) or with a single path to a Go binary. In the latter case module and symbol
information will be extracted from the binary in order to detect vulnerable symbols
and the -imports flag is disregarded.

The environment variable GOVULNDB can be set to a comma-separate list of vulnerability
database URLs, with http://, https://, or file:// protocols. Entries from multiple
databases are merged.
`

func init() {
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
}

func main() {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	dbs := []string{"https://storage.googleapis.com/go-vulndb"}
	if GOVULNDB := os.Getenv("GOVULNDB"); GOVULNDB != "" {
		dbs = strings.Split(GOVULNDB, ",")
	}
	dbClient, err := client.NewClient(dbs, client.Options{HTTPCache: defaultCache()})
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}

	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedModule,
		Tests:      *testsFlag,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(build.Default.BuildTags, ","))},
	}

	r, err := run(cfg, flag.Args(), *importsFlag, dbClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}

	if !*allFlag {
		r = projectToSingleFinding(r)
	}

	writeOut(r, *jsonFlag)
}

func projectToSingleFinding(r *audit.Results) *audit.Results {
	nr := &audit.Results{
		SearchMode:      r.SearchMode,
		Vulnerabilities: r.Vulnerabilities,
		VulnFindings:    make(map[string][]audit.Finding),
	}

	for id, findings := range r.VulnFindings {
		if len(findings) > 0 {
			nr.VulnFindings[id] = []audit.Finding{findings[0]}
		}
	}

	return nr
}

func writeOut(r *audit.Results, toJson bool) {
	if !toJson {
		os.Stdout.Write([]byte(r.String()))
		return
	}

	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck: %s\n", err)
		os.Exit(1)
	}
	os.Stdout.Write(b)
	os.Stdout.Write([]byte{'\n'})
}

// extractModules collects modules in `pkgs` up to uniqueness of
// module path and version.
func extractModules(pkgs []*packages.Package) []*packages.Module {
	modMap := map[string]*packages.Module{}
	modKey := func(mod *packages.Module) string {
		if mod.Replace != nil {
			return fmt.Sprintf("%s@%s", mod.Replace.Path, mod.Replace.Version)
		}
		return fmt.Sprintf("%s@%s", mod.Path, mod.Version)
	}

	seen := map[*packages.Package]bool{}
	var extract func(*packages.Package, map[string]*packages.Module)
	extract = func(pkg *packages.Package, modMap map[string]*packages.Module) {
		if pkg == nil || seen[pkg] {
			return
		}
		if pkg.Module != nil {
			modMap[modKey(pkg.Module)] = pkg.Module
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
	for _, mod := range modMap {
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

func run(cfg *packages.Config, patterns []string, importsOnly bool, dbClient client.Client) (*audit.Results, error) {
	if len(patterns) == 1 && isFile(patterns[0]) {
		modules, symbols, err := binscan.ExtractPackagesAndSymbols(patterns[0])
		if err != nil {
			return nil, err
		}

		vulns, err := audit.FetchVulnerabilities(dbClient, modules)
		if err != nil {
			return nil, fmt.Errorf("failed to load vulnerability dbs: %v", err)
		}
		vulns = vulns.Filter(runtime.GOOS, runtime.GOARCH)

		results := audit.VulnerablePackageSymbols(symbols, vulns)
		return &results, nil
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

	modVulns, err := audit.FetchVulnerabilities(dbClient, extractModules(pkgs))
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
	var results audit.Results
	if importsOnly {
		results = audit.VulnerableImports(ssaPkgs, modVulns)
	} else {
		results = audit.VulnerableSymbols(ssaPkgs, modVulns)
	}
	return &results, nil
}
