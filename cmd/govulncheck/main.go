// Copyright 2022 The Go Authors. All rights reserved.
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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"strings"

	"golang.org/x/exp/vulncheck"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/vuln/client"
	"golang.org/x/vuln/osv"
)

var (
	jsonFlag    = flag.Bool("json", false, "")
	verboseFlag = flag.Bool("v", false, "")
	testsFlag   = flag.Bool("tests", false, "")
)

const usage = `govulncheck: identify known vulnerabilities by call graph traversal.

Usage:

	govulncheck [-json] [-all] [-tests] [-tags] {package pattern...}

	govulncheck {binary path}

Flags:

	-json  	   Print vulnerability findings in JSON format.

	-tags	   Comma-separated list of build tags.

	-tests     Boolean flag indicating if test files should be analyzed too.

govulncheck can be used with either one or more package patterns (i.e. golang.org/x/crypto/...
or ./...) or with a single path to a Go binary. In the latter case module and symbol
information will be extracted from the binary in order to detect vulnerable symbols.

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
		die("%s", usage)
	}

	dbs := []string{"https://storage.googleapis.com/go-vulndb"}
	if GOVULNDB := os.Getenv("GOVULNDB"); GOVULNDB != "" {
		dbs = strings.Split(GOVULNDB, ",")
	}
	dbClient, err := client.NewClient(dbs, client.Options{HTTPCache: defaultCache()})
	if err != nil {
		die("govulncheck: %s", err)
	}
	vcfg := &vulncheck.Config{
		Client: dbClient,
	}
	ctx := context.Background()

	patterns := flag.Args()
	var r *vulncheck.Result
	var pkgs []*packages.Package
	if len(patterns) == 1 && isFile(patterns[0]) {
		f, err := os.Open(patterns[0])
		if err != nil {
			die("govulncheck: %v", err)
		}
		defer f.Close()
		r, err = vulncheck.Binary(ctx, f, vcfg)
		if err != nil {
			die("govulncheck: %v", err)
		}
	} else {
		cfg := &packages.Config{
			Mode:       packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedModule,
			Tests:      *testsFlag,
			BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(build.Default.BuildTags, ","))},
		}
		pkgs, err = loadPackages(cfg, patterns)
		if err != nil {
			die("govulncheck: %v", err)
		}
		r, err = vulncheck.Source(ctx, vulncheck.Convert(pkgs), vcfg)
		if err != nil {
			die("govulncheck: %v", err)
		}
	}
	if *jsonFlag {
		writeJSON(r)
	} else {
		writeText(r, pkgs)
	}

}

func writeJSON(r *vulncheck.Result) {
	b, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		die("govulncheck: %s", err)
	}
	os.Stdout.Write(b)
	fmt.Println()
}

func writeText(r *vulncheck.Result, pkgs []*packages.Package) {
	if len(r.Vulns) == 0 {
		return
	}
	// Build a map from module paths to versions.
	moduleVersions := map[string]string{}
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		m := p.Module
		if m != nil {
			if m.Replace != nil {
				m = m.Replace
			}
			moduleVersions[m.Path] = m.Version
		}
	})
	t := newTable("Info", "Description", "Symbols")
	for _, v := range r.Vulns {
		desc := wrap(v.OSV.Details, 30)
		current := moduleVersions[v.ModPath]
		fixed := "v" + latestFixed(v.OSV.Affected)
		ref := fmt.Sprintf("https://pkg.go.dev/vuln/%s", v.OSV.ID)
		col1 := fmt.Sprintf("%s\nyours: %s\nfixed: %s\n%s",
			v.PkgPath, current, fixed, ref)
		t.row(col1, desc, " "+v.Symbol)
		// empty row
		t.row("", "", "")
	}
	if err := t.write(os.Stdout); err != nil {
		die("govulncheck: %v", err)
	}
}

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func loadPackages(cfg *packages.Config, patterns []string) ([]*packages.Package, error) {
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
	return pkgs, nil
}

// latestFixed returns the latest fixed version in the list of affected ranges,
// or the empty string if there are no fixed versions.
func latestFixed(as []osv.Affected) string {
	v := ""
	for _, a := range as {
		for _, r := range a.Ranges {
			if r.Type == osv.TypeSemver {
				for _, e := range r.Events {
					if e.Fixed != "" && (v == "" || semver.Compare(e.Fixed, v) > 0) {
						v = e.Fixed
					}
				}
			}
		}
	}
	return v
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
