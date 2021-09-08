// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package audit

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/types/typeutil"

	"golang.org/x/tools/go/ssa"

	"golang.org/x/vulndb/osv"
)

// instrPosition gives the position of `instr`. Returns empty token.Position
// if no file information on `instr` is available.
func instrPosition(instr ssa.Instruction) *token.Position {
	pos := instr.Parent().Prog.Fset.Position(instr.Pos())
	return &pos
}

// valPosition gives the position of `v` inside of `f`. Assumes `v` is used in
// `f`. Returns empty token.Position if no file information on `f` is available.
func valPosition(v ssa.Value, f *ssa.Function) *token.Position {
	pos := f.Prog.Fset.Position(v.Pos())
	return &pos
}

// funcPosition gives the position of `f`. Returns empty token.Position
// if no file information on `f` is available.
func funcPosition(f *ssa.Function) *token.Position {
	pos := f.Prog.Fset.Position(f.Pos())
	return &pos
}

// siteCallees computes a set of callees for call site `call` given program `callgraph`.
func siteCallees(call ssa.CallInstruction, callgraph *callgraph.Graph) []*ssa.Function {
	var matches []*ssa.Function

	node := callgraph.Nodes[call.Parent()]
	if node == nil {
		return nil
	}

	for _, edge := range node.Out {
		callee := edge.Callee.Func
		// Some callgraph analyses, such as CHA, might return synthetic (interface)
		// methods as well as the concrete methods. Skip such synthetic functions.
		if edge.Site == call {
			matches = append(matches, callee)
		}
	}
	return matches
}

func callName(call ssa.CallInstruction) string {
	if !call.Common().IsInvoke() {
		return fmt.Sprintf("%s.%s", call.Parent().Pkg.Pkg.Path(), call.Common().Value.Name())
	}
	buf := new(bytes.Buffer)
	types.WriteType(buf, call.Common().Value.Type(), nil)
	return fmt.Sprintf("%s.%s", buf, call.Common().Method.Name())
}

func unresolved(call ssa.CallInstruction) bool {
	if call == nil {
		return false
	}
	return call.Common().StaticCallee() == nil
}

// pkgsProgram returns the single common program to which all pkgs belong, if such.
// Otherwise, returns nil.
func pkgsProgram(pkgs []*ssa.Package) *ssa.Program {
	var prog *ssa.Program
	for _, pkg := range pkgs {
		if prog == nil {
			prog = pkg.Prog
		} else if prog != pkg.Prog {
			return nil
		}
	}
	return prog
}

// globalUses returns a list of global uses by an instruction.
// Global function callees are disregarded as they are preferred as call uses.
func globalUses(instr ssa.Instruction) []*ssa.Value {
	ops := instr.Operands(nil)
	if _, ok := instr.(ssa.CallInstruction); ok {
		ops = ops[1:]
	}

	var glbs []*ssa.Value
	for _, o := range ops {
		if _, ok := (*o).(*ssa.Global); ok {
			glbs = append(glbs, o)
		}
	}
	return glbs
}

// Computes function name consistent with the function namings used in vulnerability
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

// Returns the path of a package `f` belongs to. Covers both
// the case when `f` is an anonymous and a synthetic function.
func pkgPath(f *ssa.Function) string {
	// Handle all user defined functions.
	if p := f.Package(); p != nil && p.Pkg != nil {
		return p.Pkg.Path()
	}
	// Cover synthetic functions as well.
	if o := f.Object(); o != nil && o.Pkg() != nil {
		return o.Pkg().Path()
	}
	// Not reachable in principle.
	return ""
}

// serialize transforms []*osv.Entry into []osv.Entry as to
// allow serialization of Finding.
func serialize(vulns []*osv.Entry) []osv.Entry {
	var vs []osv.Entry
	for _, v := range vulns {
		vs = append(vs, *v)
	}
	return vs
}
