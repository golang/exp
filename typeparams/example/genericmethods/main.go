// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/exp/typeparams"
)

const src = `
//!+input
package p

type Pair[L, R any] struct {
	left  L
	right R
}

func (p Pair[L, _]) Left() L {
	return p.left
}

func (p Pair[_, R]) Right() R {
	return p.right
}

var IntPair Pair[int, int]
//!-input
`

// !+printmethods
func PrintMethods(pkg *types.Package) {
	// Look up *Named types in the package scope.
	lookup := func(name string) *types.Named {
		return pkg.Scope().Lookup(name).Type().(*types.Named)
	}

	Pair := lookup("Pair")
	IntPair := lookup("IntPair")
	PrintMethodSet("Pair", Pair)
	PrintMethodSet("Pair[int, int]", IntPair)
	LeftObj, _, _ := types.LookupFieldOrMethod(Pair, false, pkg, "Left")
	LeftRecvType := LeftObj.Type().(*types.Signature).Recv().Type()
	PrintMethodSet("Pair[L, _]", LeftRecvType)
}

func PrintMethodSet(name string, typ types.Type) {
	fmt.Println(name + ":")
	methodSet := types.NewMethodSet(typ)
	for i := 0; i < methodSet.Len(); i++ {
		method := methodSet.At(i).Obj()
		fmt.Println(method)
	}
	fmt.Println()
}

//!-printmethods

/*
//!+printoutput
Pair:
func (p.Pair[L, _]).Left() L
func (p.Pair[_, R]).Right() R

Pair[int, int]:
func (p.Pair[int, int]).Left() int
func (p.Pair[int, int]).Right() int

Pair[L, _]:
func (p.Pair[L, _]).Left() L
func (p.Pair[L, _]).Right() _

//!-printoutput
*/

// !+compareorigins
func CompareOrigins(pkg *types.Package) {
	Pair := pkg.Scope().Lookup("Pair").Type().(*types.Named)
	IntPair := pkg.Scope().Lookup("IntPair").Type().(*types.Named)
	Left, _, _ := types.LookupFieldOrMethod(Pair, false, pkg, "Left")
	LeftInt, _, _ := types.LookupFieldOrMethod(IntPair, false, pkg, "Left")

	fmt.Println("Pair.Left == Pair[int, int].Left:", Left == LeftInt)
	origin := typeparams.OriginMethod(LeftInt.(*types.Func))
	fmt.Println("Pair.Left == OriginMethod(Pair[int, int].Left):", Left == origin)
}

//!-compareorigins

/*
//!+compareoutput
Pair.Left == Pair[int, int].Left: false
Pair.Left == OriginMethod(Pair[int, int].Left): true
//!-compareoutput
*/

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	conf := types.Config{}
	pkg, err := conf.Check("p", fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== PrintMethods ===")
	PrintMethods(pkg)
	fmt.Println("=== CompareOrigins ===")
	CompareOrigins(pkg)
}
