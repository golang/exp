// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// M does not use type parameters, and therefore implements the Mer interface.
func (p Pair[_, _]) M() int { return 0 }

type LeftRighter[L, R comparable] interface {
	Left() L
	Right() R
}

type Mer interface {
	M() int
}

// F and G have identical signatures "modulo renaming", H does not.
func F[P any](P) int { return 0 }
func G[Q any](Q) int { return 1 }
func H[R ~int](R) int { return 2 }
//!-input
`

// !+ordinary
func OrdinaryPredicates(pkg *types.Package) {
	var (
		Pair        = pkg.Scope().Lookup("Pair").Type()
		LeftRighter = pkg.Scope().Lookup("LeftRighter").Type()
		Mer         = pkg.Scope().Lookup("Mer").Type()
		F           = pkg.Scope().Lookup("F").Type()
		G           = pkg.Scope().Lookup("G").Type()
		H           = pkg.Scope().Lookup("H").Type()
	)

	fmt.Println("AssignableTo(Pair, LeftRighter)", types.AssignableTo(Pair, LeftRighter))
	fmt.Println("AssignableTo(Pair, Mer): ", types.AssignableTo(Pair, Mer))
	fmt.Println("Identical(F, G)", types.Identical(F, G))
	fmt.Println("Identical(F, H)", types.Identical(F, H))
}

//!-ordinary

/*
//!+ordinaryoutput
AssignableTo(Pair, LeftRighter) false
AssignableTo(Pair, Mer):  true
Identical(F, G) true
Identical(F, H) false
//!-ordinaryoutput
*/

// !+generic
func GenericPredicates(pkg *types.Package) {
	var (
		Pair        = pkg.Scope().Lookup("Pair").Type()
		LeftRighter = pkg.Scope().Lookup("LeftRighter").Type()
	)
	fmt.Println("GenericAssignableTo(Pair, LeftRighter)", typeparams.GenericAssignableTo(nil, Pair, LeftRighter))
}

//!-generic

/*
//!+genericoutput
GenericAssignableTo(Pair, LeftRighter) true
//!-genericoutput
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
	fmt.Println("=== ordinary ===")
	OrdinaryPredicates(pkg)
	fmt.Println("=== generic ===")
	GenericPredicates(pkg)
}
