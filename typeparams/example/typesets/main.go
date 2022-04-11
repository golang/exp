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
package complex

type A interface{ ~string|~[]byte }

type B interface{ int|string }

type C interface { ~string|~int }

type D interface{ A|B; C }
//!-input
`

// !+print
func PrintNormalTerms(pkg *types.Package) error {
	D := pkg.Scope().Lookup("D").Type()
	terms, err := typeparams.NormalTerms(D)
	if err != nil {
		return err
	}
	for i, term := range terms {
		if i > 0 {
			fmt.Print("|")
		}
		fmt.Print(term)
	}
	fmt.Println()
	return nil
}

//!-print

/*
//!+output
~string|int
//!-output
*/

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	conf := types.Config{}
	pkg, err := conf.Check("typesets", fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := PrintNormalTerms(pkg); err != nil {
		log.Fatal(err)
	}
}
