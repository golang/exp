// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

const methods = `
//!+input
package methods

type List[E any] []E

func (l List[_]) Len() int {
	return len(l)
}

func (l *List[E]) Append(v E) {
	*l = append(*l, v)
}

type Pair[L, R comparable] struct {
	left  L
	right R
}

func (p Pair[L, _]) Left() L {
	return p.left
}

func (p Pair[_, R]) Right() R {
	return p.right
}

func (p Pair[L, R]) Equal(other Pair[L, R]) bool {
	return p.Left() == other.Left() && p.Right() == other.Right()
}
//!-input
`

// !+describe
func Describe(fset *token.FileSet, file *ast.File) error {
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	pkg, err := conf.Check("pair", fset, []*ast.File{file}, info)
	if err != nil {
		return err
	}
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		typ := obj.Type().(*types.Named)

		fmt.Printf("type %s has %d methods\n", name, typ.NumMethods())
		for i := 0; i < typ.NumMethods(); i++ {
			m := typ.Method(i)
			sig := m.Type().(*types.Signature)
			recv := sig.Recv().Type()
			fmt.Printf("  %s has %d receiver type parameters\n", m.Name(), sig.RecvTypeParams().Len())
			fmt.Printf("  ...and receiver type %+v\n", recv)
		}
	}
	// }

	// func foo() {}
	for _, decl := range file.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fmt.Printf("Declaration of %q has receiver node type %T\n", fdecl.Name, fdecl.Recv.List[0].Type)
		declObj := info.Defs[fdecl.Name]
		recvIdent := receiverTypeName(fdecl.Recv.List[0].Type)
		typ := pkg.Scope().Lookup(recvIdent.Name).Type().(*types.Named)
		// ptr := types.NewPointer(typ)
		name := fdecl.Name.Name
		methodObj, _, _ := types.LookupFieldOrMethod(typ, false, pkg, name)
		if declObj == methodObj {
			fmt.Printf("  info.Uses[%s] == types.LookupFieldOrMethod(%s, ..., %q)\n", fdecl.Name, typ, name)
		}
	}
	return nil
}

func receiverTypeName(e ast.Expr) *ast.Ident {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	switch e := e.(type) {
	case *ast.IndexExpr:
		return e.X.(*ast.Ident)
	case *ast.IndexListExpr:
		return e.X.(*ast.Ident)
	}
	panic("unexpected receiver node type")
}

//!-describe

/*
//!+output
> go run golang.org/x/tools/internal/typeparams/example/methoddecls
Pair has 2 methods
  Left has 2 receiver type parameters
  ...and receiver type pair.Pair[L, _]
  Right has 2 receiver type parameters
  ...and receiver type pair.Pair[_, R]
Declaration of "Left" has receiver node type *ast.IndexListExpr
  info.Uses[Left] == types.LookupFieldOrMethod(pair.Pair[L, R any], ..., "Left")
Declaration of "Right" has receiver node type *ast.IndexListExpr
  info.Uses[Right] == types.LookupFieldOrMethod(pair.Pair[L, R any], ..., "Right")
//!-output
*/

func main() {
	// Parse one file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "methods.go", methods, 0)
	if err != nil {
		log.Fatal(err)
	}
	if err := Describe(fset, f); err != nil {
		log.Fatal(err)
	}
}
