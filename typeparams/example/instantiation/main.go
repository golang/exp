// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.18

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
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

func Equal[L, R comparable](x, y Pair[L, R]) bool {
	return x.left == y.left && x.right == y.right
}

var X Pair[int, string]
var Y Pair[string, int]

var E = Equal[int, string]
//!-input
`

// !+check
func CheckInstances(fset *token.FileSet, file *ast.File) (*types.Package, error) {
	conf := types.Config{}
	info := &types.Info{
		Instances: make(map[*ast.Ident]types.Instance),
	}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	for id, inst := range info.Instances {
		posn := fset.Position(id.Pos())
		fmt.Printf("%s: %s instantiated with %s: %s\n", posn, id.Name, FormatTypeList(inst.TypeArgs), inst.Type)
	}
	return pkg, err
}

//!-check

/*
//!+checkoutput
hello.go:21:9: Equal instantiated with [int, string]: func(x p.Pair[int, string], y p.Pair[int, string]) bool
hello.go:10:9: Pair instantiated with [L, _]: p.Pair[L, _]
hello.go:14:34: Pair instantiated with [L, R]: p.Pair[L, R]
hello.go:18:7: Pair instantiated with [int, string]: p.Pair[int, string]
hello.go:19:7: Pair instantiated with [string, int]: p.Pair[string, int]

//!-checkoutput
*/

func FormatTypeList(list *types.TypeList) string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	for i := 0; i < list.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(list.At(i).String())
	}
	buf.WriteRune(']')
	return buf.String()
}

// !+instantiate
func Instantiate(pkg *types.Package) error {
	Pair := pkg.Scope().Lookup("Pair").Type()
	X := pkg.Scope().Lookup("X").Type()
	Y := pkg.Scope().Lookup("Y").Type()

	// X and Y have different types, because their type arguments are different.
	Compare("X", "Y", X, Y)

	// Create a shared context for the subsequent instantiations.
	ctxt := types.NewContext()

	// Instantiating with [int, string] yields an instance that is identical (but
	// not equal) to X.
	Int, String := types.Typ[types.Int], types.Typ[types.String]
	inst1, _ := types.Instantiate(ctxt, Pair, []types.Type{Int, String}, true)
	Compare("X", "inst1", X, inst1)

	// Instantiating again returns the same exact instance, because of the shared
	// Context.
	inst2, _ := types.Instantiate(ctxt, Pair, []types.Type{Int, String}, true)
	Compare("inst1", "inst2", inst1, inst2)

	// Instantiating with 'any' is an error, because any is not comparable.
	Any := types.Universe.Lookup("any").Type()
	_, err := types.Instantiate(ctxt, Pair, []types.Type{Int, Any}, true)
	var argErr *types.ArgumentError
	if errors.As(err, &argErr) {
		fmt.Printf("Argument %d: %v\n", argErr.Index, argErr.Err)
	}

	return nil
}

func Compare(leftName, rightName string, left, right types.Type) {
	fmt.Printf("Identical(%s, %s) : %t\n", leftName, rightName, types.Identical(left, right))
	fmt.Printf("%s == %s : %t\n\n", leftName, rightName, left == right)
}

//!-instantiate

/*
//!+instantiateoutput
Identical(p.Pair[int, string], p.Pair[string, int]) : false
p.Pair[int, string] == p.Pair[string, int] : false

Identical(p.Pair[int, string], p.Pair[int, string]) : true
p.Pair[int, string] == p.Pair[int, string] : false

Identical(p.Pair[string, int], p.Pair[int, string]) : false
p.Pair[string, int] == p.Pair[int, string] : false

Identical(p.Pair[int, string], p.Pair[int, string]) : true
p.Pair[int, string] == p.Pair[int, string] : true

Argument 1: any does not implement comparable
//!-instantiateoutput
*/

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hello.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== CheckInstances ===")
	pkg, err := CheckInstances(fset, f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Instantiate ===")
	if err := Instantiate(pkg); err != nil {
		log.Fatal(err)
	}
}
