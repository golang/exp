// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"context"
	"io"
	"runtime"

	"golang.org/x/exp/vulncheck/internal/binscan"
	"golang.org/x/tools/go/packages"
)

// Binary detects presence of vulnerable symbols in exe. The
// imports, require, and call graph are all unavailable (nil).
func Binary(ctx context.Context, exe io.ReaderAt, cfg *Config) (*Result, error) {
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
		for _, symbol := range symbols {
			for _, osv := range modVulns.VulnsForSymbol(pkg, symbol) {
				for _, affected := range osv.Affected {
					if affected.Package.Name != pkg {
						continue
					}
					for _, symbol := range affected.EcosystemSpecific.Symbols {
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
	}
	return result, nil
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
