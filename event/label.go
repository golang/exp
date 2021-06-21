// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

// Label is a named value.
type Label struct {
	Name string

	packed  uint64
	untyped interface{}
}

// stringptr is used in untyped when the Value is a string
type stringptr unsafe.Pointer

// bytesptr is used in untyped when the Value is a byte slice
type bytesptr unsafe.Pointer

// int64Kind is used in untyped when the Value is a signed integer
type int64Kind struct{}

// uint64Kind is used in untyped when the Value is an unsigned integer
type uint64Kind struct{}

// float64Kind is used in untyped when the Value is a floating point number
type float64Kind struct{}

// boolKind is used in untyped when the Value is a boolean
type boolKind struct{}

// durationKind is used in untyped when the Value is a time.Duration
type durationKind struct{}

// HasValue returns true if the value is set to any type.
func (l Label) HasValue() bool { return l.untyped != nil }

// Equal reports whether two labels are equal.
func (l Label) Equal(l2 Label) bool {
	if l.Name != l2.Name {
		return false
	}
	if !l.HasValue() {
		return !l2.HasValue()
	}
	if !l2.HasValue() {
		return false
	}
	switch {
	case l.IsString():
		return l2.IsString() && l.String() == l2.String()
	case l.IsInt64():
		return l2.IsInt64() && l.packed == l2.packed
	case l.IsUint64():
		return l2.IsUint64() && l.packed == l2.packed
	case l.IsFloat64():
		return l2.IsFloat64() && l.Float64() == l2.Float64()
	case l.IsBool():
		return l2.IsBool() && l.packed == l2.packed
	case l.IsDuration():
		return l2.IsDuration() && l.packed == l2.packed
	default:
		return l.untyped == l2.untyped
	}
}

// Value returns a Label for the supplied value.
func Value(name string, value interface{}) Label {
	return Label{Name: name, untyped: value}
}

// Interface returns the value.
// This will never panic, things that were not set using SetInterface will be
// unpacked and returned anyway.
func (v Label) Interface() interface{} {
	switch {
	case v.IsString():
		return v.String()
	case v.IsInt64():
		return v.Int64()
	case v.IsUint64():
		return v.Uint64()
	case v.IsFloat64():
		return v.Float64()
	case v.IsBool():
		return v.Bool()
	case v.IsDuration():
		return v.Duration()
	default:
		return v.untyped
	}
}

// String returns a new Value for a string.
func String(name string, s string) Label {
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return Label{Name: name, packed: uint64(hdr.Len), untyped: stringptr(hdr.Data)}
}

// String returns the value as a string.
// This does not panic if v's Kind is not String, instead, it returns a string
// representation of the value in all cases.
func (v Label) String() string {
	if sp, ok := v.untyped.(stringptr); ok {
		var s string
		hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
		hdr.Data = uintptr(sp)
		hdr.Len = int(v.packed)
		return s
	}
	// not a string, convert to one
	switch {
	case v.IsInt64():
		return strconv.FormatInt(v.Int64(), 10)
	case v.IsUint64():
		return strconv.FormatUint(v.Uint64(), 10)
	case v.IsFloat64():
		return strconv.FormatFloat(v.Float64(), 'g', -1, 64)
	case v.IsBool():
		if v.Bool() {
			return "true"
		} else {
			return "false"
		}
	default:
		return fmt.Sprint(v.Interface())
	}
}

// IsString returns true if the value was built with StringOf.
func (v Label) IsString() bool {
	_, ok := v.untyped.(stringptr)
	return ok
}

// Bytes returns a new Value for a string.
func Bytes(name string, data []byte) Label {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return Label{Name: name, packed: uint64(hdr.Len), untyped: bytesptr(hdr.Data)}
}

// Bytes returns the value as a bytes array.
func (v Label) Bytes() []byte {
	bp, ok := v.untyped.(bytesptr)
	if !ok {
		panic("Bytes called on non []byte value")
	}
	var buf []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	hdr.Data = uintptr(bp)
	hdr.Len = int(v.packed)
	hdr.Cap = hdr.Len
	return buf
}

// IsBytes returns true if the value was built with BytesOf.
func (v Label) IsBytes() bool {
	_, ok := v.untyped.(bytesptr)
	return ok
}

// Int64 returns a new Value for a signed integer.
func Int64(name string, u int64) Label {
	return Label{Name: name, packed: uint64(u), untyped: int64Kind{}}
}

// Int64 returns the int64 from a value that was set with SetInt64.
// It will panic for any value for which IsInt64 is not true.
func (v Label) Int64() int64 {
	if !v.IsInt64() {
		panic("Int64 called on non int64 value")
	}
	return int64(v.packed)
}

// IsInt64 returns true if the value was built with SetInt64.
func (v Label) IsInt64() bool {
	_, ok := v.untyped.(int64Kind)
	return ok
}

// Uint64 returns a new Value for an unsigned integer.
func Uint64(name string, u uint64) Label {
	return Label{Name: name, packed: u, untyped: uint64Kind{}}
}

// Uint64 returns the uint64 from a value that was set with SetUint64.
// It will panic for any value for which IsUint64 is not true.
func (v Label) Uint64() uint64 {
	if !v.IsUint64() {
		panic("Uint64 called on non uint64 value")
	}
	return v.packed
}

// IsUint64 returns true if the value was built with SetUint64.
func (v Label) IsUint64() bool {
	_, ok := v.untyped.(uint64Kind)
	return ok
}

// Float64 returns a new Value for a floating point number.
func Float64(name string, f float64) Label {
	return Label{Name: name, packed: math.Float64bits(f), untyped: float64Kind{}}
}

// Float64 returns the float64 from a value that was set with SetFloat64.
// It will panic for any value for which IsFloat64 is not true.
func (v Label) Float64() float64 {
	if !v.IsFloat64() {
		panic("Float64 called on non float64 value")
	}
	return math.Float64frombits(v.packed)
}

// IsFloat64 returns true if the value was built with SetFloat64.
func (v Label) IsFloat64() bool {
	_, ok := v.untyped.(float64Kind)
	return ok
}

// Bool returns a new Value for a bool.
func Bool(name string, b bool) Label {
	if b {
		return Label{Name: name, packed: 1, untyped: boolKind{}}
	}
	return Label{Name: name, packed: 0, untyped: boolKind{}}
}

// Bool returns the bool from a value that was set with SetBool.
// It will panic for any value for which IsBool is not true.
func (v Label) Bool() bool {
	if !v.IsBool() {
		panic("Bool called on non bool value")
	}
	if v.packed != 0 {
		return true
	}
	return false
}

// IsBool returns true if the value was built with SetBool.
func (v Label) IsBool() bool {
	_, ok := v.untyped.(boolKind)
	return ok
}

func Duration(name string, d time.Duration) Label {
	return Label{Name: name, packed: uint64(d), untyped: durationKind{}}
}

func (v Label) Duration() time.Duration {
	if !v.IsDuration() {
		panic("Duration called on non-Duration value")
	}
	return time.Duration(v.packed)
}

func (v Label) IsDuration() bool {
	_, ok := v.untyped.(durationKind)
	return ok
}
