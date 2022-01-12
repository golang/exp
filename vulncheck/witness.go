package vulncheck

import (
	"container/list"
	"sync"
)

// ImportChain is sequence of import paths starting with
// a client package and ending with a package with some
// known vulnerabilities.
type ImportChain []*PkgNode

// ImportChains performs a BFS search of res.RequireGraph for imports of vulnerable
// packages. Search is performed for each vulnerable package in res.Vulns. The search
// starts at a vulnerable package and goes up until reaching an entry package in
// res.ImportGraph.Entries, hence producing an import chain. During the search, a
// package is visited only once to avoid analyzing every possible import chain.
// Hence, not all possible vulnerable import chains are reported.
//
// Note that the resulting map produces an import chain for each Vuln. Thus, a Vuln
// with the same PkgPath will have the same list of identified import chains.
//
// The reported import chains are ordered by how seemingly easy is to understand
// them. Shorter import chains appear earlier in the returned slices.
func ImportChains(res *Result) map[*Vuln][]ImportChain {
	// Group vulns per package.
	vPerPkg := make(map[int][]*Vuln)
	for _, v := range res.Vulns {
		vPerPkg[v.ImportSink] = append(vPerPkg[v.ImportSink], v)
	}

	// Collect chains in parallel for every package path.
	var wg sync.WaitGroup
	var mu sync.Mutex
	chains := make(map[*Vuln][]ImportChain)
	for pkgID, vulns := range vPerPkg {
		pID := pkgID
		vs := vulns
		wg.Add(1)
		go func() {
			pChains := importChains(pID, res)
			mu.Lock()
			for _, v := range vs {
				chains[v] = pChains
			}
			mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	return chains
}

// importChains finds representative chains of package imports
// leading to vulnerable package identified with vulnSinkID.
func importChains(vulnSinkID int, res *Result) []ImportChain {
	if vulnSinkID == 0 {
		return nil
	}

	// Entry packages, needed for finalizing chains.
	entries := make(map[int]bool)
	for _, e := range res.Imports.Entries {
		entries[e] = true
	}

	var chains []ImportChain
	seen := make(map[int]bool)

	queue := list.New()
	queue.PushBack(&importChain{pkg: res.Imports.Packages[vulnSinkID]})
	for queue.Len() > 0 {
		front := queue.Front()
		c := front.Value.(*importChain)
		queue.Remove(front)

		pkg := c.pkg
		if seen[pkg.ID] {
			continue
		}
		seen[pkg.ID] = true

		for _, impBy := range pkg.ImportedBy {
			imp := res.Imports.Packages[impBy]
			newC := &importChain{pkg: imp, child: c}
			// If the next package is an entry, we have
			// a chain to report.
			if entries[imp.ID] {
				chains = append(chains, newC.ImportChain())
			}
			queue.PushBack(newC)
		}
	}
	return chains
}

// importChain models an chain of package imports.
type importChain struct {
	pkg   *PkgNode
	child *importChain
}

// ImportChain converts importChain to ImportChain type.
func (r *importChain) ImportChain() ImportChain {
	if r == nil {
		return nil
	}
	return append([]*PkgNode{r.pkg}, r.child.ImportChain()...)
}

// CallStack models a trace of function calls starting
// with a client function or method and ending with a
// call to a vulnerable symbol.
type CallStack []StackEntry

// StackEntry models an element of a call stack.
type StackEntry struct {
	// Function provides information on the function whose frame is on the stack.
	Function *FuncNode

	// Call provides information on the call site inducing this stack frame.
	// nil when the frame represents an entry point of the stack.
	Call *CallSite
}
