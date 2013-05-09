// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "go/ast"

// All types implement the Type interface.
type Type interface {
	// TODO(gri) define a Kind accessor and TypeKind type?

	// Len returns an array type's length.
	// It panics if the type is not an array.
	Len() int64

	// Key returns a map type's key type.
	// It panics if the type is not a map.
	Key() Type

	// Elt returns a type's element type.
	// It panics if the type is not an array, slice, pointer, map, or channel.
	Elt() Type

	// Dir returns a channel type's direction.
	// It panics if the type is not a channel.
	Dir() ast.ChanDir

	// NumField returns a struct type's field count.
	// It panics if the type is not a struct.
	NumFields() int

	// Field returns the i'th field of a struct.
	// It panics if the type is not a struct or if the field index i is out of bounds.
	Field(i int) *Field

	// Tag returns the i'th field tag of a struct.
	// It panics if the type is not a struct or if the field index i is out of bounds.
	Tag(i int) string

	// NumMethods returns the number of methods in the type's method set.
	NumMethods() int

	// Method returns the i'th method in the type's method set.
	// It panics if the method index i is out of bounds.
	Method(i int) *Func

	// TODO(gri) Do we expose all methods of all types?

	// String returns a string representation of a type.
	String() string
}

// aType provides default implementations for a Type's methods.
type aType struct{}

func (aType) Len() int64       { panic("types: Len of non-array type") }
func (aType) Key() Type        { panic("types: Key of non-map type") }
func (aType) Elt() Type        { panic("types: Elt of invalid type") }
func (aType) Dir() ast.ChanDir { panic("types: Dir of non-chan type") }
func (aType) NumFields() int   { panic("types: NumFields of non-struct type") }
func (aType) Field(int) *Field { panic("types: Field of non-struct type") }
func (aType) Tag(int) string   { panic("types: Tag of non-struct type") }
func (aType) NumMethods() int  { panic("types: NumMethods of non-interface or unnamed type") }
func (aType) Method(int) *Func { panic("types: Method of type with no methods") }
func (aType) String() string   { panic("types: String of invalid type") }

// BasicKind describes the kind of basic type.
type BasicKind int

const (
	Invalid BasicKind = iota // type is invalid

	// predeclared types
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	String
	UnsafePointer

	// types for untyped values
	UntypedBool
	UntypedInt
	UntypedRune
	UntypedFloat
	UntypedComplex
	UntypedString
	UntypedNil

	// aliases
	Byte = Uint8
	Rune = Int32
)

// BasicInfo is a set of flags describing properties of a basic type.
type BasicInfo int

// Properties of basic types.
const (
	IsBoolean BasicInfo = 1 << iota
	IsInteger
	IsUnsigned
	IsFloat
	IsComplex
	IsString
	IsUntyped

	IsOrdered   = IsInteger | IsFloat | IsString
	IsNumeric   = IsInteger | IsFloat | IsComplex
	IsConstType = IsBoolean | IsNumeric | IsString
)

// A Basic represents a basic type.
type Basic struct {
	aType
	kind BasicKind
	info BasicInfo
	size int64 // use DefaultSizeof to get size
	name string
}

func (b *Basic) Kind() BasicKind { return b.kind }
func (b *Basic) Info() BasicInfo { return b.info }
func (b *Basic) Name() string    { return b.name }

// An Array represents an array type [Len]Elt.
type Array struct {
	aType
	len int64
	elt Type
}

func NewArray(elt Type, len int64) *Array { return &Array{aType{}, len, elt} }
func (a *Array) Len() int64               { return a.len }
func (a *Array) Elt() Type                { return a.elt }

// A Slice represents a slice type []Elt.
type Slice struct {
	aType
	elt Type
}

func NewSlice(elt Type) *Slice { return &Slice{aType{}, elt} }
func (s *Slice) Elt() Type     { return s.elt }

// A QualifiedName is a name qualified with the package that declared the name.
// Note: Pkg may be a fake package (no name, no scope) because the GC compiler's
// export information doesn't provide full information in some cases.
// TODO(gri): Should change Pkg to PkgPath since it's the only thing we care about.
type QualifiedName struct {
	Pkg  *Package // nil only for predeclared error.Error (exported)
	Name string   // unqualified type name for anonymous fields
}

// IsSame reports whether p and q are the same.
func (p QualifiedName) IsSame(q QualifiedName) bool {
	// spec:
	// "Two identifiers are different if they are spelled differently,
	// or if they appear in different packages and are not exported.
	// Otherwise, they are the same."
	if p.Name != q.Name {
		return false
	}
	// p.Name == q.Name
	return ast.IsExported(p.Name) || p.Pkg.path == q.Pkg.path
}

// A Field represents a field of a struct.
type Field struct {
	QualifiedName
	Type        Type
	IsAnonymous bool
}

// A Struct represents a struct type struct{...}.
type Struct struct {
	aType
	fields  []*Field
	tags    []string // field tags; nil of there are no tags
	offsets []int64  // field offsets in bytes, lazily computed
}

func (s *Struct) NumFields() int     { return len(s.fields) }
func (s *Struct) Field(i int) *Field { return s.fields[i] }
func (s *Struct) Tag(i int) string {
	if i < len(s.tags) {
		return s.tags[i]
	}
	return ""
}
func (s *Struct) ForEachField(f func(*Field)) {
	for _, fld := range s.fields {
		f(fld)
	}
}

func (s *Struct) fieldIndex(name QualifiedName) int {
	for i, f := range s.fields {
		if f.QualifiedName.IsSame(name) {
			return i
		}
	}
	return -1
}

// A Pointer represents a pointer type *Base.
type Pointer struct {
	aType
	base Type
}

func NewPointer(elt Type) *Pointer { return &Pointer{aType{}, elt} }
func (p *Pointer) Elt() Type       { return p.base }

// A Result represents a (multi-value) function call result.
type Result struct {
	aType
	values []*Var // Signature.Results of the function called
}

func NewResult(x ...*Var) *Result {
	return &Result{aType{}, x}
}
func (r *Result) NumValues() int   { return len(r.values) }
func (r *Result) Value(i int) *Var { return r.values[i] }
func (r *Result) ForEachValue(f func(*Var)) {
	for _, val := range r.values {
		f(val)
	}
}

// A Signature represents a user-defined function type func(...) (...).
type Signature struct {
	aType
	recv       *Var   // nil if not a method
	params     []*Var // (incoming) parameters from left to right; or nil
	results    []*Var // (outgoing) results from left to right; or nil
	isVariadic bool   // true if the last parameter's type is of the form ...T
}

func NewSignature(recv *Var, params, results []*Var, isVariadic bool) *Signature {
	return &Signature{aType{}, recv, params, results, isVariadic}
}

func (s *Signature) Recv() *Var       { return s.recv }
func (s *Signature) IsVariadic() bool { return s.isVariadic }

func (s *Signature) NumParams() int   { return len(s.params) }
func (s *Signature) Param(i int) *Var { return s.params[i] }
func (s *Signature) ForEachParam(f func(*Var)) {
	for _, par := range s.params {
		f(par)
	}
}

func (s *Signature) NumResults() int   { return len(s.results) }
func (s *Signature) Result(i int) *Var { return s.results[i] }
func (s *Signature) ForEachResult(f func(*Var)) {
	for _, res := range s.results {
		f(res)
	}
}

// builtinId is an id of a builtin function.
type builtinId int

// Predeclared builtin functions.
const (
	// Universe scope
	_Append builtinId = iota
	_Cap
	_Close
	_Complex
	_Copy
	_Delete
	_Imag
	_Len
	_Make
	_New
	_Panic
	_Print
	_Println
	_Real
	_Recover

	// Unsafe package
	_Alignof
	_Offsetof
	_Sizeof

	// Testing support
	_Assert
	_Trace
)

// A builtin represents the type of a built-in function.
type builtin struct {
	aType
	id          builtinId
	name        string
	nargs       int // number of arguments (minimum if variadic)
	isVariadic  bool
	isStatement bool // true if the built-in is valid as an expression statement
}

// An Interface represents an interface type interface{...}.
type Interface struct {
	aType
	methods ObjSet
}

func (t *Interface) NumMethods() int { return len(t.methods.entries) }
func (t *Interface) Method(i int) *Func {
	return t.methods.entries[i].(*Func)
}
func (t *Interface) IsEmpty() bool { return len(t.methods.entries) == 0 }
func (t *Interface) ForEachMethod(fn func(*Func)) {
	for _, obj := range t.methods.entries {
		fn(obj.(*Func))
	}
}

// A Map represents a map type map[key]elt.
type Map struct {
	aType
	key, elt Type
}

func (m *Map) Key() Type { return m.key }
func (m *Map) Elt() Type { return m.elt }

// A Chan represents a channel type chan elt, <-chan elt, or chan<-elt.
type Chan struct {
	aType
	dir ast.ChanDir
	elt Type
}

func (c *Chan) Dir() ast.ChanDir { return c.dir }
func (c *Chan) Elt() Type        { return c.elt }

// A Named represents a named type as declared in a type declaration.
type Named struct {
	aType
	obj        *TypeName // corresponding declared object
	underlying Type      // nil if not fully declared yet; never a *Named
	methods    ObjSet
}

func NewNamed(obj *TypeName, underlying Type, methods ObjSet) *Named {
	typ := &Named{aType{}, obj, underlying, methods}
	if obj.typ == nil {
		obj.typ = typ
	}
	return typ
}

func (t *Named) Obj() *TypeName   { return t.obj }
func (t *Named) Underlying() Type { return t.underlying }

// TODO(gri) Define MethodSet type and move these accessors there.
func (t *Named) NumMethods() int { return len(t.methods.entries) }
func (t *Named) Method(i int) *Func {
	return t.methods.entries[i].(*Func)
}
func (t *Named) IsEmpty() bool { return len(t.methods.entries) == 0 }
func (t *Named) ForEachMethod(fn func(*Func)) {
	for _, obj := range t.methods.entries {
		fn(obj.(*Func))
	}
}
