// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"context"
	"io"
	"runtime"

	"golang.org/x/exp/vulncheck/internal/binscan"
	"golang.org/x/exp/vulncheck/internal/derrors"
	"golang.org/x/tools/go/packages"
)

// Binary detects presence of vulnerable symbols in exe. The
// imports, require, and call graph are all unavailable (nil).
func Binary(ctx context.Context, exe io.ReaderAt, cfg *Config) (_ *Result, err error) {
	defer derrors.Wrap(&err, "vulncheck.Binary")

	mods, packageSymbols, err := binscan.ExtractPackagesAndSymbols(exe)
	if err != nil {
		return nil, err
	}
	modVulns, err := fetchVulnerabilities(ctx, cfg.Client, convertModules(mods))
	if err != nil {
		return nil, err
	}
	modVulns = modVulns.Filter(lookupEnv("GOOS", runtime.GOOS), lookupEnv("GOARCH", runtime.GOARCH))

	result := &Result{}
	for pkg, symbols := range packageSymbols {
		if cfg.ImportsOnly {
			addImportsOnlyVulns(pkg, symbols, result, modVulns)
		} else {
			addSymbolVulns(pkg, symbols, result, modVulns)
		}
	}
	return result, nil
}

// addImportsOnlyVulns adds Vuln entries to result in imports only mode, i.e., for each vulnerable symbol
// of pkg.
func addImportsOnlyVulns(pkg string, symbols []string, result *Result, modVulns moduleVulnerabilities) {
	for _, osv := range modVulns.VulnsForPackage(pkg) {
		for _, affected := range osv.Affected {
			if affected.Package.Name != pkg {
				continue
			}

			var syms []string
			if len(affected.EcosystemSpecific.Symbols) == 0 {
				// If every symbol of pkg is vulnerable, we would ideally compute
				// every symbol mentioned in the pkg and then add Vuln entry for it,
				// just as we do in Source. However, we don't have code of pkg here
				// so we have to do best we can, which is the symbols of pkg actually
				// appearing in the binary.
				syms = symbols
			} else {
				syms = affected.EcosystemSpecific.Symbols
			}

			for _, symbol := range syms {
				vuln := &Vuln{
					OSV:     osv,
					Symbol:  symbol,
					PkgPath: pkg,
					// TODO(zpavlinovic): infer mod path from PkgPath and modules?
				}
				result.Vulns = append(result.Vulns, vuln)
			}
		}
	}
}

// addSymbolVulns adds Vuln entries to result for every symbol of pkg in the binary that is vulnerable.
func addSymbolVulns(pkg string, symbols []string, result *Result, modVulns moduleVulnerabilities) {
	for _, symbol := range symbols {
		for _, osv := range modVulns.VulnsForSymbol(pkg, symbol) {
			for _, affected := range osv.Affected {
				if affected.Package.Name != pkg {
					continue
				}
				vuln := &Vuln{
					OSV:     osv,
					Symbol:  symbol,
					PkgPath: pkg,
					// TODO(zpavlinovic): infer mod path from PkgPath and modules?
				}
				result.Vulns = append(result.Vulns, vuln)
				break
			}
		}
	}
}

func convertModules(mods []*packages.Module) []*Module {
	vmods := make([]*Module, len(mods))
	// TODO(github.com/golang/go/issues/50030): should we share unique
	// modules? Not needed nowas module info is not returned by Binary.
	for i, mod := range mods {
		vmods[i] = &Module{
			Path:    mod.Path,
			Version: mod.Version,
		}
		if mod.Replace != nil {
			vmods[i].Replace = &Module{
				Path:    mod.Replace.Path,
				Version: mod.Replace.Version,
			}
		}
	}
	return vmods
}
