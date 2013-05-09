// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the universe and unsafe package scopes.

package types

import (
	"go/ast"
	"strings"

	"code.google.com/p/go.exp/go/exact"
)

var (
	Universe     *Scope
	Unsafe       *Package
	universeIota *Const
)

// Predeclared types, indexed by BasicKind.
var Typ = [...]*Basic{
	Invalid: {aType{}, Invalid, 0, 0, "invalid type"},

	Bool:          {aType{}, Bool, IsBoolean, 1, "bool"},
	Int:           {aType{}, Int, IsInteger, 0, "int"},
	Int8:          {aType{}, Int8, IsInteger, 1, "int8"},
	Int16:         {aType{}, Int16, IsInteger, 2, "int16"},
	Int32:         {aType{}, Int32, IsInteger, 4, "int32"},
	Int64:         {aType{}, Int64, IsInteger, 8, "int64"},
	Uint:          {aType{}, Uint, IsInteger | IsUnsigned, 0, "uint"},
	Uint8:         {aType{}, Uint8, IsInteger | IsUnsigned, 1, "uint8"},
	Uint16:        {aType{}, Uint16, IsInteger | IsUnsigned, 2, "uint16"},
	Uint32:        {aType{}, Uint32, IsInteger | IsUnsigned, 4, "uint32"},
	Uint64:        {aType{}, Uint64, IsInteger | IsUnsigned, 8, "uint64"},
	Uintptr:       {aType{}, Uintptr, IsInteger | IsUnsigned, 0, "uintptr"},
	Float32:       {aType{}, Float32, IsFloat, 4, "float32"},
	Float64:       {aType{}, Float64, IsFloat, 8, "float64"},
	Complex64:     {aType{}, Complex64, IsComplex, 8, "complex64"},
	Complex128:    {aType{}, Complex128, IsComplex, 16, "complex128"},
	String:        {aType{}, String, IsString, 0, "string"},
	UnsafePointer: {aType{}, UnsafePointer, 0, 0, "Pointer"},

	UntypedBool:    {aType{}, UntypedBool, IsBoolean | IsUntyped, 0, "untyped boolean"},
	UntypedInt:     {aType{}, UntypedInt, IsInteger | IsUntyped, 0, "untyped integer"},
	UntypedRune:    {aType{}, UntypedRune, IsInteger | IsUntyped, 0, "untyped rune"},
	UntypedFloat:   {aType{}, UntypedFloat, IsFloat | IsUntyped, 0, "untyped float"},
	UntypedComplex: {aType{}, UntypedComplex, IsComplex | IsUntyped, 0, "untyped complex"},
	UntypedString:  {aType{}, UntypedString, IsString | IsUntyped, 0, "untyped string"},
	UntypedNil:     {aType{}, UntypedNil, IsUntyped, 0, "untyped nil"},
}

var aliases = [...]*Basic{
	{aType{}, Byte, IsInteger | IsUnsigned, 1, "byte"},
	{aType{}, Rune, IsInteger, 4, "rune"},
}

var predeclaredConstants = [...]*Const{
	{name: "true", typ: Typ[UntypedBool], val: exact.MakeBool(true)},
	{name: "false", typ: Typ[UntypedBool], val: exact.MakeBool(false)},
	{name: "iota", typ: Typ[UntypedInt], val: exact.MakeInt64(0)},
	{name: "nil", typ: Typ[UntypedNil], val: exact.MakeNil()},
}

var predeclaredFunctions = [...]*builtin{
	{aType{}, _Append, "append", 1, true, false},
	{aType{}, _Cap, "cap", 1, false, false},
	{aType{}, _Close, "close", 1, false, true},
	{aType{}, _Complex, "complex", 2, false, false},
	{aType{}, _Copy, "copy", 2, false, true},
	{aType{}, _Delete, "delete", 2, false, true},
	{aType{}, _Imag, "imag", 1, false, false},
	{aType{}, _Len, "len", 1, false, false},
	{aType{}, _Make, "make", 1, true, false},
	{aType{}, _New, "new", 1, false, false},
	{aType{}, _Panic, "panic", 1, false, true},
	{aType{}, _Print, "print", 0, true, true},
	{aType{}, _Println, "println", 0, true, true},
	{aType{}, _Real, "real", 1, false, false},
	{aType{}, _Recover, "recover", 0, false, true},

	{aType{}, _Alignof, "Alignof", 1, false, false},
	{aType{}, _Offsetof, "Offsetof", 1, false, false},
	{aType{}, _Sizeof, "Sizeof", 1, false, false},
}

func init() {
	Universe = new(Scope)
	Unsafe = &Package{name: "unsafe", scope: new(Scope)}

	// predeclared types
	for _, t := range Typ {
		def(&TypeName{name: t.name, typ: t})
	}
	for _, t := range aliases {
		def(&TypeName{name: t.name, typ: t})
	}

	// error type
	{
		// Error has a nil package in its qualified name since it is in no package
		var methods ObjSet
		sig := &Signature{results: []*Var{{name: "", typ: Typ[String]}}}
		methods.Insert(&Func{nil, "Error", sig, nil})
		def(&TypeName{name: "error", typ: &Named{underlying: &Interface{methods: methods}}})
	}

	for _, c := range predeclaredConstants {
		def(c)
	}

	for _, f := range predeclaredFunctions {
		def(&Func{name: f.name, typ: f})
	}

	universeIota = Universe.Lookup("iota").(*Const)
}

// Objects with names containing blanks are internal and not entered into
// a scope. Objects with exported names are inserted in the unsafe package
// scope; other objects are inserted in the universe scope.
//
func def(obj Object) {
	name := obj.Name()
	if strings.Index(name, " ") >= 0 {
		return // nothing to do
	}
	// fix Obj link for named types
	if typ, ok := obj.Type().(*Named); ok {
		typ.obj = obj.(*TypeName)
	}
	// exported identifiers go into package unsafe
	scope := Universe
	if ast.IsExported(name) {
		scope = Unsafe.scope
		// set Pkg field
		switch obj := obj.(type) {
		case *TypeName:
			obj.pkg = Unsafe
		case *Func:
			obj.pkg = Unsafe
		default:
			unreachable()
		}
	}
	if scope.Insert(obj) != nil {
		panic("internal error: double declaration")
	}
}
