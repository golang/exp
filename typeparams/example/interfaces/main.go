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
)

const src = `
//!+input
package p

type Numeric interface{
	~int | ~float64 // etc...
}

func Square[N Numeric](n N) N {
	return n*n
}

type Findable interface {
	comparable
}

func Find[T Findable](s []T, v T) int {
	for i, v2 := range s {
		if v2 == v {
			return i
		}
	}
	return -1
}
//!-input
`

// !+printsyntax
func PrintNumericSyntax(fset *token.FileSet, file *ast.File) {
	// node is the AST node corresponding to the declaration for "Numeric."
	node := file.Scope.Lookup("Numeric").Decl.(*ast.TypeSpec)
	// Find the embedded syntax node.
	embedded := node.Type.(*ast.InterfaceType).Methods.List[0].Type
	// Use go/ast's built-in Print function to inspect the parsed syntax.
	ast.Print(fset, embedded)
}

//!-printsyntax

/*
//!+outputsyntax
     0  *ast.BinaryExpr {
     1  .  X: *ast.UnaryExpr {
     2  .  .  OpPos: p.go:6:2
     3  .  .  Op: ~
     4  .  .  X: *ast.Ident {
     5  .  .  .  NamePos: p.go:6:3
     6  .  .  .  Name: "int"
     7  .  .  }
     8  .  }
     9  .  OpPos: p.go:6:7
    10  .  Op: |
    11  .  Y: *ast.UnaryExpr {
    12  .  .  OpPos: p.go:6:9
    13  .  .  Op: ~
    14  .  .  X: *ast.Ident {
    15  .  .  .  NamePos: p.go:6:10
    16  .  .  .  Name: "float64"
    17  .  .  }
    18  .  }
    19  }
//!-outputsyntax
*/

// !+printtypes
func PrintInterfaceTypes(fset *token.FileSet, file *ast.File) error {
	conf := types.Config{}
	pkg, err := conf.Check("hello", fset, []*ast.File{file}, nil)
	if err != nil {
		return err
	}

	PrintIface(pkg, "Numeric")
	PrintIface(pkg, "Findable")

	return nil
}

func PrintIface(pkg *types.Package, name string) {
	obj := pkg.Scope().Lookup(name)
	fmt.Println(obj)
	iface := obj.Type().Underlying().(*types.Interface)
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		fmt.Printf("\tembeded: %s\n", iface.EmbeddedType(i))
	}
	for i := 0; i < iface.NumMethods(); i++ {
		fmt.Printf("\tembeded: %s\n", iface.EmbeddedType(i))
	}
	fmt.Printf("\tIsComparable(): %t\n", iface.IsComparable())
	fmt.Printf("\tIsMethodSet(): %t\n", iface.IsMethodSet())
}

//!-printtypes

/*
//!+outputtypes
type hello.Numeric interface{~int|~float64}
        embeded: ~int|~float64
        IsComparable(): true
        IsMethodSet(): false
type hello.Findable interface{comparable}
        embeded: comparable
        IsComparable(): true
        IsMethodSet(): false
//!-outputtypes
*/

func main() {
	// Parse one file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== PrintNumericSyntax ==")
	PrintNumericSyntax(fset, f)
	fmt.Println("=== PrintInterfaceTypes ==")
	if err := PrintInterfaceTypes(fset, f); err != nil {
		log.Fatal(err)
	}
}
