// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.18
// +build go1.18

package typeparams_test

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// TestAPIConsistency verifies that exported APIs match at Go 1.17 and Go
// 1.18+.
//
// It relies on the convention that the names of type aliases in the typeparams
// package match the names of the types they are aliasing.
//
// This test could be made more precise.
func TestAPIConsistency(t *testing.T) {
	api118 := getAPI(buildPackage(t, true))
	api117 := getAPI(buildPackage(t, false))

	for name, api := range api117 {
		if api != api118[name] {
			t.Errorf("%q: got %s at 1.17, but %s at 1.18+", name, api, api118[name])
		}
		delete(api118, name)
	}
	for name, api := range api118 {
		if api != api117[name] {
			t.Errorf("%q: got %s at 1.18+, but %s at 1.17", name, api, api117[name])
		}
	}
}

func getAPI(pkg *types.Package) map[string]string {
	api := make(map[string]string)
	for _, name := range pkg.Scope().Names() {
		if !token.IsExported(name) {
			continue
		}
		api[name] = name
		obj := pkg.Scope().Lookup(name)
		if f, ok := obj.(*types.Func); ok {
			api[name] = formatSignature(f.Type().(*types.Signature))
		}
		typ := pkg.Scope().Lookup(name).Type()
		// Consider method sets of pointer and non-pointer receivers.
		msets := map[string]*types.MethodSet{
			name:       types.NewMethodSet(typ),
			"*" + name: types.NewMethodSet(types.NewPointer(typ)),
		}
		for name, mset := range msets {
			for i := 0; i < mset.Len(); i++ {
				f := mset.At(i).Obj().(*types.Func)
				mname := f.Name()
				if token.IsExported(mname) {
					api[name+"."+mname] = formatSignature(f.Type().(*types.Signature))
				}
			}
		}
	}
	return api
}

func formatSignature(sig *types.Signature) string {
	var b bytes.Buffer
	b.WriteString("func")
	writeTuple(&b, sig.Params())
	writeTuple(&b, sig.Results())
	return b.String()
}

func writeTuple(buf *bytes.Buffer, t *types.Tuple) {
	buf.WriteRune('(')

	// The API at Go 1.18 uses aliases for types in go/types. These types are
	// _actually_ in the go/types package, and therefore would be formatted as
	// e.g. *types.TypeParam, which would not match *typeparams.TypeParam --
	// go/types does not track aliases. As we use the same name for all aliases,
	// we can make the formatted signatures match by dropping the package
	// qualifier.
	qf := func(*types.Package) string { return "" }

	for i := 0; i < t.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(types.TypeString(t.At(i).Type(), qf))
	}
	buf.WriteRune(')')
}

func buildPackage(t *testing.T, go118 bool) *types.Package {
	ctxt := build.Default
	if !go118 {
		for i, tag := range ctxt.ReleaseTags {
			if tag == "go1.18" {
				ctxt.ReleaseTags = ctxt.ReleaseTags[:i]
				break
			}
		}
	}
	bpkg, err := ctxt.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	return typeCheck(t, bpkg.GoFiles)
}

func typeCheck(t *testing.T, filenames []string) *types.Package {
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range filenames {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	conf := types.Config{
		Importer: importer.Default(),
	}
	pkg, err := conf.Check("", fset, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}
