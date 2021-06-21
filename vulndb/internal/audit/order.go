// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"fmt"
	"strings"
)

// FindingCompare compares two findings in terms of their approximate usefulness to the user.
// A finding that either has 1) shorter trace, or 2) less unresolved call sites in the trace
// is considered smaller, i.e., better.
func FindingCompare(finding1, finding2 Finding) bool {
	if len(finding1.Trace) < len(finding2.Trace) {
		return true
	} else if len(finding2.Trace) < len(finding1.Trace) {
		return false
	}
	if finding1.weight < finding2.weight {
		return true
	} else if finding2.weight < finding1.weight {
		return false
	}
	// At this point we just need to make sure the ordering is deterministic.
	// TODO(zpavlinovic): is there a more meaningful ordering?
	return findingStrCompare(finding1, finding2)
}

// findingStrCompare compares string representation of findings pointwise by fields.
func findingStrCompare(finding1, finding2 Finding) bool {
	symCmp := strings.Compare(finding1.Symbol, finding2.Symbol)
	if symCmp == -1 {
		return true
	} else if symCmp == 1 {
		return false
	}

	typeStr1 := fmt.Sprintf("%v", finding1.Type)
	typeStr2 := fmt.Sprintf("%v", finding2.Type)
	typeCmp := strings.Compare(typeStr1, typeStr2)
	if typeCmp == -1 {
		return true
	} else if typeCmp == 1 {
		return false
	}

	posStr1 := fmt.Sprintf("%v", finding1.Position)
	posStr2 := fmt.Sprintf("%v", finding2.Position)
	posCmp := strings.Compare(posStr1, posStr2)
	if posCmp == -1 {
		return true
	} else if posCmp == 1 {
		return false
	}

	traceStr1 := fmt.Sprintf("%v", finding1.Trace)
	traceStr2 := fmt.Sprintf("%v", finding2.Trace)
	traceCmp := strings.Compare(traceStr1, traceStr2)
	if traceCmp == 1 {
		return false
	}

	return true
}
