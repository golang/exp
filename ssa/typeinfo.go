package ssa

// This file defines utilities for querying the results of typechecker:
// types of expressions, values of constant expressions, referents of identifiers.

import (
	"code.google.com/p/go.exp/go/types"
	"fmt"
	"go/ast"
)

// TypeInfo contains information provided by the type checker about
// the abstract syntax for a single package.
type TypeInfo struct {
	types     map[ast.Expr]types.Type     // inferred types of expressions
	constants map[ast.Expr]*Literal       // values of constant expressions
	idents    map[*ast.Ident]types.Object // canonical type objects for named entities
}

// TypeOf returns the type of expression e.
// Precondition: e belongs to the package's ASTs.
func (info *TypeInfo) TypeOf(e ast.Expr) types.Type {
	// For Ident, b.types may be more specific than
	// b.obj(id.(*ast.Ident)).GetType(),
	// e.g. in the case of typeswitch.
	if t, ok := info.types[e]; ok {
		return t
	}
	// The typechecker doesn't notify us of all Idents,
	// e.g. s.Key and s.Value in a RangeStmt.
	// So we have this fallback.
	// TODO(gri): This is a typechecker bug.  When fixed,
	// eliminate this case and panic.
	if id, ok := e.(*ast.Ident); ok {
		return info.ObjectOf(id).GetType()
	}
	panic("no type for expression")
}

// ValueOf returns the value of expression e if it is a constant,
// nil otherwise.
//
func (info *TypeInfo) ValueOf(e ast.Expr) *Literal {
	return info.constants[e]
}

// ObjectOf returns the typechecker object denoted by the specified id.
// Precondition: id belongs to the package's ASTs.
//
func (info *TypeInfo) ObjectOf(id *ast.Ident) types.Object {
	if obj, ok := info.idents[id]; ok {
		return obj
	}
	panic(fmt.Sprintf("no types.Object for ast.Ident %s @ %p", id.Name, id))
}

// IsType returns true iff expression e denotes a type.
// Precondition: e belongs to the package's ASTs.
//
func (info *TypeInfo) IsType(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.SelectorExpr: // pkg.Type
		if obj := info.isPackageRef(e); obj != nil {
			return objKind(obj) == ast.Typ
		}
	case *ast.StarExpr: // *T
		return info.IsType(e.X)
	case *ast.Ident:
		return objKind(info.ObjectOf(e)) == ast.Typ
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		return true
	case *ast.ParenExpr:
		return info.IsType(e.X)
	}
	return false
}

// isPackageRef returns the identity of the object if sel is a
// package-qualified reference to a named const, var, func or type.
// Otherwise it returns nil.
// Precondition: sel belongs to the package's ASTs.
//
func (info *TypeInfo) isPackageRef(sel *ast.SelectorExpr) types.Object {
	if id, ok := sel.X.(*ast.Ident); ok {
		if obj := info.ObjectOf(id); objKind(obj) == ast.Pkg {
			return obj.(*types.Package).Scope.Lookup(sel.Sel.Name)
		}
	}
	return nil
}
