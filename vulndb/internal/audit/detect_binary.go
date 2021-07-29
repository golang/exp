// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"fmt"
)

// VulnerablePackageSymbols returns a list of vulnerability findings for per-package symbols
// in packageSymbols, given the `modVulns` vulnerabilities.
//
// Findings for each vulnerability are sorted by estimated usefulness to the user and do not
// have an associated trace.
func VulnerablePackageSymbols(packageSymbols map[string][]string, modVulns ModuleVulnerabilities) Results {
	results := Results{
		SearchMode:      BinarySearch,
		Vulnerabilities: serialize(modVulns.Vulns()),
		VulnFindings:    make(map[string][]Finding),
	}
	if len(modVulns) == 0 {
		return results
	}

	for pkg, symbols := range packageSymbols {
		for _, symbol := range symbols {
			vulns := modVulns.VulnsForSymbol(pkg, symbol)
			for _, v := range serialize(vulns) {
				results.addFinding(v, Finding{
					Symbol: fmt.Sprintf("%s.%s", pkg, symbol),
					Type:   GlobalType,
				})
			}
		}
	}

	results.sort()
	return results
}
