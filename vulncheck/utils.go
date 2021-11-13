// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vulncheck

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/types/typeutil"

	"golang.org/x/tools/go/ssa"
)

// siteCallees computes a set of callees for call site `call` given program `callgraph`.
func siteCallees(call ssa.CallInstruction, callgraph *callgraph.Graph) []*ssa.Function {
	var matches []*ssa.Function

	node := callgraph.Nodes[call.Parent()]
	if node == nil {
		return nil
	}

	for _, edge := range node.Out {
		// Some callgraph analyses, such as CHA, might return synthetic (interface)
		// methods as well as the concrete methods. Skip such synthetic functions.
		if edge.Site == call {
			matches = append(matches, edge.Callee.Func)
		}
	}
	return matches
}

// dbFuncName computes a function name consistent with the namings used in vulnerability
// databases. Effectively, a qualified name of a function local to its enclosing package.
// If a receiver is a pointer, this information is not encoded in the resulting name. The
// name of anonymous functions is simply "". The function names are unique subject to the
// enclosing package, but not globally.
//
// Examples:
//   func (a A) foo (...) {...}  -> A.foo
//   func foo(...) {...}         -> foo
//   func (b *B) bar (...) {...} -> B.bar
func dbFuncName(f *ssa.Function) string {
	var typeFormat func(t types.Type) string
	typeFormat = func(t types.Type) string {
		switch tt := t.(type) {
		case *types.Pointer:
			return typeFormat(tt.Elem())
		case *types.Named:
			return tt.Obj().Name()
		default:
			return types.TypeString(t, func(p *types.Package) string { return "" })
		}
	}
	selectBound := func(f *ssa.Function) types.Type {
		// If f is a "bound" function introduced by ssa for a given type, return the type.
		// When "f" is a "bound" function, it will have 1 free variable of that type within
		// the function. This is subject to change when ssa changes.
		if len(f.FreeVars) == 1 && strings.HasPrefix(f.Synthetic, "bound ") {
			return f.FreeVars[0].Type()
		}
		return nil
	}
	selectThunk := func(f *ssa.Function) types.Type {
		// If f is a "thunk" function introduced by ssa for a given type, return the type.
		// When "f" is a "thunk" function, the first parameter will have that type within
		// the function. This is subject to change when ssa changes.
		params := f.Signature.Params() // params.Len() == 1 then params != nil.
		if strings.HasPrefix(f.Synthetic, "thunk ") && params.Len() >= 1 {
			if first := params.At(0); first != nil {
				return first.Type()
			}
		}
		return nil
	}
	var qprefix string
	if recv := f.Signature.Recv(); recv != nil {
		qprefix = typeFormat(recv.Type())
	} else if btype := selectBound(f); btype != nil {
		qprefix = typeFormat(btype)
	} else if ttype := selectThunk(f); ttype != nil {
		qprefix = typeFormat(ttype)
	}

	if qprefix == "" {
		return f.Name()
	}
	return qprefix + "." + f.Name()
}

// memberFuncs returns functions associated with the `member`:
// 1) `member` itself if `member` is a function
// 2) `member` methods if `member` is a type
// 3) empty list otherwise
func memberFuncs(member ssa.Member, prog *ssa.Program) []*ssa.Function {
	switch t := member.(type) {
	case *ssa.Type:
		methods := typeutil.IntuitiveMethodSet(t.Type(), &prog.MethodSets)
		var funcs []*ssa.Function
		for _, m := range methods {
			if f := prog.MethodValue(m); f != nil {
				funcs = append(funcs, f)
			}
		}
		return funcs
	case *ssa.Function:
		return []*ssa.Function{t}
	default:
		return nil
	}
}
