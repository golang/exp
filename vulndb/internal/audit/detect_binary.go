// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"fmt"
)

// VulnerablePackageSymbols returns a list of vulnerability findings for per-package symbols
// in packageSymbols, given the vulnerability and platform info captured in env.
//
// Returned Findings only have Symbol, Type, and Vulns fields set.
func VulnerablePackageSymbols(packageSymbols map[string][]string, env Env) []Finding {
	symVulns := createSymVulns(env.Vulns)

	var findings []Finding
	for pkg, symbols := range packageSymbols {
		for _, symbol := range symbols {
			if vulns := querySymbolVulns(symbol, pkg, symVulns, env); len(vulns) > 0 {
				findings = append(findings,
					Finding{
						Symbol: fmt.Sprintf("%s.%s", pkg, symbol),
						Type:   GlobalType,
						Vulns:  serialize(vulns),
					})
			}
		}
	}

	return findings
}
