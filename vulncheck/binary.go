// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"io"

	"golang.org/x/exp/vulncheck/internal/binscan"
)

// Binary detects presence of vulnerable symbols in exe. The
// imports, require, and call graph are all unavailable (nil).
func Binary(exe io.ReaderAt, cfg *Config) (*Result, error) {
	modules, packageSymbols, err := binscan.ExtractPackagesAndSymbols(exe)
	if err != nil {
		return nil, err
	}
	modVulns, err := fetchVulnerabilities(cfg.Client, modules)
	if err != nil {
		return nil, err
	}

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
