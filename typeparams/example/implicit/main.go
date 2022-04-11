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

func Square[N ~int|~float64](n N) N {
	return n*n
}
//!-input
`

// !+show
func ShowImplicit(pkg *types.Package) {
	Square := pkg.Scope().Lookup("Square").Type().(*types.Signature)
	N := Square.TypeParams().At(0)
	constraint := N.Constraint().(*types.Interface)
	fmt.Println(constraint)
	fmt.Println("IsImplicit:", constraint.IsImplicit())
}

//!-show

/*
//!+output
~int|~float64
IsImplicit: true
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
	ShowImplicit(pkg)
}
